package handlers

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/sms"
)

// HandleSMSWebhook processes Twilio webhook callbacks for SMS status updates
func HandleSMSWebhook(c *fiber.Ctx) error {
	// Parse the webhook data from Twilio
	webhookData, err := sms.ParseWebhook(c)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid webhook data",
		})
	}

	// Update the status tracker
	status := sms.SMSStatus(webhookData.MessageStatus)
	sms.SetMessageStatus(webhookData.MessageSid, status)

	// Handle STOP responses
	if strings.ToUpper(strings.TrimSpace(webhookData.Body)) == "STOP" {
		if err := sms.HandleStopResponse(webhookData.To); err != nil {
			log.Printf("[SMS] Failed to handle STOP response: %v", err)
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
