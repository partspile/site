package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
)

// HandleAdPage renders the full ad page view
func HandleAdPage(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	u := getUser(c)

	ad, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// If ad is deleted and user is not the owner, show deleted message
	if ad.IsArchived() && ad.UserID != u.ID {
		return render(c, ui.AdDeletedPage(u, c.Path()))
	}

	// Owner can see their deleted ads, or anyone can see active ads
	return render(c, ui.AdPage(*ad, u, u.ID, c.Path(), getLocation(c), getView(c)))
}
