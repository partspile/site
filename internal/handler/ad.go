package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/rockopinion"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
	"github.com/partspile/site/internal/vector"
)

func AdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if err := ad.LoadTags(&a); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// If ad is not live and user is not the owner, show unavailable message
	if !a.IsActive() && a.UserID != userID {
		return renderPage(c, "Ad Unavailable", ui.AdUnavailable())
	}

	// Update the ad category cookie based on the ad
	cookie.SetCategoryID(c, a.CategoryID)

	if a.IsActive() && userID != 0 {
		_ = ad.IncrementAdClickForUser(adID, userID)
	}

	csrfToken := local.GetCSRFToken(c)

	reachable, err := user.IsReachable(a.UserID)
	if err != nil {
		reachable = false
	}

	isTest, err := user.IsTestUser(a.UserID)
	if err != nil {
		isTest = false
	}

	return renderPage(c, a.Title, ui.Ad(adDetailFrom(a, userID, reachable, isTest), userID, csrfToken))
}

func adDetailFrom(a ad.Ad, viewerUserID int, reachable, isTest bool) ui.AdDetail {
	price, priceCurrency, hasPrice := a.PriceValue()
	priceDisplay := ""
	if hasPrice {
		priceDisplay = currency.Format(price, priceCurrency)
	}
	desc := ad.ParseDescriptionForDisplay(a.Description)
	var history []ui.AdHistoryEntry
	showLoginForDetails := false
	if local.IsLoggedIn(viewerUserID) {
		history = make([]ui.AdHistoryEntry, len(desc.History))
		for i, e := range desc.History {
			history[i] = ui.AdHistoryEntry{
				Header:       e.Header,
				Body:         e.Body,
				ImageIndices: e.ImageIndices,
			}
		}
	} else if len(desc.History) > 0 {
		showLoginForDetails = true
	}
	return ui.AdDetail{
		ID:                  a.ID,
		OwnerID:             a.UserID,
		ImageCount:          a.ImageCount,
		PriceDisplay:        priceDisplay,
		HasPrice:            hasPrice,
		Title:               a.Title,
		Location:            ad.AdLocationDisplay(a, viewerUserID),
		DescriptionOriginal: desc.Body,
		DescriptionHistory:  history,
		ShowLoginForDetails: showLoginForDetails,
		CreatedAt:           a.CreatedAt,
		ExpiresAt:           a.ExpiresAt,
		Bookmarked:          a.Bookmarked,
		Active:              a.IsActive(),
		Inactive:            a.IsInactive(),
		Deleted:             a.IsDeleted(),
		IsTest:              isTest,
		Reachable:           reachable,
		RockCount:           a.RockCount,
		FacetLabel:          adFacetLabel(a),
		FacetDetails:        adDetailFacetDisplays(a),
		Tags:                adTagDisplays(a),
	}
}

func adTagDisplays(a ad.Ad) []string {
	if len(a.Tags) == 0 {
		return nil
	}
	out := make([]string, len(a.Tags))
	for i, s := range a.Tags {
		out[i] = s.Display()
	}
	return out
}

func AdRockConversationHandler(c *fiber.Ctx) error {
	return renderAdRockOpinion(c)
}

func PublicAdRockOpinionHandler(c *fiber.Ctx) error {
	return renderAdRockOpinion(c)
}

func renderAdRockOpinion(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	ordinal, err := c.ParamsInt("ordinal")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid rock ordinal")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	conversationID, err := rock.GetPublicConversationIDByOrdinal(adID, ordinal)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Rock conversation not found")
	}

	conv, err := message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conv.RockThrowerID == nil {
		return fiber.NewError(fiber.StatusNotFound, "No dispute assessment")
	}

	return renderRockOpinionModal(c, conv, currentUserID, tz)
}

func AdShareHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if !a.IsActive() && a.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	path := config.CanonicalURL(fmt.Sprintf("/ad/%d", adID))
	flyerHref := ""
	if a.IsActive() && local.IsLoggedIn(userID) &&
		a.UserID == userID {
		flyerHref = fmt.Sprintf("/auth/ad/%d/flyer", adID)
	}
	return render(c, ui.AdShareModal(path, flyerHref))
}

func AdFlyerHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if !local.IsLoggedIn(userID) || a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden,
			"You are not the owner of this ad")
	}

	if err := ad.LoadTags(&a); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return render(c, ui.AdFlyerPage(adFlyerData(a, userID, adID)))
}

func adFlyerData(a ad.Ad, userID, adID int) ui.AdFlyerData {
	d := adDetailFrom(a, userID, false, false)
	return ui.AdFlyerData{
		ID:           d.ID,
		ImageCount:   d.ImageCount,
		PriceDisplay: d.PriceDisplay,
		HasPrice:     d.HasPrice,
		Title:        d.Title,
		Location:     ad.AdFlyerLocationDisplay(a, userID),
		Description:  d.DescriptionOriginal,
		FacetLabel:   d.FacetLabel,
		FacetDetails: d.FacetDetails,
		Tags:         d.Tags,
		AdURL:        config.CanonicalURL(fmt.Sprintf("/ad/%d", adID)),
	}
}

func AdShareCopyHandler(c *fiber.Ctx) error {
	path := c.Query("path", "")
	copied := c.Query("copied", "false") == "true"

	if copied {
		return render(c, ui.CopyButtonCopied(path))
	}
	return render(c, ui.CopyButton(path))
}

func AdRemoveModalHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Ad is not active")
	}

	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.AdRemoveModal(adID, csrfToken))
}

func PauseAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Ad is not active")
	}

	if err := pauseAdWithSideEffects(adID, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to pause ad")
	}

	c.Set("HX-Redirect", fmt.Sprintf("/ad/%d", adID))
	return c.SendStatus(fiber.StatusOK)
}

// pauseAdWithSideEffects pauses an ad and cleans up embeddings, rocks,
// opinions, and conversations. actorID is recorded in conversation journals.
func pauseAdWithSideEffects(adID, actorID int) error {
	if err := ad.Pause(adID); err != nil {
		return err
	}
	_ = vector.DeleteAdEmbedding(adID)
	_ = rock.UnthrowActiveForAd(adID)
	_ = rockopinion.InvalidateForAd(adID)

	convs, err := message.SuspendConversationsForPausedAd(adID, actorID)
	if err != nil {
		logger.Error("Failed to suspend conversations for paused ad",
			"error", err, "adID", adID)
		return nil
	}
	NotifyConversationsStatus(convs, actorID)
	return nil
}

func DeleteAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if a.IsDeleted() {
		return fiber.NewError(fiber.StatusBadRequest, "Ad is already deleted")
	}

	if err := ad.PermanentlyDelete(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete ad")
	}
	_ = vector.DeleteAdEmbedding(adID)
	_ = rock.UnthrowActiveForAd(adID)
	_ = rockopinion.InvalidateForAd(adID)

	convs, err := message.CloseConversationsForDeletedAd(adID, userID)
	if err != nil {
		logger.Error("Failed to close conversations for deleted ad",
			"error", err, "adID", adID)
	} else {
		NotifyConversationsClosed(convs, userID)
	}

	c.Set("HX-Redirect", fmt.Sprintf("/ad/%d", adID))
	return c.SendStatus(fiber.StatusOK)
}

func RestoreAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsInactive() {
		return fiber.NewError(fiber.StatusBadRequest, "Only paused ads can be restored")
	}

	if err := ad.Activate(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to restore ad")
	}
	vector.QueueAd(adID)

	convs, err := message.ResumeConversationsForUnpausedAd(adID, userID)
	if err != nil {
		logger.Error("Failed to resume conversations for unpaused ad",
			"error", err, "adID", adID)
	} else {
		NotifyConversationsStatus(convs, userID)
	}

	c.Set("HX-Redirect", fmt.Sprintf("/ad/%d", adID))
	return c.SendStatus(fiber.StatusOK)
}

func RenewAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Only active ads can be renewed")
	}
	if !ad.RenewEligible(a.ExpiresAt.UTC(), time.Now().UTC()) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Ad is not eligible to renew yet")
	}

	if err := ad.Renew(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to renew ad")
	}

	c.Set("HX-Redirect", fmt.Sprintf("/ad/%d", adID))
	return c.SendStatus(fiber.StatusOK)
}
