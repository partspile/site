package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	. "maragu.dev/gomponents/html"
)

func HandleSettings(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	// Fetch current user to prefill notification settings
	currentUser, err := user.GetUser(userID)
	if err != nil || currentUser.IsArchived() {
		return c.Status(fiber.StatusUnauthorized).SendString("User not found")
	}
	log.Printf("[SETTINGS] Loading settings for user %d: SMSOptedOut=%v, NotificationMethod=%s", userID, currentUser.SMSOptedOut, currentUser.NotificationMethod)
	return render(c, ui.SettingsPage(
		userID,
		userName,
		c.Path(),
		currentUser.NotificationMethod,
		currentUser.EmailAddress,
		currentUser.Phone,
		currentUser.SMSOptedOut,
	))
}

func HandleUserMenu(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	// Fetch current user data with admin status
	currentUser, err := user.GetUser(userID)
	if err != nil || currentUser.IsArchived() {
		return c.Status(fiber.StatusUnauthorized).SendString("User not found")
	}

	return render(c, ui.UserMenuPopup(&currentUser, c.Path()))
}

// HandleUpdateNotificationMethod updates the user's notification method preference
func HandleUpdateNotificationMethod(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

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
	if err := user.UpdateNotificationPreferences(userID, request.NotificationMethod, request.EmailAddress); err != nil {
		log.Printf("[API] Failed to update notification preferences for user %d: %v", userID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update notification preferences")
	}

	return render(c, ui.SuccessMessage("Notification preferences updated successfully", ""))
}

// HandleNotificationMethodChanged handles HTMX requests when notification method changes
func HandleNotificationMethodChanged(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	// Get the current user to retrieve their saved email address
	currentUser, err := user.GetUser(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user")
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

func HandleChangePassword(c *fiber.Ctx) error {
	currentUserPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	if err := password.ValidatePasswordConfirmation(newPassword, confirmNewPassword); err != nil {
		return ValidationErrorResponse(c, "New passwords do not match")
	}

	if err := password.ValidatePasswordStrength(newPassword); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	userID := local.GetUserID(c)
	u, err := user.GetUser(userID)
	if err != nil || u.IsArchived() {
		return ValidationErrorResponse(c, "User not found")
	}
	if !password.VerifyPassword(currentUserPassword, u.PasswordHash, u.PasswordSalt) {
		return ValidationErrorResponse(c, "Invalid current password")
	}
	newHash, newSalt, err := password.HashPassword(newPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to update password.")
	}
	if _, err := user.UpdateUserPassword(u.ID, newHash, newSalt, "argon2id"); err != nil {
		return ValidationErrorResponse(c, "Failed to update password")
	}
	// Log out the user after password change
	logoutUser(c)
	return render(c, ui.SuccessMessage("Password changed successfully. Please log in with your new password.", "/login"))
}

func HandleDeleteAccount(c *fiber.Ctx) error {
	userPassword := c.FormValue("password")

	userID := local.GetUserID(c)
	if userID == 0 {
		return ValidationErrorResponseWithStatus(c, "You must be logged in to delete your account", fiber.StatusUnauthorized)
	}

	u, err := user.GetUser(userID)
	if err != nil || u.IsArchived() {
		return ValidationErrorResponseWithStatus(c, "User not found", fiber.StatusUnauthorized)
	}

	if !password.VerifyPassword(userPassword, u.PasswordHash, u.PasswordSalt) {
		return ValidationErrorResponseWithStatus(c, "Invalid password", fiber.StatusUnauthorized)
	}

	// Archive all ads by this user
	err2 := ad.ArchiveAdsByUserID(u.ID)
	if err2 != nil {
		log.Printf("Warning: Failed to archive user's ads: %v", err2)
		// Continue with user deletion even if ad archiving fails
	}

	// Archive the user using soft delete
	if err := user.ArchiveUser(u.ID); err != nil {
		return ValidationErrorResponseWithStatus(c, "Failed to delete account", fiber.StatusInternalServerError)
	}

	// Log out the user after account deletion
	logoutUser(c)

	return render(c, ui.SuccessMessage("Account deleted successfully", "/login"))
}

// HandleUnstopSMS clears the user's SMS opt-out flag
func HandleUnstopSMS(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}
	if err := user.SetSMSOptOut(userID, false); err != nil {
		log.Printf("[API] Failed to clear SMS opt-out for user %d: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to resume SMS")
	}
	// Get updated user data and refresh the notification method group
	currentUser, err := user.GetUser(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user")
	}
	// Return updated notification method group to refresh the entire section
	return render(c, ui.NotificationMethodRadioGroup(
		currentUser.NotificationMethod,
		currentUser.EmailAddress,
		currentUser.Phone,
		currentUser.SMSOptedOut))
}
