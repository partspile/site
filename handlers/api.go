package handlers

import (
	"log"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
	. "maragu.dev/gomponents/html"
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

	models := vehicle.GetModels(makeName, years)
	return render(c, ui.ModelsFormGroup(models))
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

	engines := vehicle.GetEngines(makeName, years, models)
	return render(c, ui.EnginesFormGroup(engines))
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

// HandleUpdateNotificationMethod updates the user's notification method preference
func HandleUpdateNotificationMethod(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
	}

	var request struct {
		NotificationMethod string  `json:"notificationMethod"`
		EmailAddress       *string `json:"emailAddress"`
	}

	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request data")
	}

	// Validate notification method
	validMethods := map[string]bool{
		"sms":    true,
		"email":  true,
		"signal": true,
	}

	if !validMethods[request.NotificationMethod] {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid notification method")
	}

	// Validate email address if email notifications are selected
	if request.NotificationMethod == "email" {
		if request.EmailAddress == nil || *request.EmailAddress == "" {
			return render(c, ui.ValidationError("Email address is required when email notifications are selected"))
		}

		if err := ValidateEmail(*request.EmailAddress); err != nil {
			return render(c, ui.ValidationError(err.Error()))
		}
	}

	// Update both notification method and email address
	if err := user.UpdateNotificationPreferences(currentUser.ID, request.NotificationMethod, request.EmailAddress); err != nil {
		log.Printf("[API] Failed to update notification preferences for user %d: %v", currentUser.ID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update notification preferences")
	}

	return render(c, ui.SuccessMessage("Notification preferences updated successfully", ""))
}

// HandleNotificationMethodChanged handles HTMX requests when notification method changes
func HandleNotificationMethodChanged(c *fiber.Ctx) error {
	// Get the current user to retrieve their saved email address
	currentUser, err := CurrentUser(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Authentication required")
	}

	// Get the selected notification method from the form data
	notificationMethod := c.FormValue("notificationMethod")

	// Return the email input field with appropriate disabled state
	// The field is always visible but disabled when email is not selected
	if notificationMethod == "email" {
		// Email is selected - return normal input field with saved email and required attribute
		return render(c, Input(
			Type("text"),
			ID("emailAddress"),
			Name("emailAddress"),
			Placeholder("Enter email address"),
			Value(func() string {
				if currentUser.EmailAddress != nil {
					return *currentUser.EmailAddress
				}
				return ""
			}()),
			Class("w-full p-2 border rounded"),
			Required(),
		))
	} else {
		// Email is not selected - return disabled input field with saved email
		return render(c, Input(
			Type("text"),
			ID("emailAddress"),
			Name("emailAddress"),
			Placeholder("Enter email address"),
			Value(func() string {
				if currentUser.EmailAddress != nil {
					return *currentUser.EmailAddress
				}
				return ""
			}()),
			Class("w-full p-2 border rounded opacity-50 cursor-not-allowed"),
			Disabled(),
		))
	}
}
