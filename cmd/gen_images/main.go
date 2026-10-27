package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/imgconv"
	"github.com/partspile/site/internal/logger"
	"golang.org/x/image/draw"
)

type ImageSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FalRequest struct {
	Prompt       string     `json:"prompt"`
	ImageSize    *ImageSize `json:"image_size,omitempty"`
	OutputFormat string     `json:"output_format,omitempty"`
}

type FalImage struct {
	URL         string `json:"url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ContentType string `json:"content_type"`
}

type FalResponse struct {
	Images []FalImage `json:"images"`
}

func formatPrice(price int, priceCurrency string) string {
	return currency.Format(price, priceCurrency)
}

func formatLocation(city, adminArea, country string) string {
	parts := []string{}
	if city != "" {
		parts = append(parts, city)
	}
	if adminArea != "" {
		parts = append(parts, adminArea)
	}
	if country != "" {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}

func buildPrompt(ad ad.Ad, isHandwrittenNote bool) string {
	location := formatLocation(ad.City, ad.AdminArea, ad.Country)
	priceAmount, priceCurrency, _ := ad.PriceValue()
	price := formatPrice(priceAmount, priceCurrency)

	if isHandwrittenNote {
		return fmt.Sprintf("A hand-written note on paper or cardboard about: %s. The note mentions the title '%s', description '%s', price %s, and location %s. Written in casual handwriting, like someone selling something on craigslist or facebook marketplace. Non-professional photo quality, taken with a phone camera, natural lighting, slightly messy background.",
			ad.Title, ad.Title, ad.Description, price, location)
	}

	// Non-professional photo prompts - typical of craigslist/facebook marketplace
	promptVariations := []string{
		fmt.Sprintf("A casual photo of %s. %s. Price: %s. Location: %s. Non-professional photo taken with a phone camera, natural lighting, slightly messy background, typical of craigslist or facebook marketplace listing photos.",
			ad.Title, ad.Description, price, location),
		fmt.Sprintf("A quick snapshot photo of %s. Description: %s. Selling for %s in %s. Taken with a smartphone, amateur photography, realistic home lighting, cluttered background, authentic marketplace listing style.",
			ad.Title, ad.Description, price, location),
		fmt.Sprintf("An amateur photo of %s. %s. Price %s, located in %s. Phone camera quality, natural indoor or outdoor lighting, unprofessional composition, typical of online marketplace listings.",
			ad.Title, ad.Description, price, location),
		fmt.Sprintf("A casual marketplace photo of %s. Details: %s. Asking %s, location %s. Taken with a phone, realistic lighting, messy or cluttered background, authentic craigslist/facebook marketplace style.",
			ad.Title, ad.Description, price, location),
	}

	// Use a simple hash of ad ID to pick a variation (for consistency)
	variationIndex := ad.ID % len(promptVariations)
	return promptVariations[variationIndex]
}

func generateImage(apiKey, prompt string) ([]byte, error) {
	customSize := &ImageSize{
		Width:  1200,
		Height: 900, // 4:3 aspect ratio
	}

	requestBody := FalRequest{
		Prompt:       prompt,
		ImageSize:    customSize,
		OutputFormat: "jpeg",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := "https://fal.run/fal-ai/z-image/turbo"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Key "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var falResp FalResponse
	if err := json.Unmarshal(body, &falResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(falResp.Images) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	image := falResp.Images[0]

	// Download the image
	imgResp, err := http.Get(image.URL)
	if err != nil {
		return nil, fmt.Errorf("downloading image: %w", err)
	}
	defer imgResp.Body.Close()

	imageData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading image data: %w", err)
	}

	return imageData, nil
}

func deleteExistingImages(store imagestore.Store, adID int) error {
	return store.DeleteAd(adID)
}

func imageExists(store imagestore.Store, adID, imgIdx int) bool {
	_, err := store.Get(adID, imgIdx, "1200w")
	return err == nil
}

// resizeImage resizes an image to the specified width (4:3 aspect ratio)
func resizeImage(imageData []byte, targetWidth int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	targetHeight := targetWidth * 3 / 4
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(),
		draw.Src, nil)

	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: 85}
	if err := jpeg.Encode(&buf, dst, opts); err != nil {
		return nil, fmt.Errorf("encoding jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func saveImage(store imagestore.Store, adID, imgIdx int, imageData []byte) error {
	jpegData, err := imgconv.ToJPEG(imageData, 85)
	if err != nil {
		return fmt.Errorf("converting to jpeg: %w", err)
	}
	if err := store.Put(adID, imgIdx, "1200w", jpegData); err != nil {
		return fmt.Errorf("saving 1200w image: %w", err)
	}

	image480, err := resizeImage(jpegData, 480)
	if err != nil {
		return fmt.Errorf("resizing to 480w: %w", err)
	}
	if err := store.Put(adID, imgIdx, "480w", image480); err != nil {
		return fmt.Errorf("saving 480w image: %w", err)
	}

	image160, err := resizeImage(jpegData, 160)
	if err != nil {
		return fmt.Errorf("resizing to 160w: %w", err)
	}
	if err := store.Put(adID, imgIdx, "160w", image160); err != nil {
		return fmt.Errorf("saving 160w image: %w", err)
	}

	return nil
}

func getAds(startID, limit int) ([]ad.Ad, error) {
	query := `
		SELECT
			a.id,
			a.category_id,
			a.title,
			a.description,
			a.price,
			a.price_currency,
			a.created_at,
			a.deleted_at,
			a.user_id,
			a.image_count,
			a.location_id,
			l.city,
			l.admin_area,
			l.country,
			0 AS bookmarked
		FROM ads a
		LEFT JOIN locations l ON a.location_id = l.id
		WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL
	`
	args := []any{}
	if startID > 0 {
		query += " AND a.id >= $1"
		args = append(args, startID)
	}
	query += " ORDER BY a.id"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var ads []ad.Ad
	var err error
	if len(args) > 0 {
		err = db.Select(&ads, query, args...)
	} else {
		err = db.Select(&ads, query)
	}
	return ads, err
}

func main() {
	numAds := flag.Int("n", 0, "Number of ads to process (0 = all)")
	startID := flag.Int("s", 0, "Starting ad ID (process ads with ID >= this value)")
	noOverwrite := flag.Bool("x", false, "Don't overwrite existing files")
	flag.Parse()

	// Initialize logger
	if err := logger.Init("info", "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Get API key
	apiKey := os.Getenv("FAL_API_KEY")
	if apiKey == "" {
		logger.Fatal("FAL_API_KEY environment variable is required")
	}

	// Initialize database
	if err := db.Init(config.DatabaseURL); err != nil {
		logger.Fatal("Failed to open database", "error", err)
	}
	defer db.Close()

	store, err := imagestore.NewDefault()
	if err != nil {
		logger.Fatal("Failed to initialize image store", "error", err)
	}
	logger.Info("Using MinIO image store", "bucket", config.MinIOBucketName)

	// Get ads
	logger.Info("Fetching ads from database...")
	ads, err := getAds(*startID, *numAds)
	if err != nil {
		logger.Fatal("Failed to fetch ads", "error", err)
	}

	if len(ads) == 0 {
		logger.Info("No ads found")
		return
	}

	logger.Info("Found ads", "count", len(ads))

	// Track total images generated for handwritten note logic
	totalImagesGenerated := 0

	// Process each ad
	for _, ad := range ads {
		logger.Info("Processing ad", "id", ad.ID, "title", ad.Title, "image_count", ad.ImageCount)

		// Delete existing images if -x is not set
		if !*noOverwrite {
			if err := deleteExistingImages(store, ad.ID); err != nil {
				logger.Error("Failed to delete existing images", "ad_id", ad.ID, "error", err)
				continue
			}
		}

		// Generate images for this ad (using 1-based indexing: 1, 2, 3, ...)
		for imgIdx := 1; imgIdx <= ad.ImageCount; imgIdx++ {
			// Check if image already exists and -x is set
			if *noOverwrite && imageExists(store, ad.ID, imgIdx) {
				logger.Info("Skipping existing image", "ad_id", ad.ID, "img_idx", imgIdx)
				totalImagesGenerated++
				continue
			}

			// Determine if this should be a handwritten note (every 30 images)
			isHandwrittenNote := (totalImagesGenerated+1)%30 == 0

			// Build prompt
			prompt := buildPrompt(ad, isHandwrittenNote)

			logger.Info("Generating image", "ad_id", ad.ID, "img_idx", imgIdx, "handwritten_note", isHandwrittenNote)

			// Generate image
			imageData, err := generateImage(apiKey, prompt)
			if err != nil {
				logger.Error("Failed to generate image", "ad_id", ad.ID, "img_idx", imgIdx, "error", err)
				continue
			}

			// Save image (imgIdx is already 1-based)
			if err := saveImage(store, ad.ID, imgIdx, imageData); err != nil {
				logger.Error("Failed to save image", "ad_id", ad.ID, "img_idx", imgIdx, "error", err)
				continue
			}

			logger.Info("Saved images", "ad_id", ad.ID, "img_idx", imgIdx, "sizes", "1200w, 480w, 160w")
			totalImagesGenerated++

			// Small delay to avoid rate limiting
			time.Sleep(500 * time.Millisecond)
		}
	}

	logger.Info("Image generation complete", "total_images", totalImagesGenerated)
}
