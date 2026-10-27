package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/rockopinion"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
	g "maragu.dev/gomponents"
)

func ConfirmRockHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}
	currentUserID := local.GetUserID(c)
	csrfToken := local.GetCSRFToken(c)

	conv, err := message.GetConversation(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conv.OwnerID != currentUserID && conv.InquirerID != currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"Only conversation participants can throw rocks")
	}
	if !message.MessagingAllowed(conv) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Messaging is closed for this conversation")
	}

	return renderRockThrowConfirm(c, conv, currentUserID, csrfToken)
}

func ConfirmRockOnAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	currentUserID := local.GetUserID(c)
	csrfToken := local.GetCSRFToken(c)
	tz := cookie.GetTimezone(c)

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if err := ensureCanRockAd(a, currentUserID); err != nil {
		return err
	}

	conv, err := message.GetConversationByAdAndInquirer(
		adID, a.UserID, currentUserID)
	if err != nil && err != message.ErrConversationNotFound {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}
	if err == message.ErrConversationNotFound {
		conv = message.Conversation{
			ID:         0,
			AdID:       adID,
			OwnerID:    a.UserID,
			InquirerID: currentUserID,
		}
	} else if !message.MessagingAllowed(conv) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Messaging is closed for this conversation")
	}

	return renderRockThrowConfirm(c, conv, currentUserID, csrfToken)
}

func renderRockThrowConfirm(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, csrfToken string) error {
	outstanding, _ := rock.GetUserRockCount(currentUserID)
	remaining := config.MaxOutstandingRocks - outstanding
	if remaining < 0 {
		remaining = 0
	}

	atAd := currentUserID == conv.InquirerID
	inquirerName, _ := message.DisplayName(conv.InquirerID)
	ownerName, _ := message.DisplayName(conv.OwnerID)
	otherName := ownerName
	otherID := conv.OwnerID
	label := "Throw rock"
	if !atAd {
		otherName = inquirerName
		otherID = conv.InquirerID
		label = fmt.Sprintf("Throw rock at %s", inquirerName)
	}

	return render(c, ui.RockThrowConfirmModal(ui.RockThrowConfirmData{
		ConversationID: conv.ID,
		AdID:           conv.AdID,
		CSRFToken:      csrfToken,
		Remaining:      remaining,
		ThrowLabel:     label,
		AtAd:           atAd,
		OtherUserID:    otherID,
		OtherName:      otherName,
	}))
}

func PreviewRockHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}
	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	reason := c.FormValue("reason")
	if !rock.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	conv, err := message.GetConversation(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conv.OwnerID != currentUserID && conv.InquirerID != currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"Only conversation participants can preview")
	}

	return renderRockPreview(c, conv, currentUserID, reason, tz)
}

func PreviewRockOnAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	reason := c.FormValue("reason")
	if !rock.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if err := ensureCanRockAd(a, currentUserID); err != nil {
		return err
	}

	conv, err := message.GetConversationByAdAndInquirer(
		adID, a.UserID, currentUserID)
	if err != nil && err != message.ErrConversationNotFound {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}
	if err == message.ErrConversationNotFound {
		conv = message.Conversation{
			ID:         0,
			AdID:       adID,
			OwnerID:    a.UserID,
			InquirerID: currentUserID,
		}
	}

	return renderRockPreview(c, conv, currentUserID, reason, tz)
}

func renderRockPreview(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, reason string, tz *time.Location) error {
	op, err := rockopinion.Preview(conv, currentUserID, reason, tz)
	if errors.Is(err, rockopinion.ErrUnavailable) || err != nil {
		if errors.Is(err, rock.ErrInvalidReason) {
			return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
		}
		if err != nil && !errors.Is(err, rockopinion.ErrUnavailable) {
			logger.Error("Failed to preview rock opinion", "error", err,
				"conversationID", conv.ID, "userID", currentUserID)
		}
		return render(c, ui.RockThrowPreviewModal(ui.RockThrowPreviewData{
			Unavailable: true,
		}))
	}

	return render(c, ui.RockThrowPreviewModal(ui.RockThrowPreviewData{
		Summary:          op.Summary,
		AssessmentScore:  op.Assessment,
		AssessmentDetail: op.AssessmentDetail,
		Resolution:       op.Resolution,
		Reasoning:        op.Reasoning,
	}))
}

func ThrowRockHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)
	reason := c.FormValue("reason")
	if !rock.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	conv, err := message.GetConversation(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}

	if conv.OwnerID != currentUserID && conv.InquirerID != currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"Only conversation participants can throw rocks")
	}
	if !message.MessagingAllowed(conv) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Messaging is closed for this conversation")
	}

	if err := rock.ThrowRock(currentUserID, conversationID, reason); err != nil {
		return throwRockError(err, conversationID, currentUserID)
	}

	conv, err = message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}

	sendRockEventSSE(conv, currentUserID)
	return renderConversationRockEventAppend(c, conv, currentUserID, tz,
		csrfToken, true)
}

// ThrowRockOnAdHandler creates the conversation if needed, then throws a rock.
// Used when the inquirer opens a new conversation modal (conversation ID 0).
func ThrowRockOnAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)
	reason := c.FormValue("reason")
	if !rock.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if err := ensureCanRockAd(a, currentUserID); err != nil {
		return err
	}

	conv, err := message.GetConversationByAdAndInquirer(
		adID, a.UserID, currentUserID)
	if err != nil {
		if err != message.ErrConversationNotFound {
			return fiber.NewError(fiber.StatusInternalServerError,
				"Failed to get conversation")
		}
		conv, err = message.CreateConversation(adID, a.UserID, currentUserID)
		if err != nil {
			logger.Error("Failed to create conversation for rock throw",
				"error", err, "adID", adID, "ownerID", a.UserID,
				"inquirerID", currentUserID)
			return fiber.NewError(fiber.StatusInternalServerError,
				"Failed to create conversation")
		}
	}

	if !message.MessagingAllowed(conv) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Messaging is closed for this conversation")
	}

	if err := rock.ThrowRock(currentUserID, conv.ID, reason); err != nil {
		return throwRockError(err, conv.ID, currentUserID)
	}

	conv, err = message.GetConversationByID(conv.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}

	sendRockEventSSE(conv, currentUserID)

	view, err := message.BuildConversationModal(conv, currentUserID, tz)
	if err != nil {
		return buildConversationModalError(err)
	}
	messageNodes := messageTimelineFromView(view, currentUserID, tz)
	data := conversationModalDataFromView(view, currentUserID, csrfToken,
		"conversation-0-modal", messageNodes)
	nodes := []g.Node{ui.ConversationModalSwapOOB(data)}
	nodes = append(nodes, ui.CloseRockThrowModalsOOB()...)
	return render(c, g.Group(nodes))
}

func UnthrowRockHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)

	conv, err := message.GetConversation(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}

	if conv.OwnerID != currentUserID && conv.InquirerID != currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"Only conversation participants can remove rocks")
	}

	err = rock.UnthrowRock(currentUserID, conversationID)
	if err != nil {
		if err == rock.ErrRockNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Rock not found")
		}
		logger.Error("Failed to unthrow rock", "error", err,
			"conversationID", conversationID, "userID", currentUserID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to remove rock")
	}
	if err := rockopinion.Invalidate(conversationID); err != nil {
		logger.Error("Failed to invalidate rock opinion",
			"error", err, "conversationID", conversationID)
	}

	conv, err = message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}

	sendRockEventSSE(conv, currentUserID)
	return renderConversationRockEventAppend(c, conv, currentUserID, tz,
		csrfToken, false)
}

func ensureCanRockAd(a ad.Ad, currentUserID int) error {
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest,
			"Cannot throw a rock at an inactive or deleted ad")
	}
	if !user.Exists(a.UserID) {
		return fiber.NewError(fiber.StatusBadRequest,
			"This account is no longer available")
	}
	if a.UserID == currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"You cannot throw a rock at your own ad")
	}
	return nil
}

func throwRockError(err error, conversationID, userID int) error {
	if err == rock.ErrMaxRocksReached {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf(
			"You have reached the maximum of %d outstanding rocks",
			config.MaxOutstandingRocks,
		))
	}
	if err == rock.ErrRockAlreadyThrown {
		return fiber.NewError(fiber.StatusBadRequest,
			"An rock has already been thrown at this conversation")
	}
	if err == rock.ErrInvalidReason {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}
	logger.Error("Failed to throw rock", "error", err,
		"conversationID", conversationID, "userID", userID)
	return fiber.NewError(fiber.StatusInternalServerError,
		"Failed to throw rock")
}
