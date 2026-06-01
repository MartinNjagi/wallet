package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wallet/connections"
	"wallet/docs"
	routers "wallet/router"
)

// @title           identity-service API Backend
// @version         1.0
// @description     This is a REST server for a Gin-based application.
// @termsOfService  https://dreamhubtech.com/terms/

// @contact.name   API Support
// @contact.url    https://www.dreamhubtech.com/support
// @contact.email  support@dreamhubtech.com

// @license.name  Apache 2.0
// @license.url   https://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath  /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {

	// ----- 1. Configure Context -----
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Config & Logger First
	cfg := connections.InitConfig()
	connections.LoadLogging()
	logrus.Infof("Loaded config: environment=%s", cfg.Env)

	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	docs.SwaggerInfo.Schemes = []string{"https"}
	envErr := os.Setenv("TZ", "Africa/Nairobi")
	if envErr != nil {
		logrus.Warnf("Failed to set environment variable TZ: %v", envErr)
	}

	// Initialize Connections
	db := connections.InitDB(cfg)
	rdc := connections.InitRedis()

	// Initialize Application Container
	var app routers.App
	app.Initialize(ctx, cfg, db, rdc)

	// Start Gin Server
	r := app.SetupRouter()

	addr := cfg.ServerHost + ":" + cfg.ServerPort
	logrus.Infof("🚀 Identity API starting on %s", addr)

	// Initialize standard HTTP Server
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Run in a goroutine
	go func() {
		// ListenAndServe returns http.ErrServerClosed when Shutdown() is called
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Warn("Shutting down server...")

	// Tell the server it has 5 seconds to finish active requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logrus.Fatal("Server forced to shutdown: ", err)
	}

	// Cancel your global background context for things like the Reconciliation Cron
	cancel()

	logrus.Info("Server stopped cleanly ✅")
}
