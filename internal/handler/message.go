package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/rockopinion"
	"github.com/partspile/site/internal/service/sms"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
	g "maragu.dev/gomponents"
)

type conversationModalRenderMode int

const (
	modalRenderInitial conversationModalRenderMode = iota
	modalRenderSwapOOB
)

func conversationModalData(conv message.Conversation, currentUserID,
	inquirerRockCount, ownerRockCount int, adTitle, ownerName, inquirerName,
	csrfToken string, canPost, hasThrownRock, canThrowRock bool,
	ownerDeleted, inquirerDeleted bool,
	messageNodes []g.Node, targetModalID string) ui.ConversationModalData {
	return ui.ConversationModalData{
		ConversationID:    conv.ID,
		AdID:              conv.AdID,
		OwnerID:           conv.OwnerID,
		InquirerID:        conv.InquirerID,
		CurrentUserID:     currentUserID,
		InquirerRockCount: inquirerRockCount,
		OwnerRockCount:    ownerRockCount,
		AdTitle:           adTitle,
		OwnerName:         ownerName,
		InquirerName:      inquirerName,
		OwnerDeleted:      ownerDeleted,
		InquirerDeleted:   inquirerDeleted,
		CSRFToken:         csrfToken,
		CanPost:           canPost,
		HasThrownRock:     hasThrownRock,
		CanThrowRock:      canThrowRock,
		MessageNodes:      messageNodes,
		RockThrowerID:     conv.RockThrowerID,
		TargetModalID:     targetModalID,
	}
}

func messageItemData(msg message.Message, currentUserID int,
	tz *time.Location) ui.MessageItemData {
	return ui.MessageItemData{
		SenderID:      msg.SenderID,
		CurrentUserID: currentUserID,
		Content:       msg.Content,
		CreatedAt:     msg.CreatedAt.In(tz),
	}
}

func rockEventData(event message.RockJournalEvent, conversationID,
	currentUserID, ownerID, inquirerID int,
	ownerName, inquirerName string) ui.RockEventData {
	kind := ui.RockEventThrown
	if event.Kind == "unthrown" {
		kind = ui.RockEventUnthrown
	}
	return ui.RockEventData{
		ConversationID: conversationID,
		ThrowerID:      event.UserID,
		CurrentUserID:  currentUserID,
		Kind:           kind,
		EventAt:        event.CreatedAt,
		OwnerID:        ownerID,
		InquirerID:     inquirerID,
		OwnerName:      ownerName,
		InquirerName:   inquirerName,
		Reason:         event.Reason,
	}
}

func rockEventsFromView(view message.ConversationModalView,
	currentUserID int) []ui.RockEventData {
	conv := view.Conversation
	events := make([]ui.RockEventData, len(view.RockEvents))
	for i, event := range view.RockEvents {
		events[i] = rockEventData(event, conv.ID, currentUserID, conv.OwnerID,
			conv.InquirerID, view.OwnerName, view.InquirerName)
	}
	if conv.RockThrowerID != nil && len(events) > 0 {
		last := &events[len(events)-1]
		if last.Kind == ui.RockEventThrown && last.ConversationID > 0 {
			last.ShowAssessmentHint = true
		}
	}
	return events
}

func closeEventData(event message.CloseJournalEvent) ui.CloseEventData {
	name, _ := message.DisplayName(event.UserID)
	return ui.CloseEventData{
		Text:    message.CloseEventText(event.Kind, name),
		EventAt: event.CreatedAt,
	}
}

func messageTimelineFromView(view message.ConversationModalView,
	currentUserID int, tz *time.Location) []g.Node {
	msgs := make([]ui.MessageItemData, len(view.Messages))
	for i, msg := range view.Messages {
		msgs[i] = messageItemData(msg, currentUserID, tz)
	}
	closes := make([]ui.CloseEventData, len(view.CloseEvents))
	for i, event := range view.CloseEvents {
		closes[i] = closeEventData(event)
	}
	return ui.MessageTimeline(msgs, rockEventsFromView(view, currentUserID),
		closes)
}

func conversationListItemData(conversationID, adID, ownerID, inquirerID,
	currentUserID int, adTitle, lastMessageContent, otherUserName string,
	lastMessageAt *time.Time, updatedAt time.Time, hasUnread bool, rockCount,
	otherUserRockCount int, tz *time.Location) ui.ConversationListItemData {
	d := ui.ConversationListItemData{
		ConversationID:     conversationID,
		AdID:               adID,
		OwnerID:            ownerID,
		InquirerID:         inquirerID,
		CurrentUserID:      currentUserID,
		AdTitle:            adTitle,
		LastMessageContent: lastMessageContent,
		OtherUserName:      otherUserName,
		UpdatedAt:          updatedAt.In(tz),
		HasUnread:          hasUnread,
		RockCount:          rockCount,
		OtherUserRockCount: otherUserRockCount,
	}
	if lastMessageAt != nil {
		t := lastMessageAt.In(tz)
		d.LastMessageAt = &t
	}
	return d
}

func conversationListItemFromConv(conv message.ConversationWithLastMessage,
	currentUserID int) ui.ConversationListItemData {
	return ui.ConversationListItemData{
		ConversationID:     conv.ID,
		AdID:               conv.AdID,
		OwnerID:            conv.OwnerID,
		InquirerID:         conv.InquirerID,
		CurrentUserID:      currentUserID,
		AdTitle:            conv.AdTitle,
		LastMessageContent: conv.LastMessageContent,
		OtherUserName:      conv.OtherUserName,
		LastMessageAt:      conv.LastMessageAt,
		UpdatedAt:          conv.UpdatedAt,
		HasUnread:          conv.HasUnread,
		RockCount:          conv.RockCount,
		OtherUserRockCount: conv.OtherUserRockCount,
		OtherUserDeleted:   conv.OtherUserDeleted,
	}
}

func conversationModalDataFromView(view message.ConversationModalView,
	currentUserID int, csrfToken, targetModalID string, messageNodes []g.Node) ui.ConversationModalData {
	return conversationModalData(
		view.Conversation, currentUserID,
		view.InquirerRockCount, view.OwnerRockCount,
		view.AdTitle, view.OwnerName, view.InquirerName, csrfToken,
		view.CanPost, view.HasThrownRock, view.CanThrowRock,
		view.OwnerDeleted, view.InquirerDeleted,
		messageNodes, targetModalID,
	)
}

func buildConversationModalError(err error) error {
	if errors.Is(err, message.ErrModalAdNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	logger.Error("Failed to build conversation modal", "error", err)
	return fiber.NewError(fiber.StatusInternalServerError, "Failed to load conversation")
}

func renderConversationModalView(c *fiber.Ctx,
	view message.ConversationModalView, currentUserID int, tz *time.Location,
	csrfToken, targetModalID string, mode conversationModalRenderMode) error {
	messageNodes := messageTimelineFromView(view, currentUserID, tz)
	data := conversationModalDataFromView(view, currentUserID, csrfToken, targetModalID, messageNodes)
	switch mode {
	case modalRenderInitial:
		return render(c, ui.ConversationModalWithRock(data))
	case modalRenderSwapOOB:
		return render(c, ui.ConversationModalSwapOOB(data))
	default:
		return render(c, ui.ConversationModalSwapOOB(data))
	}
}

func renderConversationModal(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, tz *time.Location, csrfToken string) error {
	view, err := message.BuildConversationModal(conv, currentUserID, tz)
	if err != nil {
		return buildConversationModalError(err)
	}
	return renderConversationModalView(c, view, currentUserID, tz, csrfToken, "", modalRenderInitial)
}

func buildRockEventUpdateNodes(conv message.Conversation, viewerID int,
	tz *time.Location, csrfToken string, rockEventOOB bool) ([]g.Node, error) {
	view, err := message.BuildConversationModal(conv, viewerID, tz)
	if err != nil {
		return nil, err
	}

	events := message.RockEventsFromJournal(conv.Journal, tz)
	if len(events) == 0 {
		return nil, fmt.Errorf("no rock events in journal")
	}
	lastEvent := events[len(events)-1]
	clearEmpty := len(view.Messages) == 0 && len(events) == 1

	var rockAttrs []g.Node
	if rockEventOOB {
		rockAttrs = append(rockAttrs,
			ui.ConversationMessageAppendOOB(conv.ID))
	}

	d := rockEventData(lastEvent, conv.ID, viewerID,
		conv.OwnerID, conv.InquirerID, view.OwnerName, view.InquirerName)
	if conv.RockThrowerID != nil && d.Kind == ui.RockEventThrown {
		d.ShowAssessmentHint = true
	}

	nodes := []g.Node{
		ui.RockEventMessage(d, rockAttrs...),
		ui.ConversationMessageActionsSwapOOB(
			conversationModalDataFromView(view, viewerID, csrfToken, "", nil)),
	}
	if clearEmpty {
		nodes = append(nodes,
			ui.ConversationEmptyMessageDeleteSwapOOB(conv.ID))
	}
	return nodes, nil
}

func renderConversationRockEventAppend(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, tz *time.Location, csrfToken string,
	closeThrowConfirm bool) error {
	nodes, err := buildRockEventUpdateNodes(conv, currentUserID, tz,
		csrfToken, false)
	if err != nil {
		logger.Error("Failed to build rock event response", "error", err,
			"conversationID", conv.ID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update conversation")
	}
	if closeThrowConfirm {
		nodes = append(nodes, ui.CloseRockThrowModalsOOB()...)
	}
	return render(c, g.Group(nodes))
}

func sendRockEventSSE(conv message.Conversation, throwerID int) {
	recipientID := conv.OwnerID
	if conv.OwnerID == throwerID {
		recipientID = conv.InquirerID
	}

	tz := time.UTC
	nodes, err := buildRockEventUpdateNodes(conv, recipientID, tz, "", true)
	if err != nil {
		logger.Error("Failed to build rock event for SSE", "error", err,
			"conversationID", conv.ID, "recipientID", recipientID)
		return
	}

	modalHTML, err := renderToString(g.Group(nodes))
	if err != nil {
		logger.Error("Failed to render rock event for SSE", "error", err,
			"conversationID", conv.ID, "recipientID", recipientID)
		return
	}

	SendSSEEvent(recipientID, SSEEvent{
		Event: ui.SSEEventConversation(conv.ID),
		Data:  modalHTML,
	})
}

func NotifyConversationsClosed(convs []message.Conversation, actorID int) {
	for _, conv := range convs {
		sendConversationStatusSSE(conv, actorID)
	}
}

func NotifyConversationsStatus(convs []message.Conversation, actorID int) {
	for _, conv := range convs {
		sendConversationStatusSSE(conv, actorID)
	}
}

func sendConversationStatusSSE(conv message.Conversation, actorID int) {
	recipientID := message.OtherParticipantID(conv, actorID)
	if recipientID == 0 || recipientID == actorID {
		return
	}

	tz := time.UTC
	events := message.CloseEventsFromJournal(conv.Journal, tz)
	if len(events) == 0 {
		return
	}
	last := events[len(events)-1]

	view, err := message.BuildConversationModal(conv, recipientID, tz)
	if err != nil {
		logger.Error("Failed to build modal for status SSE", "error", err,
			"conversationID", conv.ID, "recipientID", recipientID)
		return
	}

	clearEmpty := len(view.Messages) == 0 &&
		len(view.RockEvents) == 0 && len(view.CloseEvents) == 1

	nodes := []g.Node{
		ui.CloseEventMessage(closeEventData(last),
			ui.ConversationMessageAppendOOB(conv.ID)),
		ui.ConversationFormSwapOOB(conv.ID, conv.AdID, "", view.CanPost),
		ui.ConversationMessageActionsSwapOOB(
			conversationModalDataFromView(view, recipientID, "", "", nil)),
	}
	if clearEmpty {
		nodes = append(nodes,
			ui.ConversationEmptyMessageDeleteSwapOOB(conv.ID))
	}

	modalHTML, err := renderToString(g.Group(nodes))
	if err != nil {
		logger.Error("Failed to render status SSE", "error", err,
			"conversationID", conv.ID, "recipientID", recipientID)
		return
	}

	SendSSEEvent(recipientID, SSEEvent{
		Event: ui.SSEEventConversation(conv.ID),
		Data:  modalHTML,
	})

	sendClosedConversationListItemUpdate(conv, recipientID)
}

func sendClosedConversationListItemUpdate(conv message.Conversation,
	viewerID int) {
	tz := time.UTC
	a, err := ad.GetAd(viewerID, conv.AdID, tz)
	if err != nil {
		logger.Error("Failed to get ad for close list SSE", "error", err,
			"conversationID", conv.ID)
		return
	}
	otherUserName, err := message.OtherUserName(conv, viewerID)
	if err != nil {
		logger.Error("Failed to get other user name for close list SSE",
			"error", err, "conversationID", conv.ID)
		return
	}
	otherUserID := message.OtherParticipantID(conv, viewerID)
	otherUserRockCount, _ := rock.GetRockCountForUser(otherUserID)
	_, otherDeleted := message.DisplayName(otherUserID)

	content, at, ok := message.LastMessagePreview(conv.Journal)
	var lastMessageAt *time.Time
	if ok {
		t := at.In(tz)
		lastMessageAt = &t
	}

	hasUnread := false
	if conv.OwnerID == viewerID {
		hasUnread = conv.OwnerHasUnread
	} else if conv.InquirerID == viewerID {
		hasUnread = conv.InquirerHasUnread
	}

	item := ui.ConversationListItem(ui.ConversationListItemData{
		ConversationID:     conv.ID,
		AdID:               conv.AdID,
		OwnerID:            conv.OwnerID,
		InquirerID:         conv.InquirerID,
		CurrentUserID:      viewerID,
		AdTitle:            a.Title,
		LastMessageContent: content,
		OtherUserName:      otherUserName,
		LastMessageAt:      lastMessageAt,
		UpdatedAt:          conv.UpdatedAt.In(tz),
		HasUnread:          hasUnread,
		RockCount:          a.RockCount,
		OtherUserRockCount: otherUserRockCount,
		OtherUserDeleted:   otherDeleted,
	})
	itemHTML, err := renderToString(ui.ConversationListItemSwapOOB(conv.ID, item))
	if err != nil {
		logger.Error("Failed to render close list SSE", "error", err,
			"conversationID", conv.ID)
		return
	}
	SendSSEEvent(viewerID, SSEEvent{
		Event: ui.SSEEventConversationList,
		Data:  itemHTML,
	})
}

func renderConversationMessageAppend(c *fiber.Ctx, conversationID int,
	appendNode g.Node, clearEmpty bool) error {
	nodes := []g.Node{
		appendNode,
		ui.ConversationContentInputClearSwapOOB(conversationID),
	}
	if clearEmpty {
		nodes = append(nodes, ui.ConversationEmptyMessageDeleteSwapOOB(conversationID))
	}
	return render(c, g.Group(nodes))
}

func sendMessageSSE(conv message.Conversation, senderID int,
	msg message.Message) {
	// Determine recipient user ID
	recipientID := conv.OwnerID
	if conv.OwnerID == senderID {
		recipientID = conv.InquirerID
	}

	// Send message update
	sendMessageUpdate(conv.ID, msg, recipientID)

	// Send indicator update - recipient always has unread messages after receiving a new message
	sendUnreadIndicatorUpdate(recipientID, true)

	// Send conversation list item update - show green dot for unread conversation
	sendConversationListItemUpdate(conv, recipientID, true)
}

func sendMessageUpdate(conversationID int, msg message.Message, recipientID int) {
	conv, err := message.GetConversationByID(conversationID)
	if err != nil {
		logger.Error("Failed to get conversation for SSE modal update", "error", err, "conversationID", conversationID, "recipientID", recipientID)
		return
	}

	tz := time.UTC
	msgData := messageItemData(msg, recipientID, tz)
	var nodes []g.Node
	if len(message.MessagesFromJournal(conversationID, conv.Journal, tz)) == 1 {
		nodes = append(nodes, ui.ConversationEmptyMessageDeleteSwapOOB(conversationID))
	}
	nodes = append(nodes, ui.MessageItem(msgData,
		ui.ConversationMessageAppendOOB(conversationID)))

	modalHTML, err := renderToString(g.Group(nodes))
	if err != nil {
		logger.Error("Failed to render messages for SSE", "error", err, "conversationID", conversationID, "recipientID", recipientID)
		return
	}

	SendSSEEvent(recipientID, SSEEvent{
		Event: ui.SSEEventConversation(conversationID),
		Data:  modalHTML,
	})
}

func sendUnreadIndicatorUpdate(userID int, hasUnread bool) {
	indicatorSwapOOB := ui.UnreadIndicatorSwapOOB(hasUnread)
	indicatorHTML, err := renderToString(indicatorSwapOOB)
	if err != nil {
		logger.Error("Failed to render unread indicator for SSE", "error", err, "userID", userID)
		return
	}

	SendSSEEvent(userID, SSEEvent{
		Event: ui.SSEEventUnread,
		Data:  indicatorHTML,
	})
}

func sendMessageAndRenderUpdate(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, tz *time.Location, isNewConversation bool) error {

	content := c.FormValue("content")
	if content == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Message content cannot be empty")
	}

	// Step 1: Update database - if it fails, stop
	msg, err := message.CreateMessage(conv.ID, currentUserID, content)
	if err != nil {
		if errors.Is(err, message.ErrMessagingClosed) {
			return fiber.NewError(fiber.StatusBadRequest, "Messaging is closed for this conversation")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to send message")
	}
	if err := rockopinion.Invalidate(conv.ID); err != nil {
		logger.Error("Failed to invalidate rock opinion",
			"error", err, "conversationID", conv.ID)
	}

	// Step 2: Send SSE updates
	sendMessageSSE(conv, currentUserID, msg)

	// Step 3: Enqueue SMS notification (non-blocking)
	// Determine recipient user ID
	recipientID := conv.OwnerID
	if conv.OwnerID == currentUserID {
		recipientID = conv.InquirerID
	}
	if err := sms.EnqueueNotification(recipientID, conv.ID); err != nil {
		logger.Error("Failed to enqueue SMS notification",
			"error", err,
			"component", "message",
			"conversationID", conv.ID,
			"recipientID", recipientID,
			"senderID", currentUserID)
		// Don't fail the request, just log
	}

	// Get updated conversation to render full modal
	updatedConv, err := message.GetConversationByID(conv.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get updated conversation")
	}

	csrfToken := local.GetCSRFToken(c)
	view, err := message.BuildConversationModal(updatedConv, currentUserID, tz)
	if err != nil {
		return buildConversationModalError(err)
	}

	targetModalID := ""
	if isNewConversation {
		targetModalID = "conversation-0-modal"
		return renderConversationModalView(c, view, currentUserID, tz, csrfToken, targetModalID, modalRenderSwapOOB)
	}

	msgData := messageItemData(msg, currentUserID, tz)
	return renderConversationMessageAppend(c, conv.ID, ui.MessageItem(msgData),
		len(view.Messages) == 1)
}

func sendConversationListItemUpdate(conv message.Conversation, currentUserID int,
	hasUnread bool) {
	// Get ad title
	a, err := ad.GetAd(currentUserID, conv.AdID, time.UTC)
	if err != nil {
		logger.Error("Failed to get ad for conversation list item update", "error", err, "conversationID", conv.ID, "adID", conv.AdID)
		return
	}

	// Get other user name
	otherUserName, err := message.OtherUserName(conv, currentUserID)
	if err != nil {
		logger.Error("Failed to get other user name for conversation list item update", "error", err, "conversationID", conv.ID)
		return
	}

	// Get other user ID
	var otherUserID int
	if conv.OwnerID == currentUserID {
		otherUserID = conv.InquirerID
	} else {
		otherUserID = conv.OwnerID
	}

	// Get rock count for the other user
	otherUserRockCount, _ := rock.GetRockCountForUser(otherUserID)

	// Get last message for preview
	messages, err := message.GetConversationMessages(conv.ID, currentUserID, time.UTC)
	var lastMessageContent string
	var lastMessageAt *time.Time
	if err == nil && len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		lastMessageContent = lastMsg.Content
		lastMessageAt = &lastMsg.CreatedAt
	}

	// Render conversation list item (always includes ID)
	conversationItem := ui.ConversationListItem(conversationListItemData(
		conv.ID, conv.AdID, conv.OwnerID, conv.InquirerID, currentUserID,
		a.Title, lastMessageContent, otherUserName,
		lastMessageAt, conv.UpdatedAt,
		hasUnread, a.RockCount, otherUserRockCount,
		time.UTC,
	))
	conversationItemSwapOOB := ui.ConversationListItemSwapOOB(conv.ID, conversationItem)
	itemHTML, err := renderToString(conversationItemSwapOOB)
	if err != nil {
		logger.Error("Failed to render conversation list item for SSE", "error", err, "conversationID", conv.ID)
		return
	}

	SendSSEEvent(currentUserID, SSEEvent{
		Event: ui.SSEEventConversationList,
		Data:  itemHTML,
	})
}

func MessageModalHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot message an inactive or deleted ad")
	}

	if !local.IsLoggedIn(currentUserID) {
		return renderLoggedOutConversationModal(c, a)
	}

	if a.UserID == currentUserID {
		return fiber.NewError(fiber.StatusForbidden, "You cannot message your own ad")
	}

	// Try to get existing conversation
	conv, err := message.GetConversationByAdAndInquirer(adID, a.UserID, currentUserID)
	if err != nil {
		if err == message.ErrConversationNotFound {
			// No conversation exists yet - create a temporary conversation struct for the modal
			// The conversation will be created when the first message is sent
			conv = message.Conversation{
				ID:            0, // 0 indicates conversation doesn't exist yet
				AdID:          adID,
				OwnerID:       a.UserID,
				InquirerID:    currentUserID,
				RockThrowerID: nil,
				RockThrownAt:  nil,
			}
		} else {
			logger.Error("Failed to get conversation", "error", err, "adID", adID, "ownerID", a.UserID, "inquirerID", currentUserID)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get conversation")
		}
	}

	return renderConversationModal(c, conv, currentUserID, tz, csrfToken)
}

func renderLoggedOutConversationModal(c *fiber.Ctx, a ad.Ad) error {
	// Do not reveal the owner's username (or a profile link) while logged out.
	conv := message.Conversation{
		ID:         0,
		AdID:       a.ID,
		OwnerID:    0,
		InquirerID: 0,
	}
	data := conversationModalData(
		conv, 0, 0, 0,
		a.Title, "<hidden>", "Not logged in",
		"", false, false, false,
		false, false, nil, "",
	)
	data.DisabledInputPlaceholder = "Login to send a message"
	return render(c, ui.ConversationModalWithRock(data))
}

func SendMessageHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	a, err := ad.GetAd(currentUserID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot message an inactive or deleted ad")
	}
	if !user.Exists(a.UserID) {
		return fiber.NewError(fiber.StatusBadRequest, "This account is no longer available")
	}

	// Try to get existing conversation
	conv, err := message.GetConversationByAdAndInquirer(adID, a.UserID, currentUserID)
	isNewConversation := false
	if err != nil {
		if err == message.ErrConversationNotFound {
			// Create conversation when sending first message
			conv, err = message.CreateConversation(adID, a.UserID, currentUserID)
			if err != nil {
				logger.Error("Failed to create conversation", "error", err, "adID", adID, "ownerID", a.UserID, "inquirerID", currentUserID)
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create conversation")
			}
			isNewConversation = true
		} else {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get conversation")
		}
	}

	return sendMessageAndRenderUpdate(c, conv, currentUserID, tz, isNewConversation)
}

func SendConversationMessageHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	// Check if user can post (must be participant)
	canPost, err := message.CanUserPost(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to check permissions")
	}
	if !canPost {
		return fiber.NewError(fiber.StatusForbidden, "You can only read this conversation")
	}

	conv, err := message.GetConversation(conversationID, currentUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}

	return sendMessageAndRenderUpdate(c, conv, currentUserID, tz, false)
}

func rockOpinionModalData(conv message.Conversation, currentUserID int,
	tz *time.Location) (ui.RockOpinionModalData, error) {
	a, err := ad.GetAd(currentUserID, conv.AdID, tz)
	if err != nil {
		return ui.RockOpinionModalData{}, err
	}

	d := ui.RockOpinionModalData{
		ConversationID: conv.ID,
		AdID:           conv.AdID,
		AdTitle:        a.Title,
		AdFacts:        rockopinion.AdFactLines(a, tz),
	}

	op, err := rockopinion.GetOrGenerate(conv, tz)
	if errors.Is(err, rockopinion.ErrUnavailable) {
		d.Unavailable = true
		return d, nil
	}
	if err != nil {
		return ui.RockOpinionModalData{}, err
	}

	d.Summary = op.Summary
	d.AssessmentScore = op.Assessment
	d.AssessmentDetail = op.AssessmentDetail
	d.Resolution = op.Resolution
	d.Reasoning = op.Reasoning
	genAt := op.GeneratedAt
	if tz != nil {
		genAt = genAt.In(tz)
	}
	d.GeneratedAt = &genAt
	return d, nil
}

func renderRockOpinionModal(c *fiber.Ctx, conv message.Conversation,
	currentUserID int, tz *time.Location) error {
	d, err := rockOpinionModalData(conv, currentUserID, tz)
	if err != nil {
		logger.Error("Failed to build rock opinion modal", "error", err)
		return fiber.NewError(
			fiber.StatusInternalServerError,
			"Failed to load dispute assessment",
		)
	}
	return render(c, ui.RockOpinionModal(d))
}

func RockOpinionHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	conv, err := message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conv.RockThrowerID == nil {
		return fiber.NewError(fiber.StatusNotFound, "No dispute assessment")
	}

	return renderRockOpinionModal(c, conv, currentUserID, tz)
}

func ConversationModalHandler(c *fiber.Ctx) error {
	conversationID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid conversation ID")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)

	conv, markedRead, err := message.OpenConversation(conversationID, currentUserID)
	if errors.Is(err, message.ErrModalAccess) {
		return fiber.NewError(fiber.StatusForbidden, "Conversation not found")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}

	if markedRead {
		sendConversationListItemUpdate(conv, currentUserID, false)

		hasUnread, err := message.GetHasUnread(currentUserID)
		if err == nil {
			sendUnreadIndicatorUpdate(currentUserID, hasUnread)
		}
	}

	return renderConversationModal(c, conv, currentUserID, tz, csrfToken)
}

func UserMessagesHandler(c *fiber.Ctx) error {
	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)

	conversations, err := message.GetUserConversations(currentUserID, tz)
	if err != nil {
		logger.Error("Failed to get user conversations", "error", err, "userID", currentUserID)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load conversations")
	}

	items := make([]ui.ConversationListItemData, len(conversations))
	for i, conv := range conversations {
		items[i] = conversationListItemFromConv(conv, currentUserID)
	}

	return renderPage(c, "Messages", ui.MessagesPage(items))
}
