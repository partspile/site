package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// HandleDeleteAd archives an ad
func HandleDeleteAd(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return err
	}
	currentUser, _ := CurrentUser(c)
	adObj, err := ad.GetAdByID(adID, currentUser)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete ad")
	}

	// Delete from vector database (Qdrant) - this is done after DB commit
	if err := vector.DeleteAdEmbedding(adID); err != nil {
		// Log the error but don't fail the entire operation since DB is already committed
		log.Printf("Warning: Failed to delete ad %d from vector database: %v", adID, err)
	}

	log.Printf("Delete ad %d: HX-Request header = '%s'", adID, c.Get("HX-Request"))
	if c.Get("HX-Request") != "" {
		log.Printf("Returning 200 with empty body for HTMX request to delete ad %d", adID)
		// Return 200 with empty body instead of 204 because HTMX only swaps on 200/300 responses
		// See: https://github.com/bigskysoftware/htmx/issues/1130
		return render(c, ui.EmptyResponse())
	}
	log.Printf("Returning success page for non-HTMX request to delete ad %d", adID)
	return render(c, ui.SuccessMessage("Ad deleted successfully", "/"))
}

// HandleRestoreAd restores a deleted ad
func HandleRestoreAd(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return err
	}
	currentUser, userID := CurrentUser(c)
	adObj, err := ad.GetAdDetailByID(adID, currentUser)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if !adObj.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Ad is not deleted")
	}
	if err := ad.RestoreAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to restore ad")
	}

	// Queue ad for background re-addition to vector database (Qdrant)
	vector.QueueAd(adObj.Ad)

	// Return empty response to remove the ad from the deleted ads list
	log.Printf("Restore ad %d, removing from DOM", adID)
	return render(c, ui.EmptyResponse())
}
