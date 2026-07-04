package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"wallet/data"
	"wallet/library"
	"wallet/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 1. THE REQUEST: Auto-Verify Claim
func (ctr *Controller) ClaimMpesaPayment(ctx *gin.Context) {
	clientID := ctx.MustGet("client_id").(uint)

	var req struct {
		ReceiptNumber string `json:"receipt_number" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receipt := strings.TrimSpace(req.ReceiptNumber)

	// Quick check: Already processed?
	var count int64
	ctr.DB.Model(&models.MpesaTransaction{}).Where("receipt_number = ?", receipt).Count(&count)
	ctr.DB.Model(&models.C2BTransaction{}).Where("transaction_id = ?", receipt).Count(&count)

	if count > 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "This receipt has already been processed."})
		return
	}

	// Save dispute as PENDING
	dispute := models.MpesaDispute{
		ClientID:      clientID,
		ReceiptNumber: receipt,
		Amount:        0, // We will figure out the amount from Daraja
		Status:        "PENDING",
	}
	if err := ctr.DB.Create(&dispute).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Claim already submitted."})
		return
	}

	// Prepare Daraja API Request
	initiatorName := os.Getenv("MPESA_INITIATOR_NAME")
	initiatorPass := os.Getenv("MPESA_INITIATOR_PASSWORD")
	certBucket := os.Getenv("S3_BUCKET")
	certKey := "certs/ProductionCertificate.cer" // The path in your S3 bucket

	// Use our new S3 Helper!
	encryptedPass, err := library.EncryptInitiatorPasswordS3(
		ctx.Request.Context(),
		ctr.S3Client,
		certBucket,
		certKey,
		initiatorPass,
	)

	if err != nil {
		logrus.Errorf("Failed to encrypt password: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal security error connecting to Safaricom"})
		return
	}

	payload := map[string]string{
		"Initiator":          initiatorName,
		"SecurityCredential": encryptedPass,
		"CommandID":          "TransactionStatusQuery",
		"TransactionID":      receipt,
		"PartyA":             os.Getenv("c2b_paybill"),
		"IdentifierType":     "4",
		"ResultURL":          os.Getenv("SERVER_HOST") + "/api/v1/webhooks/mpesa/tx-status",
		"QueueTimeOutURL":    os.Getenv("SERVER_HOST") + "/api/v1/webhooks/mpesa/tx-timeout",
		"Remarks":            "Dispute verification",
		"Occasion":           "Dispute",
	}

	// Get Token and Fire Request (Fire and Forget)
	go func() {
		token, _ := ctr.getAccessToken(context.Background(), os.Getenv("c2b_paybill"), os.Getenv("c2b_consumer_key"), os.Getenv("c2b_consumer_secret"))
		jsonPayload, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		res, _ := client.Do(req)
		defer res.Body.Close()
	}()

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Claim submitted. We are verifying your receipt with Safaricom. Your account will be credited automatically once confirmed.",
	})
}

// 2. THE RESULT CALLBACK
func (ctr *Controller) TxStatusWebhook(ctx *gin.Context) {
	var payload data.TxStatusCallbackPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 1, "ResultDesc": "Invalid JSON"})
		return
	}

	result := payload.Result
	receiptNumber := result.TransactionID

	var dispute models.MpesaDispute
	if err := ctr.DB.Where("receipt_number = ? AND status = 'PENDING'", receiptNumber).First(&dispute).Error; err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Ignored"})
		return
	}

	// Did Daraja find it?
	if result.ResultCode != 0 {
		ctr.DB.Model(&dispute).Update("status", "REJECTED")
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Acknowledged Failure"})
		return
	}

	// Extract Amount from Safaricom Metadata
	var actualAmount float64
	for _, param := range result.ResultParameters.ResultParameter {
		if param.Key == "Amount" {
			// Daraja might return a string or a float, safe assertion
			if valFloat, ok := param.Value.(float64); ok {
				actualAmount = valFloat
			} else if valStr, ok := param.Value.(string); ok {
				fmt.Sscanf(valStr, "%f", &actualAmount)
			}
		}
	}

	if actualAmount <= 0 {
		ctr.DB.Model(&dispute).Update("status", "REJECTED")
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Invalid Amount"})
		return
	}

	// Credit Wallet Ledger
	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", dispute.ClientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: dispute.ClientID, BaseSmsRate: 1.0})
	credits := int64(actualAmount / config.BaseSmsRate)

	err := ctr.ApplyWalletOperation(ctr.DB, data.WalletOperation{
		ClientID:    dispute.ClientID,
		Action:      data.WalletActionCredit,
		Credits:     credits,
		Type:        data.RefPrefixMpesa,
		Description: "Auto-Recovered M-Pesa Payment",
		Reference:   receiptNumber,
		FiatPaid:    &actualAmount,
		Currency:    StringPtr(data.DefaultCurrency),
	})

	if err == nil {
		ctr.DB.Model(&dispute).Updates(map[string]interface{}{
			"status": "APPROVED",
			"amount": actualAmount,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}

// TxTimeoutWebhook THE TIMEOUT CALLBACK
func (ctr *Controller) TxTimeoutWebhook(ctx *gin.Context) {
	// Safaricom sends basic info when it times out internally
	var payload map[string]interface{}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ResultCode": 1, "ResultDesc": "Invalid payload"})
		return
	}

	// Safaricom timeout payloads usually only contain the OriginatorConversationID
	// You generally don't have the receipt number here.
	// The best approach is simply to acknowledge the timeout.
	logrus.Warnf("Daraja Transaction Status API Timed out: %+v", payload)

	// We leave the dispute as 'PENDING' in the database.
	// This means an Admin can manually look at it and approve it later.

	ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
}
