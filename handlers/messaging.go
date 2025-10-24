package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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
	userID := getUserID(c)
	userName := getUserName(c)

	conversations, err := messaging.GetConversationsForUser(userID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load conversations", userID, userName))
	}

	// Check if we should expand a specific conversation
	expandID := c.Query("expand")
	if expandID != "" {
		// Return the page with the conversation pre-expanded
		return render(c, ui.MessagesPageWithExpanded(userID, userName, conversations, expandID))
	}

	return render(c, ui.MessagesPage(userID, userName, conversations))
}

// HandleExpandConversation handles expanding a conversation in-place
func HandleExpandConversation(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid conversation ID")
	}

	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).SendString("Conversation not found")
	}

	// Check if user is part of this conversation
	if conversation.User1ID != userID && conversation.User2ID != userID {
		return c.Status(403).SendString("Access denied")
	}

	// Mark conversation as read for current user
	err = messaging.MarkConversationAsRead(conversationID, userID)
	if err != nil {
		log.Printf("Failed to mark conversation as read: %v", err)
	}

	// Mark messages as read
	err = messaging.MarkMessagesAsRead(conversationID, userID)
	if err != nil {
		log.Printf("Failed to mark messages as read: %v", err)
	}

	messages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return c.Status(500).SendString("Failed to load messages")
	}

	component := ui.ExpandedConversation(userID, userName, conversation, messages)
	return render(c, component)
}

// HandleCollapseConversation handles collapsing a conversation back to the list view
func HandleCollapseConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid conversation ID")
	}

	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).SendString("Conversation not found")
	}

	// Check if user is part of this conversation
	if conversation.User1ID != userID && conversation.User2ID != userID {
		return c.Status(403).SendString("Access denied")
	}

	// Return just the collapsed conversation item
	return render(c, ui.ConversationListItem(conversation, userID))
}

// HandleStartConversation handles starting a new conversation about an ad
func HandleStartConversation(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)

	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return render(c, ui.ErrorPage(400, "Invalid ad ID", userID, userName))
	}

	// Get ad details to check ownership
	adObj, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return render(c, ui.ErrorPage(404, "Ad not found", userID, userName))
	}

	// Check if user can message this ad
	err = messaging.CanUserMessageAd(userID, adObj.UserID)
	if err != nil {
		return render(c, ui.ErrorPage(400, err.Error(), userID, userName))
	}

	// Get or create conversation
	conversationID, err := messaging.GetOrCreateConversation(userID, adObj.UserID, adID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to create conversation", userID, userName))
	}

	// Redirect to messages page with conversation expanded
	return c.Redirect(fmt.Sprintf("/messages?expand=%d", conversationID))
}

// HandleSendMessage handles sending a new message (works for both main messages page and modal)
func HandleSendMessage(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return render(c, ui.ErrorPage(400, "Invalid conversation ID", userID, userName))
	}

	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(404, "Conversation not found", userID, userName))
	}

	// Check if user is part of this conversation
	if conversation.User1ID != userID && conversation.User2ID != userID {
		return render(c, ui.ErrorPage(403, "Access denied", userID, userName))
	}

	// Get message content from form
	content := c.FormValue("message")
	if content == "" {
		return render(c, ui.ErrorPage(400, "Message cannot be empty", userID, userName))
	}

	// Add message to conversation
	_, err = messaging.AddMessage(conversationID, userID, content)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to send message", userID, userName))
	}

	// Determine recipient ID
	recipientID := conversation.User1ID
	if userID == conversation.User1ID {
		recipientID = conversation.User2ID
	}

	// Send notification to recipient
	notificationService, err := notification.NewNotificationService()
	if err != nil {
		// Log error but don't fail the request
		log.Printf("Failed to create notification service: %v", err)
	} else {
		go func() {
			err := notificationService.NotifyNewMessage(conversationID, userID, recipientID, content)
			if err != nil {
				log.Printf("Failed to send notification: %v", err)
			}
		}()
	}

	// Get updated messages
	updatedMessages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load updated messages", userID, userName))
	}

	// Determine context from query parameter or form data
	context := c.Query("context", c.FormValue("context"))

	// Return appropriate UI component based on context
	switch context {
	case "modal":
		// Return modal conversation for ad detail modal
		return render(c, ui.ModalConversation(updatedMessages, userID))
	default:
		// Return expanded conversation for main messages page
		updatedConversation, err := messaging.GetConversationWithDetails(conversationID)
		if err != nil {
			return render(c, ui.ErrorPage(500, "Failed to load updated conversation", userID, userName))
		}
		return render(c, ui.ExpandedConversation(userID, userName, updatedConversation, updatedMessages))
	}
}

// HandleModalMessaging handles modal messaging interface for ad detail pages
func HandleModalMessaging(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)

	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return render(c, ui.ErrorPage(400, "Invalid ad ID", userID, userName))
	}

	// Get ad details to check ownership
	adObj, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return render(c, ui.ErrorPage(404, "Ad not found", userID, userName))
	}

	// Check if user can message this ad
	err = messaging.CanUserMessageAd(userID, adObj.UserID)
	if err != nil {
		return render(c, ui.ErrorPage(400, err.Error(), userID, userName))
	}

	// Get or create conversation
	conversationID, err := messaging.GetOrCreateConversation(userID, adObj.UserID, adID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to create conversation", userID, userName))
	}

	// Get conversation details and messages
	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load conversation", userID, userName))
	}

	messages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load messages", userID, userName))
	}

	// Return the modal messaging interface
	return render(c, ui.ModalMessagingInterface(userID, userName, *adObj, conversation, messages))
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
	userID := getUserID(c)
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
	if conversation.User1ID != userID && conversation.User2ID != userID {
		return c.Status(403).SendString("Access denied")
	}

	// Return the updated conversation item HTML
	return render(c, ui.ConversationListItem(conversation, userID))
}

// HandleMessageModal shows the message modal for an ad
func HandleMessageModal(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)
	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	adObj, err := ad.GetAdByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Check if user is logged in and not viewing their own ad
	if userID == 0 {
		return fiber.NewError(fiber.StatusUnauthorized, "Must be logged in to message")
	}
	if adObj.UserID == userID {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot message yourself")
	}
	if adObj.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot message about archived ad")
	}

	// Check if user can message this ad
	err = messaging.CanUserMessageAd(userID, adObj.UserID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Get or create conversation
	conversationID, err := messaging.GetOrCreateConversation(userID, adObj.UserID, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create conversation")
	}

	// Get conversation details and messages
	conversation, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load conversation")
	}

	messages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load messages")
	}

	// Return the complete modal with conversation content
	conversationContent := ui.ModalMessagingInterface(userID, userName, *adObj, conversation, messages)
	return render(c, ui.MessageModal(*adObj, conversationContent))
}
