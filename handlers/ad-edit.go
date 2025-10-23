package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// HandleUpdateAdPrice updates only the price of an ad
func HandleUpdateAdPrice(c *fiber.Ctx) error {
	u := getUser(c)
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, err := ad.GetAdByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if existingAd.UserID != u.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}

	// Validate and parse price
	price, err := ValidateAndParsePrice(c)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Update only the price
	_, err = db.Exec("UPDATE Ad SET price = ? WHERE id = ?", price, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update price")
	}

	// Fetch updated ad for display
	updatedAd, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to fetch updated ad")
	}

	// Queue for vector update since price affects search
	vector.QueueAd(updatedAd.Ad)

	return render(c, ui.AdDetail(*updatedAd, getLocation(c),
		u.ID, getView(c)))
}

// HandleUpdateAdDescription appends to the description of an ad
func HandleUpdateAdDescription(c *fiber.Ctx) error {
	u := getUser(c)
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Get full ad detail for description access
	existingAd, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if existingAd.UserID != u.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}

	// Handle description addition
	descriptionAddition := c.FormValue("description_addition")
	updatedDescription := existingAd.Description

	if descriptionAddition != "" {
		// Clean the addition text
		descriptionAddition = strings.TrimSpace(descriptionAddition)
		if descriptionAddition != "" {
			// Create timestamp
			timestamp := time.Now().Format("2006-01-02 15:04")
			// Append with timestamp
			addition := fmt.Sprintf("\n\n[%s] %s",
				timestamp, descriptionAddition)
			updatedDescription = existingAd.Description + addition

			// Validate total length
			if len(updatedDescription) > 500 {
				return ValidationErrorResponse(c, fmt.Sprintf(
					"Total description would be %d characters (max 500). "+
						"Please shorten your addition.",
					len(updatedDescription)))
			}
		}
	}

	// Update only the description
	_, err = db.Exec("UPDATE Ad SET description = ? WHERE id = ?",
		updatedDescription, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update description")
	}

	// Fetch updated ad for display
	updatedAd, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to fetch updated ad")
	}

	// Queue for vector update since description affects search
	vector.QueueAd(updatedAd.Ad)

	return render(c, ui.AdDetail(*updatedAd, getLocation(c),
		u.ID, getView(c)))
}

// HandleUpdateAdLocation updates only the location of an ad
func HandleUpdateAdLocation(c *fiber.Ctx) error {
	u := getUser(c)
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, err := ad.GetAdByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if existingAd.UserID != u.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}

	// Resolve and store location
	locationRaw := c.FormValue("location")
	locID, err := resolveAndStoreLocation(locationRaw)
	if err != nil {
		return ValidationErrorResponse(c, "Could not resolve location.")
	}

	// Update only the location
	_, err = db.Exec("UPDATE Ad SET location_id = ? WHERE id = ?", locID, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update location")
	}

	// Fetch updated ad for display
	updatedAd, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to fetch updated ad")
	}

	// Queue for vector update since location affects search
	vector.QueueAd(updatedAd.Ad)

	return render(c, ui.AdDetail(*updatedAd, getLocation(c),
		u.ID, getView(c)))
}

// HandlePriceModal shows the price edit modal for an ad
func HandlePriceModal(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	u := getUser(c)
	adObj, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Check ownership and archived status
	if adObj.UserID != u.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if adObj.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot edit archived ad")
	}

	return render(c, ui.PriceEditModal(*adObj, getView(c)))
}

// HandleDescriptionModal shows the description edit modal for an ad
func HandleDescriptionModal(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	u := getUser(c)
	adObj, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Check ownership and archived status
	if adObj.UserID != u.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if adObj.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot edit archived ad")
	}

	return render(c, ui.DescriptionEditModal(*adObj, getView(c)))
}

// HandleLocationModal shows the location edit modal for an ad
func HandleLocationModal(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	u := getUser(c)
	adObj, err := ad.GetAdDetailByID(adID, u)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Check ownership and archived status
	if adObj.UserID != u.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if adObj.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot edit archived ad")
	}

	return render(c, ui.LocationEditModal(*adObj, getView(c)))
}
