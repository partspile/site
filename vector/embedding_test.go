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

	// Test that cache is initialized
	if embeddingCache == nil {
		t.Fatal("Embedding cache should not be nil after initialization")
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
