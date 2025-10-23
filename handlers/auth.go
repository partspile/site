package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

func logoutUser(c *fiber.Ctx) {
	u := getUser(c)
	redisSetUserInvalid(u.ID)
	sessionDestroy(c)
}

func redirectToLogin(c *fiber.Ctx) error {
	// For HTMX requests, return a redirect response that HTMX can handle
	if c.Get("HX-Request") != "" {
		c.Set("HX-Redirect", "/login")
		return c.Status(fiber.StatusSeeOther).SendString("")
	}
	// For regular requests, redirect to login page
	return c.Redirect("/login", fiber.StatusSeeOther)
}

// AuthRequired is a middleware that requires a user to be logged in.
func AuthRequired(c *fiber.Ctx) error {
	u := getUser(c)
	if u == nil {
		return redirectToLogin(c)
	}
	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
func AdminRequired(c *fiber.Ctx) error {
	u := getUser(c)
	if u == nil || !u.IsAdmin {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}
	return c.Next()
}

func HandleLogin(c *fiber.Ctx) error {
	return render(c, ui.LoginPage(getUser(c), c.Path()))
}

func HandleLogout(c *fiber.Ctx) error {
	logoutUser(c)
	return redirectToLogin(c)
}

func HandleLoginSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	userPassword := c.FormValue("password")

	log.Printf("[AUTH] Login attempt: name=%s", name)

	u, err := user.GetUserByName(name)
	if err != nil {
		log.Printf("[AUTH] Login failed: user not found: %s", name)
		return ValidationErrorResponse(c, "Invalid username or password")
	}

	if !password.VerifyPassword(userPassword, u.PasswordHash, u.PasswordSalt) {
		log.Printf("[AUTH] Login failed: bad password for user=%s", name)
		return ValidationErrorResponse(c, "Invalid username or password")
	}

	if err := sessionSetUser(c, &u); err != nil {
		log.Printf("[AUTH] Login failed: session set user error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	redisSetUserValid(u.ID)

	log.Printf("[AUTH] Login successful: userID=%d, name=%s", u.ID, name)
	return render(c, ui.SuccessMessage("Login successful", "/"))
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

	u := getUser(c)
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

	u := getUser(c)
	if u == nil {
		return ValidationErrorResponseWithStatus(c, "You must be logged in to delete your account", fiber.StatusUnauthorized)
	}

	if !password.VerifyPassword(userPassword, u.PasswordHash, u.PasswordSalt) {
		return ValidationErrorResponseWithStatus(c, "Invalid password", fiber.StatusUnauthorized)
	}

	// Archive all ads by this user
	err := ad.ArchiveAdsByUserID(u.ID)
	if err != nil {
		log.Printf("Warning: Failed to archive user's ads: %v", err)
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
