package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"wallet/data"
	"wallet/models"
)

func (ctr *Controller) RegisterC2BUrls(c *gin.Context) {
	ctx := c.Request.Context()
	shortCode := os.Getenv("c2b_paybill")
	consumerKey := os.Getenv("c2b_consumer_key")
	consumerSecret := os.Getenv("c2b_consumer_secret")
	baseURL := os.Getenv("MPESA_WEBHOOK_HOST")
	secret := os.Getenv("MPESA_WEBHOOK_SECRET")

	payload := map[string]string{
		"ShortCode":       shortCode,
		"ResponseType":    "Completed",
		"ConfirmationURL": fmt.Sprintf("%s/api/v1/webhooks/%s/mpesa/confirm", baseURL, secret),
		"ValidationURL":   fmt.Sprintf("%s/api/v1/webhooks/%s/mpesa/validate", baseURL, secret),
	}
	// Get access token
	token, err := ctr.getAccessToken(ctx, shortCode, consumerKey, consumerSecret)
	if err != nil {
		logrus.Printf("Failed to get access token: %v", err)
		c.JSON(http.StatusInternalServerError, "Failed to get access token")
		return
	}

	jsonPayload, _ := json.Marshal(payload)

	// Create request
	req, err := http.NewRequest("POST", "https://api.safaricom.co.ke/mpesa/c2b/v2/registerurl", bytes.NewBuffer(jsonPayload))
	if err != nil {
		logrus.Printf("Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, "Failed to create request ")
		return
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logrus.Printf("Request failed: %v", err)
		c.JSON(http.StatusInternalServerError, "Request failed")
		return
	}
	defer res.Body.Close()

	// Read response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.Printf("Failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, "Failed to read response body")
		return
	}

	fmt.Println("C2B Registration Response:", string(body))
	c.JSON(http.StatusOK, gin.H{
		"message": "C2B URLs registered successfully",
	})

}

// MpesaWebhook handles the M-Pesa STK push callback.
func (ctr *Controller) MpesaWebhook(ctx *gin.Context) {
	// ── 1. Parse callback payload ─────────────────────────────────────────
	var payload data.StkCallbackPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		logrus.WithContext(ctx).Printf("STKCallback | JSON bind error: %v", err)
		// Safaricom expects this exact format for errors
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 1, "ResultDesc": "Invalid JSON payload"})
		return
	}

	stkCallback := payload.Body.StkCallback
	checkoutRequestID := stkCallback.CheckoutRequestID
	resultCode := stkCallback.ResultCode
	resultDesc := stkCallback.ResultDesc

	logrus.WithContext(ctx).Printf("STKCallback | CheckoutRequestID=%s ResultCode=%d ResultDesc=%s",
		checkoutRequestID, resultCode, resultDesc)

	// Validate we received the CheckoutRequestID
	if checkoutRequestID == "" {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 1, "ResultDesc": "Missing CheckoutRequestID"})
		return
	}

	// ── 2. Fetch STK push record securely using CheckoutRequestID ─────────
	// Using the MpesaTransaction model we set up for the Wallet service
	var mpesaTx models.MpesaTransaction
	if err := ctr.DB.Where("checkout_request_id = ?", checkoutRequestID).First(&mpesaTx).Error; err != nil {
		logrus.WithContext(ctx).Printf("STKCallback | Record not found for %s: %v", checkoutRequestID, err)
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Ignored - Transaction not found"})
		return
	}

	// ── 3. Idempotency Check ──────────────────────────────────────────────
	// Safaricom sometimes sends the same callback twice. Ignore if already processed.
	if mpesaTx.Status != "PENDING" {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Already processed"})
		return
	}

	// ── 4. Handle Failure ─────────────────────────────────────────────────
	if resultCode != 0 {
		ctr.DB.Model(&mpesaTx).Updates(map[string]interface{}{
			"status": "FAILED",
		})
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Acknowledged Failure"})
		return
	}

	// ── 5. Handle Success & Extract Metadata ──────────────────────────────
	var receiptNumber string
	if stkCallback.CallbackMetadata != nil {
		for _, item := range stkCallback.CallbackMetadata.Item {
			if item.Name == "MpesaReceiptNumber" {
				if v, ok := item.Value.(string); ok {
					receiptNumber = v
				}
			}
		}
	}

	// ── 6. Credit the Wallet Ledger (CRITICAL STEP) ───────────────────────
	err := ctr.ApplyWalletOperation(ctr.DB, data.WalletOperation{
		ClientID:    mpesaTx.ClientID,
		Action:      data.WalletActionCredit,
		Credits:     mpesaTx.Credits,
		Type:        data.RefPrefixMpesa,
		Description: "M-Pesa STK Top Up",
		Reference:   checkoutRequestID, // Keep Daraja Request ID as the absolute ledger reference
		FiatPaid:    &mpesaTx.Amount,
		Currency:    StringPtr(data.DefaultCurrency),
	})

	if err != nil {
		logrus.WithContext(ctx).Errorf("STKCallback | Wallet Credit Failed for %s: %v", checkoutRequestID, err)
		// Even if the DB fails, tell Safaricom we received it so they stop retrying
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Internal Ledger Error"})
		return
	}

	// ── 7. Mark STK record as Success ─────────────────────────────────────
	ctr.DB.Model(&mpesaTx).Updates(map[string]interface{}{
		"status":         "SUCCESS",
		"receipt_number": receiptNumber,
	})

	// ── 8. Return exactly what Safaricom Daraja API expects ───────────────
	ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}

// MpesaConfirmation credits the wallet after a successful Paybill payment.
func (ctr *Controller) MpesaConfirmation(ctx *gin.Context) {
	var payload data.C2BValidationPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": "0", "ResultDesc": "Ignored"})
		return
	}

	amount, _ := strconv.ParseFloat(payload.TransAmount, 64)
	// Look up Wallet to get the actual ClientID
	var wallet models.Wallet
	if err := ctr.DB.Where("payment_ref = ?", payload.BillRefNumber).First(&wallet).Error; err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Client not found"})
		return
	}

	clientID := wallet.ClientID
	// 1. Get Client's Rate
	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", clientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: clientID, BaseSmsRate: 1.0})
	credits := int64(amount / config.BaseSmsRate)

	// 2. Save C2B Record
	txn := models.C2BTransaction{
		ClientID:      clientID,
		TransactionID: payload.TransID,
		Amount:        amount,
		Credits:       credits,
		BillRefNumber: payload.BillRefNumber,
		MSISDN:        payload.MSISDN,
	}

	if err := ctr.DB.Create(&txn).Error; err != nil {
		// Duplicate IPN from Safaricom, ignore safely
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": "0", "ResultDesc": "Success"})
		return
	}

	// 3. Apply to Ledger
	err := ctr.ApplyWalletOperation(ctr.DB, data.WalletOperation{
		ClientID:    uint(clientID),
		Action:      data.WalletActionCredit,
		Credits:     credits,
		Type:        data.RefPrefixMpesa,
		Description: "M-Pesa Paybill Top Up",
		Reference:   payload.TransID, // Use Safaricom receipt as reference
		FiatPaid:    &amount,
		Currency:    StringPtr(data.DefaultCurrency),
	})
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Internal Ledger Error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ResultCode": "0", "ResultDesc": "Success"})
}
