package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/sms"
)

// HandleSMSWebhook processes Twilio webhook callbacks for SMS status updates
func HandleSMSWebhook(c *fiber.Ctx) error {
	// Parse the webhook data from Twilio
	var webhookData sms.SMSWebhookData
	if err := c.BodyParser(&webhookData); err != nil {
		log.Printf("[SMS] Failed to parse webhook data: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid webhook data",
		})
	}

	// Create SMS service and handle the webhook
	smsService, err := sms.NewSMSService()
	if err != nil {
		log.Printf("[SMS] Failed to create SMS service: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	// Process the webhook
	if err := smsService.HandleWebhook(webhookData); err != nil {
		log.Printf("[SMS] Failed to handle webhook: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to process webhook",
		})
	}

	// Return success to Twilio
	return c.JSON(fiber.Map{
		"status": "success",
	})
}
