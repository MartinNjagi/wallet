package controllers

import (
	"fmt"
	"math"
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

// ListBankTransfers (Admin facing)
func (ctr *Controller) ListBankTransfers(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		SendJSON(ctx, data.APIResponse{Status: http.StatusForbidden, Message: "SuperAdmin access required"})
		return
	}

	page, pageSize, offset := getPaginationParams(ctx)

	query := ctr.DB.Model(&models.BankTransaction{})
	if status := ctx.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var transactions []models.BankTransaction
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&transactions).Error; err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusInternalServerError, Message: "Failed to load bank transfers"})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   transactions,
		Pagination: &data.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// ApproveBankTransfer (Admin facing)
func (ctr *Controller) ApproveBankTransfer(ctx *gin.Context) {
	adminID := ctx.MustGet("user_id").(uint)
	adminName := ctx.GetString("username")
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

	// Log Audit
	_ = ctr.LogAudit(nil, data.AuditLogParams{
		UserID:          adminID,
		Username:        adminName,
		Action:          "BANK_TRANSFER_REVIEW",
		OldData:         map[string]interface{}{"status": "PENDING"},
		NewData:         map[string]interface{}{"status": req.Status, "description": req.Description, "txn_id": txn.ID},
		PerformedBy:     &adminID,
		PerformedByName: &adminName,
		IPAddress:       ctx.ClientIP(),
	})

	SendJSON(ctx, data.APIResponse{Status: http.StatusOK, Message: "Bank transfer " + req.Status})
}
