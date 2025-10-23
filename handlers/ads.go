package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
)

// HandleAdsPage handles the main /ads page
func HandleAdsPage(c *fiber.Ctx) error {
	u := getUser(c)
	return render(c, ui.AdsPage(u, c.Path(), "bookmarked"))
}

// HandleBookmarkedAdsPage handles the /ads/bookmarked sub-page
func HandleBookmarkedAdsPage(c *fiber.Ctx) error {
	u := getUser(c)
	adIDs, err := ad.GetBookmarkedAdIDs(u.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs, u)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ads")
	}

	// Return navigation with content
	content := ui.BookmarkedAdsPage(ads, u, c.Path(), getLocation(c))
	return render(c, ui.AdsPageWithContent(u, c.Path(), "bookmarked", content))
}

// HandleActiveAdsPage handles the /ads/active sub-page
func HandleActiveAdsPage(c *fiber.Ctx) error {
	u := getUser(c)
	adIDs, err := ad.GetUserActiveAdIDs(u.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get active ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs, u)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get active ads")
	}

	// Return navigation with content
	content := ui.ActiveAdsPage(ads, u, c.Path(), getLocation(c))
	return render(c, ui.AdsPageWithContent(u, c.Path(), "active", content))
}

// HandleDeletedAdsPage handles the /ads/deleted sub-page
func HandleDeletedAdsPage(c *fiber.Ctx) error {
	u := getUser(c)
	adIDs, err := ad.GetUserDeletedAdIDs(u.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get deleted ad IDs")
	}
	ads, err := ad.GetAdsByIDsWithDeleted(adIDs, u, true)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get deleted ads")
	}

	// Return navigation with content
	content := ui.DeletedAdsPage(ads, u, c.Path(), getLocation(c))
	return render(c, ui.AdsPageWithContent(u, c.Path(), "deleted", content))
}
