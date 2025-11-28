package handlers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"strconv"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/minio"
	"github.com/parts-pile/site/ui"
	"golang.org/x/image/draw"
)

// uploadAdImagesToMinIO uploads user-uploaded images to MinIO with multiple sizes
func uploadAdImagesToMinIO(adID int, files []*multipart.FileHeader) {
	logger.Info("Starting upload",
		"component", "MinIO", "adID", adID, "imageCount", len(files))

	sizes := []struct {
		Width   int
		Suffix  string
		Quality float32
	}{
		{160, "160w", 60},
		{480, "480w", 70},
		{1200, "1200w", 80},
	}

	successCount := 0
	totalExpected := len(files) * len(sizes)

	for i, fileHeader := range files {
		logger.Debug("Processing image",
			"component", "MinIO", "imageIndex", i+1,
			"totalImages", len(files), "adID", adID)

		file, err := fileHeader.Open()
		if err != nil {
			logger.Error("Failed to open file",
				"error", err, "component", "MinIO", "imageIndex",
				i+1, "adID", adID)
			continue
		}

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(file); err != nil {
			logger.Error("Failed to read file",
				"error", err, "component", "MinIO", "imageIndex",
				i+1, "adID", adID)
			file.Close()
			continue
		}
		file.Close()

		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			logger.Error("Failed to decode image",
				"error", err, "component", "MinIO", "imageIndex",
				i+1, "adID", adID)
			continue
		}

		bounds := img.Bounds()
		logger.Debug("Image dimensions",
			"component", "MinIO", "imageIndex", i+1, "adID", adID,
			"width", bounds.Dx(), "height", bounds.Dy())

		for _, sz := range sizes {
			w := sz.Width
			h := bounds.Dy() * w / bounds.Dx()
			dst := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

			var webpBuf bytes.Buffer
			opt := &webp.Options{Lossless: false, Quality: sz.Quality}
			if err := webp.Encode(&webpBuf, dst, opt); err != nil {
				logger.Error("WebP encode error",
					"error", err, "component", "MinIO",
					"imageIndex", i+1, "size", sz.Suffix, "adID",
					adID)
				continue
			}

			logger.Debug("Uploading to MinIO",
				"component", "MinIO", "suffix", sz.Suffix,
				"adID", adID, "imageIndex", i+1)
			err = minio.PutImageObject(adID, i+1, sz.Suffix, webpBuf.Bytes())
			if err != nil {
				logger.Error("Upload failed",
					"error", err, "component", "MinIO",
					"suffix", sz.Suffix, "adID", adID, "imageIndex", i+1)
			} else {
				logger.Debug("Uploaded successfully",
					"component", "MinIO", "suffix", sz.Suffix,
					"adID", adID, "imageIndex", i+1)
				successCount++
			}
		}
	}

	logger.Info("Upload complete",
		"component", "MinIO", "adID", adID, "successCount",
		successCount, "totalExpected", totalExpected)
}

// HandleAdImage serves ad detail carousel images with navigation
func HandleAdImage(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	idx, err := strconv.Atoi(c.Params("idx"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}

	// Get image count from query parameter (passed by thumbnails/nav buttons)
	imageCountStr := c.Query("count", "1")
	imageCount, err := strconv.Atoi(imageCountStr)
	if err != nil || imageCount < 0 {
		imageCount = 1 // Default to 1 if invalid
	}

	return render(c, ui.CarouselImageContainer(adID, imageCount, idx))
}

// HandleAdGridImage serves grid view images with navigation
func HandleAdGridImage(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	idx, err := strconv.Atoi(c.Params("idx"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}

	// Get image count and title from query parameters (passed by nav buttons)
	imageCountStr := c.Query("count", "1")
	imageCount, err := strconv.Atoi(imageCountStr)
	if err != nil || imageCount < 0 {
		imageCount = 1 // Default to 1 if invalid
	}

	title := c.Query("title", "")

	return render(c, ui.GridImageWithNav(adID, imageCount, title, idx))
}

// HandleImageProxy proxies images from MinIO
func HandleImageProxy(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	idx, err := strconv.Atoi(c.Params("idx"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}

	size := c.Params("size", "1200w") // Default to 1200w if not specified

	// Get image from MinIO
	imageData, err := minio.GetImageObject(adID, idx, size)
	if err != nil {
		logger.Error("Failed to get image from MinIO",
			"error", err, "adID", adID, "idx", idx, "size", size)
		return fiber.NewError(fiber.StatusNotFound, "Image not found")
	}

	// Set appropriate headers for caching and content type
	c.Set("Content-Type", "image/webp")
	c.Set("Cache-Control", "public, max-age=2592000, immutable") // Cache for 30 days
	c.Set("ETag", fmt.Sprintf(`"%d-%d-%s"`, adID, idx, size))

	// Handle If-None-Match for 304 Not Modified
	if match := c.Get("If-None-Match"); match != "" && match == fmt.Sprintf(`"%d-%d-%s"`, adID, idx, size) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return c.Send(imageData)
}
