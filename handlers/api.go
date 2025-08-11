package handlers

import (
	"log"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
)

func HandleMakes(c *fiber.Ctx) error {
	makes := vehicle.GetMakes()
	return c.JSON(makes)
}

func HandleYears(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	years := vehicle.GetYears(makeName)
	return render(c, ui.YearsFormGroup(years))
}

func HandleModels(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one year is required")
	}

	modelAvailability := vehicle.GetModelsWithAvailability(makeName, years)
	return render(c, ui.ModelsFormGroup(modelAvailability))
}

func HandleEngines(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one year is required")
	}

	models := q["models"]
	if len(models) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one model is required")
	}

	engineAvailability := vehicle.GetEnginesWithAvailability(makeName, years, models)
	return render(c, ui.EnginesFormGroup(engineAvailability))
}

func HandleCategories(c *fiber.Ctx) error {
	categories, err := part.GetAllCategories()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get categories")
	}
	return c.JSON(categories)
}

func HandleSubCategories(c *fiber.Ctx) error {
	categoryName := c.Query("category")
	if categoryName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Category is required")
	}

	subCategories, err := part.GetSubCategoriesForCategory(categoryName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get subcategories")
	}
	return render(c, ui.SubCategoriesFormGroupFromStruct(subCategories, ""))
}

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
