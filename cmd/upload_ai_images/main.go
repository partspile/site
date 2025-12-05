package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/chai2010/webp"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/minio"
	"golang.org/x/image/draw"
)

type ImageToUpload struct {
	AdID     int
	ImageNum int
	Size     string
	Data     []byte
}

func main() {
	var inputDir string
	var workers int

	flag.StringVar(&inputDir, "input", "generated_images", "Input directory containing generated images")
	flag.IntVar(&workers, "workers", 5, "Number of concurrent workers")
	flag.Parse()

	// Initialize logging
	if err := logger.Init(config.LogLevel, config.LogFormat, config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	// Initialize encryption keys (required for user decryption)
	config.SecurityCheck()

	// Initialize MinIO
	if err := minio.Init(); err != nil {
		logger.Fatal("Failed to initialize MinIO", "error", err)
	}

	// Find all images to process
	imagesToUpload, err := findImagesToUpload(inputDir)
	if err != nil {
		logger.Fatal("Failed to find images", "error", err)
	}

	if len(imagesToUpload) == 0 {
		fmt.Println("No images found to upload")
		return
	}

	fmt.Printf("Found %d images to resize and upload\n", len(imagesToUpload))

	// Process and upload images
	if err := processAndUploadImages(imagesToUpload, workers); err != nil {
		logger.Fatal("Failed to process images", "error", err)
	}

	fmt.Println("Upload complete!")
}

func findImagesToUpload(inputDir string) ([]string, error) {
	var imagePaths []string

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for PNG files
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".png") {
			imagePaths = append(imagePaths, path)
		}

		return nil
	})

	return imagePaths, err
}

func processAndUploadImages(imagePaths []string, workers int) error {
	jobChan := make(chan string, len(imagePaths))
	var wg sync.WaitGroup
	var mu sync.Mutex
	processed := 0
	errors := 0

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for imagePath := range jobChan {
				if err := processImage(imagePath); err != nil {
					logger.Error("Failed to process image",
						"workerID", workerID, "path", imagePath, "error", err)
					mu.Lock()
					errors++
					mu.Unlock()
				} else {
					mu.Lock()
					processed++
					if processed%10 == 0 {
						fmt.Printf("[Progress] Processed: %d, Errors: %d\n",
							processed, errors)
					}
					mu.Unlock()
				}
			}
		}(i)
	}

	// Send jobs
	for _, path := range imagePaths {
		jobChan <- path
	}
	close(jobChan)

	wg.Wait()

	fmt.Printf("[Final] Processed: %d, Errors: %d\n", processed, errors)

	if errors > 0 {
		return fmt.Errorf("failed to process %d images", errors)
	}

	return nil
}

func processImage(imagePath string) error {
	// Parse ad ID and image number from path: generated_images/<ad_id>/<image_num>.png
	parts := strings.Split(imagePath, string(filepath.Separator))
	if len(parts) < 3 {
		return fmt.Errorf("invalid image path format: %s", imagePath)
	}

	adIDStr := parts[len(parts)-2]
	imageName := parts[len(parts)-1]

	adID, err := strconv.Atoi(adIDStr)
	if err != nil {
		return fmt.Errorf("invalid ad ID in path: %w", err)
	}

	// Extract image number from filename (e.g., "1.png" -> 1)
	imageNumStr := strings.TrimSuffix(imageName, ".png")
	imageNum, err := strconv.Atoi(imageNumStr)
	if err != nil {
		return fmt.Errorf("invalid image number in filename: %w", err)
	}

	// Read source image
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Define sizes to generate
	sizes := []struct {
		Width   int
		Suffix  string
		Quality float32
	}{
		{160, "160w", 60},
		{480, "480w", 70},
		{1200, "1200w", 80},
	}

	bounds := img.Bounds()
	sourceWidth := bounds.Dx()
	sourceHeight := bounds.Dy()

	// Process each size
	for _, sz := range sizes {
		w := sz.Width
		// Maintain 4:3 aspect ratio (crop if needed to maintain aspect ratio)
		h := w * 3 / 4

		// Resize image
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		
		// Calculate source rectangle to maintain aspect ratio
		// If source is wider than 4:3, crop width; if taller, crop height
		sourceAspect := float64(sourceWidth) / float64(sourceHeight)
		targetAspect := float64(w) / float64(h) // 4:3 = 1.333...

		var srcRect image.Rectangle
		if sourceAspect > targetAspect {
			// Source is wider - crop width
			croppedWidth := int(float64(sourceHeight) * targetAspect)
			x0 := (sourceWidth - croppedWidth) / 2
			srcRect = image.Rect(x0, 0, x0+croppedWidth, sourceHeight)
		} else {
			// Source is taller - crop height
			croppedHeight := int(float64(sourceWidth) / targetAspect)
			y0 := (sourceHeight - croppedHeight) / 2
			srcRect = image.Rect(0, y0, sourceWidth, y0+croppedHeight)
		}

		draw.CatmullRom.Scale(dst, dst.Bounds(), img, srcRect, draw.Over, nil)

		// Encode to WebP
		var webpBuf bytes.Buffer
		opt := &webp.Options{Lossless: false, Quality: sz.Quality}
		if err := webp.Encode(&webpBuf, dst, opt); err != nil {
			return fmt.Errorf("failed to encode WebP: %w", err)
		}

		// Upload to MinIO
		if err := minio.PutImageObject(adID, imageNum, sz.Suffix, webpBuf.Bytes()); err != nil {
			return fmt.Errorf("failed to upload to MinIO: %w", err)
		}
	}

	return nil
}

