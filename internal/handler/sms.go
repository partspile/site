package handler

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/accountrecovery"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/service/sms"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
)

var smsOptOutKeywords = map[string]struct{}{
	"STOP": {}, "STOPALL": {}, "UNSUBSCRIBE": {},
	"CANCEL": {}, "END": {}, "QUIT": {},
	"REVOKE": {}, "OPTOUT": {},
}

var smsOptInKeywords = map[string]struct{}{
	"START": {}, "YES": {}, "UNSTOP": {},
}

func smsOptKeyword(body string) (optedOut bool, ok bool) {
	key := strings.ToUpper(strings.TrimSpace(body))
	if _, ok := smsOptOutKeywords[key]; ok {
		return true, true
	}
	if _, ok := smsOptInKeywords[key]; ok {
		return false, true
	}
	return false, false
}

// applySMSPreferenceFromInbound syncs sms_opted_out for STOP/START keywords.
// Returns true if the body was an opt keyword (handled).
func applySMSPreferenceFromInbound(phone, body string) bool {
	optedOut, ok := smsOptKeyword(body)
	if !ok {
		return false
	}
	key := strings.ToUpper(strings.TrimSpace(body))

	u, err := user.GetByPhoneE64(phone)
	if errors.Is(err, sql.ErrNoRows) {
		logger.Info("SMS opt keyword for unknown phone",
			"component", "SMS", "phone", phone, "keyword", key)
		return true
	}
	if err != nil {
		logger.Error("Failed to look up user for SMS opt",
			"error", err, "component", "SMS", "phone", phone,
			"keyword", key)
		return true
	}

	if err := user.SetSMSOptOut(u.ID, optedOut); err != nil {
		logger.Error("Failed to sync SMS opt preference",
			"error", err, "component", "SMS", "phone", phone,
			"keyword", key)
		return true
	}
	logger.Info("SMS opt preference synced from inbound",
		"component", "SMS", "phone", phone, "keyword", key,
		"optedOut", optedOut)
	sendNotificationsSSE(u.ID, optedOut)
	return true
}

func sendNotificationsSSE(userID int, smsOptedOut bool) {
	html, err := renderToString(ui.NotificationsSectionSwapOOB(smsOptedOut))
	if err != nil {
		logger.Error("Failed to render notifications SSE",
			"error", err, "userID", userID)
		return
	}
	SendSSEEvent(userID, SSEEvent{
		Event: ui.SSEEventNotifications,
		Data:  html,
	})
}

// SMSWebhookHandler processes Twilio webhook callbacks for SMS status updates
// and incoming messages
func SMSWebhookHandler(c *fiber.Ctx) error {
	if !sms.VerifyTwilioSignature(c) {
		logger.Warn("Webhook rejected: invalid signature",
			"component", "SMS", "ip", c.IP())
		return c.Status(403).JSON(fiber.Map{
			"error": "Invalid signature",
		})
	}

	webhookData, err := sms.ParseWebhook(c)
	if err != nil {
		logger.Error("Failed to parse webhook",
			"error", err, "component", "SMS")
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid webhook data",
		})
	}

	bodyTrimmed := strings.TrimSpace(webhookData.Body)
	_, isRecover := accountrecovery.ParseRecoverCode(bodyTrimmed)
	logBody := bodyTrimmed
	if isRecover {
		logBody = "RECOVER [redacted]"
	}

	logger.Debug("Webhook received",
		"component", "SMS",
		"hasFrom", webhookData.From != "",
		"hasTo", webhookData.To != "",
		"body", logBody,
		"status", webhookData.MessageStatus,
		"messageSid", webhookData.MessageSid)

	if webhookData.MessageStatus != "" && webhookData.MessageSid != "" {
		status := sms.SMSStatus(webhookData.MessageStatus)
		sms.SetMessageStatus(webhookData.MessageSid, status)
	}

	phone := webhookData.From
	if applySMSPreferenceFromInbound(phone, bodyTrimmed) {
		return c.JSON(fiber.Map{"status": "success"})
	}

	if isRecover {
		logger.Info("Recovery SMS received", "component", "SMS")
		err := accountrecovery.CompleteFromSMS(phone, bodyTrimmed)
		if err != nil {
			if errors.Is(err, accountrecovery.ErrNoUser) ||
				errors.Is(err, accountrecovery.ErrNotFound) ||
				errors.Is(err, accountrecovery.ErrInvalidSMS) {
				logger.Info("Recovery SMS not applied",
					"component", "SMS", "reason", err.Error())
			} else {
				logger.Error("Failed to complete recovery from SMS",
					"error", err, "component", "SMS")
			}
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
	})
}
