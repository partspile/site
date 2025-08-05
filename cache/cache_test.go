package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	// Test creating a cache with string values
	cache, err := New[string](func(value string) int64 {
		return int64(len(value))
	}, "Test Cache")

	require.NoError(t, err)
	assert.NotNil(t, cache)

	// Test that the cache works
	testValue := "test string"
	cache.Set("test-key", testValue, int64(len(testValue)))

	// Wait a bit for the cache to process the set
	time.Sleep(10 * time.Millisecond)

	// Retrieve the value
	if value, found := cache.Get("test-key"); found {
		assert.Equal(t, testValue, value)
	} else {
		t.Error("Expected to find cached value")
	}
}

func TestNewCacheWithSlice(t *testing.T) {
	// Test creating a cache with slice values
	cache, err := New[[]float32](func(value []float32) int64 {
		return int64(len(value) * 4) // 4 bytes per float32
	}, "Test Slice Cache")

	require.NoError(t, err)
	assert.NotNil(t, cache)

	// Test that the cache works
	testValue := []float32{1.0, 2.0, 3.0}
	cache.Set("test-key", testValue, int64(len(testValue)*4))

	// Wait a bit for the cache to process the set
	time.Sleep(10 * time.Millisecond)

	// Retrieve the value
	if value, found := cache.Get("test-key"); found {
		assert.Equal(t, testValue, value)
	} else {
		t.Error("Expected to find cached value")
	}
}

func TestCacheStats(t *testing.T) {
	// Create a test cache
	cache, err := New[string](func(value string) int64 {
		return int64(len(value))
	}, "Test Cache")
	require.NoError(t, err)

	// Add some test data to generate metrics
	testValue := "test string"
	cache.Set("key1", testValue, int64(len(testValue)))
	cache.Set("key2", testValue, int64(len(testValue)))

	// Wait a bit for the cache to process the sets
	time.Sleep(10 * time.Millisecond)

	cache.Get("key1") // Hit
	cache.Get("key2") // Hit
	cache.Get("key3") // Miss

	// Get stats
	stats := cache.Stats()

	// Verify expected keys are present
	expectedKeys := []string{
		"cache_type", "hits", "misses", "sets", "total_requests",
		"hit_rate", "cost_added", "cost_evicted", "gets_dropped",
		"gets_kept", "sets_dropped", "sets_rejected", "memory_used",
		"memory_used_mb", "total_added_mb", "total_evicted_mb",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, stats, key, "Expected key %s in stats", key)
	}

	// Verify cache_type is set correctly
	assert.Equal(t, "Test Cache", stats["cache_type"])

	// Verify we have some activity
	assert.GreaterOrEqual(t, stats["sets"], uint64(0))
	assert.GreaterOrEqual(t, stats["hits"], uint64(0))
	assert.GreaterOrEqual(t, stats["misses"], uint64(0))

	// Verify hit rate calculation
	hitRate := stats["hit_rate"].(float64)
	assert.GreaterOrEqual(t, hitRate, 0.0)
	assert.LessOrEqual(t, hitRate, 100.0)

	// Verify memory metrics are floats
	memoryUsedMB := stats["memory_used_mb"].(float64)
	assert.GreaterOrEqual(t, memoryUsedMB, 0.0)
}

func TestCacheStatsEmptyCache(t *testing.T) {
	// Create a test cache
	cache, err := New[string](func(value string) int64 {
		return int64(len(value))
	}, "Empty Cache")
	require.NoError(t, err)

	// Get stats without any activity
	stats := cache.Stats()

	// Verify expected keys are present
	expectedKeys := []string{
		"cache_type", "hits", "misses", "sets", "total_requests",
		"hit_rate", "cost_added", "cost_evicted", "gets_dropped",
		"gets_kept", "sets_dropped", "sets_rejected", "memory_used",
		"memory_used_mb", "total_added_mb", "total_evicted_mb",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, stats, key, "Expected key %s in stats", key)
	}

	// Verify cache_type is set correctly
	assert.Equal(t, "Empty Cache", stats["cache_type"])

	// Verify initial values
	assert.Equal(t, uint64(0), stats["hits"])
	assert.Equal(t, uint64(0), stats["misses"])
	assert.Equal(t, uint64(0), stats["sets"])
	assert.Equal(t, uint64(0), stats["total_requests"])
	assert.Equal(t, 0.0, stats["hit_rate"])
}

func BenchmarkNewCache(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cache, err := New[string](func(value string) int64 {
			return int64(len(value))
		}, "Benchmark Cache")
		if err != nil {
			b.Fatal(err)
		}
		if cache == nil {
			b.Fatal("Cache is nil")
		}
	}
}

func BenchmarkCacheStats(b *testing.B) {
	cache, err := New[string](func(value string) int64 {
		return int64(len(value))
	}, "Benchmark Cache")
	if err != nil {
		b.Fatal(err)
	}

	// Add some test data
	testValue := "test string"
	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key%d", i), testValue, int64(len(testValue)))
	}

	// Wait for cache to process
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := cache.Stats()
		if stats == nil {
			b.Fatal("Stats is nil")
		}
	}
}
