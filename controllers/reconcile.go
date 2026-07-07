package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"wallet/data"
	"wallet/library"
	"wallet/models"

	"github.com/sirupsen/logrus"
)

// ReconcilePendingMpesaTransactions should be hooked to a cron scheduler
func (ctr *Controller) ReconcilePendingMpesaTransactions() {
	logrus.Info("[Cron] Running M-Pesa Reconciliation...")

	var pendingTxs []models.MpesaTransaction
	// Find PENDING transactions older than 5 minutes
	fiveMinsAgo := time.Now().Add(-5 * time.Minute)
	if err := ctr.DB.Where("status = ? AND created_at < ?", "PENDING", fiveMinsAgo).Find(&pendingTxs).Error; err != nil {
		logrus.Errorf("Failed to query pending M-Pesa txs: %v", err)
		return
	}

	for _, tx := range pendingTxs {
		// 1. Call Safaricom M-Pesa Query Status API
		isPaid, err := ctr.querySafaricomStatus(context.Background(), tx.CheckoutRequestID)

		// 2. If the API call itself failed (e.g. timeout or Safaricom downtime), skip and try again next time
		if err != nil {
			logrus.Warnf("Reconciliation deferred for %s: %v", tx.CheckoutRequestID, err)
			continue
		}

		if isPaid {
			// 3. It was successful, but webhook dropped. Apply to Wallet Ledger.
			err := ctr.ApplyWalletOperation(
				ctr.DB, data.WalletOperation{
					ClientID:    tx.ClientID,
					Action:      data.WalletActionCredit,
					Credits:     tx.Credits,
					Type:        data.RefPrefixMpesa,
					Description: "M-Pesa STK Top Up (Recovered via Cron)",
					Reference:   tx.CheckoutRequestID,
					FiatPaid:    Float64Ptr(tx.Amount),
					Currency:    StringPtr(data.DefaultCurrency),
				},
			)

			if err == nil {
				// Mark as finalized to stop cron from processing it again
				ctr.DB.Model(&tx).Update("status", "SUCCESS")
				logrus.Infof("Cron reconciled and credited wallet for %s", tx.CheckoutRequestID)
			} else {
				logrus.Errorf("Cron failed to credit wallet for %s: %v", tx.CheckoutRequestID, err)
			}
		} else {
			// 4. Failed/Cancelled by user. Mark as failed securely.
			ctr.DB.Model(&tx).Update("status", "FAILED")
			logrus.Infof("Cron marked transaction %s as failed (User cancelled or insufficient funds)", tx.CheckoutRequestID)
		}
	}
}

// querySafaricomStatus checks the actual status of the STK push via Daraja
func (ctr *Controller) querySafaricomStatus(ctx context.Context, checkoutRequestID string) (bool, error) {
	shortCode := os.Getenv("c2b_paybill")
	passKey := os.Getenv("c2b_passkey")
	consumerKey := os.Getenv("c2b_consumer_key")
	consumerSecret := os.Getenv("c2b_consumer_secret")

	// STK Push Query Endpoint
	queryEndpoint := os.Getenv("mpesa_stk_push_query_endpoint")
	if queryEndpoint == "" {
		queryEndpoint = "https://api.safaricom.co.ke/mpesa/stkpushquery/v1/query" // Production default
	}

	// 1. Get Access Token (Reusing your existing cached method)
	token, err := ctr.getAccessToken(ctx, shortCode, consumerKey, consumerSecret)
	if err != nil {
		return false, fmt.Errorf("failed to get access token: %w", err)
	}

	// 2. Generate M-Pesa Password
	timestamp := library.MpesaTimestamp()
	auth := fmt.Sprintf("%s%s%s", shortCode, passKey, timestamp)
	password := base64.StdEncoding.EncodeToString([]byte(auth))

	payload := map[string]string{
		"BusinessShortCode": shortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"CheckoutRequestID": checkoutRequestID,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}

	// 3. Make the API Call
	responseBytes, status, err := library.HTTPRequest(ctx, "POST", queryEndpoint, payload, headers)
	if err != nil {
		return false, fmt.Errorf("HTTP request failed: %w", err)
	}

	// 4. Parse Response Safely
	var result map[string]interface{}
	if err := json.Unmarshal(responseBytes, &result); err != nil {
		return false, fmt.Errorf("failed to parse Daraja response: %w", err)
	}

	// Safaricom returns 400 or 500 when the transaction is still processing or missing.
	// We return an error here so the Cron leaves the status as PENDING and tries again later.
	if status >= 400 {
		return false, fmt.Errorf("Daraja API Error (%d): %s", status, string(responseBytes))
	}

	// Safaricom mixes types (ResultCode is sometimes a string, sometimes a float).
	// We cast it to a string safely.
	var resultCodeStr string
	if val, ok := result["ResultCode"]; ok {
		resultCodeStr = fmt.Sprintf("%v", val)
	} else {
		// If ResultCode doesn't exist, it's an anomalous response from Safaricom. Keep it PENDING.
		return false, fmt.Errorf("ResultCode missing from Daraja response: %s", string(responseBytes))
	}

	// ResultCode "0" specifically means the payment was completely successful.
	if resultCodeStr == "0" {
		return true, nil
	}

	// Any other ResultCode (e.g., 1032, 1037) means the transaction definitively failed or was cancelled by the user.
	// We return false, nil so the Cron marks it as FAILED in our database.
	return false, nil
}
