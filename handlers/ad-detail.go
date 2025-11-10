package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// HandleAdDetail handles ad detail (expanded view)
func HandleAdDetail(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	a, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Only increment click counts for non-deleted ads
	if !a.IsArchived() {
		_ = ad.IncrementAdClick(adID)
		if userID != 0 {
			// Record the click for the user
			_ = ad.IncrementAdClickForUser(adID, userID)
			// Queue user for background embedding update
			vector.QueueUserForUpdate(userID)
		}
	}

	loc := cookie.GetLocation(c)
	csrfToken := local.GetCSRFToken(c)
	view := c.Query("view", "")

	return render(c, ui.AdDetail(*a, userID, loc, csrfToken, view))
}

// HandleAdCollapse handles ad collapse
func HandleAdCollapse(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	userID := local.GetUserID(c)
	a, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	loc := cookie.GetLocation(c)
	csrfToken := local.GetCSRFToken(c)

	// Get view from query parameter, fall back to cookie if not provided
	view := ui.StringToView(c.Query("view"))

	switch view {
	case ui.ViewList, ui.ViewTree:
		return render(c, ui.AdListNode(*a, userID, loc, csrfToken))
	case ui.ViewGrid:
		return render(c, ui.AdGridNode(*a, userID, loc, csrfToken))
	}
	return nil
}
