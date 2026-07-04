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

	// ----- AWS S3 (file uploads) --------------------------------------------
	AWSRegion          string // AWS_REGION
	AWSAccessKeyID     string // AWS_ACCESS_KEY_ID
	AWSSecretAccessKey string // AWS_SECRET_ACCESS_KEY
	S3Bucket           string // S3_BUCKET

	// ----- Minio S3 (file uploads) --------------------------------------------
	MinioEndpoint  string // AWS_REGION
	MinioAccessKey string // AWS_ACCESS_KEY_ID
	MinioSecretKey string // AWS_SECRET_ACCESS_KEY
	MinioBucket    string // S3_BUCKET

	// SECURE DETAILS
	PaybillNumber string
	SenderID      string
}
