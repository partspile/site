package handlers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
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
	"github.com/parts-pile/site/ui"
	"golang.org/x/image/draw"
	"gopkg.in/kothar/go-backblaze.v0"
)

// uploadAdImagesToB2 uploads user-uploaded images to B2 with multiple sizes
func uploadAdImagesToB2(adID int, files []*multipart.FileHeader) {
	log.Printf("[B2] Starting upload for ad %d with %d images", adID, len(files))

	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey

	log.Printf("[B2] B2 config check for ad %d: accountID=%s, keyID=%s, appKey=%s", adID,
		func() string {
			if accountID == "" {
				return "EMPTY"
			} else {
				return "SET"
			}
		}(),
		func() string {
			if keyID == "" {
				return "EMPTY"
			} else {
				return "SET"
			}
		}(),
		func() string {
			if appKey == "" {
				return "EMPTY"
			} else {
				return "SET"
			}
		}())

	if accountID == "" || appKey == "" || keyID == "" {
		log.Printf("[B2] ERROR: B2 credentials not set in env vars for ad %d", adID)
		return
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		log.Printf("[B2] ERROR: B2 auth error for ad %d: %v", adID, err)
		return
	}

	log.Printf("[B2] Using bucket name: %s for ad %d", config.B2BucketName, adID)
	bucket, err := b2.Bucket(config.B2BucketName)
	if err != nil {
		log.Printf("[B2] ERROR: B2 bucket error for ad %d: %v", adID, err)
		return
	}
	log.Printf("[B2] Successfully connected to bucket for ad %d", adID)

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
		log.Printf("[B2] Processing image %d/%d for ad %d", i+1, len(files), adID)

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("[B2] ERROR: Failed to open file %d for ad %d: %v", i+1, adID, err)
			continue
		}

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(file); err != nil {
			log.Printf("[B2] ERROR: Failed to read file %d for ad %d: %v", i+1, adID, err)
			file.Close()
			continue
		}
		file.Close()

		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Printf("[B2] ERROR: Failed to decode image %d for ad %d: %v", i+1, adID, err)
			continue
		}

		bounds := img.Bounds()
		log.Printf("[B2] Image %d for ad %d: %dx%d pixels", i+1, adID, bounds.Dx(), bounds.Dy())

		for _, sz := range sizes {
			w := sz.Width
			h := bounds.Dy() * w / bounds.Dx()
			dst := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

			var webpBuf bytes.Buffer
			opt := &webp.Options{Lossless: false, Quality: sz.Quality}
			if err := webp.Encode(&webpBuf, dst, opt); err != nil {
				log.Printf("[B2] ERROR: WebP encode error for image %d size %s ad %d: %v", i+1, sz.Suffix, adID, err)
				continue
			}

			b2Path := filepath.Join(
				fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
			)

			log.Printf("[B2] Uploading %s to %s for ad %d", sz.Suffix, b2Path, adID)
			_, err = bucket.UploadTypedFile(b2Path, "image/webp", nil, bytes.NewReader(webpBuf.Bytes()))
			if err != nil {
				log.Printf("[B2] ERROR: Upload failed for %s to %s ad %d: %v", sz.Suffix, b2Path, adID, err)
			} else {
				log.Printf("[B2] SUCCESS: Uploaded %s to %s for ad %d", sz.Suffix, b2Path, adID)
				successCount++
			}
		}
	}

	log.Printf("[B2] Upload complete for ad %d: %d/%d files uploaded successfully", adID, successCount, totalExpected)
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

// HandleAdImage serves ad images with proper caching headers
func HandleAdImage(c *fiber.Ctx) error {
	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Get signed URL and redirect to it
	token, err := b2util.GetB2DownloadTokenForPrefixCached(fmt.Sprintf("%d/", adID))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate signed URL")
	}

	// Set cache headers
	c.Set("Cache-Control", "public, max-age=31536000") // 1 year
	c.Set("Expires", time.Now().Add(365*24*time.Hour).Format(http.TimeFormat))

	// For now, just return the token - the actual image serving logic needs to be implemented
	return c.JSON(fiber.Map{"token": token})
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

	u := getUser(c)
	adObj, err := ad.GetAdByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	return render(c, ui.GridImageWithNav(*adObj, idx))
}
