package b2util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/config"
)

var tokenCache *cache.Cache[string]

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

	// Add B2-specific TTL information
	stats["b2_token_ttl_seconds"] = config.B2DownloadTokenExpiry
	stats["b2_cache_ttl_seconds"] = config.B2DownloadTokenExpiry - 600 // 50 minutes
	stats["b2_cache_ttl_formatted"] = fmt.Sprintf("%.1f minutes", float64(config.B2DownloadTokenExpiry-600)/60)
	stats["b2_token_expiry_formatted"] = fmt.Sprintf("%.1f minutes", float64(config.B2DownloadTokenExpiry)/60)

	return stats
}

// ClearCache clears all cached tokens
func ClearCache() {
	tokenCache.Clear()
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
