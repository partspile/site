package handlers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"context"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	miniosdk "github.com/minio/minio-go/v7"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/minio"
	"github.com/parts-pile/site/ui"
	"golang.org/x/image/draw"
)

// uploadAdImagesToMinIO uploads user-uploaded images to MinIO with multiple sizes
func uploadAdImagesToMinIO(adID int, files []*multipart.FileHeader) {
	logger.Info("Starting upload",
		"component", "MinIO", "adID", adID, "imageCount", len(files))

	minioClient := minio.GetMinIOClient()
	if minioClient == nil {
		logger.Error("MinIO client not initialized",
			"component", "MinIO", "adID", adID)
		return
	}

	if config.MinIOBucketName == "" {
		logger.Error("MinIO bucket name not set in env vars",
			"component", "MinIO", "adID", adID)
		return
	}

	logger.Debug("Using bucket name",
		"component", "MinIO", "bucketName", config.MinIOBucketName,
		"adID", adID)

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

	ctx := context.Background()

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

			objectPath := filepath.Join(
				fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
			)

			logger.Debug("Uploading to MinIO",
				"component", "MinIO", "suffix", sz.Suffix,
				"objectPath", objectPath, "adID", adID)
			_, err = minioClient.PutObject(ctx, config.MinIOBucketName, objectPath,
				bytes.NewReader(webpBuf.Bytes()), int64(webpBuf.Len()),
				miniosdk.PutObjectOptions{ContentType: "image/webp"})
			if err != nil {
				logger.Error("Upload failed",
					"error", err, "component", "MinIO",
					"suffix", sz.Suffix, "objectPath", objectPath, "adID",
					adID)
			} else {
				logger.Debug("Uploaded successfully",
					"component", "MinIO", "suffix", sz.Suffix,
					"objectPath", objectPath, "adID", adID)
				successCount++
			}
		}
	}

	logger.Info("Upload complete",
		"component", "MinIO", "adID", adID, "successCount",
		successCount, "totalExpected", totalExpected)
}

// HandleAdImageSignedURL generates a presigned URL for an ad image
func HandleAdImageSignedURL(c *fiber.Ctx) error {
	adIDStr := c.Params("adID")
	if adIDStr == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "adID required"})
	}
	adID, err := strconv.Atoi(adIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid adID"})
	}
	// Default to first image, 1200w size
	url, err := minio.GetMinIOPresignedURLForPrefixCached(adID, 1, "1200w")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"url":     url,
		"expires": time.Now().Unix() + config.MinIOPresignedURLExpiry,
	})
}

// HandleAdImage serves ad detail carousel images with navigation
func HandleAdImage(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	idx, err := strconv.Atoi(c.Params("idx"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}

	adDetail, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	return render(c, ui.CarouselImageContainer(*adDetail, idx))
}

// HandleAdGridImage serves grid view images with navigation
func HandleAdGridImage(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	idx, err := strconv.Atoi(c.Params("idx"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}

	adObj, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	return render(c, ui.GridImageWithNav(*adObj, idx))
}
