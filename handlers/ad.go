package handlers

import (
	"fmt"

	"mime/multipart"

	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
	"gopkg.in/kothar/go-backblaze.v0"
)

func HandleNewAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	makes := vehicle.GetMakes()
	return render(c, ui.NewAdPage(currentUser, c.Path(), makes))
}

func HandleNewAdSubmission(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	newAd, imageFiles, err := BuildAdFromForm(c, currentUser.ID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}
	ad.AddAd(newAd)
	uploadAdImagesToB2(newAd.ID, imageFiles)
	return render(c, ui.SuccessMessage("Ad created successfully", "/"))
}

func HandleViewAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, _ := GetCurrentUser(c)

	// Get ad from either active or archived tables
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	bookmarked := false
	if currentUser != nil {
		bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
	}

	return render(c, ui.ViewAdPage(currentUser, c.Path(), adObj, bookmarked))
}

func HandleEditAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	adObj, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, adObj.UserID)
	if err != nil {
		return err
	}

	// Prepare make options
	makes := vehicle.GetMakes()
	// Prepare year checkboxes
	years := vehicle.GetYears(adObj.Make)
	// Prepare model checkboxes
	modelAvailability := vehicle.GetModelsWithAvailability(adObj.Make, adObj.Years)
	// Prepare engine checkboxes
	engineAvailability := vehicle.GetEnginesWithAvailability(adObj.Make, adObj.Years, adObj.Models)

	return render(c, ui.EditAdPage(currentUser, c.Path(), adObj, makes, years, modelAvailability, engineAvailability))
}

func HandleUpdateAdSubmission(c *fiber.Ctx) error {
	println("HandleUpdateAdSubmission")
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, existingAd.UserID)
	if err != nil {
		return err
	}

	updatedAd, imageFiles, err := BuildAdFromForm(c, currentUser.ID, adID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	ad.UpdateAd(updatedAd)
	uploadAdImagesToB2(updatedAd.ID, imageFiles)

	if c.Get("HX-Request") != "" {
		// For htmx, return the updated detail partial
		bookmarked := false
		if currentUser != nil {
			bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
		}
		view := c.Query("view", "list")
		return render(c, ui.AdDetailPartial(updatedAd, bookmarked, currentUser.ID, view))
	}
	return render(c, ui.SuccessMessage("Ad updated successfully", fmt.Sprintf("/ad/%d", adID)))
}

// Handler to bookmark an ad
func HandleBookmarkAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.BookmarkAd(currentUser.ID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to bookmark ad")
	}
	// Return the bookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(true, adID))
}

// Handler to unbookmark an ad
func HandleUnbookmarkAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.UnbookmarkAd(currentUser.ID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to unbookmark ad")
	}
	// Return the unbookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(false, adID))
}

// Handler to get bookmarked ads for the current user (for settings page)
func HandleBookmarkedAds(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adIDs, err := ad.GetBookmarkedAdIDsByUser(currentUser.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ads")
	}
	return render(c, ui.BookmarkedAdsSection(currentUser, ads))
}

func HandleArchiveAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to archive ad")
	}
	return render(c, ui.SuccessMessage("Ad archived successfully", "/"))
}

// Handler for ad card partial (collapse)
func HandleAdCardPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	currentUser, _ := CurrentUser(c)
	bookmarked := false
	userID := 0
	if currentUser != nil {
		bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
		userID = currentUser.ID
	}
	loc := c.Context().Time().Location()
	view := c.Query("view", "list")
	return render(c, ui.AdCardExpandable(adObj, loc, bookmarked, userID, view))
}

// Handler for ad detail partial (expand)
func HandleAdDetailPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Increment global click count
	_ = ad.IncrementAdClick(adID)

	currentUser, _ := GetCurrentUser(c)
	if currentUser != nil {
		_ = ad.IncrementAdClickForUser(adID, currentUser.ID)
	}

	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	bookmarked := false
	userID := 0
	if currentUser != nil {
		bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
		userID = currentUser.ID
	}
	view := c.Query("view", "list")
	return render(c, ui.AdDetailPartial(adObj, bookmarked, userID, view))
}

// Add this handler for deleting an ad
func HandleDeleteAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adObj, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete ad")
	}
	if c.Get("HX-Request") != "" {
		return c.SendStatus(fiber.StatusNoContent) // 204, so htmx removes the card
	}
	return render(c, ui.SuccessMessage("Ad deleted successfully", "/"))
}

// Handler for ad edit partial (inline edit)
func HandleEditAdPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	adObj, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	makes := vehicle.GetMakes()
	years := vehicle.GetYears(adObj.Make)
	modelAvailability := vehicle.GetModelsWithAvailability(adObj.Make, adObj.Years)
	engineAvailability := vehicle.GetEnginesWithAvailability(adObj.Make, adObj.Years, adObj.Models)
	view := c.Query("view", "list")
	cancelTarget := fmt.Sprintf("/ad/detail/%d?view=%s", adObj.ID, view)
	htmxTarget := fmt.Sprintf("#ad-%d", adObj.ID)
	if view == "grid" {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", adObj.ID)
	}
	return render(c, ui.AdEditPartial(adObj, makes, years, modelAvailability, engineAvailability, cancelTarget, htmxTarget, view))
}

// uploadAdImagesToB2 is a stub for now
func uploadAdImagesToB2(adID int, files []*multipart.FileHeader) {
	accountID := os.Getenv("BACKBLAZE_KEY_ID")
	appKey := os.Getenv("BACKBLAZE_APP_KEY")
	if accountID == "" || appKey == "" {
		log.Println("B2 credentials not set in env vars")
		return
	}
	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
	})
	if err != nil {
		log.Println("B2 auth error:", err)
		return
	}
	bucket, err := b2.Bucket("parts-pile")
	if err != nil {
		log.Println("B2 bucket error:", err)
		return
	}
	for i, fileHeader := range files {
		if filepath.Ext(fileHeader.Filename) != ".webp" {
			continue
		}
		file, err := fileHeader.Open()
		if err != nil {
			log.Println("B2 open file error:", err)
			continue
		}
		defer file.Close()
		b2Path := filepath.Join(
			fmt.Sprintf("%d", adID),
			fmt.Sprintf("%d.webp", i+1),
		)
		_, err = bucket.UploadTypedFile(b2Path, "image/webp", nil, file)
		if err != nil {
			log.Println("B2 upload error for", b2Path, ":", err)
		}
	}
}
