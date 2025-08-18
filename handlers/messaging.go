package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/messaging"
	"github.com/parts-pile/site/notification"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

// HandleMessagesPage handles the main messages page
func HandleMessagesPage(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	conversations, err := messaging.GetConversationsForUser(currentUser.ID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load conversations"))
	}

	// Check if we should expand a specific conversation
	expandID := c.Query("expand")
	if expandID != "" {
		// Return the page with the conversation pre-expanded
		return render(c, ui.MessagesPageWithExpanded(currentUser, conversations, expandID))
	}

	return render(c, ui.MessagesPage(currentUser, conversations))
}

// HandleExpandConversation handles expanding a conversation in-place
func HandleExpandConversation(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid conversation ID")
	}

	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).SendString("Conversation not found")
	}

	// Check if user is part of this conversation
	if conversation.User1ID != currentUser.ID && conversation.User2ID != currentUser.ID {
		return c.Status(403).SendString("Access denied")
	}

	// Mark conversation as read for current user
	err = messaging.MarkConversationAsRead(conversationID, currentUser.ID)
	if err != nil {
		log.Printf("Failed to mark conversation as read: %v", err)
	}

	// Mark messages as read
	err = messaging.MarkMessagesAsRead(conversationID, currentUser.ID)
	if err != nil {
		log.Printf("Failed to mark messages as read: %v", err)
	}

	messages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return c.Status(500).SendString("Failed to load messages")
	}

	component := ui.ExpandedConversation(*currentUser, conversation, messages)
	return render(c, component)
}

// HandleCollapseConversation handles collapsing a conversation back to the list view
func HandleCollapseConversation(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid conversation ID")
	}

	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).SendString("Conversation not found")
	}

	// Check if user is part of this conversation
	if conversation.User1ID != currentUser.ID && conversation.User2ID != currentUser.ID {
		return c.Status(403).SendString("Access denied")
	}

	// Return just the collapsed conversation item
	return render(c, ui.ConversationListItem(conversation, currentUser.ID))
}

// HandleStartConversation handles starting a new conversation about an ad
func HandleStartConversation(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return render(c, ui.ErrorPage(400, "Invalid ad ID"))
	}

	// Get ad details to check ownership
	ad, found := ad.GetAd(adID, currentUser)
	if !found {
		return render(c, ui.ErrorPage(404, "Ad not found"))
	}

	// Check if user can message this ad
	err = messaging.CanUserMessageAd(currentUser.ID, ad.UserID)
	if err != nil {
		return render(c, ui.ErrorPage(400, err.Error()))
	}

	// Get or create conversation
	conversationID, err := messaging.GetOrCreateConversation(currentUser.ID, ad.UserID, adID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to create conversation"))
	}

	// Redirect to messages page with conversation expanded
	return c.Redirect(fmt.Sprintf("/messages?expand=%d", conversationID))
}

// HandleSendMessage handles sending a new message
func HandleSendMessage(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return render(c, ui.ErrorPage(400, "Invalid conversation ID"))
	}

	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(404, "Conversation not found"))
	}

	// Check if user is part of this conversation
	if conversation.User1ID != currentUser.ID && conversation.User2ID != currentUser.ID {
		return render(c, ui.ErrorPage(403, "Access denied"))
	}

	// Get message content from form
	content := c.FormValue("message")
	if content == "" {
		return render(c, ui.ErrorPage(400, "Message cannot be empty"))
	}

	// Add message to conversation
	_, err = messaging.AddMessage(conversationID, currentUser.ID, content)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to send message"))
	}

	// Determine recipient ID
	recipientID := conversation.User1ID
	if currentUser.ID == conversation.User1ID {
		recipientID = conversation.User2ID
	}

	// Send notification to recipient
	notificationService, err := notification.NewNotificationService()
	if err != nil {
		// Log error but don't fail the request
		log.Printf("Failed to create notification service: %v", err)
	} else {
		go func() {
			err := notificationService.NotifyNewMessage(conversationID, currentUser.ID, recipientID, content)
			if err != nil {
				log.Printf("Failed to send notification: %v", err)
			}
		}()
	}

	// Get updated conversation and messages
	updatedConversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load updated conversation"))
	}

	updatedMessages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load updated messages"))
	}

	// Check if this is an inline conversation by looking at the referrer
	referrer := c.Get("Referer")
	isInline := strings.Contains(referrer, "/ad/detail/")

	// Return the appropriate conversation editor based on context
	if isInline {
		// Extract ad ID and view from referrer for inline mode
		// For now, default to tree view - this could be enhanced to extract actual view
		view := "tree"
		if strings.Contains(referrer, "view=grid") {
			view = "grid"
		} else if strings.Contains(referrer, "view=list") {
			view = "list"
		}

		// Get ad details for inline mode
		adObj, ok := ad.GetAd(updatedConversation.AdID, currentUser)
		if !ok {
			return render(c, ui.ErrorPage(500, "Failed to get ad details"))
		}

		// Get ad owner for inline mode
		adOwner, err := user.GetUser(adObj.UserID)
		if err != nil {
			return render(c, ui.ErrorPage(500, "Failed to get ad owner"))
		}

		htmxTarget := fmt.Sprintf("#ad-%d", updatedConversation.AdID)
		if view == "grid" {
			htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", updatedConversation.AdID)
		}

		return render(c, ui.ConversationInline(updatedConversation, updatedMessages, adObj, adOwner, *currentUser, htmxTarget, view))
	} else {
		// Return the messages page version
		return render(c, ui.ExpandedConversation(*currentUser, updatedConversation, updatedMessages))
	}
}

// HandleMessagesAPI handles AJAX requests for messages
func HandleMessagesAPI(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	action := c.Params("action")
	switch action {
	case "conversations":
		conversations, err := messaging.GetConversationsForUser(currentUser.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to load conversations"})
		}
		return c.JSON(conversations)
	case "unread-count":
		count, err := messaging.GetUnreadCount(currentUser.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to get unread count"})
		}
		return c.JSON(fiber.Map{"count": count})
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Invalid action"})
	}
}

// HandleSSE handles Server-Sent Events for real-time messaging updates
func HandleSSE(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create a channel for this user's updates
	userUpdates := make(chan messaging.ConversationUpdate, 10)
	defer close(userUpdates)

	// Register this user's update channel
	messaging.RegisterUserUpdates(currentUser.ID, userUpdates)
	defer messaging.UnregisterUserUpdates(currentUser.ID)

	// Send initial unread count
	unreadCount, err := messaging.GetUnreadCount(currentUser.ID)
	if err == nil {
		c.WriteString(fmt.Sprintf("event: unread_count\ndata: %d\n\n", unreadCount))
	}

	// Keep connection alive and send updates
	ticker := time.NewTicker(30 * time.Second) // Keep-alive ping
	defer ticker.Stop()

	for {
		select {
		case update := <-userUpdates:
			// Send conversation update based on type
			switch update.Type {
			case "new_message":
				// Send new message event with conversation ID
				c.WriteString(fmt.Sprintf("event: new_message\ndata: %d\n\n", update.ConversationID))
			case "unread_count":
				// Send unread count update
				c.WriteString(fmt.Sprintf("event: unread_count\ndata: %d\n\n", update.UnreadCount))
			default:
				// Send generic update
				updateJSON, _ := json.Marshal(update)
				c.WriteString(fmt.Sprintf("event: update\ndata: %s\n\n", updateJSON))
			}

		case <-ticker.C:
			// Send keep-alive ping
			c.WriteString("data: {\"type\":\"ping\"}\n\n")

		case <-c.Context().Done():
			// Client disconnected
			return nil
		}
	}
}

// HandleSSEConversationUpdate handles SSE requests for updated conversation items
func HandleSSEConversationUpdate(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid conversation ID")
	}

	// Get the updated conversation
	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).SendString("Conversation not found")
	}

	// Check if user is part of this conversation
	if conversation.User1ID != currentUser.ID && conversation.User2ID != currentUser.ID {
		return c.Status(403).SendString("Access denied")
	}

	// Return the updated conversation item HTML
	return render(c, ui.ConversationListItem(conversation, currentUser.ID))
}

// HandleOpenConversationInline opens a conversation inline within an ad view
func HandleOpenConversationInline(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Prevent users from messaging themselves
	if adObj.UserID == currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You cannot message yourself")
	}

	// Get or create conversation
	convID, err := messaging.GetOrCreateConversation(currentUser.ID, adObj.UserID, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get or create conversation")
	}

	// Get conversation details
	conv, err := messaging.GetConversationByID(convID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get conversation")
	}

	// Get messages for this conversation
	messages, err := messaging.GetMessages(convID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get messages")
	}

	// Get ad owner user info
	adOwner, err := user.GetUser(adObj.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get ad owner")
	}

	// Mark conversation as read for current user
	err = messaging.MarkConversationAsRead(convID, currentUser.ID)
	if err != nil {
		log.Printf("Failed to mark conversation as read: %v", err)
	}

	// Determine the view type from the referrer URL
	referrer := c.Get("Referer")
	view := "tree" // default
	if strings.Contains(referrer, "view=grid") {
		view = "grid"
	} else if strings.Contains(referrer, "view=list") {
		view = "list"
	}

	htmxTarget := fmt.Sprintf("#ad-%d", adID)
	if view == "grid" {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", adID)
	}

	return render(c, ui.ConversationInline(conv, messages, adObj, adOwner, *currentUser, htmxTarget, view))
}
