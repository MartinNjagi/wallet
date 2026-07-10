package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
	"wallet/library"

	"wallet/data"
	"wallet/models"
)

// GetBalance returns the real-time active balance for the tenant
// @Summary Client Wallet Balance
func (ctr *Controller) GetBalance(ctx *gin.Context) {
	tokenClientID, exists := getClientID(ctx)
	if !exists {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized",
		})
		return
	}

	targetClientID := tokenClientID

	// SUPERADMIN VISIBILITY GATE
	if tokenClientID == 1 {
		if targetQuery := ctx.Query("client_id"); targetQuery != "" {
			if parsedID, err := strconv.ParseUint(targetQuery, 10, 32); err == nil {
				targetClientID = uint(parsedID)
			}
		}
	}

	var wallet models.Wallet
	if err := ctr.DB.Where("client_id = ?", targetClientID).First(&wallet).Error; err != nil {

		// Generate the PaymentRef using hashids
		friendlyRef, _ := library.GenerateFriendlyCode(uint64(targetClientID))

		wallet = models.Wallet{
			ClientID:   targetClientID,
			PaymentRef: friendlyRef,
			Balance:    0,
			Currency:   data.DefaultCurrency,
		}

		// Use a database transaction to safely create the wallet, the configuration, and the audit log
		err := ctr.DB.Transaction(func(tx *gorm.DB) error {
			// 1. Create the Wallet
			if err := tx.Create(&wallet).Error; err != nil {
				return err
			}

			// 2. Auto-Create default ClientBillingConfig alongside it
			config := models.ClientBillingConfig{
				ClientID:               targetClientID,
				BaseSmsRate:            1.0,  // Standard default rate per SMS unit
				RefundOnFailedDelivery: true, // Standard default logic
			}
			if err := tx.FirstOrCreate(&config, models.ClientBillingConfig{ClientID: targetClientID}).Error; err != nil {
				return err
			}

			// 3. Extract the active user for the audit log
			var userID uint
			if uid, ok := ctx.Get("user_id"); ok {
				userID = uid.(uint)
			}
			username := ctx.GetString("username")

			// 4. Log Audit (Pass `tx` so it is part of the atomic transaction)
			if err := ctr.LogAudit(tx, data.AuditLogParams{
				UserID:          userID,
				Username:        username,
				Action:          "AUTO_INITIALIZE_WALLET_AND_CONFIG",
				NewData:         map[string]interface{}{"wallet": wallet, "config": config},
				PerformedBy:     &userID,
				PerformedByName: &username,
				IPAddress:       ctx.ClientIP(),
			}); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			SendJSON(ctx, data.APIResponse{
				Status:  http.StatusInternalServerError,
				Message: "Failed to initialize wallet account or billing configuration",
			})
			return
		}
	}

	resp := data.WalletBalanceResponse{
		ClientID:   int64(wallet.ClientID),
		PaymentRef: wallet.PaymentRef,
		Balance:    wallet.Balance,
		Currency:   wallet.Currency,
	}

	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   &resp,
	})
}

func (ctr *Controller) InitiateTopUp(ctx *gin.Context) {
	// 1. Authenticate: Get Client ID from JWT
	clientID, exists := getClientID(ctx)
	if !exists {
		SendJSON(ctx, data.APIResponse{Status: http.StatusUnauthorized, Message: "Unauthorized"})
		return
	}

	var req data.InitiateTopUpRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}

	// 2. Rate Limiting: 1 request per 60 seconds per client using Redis SETNX
	rateLimitKey := fmt.Sprintf("stk_limit:client:%d", clientID)
	set, err := ctr.Redis.SetNX(ctx.Request.Context(), rateLimitKey, 1, 60*time.Second).Result()
	if err != nil || !set {
		SendJSON(ctx, data.APIResponse{Status: http.StatusTooManyRequests, Message: "Please wait 60 seconds before requesting another M-PESA prompt."})
		return
	}

	// 3. Generate Secure Reference (Decoupled from DB IDs)
	secureRef := library.GenerateSecureRef(7)

	// 4. Calculate Credits (from wallet config)
	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", clientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: clientID, BaseSmsRate: 1.0})
	credits := int64(req.Amount / config.BaseSmsRate)

	// 5. Fire STK Push to Daraja (Use secureRef as AccountReference)
	// mockCheckoutID := ctr.CallSafaricomSTK(req.Phone, req.Amount, secureRef)
	checkoutRequestID := fmt.Sprintf("ws_CO_%s", library.GenerateSecureRef(8))

	// 6. Save Intent to DB
	mpesaTx := models.MpesaTransaction{
		ClientID:          clientID,
		SecureReference:   secureRef,
		CheckoutRequestID: checkoutRequestID,
		Amount:            req.Amount,
		Credits:           credits,
		Status:            "PENDING",
	}
	ctr.DB.Create(&mpesaTx)

	// 7. Return Response with Fallback Instructions
	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: fmt.Sprintf("Payment prompt sent. If it fails, use Paybill %s with Account No: %s", ctr.Config.PaybillNumber, secureRef),
		Data: map[string]string{
			"checkout_request_id": checkoutRequestID,
			"account_reference":   secureRef,
		},
	})
}

// MpesaValidation accepts or rejects a Paybill payment before it completes.
func (ctr *Controller) MpesaValidation(ctx *gin.Context) {
	var payload data.C2BValidationPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 1, "ResultDesc": "Invalid payload", "ThirdPartyTransID": 0})
		return
	}

	// Look up the Wallet using the friendly PaymentRef instead of casting to integer!
	var wallet models.Wallet
	if err := ctr.DB.Where("payment_ref = ?", payload.BillRefNumber).First(&wallet).Error; err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 1, "ResultDesc": "Invalid Account Number", "ThirdPartyTransID": 0})
		return
	}

	// Accept
	ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted", "ThirdPartyTransID": 0})
}

// Admin Tools below: Require userClientID == 1

// ManualAdjustment SuperAdmin Manual Adjustments
func (ctr *Controller) ManualAdjustment(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		SendJSON(ctx, data.APIResponse{Status: http.StatusForbidden, Message: "SuperAdmin access required"})
		return
	}

	var req data.ManualAdjustmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: "Invalid payload"})
		return
	}

	// Unique ref ID based on timestamp and Admin ID
	adminID := ctx.MustGet("user_id").(uint)
	adminName := ctx.GetString("username")
	refID := fmt.Sprintf("%d_%s", adminID, generateSecureToken(6))

	err := ctr.ApplyWalletOperation(ctr.DB, data.WalletOperation{
		ClientID:    req.ClientID,
		Action:      data.WalletAction(req.Action),
		Credits:     req.Credits,
		Type:        data.RefPrefixAdj,
		Description: req.Description,
		Reference:   refID,
		FiatPaid:    nil,
		Currency:    nil,
	})

	if err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// Log Audit
	_ = ctr.LogAudit(nil, data.AuditLogParams{
		UserID:          adminID,
		Username:        adminName,
		Action:          "MANUAL_WALLET_ADJUSTMENT",
		NewData:         req,
		PerformedBy:     &adminID,
		PerformedByName: &adminName,
		IPAddress:       ctx.ClientIP(),
	})

	SendJSON(ctx, data.APIResponse{Status: http.StatusOK, Message: "Wallet adjusted successfully"})
}

// UpdateClientConfig allows admins to modify individual tenant SMS rates
func (ctr *Controller) UpdateClientConfig(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		SendJSON(ctx, data.APIResponse{Status: http.StatusForbidden, Message: "SuperAdmin access required"})
		return
	}

	targetClientID := ctx.Param("id")
	var req data.UpdateBillingConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: "Invalid payload"})
		return
	}

	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", targetClientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: 1, BaseSmsRate: 1.0})

	oldConfig := config // Copy for audit log

	if req.BaseSmsRate != nil {
		config.BaseSmsRate = *req.BaseSmsRate
	}
	if req.RefundOnFailedDelivery != nil {
		config.RefundOnFailedDelivery = *req.RefundOnFailedDelivery
	}

	ctr.DB.Save(&config)

	// Log Audit
	adminID := ctx.MustGet("user_id").(uint)
	adminName := ctx.GetString("username")
	_ = ctr.LogAudit(nil, data.AuditLogParams{
		UserID:          adminID,
		Username:        adminName,
		Action:          "UPDATE_BILLING_CONFIG",
		OldData:         oldConfig,
		NewData:         config,
		PerformedBy:     &adminID,
		PerformedByName: &adminName,
		IPAddress:       ctx.ClientIP(),
	})

	SendJSON(ctx, data.APIResponse{Status: http.StatusOK, Message: "Client billing configuration updated"})
}

// GetClientConfig fetches the billing config for a specific tenant
// @Summary Fetch Client Billing Config
func (ctr *Controller) GetClientConfig(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		SendJSON(ctx, data.APIResponse{Status: http.StatusForbidden, Message: "SuperAdmin access required"})
		return
	}

	targetClientID := ctx.Param("id")
	clientID, _ := strconv.Atoi(targetClientID)
	var config models.ClientBillingConfig
	var clientResp data.ClientConfiguration

	if err := ctr.DB.Where("client_id = ?", targetClientID).First(&config).Error; err != nil {
		// If not found, return default fallback config
		clientResp = data.ClientConfiguration{
			ClientID:               uint(clientID),
			BaseSMSRate:            1.0,
			RefundOnFailedDelivery: true,
		}
	}

	clientResp = data.ClientConfiguration{
		ClientID:               config.ClientID,
		BaseSMSRate:            config.BaseSmsRate,
		RefundOnFailedDelivery: config.RefundOnFailedDelivery,
	}
	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   clientResp,
	})
}
