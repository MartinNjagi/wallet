package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"wallet/data"
	"wallet/models"
)

func (ctr *Controller) InternalBalanceCampaign(ctx *gin.Context) {

	var payload data.BalanceCreditRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	targetClientID, _ := strconv.Atoi(payload.ClientID)

	var wallet models.Wallet
	if err := ctr.DB.Where("client_id = ?", uint(targetClientID)).First(&wallet).Error; err != nil {
		// Return 0 if they've never transacted
		SendJSON(ctx, data.APIResponse{
			Status: http.StatusOK,
			Data: data.WalletBalanceResponse{
				ClientID: 0,
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

func (ctr *Controller) InternalDeductCampaign(ctx *gin.Context) {
	var payload data.DeductCreditRequest

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if payload.Amount <= 0 {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusBadRequest,
			Message: "Amount must be greater than zero",
		})
	}

	err := ctr.DeductCampaignCredits(payload)
	if err != nil {
		SendJSON(ctx, data.APIResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
	}

	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: "Credits successfully deducted",
	})
}

func (ctr *Controller) InternalRefundCampaign(ctx *gin.Context) {

	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: "Work In Progress",
	})
}
