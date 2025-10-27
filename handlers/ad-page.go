package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

// HandleAdPage renders the full ad page view
func HandleAdPage(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	userName := local.GetUserName(c)

	a, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// If ad is deleted and user is not the owner, show deleted message
	if a.IsArchived() && a.UserID != userID {
		return render(c, ui.AdDeletedPage(userID, userName, c.Path()))
	}

	// Owner can see their deleted ads, or anyone can see active ads
	return render(c, ui.AdPage(*a, userID, userName, c.Path(), getLocation(c)))
}
