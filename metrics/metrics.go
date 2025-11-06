package metrics

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Metric represents a single server metric snapshot
type Metric struct {
	RecordedAt time.Time
	PIDCPU     float64
	PIDRAM     int64
	PIDConns   int64
}

var (
	metricsLock sync.RWMutex
	metricsData []Metric
	maxMetrics  = 24 * 60 * 60 / 5 // 24 hours * 60 min * 60 sec / 5 sec = 17,280 samples
	proc        *process.Process
)

// StartCollector starts the metrics collection goroutine that samples
// every 5 seconds and stores data in memory (max 24 hours)
func StartCollector() {
	go func() {
		log.Printf("[metrics] Starting metrics collector (samples every 5s, max %d samples)", maxMetrics)

		// Initialize process handle
		var err error
		proc, err = process.NewProcess(int32(os.Getpid()))
		if err != nil {
			log.Printf("[metrics] Warning: Failed to create process handle: %v", err)
			proc = nil
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			collectMetrics()
		}
	}()
}

// collectMetrics collects app process metrics
func collectMetrics() {
	now := time.Now()

	var pidCPU float64
	var pidRAM int64
	var pidConns int64

	// Process metrics
	if proc != nil {
		// Process CPU usage
		if cpuPercent, err := proc.CPUPercent(); err == nil {
			pidCPU = cpuPercent
		}

		// Process memory
		if memInfo, err := proc.MemoryInfo(); err == nil {
			pidRAM = int64(memInfo.RSS)
		}

		// Process connections
		if conns, err := proc.Connections(); err == nil {
			pidConns = int64(len(conns))
		}
	}

	metric := Metric{
		RecordedAt: now,
		PIDCPU:     pidCPU,
		PIDRAM:     pidRAM,
		PIDConns:   pidConns,
	}

	metricsLock.Lock()
	defer metricsLock.Unlock()

	// If slice is full, remove first element (oldest)
	if len(metricsData) >= maxMetrics {
		metricsData = metricsData[1:]
	}

	// Add new metric to end
	metricsData = append(metricsData, metric)
}

// GetMetrics retrieves metrics within a time range from memory
func GetMetrics(startTime, endTime time.Time) ([]Metric, error) {
	metricsLock.RLock()
	defer metricsLock.RUnlock()

	var result []Metric
	for _, m := range metricsData {
		if (m.RecordedAt.Equal(startTime) || m.RecordedAt.After(startTime)) &&
			(m.RecordedAt.Equal(endTime) || m.RecordedAt.Before(endTime)) {
			result = append(result, m)
		}
	}

	return result, nil
}

// GetRecentMetrics returns metrics from the last N hours
func GetRecentMetrics(hours int) ([]Metric, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	endTime := time.Now()
	return GetMetrics(startTime, endTime)
}
