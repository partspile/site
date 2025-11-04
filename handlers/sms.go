package handlers

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/sms"
)

// HandleSMSWebhook processes Twilio webhook callbacks for SMS status updates and incoming messages
func HandleSMSWebhook(c *fiber.Ctx) error {
	// Verify Twilio signature to ensure request is authentic
	if !sms.VerifyTwilioSignature(c) {
		log.Printf("[SMS] Webhook rejected: invalid signature from %s", c.IP())
		return c.Status(403).JSON(fiber.Map{
			"error": "Invalid signature",
		})
	}

	// Parse the webhook data from Twilio
	webhookData, err := sms.ParseWebhook(c)
	if err != nil {
		log.Printf("[SMS] Failed to parse webhook: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid webhook data",
		})
	}

	// Log all incoming webhook data for debugging
	log.Printf("[SMS] Webhook received - From: %s, To: %s, Body: %s, Status: %s, MessageSid: %s",
		webhookData.From, webhookData.To, webhookData.Body, webhookData.MessageStatus, webhookData.MessageSid)

	// Update the status tracker if this is a status update
	if webhookData.MessageStatus != "" && webhookData.MessageSid != "" {
		status := sms.SMSStatus(webhookData.MessageStatus)
		sms.SetMessageStatus(webhookData.MessageSid, status)
	}

	// Handle STOP/UNSTOP responses (incoming messages have Body populated)
	body := strings.ToUpper(strings.TrimSpace(webhookData.Body))
	if body == "STOP" {
		log.Printf("[SMS] Processing STOP request from %s", webhookData.From)
		if err := sms.HandleStopResponse(webhookData.From); err != nil {
			log.Printf("[SMS] Failed to handle STOP response: %v", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to process webhook",
			})
		}
	}
	if body == "UNSTOP" {
		log.Printf("[SMS] Processing UNSTOP request from %s", webhookData.From)
		if err := sms.HandleUnstopResponse(webhookData.From); err != nil {
			log.Printf("[SMS] Failed to handle UNSTOP response: %v", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to process webhook",
			})
		}
	}

	// Return success to Twilio
	return c.JSON(fiber.Map{
		"status": "success",
	})
}
