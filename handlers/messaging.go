package handlers

import (
	"fmt"
	"log"
	"strconv"

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

	return render(c, ui.MessagesPage(currentUser, conversations))
}

// HandleConversationPage handles a specific conversation page
func HandleConversationPage(c *fiber.Ctx) error {
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

	// Mark messages as read
	err = messaging.MarkMessagesAsRead(conversationID, currentUser.ID)
	if err != nil {
		// Log error but don't fail the request
		log.Printf("Failed to mark messages as read: %v", err)
	}

	messages, err := messaging.GetMessages(conversationID)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to load messages"))
	}

	return render(c, ui.ConversationPage(currentUser, conversation, messages))
}

// HandleStartConversation handles starting a new conversation about an ad
func HandleStartConversation(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := strconv.Atoi(c.Params("adID"))
	if err != nil {
		return render(c, ui.ErrorPage(400, "Invalid ad ID"))
	}

	// Get ad details to check ownership
	ad, found := ad.GetAd(adID)
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

	// Redirect to conversation page
	return c.Redirect(fmt.Sprintf("/messages/%d", conversationID))
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

	// Redirect back to conversation page
	return c.Redirect(fmt.Sprintf("/messages/%d", conversationID))
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
