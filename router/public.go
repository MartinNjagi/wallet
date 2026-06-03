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
