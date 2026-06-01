package controllers

import (
	"time"
	"wallet/data"

	"github.com/sirupsen/logrus"

	"wallet/models"
)

// ReconcilePendingMpesaTransactions should be hooked to a cron scheduler (e.g., robfig/cron)
// to run every ~5 minutes. It catches transactions where Safaricom dropped the callback.
func (ctr *Controller) ReconcilePendingMpesaTransactions() {
	logrus.Info("[Cron] Running M-Pesa Reconciliation...")

	var pendingTxs []models.MpesaTransaction
	// Find transactions that are PENDING and older than 5 minutes (to avoid clashing with incoming callbacks)
	fiveMinsAgo := time.Now().Add(-5 * time.Minute)
	if err := ctr.DB.Where("status = ? AND created_at < ?", "PENDING", fiveMinsAgo).Find(&pendingTxs).Error; err != nil {
		logrus.Errorf("Failed to query pending M-Pesa txs: %v", err)
		return
	}

	for _, tx := range pendingTxs {
		// 1. Call the Safaricom M-Pesa Query Status API
		//    (Mocked: Assuming success for demonstration purposes)
		isPaid, _ := ctr.querySafaricomStatus(tx.CheckoutRequestID)

		if isPaid {
			// 2. It was successful, but webhook dropped. Apply to Wallet Ledger.
			err := ctr.ApplyWalletOperation(
				ctr.DB, WalletOperation{
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
			// Failed/Cancelled by user. Mark as failed.
			ctr.DB.Model(&tx).Update("status", "FAILED")
			logrus.Infof("Cron marked transaction %s as failed", tx.CheckoutRequestID)
		}
	}
}

// Mock method for checking upstream provider
func (ctr *Controller) querySafaricomStatus(checkoutRequestID string) (bool, error) {
	// Normally you'd make an HTTP GET to Daraja API using your credentials
	return false, nil
}
