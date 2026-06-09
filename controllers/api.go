package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"

	"wallet/data"
	"wallet/models"
)

// GetBalance returns the real-time active balance for the tenant
// @Summary Client Wallet Balance
func (ctr *Controller) GetBalance(ctx *gin.Context) {
	// Use the same helper from ListLedger
	tokenClientID, exists := getClientID(ctx)
	if !exists {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized",
		})
		return
	}

	targetClientID := tokenClientID

	// SUPERADMIN VISIBILITY GATE (Matches ListLedger logic)
	if tokenClientID == 1 {
		if targetQuery := ctx.Query("client_id"); targetQuery != "" {
			if parsedID, err := strconv.ParseUint(targetQuery, 10, 32); err == nil {
				targetClientID = uint(parsedID)
			}
		}
	}

	var wallet models.Wallet
	if err := ctr.DB.Where("client_id = ?", targetClientID).First(&wallet).Error; err != nil {
		// Return 0 if they've never transacted
		SendJSON(ctx, data.APIResponse{
			Status: http.StatusOK,
			Data: data.WalletBalanceResponse{
				ClientID: int64(targetClientID),
				Balance:  0,
				Currency: data.DefaultCurrency,
			},
		})
		return
	}

	resp := data.WalletBalanceResponse{
		ClientID: int64(wallet.ClientID),
		Balance:  wallet.Balance,
		Currency: wallet.Currency,
	}
	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   &resp,
	})
}

// InitiateTopUp triggers the M-Pesa STK push after calculating exchange rates
func (ctr *Controller) InitiateTopUp(ctx *gin.Context) {
	clientID := ctx.MustGet("client_id").(uint)

	var req data.InitiateTopUpRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// 1. Get Client's Rate
	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", clientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: clientID, BaseSmsRate: 1.0})

	// 2. Calculate Credits
	credits := int64(req.Amount / config.BaseSmsRate)

	// 3. Fire to M-Pesa API (Mocked here)
	checkoutRequestID := fmt.Sprintf("ws_CO_%s", generateSecureToken(8)) // Mock ID from Safaricom

	// 4. Record Pending Transaction for Cron reconciliation
	mpesaTx := models.MpesaTransaction{
		ClientID:          clientID,
		CheckoutRequestID: checkoutRequestID,
		Amount:            req.Amount,
		Credits:           credits,
		Status:            "PENDING",
	}
	ctr.DB.Create(&mpesaTx)

	ctx.JSON(http.StatusOK, gin.H{
		"message":             "STK Push Initiated",
		"checkout_request_id": checkoutRequestID,
		"expected_credits":    credits,
	})
}

// MpesaWebhook receives IPN from Safaricom
// Idempotent and highly concurrent safe.
func (ctr *Controller) MpesaWebhook(ctx *gin.Context) {
	// 1. Parse Callback (Mock format)
	var callback struct {
		Body struct {
			StkCallback struct {
				CheckoutRequestID string `json:"CheckoutRequestID"`
				ResultCode        int    `json:"ResultCode"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}
	if err := ctx.ShouldBindJSON(&callback); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}

	reqID := callback.Body.StkCallback.CheckoutRequestID
	success := callback.Body.StkCallback.ResultCode == 0

	// 2. Update Mpesa Log
	var mpesaTx models.MpesaTransaction
	if err := ctr.DB.Where("checkout_request_id = ?", reqID).First(&mpesaTx).Error; err != nil {
		ctx.Status(http.StatusOK) // Always 200 back to Safaricom
		return
	}

	// 3. If failed, just update status and exit
	if !success {
		ctr.DB.Model(&mpesaTx).Update("status", "FAILED")
		ctx.Status(http.StatusOK)
		return
	}

	// 4. If success, credit the wallet using the Core Ledger Engine!
	err := ctr.ApplyWalletOperation(
		ctr.DB, data.WalletOperation{
			ClientID:    mpesaTx.ClientID,
			Action:      data.WalletActionCredit,
			Credits:     mpesaTx.Credits,
			Type:        data.RefPrefixMpesa,
			Description: "M-Pesa STK Top Up",
			Reference:   mpesaTx.CheckoutRequestID,
			FiatPaid:    Float64Ptr(mpesaTx.Amount),
			Currency:    StringPtr(data.DefaultCurrency),
		},
	)

	if err == nil {
		// Mark Safaricom transaction as finalized
		ctr.DB.Model(&mpesaTx).Updates(map[string]interface{}{
			"status":         "SUCCESS",
			"receipt_number": "RGT45MOCK", // Mock extraction from IPN data
		})
	} else {
		logrus.Errorf("Wallet Credit Failed for STK %s: %v", reqID, err)
	}

	ctx.Status(http.StatusOK)
}

// Admin Tools below: Require userClientID == 1

// ManualAdjustment allows SuperAdmins to fix billing errors or give promos
// @Summary SuperAdmin Manual Adjustments
func (ctr *Controller) ManualAdjustment(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusForbidden,
			Message: "SuperAdmin access required",
		})
		return
	}

	var req data.ManualAdjustmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusBadRequest,
			Message: "Invalid payload",
		})
		return
	}

	// Unique ref ID based on timestamp and Admin ID
	adminID := ctx.GetUint("user_id")
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
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: "Wallet adjusted successfully",
	})
}

// UpdateClientConfig allows admins to modify individual tenant SMS rates
func (ctr *Controller) UpdateClientConfig(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusForbidden,
			Message: "SuperAdmin access required",
		})
		return
	}

	targetClientID := ctx.Param("id")
	var req data.UpdateBillingConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusBadRequest,
			Message: "Invalid payload",
		})
		return
	}

	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", targetClientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: 1, BaseSmsRate: 1.0})

	if req.BaseSmsRate != nil {
		config.BaseSmsRate = *req.BaseSmsRate
	}
	if req.RefundOnFailedDelivery != nil {
		config.RefundOnFailedDelivery = *req.RefundOnFailedDelivery
	}

	ctr.DB.Save(&config)

	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: "Client billing configuration updated",
	})
}
