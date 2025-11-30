package config

import (
	"encoding/base64"
	"os"
	"time"

	"github.com/parts-pile/site/logger"
)

const (
	// Server configuration
	ServerUploadLimit        = 25 * 1024 * 1024  // 25 MB per file
	ServerBodyLimit          = 150 * 1024 * 1024 // 150 MB total request body (for multiple files + form data)
	ServerRedirectDelay      = 1 * time.Second
	ServerRateLimitMax       = 600
	ServerRateLimitExp       = 1 * time.Minute
	RegistrationRateLimitMax = 3                // Allow 3 registration attempts per IP
	RegistrationRateLimitExp = 15 * time.Minute // Within 15 minutes

	// MinIO configuration
	MinIOPresignedURLExpiry = 3600 // seconds (1 hour)

	// Vector search configuration
	VectorSearchInitialK          = 200 // Number of results to fetch for tree view
	VectorSearchPageSize          = 20  // Number of results per page for list/grid views
	VectorSearchThreshold         = 0.6 // Similarity threshold for filtering results (0.0 to 1.0)
	VectorProcessingQueueSize     = 100
	VectorProcessingSleepInterval = 100 * time.Millisecond
	VectorUserEmbeddingLimit      = 30
	VectorSystemEmbeddingLimit    = 100

	// Embedding cache TTL configuration
	VectorQueryEmbeddingCacheTTL = 1 * time.Hour // TTL for query embedding cache
	VectorUserEmbeddingCacheTTL  = 1 * time.Hour // TTL for user embedding cache
	VectorSiteEmbeddingCacheTTL  = 1 * time.Hour // TTL for site embedding cache

	// Grok API configuration
	GrokAPIURL = "https://api.x.ai/v1/chat/completions"
	GrokModel  = "grok-3-mini"

	// Gemini API configuration
	GeminiEmbeddingModel      = "gemini-embedding-001"
	GeminiEmbeddingDimensions = 3072

	// Ad configuration
	MaxImagesPerAd         = 20
	MaxRockCount           = 2 // Maximum rock count to allow in search results
	DefaultAdCategoryName  = "Car & Truck Parts"
	MaxAdDescriptionLength = 1000 // Maximum length for ad description

	// Password/Argon2 configuration
	Argon2Memory = 64 * 1024

	// CDN URLs for external resources
	HTMXURL    = "https://unpkg.com/htmx.org@2.0.7"
	HTMXSSEURL = "https://unpkg.com/htmx-ext-sse@2.2.3/dist/sse.min.js"
)

// Global configuration variables
var (
	// Database configuration
	DatabaseURL = getEnvWithDefault("DATABASE_URL", "project.db")

	// MinIO configuration
	MinIOAPIURL     = getEnvWithDefault("MINIO_API_URL", "")
	MinIOUsername   = getEnvWithDefault("MINIO_USERNAME", "")
	MinIOPassword   = getEnvWithDefault("MINIO_PASSWORD", "")
	MinIOBucketName = getEnvWithDefault("MINIO_BUCKET_NAME", "")

	// AI/ML API configuration
	GeminiAPIKey = getEnvWithDefault("GEMINI_API_KEY", "")
	GrokAPIKey   = getEnvWithDefault("GROK_API_KEY", "")

	// SMS/Twilio configuration
	TwilioAccountSID = getEnvWithDefault("TWILIO_ACCOUNT_SID", "")
	TwilioAuthToken  = getEnvWithDefault("TWILIO_AUTH_TOKEN", "")
	TwilioFromNumber = getEnvWithDefault("TWILIO_FROM_NUMBER", "")

	// Twilio SendGrid email configuration
	TwilioSendGridAPIKey = getEnvWithDefault("TWILIO_SENDGRID_API_KEY", "")
	TwilioFromEmail      = getEnvWithDefault("TWILIO_FROM_EMAIL", "")

	// Twilio webhook URL - used for webhook callbacks and notification links
	TwilioWebhookURL = getTwilioWebhookURL("TWILIO_WEBHOOK_URL")

	// JWT configuration
	JWTSecret = []byte(getEnvWithDefault("JWT_SECRET", ""))

	// Message encryption configuration
	MessageEncryptionKey = getEncryptionKey("MESSAGE_ENCRYPTION_KEY")

	// User data encryption configuration
	UserEncryptionKey = getEncryptionKey("USER_ENCRYPTION_KEY")

	// Server configuration
	ServerPort = getEnvWithDefault("PORT", "10000")

	// Cookie security - set LOCAL_DEVELOPMENT=true to relax cookie security for local HTTP access
	// When LOCAL_DEVELOPMENT is set, cookies will work over HTTP even if TWILIO_WEBHOOK_URL is HTTPS
	CookieSecure = os.Getenv("LOCAL_DEVELOPMENT") != "true"

	// Logging configuration
	LogLevel  = getEnvWithDefault("LOG_LEVEL", "info")
	LogFormat = getEnvWithDefault("LOG_FORMAT", "json")
	LogFile   = getEnvWithDefault("LOG_FILE", "")
)

// getEnvWithDefault returns the environment variable value or a default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getTwilioWebhookURL returns the Twilio webhook URL using the following logic:
// 1. Use key if not empty
// 2. Otherwise, if RENDER is set, use RENDER_EXTERNAL_URL
// 3. Otherwise, return empty string
func getTwilioWebhookURL(key string) string {
	if url := os.Getenv(key); url != "" {
		return url
	}
	if os.Getenv("RENDER") != "" {
		if url := os.Getenv("RENDER_EXTERNAL_URL"); url != "" {
			return url
		}
	}
	return ""
}

// getEncryptionKey loads and decodes a base64-encoded encryption key from environment
// Returns empty slice if not set (validation happens in SecurityCheck)
func getEncryptionKey(envKey string) []byte {
	keyStr := os.Getenv(envKey)
	if keyStr == "" {
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil // Validation will catch this in SecurityCheck
	}
	return key
}

// SecurityCheck validates security configuration and logs warnings
// Logs fatal errors for required configs, warnings for insecure defaults
func SecurityCheck() {
	// Validate JWT secret (required)
	if len(JWTSecret) == 0 {
		logger.Fatal("Security configuration error",
			"error", "JWT_SECRET environment variable is required but not set")
	}

	// Require at least 32 bytes (256 bits) for HS256
	minLength := 32
	if len(JWTSecret) < minLength {
		logger.Fatal("Security configuration error",
			"error", "JWT_SECRET must be at least 32 characters long for security")
	}

	// Log security status
	logger.Info("JWT secret configured", "length", len(JWTSecret))

	// Validate message encryption key
	messageKeyStr := os.Getenv("MESSAGE_ENCRYPTION_KEY")
	if messageKeyStr == "" {
		logger.Fatal("Security configuration error",
			"error", "MESSAGE_ENCRYPTION_KEY environment variable is required but not set")
	}
	if len(MessageEncryptionKey) == 0 {
		logger.Fatal("Security configuration error",
			"error", "MESSAGE_ENCRYPTION_KEY is invalid base64 format")
	}
	if len(MessageEncryptionKey) != 32 {
		logger.Fatal("MESSAGE_ENCRYPTION_KEY must be 32 bytes (256 bits)")
	}
	logger.Info("Message encryption key configured", "length", len(MessageEncryptionKey))

	// Validate user data encryption key
	userKeyStr := os.Getenv("USER_ENCRYPTION_KEY")
	if userKeyStr == "" {
		logger.Fatal("Security configuration error",
			"error", "USER_ENCRYPTION_KEY environment variable is required but not set")
	}
	if len(UserEncryptionKey) == 0 {
		logger.Fatal("Security configuration error",
			"error", "USER_ENCRYPTION_KEY is invalid base64 format")
	}
	if len(UserEncryptionKey) != 32 {
		logger.Fatal("USER_ENCRYPTION_KEY must be 32 bytes (256 bits)")
	}
	logger.Info("User encryption key configured", "length", len(UserEncryptionKey))
}
