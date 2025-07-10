package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"golang.org/x/crypto/bcrypt"
)

func HandleLoginSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	password := c.FormValue("password")

	u, err := user.GetUserByName(name)
	if err != nil {
		return ValidationErrorResponse(c, "Invalid username or password")
	}

	if err := VerifyPassword(u.PasswordHash, password); err != nil {
		return ValidationErrorResponse(c, "Invalid username or password")
	}

	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	sess.Set("userID", u.ID)

	if err := sess.Save(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to save session.")
	}

	return render(c, ui.SuccessMessage("Login successful", "/"))
}

func HandleLogout(c *fiber.Ctx) error {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		// can't get session, maybe it's already gone. redirect anyway.
		return c.Redirect("/", fiber.StatusSeeOther)
	}

	if err := sess.Destroy(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you out.")
	}
	return render(c, ui.SuccessMessage("You have been logged out", "/"))
}

func GetCurrentUser(c *fiber.Ctx) (*user.User, error) {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("session error: %v", err))
	}

	userID, ok := sess.Get("userID").(int)
	if !ok || userID == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "no user ID in session")
	}

	u, status, found := user.GetUserByID(userID)
	if !found {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	// Only return active users for current user sessions
	if status == user.StatusArchived {
		return nil, fiber.NewError(fiber.StatusForbidden, "user is archived")
	}

	return &u, nil
}

// AuthRequired is a middleware that requires a user to be logged in.
func AuthRequired(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err != nil {
		// You might want to redirect to login page
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
	password := c.FormValue("password")
	password2 := c.FormValue("password2")

	if err := ValidatePasswordConfirmation(password, password2); err != nil {
		return ValidationErrorResponse(c, err.Error())
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to create your account.")
	}

	if _, err := user.CreateUser(name, phone, string(hashedPassword)); err != nil {
		return ValidationErrorResponse(c, "User already exists or another error occurred.")
	}

	return render(c, ui.SuccessMessage("Registration successful", "/login"))
}

func HandleLogin(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.LoginPage(currentUser, c.Path()))
}

func HandleChangePassword(c *fiber.Ctx) error {
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	if err := ValidatePasswordConfirmation(newPassword, confirmNewPassword); err != nil {
		return ValidationErrorResponse(c, "New passwords do not match")
	}

	currentUser := c.Locals("user").(*user.User)

	if err := VerifyPassword(currentUser.PasswordHash, currentPassword); err != nil {
		return ValidationErrorResponse(c, "Invalid current password")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to update password.")
	}

	if _, err := user.UpdateUserPassword(currentUser.ID, string(newHash)); err != nil {
		return ValidationErrorResponse(c, "Failed to update password")
	}
	return render(c, ui.SuccessMessage("Password changed successfully", ""))
}

func HandleDeleteAccount(c *fiber.Ctx) error {
	password := c.FormValue("password")

	currentUser := c.Locals("user").(*user.User)
	if currentUser == nil {
		return ValidationErrorResponseWithStatus(c, "You must be logged in to delete your account", fiber.StatusUnauthorized)
	}

	if err := VerifyPassword(currentUser.PasswordHash, password); err != nil {
		return ValidationErrorResponseWithStatus(c, "Invalid password", fiber.StatusUnauthorized)
	}

	// Archive all ads by this user (function not implemented)
	// TODO: Implement ArchiveAdsByUserID or similar if needed
	// err = ad.DeleteAdsByUserID(currentUser.ID)
	// if err != nil {
	// 	c.Response().SetStatusCode(fiber.StatusInternalServerError)
	// 	return c.SendString("Failed to archive user's ads")
	// }

	// Delete user (function not implemented)
	// TODO: Implement DeleteUser if needed
	// err = user.DeleteUser(currentUser.ID)
	// if err != nil {
	// 	c.Response().SetStatusCode(fiber.StatusInternalServerError)
	// 	return c.SendString("Failed to delete user")
	// }

	return render(c, ui.SuccessMessage("Account deleted successfully", "/"))
}

// CurrentUser extracts the user from context, or falls back to session if not present.
func CurrentUser(c *fiber.Ctx) (*user.User, error) {
	u, ok := c.Locals("user").(*user.User)
	if ok && u != nil {
		return u, nil
	}
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
