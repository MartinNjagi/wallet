package controllers

import (
	"errors"
	"fmt"
	"strings"
	"wallet/data"
	"wallet/library"
	"wallet/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ApplyWalletOperation is the central ACID-compliant ledger writer.
// It uses row-level locking (SELECT FOR UPDATE) to prevent race conditions during high-throughput SMS bursts.
func (ctr *Controller) ApplyWalletOperation(db *gorm.DB, op data.WalletOperation) error {
	if op.Credits <= 0 {
		return errors.New("credits must be positive")
	}

	var signedCredits int64

	switch op.Action {
	case data.WalletActionCredit:
		signedCredits = op.Credits
	case data.WalletActionDebit:
		signedCredits = -op.Credits
	default:
		return fmt.Errorf("invalid wallet action: %s", op.Action)
	}

	// Clean reference ID (Combine prefix + ID if not already combined)
	fullRef := op.Reference
	if op.Type != "" && !strings.HasPrefix(op.Reference, op.Type) {
		fullRef = fmt.Sprintf("%s%s", op.Type, op.Reference)
	}

	if db == nil {
		db = ctr.DB
	}

	// Always wrap in a transaction. If a parent tx is passed, GORM handles it gracefully via SavePoints
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Check Idempotency immediately (prevents duplicate webhooks/DLRs from executing)
		var count int64
		tx.Model(&models.WalletTransaction{}).Where("reference_id = ?", fullRef).Count(&count)
		if count > 0 {
			// Already processed. Return nil or a specific error so the caller can ignore.
			return errors.New("idempotency violation: transaction already recorded")
		}

		// 2. Fetch or Create Wallet with an Exclusive Lock
		var wallet models.Wallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("client_id = ?", op.ClientID).First(&wallet).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Initialize missing wallet on the fly

				// Generate a secure alphanumeric string for the user (e.g., "K8X9P2")
				friendlyRef, _ := library.GenerateFriendlyCode(uint64(op.ClientID))
				wallet = models.Wallet{
					ClientID:   op.ClientID,
					PaymentRef: friendlyRef,
					Balance:    0}
				if err := tx.Create(&wallet).Error; err != nil {
					return fmt.Errorf("failed to create wallet: %v", err)
				}
			} else {
				return err
			}
		}

		// 3. Balance Checks
		newBalance := wallet.Balance + signedCredits
		if newBalance < 0 {
			return fmt.Errorf("insufficient wallet balance: have %d need %d", wallet.Balance, op.Credits)
		}

		// 4. Update the Wallet
		wallet.Balance = newBalance
		if err := tx.Save(&wallet).Error; err != nil {
			return fmt.Errorf("failed to update wallet balance: %v", err)
		}

		// 5. Write to the Immutable Ledger
		transType := "CREDIT"
		if op.Action == data.WalletActionDebit {
			transType = "DEBIT"
		}

		fiatAmount := float64(0)
		if op.FiatPaid != nil {
			fiatAmount = *op.FiatPaid
		}

		txn := models.WalletTransaction{
			WalletID:        wallet.ID,
			ClientID:        op.ClientID,
			Amount:          signedCredits,
			TransactionType: transType,
			ReferenceType:   strings.TrimSuffix(op.Type, "_"),
			ReferenceID:     fullRef,
			Description:     op.Description,
			FiatAmount:      fiatAmount,
		}

		if err := tx.Create(&txn).Error; err != nil {
			return fmt.Errorf("failed to write to ledger: %v", err)
		}

		return nil
	})
}

// ProcessCampaignRefund handles bulk refunds for failed messages in a specific campaign/batch
func (ctr *Controller) ProcessCampaignRefund(req data.RefundCreditRequest) error {
	// If there were no failures, skip processing
	failedCount := int64(req.Amount)
	if failedCount <= 0 {
		return nil
	}

	var config models.ClientBillingConfig
	// Get config, default to refunding if config table entry is missing
	if err := ctr.DB.Where("client_id = ?", req.ClientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: req.ClientID, RefundOnFailedDelivery: true}).Error; err != nil {
		return err
	}

	// Respect the Client's Refund Strategy Config
	if !config.RefundOnFailedDelivery {
		// Just log it and return; no funds given back
		return nil
	}

	// Refund the total failed credits in one bulk transaction
	err := ctr.ApplyWalletOperation(
		ctr.DB,
		data.WalletOperation{
			ClientID:    req.ClientID,
			Action:      data.WalletActionCredit,
			Credits:     failedCount,
			Type:        data.RefPrefixRev,
			Reference:   req.CampaignID,
			Description: fmt.Sprintf("Bulk refund for %d failed SMS in campaign", failedCount),
		},
	)

	// If err contains idempotency violation, we safely ignore it (we already refunded this campaign)
	if err != nil && strings.Contains(err.Error(), "idempotency") {
		return nil
	}
	return err
}

// DeductCampaignCredits handles the bulk deduction for a campaign before sending starts.
// The SMS engine must call this to lock in the funds required for the whole campaign.
func (ctr *Controller) DeductCampaignCredits(req data.DeductCreditRequest) error {
	if req.Amount <= 0 {
		return nil
	}

	totalCredits := int64(req.Amount)
	// Deduct the total credits in one bulk transaction
	err := ctr.ApplyWalletOperation(
		ctr.DB,
		data.WalletOperation{
			ClientID:    req.ClientID,
			Action:      data.WalletActionDebit,
			Credits:     totalCredits,
			Type:        data.RefPrefixSMS,
			Reference:   req.CampaignID,
			Description: fmt.Sprintf("Bulk deduction for %d SMS in campaign", totalCredits),
		},
	)

	return err
}
