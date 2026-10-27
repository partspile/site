package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/rockopinion"
	"github.com/partspile/site/internal/vector"
)

type presignImagesRequest struct {
	Count int `json:"count"`
}

type presignUpload struct {
	Index  int    `json:"index"`
	Size   string `json:"size"`
	PutURL string `json:"putUrl"`
}

type confirmImagesRequest struct {
	ImageCount int `json:"imageCount"`
}

// PresignAdImagesHandler returns short-lived PUT URLs for new ad images.
func PresignAdImagesHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	adID, err := param.GetAdID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ad ID",
		})
	}

	a, err := ad.GetAd(userID, adID, cookie.GetTimezone(c))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ad not found",
		})
	}
	if a.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You are not the owner of this ad",
		})
	}
	if !a.IsActive() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot edit a deleted or inactive ad",
		})
	}

	var req presignImagesRequest
	if err := c.BodyParser(&req); err != nil || req.Count < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "count must be a positive integer",
		})
	}
	if a.ImageCount+req.Count > config.MaxImagesPerAd {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "too many images",
		})
	}

	start := a.ImageCount + 1
	uploads := make([]presignUpload, 0, req.Count*len(imagestore.ImageSizes))
	for i := 0; i < req.Count; i++ {
		index := start + i
		for _, size := range imagestore.ImageSizes {
			putURL, err := adImageStore.PresignPut(adID, index, size,
				config.MinIOPresignedPutExpiry)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to presign upload",
				})
			}
			uploads = append(uploads, presignUpload{
				Index: index, Size: size, PutURL: putURL,
			})
		}
	}

	return c.JSON(fiber.Map{
		"adId":       adID,
		"startIndex": start,
		"uploads":    uploads,
	})
}

// ConfirmAdImagesHandler sets image_count after successful client uploads.
func ConfirmAdImagesHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	adID, err := param.GetAdID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ad ID",
		})
	}

	var req confirmImagesRequest
	if err := c.BodyParser(&req); err != nil || req.ImageCount < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "imageCount must be a non-negative integer",
		})
	}

	if err := ad.ConfirmImages(userID, adID, req.ImageCount,
		cookie.GetTimezone(c)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if err := rockopinion.InvalidateForAd(adID); err != nil {
		logger.Error("Failed to invalidate rock opinions after images",
			"error", err, "adID", adID)
	}
	vector.QueueAd(adID)

	return c.JSON(fiber.Map{
		"adId":       adID,
		"imageCount": req.ImageCount,
	})
}

func wantsJSON(c *fiber.Ctx) bool {
	return c.Get("X-Ad-Upload") == "1" ||
		c.Get("Accept") == "application/json"
}

func respondAdSaved(c *fiber.Ctx, adID, imageCount int) error {
	if wantsJSON(c) {
		return c.JSON(fiber.Map{
			"adId":       adID,
			"imageCount": imageCount,
		})
	}
	redirect := "/ad/" + strconv.Itoa(adID)
	if c.Get("HX-Request") != "" {
		c.Set("HX-Redirect", redirect)
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(redirect, fiber.StatusFound)
}
