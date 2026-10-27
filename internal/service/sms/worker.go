package sms

import (
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/user"
)

// StartSMSWorker starts the background worker that processes SMS notifications
func StartSMSWorker() {
	logger.Info("Starting SMS worker", "component", "SMS")

	go processSMSQueue()
	go cleanupOldQueueRecords()
}

// processSMSQueue processes pending SMS notifications
func processSMSQueue() {
	ticker := time.NewTicker(config.SMSWorkerPollInterval)
	defer ticker.Stop()

	for range ticker.C {
		notifications, err := DequeuePendingNotifications(10)
		if err != nil {
			logger.Error("Failed to dequeue notifications",
				"error", err,
				"component", "SMS")
			continue
		}

		for _, notification := range notifications {
			processNotification(notification)
		}
	}
}

// processNotification processes a single notification
func processNotification(n Notification) {
	// Check if SMS should be suppressed
	shouldSuppress, err := ShouldSuppressSMS(n.RecipientUserID)
	if err != nil {
		logger.Error("Failed to check suppression",
			"error", err,
			"component", "SMS",
			"notificationID", n.ID,
			"recipientUserID", n.RecipientUserID)
		// Don't mark as suppressed on error, let it retry
		return
	}

	if shouldSuppress {
		if err := MarkSuppressed(n.ID); err != nil {
			logger.Error("Failed to mark notification as suppressed",
				"error", err,
				"component", "SMS",
				"notificationID", n.ID)
		} else {
			logger.Debug("SMS notification suppressed",
				"component", "SMS",
				"notificationID", n.ID,
				"recipientUserID", n.RecipientUserID)
		}
		return
	}

	// Generate and send SMS
	smsMessage, err := generateSMSMessage(n.RecipientUserID)
	if err != nil {
		logger.Error("Failed to generate SMS message",
			"error", err,
			"component", "SMS",
			"notificationID", n.ID,
			"recipientUserID", n.RecipientUserID)
		// Don't mark as processed on error, let it retry
		return
	}

	// Get recipient user to get phone number
	recipientUser, err := user.GetByID(n.RecipientUserID)
	if err != nil {
		logger.Error("Failed to get recipient user",
			"error", err,
			"component", "SMS",
			"notificationID", n.ID,
			"recipientUserID", n.RecipientUserID)
		return
	}

	// Check if test number (pattern: +1555010xxxx)
	if isTestPhoneNumber(recipientUser.PhoneE64) {
		logger.Info("SMS notification (test number - not sent)",
			"component", "SMS",
			"notificationID", n.ID,
			"recipientUserID", n.RecipientUserID,
			"phoneNumber", recipientUser.PhoneE64)
		if err := MarkProcessed(n.ID); err != nil {
			logger.Error("Failed to mark test notification as processed",
				"error", err,
				"component", "SMS",
				"notificationID", n.ID)
		}
		return
	}

	// Send SMS
	err = SendMessage(recipientUser.PhoneE64, smsMessage)
	if err != nil {
		logger.Error("Failed to send SMS",
			"error", err,
			"component", "SMS",
			"notificationID", n.ID,
			"recipientUserID", n.RecipientUserID,
			"phoneNumber", recipientUser.PhoneE64)
		// Don't mark as processed on error, let it retry
		return
	}

	// Update last SMS sent timestamp
	if err := UpdateLastSMSSent(n.RecipientUserID); err != nil {
		logger.Error("Failed to update last SMS sent timestamp",
			"error", err,
			"component", "SMS",
			"recipientUserID", n.RecipientUserID)
		// Continue even if this fails
	}

	// Mark as processed
	if err := MarkProcessed(n.ID); err != nil {
		logger.Error("Failed to mark notification as processed",
			"error", err,
			"component", "SMS",
			"notificationID", n.ID)
	} else {
		logger.Info("SMS notification processed successfully",
			"component", "SMS",
			"notificationID", n.ID,
			"recipientUserID", n.RecipientUserID,
			"phoneNumber", recipientUser.PhoneE64)
	}
}

// generateSMSMessage generates the SMS message with unread count and link
func generateSMSMessage(userID int) (string, error) {
	unreadCount, err := GetUnreadMessageCount(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get unread count: %w", err)
	}

	messageURL := fmt.Sprintf("%s/auth/user/messages", config.PublicSiteURL)
	var message string
	if unreadCount == 1 {
		message = fmt.Sprintf("You have 1 new message. View: %s. Reply STOP to unsubscribe.", messageURL)
	} else {
		message = fmt.Sprintf("You have %d new messages. View: %s. Reply STOP to unsubscribe.", unreadCount, messageURL)
	}

	return message, nil
}

// cleanupOldQueueRecords periodically cleans up old processed/suppressed records
func cleanupOldQueueRecords() {
	ticker := time.NewTicker(config.SMSQueueCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := CleanupOldRecords(config.SMSQueueRetentionHours); err != nil {
			logger.Error("Failed to cleanup old queue records",
				"error", err,
				"component", "SMS")
		} else {
			logger.Debug("Cleaned up old queue records",
				"component", "SMS",
				"retentionHours", config.SMSQueueRetentionHours)
		}
	}
}
