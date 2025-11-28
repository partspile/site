package minio

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
)

var urlCache *cache.Cache[string]
var minioClient *minio.Client

// Init initializes the MinIO cache and client. This should be called during application startup.
// If initialization fails, the application should exit.
func Init() error {
	var err error
	urlCache, err = cache.New(func(value string) int64 {
		return int64(len(value))
	}, "MinIO Presigned URL Cache")
	if err != nil {
		return err
	}

	// Initialize MinIO client
	if config.MinIOAPIURL == "" {
		return fmt.Errorf("MINIO_API_URL environment variable not set")
	}
	if config.MinIOUsername == "" {
		return fmt.Errorf("MINIO_USERNAME environment variable not set")
	}
	if config.MinIOPassword == "" {
		return fmt.Errorf("MINIO_PASSWORD environment variable not set")
	}

	// Parse the API URL to extract endpoint
	// If URL doesn't have a scheme, add http:// as default
	apiURL := config.MinIOAPIURL
	if !strings.Contains(apiURL, "://") {
		apiURL = "http://" + apiURL
	}

	endpointURL, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("invalid MinIO API URL %q: %w", config.MinIOAPIURL, err)
	}

	// Validate that we have a host
	endpoint := endpointURL.Host
	if endpoint == "" {
		// If Host is still empty after parsing, the URL format is invalid
		return fmt.Errorf("MinIO API URL missing host: %q (expected format: http://host:port or https://host:port, got: %q)", config.MinIOAPIURL, apiURL)
	}

	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinIOUsername, config.MinIOPassword, ""),
		Secure: endpointURL.Scheme == "https",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return nil
}

// GetMinIOPresignedURLForPrefixCached returns a cached MinIO presigned URL for a given image path
func GetMinIOPresignedURLForPrefixCached(adID, imageIndex int, size string) (string, error) {
	imagePath := filepath.Join(fmt.Sprintf("%d", adID), fmt.Sprintf("%d-%s.webp", imageIndex, size))
	cacheKey := imagePath

	if url, found := urlCache.Get(cacheKey); found {
		return url, nil
	}

	presignedURL, err := getMinIOPresignedURL(imagePath)
	if err != nil {
		logger.Error("Failed to generate MinIO presigned URL",
			"error", err, "imagePath", imagePath, "adID", adID, "imageIndex", imageIndex, "size", size)
		return "", err
	}

	// Set TTL to be slightly less than the actual URL expiry to ensure we refresh before expiration
	// MinIO presigned URLs expire in 1 hour (3600 seconds), so we'll cache for 50 minutes (3000 seconds)
	ttl := time.Duration(config.MinIOPresignedURLExpiry-600) * time.Second
	urlCache.SetWithTTL(cacheKey, presignedURL, int64(len(presignedURL)), ttl)
	return presignedURL, nil
}

// GetCacheStats returns cache statistics for admin monitoring
func GetCacheStats() map[string]interface{} {
	stats := urlCache.Stats()

	// Add MinIO-specific TTL information
	stats["minio_url_ttl_seconds"] = config.MinIOPresignedURLExpiry
	stats["minio_cache_ttl_seconds"] = config.MinIOPresignedURLExpiry - 600 // 50 minutes
	stats["minio_cache_ttl_formatted"] = fmt.Sprintf("%.1f minutes", float64(config.MinIOPresignedURLExpiry-600)/60)
	stats["minio_url_expiry_formatted"] = fmt.Sprintf("%.1f minutes", float64(config.MinIOPresignedURLExpiry)/60)

	return stats
}

// ClearCache clears all cached URLs and returns updated stats
func ClearCache() map[string]interface{} {
	urlCache.Clear()
	return GetCacheStats()
}

// ForceRefreshURL forces a refresh of the presigned URL for a specific image path
func ForceRefreshURL(adID, imageIndex int, size string) (string, error) {
	imagePath := filepath.Join(fmt.Sprintf("%d", adID), fmt.Sprintf("%d-%s.webp", imageIndex, size))
	cacheKey := imagePath

	presignedURL, err := getMinIOPresignedURL(imagePath)
	if err != nil {
		return "", err
	}

	// Set TTL to be slightly less than the actual URL expiry to ensure we refresh before expiration
	ttl := time.Duration(config.MinIOPresignedURLExpiry-600) * time.Second
	urlCache.SetWithTTL(cacheKey, presignedURL, int64(len(presignedURL)), ttl)
	return presignedURL, nil
}

// getMinIOPresignedURL generates a presigned URL for a given image path
func getMinIOPresignedURL(imagePath string) (string, error) {
	if minioClient == nil {
		return "", fmt.Errorf("MinIO client not initialized")
	}

	if config.MinIOBucketName == "" {
		return "", fmt.Errorf("MinIO bucket name not set")
	}

	ctx := context.Background()
	expires := time.Duration(config.MinIOPresignedURLExpiry) * time.Second

	presignedURL, err := minioClient.PresignedGetObject(ctx, config.MinIOBucketName, imagePath, expires, nil)
	if err != nil {
		logger.Error("MinIO PresignedGetObject failed",
			"error", err, "bucket", config.MinIOBucketName, "imagePath", imagePath)
		return "", fmt.Errorf("MinIO presigned URL error: %w", err)
	}

	return presignedURL.String(), nil
}

// GetMinIOImageURL returns the complete MinIO presigned URL for a given ad and image index
func GetMinIOImageURL(adID, imageIndex int, size string) (string, error) {
	return GetMinIOPresignedURLForPrefixCached(adID, imageIndex, size)
}

// GetMinIOClient returns the MinIO client instance (for uploads)
func GetMinIOClient() *minio.Client {
	return minioClient
}
