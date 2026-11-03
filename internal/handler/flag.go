package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/flag"
	"github.com/partspile/site/internal/flagopinion"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
	g "maragu.dev/gomponents"
)

func ConfirmFlagHandler(c *fiber.Ctx) error {
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
			"Only conversation participants can raise flags")
	}
	if !message.MessagingAllowed(conv) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Messaging is closed for this conversation")
	}

	return renderFlagThrowConfirm(c, conv, currentUserID, csrfToken)
}

func ConfirmFlagOnAdHandler(c *fiber.Ctx) error {
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
	if err := ensureCanFlagAd(a, currentUserID); err != nil {
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

	return renderFlagThrowConfirm(c, conv, currentUserID, csrfToken)
}

func renderFlagThrowConfirm(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, csrfToken string) error {
	outstanding, _ := flag.GetUserFlagCount(currentUserID)
	remaining := config.MaxOutstandingFlags - outstanding
	if remaining < 0 {
		remaining = 0
	}

	atAd := currentUserID == conv.InquirerID
	inquirerName, _ := message.DisplayName(conv.InquirerID)
	ownerName, _ := message.DisplayName(conv.OwnerID)
	otherName := ownerName
	otherID := conv.OwnerID
	label := "Raise flag"
	if !atAd {
		otherName = inquirerName
		otherID = conv.InquirerID
		label = fmt.Sprintf("Raise flag on %s", inquirerName)
	}

	return render(c, ui.FlagThrowConfirmModal(ui.FlagThrowConfirmData{
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

func PreviewFlagHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}
	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	reason := c.FormValue("reason")
	if !flag.ValidReason(reason) {
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

	return renderFlagPreview(c, conv, currentUserID, reason, tz)
}

func PreviewFlagOnAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	reason := c.FormValue("reason")
	if !flag.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if err := ensureCanFlagAd(a, currentUserID); err != nil {
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

	return renderFlagPreview(c, conv, currentUserID, reason, tz)
}

func renderFlagPreview(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, reason string, tz *time.Location) error {
	op, err := flagopinion.Preview(conv, currentUserID, reason, tz)
	if errors.Is(err, flagopinion.ErrUnavailable) || err != nil {
		if errors.Is(err, flag.ErrInvalidReason) {
			return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
		}
		if err != nil && !errors.Is(err, flagopinion.ErrUnavailable) {
			logger.Error("Failed to preview flag opinion", "error", err,
				"conversationID", conv.ID, "userID", currentUserID)
		}
		return render(c, ui.FlagThrowPreviewModal(ui.FlagThrowPreviewData{
			Unavailable: true,
		}))
	}

	return render(c, ui.FlagThrowPreviewModal(ui.FlagThrowPreviewData{
		Summary:          op.Summary,
		AssessmentScore:  op.Assessment,
		AssessmentDetail: op.AssessmentDetail,
		Resolution:       op.Resolution,
		Reasoning:        op.Reasoning,
	}))
}

func RaiseFlagHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)
	reason := c.FormValue("reason")
	if !flag.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	conv, err := message.GetConversation(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}

	if conv.OwnerID != currentUserID && conv.InquirerID != currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"Only conversation participants can raise flags")
	}
	if !message.MessagingAllowed(conv) {
		return fiber.NewError(fiber.StatusBadRequest,
			"Messaging is closed for this conversation")
	}

	if err := flag.RaiseFlag(currentUserID, conversationID, reason); err != nil {
		return raiseFlagError(err, conversationID, currentUserID)
	}

	conv, err = message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}

	sendFlagEventSSE(conv, currentUserID)
	return renderConversationFlagEventAppend(c, conv, currentUserID, tz,
		csrfToken, true)
}

// RaiseFlagOnAdHandler creates the conversation if needed, then throws a flag.
// Used when the inquirer opens a new conversation modal (conversation ID 0).
func RaiseFlagOnAdHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)
	reason := c.FormValue("reason")
	if !flag.ValidReason(reason) {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if err := ensureCanFlagAd(a, currentUserID); err != nil {
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
			logger.Error("Failed to create conversation for flag throw",
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

	if err := flag.RaiseFlag(currentUserID, conv.ID, reason); err != nil {
		return raiseFlagError(err, conv.ID, currentUserID)
	}

	conv, err = message.GetConversationByID(conv.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}

	sendFlagEventSSE(conv, currentUserID)

	view, err := message.BuildConversationModal(conv, currentUserID, tz)
	if err != nil {
		return buildConversationModalError(err)
	}
	messageNodes := messageTimelineFromView(view, currentUserID, tz)
	data := conversationModalDataFromView(view, currentUserID, csrfToken,
		"conversation-0-modal", messageNodes)
	nodes := []g.Node{ui.ConversationModalSwapOOB(data)}
	nodes = append(nodes, ui.CloseFlagThrowModalsOOB()...)
	return render(c, g.Group(nodes))
}

func LowerFlagHandler(c *fiber.Ctx) error {
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
			"Only conversation participants can lower flags")
	}

	err = flag.LowerFlag(currentUserID, conversationID)
	if err != nil {
		if err == flag.ErrFlagNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Flag not found")
		}
		logger.Error("Failed to lower flag", "error", err,
			"conversationID", conversationID, "userID", currentUserID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to lower flag")
	}
	if err := flagopinion.Invalidate(conversationID); err != nil {
		logger.Error("Failed to invalidate flag opinion",
			"error", err, "conversationID", conversationID)
	}

	conv, err = message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to get conversation")
	}

	sendFlagEventSSE(conv, currentUserID)
	return renderConversationFlagEventAppend(c, conv, currentUserID, tz,
		csrfToken, false)
}

func ensureCanFlagAd(a ad.Ad, currentUserID int) error {
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest,
			"Cannot raise a flag on an inactive or deleted ad")
	}
	if !user.Exists(a.UserID) {
		return fiber.NewError(fiber.StatusBadRequest,
			"This account is no longer available")
	}
	if a.UserID == currentUserID {
		return fiber.NewError(fiber.StatusForbidden,
			"You cannot raise a flag on your own ad")
	}
	return nil
}

func raiseFlagError(err error, conversationID, userID int) error {
	if err == flag.ErrMaxFlagsReached {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf(
			"You have reached the maximum of %d outstanding flags",
			config.MaxOutstandingFlags,
		))
	}
	if err == flag.ErrFlagAlreadyRaised {
		return fiber.NewError(fiber.StatusBadRequest,
			"A flag has already been raised on this conversation")
	}
	if err == flag.ErrInvalidReason {
		return fiber.NewError(fiber.StatusBadRequest, "Choose a reason")
	}
	logger.Error("Failed to throw flag", "error", err,
		"conversationID", conversationID, "userID", userID)
	return fiber.NewError(fiber.StatusInternalServerError,
		"Failed to raise flag")
}
