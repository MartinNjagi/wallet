package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wallet/data"
)

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
