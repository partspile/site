package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

// HandleAdsPage handles the main /ads page
func HandleAdsPage(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	return render(c, ui.AdsPage(userID, userName, c.Path(), "bookmarked"))
}

// HandleBookmarkedAdsPage handles the /ads/bookmarked sub-page
func HandleBookmarkedAdsPage(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	adIDs, err := ad.GetBookmarkedAdIDs(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ads")
	}

	// Return navigation with content
	content := ui.BookmarkedAdsPage(ads, userID, userName, c.Path(), getLocation(c))
	return render(c, ui.AdsPageWithContent(userID, userName, c.Path(), "bookmarked", content))
}

// HandleActiveAdsPage handles the /ads/active sub-page
func HandleActiveAdsPage(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	adIDs, err := ad.GetUserActiveAdIDs(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get active ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get active ads")
	}

	// Return navigation with content
	content := ui.ActiveAdsPage(ads, userID, userName, c.Path(), getLocation(c))
	return render(c, ui.AdsPageWithContent(userID, userName, c.Path(), "active", content))
}

// HandleDeletedAdsPage handles the /ads/deleted sub-page
func HandleDeletedAdsPage(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	adIDs, err := ad.GetUserDeletedAdIDs(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get deleted ad IDs")
	}
	ads, err := ad.GetAdsByIDsWithDeleted(adIDs, userID, true)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get deleted ads")
	}

	// Return navigation with content
	content := ui.DeletedAdsPage(ads, userID, userName, c.Path(), getLocation(c))
	return render(c, ui.AdsPageWithContent(userID, userName, c.Path(), "deleted", content))
}
