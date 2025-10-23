package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
)

// HandleShareModal shows the share modal for an ad
func HandleShareModal(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	u := getUser(c)
	adObj, err := ad.GetAdByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	return render(c, ui.ShareModal(*adObj))
}
