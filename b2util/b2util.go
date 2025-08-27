package b2util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/config"
)

var tokenCache *cache.Cache[string]
var fileExistsCache *cache.Cache[bool]

// Init initializes the B2 cache. This should be called during application startup.
// If initialization fails, the application should exit.
func Init() error {
	var err error
	tokenCache, err = cache.New[string](func(value string) int64 {
		return int64(len(value))
	}, "B2 Token Cache")
	if err != nil {
		return err
	}

	fileExistsCache, err = cache.New[bool](func(value bool) int64 {
		return 1 // Boolean values have size 1
	}, "B2 File Exists Cache")
	return err
}

// GetB2DownloadTokenForPrefixCached returns a cached B2 download authorization token for a given ad directory prefix (e.g., "22/")
func GetB2DownloadTokenForPrefixCached(prefix string) (string, error) {
	if token, found := tokenCache.Get(prefix); found {
		return token, nil
	}
	token, err := getB2DownloadTokenForPrefix(prefix)
	if err != nil {
		return "", err
	}
	// Set TTL to be slightly less than the actual token expiry to ensure we refresh before expiration
	// B2 tokens expire in 1 hour (3600 seconds), so we'll cache for 50 minutes (3000 seconds)
	ttl := time.Duration(config.B2DownloadTokenExpiry-600) * time.Second
	tokenCache.SetWithTTL(prefix, token, int64(len(token)), ttl)
	return token, nil
}

// GetCacheStats returns cache statistics for admin monitoring
func GetCacheStats() map[string]interface{} {
	stats := tokenCache.Stats()
	fileExistsStats := fileExistsCache.Stats()

	// Add B2-specific TTL information
	stats["b2_token_ttl_seconds"] = config.B2DownloadTokenExpiry
	stats["b2_cache_ttl_seconds"] = config.B2DownloadTokenExpiry - 600 // 50 minutes
	stats["b2_cache_ttl_formatted"] = fmt.Sprintf("%.1f minutes", float64(config.B2DownloadTokenExpiry-600)/60)
	stats["b2_token_expiry_formatted"] = fmt.Sprintf("%.1f minutes", float64(config.B2DownloadTokenExpiry)/60)

	// Add file exists cache stats
	stats["b2_file_exists_cache"] = fileExistsStats

	return stats
}

// ClearCache clears all cached tokens
func ClearCache() {
	tokenCache.Clear()
	fileExistsCache.Clear()
}

// ForceRefreshToken forces a refresh of the token for a specific prefix
func ForceRefreshToken(prefix string) (string, error) {
	token, err := getB2DownloadTokenForPrefix(prefix)
	if err != nil {
		return "", err
	}
	// Set TTL to be slightly less than the actual token expiry to ensure we refresh before expiration
	// B2 tokens expire in 1 hour (3600 seconds), so we'll cache for 50 minutes (3000 seconds)
	ttl := time.Duration(config.B2DownloadTokenExpiry-600) * time.Second
	tokenCache.SetWithTTL(prefix, token, int64(len(token)), ttl)
	return token, nil
}

// getB2DownloadTokenForPrefix returns a B2 download authorization token for a given ad directory prefix (e.g., "22/")
func getB2DownloadTokenForPrefix(prefix string) (string, error) {
	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey
	bucketID := config.B2BucketID
	if accountID == "" || appKey == "" || keyID == "" || bucketID == "" {
		return "", fmt.Errorf("B2 credentials not set")
	}
	req, _ := http.NewRequest("GET", config.B2AuthEndpoint, nil)
	req.SetBasicAuth(keyID, appKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("B2 auth error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("B2 auth failed: %s", resp.Status)
	}
	var authResp struct {
		APIURL    string `json:"apiUrl"`
		AuthToken string `json:"authorizationToken"`
		AccountID string `json:"accountId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("B2 auth decode error: %w", err)
	}
	apiURL := authResp.APIURL + config.B2DownloadAuthEndpoint
	expires := int64(config.B2DownloadTokenExpiry) // 1 hour
	body, _ := json.Marshal(map[string]interface{}{
		"bucketId":               bucketID,
		"fileNamePrefix":         prefix,
		"validDurationInSeconds": expires,
	})
	req2, _ := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	req2.Header.Set("Authorization", authResp.AuthToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return "", fmt.Errorf("B2 get_download_authorization error: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("B2 get_download_authorization failed: %s", resp2.Status)
	}
	var tokenResp struct {
		AuthorizationToken string `json:"authorizationToken"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("B2 token decode error: %w", err)
	}
	return tokenResp.AuthorizationToken, nil
}

// CheckIfAdHasImagesOnB2 checks if any image files exist on B2 for a given ad ID
func CheckIfAdHasImagesOnB2(adID int) bool {
	// Check cache first
	cacheKey := fmt.Sprintf("ad_%d", adID)
	if exists, found := fileExistsCache.Get(cacheKey); found {
		log.Printf("[B2] Cache hit for ad %d: files exist = %v (cache key: %s)", adID, exists, cacheKey)
		return exists
	}

	log.Printf("[B2] Cache miss for ad %d, checking B2 API (cache key: %s)", adID, cacheKey)

	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey
	bucketName := config.B2BucketName
	if accountID == "" || appKey == "" || keyID == "" || bucketName == "" {
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}

	// Get auth token first
	req, _ := http.NewRequest("GET", config.B2AuthEndpoint, nil)
	req.SetBasicAuth(keyID, appKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}

	var authResp struct {
		APIURL    string `json:"apiUrl"`
		AuthToken string `json:"authorizationToken"`
		AccountID string `json:"accountId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}

	// List files with the ad ID prefix
	listURL := authResp.APIURL + "/b2api/v2/b2_list_file_names"
	prefix := fmt.Sprintf("%d/", adID)
	body, _ := json.Marshal(map[string]interface{}{
		"bucketName":   bucketName,
		"prefix":       prefix,
		"maxFileCount": 1, // We only need to know if any files exist
		"delimiter":    "",
	})

	log.Printf("[B2] Listing files for ad %d with prefix: %s, bucket: %s", adID, prefix, bucketName)
	log.Printf("[B2] API URL: %s", listURL)

	req2, _ := http.NewRequest("POST", listURL, bytes.NewReader(body))
	req2.Header.Set("Authorization", authResp.AuthToken)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}

	var listResp struct {
		Files []struct{} `json:"files"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listResp); err != nil {
		log.Printf("[B2] ERROR: Failed to decode B2 list response for ad %d: %v", adID, err)
		// Cache the result (false) for 1 hour to avoid repeated failures
		fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Hour)
		return false
	}

	// Check if any files were found
	hasFiles := len(listResp.Files) > 0
	log.Printf("[B2] Ad %d has %d files on B2: %v", adID, len(listResp.Files), hasFiles)

	// Cache the result for 1 hour
	fileExistsCache.SetWithTTL(cacheKey, hasFiles, 1, time.Hour)

	return hasFiles
}

// ClearFileExistsCacheForAd clears the file exists cache for a specific ad ID
func ClearFileExistsCacheForAd(adID int) {
	cacheKey := fmt.Sprintf("ad_%d", adID)
	// Force a fresh check by setting a very short TTL
	// This will ensure the next call to CheckIfAdHasImagesOnB2 makes a fresh API call
	fileExistsCache.SetWithTTL(cacheKey, false, 1, time.Microsecond)
	log.Printf("[B2] Cleared file exists cache for ad %d (cache key: %s)", adID, cacheKey)
}

// ForceRefreshFileExistsCache forces a refresh of the file existence cache for a specific ad ID
func ForceRefreshFileExistsCache(adID int) {
	cacheKey := fmt.Sprintf("ad_%d", adID)
	// Set to true with very short TTL to force a fresh check
	fileExistsCache.SetWithTTL(cacheKey, true, 1, time.Microsecond)
	log.Printf("[B2] Force refreshed file exists cache for ad %d (cache key: %s)", adID, cacheKey)
}
