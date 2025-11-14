package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// Track inserted ad IDs for cleanup
var insertedAdIDs sync.Map

type TestResult struct {
	Name     string
	Duration time.Duration
	Success  bool
	Error    error
	Latency  time.Duration
}

type TestStats struct {
	Name          string
	TotalOps      int64
	SuccessOps    int64
	ErrorOps      int64
	MinLatency    time.Duration
	MaxLatency    time.Duration
	P50Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	AvgLatency    time.Duration
	TotalDuration time.Duration
	Throughput    float64 // ops/sec
	Latencies     []time.Duration
	Errors        map[string]int // unique error messages and their counts
}

func main() {
	var (
		duration    = flag.Duration("duration", 30*time.Second, "Test duration")
		concurrency = flag.Int("concurrency", 10, "Number of concurrent workers")
		dbURL       = flag.String("db", "", "Database URL (overrides DATABASE_URL env var)")
		verbose     = flag.Bool("verbose", false, "Verbose output")
		testReads   = flag.Bool("reads", true, "Test read operations")
		testWrites  = flag.Bool("writes", true, "Test write operations")
		testMixed   = flag.Bool("mixed", true, "Test mixed read/write operations")
	)
	flag.Parse()

	// Initialize logging
	logLevel := "info"
	if *verbose {
		logLevel = "debug"
	}
	if err := logger.Init(logLevel, "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Get database URL
	databaseURL := *dbURL
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		fmt.Fprintf(os.Stderr, "Database URL required (use -db flag or DATABASE_URL env var)\n")
		os.Exit(1)
	}

	// Initialize database
	if err := db.Init(databaseURL); err != nil {
		logger.Fatal("Failed to initialize database", "error", err)
	}

	// Cleanup inserted records on exit
	defer cleanupInsertedRecords()

	// Verify database connection and get some stats
	if err := verifyDatabase(); err != nil {
		logger.Fatal("Database verification failed", "error", err)
	}

	fmt.Printf("Database Performance Test\n")
	fmt.Printf("========================\n")
	fmt.Printf("Database URL: %s\n", maskURL(databaseURL))
	fmt.Printf("Duration: %v\n", *duration)
	fmt.Printf("Concurrency: %d\n", *concurrency)
	fmt.Printf("\n")

	var allStats []TestStats

	// Run read tests
	if *testReads {
		fmt.Println("Running read performance tests...")
		readStats := runReadTests(*duration, *concurrency)
		allStats = append(allStats, readStats...)
		fmt.Println()
	}

	// Run write tests
	if *testWrites {
		fmt.Println("Running write performance tests...")
		writeStats := runWriteTests(*duration, *concurrency)
		allStats = append(allStats, writeStats...)
		fmt.Println()
	}

	// Run mixed tests
	if *testMixed {
		fmt.Println("Running mixed read/write performance tests...")
		mixedStats := runMixedTests(*duration, *concurrency)
		allStats = append(allStats, mixedStats...)
		fmt.Println()
	}

	// Print summary
	printSummary(allStats)
}

func verifyDatabase() error {
	// Check if we have data to test with
	var adCount int
	err := db.QueryRow("SELECT COUNT(*) FROM Ad WHERE deleted_at IS NULL").Scan(&adCount)
	if err != nil {
		return fmt.Errorf("failed to query ad count: %w", err)
	}

	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM Users WHERE deleted_at IS NULL").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to query user count: %w", err)
	}

	var locationCount int
	err = db.QueryRow("SELECT COUNT(*) FROM Location").Scan(&locationCount)
	if err != nil {
		return fmt.Errorf("failed to query location count: %w", err)
	}

	fmt.Printf("Database stats:\n")
	fmt.Printf("  Ads: %d\n", adCount)
	fmt.Printf("  Users: %d\n", userCount)
	fmt.Printf("  Locations: %d\n", locationCount)
	fmt.Printf("\n")

	if adCount == 0 {
		return fmt.Errorf("no ads found in database - need data to run performance tests")
	}
	if userCount == 0 {
		return fmt.Errorf("no users found in database - need data to run performance tests")
	}

	return nil
}

func runReadTests(duration time.Duration, concurrency int) []TestStats {
	var stats []TestStats

	// Test 1: Simple SELECT by ID
	stats = append(stats, runTest("Read: SELECT by ID", duration, concurrency,
		func(ctx context.Context) error {
			var id int
			err := db.QueryRow(`
				SELECT id FROM Ad 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&id)
			if err != nil {
				return err
			}
			var count int
			return db.QueryRow("SELECT COUNT(*) FROM Ad WHERE id = $1 AND deleted_at IS NULL", id).Scan(&count)
		}))

	// Test 2: SELECT with JOIN (Ad + Location)
	stats = append(stats, runTest("Read: SELECT with JOIN (Ad+Location)", duration, concurrency,
		func(ctx context.Context) error {
			var id int
			err := db.QueryRow(`
				SELECT id FROM Ad 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&id)
			if err != nil {
				return err
			}
			var city sql.NullString
			return db.QueryRow(`
				SELECT l.city
				FROM Ad a
				LEFT JOIN Location l ON a.location_id = l.id
				WHERE a.id = $1 AND a.deleted_at IS NULL
			`, id).Scan(&city)
		}))

	// Test 3: SELECT with multiple JOINs (Ad + Location + BookmarkedAd + Users)
	stats = append(stats, runTest("Read: SELECT with multiple JOINs", duration, concurrency,
		func(ctx context.Context) error {
			var id int
			err := db.QueryRow(`
				SELECT id FROM Ad 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&id)
			if err != nil {
				return err
			}
			var userID int
			err = db.QueryRow(`
				SELECT id FROM Users 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&userID)
			if err != nil {
				return err
			}
			var isBookmarked int
			return db.QueryRow(`
				SELECT CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked
				FROM Ad a
				LEFT JOIN Location l ON a.location_id = l.id
				LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = $1
				LEFT JOIN Users u ON a.user_id = u.id
				WHERE a.id = $2 AND a.deleted_at IS NULL
			`, userID, id).Scan(&isBookmarked)
		}))

	// Test 4: SELECT with ORDER BY and LIMIT
	stats = append(stats, runTest("Read: SELECT with ORDER BY + LIMIT", duration, concurrency,
		func(ctx context.Context) error {
			rows, err := db.Query(`
				SELECT id, title, price, created_at
				FROM Ad
				WHERE deleted_at IS NULL
				ORDER BY created_at DESC, id DESC
				LIMIT 20
			`)
			if err != nil {
				return err
			}
			defer rows.Close()
			count := 0
			for rows.Next() {
				count++
			}
			return rows.Err()
		}))

	// Test 5: SELECT with aggregation (COUNT, GROUP BY)
	stats = append(stats, runTest("Read: SELECT with aggregation", duration, concurrency,
		func(ctx context.Context) error {
			var count int
			return db.QueryRow(`
				SELECT COUNT(*) 
				FROM Ad a
				JOIN AdSpecification ad_spec ON a.id = ad_spec.ad_id
				WHERE a.deleted_at IS NULL
				GROUP BY a.ad_category_id
				LIMIT 1
			`).Scan(&count)
		}))

	// Test 6: SELECT with IN clause (multiple IDs)
	stats = append(stats, runTest("Read: SELECT with IN clause", duration, concurrency,
		func(ctx context.Context) error {
			var maxID int
			err := db.QueryRow("SELECT COALESCE(MAX(id), 1) FROM Ad WHERE deleted_at IS NULL").Scan(&maxID)
			if err != nil {
				return err
			}
			if maxID == 0 {
				return fmt.Errorf("no ads found")
			}
			// Select 10 random IDs
			ids := make([]int, 10)
			for i := range ids {
				ids[i] = rand.Intn(maxID) + 1
			}
			rows, err := db.Query(`
				SELECT id, title, price
				FROM Ad
				WHERE id = ANY($1) AND deleted_at IS NULL
			`, ids)
			if err != nil {
				return err
			}
			defer rows.Close()
			count := 0
			for rows.Next() {
				count++
			}
			return rows.Err()
		}))

	return stats
}

func runWriteTests(duration time.Duration, concurrency int) []TestStats {
	var stats []TestStats

	// Test 1: Simple INSERT
	stats = append(stats, runTest("Write: INSERT", duration, concurrency,
		func(ctx context.Context) error {
			// Get an actual existing user and category
			var userID, categoryID int
			err := db.QueryRow(`
				SELECT id FROM Users 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&userID)
			if err != nil {
				return err
			}
			err = db.QueryRow(`
				SELECT id FROM AdCategory 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&categoryID)
			if err != nil {
				return err
			}

			// Insert a test ad (we'll delete it later)
			title := fmt.Sprintf("Perf test ad %d", time.Now().UnixNano())
			var adID int
			err = db.QueryRow(`
				INSERT INTO Ad (ad_category_id, title, description, price, user_id, created_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
				RETURNING id
			`, categoryID, title, "Test description", rand.Float64()*1000, userID).Scan(&adID)
			if err != nil {
				return err
			}
			// Track the inserted ID for cleanup
			insertedAdIDs.Store(adID, true)
			return nil
		}))

	// Test 2: INSERT with transaction
	stats = append(stats, runTest("Write: INSERT in transaction", duration, concurrency,
		func(ctx context.Context) error {
			// Get an actual existing user and category
			var userID, categoryID int
			err := db.QueryRow(`
				SELECT id FROM Users 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&userID)
			if err != nil {
				return err
			}
			err = db.QueryRow(`
				SELECT id FROM AdCategory 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&categoryID)
			if err != nil {
				return err
			}

			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			title := fmt.Sprintf("Perf test ad %d", time.Now().UnixNano())
			var adID int
			err = tx.QueryRow(`
				INSERT INTO Ad (ad_category_id, title, description, price, user_id, created_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
				RETURNING id
			`, categoryID, title, "Test description", rand.Float64()*1000, userID).Scan(&adID)
			if err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}

			// Track the inserted ID for cleanup
			insertedAdIDs.Store(adID, true)
			return nil
		}))

	// Test 3: UPDATE
	stats = append(stats, runTest("Write: UPDATE", duration, concurrency,
		func(ctx context.Context) error {
			var id int
			err := db.QueryRow(`
				SELECT id FROM Ad 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&id)
			if err != nil {
				return err
			}
			_, err = db.Exec(`
				UPDATE Ad 
				SET click_count = click_count + 1, last_clicked_at = NOW()
				WHERE id = $1 AND deleted_at IS NULL
			`, id)
			return err
		}))

	return stats
}

func runMixedTests(duration time.Duration, concurrency int) []TestStats {
	var stats []TestStats

	// Mixed workload: 80% reads, 20% writes
	stats = append(stats, runTest("Mixed: 80% reads, 20% writes", duration, concurrency,
		func(ctx context.Context) error {
			var id int
			err := db.QueryRow(`
				SELECT id FROM Ad 
				WHERE deleted_at IS NULL 
				ORDER BY RANDOM() 
				LIMIT 1
			`).Scan(&id)
			if err != nil {
				return err
			}
			if rand.Float64() < 0.8 {
				// Read operation
				var count int
				return db.QueryRow("SELECT COUNT(*) FROM Ad WHERE id = $1 AND deleted_at IS NULL", id).Scan(&count)
			} else {
				// Write operation
				_, err = db.Exec(`
					UPDATE Ad 
					SET click_count = click_count + 1, last_clicked_at = NOW()
					WHERE id = $1 AND deleted_at IS NULL
				`, id)
				return err
			}
		}))

	return stats
}

func runTest(name string, duration time.Duration, concurrency int, op func(context.Context) error) TestStats {
	fmt.Printf("  Running: %s...\n", name)

	var (
		totalOps      int64
		successOps    int64
		errorOps      int64
		latencies     []time.Duration
		latenciesLock sync.Mutex
		errors        = make(map[string]int)
		errorsLock    sync.Mutex
		startTime     = time.Now()
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					opStart := time.Now()
					err := op(ctx)
					latency := time.Since(opStart)

					atomic.AddInt64(&totalOps, 1)
					if err != nil {
						atomic.AddInt64(&errorOps, 1)
						// Collect unique error messages
						errMsg := err.Error()
						errorsLock.Lock()
						errors[errMsg]++
						errorsLock.Unlock()
					} else {
						atomic.AddInt64(&successOps, 1)
					}

					latenciesLock.Lock()
					latencies = append(latencies, latency)
					latenciesLock.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Calculate statistics
	stats := calculateStats(name, totalOps, successOps, errorOps, latencies, totalDuration, errors)
	return stats
}

func calculateStats(name string, totalOps, successOps, errorOps int64, latencies []time.Duration, totalDuration time.Duration, errors map[string]int) TestStats {
	if errors == nil {
		errors = make(map[string]int)
	}
	stats := TestStats{
		Name:          name,
		TotalOps:      totalOps,
		SuccessOps:    successOps,
		ErrorOps:      errorOps,
		TotalDuration: totalDuration,
		Latencies:     latencies,
		Errors:        errors,
	}

	if len(latencies) == 0 {
		return stats
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	stats.MinLatency = sorted[0]
	stats.MaxLatency = sorted[len(sorted)-1]
	stats.P50Latency = percentile(sorted, 0.50)
	stats.P95Latency = percentile(sorted, 0.95)
	stats.P99Latency = percentile(sorted, 0.99)

	// Calculate average
	var sum time.Duration
	for _, lat := range sorted {
		sum += lat
	}
	stats.AvgLatency = sum / time.Duration(len(sorted))

	// Calculate throughput
	if totalDuration > 0 {
		stats.Throughput = float64(successOps) / totalDuration.Seconds()
	}

	return stats
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func printSummary(stats []TestStats) {
	fmt.Println("Performance Test Summary")
	fmt.Println("=======================")
	fmt.Println()

	for _, stat := range stats {
		fmt.Printf("Test: %s\n", stat.Name)
		fmt.Printf("  Total Operations: %d\n", stat.TotalOps)
		fmt.Printf("  Successful: %d\n", stat.SuccessOps)
		fmt.Printf("  Errors: %d (%.2f%%)\n", stat.ErrorOps,
			float64(stat.ErrorOps)/float64(stat.TotalOps)*100)
		fmt.Printf("  Duration: %v\n", stat.TotalDuration)
		fmt.Printf("  Throughput: %.2f ops/sec\n", stat.Throughput)
		if len(stat.Latencies) > 0 {
			fmt.Printf("  Latency:\n")
			fmt.Printf("    Min:   %v\n", stat.MinLatency)
			fmt.Printf("    Avg:   %v\n", stat.AvgLatency)
			fmt.Printf("    P50:   %v\n", stat.P50Latency)
			fmt.Printf("    P95:   %v\n", stat.P95Latency)
			fmt.Printf("    P99:   %v\n", stat.P99Latency)
			fmt.Printf("    Max:   %v\n", stat.MaxLatency)
		}
		if len(stat.Errors) > 0 {
			fmt.Printf("  Unique Errors (%d):\n", len(stat.Errors))
			// Sort errors by count (descending) for better visibility
			type errorCount struct {
				msg   string
				count int
			}
			errorList := make([]errorCount, 0, len(stat.Errors))
			for msg, count := range stat.Errors {
				errorList = append(errorList, errorCount{msg, count})
			}
			sort.Slice(errorList, func(i, j int) bool {
				return errorList[i].count > errorList[j].count
			})
			// Show top 10 errors to avoid overwhelming output
			maxErrors := 10
			if len(errorList) < maxErrors {
				maxErrors = len(errorList)
			}
			for i := 0; i < maxErrors; i++ {
				fmt.Printf("    [%d] %s\n", errorList[i].count, errorList[i].msg)
			}
			if len(errorList) > maxErrors {
				fmt.Printf("    ... and %d more unique errors\n", len(errorList)-maxErrors)
			}
		}
		fmt.Println()
	}
}

func cleanupInsertedRecords() {
	// Collect all inserted ad IDs
	var ids []int
	insertedAdIDs.Range(func(key, value interface{}) bool {
		if adID, ok := key.(int); ok {
			ids = append(ids, adID)
		}
		return true
	})

	if len(ids) == 0 {
		return
	}

	fmt.Printf("\nCleaning up %d inserted test records...\n", len(ids))

	// Delete in batches to avoid very large IN clauses
	batchSize := 1000
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		// Use soft delete (set deleted_at) to match application behavior
		_, err := db.Exec(`
			UPDATE Ad 
			SET deleted_at = NOW() 
			WHERE id = ANY($1) AND deleted_at IS NULL
		`, batch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup some test records: %v\n", err)
		}
	}

	fmt.Printf("Cleanup complete.\n")
}

func maskURL(url string) string {
	// Simple masking - just show the scheme and host
	// For postgres://user:pass@host:port/db, show postgres://***@host:port/db
	if len(url) < 10 {
		return "***"
	}
	// Find @ symbol
	atIdx := -1
	for i := 0; i < len(url); i++ {
		if url[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx > 0 {
		// Mask the part before @
		schemeEnd := 0
		for i := 0; i < len(url); i++ {
			if url[i] == ':' {
				schemeEnd = i
				break
			}
		}
		if schemeEnd > 0 {
			return url[:schemeEnd+3] + "***" + url[atIdx:]
		}
	}
	return "***"
}
