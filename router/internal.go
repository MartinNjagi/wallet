package routers

import "github.com/gin-gonic/gin"

// --- Internal M2M Endpoints (SMS Engine) ---

// @Summary Poll Credits for Campaign
// @Tags Internal Wallet
func (a *App) InternalBalanceCampaign(c *gin.Context) { a.Controller.InternalBalanceCampaign(c) }

// @Summary Deduct Credits for Campaign
// @Tags Internal Wallet
func (a *App) InternalDeductCampaign(c *gin.Context) { a.Controller.InternalDeductCampaign(c) }

// @Summary Refund Credits for Failed SMS
// @Tags Internal Wallet
func (a *App) InternalRefundCampaign(c *gin.Context) { a.Controller.InternalRefundCampaign(c) }
