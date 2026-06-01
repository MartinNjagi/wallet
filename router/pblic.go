package routers

import "github.com/gin-gonic/gin"

// --- Dashboard (Tenant) Endpoints ---

// @Summary Client Wallet Balance
// @Tags Wallet
func (a *App) GetBalance(c *gin.Context) { a.Controller.GetBalance(c) }

// @Summary Billing History / Ledger Table
// @Tags Wallet
func (a *App) ListLedger(c *gin.Context) { a.Controller.ListLedger(c) }

// @Summary Initiate M-Pesa Top Up
// @Tags Wallet
func (a *App) InitiateTopUp(c *gin.Context) { a.Controller.InitiateTopUp(c) }

// --- SuperAdmin Endpoints ---

// @Summary SuperAdmin Manual Adjustments
// @Tags Admin Wallet
func (a *App) ManualAdjustment(c *gin.Context) { a.Controller.ManualAdjustment(c) }

// @Summary Update Client Billing Config
// @Tags Admin Wallet
func (a *App) UpdateClientConfig(c *gin.Context) { a.Controller.UpdateClientConfig(c) }

// --- Internal M2M Endpoints (SMS Engine) ---

// @Summary Deduct Credits for Campaign
// @Tags Internal Wallet
func (a *App) InternalDeductCampaign(c *gin.Context) { a.Controller.InternalDeductCampaign(c) }

// @Summary Refund Credits for Failed SMS
// @Tags Internal Wallet
func (a *App) InternalRefundCampaign(c *gin.Context) { a.Controller.InternalRefundCampaign(c) }

// --- Public Webhooks ---

// @Summary M-Pesa IPN Webhook
// @Tags Webhooks
func (a *App) MpesaWebhook(c *gin.Context) { a.Controller.MpesaWebhook(c) }
