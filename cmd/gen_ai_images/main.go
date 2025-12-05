package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	genai "google.golang.org/genai"
)

type AdImageJob struct {
	AdID     int
	ImageNum int
	AdDetail *ad.AdDetail
}

func main() {
	var outputDir string
	var workers int
	var model string
	var limit int

	flag.StringVar(&outputDir, "output", "generated_images", "Output directory for images")
	flag.IntVar(&workers, "workers", 5, "Number of concurrent workers")
	flag.StringVar(&model, "model", "gemini-3-pro-image-preview", "Image generation model (imagen-4.0-fast-generate-001, gemini-2.5-flash-image, etc.)")
	flag.IntVar(&limit, "limit", 0, "Limit number of ads to process (0 = all)")
	flag.Parse()

	// Initialize logging
	if err := logger.Init(config.LogLevel, config.LogFormat, config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	// Initialize encryption keys (required for user decryption)
	config.SecurityCheck()

	// Load API key
	apiKey := config.GeminiAPIKey
	if apiKey == "" {
		logger.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Create Gemini client to list available models
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		logger.Fatal("Failed to create Gemini client", "error", err)
	}

	// List and print available models
	if err := listAvailableModels(ctx, client); err != nil {
		logger.Warn("Failed to list models", "error", err)
	}

	// Initialize database
	if err := db.Init(config.DatabaseURL); err != nil {
		logger.Fatal("Failed to initialize database", "error", err, "databaseURL", config.DatabaseURL)
	}

	// Open DB for direct access
	database, err := sql.Open("pgx", config.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to open DB", "error", err, "databaseURL", config.DatabaseURL)
	}
	defer database.Close()

	// Query all ad IDs
	adIDs, err := getAllAdIDs(database, limit)
	if err != nil {
		logger.Fatal("Failed to get ad IDs", "error", err)
	}

	logger.Info("Found ads to process", "count", len(adIDs))
	fmt.Printf("Found %d ads to process\n", len(adIDs))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Fatal("Failed to create output directory", "error", err)
	}

	// Build jobs for all ads
	var jobs []AdImageJob
	for _, adID := range adIDs {
		adDetail, err := ad.GetAdDetailByID(adID, 0)
		if err != nil {
			logger.Warn("Failed to get ad detail, skipping", "adID", adID, "error", err)
			continue
		}

		// Generate images based on image_count (1 to image_count)
		numImages := adDetail.ImageCount
		if numImages < 1 {
			numImages = 1
		}
		if numImages > 20 {
			numImages = 20 // MaxImagesPerAd
		}

		for i := 1; i <= numImages; i++ {
			jobs = append(jobs, AdImageJob{
				AdID:     adID,
				ImageNum: i,
				AdDetail: adDetail,
			})
		}
	}

	fmt.Printf("Generating %d images using %s with %d workers...\n", len(jobs), model, workers)

	// Start batch processing
	start := time.Now()
	generateImagesConcurrent(jobs, outputDir, apiKey, model, workers)
	fmt.Printf("Done in %.2f seconds. Images saved to %s\n", time.Since(start).Seconds(), outputDir)
}

func getAllAdIDs(database *sql.DB, limit int) ([]int, error) {
	query := "SELECT id FROM Ad WHERE deleted_at IS NULL ORDER BY id"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := database.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ad IDs: %v", err)
	}
	defer rows.Close()

	var adIDs []int
	for rows.Next() {
		var adID int
		if err := rows.Scan(&adID); err != nil {
			return nil, fmt.Errorf("failed to scan ad ID: %v", err)
		}
		adIDs = append(adIDs, adID)
	}

	return adIDs, nil
}

func listAvailableModels(ctx context.Context, client *genai.Client) error {
	fmt.Println("\n=== Available Gemini Models ===")
	imageModels := []string{}
	otherModels := []string{}

	// Use All() method to iterate through all models
	for model, err := range client.Models.All(ctx) {
		if err != nil {
			return fmt.Errorf("error iterating models: %w", err)
		}

		name := model.Name
		// Extract just the model name (remove "models/" prefix if present)
		name = strings.TrimPrefix(name, "models/")

		// Check if this model supports image generation
		// Look for "image" in the name
		if strings.Contains(strings.ToLower(name), "image") {
			imageModels = append(imageModels, name)
		} else {
			otherModels = append(otherModels, name)
		}
	}

	if len(imageModels) > 0 {
		fmt.Println("\nImage Generation Models:")
		for _, name := range imageModels {
			fmt.Printf("  - %s\n", name)
		}
	}

	if len(otherModels) > 0 {
		fmt.Println("\nOther Available Models:")
		for _, name := range otherModels {
			fmt.Printf("  - %s\n", name)
		}
	}

	fmt.Println("")
	return nil
}

func generateImagesConcurrent(jobs []AdImageJob, outputDir, apiKey, modelName string, workers int) {
	ctx := context.Background()
	jobChan := make(chan AdImageJob, len(jobs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	processed := 0
	skipped := 0
	errors := 0

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Create Gemini client for this worker
			client, err := genai.NewClient(ctx, &genai.ClientConfig{
				APIKey: apiKey,
			})
			if err != nil {
				logger.Error("Failed to create Gemini client", "workerID", workerID, "error", err)
				return
			}

			for job := range jobChan {
				// Build output path
				adDir := filepath.Join(outputDir, fmt.Sprintf("%d", job.AdID))
				imagePath := filepath.Join(adDir, fmt.Sprintf("%d.png", job.ImageNum))

				// Skip if image already exists
				if _, err := os.Stat(imagePath); err == nil {
					mu.Lock()
					skipped++
					mu.Unlock()
					continue
				}

				// Build prompt for this image
				prompt := buildImagePrompt(job.AdDetail, job.ImageNum)

				// Generate image with retry logic
				maxRetries := 5
				var err error
				for attempt := 0; attempt < maxRetries; attempt++ {
					err = generateImage(ctx, client, prompt, imagePath, modelName, job.AdID, job.ImageNum)
					if err == nil {
						break
					}

					// Check if it's a rate limit error (429)
					errStr := err.Error()
					if strings.Contains(errStr, "429") || strings.Contains(errStr, "RESOURCE_EXHAUSTED") || strings.Contains(errStr, "quota") {
						// Extract retry delay from error message if present
						retryDelay := extractRetryDelay(errStr)
						if retryDelay == 0 {
							// Default exponential backoff: 2s, 4s, 8s, 16s, 32s
							retryDelay = time.Duration(1<<uint(attempt+1)) * time.Second
						}
						// Add a small buffer
						retryDelay += 1 * time.Second

						if attempt < maxRetries-1 {
							logger.Debug("Rate limited, retrying",
								"workerID", workerID, "adID", job.AdID, "imageNum", job.ImageNum,
								"attempt", attempt+1, "retryDelay", retryDelay)
							time.Sleep(retryDelay)
							continue
						}
					}

					// For non-rate-limit errors, don't retry
					break
				}

				if err != nil {
					logger.Error("Error generating image after retries",
						"workerID", workerID, "adID", job.AdID, "imageNum", job.ImageNum, "error", err)
					mu.Lock()
					errors++
					mu.Unlock()
				} else {
					mu.Lock()
					processed++
					if processed%10 == 0 {
						fmt.Printf("[Progress] Processed: %d, Skipped: %d, Errors: %d\n",
							processed, skipped, errors)
					}
					mu.Unlock()
				}

				// Rate limit buffer - longer delay for Imagen models
				if strings.HasPrefix(modelName, "imagen-") {
					// Imagen models have stricter rate limits (10/min), so wait longer
					time.Sleep(7 * time.Second) // ~8-9 requests per minute
				} else {
					time.Sleep(2 * time.Second)
				}
			}
		}(i)
	}

	// Send jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	wg.Wait()

	fmt.Printf("[Final] Processed: %d, Skipped: %d, Errors: %d\n",
		processed, skipped, errors)
}

func buildImagePrompt(adDetail *ad.AdDetail, imageNum int) string {
	// Get category fields to determine which fields to include
	fields := ad.GetCategoryFields(adDetail.AdCategoryID)

	// Build prompt similar to embedding prompt but optimized for image generation
	var promptParts []string

	// Start with a clear image description request
	promptParts = append(promptParts, "Generate a realistic, high-quality product photograph showing:")

	// Add title and description (similar to embedding prompt)
	if adDetail.Title != "" {
		promptParts = append(promptParts, fmt.Sprintf("Title: %s", adDetail.Title))
	}
	if adDetail.Description != "" {
		// Truncate description if too long for prompt
		desc := adDetail.Description
		if len(desc) > 300 {
			desc = desc[:300] + "..."
		}
		promptParts = append(promptParts, fmt.Sprintf("Description: %s", desc))
	}

	// Add spec fields dynamically - each field generates its own snippet
	// This includes all fields including part_category and part_subcategory
	for _, field := range fields {
		fieldValues, exists := adDetail.SpecData[field.Name()]
		if !exists || len(fieldValues) == 0 {
			continue
		}

		snippet := field.EmbeddingPrompt(fieldValues)
		if snippet != "" {
			promptParts = append(promptParts, snippet)
		}
	}

	// Add location, price (similar to embedding prompt)
	if adDetail.City != "" || adDetail.AdminArea != "" || adDetail.Country != "" {
		locationParts := []string{}
		if adDetail.City != "" {
			locationParts = append(locationParts, adDetail.City)
		}
		if adDetail.AdminArea != "" {
			locationParts = append(locationParts, adDetail.AdminArea)
		}
		if adDetail.Country != "" {
			locationParts = append(locationParts, adDetail.Country)
		}
		if len(locationParts) > 0 {
			promptParts = append(promptParts, fmt.Sprintf("Location: %s", strings.Join(locationParts, ", ")))
		}
	}

	if adDetail.Price > 0 {
		promptParts = append(promptParts, fmt.Sprintf("Price: $%.0f", adDetail.Price))
	}

	// For multiple images, ensure consistency by referencing the same item
	if imageNum > 1 {
		// Build a consistency instruction based on the item description
		consistencyNote := fmt.Sprintf(
			"This is image %d of %d for the same item. Show the exact same item from a different angle, perspective, or detail view. Maintain visual consistency: same colors, same model/type, same condition, same brand if specified. Only change the camera angle or what part of the item is visible.",
			imageNum, adDetail.ImageCount)
		promptParts = append(promptParts, consistencyNote)
	}

	// Add style and quality requirements
	styleNote := "The image should be photographed in the style of an amateur photographer - like a regular person taking pictures with their phone camera. Use natural, casual lighting and composition. Do NOT include any ad text like title, description, or price written on the image itself. Branding text is acceptable if it's naturally part of the item or appears in the background."

	promptParts = append(promptParts, styleNote)
	promptParts = append(promptParts, "Generate a 4:3 aspect ratio image.")

	return strings.Join(promptParts, "\n")
}

// extractRetryDelay tries to extract retry delay from error message
// Looks for patterns like "Please retry in 440.875344ms" or "retryDelay:59s"
func extractRetryDelay(errStr string) time.Duration {
	// Look for "retry in" pattern (e.g., "Please retry in 440.875344ms")
	if idx := strings.Index(errStr, "retry in"); idx != -1 {
		remaining := strings.TrimSpace(errStr[idx+8:])
		// Find end of duration (space, comma, period, or end of string)
		end := len(remaining)
		for i, r := range remaining {
			if r == ' ' || r == ',' || r == '.' || r == ']' || r == '}' || r == '\n' {
				if i > 0 {
					end = i
					break
				}
			}
		}
		durationStr := remaining[:end]
		duration, err := time.ParseDuration(durationStr)
		if err == nil && duration > 0 {
			return duration
		}
	}

	// Look for "retryDelay:" pattern (e.g., "retryDelay:59s")
	if idx := strings.Index(errStr, "retryDelay:"); idx != -1 {
		remaining := strings.TrimSpace(errStr[idx+11:])
		// Find end of duration (space, comma, or end of string)
		end := len(remaining)
		for i, r := range remaining {
			if r == ' ' || r == ',' || r == ']' || r == '}' || r == '\n' {
				end = i
				break
			}
		}
		durationStr := remaining[:end]
		duration, err := time.ParseDuration(durationStr)
		if err == nil && duration > 0 {
			return duration
		}
	}

	return 0
}

func generateImage(ctx context.Context, client *genai.Client, prompt, outputPath, modelName string, adID, imageNum int) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var imgData []byte

	// Check if this is an Imagen model (starts with "imagen-")
	if strings.HasPrefix(modelName, "imagen-") {
		// Use GenerateImages API for Imagen models
		config := &genai.GenerateImagesConfig{
			AspectRatio:    "4:3",
			NumberOfImages: 1,
		}

		result, err := client.Models.GenerateImages(ctx, modelName, prompt, config)
		if err != nil {
			return fmt.Errorf("generate images failed: %w", err)
		}

		if len(result.GeneratedImages) == 0 {
			return fmt.Errorf("no images generated")
		}

		// Get the first generated image
		generatedImage := result.GeneratedImages[0]
		if generatedImage.Image == nil {
			return fmt.Errorf("generated image has no image data")
		}

		imgData = generatedImage.Image.ImageBytes
		if len(imgData) == 0 {
			return fmt.Errorf("no image bytes found in response")
		}
	} else {
		// Use GenerateContent API for Gemini models
		result, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), nil)
		if err != nil {
			return fmt.Errorf("generate content failed: %w", err)
		}

		if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			return fmt.Errorf("no content generated")
		}

		// Find the first image part (InlineData.Data is raw bytes)
		for _, part := range result.Candidates[0].Content.Parts {
			if part.InlineData != nil {
				imgData = part.InlineData.Data
				break
			}
		}

		if len(imgData) == 0 {
			return fmt.Errorf("no image data found in response")
		}
	}

	// Write image to file
	if err := os.WriteFile(outputPath, imgData, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}
