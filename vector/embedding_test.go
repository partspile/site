package vector

import (
	"testing"
)

func TestEmbeddingCacheInitialization(t *testing.T) {
	// Initialize the cache
	err := InitEmbeddingCache()
	if err != nil {
		t.Fatalf("Failed to initialize embedding cache: %v", err)
	}

	// Test that all caches are initialized
	if embeddingCache == nil {
		t.Fatal("Legacy embedding cache should not be nil after initialization")
	}
	if queryEmbeddingCache == nil {
		t.Fatal("Query embedding cache should not be nil after initialization")
	}
	if userEmbeddingCache == nil {
		t.Fatal("User embedding cache should not be nil after initialization")
	}
	if siteEmbeddingCache == nil {
		t.Fatal("Site embedding cache should not be nil after initialization")
	}

	// Test cache stats
	stats := GetEmbeddingCacheStats()
	if stats == nil {
		t.Fatal("Cache stats should not be nil")
	}

	// Check that required stats fields exist
	requiredFields := []string{"hits", "misses", "sets", "hit_rate", "memory_used_mb"}
	for _, field := range requiredFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Required field '%s' missing from cache stats", field)
		}
	}

	// Test cache clear
	ClearEmbeddingCache()
	statsAfterClear := GetEmbeddingCacheStats()

	// Check that cache is cleared
	if statsAfterClear["hits"].(uint64) != 0 {
		t.Error("Expected cache hits to be 0 after clear")
	}
	if statsAfterClear["sets"].(uint64) != 0 {
		t.Error("Expected cache sets to be 0 after clear")
	}
}

func TestEmbeddingCacheWithNilCache(t *testing.T) {
	// Test that EmbedTextCached works when cache is nil
	embeddingCache = nil

	testQuery := "test query with nil cache"
	// This should not panic and should return an error about Gemini client
	_, err := EmbedTextCached(testQuery)
	if err == nil {
		t.Error("Expected error when cache is nil and Gemini client not initialized")
	}
}

func TestNewHelperFunctions(t *testing.T) {
	// Save current cache state
	originalQueryCache := queryEmbeddingCache
	originalUserCache := userEmbeddingCache
	originalSiteCache := siteEmbeddingCache

	// Set caches to nil for testing
	queryEmbeddingCache = nil
	userEmbeddingCache = nil
	siteEmbeddingCache = nil

	// Restore caches after test
	defer func() {
		queryEmbeddingCache = originalQueryCache
		userEmbeddingCache = originalUserCache
		siteEmbeddingCache = originalSiteCache
	}()

	// Test GetUserEmbedding with nil cache
	_, err := GetUserEmbedding(123)
	if err == nil {
		t.Error("Expected error when user embedding cache is nil")
	}

	// Test SetUserEmbedding with nil cache
	err = SetUserEmbedding(123, []float32{1.0, 2.0, 3.0})
	if err == nil {
		t.Error("Expected error when user embedding cache is nil")
	}

	// Test GetSiteEmbedding with nil cache
	_, err = GetSiteEmbedding("test-campaign")
	if err == nil {
		t.Error("Expected error when site embedding cache is nil")
	}

}
