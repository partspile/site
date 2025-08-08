package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/parts-pile/site/config"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"gopkg.in/kothar/go-backblaze.v0"
)

// generateAdImage creates a placeholder image for an ad with the given title
func generateAdImage(title string, width, height int, adID int, imageNum int) (*bytes.Buffer, error) {
	// Create a new RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Generate a random background color (darker palette)
	rand.Seed(time.Now().UnixNano())
	bgColor := color.RGBA{
		R: uint8(50 + rand.Intn(100)), // Darker colors
		G: uint8(50 + rand.Intn(100)),
		B: uint8(50 + rand.Intn(100)),
		A: 255,
	}

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

	// Create text content with ad ID, image number, and resolution
	textContent := fmt.Sprintf("Ad #%d - Image %d\n%s\n%d×%d", adID, imageNum, title, width, height)

	// Add text
	textColor := color.RGBA{255, 255, 255, 255} // White text for dark background
	addText(img, textContent, textColor, width/2, height/2)

	// Encode to PNG first
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %v", err)
	}

	return &pngBuf, nil
}

// addText adds text to an image at the specified position
func addText(img *image.RGBA, text string, textColor color.Color, x, y int) {
	// Use basic font for simplicity
	f := basicfont.Face7x13

	// Split text into lines
	lines := strings.Split(text, "\n")

	// Calculate total height of all lines with larger spacing
	lineHeight := f.Height + 20 // Add more spacing between lines
	totalHeight := len(lines) * lineHeight

	// Start position (center vertically)
	startY := y - totalHeight/2

	for i, line := range lines {
		// Center the text horizontally
		textWidth := len(line) * f.Width
		startX := x - textWidth/2

		// Draw the text using font.Drawer
		point := fixed.Point26_6{
			X: fixed.Int26_6(startX * 64),
			Y: fixed.Int26_6((startY + i*lineHeight) * 64),
		}

		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(textColor),
			Face: f,
			Dot:  point,
		}
		d.DrawString(line)
	}
}

// uploadAdImagesToB2 uploads generated images for an ad to B2
func uploadAdImagesToB2(adID int, numImages int, title string) error {
	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey
	if accountID == "" || appKey == "" || keyID == "" {
		log.Println("B2 credentials not set in env vars")
		return fmt.Errorf("B2 credentials not set")
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		log.Println("B2 auth error:", err)
		return fmt.Errorf("B2 auth error: %v", err)
	}

	bucket, err := b2.Bucket(config.B2BucketName)
	if err != nil {
		log.Println("B2 bucket error:", err)
		return fmt.Errorf("B2 bucket error: %v", err)
	}

	// Generate between 1 and 5 images per ad
	if numImages < 1 {
		numImages = 1
	}
	if numImages > 5 {
		numImages = 5
	}

	// Use the provided title, or fallback to generic if empty
	if title == "" {
		title = fmt.Sprintf("Ad #%d", adID)
	}

	sizes := []struct {
		Width   int
		Suffix  string
		Quality float32
	}{
		{160, "160w", 60},
		{480, "480w", 70},
		{1200, "1200w", 80},
	}

	for i := 0; i < numImages; i++ {
		// Generate base image at 1200px width
		pngBuf, err := generateAdImage(title, 1200, 800, adID, i+1)
		if err != nil {
			log.Printf("Failed to generate image for ad %d: %v", adID, err)
			continue
		}

		// Decode the PNG
		img, _, err := image.Decode(bytes.NewReader(pngBuf.Bytes()))
		if err != nil {
			log.Printf("Failed to decode image for ad %d: %v", adID, err)
			continue
		}

		bounds := img.Bounds()

		// Generate different sizes
		for _, sz := range sizes {
			w := sz.Width
			h := bounds.Dy() * w / bounds.Dx()
			dst := image.NewRGBA(image.Rect(0, 0, w, h))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

			// Encode to WebP
			var webpBuf bytes.Buffer
			opt := &webp.Options{Lossless: false, Quality: sz.Quality}
			if err := webp.Encode(&webpBuf, dst, opt); err != nil {
				log.Printf("WebP encode error for ad %d: %v", adID, err)
				continue
			}

			// Upload to B2
			b2Path := filepath.Join(
				fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
			)

			_, err = bucket.UploadTypedFile(b2Path, "image/webp", nil, bytes.NewReader(webpBuf.Bytes()))
			if err != nil {
				log.Printf("B2 upload error for %s: %v", b2Path, err)
			} else {
				fmt.Printf("Uploaded %s for ad %d\n", b2Path, adID)
			}
		}
	}

	return nil
}
