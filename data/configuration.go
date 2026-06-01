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

	LogLevel      string
	LogsDir       string
	LogFilename   string
	LogMaxSize    int
	LogMaxBackups int
	LogMaxAge     int
	LogCompress   bool
	LogToConsole  bool
	LogBucket     string // GCS bucket name for logs

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
