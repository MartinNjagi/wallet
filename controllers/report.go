package controllers

import (
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"wallet/data"
	"wallet/models"
)

// ListLedger returns the history of transactions for the Dashboard with pagination
func (ctr *Controller) ListLedger(ctx *gin.Context) {
	userClientID := ctx.MustGet("client_id").(uint)
	page, pageSize, offset := getPaginationParams(ctx)

	query := ctr.DB.Model(&models.WalletTransaction{})

	// MULTI-TENANT & SUPERADMIN VISIBILITY GATE
	if userClientID != 1 {
		query = query.Where("client_id = ?", userClientID)
	} else {
		// SuperAdmin: Allow filtering by specific client if requested
		if targetClient := ctx.Query("client_id"); targetClient != "" {
			query = query.Where("client_id = ?", targetClient)
		}
	}

	var total int64
	query.Count(&total)

	var transactions []models.WalletTransaction
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&transactions).Error; err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusInternalServerError,
			Message: "Failed to load ledger",
		})
		return
	}

	var response []data.WalletTransactionResponse
	for _, t := range transactions {
		response = append(response, data.WalletTransactionResponse{
			ID:              t.ID,
			ClientID:        t.ClientID, // Useful for SuperAdmins looking at the global ledger
			Amount:          t.Amount,
			TransactionType: t.TransactionType,
			ReferenceType:   t.ReferenceType,
			ReferenceID:     t.ReferenceID,
			Description:     t.Description,
			FiatAmount:      t.FiatAmount,
			CreatedAt:       t.CreatedAt,
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   response,
		Pagination: &data.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// ListMpesaTransactions returns the history of Top-Up attempts with pagination
func (ctr *Controller) ListMpesaTransactions(ctx *gin.Context) {
	userClientID := ctx.MustGet("client_id").(uint)
	page, pageSize, offset := getPaginationParams(ctx)

	query := ctr.DB.Model(&models.MpesaTransaction{})

	// MULTI-TENANT & SUPERADMIN VISIBILITY GATE
	if userClientID != 1 {
		query = query.Where("client_id = ?", userClientID)
	} else {
		if targetClient := ctx.Query("client_id"); targetClient != "" {
			query = query.Where("client_id = ?", targetClient)
		}
		if status := ctx.Query("status"); status != "" {
			query = query.Where("status = ?", status) // e.g., PENDING, SUCCESS, FAILED
		}
	}

	var total int64
	query.Count(&total)

	var transactions []models.MpesaTransaction
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&transactions).Error; err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusInternalServerError,
			Message: "Failed to load M-Pesa transactions",
		})
		return
	}

	var response []data.MpesaTransactionResponse
	for _, t := range transactions {
		response = append(response, data.MpesaTransactionResponse{
			ID:                t.ID,
			ClientID:          t.ClientID,
			CheckoutRequestID: t.CheckoutRequestID,
			Amount:            t.Amount,
			Credits:           t.Credits,
			Status:            t.Status,
			ReceiptNumber:     t.ReceiptNumber,
			CreatedAt:         t.CreatedAt,
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   response,
		Pagination: &data.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// AdminWalletSummary provides a high-level system overview for SuperAdmins
func (ctr *Controller) AdminWalletSummary(ctx *gin.Context) {
	if ctx.MustGet("client_id").(uint) != 1 {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "SuperAdmin access required"})
		return
	}

	// Optional client filter for the summary
	targetClient := ctx.Query("client_id")
	var args []interface{}
	if targetClient != "" {
		args = append(args, targetClient)
	}

	var summary data.AdminWalletSummaryResponse

	// 1. Total System Balance
	walletQ := ctr.DB.Model(&models.Wallet{})
	if targetClient != "" {
		walletQ = walletQ.Where("client_id = ?", targetClient)
	}
	walletQ.Select("COALESCE(SUM(balance), 0)").Scan(&summary.TotalSystemBalance)

	// 2. Total Fiat Processed (SUCCESSFUL M-Pesa only)
	mpesaQ := ctr.DB.Model(&models.MpesaTransaction{}).Where("status = ?", "SUCCESS")
	if targetClient != "" {
		mpesaQ = mpesaQ.Where("client_id = ?", targetClient)
	}
	mpesaQ.Select("COALESCE(SUM(amount), 0)").Scan(&summary.TotalFiatProcessed)

	// 3. Total Credits Burned (Debits are stored as negative numbers, so we sum and make absolute)
	burnQ := ctr.DB.Model(&models.WalletTransaction{}).Where("transaction_type = ?", "DEBIT")
	if targetClient != "" {
		burnQ = burnQ.Where("client_id = ?", targetClient)
	}

	var rawBurn int64
	burnQ.Select("COALESCE(SUM(amount), 0)").Scan(&rawBurn)
	summary.TotalCreditsBurned = int64(math.Abs(float64(rawBurn)))

	// 4. Total Credits Granted (Credits)
	grantQ := ctr.DB.Model(&models.WalletTransaction{}).Where("transaction_type = ?", "CREDIT")
	if targetClient != "" {
		grantQ = grantQ.Where("client_id = ?", targetClient)
	}
	grantQ.Select("COALESCE(SUM(amount), 0)").Scan(&summary.TotalCreditsGranted)

	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data:   summary,
	})
}
