package handlers

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

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

	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		log.Printf("[AUTH] Login failed: session store error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	sess.Set("userID", u.ID)

	if err := sess.Save(); err != nil {
		log.Printf("[AUTH] Login failed: session save error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to save session.")
	}

	log.Printf("[AUTH] Login successful: userID=%d, name=%s", u.ID, name)
	return render(c, ui.SuccessMessage("Login successful", "/"))
}

// logoutUser destroys the user's session
func logoutUser(c *fiber.Ctx) error {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		// Session might already be gone, that's okay
		return nil
	}
	return sess.Destroy()
}

func HandleLogout(c *fiber.Ctx) error {
	logoutUser(c)
	return render(c, ui.LoginPage(nil, "/login"))
}

func GetCurrentUser(c *fiber.Ctx) (*user.User, error) {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		log.Printf("[AUTH] GetCurrentUser: session error: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("session error: %v", err))
	}

	userID, ok := sess.Get("userID").(int)
	if !ok || userID == 0 {
		log.Printf("[AUTH] GetCurrentUser: no user ID in session")
		return nil, fiber.NewError(fiber.StatusUnauthorized, "no user ID in session")
	}

	u, status, found := user.GetUserByID(userID)
	if !found {
		log.Printf("[AUTH] GetCurrentUser: user not found for userID=%d", userID)
		return nil, fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	// Only return active users for current user sessions
	if status == user.StatusArchived {
		log.Printf("[AUTH] GetCurrentUser: user archived userID=%d", userID)
		return nil, fiber.NewError(fiber.StatusForbidden, "user is archived")
	}

	log.Printf("[AUTH] GetCurrentUser: userID=%d found and active", userID)
	return &u, nil
}

// AuthRequired is a middleware that requires a user to be logged in.
// Assumes StashUser middleware has already run and populated c.Locals("user").
func AuthRequired(c *fiber.Ctx) error {
	u, _ := CurrentUser(c)
	if u == nil {
		// For HTMX requests, return a redirect response that HTMX can handle
		if c.Get("HX-Request") != "" {
			c.Set("HX-Redirect", "/login")
			return c.Status(fiber.StatusSeeOther).SendString("")
		}
		// For regular requests, redirect to login page
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	return c.Next()
}

// StashUser is a global middleware that populates c.Locals("user") for all requests.
// It either stores a valid user object or nil, ensuring subsequent middleware/handlers
// can always rely on c.Locals("user") being populated.
func StashUser(c *fiber.Ctx) error {
	// Always try to get user from session and stash in context
	user, err := GetCurrentUser(c)
	if err == nil {
		c.Locals("user", user)
	} else {
		c.Locals("user", nil) // Explicitly set to nil for consistency
	}
	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
// Assumes StashUser middleware has already run and populated c.Locals("user").
func AdminRequired(c *fiber.Ctx) error {
	u, _ := CurrentUser(c)
	if u == nil {
		// For HTMX requests, return a redirect response that HTMX can handle
		if c.Get("HX-Request") != "" {
			c.Set("HX-Redirect", "/login")
			return c.Status(fiber.StatusSeeOther).SendString("")
		}
		// For regular requests, redirect to login page
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	if !u.IsAdmin {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	return c.Next()
}

func HandleLogin(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.LoginPage(currentUser, c.Path()))
}

func HandleChangePassword(c *fiber.Ctx) error {
	currentUserPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	if err := password.ValidatePasswordConfirmation(newPassword, confirmNewPassword); err != nil {
		return ValidationErrorResponse(c, "New passwords do not match")
	}

	// Validate new password strength
	if err := password.ValidatePasswordStrength(newPassword); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	currentUser := c.Locals("user").(*user.User)
	if !password.VerifyPassword(currentUserPassword, currentUser.PasswordHash, currentUser.PasswordSalt) {
		return ValidationErrorResponse(c, "Invalid current password")
	}
	newHash, newSalt, err := password.HashPassword(newPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to update password.")
	}
	if _, err := user.UpdateUserPassword(currentUser.ID, newHash, newSalt, "argon2id"); err != nil {
		return ValidationErrorResponse(c, "Failed to update password")
	}
	// Log out the user after password change
	logoutUser(c)
	return render(c, ui.SuccessMessage("Password changed successfully. Please log in with your new password.", "/login"))
}

func HandleDeleteAccount(c *fiber.Ctx) error {
	userPassword := c.FormValue("password")

	currentUser := c.Locals("user").(*user.User)
	if currentUser == nil {
		return ValidationErrorResponseWithStatus(c, "You must be logged in to delete your account", fiber.StatusUnauthorized)
	}

	if !password.VerifyPassword(userPassword, currentUser.PasswordHash, currentUser.PasswordSalt) {
		return ValidationErrorResponseWithStatus(c, "Invalid password", fiber.StatusUnauthorized)
	}

	// Archive all ads by this user
	err := ad.ArchiveAdsByUserID(currentUser.ID)
	if err != nil {
		log.Printf("Warning: Failed to archive user's ads: %v", err)
		// Continue with user deletion even if ad archiving fails
	}

	// Archive the user using soft delete
	if err := user.ArchiveUser(currentUser.ID); err != nil {
		return ValidationErrorResponseWithStatus(c, "Failed to delete account", fiber.StatusInternalServerError)
	}

	// Log out the user after account deletion
	logoutUser(c)

	return render(c, ui.SuccessMessage("Account deleted successfully", "/"))
}

// CurrentUser extracts the user from context and returns both user and userID.
// Returns (nil, 0) if no user is logged in.
// Assumes StashUser middleware has already run and populated c.Locals("user").
func CurrentUser(c *fiber.Ctx) (*user.User, int) {
	localUser := c.Locals("user")
	if localUser == nil {
		return nil, 0
	}

	u, ok := localUser.(*user.User)
	if !ok || u == nil {
		return nil, 0
	}

	log.Printf("[DEBUG] CurrentUser returning user from context: userID=%d", u.ID)
	return u, u.ID
}

// RequireOwnership checks if the current user owns the resource.
func RequireOwnership(c *fiber.Ctx, resourceUserID int) (*user.User, error) {
	u, _ := CurrentUser(c)
	if u == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "no user in session")
	}
	if u.ID != resourceUserID {
		return nil, fiber.NewError(fiber.StatusForbidden, "not resource owner")
	}
	return u, nil
}
