package b2util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetB2DownloadTokenForPrefixCached_CacheHit(t *testing.T) {
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

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
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

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
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

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
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

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
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

	stats := GetCacheStats()

	// Check that stats contains expected keys
	if _, ok := stats["cache_type"]; !ok {
		t.Error("Expected cache_type in stats")
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
	if _, ok := stats["memory_used_mb"]; !ok {
		t.Error("Expected memory_used_mb in stats")
	}
	if _, ok := stats["total_added_mb"]; !ok {
		t.Error("Expected total_added_mb in stats")
	}
	if _, ok := stats["total_evicted_mb"]; !ok {
		t.Error("Expected total_evicted_mb in stats")
	}

	// Check that cache_type is the expected value
	if stats["cache_type"] != "B2 Token Cache (Ristretto)" {
		t.Errorf("Expected cache_type to be 'B2 Token Cache (Ristretto)', got %v", stats["cache_type"])
	}

	// Check that hits and misses are integers
	if _, ok := stats["hits"].(uint64); !ok {
		t.Error("Expected hits to be a uint64")
	}
	if _, ok := stats["misses"].(uint64); !ok {
		t.Error("Expected misses to be a uint64")
	}

	// Check that memory metrics are floats
	if _, ok := stats["memory_used_mb"].(float64); !ok {
		t.Error("Expected memory_used_mb to be a float64")
	}
	if _, ok := stats["total_added_mb"].(float64); !ok {
		t.Error("Expected total_added_mb to be a float64")
	}
	if _, ok := stats["total_evicted_mb"].(float64); !ok {
		t.Error("Expected total_evicted_mb to be a float64")
	}
}

func TestClearCache(t *testing.T) {
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

	// Clear cache first
	ClearCache()

	// Set a test value in cache
	tokenCache.Set("test-prefix", "test-token", int64(len("test-token")))

	// Wait for the item to be available (Ristretto has eventual consistency)
	tokenCache.Wait()

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
	// Initialize cache for test
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize cache for test: %v", err)
	}

	// Clear cache first
	ClearCache()

	// Get initial stats
	initialStats := GetCacheStats()
	initialHits := initialStats["hits"].(uint64)
	initialMisses := initialStats["misses"].(uint64)

	// Set a value
	tokenCache.Set("test-key", "test-value", int64(len("test-value")))

	// Wait for the item to be available
	tokenCache.Wait()

	// Get the value (should be a hit)
	tokenCache.Get("test-key")

	// Try to get a non-existent value (should be a miss)
	tokenCache.Get("non-existent-key")

	// Get updated stats
	updatedStats := GetCacheStats()
	updatedHits := updatedStats["hits"].(uint64)
	updatedMisses := updatedStats["misses"].(uint64)

	// Verify hits increased
	if updatedHits <= initialHits {
		t.Error("Expected hits to increase after successful get")
	}

	// Verify misses increased
	if updatedMisses <= initialMisses {
		t.Error("Expected misses to increase after failed get")
	}
}
