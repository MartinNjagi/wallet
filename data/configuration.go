package data

// AppConfig holds all application configuration
type AppConfig struct {
	// Server
	ServerPort string
	ServerHost string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisAddr     string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	//Service URLs
	IdentityServiceURL string
	SMSServiceURL      string
	WalletServiceURL   string
	SDPServiceURL      string

	// Security
	JWTSecret            string
	AllowedCORS          string
	InternalServiceToken string

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   int // seconds

	// Staging/Production
	Env string

	// SECURE DETAILS
	PaybillNumber string
	SenderID      string
}
