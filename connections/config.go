package connections

import (
	"io"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"wallet/data"
)

func InitConfig() *data.AppConfig {

	env := os.Getenv("APP_ENV")

	// ONLY load .env locally
	if env == "local" || env == "development" {
		_ = godotenv.Load()
	}

	Config = &data.AppConfig{
		ServerPort: mustGetEnv("SERVER_PORT"),
		ServerHost: mustGetEnv("SERVER_HOST"),

		DBHost:     mustGetEnv("DB_HOST"),
		DBPort:     mustGetEnv("DB_PORT"),
		DBUser:     mustGetEnv("DB_USER"),
		DBPassword: mustGetEnv("DB_PASSWORD"),
		DBName:     mustGetEnv("DB_NAME"),

		RedisAddr:     getEnv("REDIS_ADDR", ""),
		RedisHost:     mustGetEnv("REDIS_HOST"),
		RedisPort:     mustGetEnv("REDIS_PORT"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		JWTSecret:            mustGetEnv("JWT_SECRET"),
		InternalServiceToken: mustGetEnv("INTERNAL_SERVICE_TOKEN"),
		Env:                  mustGetEnv("APP_ENV"),
	}

	logrus.Info("✓ Config initialized")

	return Config
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		logrus.Fatalf("missing required env: %s", key)
	}
	return v
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func LoadLogging() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logrus.SetOutput(os.Stdout)

	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logrus.Info("logging mode: stdout only")
		return
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		logrus.Warnf("failed to create log dir: %v", err)
		return
	}

	file, err := os.OpenFile(
		logDir+"/app.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		logrus.Warnf("failed to open log file: %v", err)
		return
	}

	logrus.SetOutput(io.MultiWriter(os.Stdout, file))
	logrus.Infof("logging mode: stdout + %s/app.log", logDir)
}
