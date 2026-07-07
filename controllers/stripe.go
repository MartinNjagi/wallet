package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"

	"wallet/data"
	"wallet/models"
)

// InitiateStripeTopUp creates a Stripe Checkout Session
func (ctr *Controller) InitiateStripeTopUp(ctx *gin.Context) {
	clientID := ctx.MustGet("client_id").(uint)

	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusBadRequest, Message: err.Error()})
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// 1. Calculate Credits
	var config models.ClientBillingConfig
	ctr.DB.Where("client_id = ?", clientID).FirstOrCreate(&config, models.ClientBillingConfig{ClientID: clientID, BaseSmsRate: 1.0})
	credits := int64(req.Amount / config.BaseSmsRate)

	// 2. Create Stripe Session
	domain := os.Getenv("FRONTEND_URL") // e.g., https://dashboard.yourdomain.com
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID:  stripe.String(fmt.Sprintf("%d", clientID)), // Crucial: Attach ClientID securely
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("kes"), // Or USD based on your config
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Wallet Top-up"),
					},
					UnitAmount: stripe.Int64(int64(req.Amount * 100)), // Stripe expects cents
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(domain + "/wallet?success=true&session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(domain + "/wallet?canceled=true"),
	}

	s, err := session.New(params)
	if err != nil {
		SendJSON(ctx, data.APIResponse{Status: http.StatusInternalServerError, Message: "Failed to initialize Stripe"})
		return
	}

	// 3. Save Pending Transaction to DB
	stripeTx := models.StripeTransaction{
		ClientID:  clientID,
		SessionID: s.ID,
		Amount:    req.Amount,
		Credits:   credits,
		Currency:  "KES",
		Status:    "PENDING",
	}
	ctr.DB.Create(&stripeTx)

	// Return the checkout URL to the frontend
	SendJSON(ctx, data.APIResponse{
		Status: http.StatusOK,
		Data: map[string]string{
			"checkout_url": s.URL,
		},
	})
}

// StripeWebhook processes background notifications from Stripe
func (ctr *Controller) StripeWebhook(ctx *gin.Context) {
	const MaxBodyBytes = int64(65536)
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, MaxBodyBytes)
	payload, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error reading request body"})
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	signatureHeader := ctx.GetHeader("Stripe-Signature")

	// 1. Verify Stripe Signature (Critical for security)
	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	// 2. Handle successful payment
	if event.Type == "checkout.session.completed" {
		var checkOutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkOutSession)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})
			return
		}

		if checkOutSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
			// Find our transaction record
			var stripeTx models.StripeTransaction
			if err := ctr.DB.Where("session_id = ?", checkOutSession.ID).First(&stripeTx).Error; err != nil {
				ctx.JSON(http.StatusOK, gin.H{"status": "ignored - not found"})
				return
			}

			if stripeTx.Status != "PENDING" {
				ctx.JSON(http.StatusOK, gin.H{"status": "already processed"})
				return
			}

			// Credit the Wallet securely using existing idempotent logic
			err = ctr.ApplyWalletOperation(ctr.DB, data.WalletOperation{
				ClientID:    stripeTx.ClientID,
				Action:      data.WalletActionCredit,
				Credits:     stripeTx.Credits,
				Type:        "STRIPE_", // E.g., STRIPE_cs_test_a1b2c3
				Description: "Stripe Card Top Up",
				Reference:   checkOutSession.ID,
				FiatPaid:    &stripeTx.Amount,
				Currency:    StringPtr(stripeTx.Currency),
			})

			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "ledger update failed"})
				return
			}

			// Mark successful
			ctr.DB.Model(&stripeTx).Updates(map[string]interface{}{
				"status":         "SUCCESS",
				"payment_intent": checkOutSession.PaymentIntent.ID,
			})
		}
	}

	// Acknowledge receipt
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}
