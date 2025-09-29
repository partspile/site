package cache

import (
	"time"

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

// Set stores a value in the cache with a default TTL of 1 hour
func (c *Cache[T]) Set(key string, value T, cost int64) bool {
	return c.SetWithTTL(key, value, cost, time.Hour)
}

// SetWithTTL stores a value in the cache with a specific TTL
func (c *Cache[T]) SetWithTTL(key string, value T, cost int64, ttl time.Duration) bool {
	return c.impl.SetWithTTL(key, value, cost, ttl)
}

// Clear removes all items from the cache
func (c *Cache[T]) Clear() {
	c.impl.Clear()
}

// Wait waits for the cache to finish processing
func (c *Cache[T]) Wait() {
	c.impl.Wait()
}

// GetItemCount returns the current number of items in the cache
func (c *Cache[T]) GetItemCount() int64 {
	return int64(c.impl.Metrics.KeysAdded() - c.impl.Metrics.KeysEvicted())
}

// Stats returns cache statistics for admin monitoring
func (c *Cache[T]) Stats() map[string]interface{} {
	metrics := c.impl.Metrics

	// Calculate memory usage in bytes
	memoryUsed := metrics.CostAdded() - metrics.CostEvicted()
	memoryUsedKB := float64(memoryUsed) / 1024
	totalAddedKB := float64(metrics.CostAdded()) / 1024
	totalEvictedKB := float64(metrics.CostEvicted()) / 1024

	// Calculate hit rate from metrics
	hitRate := 0.0
	totalRequests := metrics.Hits() + metrics.Misses()
	if totalRequests > 0 {
		hitRate = float64(metrics.Hits()) / float64(totalRequests) * 100
	}

	// Get TTL-related statistics
	lifeExpectancy := metrics.LifeExpectancySeconds()
	ttlStats := map[string]interface{}{
		"life_expectancy_count": 0,
		"life_expectancy_min":   0,
		"life_expectancy_max":   0,
		"life_expectancy_mean":  0.0,
		"life_expectancy_p50":   0.0,
		"life_expectancy_p95":   0.0,
		"life_expectancy_p99":   0.0,
	}

	if lifeExpectancy != nil {
		ttlStats["life_expectancy_count"] = lifeExpectancy.Count
		ttlStats["life_expectancy_min"] = lifeExpectancy.Min
		ttlStats["life_expectancy_max"] = lifeExpectancy.Max
		ttlStats["life_expectancy_mean"] = lifeExpectancy.Mean()
		ttlStats["life_expectancy_p50"] = lifeExpectancy.Percentile(50)
		ttlStats["life_expectancy_p95"] = lifeExpectancy.Percentile(95)
		ttlStats["life_expectancy_p99"] = lifeExpectancy.Percentile(99)
	}

	// Return standardized metrics for admin monitoring
	stats := map[string]interface{}{
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
		"memory_used_kb":   memoryUsedKB,
		"total_added_kb":   totalAddedKB,
		"total_evicted_kb": totalEvictedKB,
		"current_items":    c.GetItemCount(),
	}

	// Add TTL stats
	for k, v := range ttlStats {
		stats[k] = v
	}

	return stats
}
