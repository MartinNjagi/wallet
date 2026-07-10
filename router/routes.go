package routers

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"wallet/middleware"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"wallet/controllers"
	"wallet/data"
)

// App acts as the central DI container for your routers
type App struct {
	Ctx    context.Context
	Config *data.AppConfig
	DB     *gorm.DB
	Redis  *redis.Client

	Controller *controllers.Controller
}

// Initialize populates the App instance with active connections
func (a *App) Initialize(ctx context.Context, cfg *data.AppConfig, db *gorm.DB, rdc *redis.Client, s3Client *s3.Client) {
	a.Ctx = ctx
	a.Config = cfg
	a.DB = db
	a.Redis = rdc

	// Inject all connections into the Controller
	a.Controller = &controllers.Controller{
		Ctx:      ctx,
		DB:       db,
		Redis:    rdc,
		Config:   cfg,
		S3Client: s3Client,
	}
}
func (a *App) SetupRouter() *gin.Engine {
	gin.DisableConsoleColor()
	r := gin.New()

	err := r.SetTrustedProxies([]string{
		"127.0.0.1",
		"::1",
	})
	if err != nil {
		logrus.Fatal(err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CaptureRawBodyMiddleware())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Replace with specific BFF origins in prod
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PATCH", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Signature"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Pass the initialized App instance into the router registrar
	RegisterRoutes(r, a)

	return r
}

// RegisterRoutes bootstraps all core IAM and Wallet modules.
func RegisterRoutes(r *gin.Engine, app *App) {
	// ==========================================
	// 1. PUBLIC WEBHOOKS (No HMAC, No Auth)
	// ==========================================
	webhooks := r.Group("/webhooks/:secret")
	webhooks.Use(middleware.DarajaWebhookGuard())
	{
		webhooks.POST("/mpesa", app.MpesaWebhook)              // STK Callback
		webhooks.POST("/mpesa/validate", app.MpesaValidation)  // C2B Validate
		webhooks.POST("/mpesa/confirm", app.MpesaConfirmation) // C2B Confirm

		// Transaction Status Webhooks
		webhooks.POST("/mpesa/tx-status", app.TxStatusWebhook)
		webhooks.POST("/mpesa/tx-timeout", app.TxTimeoutWebhook)
	}
	r.POST("/webhooks/stripe", app.Controller.StripeWebhook)
	// ==========================================
	// 2. SECURE INTERNAL & BFF API
	// ==========================================
	api := r.Group("/api/v1")
	// Enforce HMAC Signature for all requests coming from BFF or Internal Microservices
	api.Use(middleware.VerifySignature(app.Config.InternalServiceToken, app.Redis))

	// --- Internal M2M Microservice Routes (SMS Engine) ---
	// No JWT needed here, the HMAC signature (above) acts as the Service Token
	internal := api.Group("/internal")
	app.RegisterInternalRoutes(internal)

	// --- Protected Dashboard/IAM Routes ---
	// Enforce JWT validation and Redis session checking
	iam := api.Group("")
	iam.Use(middleware.JWTAuthRedis(app.Redis, []byte(app.Config.JWTSecret)))
	app.RegisterWalletRoutes(iam) // Attach wallet dashboard routes
}

// --- Route Registrars ---

// RegisterInternalRoutes handles M2M backend communication
func (a *App) RegisterInternalRoutes(rg *gin.RouterGroup) {
	wallet := rg.Group("/wallet")
	{
		wallet.POST("/balance", a.InternalBalanceCampaign)
		wallet.POST("/deduct", a.InternalDeductCampaign)
		wallet.POST("/refund", a.InternalRefundCampaign)
	}
}

// RegisterWalletRoutes handles the Wallet Dashboard & Admin actions
func (a *App) RegisterWalletRoutes(rg *gin.RouterGroup) {
	wallet := rg.Group("/wallet")
	wallet.Use(middleware.RoleAuth("read wallet"))
	{
		wallet.GET("/balance", a.GetBalance)
		wallet.GET("/ledger", a.ListLedger)
		wallet.POST("/topup", a.InitiateTopUp)
		wallet.POST("/bank-transfer", a.SubmitBankTransfer) // <-- NEW
		// NEW: Client submits a claim for a missing payment
		wallet.POST("/claim-mpesa", a.ClaimMpesaPayment)
		wallet.POST("/stripe-topup", a.InitiateStripeTopUp)
	}

	// SuperAdmin Wallet Tools
	adminWallet := rg.Group("/admin/wallet")
	adminWallet.Use(middleware.RoleAuth("manage wallet"))
	{
		adminWallet.POST("/adjust", a.ManualAdjustment)
		adminWallet.GET("/bank-transfers", a.ListBankTransfers)
		adminWallet.POST("/bank-transfer/:id/approve", a.ApproveBankTransfer)
	}
	adminWallet.Use(middleware.RoleAuth("manage billing")) // Restrict to financial admins
	{
		adminWallet.GET("/config/:id", a.GetClientConfig)
		adminWallet.PUT("/config/:id", a.UpdateClientConfig)
	}
	adminWallet.Use(middleware.RoleAuth("read billing"))
	{
		adminWallet.GET("/summary", a.AdminWalletSummary)
	}
}
