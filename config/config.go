package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"
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
	MaxImagesPerAd = 20
	MaxRockCount   = 2 // Maximum rock count to allow in search results

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
	JWTSecret = getEnvWithDefault("JWT_SECRET", "")

	// Server configuration
	ServerPort = getEnvWithDefault("PORT", "8000")
	BaseURL    = getEnvWithDefault("BASE_URL", "http://localhost:8000")

	// Cookie security - set secure flag based on environment
	CookieSecure = baseURLIsSecure()
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

// ValidateJWTSecret validates that JWT secret is set and sufficiently strong
// Minimum requirements: at least 32 bytes (256 bits) of entropy
func ValidateJWTSecret() error {
	if JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required but not set")
	}

	// Require at least 32 bytes (256 bits) for HS256
	// Base64 encoding adds ~33% overhead, so 32 bytes = ~43 base64 chars
	minLength := 32
	if len(JWTSecret) < minLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters long for security", minLength)
	}

	return nil
}

// GenerateSecureJWTSecret generates a cryptographically secure random JWT secret
// Returns a base64-encoded 32-byte (256-bit) secret suitable for HS256
func GenerateSecureJWTSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// ValidateSecurityConfig validates critical security configuration
func ValidateSecurityConfig() error {
	if err := ValidateJWTSecret(); err != nil {
		return fmt.Errorf("security configuration error: %w", err)
	}
	return nil
}

// InitSecurityConfig initializes and validates security configuration
// Logs warnings for missing optional configs, fatal errors for required ones
func InitSecurityConfig() {
	// Validate JWT secret (required)
	if err := ValidateJWTSecret(); err != nil {
		log.Fatalf("[SECURITY] %v", err)
	}

	// Log security status
	log.Printf("[SECURITY] JWT secret configured (length: %d)", len(JWTSecret))

	// Warn about insecure defaults in production
	if len(BaseURL) >= 5 && BaseURL[:5] != "https" && os.Getenv("ENVIRONMENT") != "development" {
		log.Printf("[SECURITY] WARNING: BaseURL is not HTTPS (%s)", BaseURL)
	}
}
