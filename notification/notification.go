package notification

import (
	"fmt"
	"log"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/email"
	"github.com/parts-pile/site/messaging"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/user"
)

// NotificationService handles sending notifications for new messages
type NotificationService struct {
	smsService   *sms.SMSService
	emailService *email.EmailService
}

// NewNotificationService creates a new notification service
func NewNotificationService() (*NotificationService, error) {
	smsService, err := sms.NewSMSService()
	if err != nil {
		// Log error but don't fail - email notifications can still work
		log.Printf("Warning: SMS service not available: %v", err)
	}

	emailService, err := email.NewEmailService()
	if err != nil {
		// Log error but don't fail - SMS notifications can still work
		log.Printf("Warning: Email service not available: %v", err)
	}

	// At least one service should be available
	if smsService == nil && emailService == nil {
		return nil, fmt.Errorf("no notification services available - check configuration")
	}

	return &NotificationService{
		smsService:   smsService,
		emailService: emailService,
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
		log.Printf("Unknown notification method: %s for user %d", recipient.NotificationMethod, recipientID)
		return nil
	}
}

// sendSMSNotification sends an SMS notification about a new message
func (n *NotificationService) sendSMSNotification(phoneNumber, senderName, adTitle, messageContent string, conversationID int) error {
	if n.smsService == nil {
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
	_, err := n.smsService.SendGeneralMessage(phoneNumber, message)
	if err != nil {
		log.Printf("SMS notification failed for %s: %v", phoneNumber, err)
		return err
	}

	log.Printf("SMS notification sent successfully to %s", phoneNumber)
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
	ads, err := ad.GetAdsByIDs([]int{adID}, nil)
	if err != nil || len(ads) == 0 {
		return struct{ Title string }{}, fmt.Errorf("ad not found")
	}
	return struct{ Title string }{Title: ads[0].Title}, nil
}
