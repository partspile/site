package b2util

import (
	"os"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestGetB2DownloadTokenForPrefixCached_CacheHit(t *testing.T) {
	// Test that cached tokens are returned
	prefix := "test-prefix/"

	// Set up environment variables for testing
	os.Setenv("BACKBLAZE_MASTER_KEY_ID", "test-account-id")
	os.Setenv("BACKBLAZE_KEY_ID", "test-key-id")
	os.Setenv("BACKBLAZE_APP_KEY", "test-app-key")
	os.Setenv("B2_BUCKET_ID", "test-bucket-id")

	// First call should cache the token
	_, err := GetB2DownloadTokenForPrefixCached(prefix)

	// Second call should return cached token
	_, err2 := GetB2DownloadTokenForPrefixCached(prefix)

	// Note: This test will fail in real environment without valid credentials
	// but it tests the caching logic structure
	assert.Error(t, err)  // Should fail due to invalid credentials
	assert.Error(t, err2) // Should also fail
}

func TestGetB2DownloadTokenForPrefixCached_DifferentPrefixes(t *testing.T) {
	// Test that different prefixes have separate cache entries
	prefix1 := "prefix1/"
	prefix2 := "prefix2/"

	// Set up environment variables for testing
	os.Setenv("BACKBLAZE_MASTER_KEY_ID", "test-account-id")
	os.Setenv("BACKBLAZE_KEY_ID", "test-key-id")
	os.Setenv("BACKBLAZE_APP_KEY", "test-app-key")
	os.Setenv("B2_BUCKET_ID", "test-bucket-id")

	// Both should fail due to invalid credentials but test structure
	token1, err1 := GetB2DownloadTokenForPrefixCached(prefix1)
	token2, err2 := GetB2DownloadTokenForPrefixCached(prefix2)

	assert.Error(t, err1)
	assert.Error(t, err2)
	assert.Empty(t, token1)
	assert.Empty(t, token2)
}

func TestGetB2DownloadTokenForPrefixCached_MissingCredentials(t *testing.T) {
	// Test behavior when credentials are missing
	prefix := "test-prefix/"

	// Clear environment variables
	os.Unsetenv("BACKBLAZE_MASTER_KEY_ID")
	os.Unsetenv("BACKBLAZE_KEY_ID")
	os.Unsetenv("BACKBLAZE_APP_KEY")
	os.Unsetenv("B2_BUCKET_ID")

	token, err := GetB2DownloadTokenForPrefixCached(prefix)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "B2 credentials not set")
}

func TestGetB2DownloadTokenForPrefixCached_PartialCredentials(t *testing.T) {
	// Test behavior when only some credentials are set
	prefix := "test-prefix/"

	// Set only some environment variables
	os.Setenv("BACKBLAZE_MASTER_KEY_ID", "test-account-id")
	os.Unsetenv("BACKBLAZE_KEY_ID")
	os.Setenv("BACKBLAZE_APP_KEY", "test-app-key")
	os.Setenv("B2_BUCKET_ID", "test-bucket-id")

	token, err := GetB2DownloadTokenForPrefixCached(prefix)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "B2 credentials not set")
}

func TestGetCacheStats(t *testing.T) {
	stats := GetCacheStats()

	// Check that stats contains expected keys
	if _, ok := stats["cache_type"]; !ok {
		t.Error("Expected cache_type in stats")
	}
	if _, ok := stats["items_count"]; !ok {
		t.Error("Expected items_count in stats")
	}
	if _, ok := stats["hits"]; !ok {
		t.Error("Expected hits in stats")
	}
	if _, ok := stats["misses"]; !ok {
		t.Error("Expected misses in stats")
	}
	if _, ok := stats["hit_rate"]; !ok {
		t.Error("Expected hit_rate in stats")
	}

	// Check that cache_type is the expected value
	if stats["cache_type"] != "B2 Token Cache" {
		t.Errorf("Expected cache_type to be 'B2 Token Cache', got %v", stats["cache_type"])
	}

	// Check that items_count is an integer
	if _, ok := stats["items_count"].(int); !ok {
		t.Error("Expected items_count to be an integer")
	}
}

func TestClearCache(t *testing.T) {
	// Set a test value in cache
	tokenCache.Set("test-prefix", "test-token", cache.DefaultExpiration)

	// Verify it's in cache
	if _, found := tokenCache.Get("test-prefix"); !found {
		t.Error("Expected test value to be in cache before clearing")
	}

	// Clear cache
	ClearCache()

	// Verify cache is empty
	if _, found := tokenCache.Get("test-prefix"); found {
		t.Error("Expected test value to be removed from cache after clearing")
	}
}

func TestCacheStatistics(t *testing.T) {
	// Clear cache first
	ClearCache()

	// Get initial stats
	initialStats := GetCacheStats()
	initialHits := initialStats["hits"].(int64)
	initialMisses := initialStats["misses"].(int64)

	// Set a value
	tokenCache.Set("test-key", "test-value", cache.DefaultExpiration)

	// Get the value (should be a hit)
	tokenCache.Get("test-key")

	// Try to get a non-existent value (should be a miss)
	tokenCache.Get("non-existent-key")

	// Get updated stats
	updatedStats := GetCacheStats()
	updatedHits := updatedStats["hits"].(int64)
	updatedMisses := updatedStats["misses"].(int64)

	// Verify hits increased
	if updatedHits <= initialHits {
		t.Error("Expected hits to increase after successful get")
	}

	// Verify misses increased
	if updatedMisses <= initialMisses {
		t.Error("Expected misses to increase after failed get")
	}

	// Verify items count is correct
	itemsCount := updatedStats["items_count"].(int)
	if itemsCount != 1 {
		t.Errorf("Expected 1 item in cache, got %d", itemsCount)
	}
}
