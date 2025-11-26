package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"context"
	"net/url"

	"github.com/chai2010/webp"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/user"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type TestAd struct {
	ID         int
	Title      string
	ImageCount int
}

type ImageToUpload struct {
	AdID     int
	ImageNum int
	Size     string
	Data     []byte
	Path     string
}

func main() {
	// Initialize logging
	if err := logger.Init(config.LogLevel, config.LogFormat, config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	// Initialize encryption keys (required for user decryption)
	config.SecurityCheck()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

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

	// Look up test user by name
	testUser, err := user.GetUserByName("test")
	if err != nil {
		logger.Fatal("Failed to find test user", "error", err, "name", "test")
	}

	// Query test ads for the test user
	testAds, err := getTestAds(database, testUser.ID)
	if err != nil {
		logger.Fatal("Failed to get test ads", "error", err)
	}

	logger.Info("Found test ads to process", "count", len(testAds))
	fmt.Printf("Found %d test ads to process\n", len(testAds))

	// Generate images for all test ads
	fmt.Println("Starting image generation...")
	imagesToUpload := generateImagesForAds(testAds)

	fmt.Printf("Generated %d images to upload\n", len(imagesToUpload))

	// Batch upload to MinIO
	if err := batchUploadToMinIO(imagesToUpload); err != nil {
		logger.Fatal("Failed to upload images to MinIO", "error", err)
	}

	logger.Info("Test image generation and upload complete")
	fmt.Println("Test image generation and upload complete!")
}

// getTestAds queries the database for test ads for a specific user
func getTestAds(database *sql.DB, userID int) ([]TestAd, error) {
	query := `
		SELECT id, title, image_count 
		FROM Ad 
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY id
	`

	rows, err := database.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query test ads: %v", err)
	}
	defer rows.Close()

	var testAds []TestAd
	for rows.Next() {
		var ad TestAd
		if err := rows.Scan(&ad.ID, &ad.Title, &ad.ImageCount); err != nil {
			return nil, fmt.Errorf("failed to scan test ad: %v", err)
		}
		testAds = append(testAds, ad)
	}

	return testAds, nil
}

// generateImagesForAds generates images for all test ads
func generateImagesForAds(testAds []TestAd) []ImageToUpload {
	var imagesToUpload []ImageToUpload
	sizes := []struct {
		Width   int
		Suffix  string
		Quality float32
	}{
		{160, "160w", 60},
		{480, "480w", 70},
		{1200, "1200w", 80},
	}

	processedAds := 0
	for _, ad := range testAds {
		// Generate between 1 and 5 images per ad (matching the image_count)
		numImages := ad.ImageCount
		if numImages < 1 {
			numImages = 1
		}
		if numImages > 5 {
			numImages = 5
		}

		for i := 0; i < numImages; i++ {
			// Generate different sizes directly (no base image)
			for _, sz := range sizes {
				w := sz.Width
				h := w * 800 / 1200 // Maintain 1200:800 aspect ratio

				// Generate image with correct dimensions
				pngBuf, err := generateTestAdImage(ad.ID, i+1, w, h)
				if err != nil {
					logger.Error("Failed to generate image for ad",
						"adID", ad.ID, "error", err)
					continue
				}

				// Decode the PNG
				img, _, err := image.Decode(bytes.NewReader(pngBuf.Bytes()))
				if err != nil {
					logger.Error("Failed to decode image for ad",
						"adID", ad.ID, "error", err)
					continue
				}

				// Encode to WebP for better compression
				var webpBuf bytes.Buffer
				opt := &webp.Options{Lossless: false, Quality: sz.Quality}
				if err := webp.Encode(&webpBuf, img, opt); err != nil {
					logger.Error("WebP encode error for ad",
						"adID", ad.ID, "error", err)
					continue
				}

				// Create MinIO object path
				objectPath := filepath.Join(
					fmt.Sprintf("%d", ad.ID),
					fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
				)

				imagesToUpload = append(imagesToUpload, ImageToUpload{
					AdID:     ad.ID,
					ImageNum: i + 1,
					Size:     sz.Suffix,
					Data:     webpBuf.Bytes(),
					Path:     objectPath,
				})
			}
		}
		processedAds++
		if processedAds%50 == 0 {
			fmt.Printf("Generated images for %d/%d ads\n", processedAds, len(testAds))
		}
	}

	fmt.Printf("Completed image generation for %d ads\n", processedAds)
	return imagesToUpload
}

// generateTestAdImage creates a simple test ad image with background color and centered text
func generateTestAdImage(adID, imageNum, width, height int) (*bytes.Buffer, error) {
	// Create a new RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Generate a consistent background color for this ad using muted/semi-opaque primary colors
	bgColor := getAdBackgroundColor(adID)

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Add a border
	borderColor := color.RGBA{200, 200, 200, 255}
	for x := 0; x < width; x++ {
		img.Set(x, 0, borderColor)
		img.Set(x, height-1, borderColor)
	}
	for y := 0; y < height; y++ {
		img.Set(0, y, borderColor)
		img.Set(width-1, y, borderColor)
	}

	// Create text content
	textContent := fmt.Sprintf("Ad: %d, Image: %d, Size: %d,%d", adID, imageNum, width, height)

	// Add text
	textColor := color.RGBA{255, 255, 255, 255} // White text for dark background
	addText(img, textContent, textColor, width, height)

	// Encode to PNG first
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %v", err)
	}

	return &pngBuf, nil
}

// getAdBackgroundColor returns a consistent muted/semi-opaque primary color for a given ad ID
func getAdBackgroundColor(adID int) color.RGBA {
	// Define muted/semi-opaque primary colors
	mutedColors := []color.RGBA{
		{120, 60, 60, 200},  // Muted red
		{60, 120, 60, 200},  // Muted green
		{60, 60, 120, 200},  // Muted blue
		{120, 120, 60, 200}, // Muted yellow
		{120, 60, 120, 200}, // Muted magenta
		{60, 120, 120, 200}, // Muted cyan
		{100, 80, 60, 200},  // Muted orange
		{80, 60, 100, 200},  // Muted purple
		{80, 100, 60, 200},  // Muted lime
		{100, 60, 80, 200},  // Muted pink
		{60, 100, 80, 200},  // Muted teal
		{100, 80, 80, 200},  // Muted brown
	}

	// Use ad ID to consistently select a color
	colorIndex := adID % len(mutedColors)
	return mutedColors[colorIndex]
}

// addText adds text to an image centered with gomono font
func addText(img *image.RGBA, text string, textColor color.Color, imgWidth, imgHeight int) {
	// Load and configure gomono font
	fontData, err := opentype.Parse(gomono.TTF)
	if err != nil {
		logger.Error("Failed to parse gomono font", "error", err)
		return
	}

	// Set font size based on image width
	var fontSize float64 = float64(imgWidth) * 0.0333

	fontFace, err := opentype.NewFace(fontData, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		logger.Error("Failed to create font face", "error", err)
		return
	}
	defer fontFace.Close()

	// Calculate text dimensions for centering
	advance := font.MeasureString(fontFace, text)
	textWidth := int(advance >> 6) // Convert fixed.Int26_6 to pixels (/64)

	ascent := int(fontFace.Metrics().Ascent >> 6)
	descent := int(fontFace.Metrics().Descent >> 6)
	textHeight := ascent + descent

	// Center the text
	x := (imgWidth - textWidth) / 2
	y := (imgHeight - textHeight) / 2 // Top of text bounding box

	baselineY := y + ascent

	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baselineY)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: fontFace,
		Dot:  point,
	}
	d.DrawString(text)
}

// batchUploadToMinIO uploads all images to MinIO in batches
func batchUploadToMinIO(imagesToUpload []ImageToUpload) error {
	if config.MinIOAPIURL == "" || config.MinIOUsername == "" || config.MinIOPassword == "" {
		return fmt.Errorf("MinIO credentials not set in env vars")
	}

	if config.MinIOBucketName == "" {
		return fmt.Errorf("MinIO bucket name not set in env vars")
	}

	// Parse the API URL to extract endpoint
	endpointURL, err := url.Parse(config.MinIOAPIURL)
	if err != nil {
		return fmt.Errorf("invalid MinIO API URL: %w", err)
	}

	minioClient, err := minio.New(endpointURL.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinIOUsername, config.MinIOPassword, ""),
		Secure: endpointURL.Scheme == "https",
	})
	if err != nil {
		return fmt.Errorf("MinIO client error: %v", err)
	}

	// Upload in batches with concurrency control
	batchSize := 10 // Upload 10 images concurrently
	semaphore := make(chan struct{}, batchSize)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errorCount := 0

	fmt.Printf("Starting batch upload of %d images with batch size %d\n", len(imagesToUpload), batchSize)

	for i, img := range imagesToUpload {
		wg.Add(1)
		go func(index int, imageToUpload ImageToUpload) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			// Retry upload up to 3 times
			maxRetries := 3
			var err error
			ctx := context.Background()
			for attempt := 0; attempt < maxRetries; attempt++ {
				_, err = minioClient.PutObject(ctx, config.MinIOBucketName, imageToUpload.Path,
					bytes.NewReader(imageToUpload.Data), int64(len(imageToUpload.Data)),
					minio.PutObjectOptions{ContentType: "image/webp"})

				if err == nil {
					// Success
					mu.Lock()
					successCount++
					if successCount%50 == 0 {
						fmt.Printf("Uploaded %d/%d images\n", successCount, len(imagesToUpload))
					}
					mu.Unlock()
					return
				}

				// Log retry attempt
				if attempt < maxRetries-1 {
					logger.Warn("MinIO upload attempt failed, retrying",
						"attempt", attempt+1, "path", imageToUpload.Path,
						"error", err)
					// Wait a bit before retry (exponential backoff)
					time.Sleep(time.Duration(attempt+1) * time.Second)
				}
			}

			// All retries failed
			mu.Lock()
			logger.Error("MinIO upload failed after all retries",
				"attempts", maxRetries, "path", imageToUpload.Path, "error", err)
			errorCount++
			mu.Unlock()
		}(i, img)
	}

	wg.Wait()

	fmt.Printf("Upload complete: %d successful, %d failed\n", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("failed to upload %d images", errorCount)
	}

	return nil
}
