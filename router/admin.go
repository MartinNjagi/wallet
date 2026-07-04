package routers

import "github.com/gin-gonic/gin"

// --- SuperAdmin Endpoints ---

// @Summary SuperAdmin Manual Adjustments
// @Tags Admin Wallet
func (a *App) ManualAdjustment(c *gin.Context) { a.Controller.ManualAdjustment(c) }

// @Summary Update Client Billing Config
// @Tags Admin Wallet
func (a *App) UpdateClientConfig(c *gin.Context) { a.Controller.UpdateClientConfig(c) }

// @Summary Read allSummary Data
// @Tags Admin Wallet
func (a *App) AdminWalletSummary(c *gin.Context) { a.Controller.AdminWalletSummary(c) }

func (a *App) ApproveBankTransfer(c *gin.Context) { a.Controller.ApproveBankTransfer(c) }
