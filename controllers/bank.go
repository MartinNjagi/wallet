package controllers

import (
	"fmt"
	"net/http"
	"wallet/data"
	"wallet/models"

	"github.com/gin-gonic/gin"
)

// SubmitBankTransfer (Client facing)
func (ctr *Controller) SubmitBankTransfer(ctx *gin.Context) {
	clientID := ctx.MustGet("client_id").(uint)

	var req data.SubmitBankTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}

	// Calculate expected credits
	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", clientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: clientID, BaseSmsRate: 1.0})
	credits := int64(req.Amount / config.BaseSmsRate)

	txn := models.BankTransaction{
		ClientID:        clientID,
		Amount:          req.Amount,
		Credits:         credits,
		ReferenceNumber: req.ReferenceNumber,
		ProofURL:        req.ProofURL,
		Status:          "PENDING",
	}

	if err := ctr.DB.Create(&txn).Error; err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusInternalServerError, Message: "Failed to submit request. Ref might be duplicate."})
		return
	}

	SendJSON(ctx, data.APIResponse{Status: http.StatusCreated, Message: "Bank transfer submitted for approval", Data: txn})
}

// ApproveBankTransfer (Admin facing)
func (ctr *Controller) ApproveBankTransfer(ctx *gin.Context) {
	adminID := ctx.MustGet("user_id").(uint)
	txnID := ctx.Param("id")

	var req data.ApproveBankTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}

	var txn models.BankTransaction
	if err := ctr.DB.First(&txn, txnID).Error; err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusNotFound, Message: "Transaction not found"})
		return
	}

	if txn.Status != "PENDING" {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: "Transaction already processed"})
		return
	}

	// If approved, update ledger
	if req.Status == "APPROVED" {
		err := ctr.ApplyWalletOperation(ctr.DB, data.WalletOperation{
			ClientID:    txn.ClientID,
			Action:      data.WalletActionCredit,
			Credits:     txn.Credits,
			Type:        "BANK_",
			Description: fmt.Sprintf("Bank Transfer Approved: %s. %s", txn.ReferenceNumber, req.Description),
			Reference:   txn.ReferenceNumber,
			FiatPaid:    &txn.Amount,
			Currency:    StringPtr(data.DefaultCurrency),
		})

		if err != nil {
			SendJSON(ctx, data.APIResponse{Status: http.StatusInternalServerError, Message: err.Error()})
			return
		}
	}

	// Update record
	ctr.DB.Model(&txn).Updates(map[string]interface{}{
		"status":      req.Status,
		"approved_by": adminID,
	})

	SendJSON(ctx, data.APIResponse{Status: http.StatusOK, Message: "Bank transfer " + req.Status})
}
