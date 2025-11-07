package config

import (
	"os"
	"time"

	"github.com/parts-pile/site/logger"
)

const (
	// Server configuration
	ServerUploadLimit        = 20 * 1024 * 1024 // 20 MB
	ServerRedirectDelay      = 1 * time.Second
	ServerRateLimitMax       = 600
	ServerRateLimitExp       = 1 * time.Minute
	RegistrationRateLimitMax = 3                // Allow 3 registration attempts per IP
	RegistrationRateLimitExp = 15 * time.Minute // Within 15 minutes

	// Backblaze B2 configuration
	B2TokenCacheDuration   = 55 * time.Minute
	B2TokenCacheCleanup    = 10 * time.Minute
	B2DownloadTokenExpiry  = 3600 // seconds (1 hour)
	B2AuthEndpoint         = "https://api.backblazeb2.com/b2api/v2/b2_authorize_account"
	B2DownloadAuthEndpoint = "/b2api/v2/b2_get_download_authorization"

	// Qdrant vector database configuration
	QdrantPort       = 6334
	QdrantMaxRetries = 10
	QdrantRetryDelay = 1 * time.Second

	// Qdrant vector search configuration
	QdrantSearchInitialK          = 200 // Number of results to fetch from Qdrant for tree view
	QdrantSearchPageSize          = 10  // Number of results per page for list/grid views
	QdrantSearchThreshold         = 0.6 // Similarity threshold for filtering results (0.0 to 1.0)
	QdrantTTL                     = 10 * time.Minute
	QdrantProcessingQueueSize     = 100
	QdrantProcessingSleepInterval = 100 * time.Millisecond
	QdrantUserEmbeddingLimit      = 10

	// Grok API configuration
	GrokAPIURL = "https://api.x.ai/v1/chat/completions"
	GrokModel  = "grok-3-mini"

	// Gemini API configuration
	GeminiEmbeddingModel      = "gemini-embedding-001"
	GeminiEmbeddingDimensions = 3072

	// Ad configuration
	MaxImagesPerAd        = 20
	MaxRockCount          = 2 // Maximum rock count to allow in search results
	DefaultAdCategoryName = "Car & Truck Parts"

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

	// Backblaze B2 configuration
	B2MasterKeyID = getEnvWithDefault("BACKBLAZE_MASTER_KEY_ID", "")
	B2KeyID       = getEnvWithDefault("BACKBLAZE_KEY_ID", "")
	B2AppKey      = getEnvWithDefault("BACKBLAZE_APP_KEY", "")
	B2BucketID    = getEnvWithDefault("B2_BUCKET_ID", "")
	B2BucketName  = getEnvWithDefault("B2_BUCKET_NAME", "")

	// Qdrant vector database configuration
	QdrantHost       = getEnvWithDefault("QDRANT_HOST", "")
	QdrantAPIKey     = getEnvWithDefault("QDRANT_API_KEY", "")
	QdrantCollection = getEnvWithDefault("QDRANT_COLLECTION", "")

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

	// JWT configuration
	JWTSecret = []byte(getEnvWithDefault("JWT_SECRET", ""))

	// Server configuration
	ServerPort = getEnvWithDefault("PORT", "8000")
	BaseURL    = getEnvWithDefault("BASE_URL", "http://localhost:8000")

	// Cookie security - set secure flag based on environment
	CookieSecure = baseURLIsSecure()

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

func baseURLIsSecure() bool {
	return len(BaseURL) > 5 && BaseURL[:5] == "https"
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

	// Warn about insecure defaults in production
	if len(BaseURL) >= 5 && BaseURL[:5] != "https" &&
		os.Getenv("ENVIRONMENT") != "development" {
		logger.Warn("BaseURL is not HTTPS", "baseURL", BaseURL)
	}
}
