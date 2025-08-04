package b2util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/parts-pile/site/config"
	"github.com/patrickmn/go-cache"
)

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits   int64
	Misses int64
	Sets   int64
}

// B2Cache wraps the go-cache with statistics tracking
type B2Cache struct {
	cache *cache.Cache
	stats CacheStats
	mu    sync.RWMutex
}

var tokenCache = &B2Cache{
	cache: cache.New(config.B2TokenCacheDuration, config.B2TokenCacheCleanup),
}

// Get retrieves a value from cache and updates statistics
func (b *B2Cache) Get(key string) (interface{}, bool) {
	value, found := b.cache.Get(key)
	b.mu.Lock()
	if found {
		b.stats.Hits++
	} else {
		b.stats.Misses++
	}
	b.mu.Unlock()
	return value, found
}

// Set stores a value in cache and updates statistics
func (b *B2Cache) Set(key string, value interface{}, duration time.Duration) {
	b.cache.Set(key, value, duration)
	b.mu.Lock()
	b.stats.Sets++
	b.mu.Unlock()
}

// Flush clears all cached items
func (b *B2Cache) Flush() {
	b.cache.Flush()
}

// GetStats returns current cache statistics
func (b *B2Cache) GetStats() CacheStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}

// GetItems returns all cached items (for admin display)
func (b *B2Cache) GetItems() map[string]cache.Item {
	return b.cache.Items()
}

// GetB2DownloadTokenForPrefixCached returns a cached B2 download authorization token for a given ad directory prefix (e.g., "22/")
func GetB2DownloadTokenForPrefixCached(prefix string) (string, error) {
	if token, found := tokenCache.Get(prefix); found {
		return token.(string), nil
	}
	token, err := getB2DownloadTokenForPrefix(prefix)
	if err != nil {
		return "", err
	}
	tokenCache.Set(prefix, token, cache.DefaultExpiration)
	return token, nil
}

// GetCacheStats returns cache statistics for admin monitoring
func GetCacheStats() map[string]interface{} {
	stats := tokenCache.GetStats()
	items := tokenCache.GetItems()

	totalRequests := stats.Hits + stats.Misses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(stats.Hits) / float64(totalRequests) * 100
	}

	// Convert cache items to a more displayable format
	itemList := make([]map[string]interface{}, 0, len(items))
	for key, item := range items {
		var expiresDisplay string
		if item.Expiration == 0 {
			expiresDisplay = "No Expiry"
		} else {
			expiresTime := time.Unix(0, item.Expiration)
			expiresDisplay = expiresTime.Format("2006-01-02 15:04:05")
		}

		itemList = append(itemList, map[string]interface{}{
			"key":             key,
			"value":           item.Object,
			"expires":         item.Expiration,
			"expires_display": expiresDisplay,
			"expired":         item.Expiration > 0 && time.Now().UnixNano() > item.Expiration,
		})
	}

	return map[string]interface{}{
		"cache_type":     "B2 Token Cache",
		"items_count":    len(items),
		"hits":           stats.Hits,
		"misses":         stats.Misses,
		"sets":           stats.Sets,
		"total_requests": totalRequests,
		"hit_rate":       hitRate,
		"items":          itemList,
	}
}

// ClearCache clears all cached tokens
func ClearCache() {
	tokenCache.Flush()
}

// getB2DownloadTokenForPrefix returns a B2 download authorization token for a given ad directory prefix (e.g., "22/")
func getB2DownloadTokenForPrefix(prefix string) (string, error) {
	accountID := config.BackblazeMasterKeyID
	keyID := config.BackblazeKeyID
	appKey := config.BackblazeAppKey
	bucketID := config.B2BucketID
	if accountID == "" || appKey == "" || keyID == "" || bucketID == "" {
		return "", fmt.Errorf("B2 credentials not set")
	}
	req, _ := http.NewRequest("GET", "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	req.SetBasicAuth(keyID, appKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("B2 auth error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
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
	apiURL := authResp.APIURL + "/b2api/v2/b2_get_download_authorization"
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
	if resp2.StatusCode != 200 {
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
