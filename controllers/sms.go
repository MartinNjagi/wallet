package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wallet/data"
)

func (ctr *Controller) InternalDeductCampaign(ctx *gin.Context) {
	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: "Work In Progress",
	})
}

func (ctr *Controller) InternalRefundCampaign(ctx *gin.Context) {
	SendJSON(ctx, data.APIResponse{
		Status:  http.StatusOK,
		Message: "Work In Progress",
	})
}
