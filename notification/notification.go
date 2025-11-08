package notification

import (
	"fmt"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/email"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/messaging"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/user"
)

// NotificationService handles sending notifications for new messages
type NotificationService struct {
	emailService *email.EmailService
	hasSMS       bool
}

// NewNotificationService creates a new notification service
func NewNotificationService() (*NotificationService, error) {
	hasSMS := sms.IsSMSAvailable()

	emailService, err := email.NewEmailService()
	if err != nil {
		// Log error but don't fail - SMS notifications can still work
		logger.Warn("Email service not available",
			"error", err)
	}

	// At least one service should be available
	if !hasSMS && emailService == nil {
		return nil, fmt.Errorf("no notification services available - check configuration")
	}

	return &NotificationService{
		emailService: emailService,
		hasSMS:       hasSMS,
	}, nil
}

// NotifyNewMessage sends a notification to the recipient about a new message
func (n *NotificationService) NotifyNewMessage(conversationID int, senderID, recipientID int, messageContent string) error {
	// Get conversation details
	conversation, err := messaging.GetConversationByID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Get sender and recipient details
	sender, _, found := user.GetUserByID(senderID)
	if !found {
		return fmt.Errorf("sender not found")
	}

	recipient, _, found := user.GetUserByID(recipientID)
	if !found {
		return fmt.Errorf("recipient not found")
	}

	// Get ad details
	ad, err := getAdDetails(conversation.AdID)
	if err != nil {
		return fmt.Errorf("failed to get ad details: %w", err)
	}

	// Send notification based on recipient's preference
	switch recipient.NotificationMethod {
	case user.NotificationMethodSMS:
		return n.sendSMSNotification(recipient.Phone, sender.Name, ad.Title, messageContent, conversationID)
	case user.NotificationMethodEmail:
		if recipient.EmailAddress == nil {
			return fmt.Errorf("recipient has email notifications enabled but no email address")
		}
		return n.sendEmailNotification(*recipient.EmailAddress, sender.Name, ad.Title, messageContent, conversationID)
	default:
		logger.Warn("Unknown notification method",
			"notificationMethod", recipient.NotificationMethod,
			"userID", recipientID)
		return nil
	}
}

// sendSMSNotification sends an SMS notification about a new message
func (n *NotificationService) sendSMSNotification(phoneNumber, senderName, adTitle, messageContent string, conversationID int) error {
	if !n.hasSMS {
		return fmt.Errorf("SMS service not available")
	}

	// Validate phone number format (basic check)
	if len(phoneNumber) < 10 {
		return fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	// Truncate message content for SMS
	truncatedContent := messageContent
	if len(truncatedContent) > 50 {
		truncatedContent = truncatedContent[:47] + "..."
	}

	// Create SMS message
	message := fmt.Sprintf("New message from %s about '%s': %s. View at: %s/messages/%d",
		senderName, adTitle, truncatedContent, config.BaseURL, conversationID)

	// Send SMS using Twilio
	err := sms.SendMessage(phoneNumber, message)
	if err != nil {
		logger.Error("SMS notification failed",
			"error", err, "phoneNumber", phoneNumber)
		return err
	}

	logger.Info("SMS notification sent successfully",
		"phoneNumber", phoneNumber)
	return nil
}

// sendEmailNotification sends an email notification about a new message
func (n *NotificationService) sendEmailNotification(emailAddress, senderName, adTitle, messageContent string, conversationID int) error {
	if n.emailService == nil {
		return fmt.Errorf("email service not available")
	}

	return n.emailService.SendNotificationEmail(emailAddress, senderName, adTitle, messageContent, conversationID)
}

// getAdDetails is a helper function to get ad details
func getAdDetails(adID int) (struct{ Title string }, error) {
	ads, err := ad.GetAdsByIDs([]int{adID}, 0)
	if err != nil || len(ads) == 0 {
		return struct{ Title string }{}, fmt.Errorf("ad not found")
	}
	return struct{ Title string }{Title: ads[0].Title}, nil
}

// NotifyPing sends a ping notification
func (n *NotificationService) NotifyPing(conversationID int, senderID, recipientID int) error {
	conversation, err := messaging.GetConversationByID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	sender, _, found := user.GetUserByID(senderID)
	if !found {
		return fmt.Errorf("sender not found")
	}

	recipient, _, found := user.GetUserByID(recipientID)
	if !found {
		return fmt.Errorf("recipient not found")
	}

	adDetails, err := getAdDetails(conversation.AdID)
	if err != nil {
		return fmt.Errorf("failed to get ad details: %w", err)
	}

	// Ping notification message
	message := fmt.Sprintf("%s pinged you about '%s'. View at: %s/messages/%d",
		sender.Name, adDetails.Title, config.BaseURL, conversationID)

	switch recipient.NotificationMethod {
	case user.NotificationMethodSMS:
		return sms.SendMessage(recipient.Phone, message)
	case user.NotificationMethodEmail:
		if recipient.EmailAddress == nil {
			return fmt.Errorf("recipient has email notifications enabled but no email address")
		}
		return n.emailService.SendPingEmail(*recipient.EmailAddress, sender.Name, adDetails.Title, conversationID)
	default:
		return nil
	}
}

// NotifySummary sends a summary notification of unread conversations
func (n *NotificationService) NotifySummary(userID int, conversations []messaging.Conversation) error {
	recipient, _, found := user.GetUserByID(userID)
	if !found {
		return fmt.Errorf("recipient not found")
	}

	if len(conversations) == 0 {
		return nil // No conversations to summarize
	}

	message := fmt.Sprintf("You have new messages in %d conversation(s). View at: %s/messages",
		len(conversations), config.BaseURL)

	switch recipient.NotificationMethod {
	case user.NotificationMethodSMS:
		return sms.SendMessage(recipient.Phone, message)
	case user.NotificationMethodEmail:
		if recipient.EmailAddress == nil {
			return fmt.Errorf("recipient has email notifications enabled but no email address")
		}
		return n.emailService.SendSummaryEmail(*recipient.EmailAddress, len(conversations))
	default:
		return nil
	}
}
