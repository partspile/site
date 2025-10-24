package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/jwt"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

func logoutUser(c *fiber.Ctx) {
	cookie.ClearJWT(c)
}

func getUserID(c *fiber.Ctx) int {
	userID, _ := c.Locals("userID").(int)
	return userID
}

func setUserID(c *fiber.Ctx, userID int) {
	c.Locals("userID", userID)
}

func setUserName(c *fiber.Ctx, userName string) {
	c.Locals("userName", userName)
}

func getUserName(c *fiber.Ctx) string {
	userName, _ := c.Locals("userName").(string)
	return userName
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

// JWTMiddleware is a middleware that validates a JWT token and sets the user in the context.
func JWTMiddleware(c *fiber.Ctx) error {
	// Get JWT token from cookie
	tokenString := cookie.GetJWT(c)
	if tokenString == "" {
		setUserID(c, 0)
		setUserName(c, "")
		return c.Next()
	}

	// Validate JWT token
	claims, err := jwt.ValidateToken(tokenString)
	if err != nil {
		// Invalid token, clear cookie
		cookie.ClearJWT(c)
		setUserID(c, 0)
		setUserName(c, "")
		return c.Next()
	}

	// Set user ID and username in context
	setUserID(c, jwt.GetUserID(claims))
	setUserName(c, jwt.GetUserName(claims))
	return c.Next()
}

// AuthRequired is a middleware that requires a user to be logged in.
func AuthRequired(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == 0 {
		return redirectToLogin(c)
	}
	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
func AdminRequired(c *fiber.Ctx) error {
	userID := getUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	// Fetch current admin status from database
	u, err := user.GetUser(userID)
	if err != nil || u.IsArchived() {
		return c.Status(fiber.StatusUnauthorized).SendString("User not found")
	}

	if !u.IsAdmin {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	return c.Next()
}

func HandleLogin(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)
	return render(c, ui.LoginPage(userID, userName, c.Path()))
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

	// Generate JWT token
	token, err := jwt.GenerateToken(&u)
	if err != nil {
		log.Printf("[AUTH] Login failed: JWT generation error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	cookie.SetJWT(c, token)

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

	userID := getUserID(c)
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

	userID := getUserID(c)
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
