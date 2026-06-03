package routers

import "github.com/gin-gonic/gin"

// --- Public Webhooks ---

// @Summary M-Pesa IPN Webhook
// @Tags Webhooks
func (a *App) MpesaWebhook(c *gin.Context) { a.Controller.MpesaWebhook(c) }
