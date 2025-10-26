package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// HandleAdDetail handles ad detail (expanded view)
func HandleAdDetail(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := getUserID(c)
	adObj, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Only increment click counts for non-deleted ads
	if !adObj.IsArchived() {
		_ = ad.IncrementAdClick(adID)
		if userID != 0 {
			_ = ad.IncrementAdClickForUser(adID, userID)
			// Queue user for background embedding update
			vector.QueueUserForUpdate(userID)
		}
	}

	loc := getLocation(c)
	view := cookie.GetView(c)

	return render(c, ui.AdDetail(adObj, userID, view, loc))
}

// HandleAdCollapse handles ad collapse
func HandleAdCollapse(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	userID := getUserID(c)
	adObj, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	loc := getLocation(c)
	view := getView(c)
	switch view {
	case "list", "tree":
		return render(c, ui.AdListNode(*adObj, userID, loc))
	case "grid":
		return render(c, ui.AdGridNode(*adObj, userID, loc))
	}
	return nil
}
