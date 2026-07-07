package connections

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"wallet/data"
	"wallet/models"
)

func InitDB(cfg *data.AppConfig) *gorm.DB {

	dsn := buildDSN(cfg)
	logrus.Infof("DB_HOST=%s DB_PORT=%s ENV=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("APP_ENV"),
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("DB connection failed: %v", err)
	}

	DB = db
	logrus.Info("✓ Database connected")

	autoMigrate(DB)

	return DB
}

func buildDSN(cfg *data.AppConfig) string {

	var tlsPart string

	switch cfg.Env {
	case "local":
		tlsPart = ""

	case "development", "staging":
		tlsPart = "&tls=skip-verify"

	case "production":
		tlsPart = "&tls=true"
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		tlsPart,
	)
}

func autoMigrate(DB *gorm.DB) {
	// RBAC-Compliant AutoMigrate
	// We only migrate the core Identity and RBAC models
	err := DB.AutoMigrate(
		&models.Wallet{},
		&models.WalletTransaction{},
		&models.ClientBillingConfig{},
		&models.MpesaTransaction{},
		&models.C2BTransaction{}, &models.STKPushRequest{},
		&models.BankTransaction{}, &models.StripeTransaction{},
		&models.MpesaDispute{},
		&models.AuditLog{},
	)

	if err != nil {
		logrus.Fatalf("Database migration failed: %v", err)
	}

	logrus.Info("Database initialized and migrated to Pure RBAC schema.")
}
