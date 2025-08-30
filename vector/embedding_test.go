package vector

import (
	"testing"
)

func TestEmbeddingCacheInitialization(t *testing.T) {
	// Initialize the caches
	err := InitEmbeddingCaches()
	if err != nil {
		t.Fatalf("Failed to initialize embedding caches: %v", err)
	}

	// Test that all caches are initialized
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

	// Check that all three cache types have stats
	if _, exists := stats["query"]; !exists {
		t.Error("Query cache stats missing")
	}
	if _, exists := stats["user"]; !exists {
		t.Error("User cache stats missing")
	}
	if _, exists := stats["site"]; !exists {
		t.Error("Site cache stats missing")
	}

	// Test cache clear
	ClearEmbeddingCache()
	statsAfterClear := GetEmbeddingCacheStats()

	// Check that all caches are cleared
	for cacheType := range statsAfterClear {
		if cacheStats, ok := statsAfterClear[cacheType].(map[string]interface{}); ok {
			if hits, exists := cacheStats["hits"]; exists && hits.(uint64) != 0 {
				t.Errorf("Expected %s cache hits to be 0 after clear, got %d", cacheType, hits)
			}
			if sets, exists := cacheStats["sets"]; exists && sets.(uint64) != 0 {
				t.Errorf("Expected %s cache sets to be 0 after clear, got %d", cacheType, sets)
			}
		}
	}
}

func TestEmbeddingCacheWithNilCache(t *testing.T) {
	// Test that EmbedTextCached works when cache is nil
	queryEmbeddingCache = nil

	testQuery := "test query with nil cache"
	// This should not panic and should return an error about cache not initialized
	_, err := GetQueryEmbedding(testQuery)
	if err == nil {
		t.Error("Expected error when query cache is nil")
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
