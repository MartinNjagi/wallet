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
func (a *App) InitiateTopUp(c *gin.Context)       { a.Controller.InitiateTopUp(c) }
func (a *App) InitiateStripeTopUp(c *gin.Context) { a.Controller.InitiateStripeTopUp(c) }

func (a *App) MpesaValidation(c *gin.Context)    { a.Controller.MpesaValidation(c) }
func (a *App) MpesaConfirmation(c *gin.Context)  { a.Controller.MpesaConfirmation(c) }
func (a *App) SubmitBankTransfer(c *gin.Context) { a.Controller.SubmitBankTransfer(c) }

func (a *App) ClaimMpesaPayment(c *gin.Context) { a.Controller.ClaimMpesaPayment(c) }
func (a *App) TxStatusWebhook(c *gin.Context)   { a.Controller.TxStatusWebhook(c) }
func (a *App) TxTimeoutWebhook(c *gin.Context)  { a.Controller.TxTimeoutWebhook(c) }
