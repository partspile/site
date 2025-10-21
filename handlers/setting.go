package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	. "maragu.dev/gomponents/html"
)

func HandleSettings(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}

func HandleUserMenu(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.UserMenuPopup(currentUser, c.Path()))
}

// HandleUpdateNotificationMethod updates the user's notification method preference
func HandleUpdateNotificationMethod(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)

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
	currentUser, _ := CurrentUser(c)

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
