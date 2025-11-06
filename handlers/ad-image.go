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

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/ui"
	"golang.org/x/image/draw"
	"gopkg.in/kothar/go-backblaze.v0"
)

// uploadAdImagesToB2 uploads user-uploaded images to B2 with multiple sizes
func uploadAdImagesToB2(adID int, files []*multipart.FileHeader) {
	logger.Info("Starting upload",
		"component", "B2", "adID", adID, "imageCount", len(files))

	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey

	logger.Debug("B2 config check",
		"component", "B2", "adID", adID,
		"accountIDSet", accountID != "", "keyIDSet",
		keyID != "", "appKeySet", appKey != "")

	if accountID == "" || appKey == "" || keyID == "" {
		logger.Error("B2 credentials not set in env vars",
			"component", "B2", "adID", adID)
		return
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		logger.Error("B2 auth error",
			"error", err, "component", "B2", "adID", adID)
		return
	}

	logger.Debug("Using bucket name",
		"component", "B2", "bucketName", config.B2BucketName,
		"adID", adID)
	bucket, err := b2.Bucket(config.B2BucketName)
	if err != nil {
		logger.Error("B2 bucket error",
			"error", err, "component", "B2", "adID", adID)
		return
	}
	logger.Debug("Successfully connected to bucket",
		"component", "B2", "adID", adID)

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
			"component", "B2", "imageIndex", i+1,
			"totalImages", len(files), "adID", adID)

		file, err := fileHeader.Open()
		if err != nil {
			logger.Error("Failed to open file",
				"error", err, "component", "B2", "imageIndex",
				i+1, "adID", adID)
			continue
		}

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(file); err != nil {
			logger.Error("Failed to read file",
				"error", err, "component", "B2", "imageIndex",
				i+1, "adID", adID)
			file.Close()
			continue
		}
		file.Close()

		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			logger.Error("Failed to decode image",
				"error", err, "component", "B2", "imageIndex",
				i+1, "adID", adID)
			continue
		}

		bounds := img.Bounds()
		logger.Debug("Image dimensions",
			"component", "B2", "imageIndex", i+1, "adID", adID,
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
					"error", err, "component", "B2",
					"imageIndex", i+1, "size", sz.Suffix, "adID",
					adID)
				continue
			}

			b2Path := filepath.Join(
				fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
			)

			logger.Debug("Uploading to B2",
				"component", "B2", "suffix", sz.Suffix,
				"b2Path", b2Path, "adID", adID)
			_, err = bucket.UploadTypedFile(b2Path, "image/webp", nil, bytes.NewReader(webpBuf.Bytes()))
			if err != nil {
				logger.Error("Upload failed",
					"error", err, "component", "B2",
					"suffix", sz.Suffix, "b2Path", b2Path, "adID",
					adID)
			} else {
				logger.Debug("Uploaded successfully",
					"component", "B2", "suffix", sz.Suffix,
					"b2Path", b2Path, "adID", adID)
				successCount++
			}
		}
	}

	logger.Info("Upload complete",
		"component", "B2", "adID", adID, "successCount",
		successCount, "totalExpected", totalExpected)
}

// HandleAdImageSignedURL generates a signed URL for an ad image
func HandleAdImageSignedURL(c *fiber.Ctx) error {
	adID := c.Params("adID")
	if adID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "adID required"})
	}
	token, err := b2util.GetB2DownloadTokenForPrefixCached(adID + "/")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"prefix":  "/" + adID + "/",
		"token":   token,
		"expires": time.Now().Unix() + config.B2DownloadTokenExpiry,
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
