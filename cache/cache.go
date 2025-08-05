package cache

import (
	"github.com/dgraph-io/ristretto/v2"
)

// Cache represents a generic cache interface
type Cache[T any] struct {
	impl      *ristretto.Cache[string, T]
	cacheType string
}

// New creates a new cache with the given cost function and cache type
func New[T any](costFunc func(T) int64, cacheType string) (*Cache[T], error) {
	impl, err := ristretto.NewCache(&ristretto.Config[string, T]{
		NumCounters: 1e6,     // number of keys to track frequency of (1M)
		MaxCost:     1 << 24, // maximum cost of cache (16MB)
		BufferItems: 64,      // number of keys per Get buffer
		Metrics:     true,    // enable metrics
		Cost:        costFunc,
	})
	if err != nil {
		return nil, err
	}

	return &Cache[T]{
		impl:      impl,
		cacheType: cacheType,
	}, nil
}

// Get retrieves a value from the cache
func (c *Cache[T]) Get(key string) (T, bool) {
	return c.impl.Get(key)
}

// Set stores a value in the cache
func (c *Cache[T]) Set(key string, value T, cost int64) bool {
	return c.impl.Set(key, value, cost)
}

// Clear removes all items from the cache
func (c *Cache[T]) Clear() {
	c.impl.Clear()
}

// Wait waits for the cache to finish processing
func (c *Cache[T]) Wait() {
	c.impl.Wait()
}

// Stats returns cache statistics for admin monitoring
func (c *Cache[T]) Stats() map[string]interface{} {
	metrics := c.impl.Metrics

	// Calculate memory usage in bytes
	memoryUsed := metrics.CostAdded() - metrics.CostEvicted()
	memoryUsedMB := float64(memoryUsed) / (1024 * 1024)
	totalAddedMB := float64(metrics.CostAdded()) / (1024 * 1024)
	totalEvictedMB := float64(metrics.CostEvicted()) / (1024 * 1024)

	// Calculate hit rate from metrics
	hitRate := 0.0
	totalRequests := metrics.Hits() + metrics.Misses()
	if totalRequests > 0 {
		hitRate = float64(metrics.Hits()) / float64(totalRequests) * 100
	}

	// Return standardized metrics for admin monitoring
	return map[string]interface{}{
		"cache_type":       c.cacheType,
		"hits":             metrics.Hits(),
		"misses":           metrics.Misses(),
		"sets":             metrics.KeysAdded(),
		"total_requests":   totalRequests,
		"hit_rate":         hitRate,
		"cost_added":       metrics.CostAdded(),
		"cost_evicted":     metrics.CostEvicted(),
		"gets_dropped":     metrics.GetsDropped(),
		"gets_kept":        metrics.GetsKept(),
		"sets_dropped":     metrics.SetsDropped(),
		"sets_rejected":    metrics.SetsRejected(),
		"memory_used":      memoryUsed,
		"memory_used_mb":   memoryUsedMB,
		"total_added_mb":   totalAddedMB,
		"total_evicted_mb": totalEvictedMB,
	}
}
