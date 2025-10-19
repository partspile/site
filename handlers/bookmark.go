package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// Handler to bookmark an ad
func HandleBookmarkAd(c *fiber.Ctx) error {
	currentUser, userID := CurrentUser(c)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.BookmarkAd(userID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to bookmark ad")
	}
	// Queue user for background embedding update
	vector.QueueUserForUpdate(userID)
	// Get the updated ad with bookmark status
	ads, err := ad.GetAdsByIDs([]int{adID}, currentUser)
	if err != nil || len(ads) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	adObj := ads[0]
	// Return the bookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(adObj))
}

// Handler to unbookmark an ad
func HandleUnbookmarkAd(c *fiber.Ctx) error {
	currentUser, userID := CurrentUser(c)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.UnbookmarkAd(userID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to unbookmark ad")
	}
	// Queue user for background embedding update
	vector.QueueUserForUpdate(userID)
	// Get the updated ad with bookmark status
	ads, err := ad.GetAdsByIDs([]int{adID}, currentUser)
	if err != nil || len(ads) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	adObj := ads[0]
	// Return the unbookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(adObj))
}
