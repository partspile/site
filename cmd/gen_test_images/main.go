package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"gopkg.in/kothar/go-backblaze.v0"
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
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	dbFile := strings.TrimPrefix(config.DatabaseURL, "file:")

	// Initialize database
	if err := db.Init(dbFile); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Open DB for direct access
	database, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer database.Close()

	// Query test ads (userId=1)
	testAds, err := getTestAds(database)
	if err != nil {
		log.Fatalf("Failed to get test ads: %v", err)
	}

	fmt.Printf("Found %d test ads to process\n", len(testAds))

	// Generate images for all test ads
	fmt.Println("Starting image generation...")
	imagesToUpload := generateImagesForAds(testAds)

	fmt.Printf("Generated %d images to upload\n", len(imagesToUpload))

	// Batch upload to B2
	if err := batchUploadToB2(imagesToUpload); err != nil {
		log.Fatalf("Failed to upload images to B2: %v", err)
	}

	fmt.Println("Test image generation and upload complete!")
}

// getTestAds queries the database for test ads (userId=1)
func getTestAds(database *sql.DB) ([]TestAd, error) {
	query := `
		SELECT id, title, image_count 
		FROM Ad 
		WHERE user_id = 1 AND deleted_at IS NULL
		ORDER BY id
	`

	rows, err := database.Query(query)
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
					log.Printf("Failed to generate image for ad %d: %v", ad.ID, err)
					continue
				}

				// Decode the PNG
				img, _, err := image.Decode(bytes.NewReader(pngBuf.Bytes()))
				if err != nil {
					log.Printf("Failed to decode image for ad %d: %v", ad.ID, err)
					continue
				}

				// Encode to WebP for better compression
				var webpBuf bytes.Buffer
				opt := &webp.Options{Lossless: false, Quality: sz.Quality}
				if err := webp.Encode(&webpBuf, img, opt); err != nil {
					log.Printf("WebP encode error for ad %d: %v", ad.ID, err)
					continue
				}

				// Create B2 path
				b2Path := filepath.Join(
					fmt.Sprintf("%d", ad.ID),
					fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
				)

				imagesToUpload = append(imagesToUpload, ImageToUpload{
					AdID:     ad.ID,
					ImageNum: i + 1,
					Size:     sz.Suffix,
					Data:     webpBuf.Bytes(),
					Path:     b2Path,
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
	addText(img, textContent, textColor, width/2, height/2, width, height)

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

// addText adds text to an image at the specified position with size-appropriate scaling
func addText(img *image.RGBA, text string, textColor color.Color, x, y, imgWidth, imgHeight int) {
	// Use larger font for better visibility
	f := basicfont.Face7x13

	// Calculate text width and height
	textWidth := len(text) * f.Width
	textHeight := f.Height

	// Scale text based on image size to maintain readability
	// For 1200px width, use scale 4; for 480px use scale 3; for 160px use scale 2
	var scale int
	switch {
	case imgWidth >= 1000:
		scale = 4
	case imgWidth >= 400:
		scale = 3
	default:
		scale = 2
	}

	// Center the text
	startX := x - (textWidth*scale)/2
	startY := y - (textHeight*scale)/2

	// Draw text using a simple pixel-by-pixel approach for crisp rendering
	for i, char := range text {
		charX := startX + i*f.Width*scale
		charY := startY

		// Draw each character by scaling the basic font
		drawScaledChar(img, char, charX, charY, scale, textColor)
	}
}

// drawScaledChar draws a single character scaled up for crisp rendering
func drawScaledChar(img *image.RGBA, char rune, x, y, scale int, textColor color.Color) {
	// Get the basic font face
	f := basicfont.Face7x13

	// Create a temporary image for the character
	charImg := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))

	// Draw the character to the temporary image
	point := fixed.Point26_6{
		X: fixed.Int26_6(0),
		Y: fixed.Int26_6(f.Height * 64),
	}

	d := &font.Drawer{
		Dst:  charImg,
		Src:  image.NewUniform(textColor),
		Face: f,
		Dot:  point,
	}
	d.DrawString(string(char))

	// Scale up the character by drawing each pixel multiple times
	for srcY := 0; srcY < f.Height; srcY++ {
		for srcX := 0; srcX < f.Width; srcX++ {
			// Get the pixel color from the source
			srcColor := charImg.RGBAAt(srcX, srcY)
			if srcColor.A > 0 { // Only draw non-transparent pixels
				// Draw the pixel scaled up
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						dstX := x + srcX*scale + dx
						dstY := y + srcY*scale + dy
						if dstX >= 0 && dstX < img.Bounds().Dx() && dstY >= 0 && dstY < img.Bounds().Dy() {
							img.Set(dstX, dstY, textColor)
						}
					}
				}
			}
		}
	}
}

// batchUploadToB2 uploads all images to B2 in batches
func batchUploadToB2(imagesToUpload []ImageToUpload) error {
	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey
	if accountID == "" || appKey == "" || keyID == "" {
		return fmt.Errorf("B2 credentials not set in env vars")
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		return fmt.Errorf("B2 auth error: %v", err)
	}

	bucket, err := b2.Bucket(config.B2BucketName)
	if err != nil {
		return fmt.Errorf("B2 bucket error: %v", err)
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
			for attempt := 0; attempt < maxRetries; attempt++ {
				_, err = bucket.UploadTypedFile(
					imageToUpload.Path,
					"image/webp",
					nil,
					bytes.NewReader(imageToUpload.Data),
				)

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
					log.Printf("B2 upload attempt %d failed for %s: %v, retrying...", attempt+1, imageToUpload.Path, err)
					// Wait a bit before retry (exponential backoff)
					time.Sleep(time.Duration(attempt+1) * time.Second)
				}
			}

			// All retries failed
			mu.Lock()
			log.Printf("B2 upload failed after %d attempts for %s: %v", maxRetries, imageToUpload.Path, err)
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
