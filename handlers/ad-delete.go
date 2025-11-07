package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// HandleDeleteAd archives an ad
func HandleDeleteAd(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return err
	}
	userID := local.GetUserID(c)
	a, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete ad")
	}

	// Delete from vector database (Qdrant) - this is done after DB commit
	if err := vector.DeleteAdEmbedding(adID); err != nil {
		// Log the error but don't fail the entire operation since DB is already committed
		logger.Warn("Failed to delete ad from vector database",
			"error", err, "adID", adID)
	}

	logger.Debug("Delete ad",
		"adID", adID, "hxRequest", c.Get("HX-Request"))
	if c.Get("HX-Request") != "" {
		logger.Debug("Returning 200 with empty body for HTMX request",
			"adID", adID)
		// Return 200 with empty body instead of 204 because HTMX only swaps on 200/300 responses
		// See: https://github.com/bigskysoftware/htmx/issues/1130
		return render(c, ui.EmptyResponse())
	}
	logger.Debug("Returning success page for non-HTMX request",
		"adID", adID)
	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	return render(c, ui.SuccessMessage("Ad deleted successfully", "/", redirectDelayMs))
}

// HandleRestoreAd restores a deleted ad
func HandleRestoreAd(c *fiber.Ctx) error {
	adID, err := AdID(c)
	if err != nil {
		return err
	}
	userID := local.GetUserID(c)
	a, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if !a.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Ad is not deleted")
	}
	if err := ad.RestoreAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to restore ad")
	}

	// Queue ad for background re-addition to vector database (Qdrant)
	vector.QueueAd(a.ID)

	// Return empty response to remove the ad from the deleted ads list
	logger.Debug("Restore ad, removing from DOM",
		"adID", adID)
	return render(c, ui.EmptyResponse())
}
