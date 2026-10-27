package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/param"
)

func validImageSize(size string) bool {
	for _, s := range imagestore.ImageSizes {
		if s == size {
			return true
		}
	}
	return false
}

// MediaAdImageHandler streams an ad image. Used in local HTTP so the
// browser does not HTTPS-upgrade MinIO img URLs (Chainguard sends HSTS).
func MediaAdImageHandler(c *fiber.Ctx) error {
	if adImageStore == nil {
		return fiber.ErrNotFound
	}
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.ErrNotFound
	}
	index, err := c.ParamsInt("index")
	if err != nil || index < 1 {
		return fiber.ErrNotFound
	}
	size := c.Params("size")
	if !validImageSize(size) {
		return fiber.ErrNotFound
	}
	data, err := adImageStore.Get(adID, index, size)
	if err != nil {
		return fiber.ErrNotFound
	}
	c.Set(fiber.HeaderContentType, imagestore.ImageMIME)
	c.Set(fiber.HeaderCacheControl, config.MinIOObjectCacheControl)
	return c.Send(data)
}
