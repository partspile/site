package config

import (
	"os"
	"time"
)

const (
	// RedirectDelay is the time to wait before redirecting the user after a successful action.
	RedirectDelay = 1 * time.Second
	// B2TokenCacheDuration is how long to cache B2 download tokens before refreshing.
	B2TokenCacheDuration = 55 * time.Minute
	// B2TokenCacheCleanup is how often the cache cleanup runs for expired tokens.
	B2TokenCacheCleanup = 10 * time.Minute
	// B2DownloadTokenExpiry is the validity duration (in seconds) for B2 download tokens requested from Backblaze.
	B2DownloadTokenExpiry = 3600 // seconds (1 hour)

	// Vector search configuration
	VectorSearchInitialK  = 200 // Number of results to fetch from Qdrant for tree view
	VectorSearchPageSize  = 10  // Number of results per page for list/grid views
	VectorSearchThreshold = 0.7 // Similarity threshold for filtering results (0.0 to 1.0)

	// Grok API configuration
	GrokAPIURL = "https://api.x.ai/v1/chat/completions"
	GrokModel  = "grok-3-mini"
)

// Global configuration variables
var (
	// Database configuration
	DatabaseURL = getEnvWithDefault("DATABASE_URL", "project.db")

	// Backblaze B2 configuration
	BackblazeMasterKeyID = getEnvWithDefault("BACKBLAZE_MASTER_KEY_ID", "")
	BackblazeKeyID       = getEnvWithDefault("BACKBLAZE_KEY_ID", "")
	BackblazeAppKey      = getEnvWithDefault("BACKBLAZE_APP_KEY", "")
	B2BucketID           = getEnvWithDefault("B2_BUCKET_ID", "")

	// Vector database configuration
	QdrantHost       = getEnvWithDefault("QDRANT_HOST", "")
	QdrantAPIKey     = getEnvWithDefault("QDRANT_API_KEY", "")
	QdrantCollection = getEnvWithDefault("QDRANT_COLLECTION", "")

	// AI/ML API configuration
	GeminiAPIKey = getEnvWithDefault("GEMINI_API_KEY", "")
	GeminiModel  = getEnvWithDefault("GEMINI_MODEL", "embedding-001")
	GrokAPIKey   = getEnvWithDefault("GROK_API_KEY", "")

	// Server configuration
	Port = getEnvWithDefault("PORT", "8000")
)

// getEnvWithDefault returns the environment variable value or a default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
