package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rocky-ads/site/internal/logger"
)

const (
	// Server configuration
	ServerUploadLimit        = 25 * 1024 * 1024  // 25 MB per file
	ServerBodyLimit          = 150 * 1024 * 1024 // 150 MB total request body (for multiple files + form data)
	ServerRateLimitMax       = 600
	ServerRateLimitExp       = 1 * time.Minute
	RegistrationRateLimitMax = 20               // Registration attempts per IP
	RegistrationRateLimitExp = 15 * time.Minute // Within 15 minutes
	RecoveryRateLimitMax     = 20               // New recovery starts per IP
	RecoveryRateLimitExp     = 15 * time.Minute // Within 15 minutes
	LoginRateLimitMax        = 20               // Allow 20 login POSTs per IP
	LoginRateLimitExp        = 15 * time.Minute // Within 15 minutes
	LoginUserFailMax         = 10               // Failed auths per username
	LoginUserFailExp         = 15 * time.Minute // Username failure window
	RecoverySessionTTL       = 10 * time.Minute // Recover session / code lifetime
	RegistrationTicketTTL    = 10 * time.Minute // Post-OTP registration ticket lifetime
	OTPStartMinInterval      = 60 * time.Second // Min time between OTP starts per phone
	OTPStartMaxPerHour       = 5                // Max OTP starts per phone per hour

	// MinIO configuration
	MinIOPresignedPutExpiry = 15 * time.Minute
	MinIOPresignedGetExpiry = 24 * time.Hour
	MinIOObjectCacheControl = "public, max-age=86400"

	// Vector search configuration
	SearchPageSize             = 20  // Number of results per page for list/grid views
	SearchThreshold            = 0.6 // Max cosine distance for vector search (pgvector <=>)
	VectorProcessingQueueSize  = 100
	VectorUserEmbeddingLimit   = 8
	VectorSiteEmbeddingLimit   = 8
	VectorSystemEmbeddingLimit = 100

	// Embedding cache TTL configuration
	VectorQueryEmbeddingCacheTTL = 1 * time.Hour // TTL for query embedding cache
	VectorUserEmbeddingCacheTTL  = 1 * time.Hour // TTL for user embedding cache
	VectorSiteEmbeddingCacheTTL  = 1 * time.Hour // TTL for site embedding cache

	// Grok API configuration
	GrokAPIURL          = "https://api.x.ai/v1/chat/completions"
	GrokModel           = "grok-4.3"
	GrokReasoningEffort = "none"

	// Geoapify geocoding
	GeoapifyGeocodeURL = "https://api.geoapify.com/v1/geocode/search"

	// Ollama embedding configuration
	OllamaEmbeddingModel      = "nomic-embed-text"
	OllamaEmbeddingDimensions = 768

	// Ad configuration
	MaxImagesPerAd            = 20
	MaxRockCount              = 2 // Maximum rock count to allow in search results
	MaxOutstandingRocks       = 3 // Rocks granted at signup; max a user can have thrown at once
	DefaultAdCategoryName     = "Car & Truck Parts"
	MaxAdTitleLength          = 64   // Maximum length for ad title
	MaxAdDescriptionLength    = 1000 // Maximum length for ad description
	AdExpireMonths            = 3    // create, reactivate, and renew lifetime
	AdExpireRenewWithinMonths = 1    // show Renew when this little time remains
	AdExpireSaleEndDelay      = 7 * 24 * time.Hour
	AdExpireWorkerInterval    = 1 * time.Hour

	// Password/Argon2 configuration
	Argon2Memory = 64 * 1024

	// HTMX scripts (self-hosted for privacy browsers that block third-party CDNs)
	HTMXURL    = "/js/htmx.min.js"
	HTMXSSEURL = "/js/htmx-ext-sse.min.js"

	// Public links
	GitHubRepoURL = "https://github.com/rocky-ads/site"

	// SSE configuration
	SSEChannelBufferSize = 100 // Buffer size for user event channels

	// SMS queue configuration
	SMSSuppressionWindowMinutes = 10              // Suppress SMS if sent within this many minutes
	SMSWorkerPollInterval       = 5 * time.Second // How often worker checks for pending notifications
	SMSQueueCleanupInterval     = 1 * time.Hour   // How often to cleanup old records
	SMSQueueRetentionHours      = 24              // How long to keep processed/suppressed records
)

// Global configuration variables
var (
	// PublicSiteURL — public base for SMS links and Twilio webhooks
	PublicSiteURL = trimURLBase(os.Getenv("PUBLIC_SITE_URL"))

	// Database configuration
	DatabaseURL = getEnvWithDefault("DATABASE_URL", "postgres://localhost:5432/rockyads?sslmode=disable")

	// Redis (shared rate limits). Required.
	// Example: redis://localhost:6379 or redis://red-xxx:6379
	RedisURL = getEnvWithDefault("REDIS_URL", "")

	// MinIO configuration
	MinIOAPIURL       = getEnvWithDefault("MINIO_API_URL", "")
	MinIOPublicURL    = getEnvWithDefault("MINIO_PUBLIC_URL", "")
	MinIORootUser     = getEnvWithDefault("MINIO_ROOT_USER", "")
	MinIORootPassword = getEnvWithDefault("MINIO_ROOT_PASSWORD", "")
	MinIOBucketName   = getEnvWithDefault("MINIO_BUCKET_NAME", "")

	// AI/ML API configuration
	GrokAPIKey     = getEnvWithDefault("GROK_API_KEY", "")
	GeoapifyAPIKey = getEnvWithDefault("GEOAPIFY_API_KEY", "")
	OllamaURL      = getEnvWithDefault("OLLAMA_URL", "http://localhost:11434")

	// SMS/Twilio configuration
	TwilioAccountSID       = getEnvWithDefault("TWILIO_ACCOUNT_SID", "")
	TwilioAuthToken        = getEnvWithDefault("TWILIO_AUTH_TOKEN", "")
	TwilioFromNumber       = getEnvWithDefault("TWILIO_FROM_NUMBER", "")
	TwilioVerifyServiceSID = getEnvWithDefault("TWILIO_VERIFY_SERVICE_SID", "")

	// Cloudflare Turnstile (bot gate before OTP start)
	TurnstileSiteKey   = getEnvWithDefault("TURNSTILE_SITE_KEY", "")
	TurnstileSecretKey = getEnvWithDefault("TURNSTILE_SECRET_KEY", "")

	// JWT configuration
	JWTSecret = []byte(getEnvWithDefault("JWT_SECRET", ""))

	// DB encryption (user name/phone; journals on restore / later at rest)
	DBEncryptionKey = getEncryptionKey("DB_ENCRYPTION_KEY")

	// Pepper for name_hash / phone_hash HMAC lookups (not the encryption key)
	DBHashPepper = getEncryptionKey("DB_HASH_PEPPER")

	// AES-256 key for opaque /u/{token} profile share links
	ShareSecret = getEncryptionKey("SHARE_SECRET")

	// Server configuration
	ServerPort   = getEnvWithDefault("PORT", "10000")
	ServerName   = os.Getenv("APP_NAME")
	ContactEmail = os.Getenv("CONTACT_EMAIL")

	// Public Bitcoin address for /donate (on-chain receive address).
	BitcoinDonateAddress = getEnvWithDefault("BITCOIN_DONATE_ADDRESS", "")

	// Test port - used when running tests to avoid conflicts with main server
	TestPort = getEnvWithDefault("PORT_TEST", "10001")

	// Cookie security - set LOCAL_DEVELOPMENT=true to relax cookie security for local HTTP access
	// When LOCAL_DEVELOPMENT is set, cookies will work over HTTP even if PUBLIC_SITE_URL is HTTPS
	CookieSecure = os.Getenv("LOCAL_DEVELOPMENT") != "true"

	// AllowTestRegistration skips SMS verification for +1555010xxxx phones (dev/test harness).
	AllowTestRegistration = os.Getenv("ALLOW_TEST_REGISTRATION") == "true"

	// Logging configuration
	LogLevel  = getEnvWithDefault("LOG_LEVEL", "info")
	LogFormat = getEnvWithDefault("LOG_FORMAT", "json")
	LogFile   = getEnvWithDefault("LOG_FILE", "")
)

// EffectiveRegistrationRateLimitMax returns the registration rate limit max attempts.
func EffectiveRegistrationRateLimitMax() int {
	if AllowTestRegistration {
		return 100
	}
	return RegistrationRateLimitMax
}

// EffectiveRecoveryRateLimitMax returns the recovery rate limit max attempts.
func EffectiveRecoveryRateLimitMax() int {
	if AllowTestRegistration {
		return 100
	}
	return RecoveryRateLimitMax
}

// EffectiveLoginRateLimitMax returns the login IP rate limit max attempts.
func EffectiveLoginRateLimitMax() int {
	if AllowTestRegistration {
		return 100
	}
	return LoginRateLimitMax
}

// EffectiveLoginUserFailMax returns the per-username failed-login cap.
func EffectiveLoginUserFailMax() int {
	if AllowTestRegistration {
		return 100
	}
	return LoginUserFailMax
}

// getEnvWithDefault returns the environment variable value or a default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func trimURLBase(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

// CanonicalURL is the absolute public URL for path (no query or fragment).
func CanonicalURL(path string) string {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if path == "" || path == "/" {
		return PublicSiteURL + "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return PublicSiteURL + path
}

// PublicHost is the hostname from PublicSiteURL.
func PublicHost() string {
	u, err := url.Parse(PublicSiteURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
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

// validateJWTSecret validates that JWT secret is set and sufficiently strong
// Minimum requirements: at least 32 bytes (256 bits) of entropy
func validateJWTSecret(secret []byte) error {
	if len(secret) == 0 {
		return fmt.Errorf("JWT_SECRET environment variable is required but not set")
	}

	// Require at least 32 bytes (256 bits) for HS256
	minLength := 32
	if len(secret) < minLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters long for security", minLength)
	}

	return nil
}

// SecurityCheck validates security configuration and logs warnings
// Logs fatal errors for required configs, warnings for insecure defaults
func SecurityCheck() {
	// Validate JWT secret (required)
	if err := validateJWTSecret(JWTSecret); err != nil {
		logger.Fatal("Security configuration error", "error", err.Error())
	}

	// Log security status
	logger.Info("JWT secret configured", "length", len(JWTSecret))

	// Validate DB encryption key
	dbKeyStr := os.Getenv("DB_ENCRYPTION_KEY")
	if dbKeyStr == "" {
		logger.Fatal("Security configuration error",
			"error", "DB_ENCRYPTION_KEY environment variable is required but not set")
	}
	if len(DBEncryptionKey) == 0 {
		logger.Fatal("Security configuration error",
			"error", "DB_ENCRYPTION_KEY is invalid base64 format")
	}
	if len(DBEncryptionKey) != 32 {
		logger.Fatal("DB_ENCRYPTION_KEY must be 32 bytes (256 bits)")
	}
	logger.Info("DB encryption key configured", "length", len(DBEncryptionKey))

	pepperStr := os.Getenv("DB_HASH_PEPPER")
	if pepperStr == "" {
		logger.Fatal("Security configuration error",
			"error", "DB_HASH_PEPPER environment variable is required but not set")
	}
	if len(DBHashPepper) == 0 {
		logger.Fatal("Security configuration error",
			"error", "DB_HASH_PEPPER is invalid base64 format")
	}
	if len(DBHashPepper) != 32 {
		logger.Fatal("DB_HASH_PEPPER must be 32 bytes (256 bits)")
	}
	if bytes.Equal(DBHashPepper, DBEncryptionKey) {
		logger.Fatal("Security configuration error",
			"error", "DB_HASH_PEPPER must differ from DB_ENCRYPTION_KEY")
	}
	logger.Info("DB hash pepper configured", "length", len(DBHashPepper))

	shareStr := os.Getenv("SHARE_SECRET")
	if shareStr == "" {
		logger.Fatal("Security configuration error",
			"error", "SHARE_SECRET environment variable is required but not set")
	}
	if len(ShareSecret) == 0 {
		logger.Fatal("Security configuration error",
			"error", "SHARE_SECRET is invalid base64 format")
	}
	if len(ShareSecret) != 32 {
		logger.Fatal("SHARE_SECRET must be 32 bytes (256 bits)")
	}
	if bytes.Equal(ShareSecret, DBEncryptionKey) ||
		bytes.Equal(ShareSecret, DBHashPepper) {
		logger.Fatal("Security configuration error",
			"error", "SHARE_SECRET must differ from DB_ENCRYPTION_KEY and DB_HASH_PEPPER")
	}
	logger.Info("Share secret configured", "length", len(ShareSecret))

	if AllowTestRegistration {
		logger.Warn("ALLOW_TEST_REGISTRATION is enabled: test phone signup skips SMS verification")
	}

	if PublicSiteURL == "" {
		logger.Fatal("Security configuration error",
			"error", "PUBLIC_SITE_URL environment variable is required but not set")
	}
	logger.Info("Public site URL configured", "url", PublicSiteURL)

	if ServerName == "" {
		logger.Fatal("Security configuration error",
			"error", "APP_NAME environment variable is required but not set")
	}
	logger.Info("App name configured", "name", ServerName)

	if ContactEmail == "" {
		logger.Fatal("Security configuration error",
			"error", "CONTACT_EMAIL environment variable is required but not set")
	}
	logger.Info("Contact email configured", "email", ContactEmail)
}
