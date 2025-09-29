package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/grok"
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
	if err := logoutUser(c); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you out.")
	}
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
func AuthRequired(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err != nil {
		// For HTMX requests, return a redirect response that HTMX can handle
		if c.Get("HX-Request") != "" {
			c.Set("HX-Redirect", "/login")
			return c.Status(fiber.StatusSeeOther).SendString("")
		}
		// For regular requests, redirect to login page
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	// Store user in context for downstream handlers
	c.Locals("user", user)

	return c.Next()
}

// OptionalAuth is a middleware that checks for a user but does not require one.
func OptionalAuth(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err == nil {
		c.Locals("user", user)
	}
	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
func AdminRequired(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err != nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	if !user.IsAdmin {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	c.Locals("user", user)

	return c.Next()
}

func HandleRegister(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.RegisterPage(currentUser, c.Path()))
}

func HandleRegisterSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	phone := c.FormValue("phone")
	phone = strings.TrimSpace(phone)

	// Validate required fields
	if strings.TrimSpace(name) == "" {
		return ValidationErrorResponse(c, "Username is required.")
	}

	if phone == "" {
		return ValidationErrorResponse(c, "Phone number is required.")
	}

	// Validate required checkbox
	offers := c.FormValue("offers")

	if offers != "true" {
		return ValidationErrorResponse(c, "You must agree to receive informational text messages to continue.")
	}
	if strings.HasPrefix(phone, "+") {
		matched, _ := regexp.MatchString(`^\+[1-9][0-9]{7,14}$`, phone)
		if !matched {
			return ValidationErrorResponse(c, "Phone must be in international format, e.g. +12025550123")
		}
	} else {
		digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
		if len(digits) == 10 {
			phone = "+1" + digits
		} else {
			return ValidationErrorResponse(c, "US/Canada numbers must have 10 digits")
		}
	}
	userPassword := c.FormValue("password")
	password2 := c.FormValue("password2")

	if err := password.ValidatePasswordConfirmation(userPassword, password2); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Validate password strength
	if err := password.ValidatePasswordStrength(userPassword); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Check for existing username and phone before GROK screening
	if _, err := user.GetUserByName(name); err == nil {
		return ValidationErrorResponse(c, "Username already exists. Please choose a different username.")
	}

	if _, err := user.GetUserByPhone(phone); err == nil {
		return ValidationErrorResponse(c, "Phone number is already registered. Please use a different phone number.")
	}

	// GROK username screening
	systemPrompt := `You are an expert parts technician. Your job is to screen potential user names for the parts-pile web site.
Reject user names that the general public would find offensive.
Car-guy humor, double entendres, and puns are allowed unless they are widely considered offensive or hateful.
Examples of acceptable usernames:
- rusty nuts
- lugnut
- fast wrench
- shift happens

Examples of unacceptable usernames:
- racial slurs
- hate speech
- explicit sexual content

If the user name is acceptable, return only: OK
If the user name is unacceptable, return a short, direct error message (1-2 sentences), and do not mention yourself, AI, or Grok in the response.
Only reject names that are truly offensive to a general audience.`
	resp, err := grok.CallGrok(systemPrompt, name)
	if err != nil {
		return ValidationErrorResponse(c, "Could not validate username. Please try again later.")
	}
	if resp != "OK" {
		return ValidationErrorResponse(c, resp)
	}

	hash, salt, err := password.HashPassword(userPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to create your account.")
	}
	_, err = user.CreateUser(name, phone, hash, salt, "argon2id")
	if err != nil {
		// This should rarely happen since we checked above, but handle any other database errors
		return ValidationErrorResponse(c, "Unable to create account. Please try again.")
	}

	return render(c, ui.SuccessMessage("Registration successful", "/login"))
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

// CurrentUser extracts the user from context, or falls back to session if not present.
func CurrentUser(c *fiber.Ctx) (*user.User, error) {
	u, ok := c.Locals("user").(*user.User)
	if ok && u != nil {
		log.Printf("[DEBUG] CurrentUser returning user from context: userID=%d", u.ID)
		return u, nil
	}
	log.Printf("[DEBUG] CurrentUser no user in context, falling back to session")
	// Fallback to session-based extraction
	return GetCurrentUser(c)
}

// RequireAdmin extracts the user from context and checks admin status.
func RequireAdmin(c *fiber.Ctx) (*user.User, error) {
	u, err := CurrentUser(c)
	if err != nil {
		return nil, err
	}
	if !u.IsAdmin {
		return nil, fiber.NewError(fiber.StatusForbidden, "admin access required")
	}
	return u, nil
}

// RequireOwnership checks if the current user owns the resource.
func RequireOwnership(c *fiber.Ctx, resourceUserID int) (*user.User, error) {
	u, err := CurrentUser(c)
	if err != nil {
		return nil, err
	}
	if u.ID != resourceUserID {
		return nil, fiber.NewError(fiber.StatusForbidden, "not resource owner")
	}
	return u, nil
}

func HandleUserMenu(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, ui.UserMenuPopup(currentUser, c.Path()))
}
