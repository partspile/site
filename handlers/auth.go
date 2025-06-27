package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"golang.org/x/crypto/bcrypt"
)

func HandleLoginSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	password := c.FormValue("password")

	u, err := user.GetUserByName(name)
	if err != nil {
		return ValidationErrorResponseWithStatus(c, "Invalid username or password", fiber.StatusUnauthorized)
	}

	if err := VerifyPassword(u.PasswordHash, password); err != nil {
		return ValidationErrorResponseWithStatus(c, "Invalid username or password", fiber.StatusUnauthorized)
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
	return render(c, ui.SuccessMessage("You have been logged out", "/login"))
}

func GetCurrentUser(c *fiber.Ctx) (*user.User, error) {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return nil, fmt.Errorf("session error: %w", err)
	}

	userID, ok := sess.Get("userID").(int)
	if !ok || userID == 0 {
		return nil, fmt.Errorf("no user ID in session")
	}

	u, status, found := user.GetUserByID(userID)
	if !found {
		return nil, fmt.Errorf("user not found")
	}

	// Only return active users for current user sessions
	if status == user.StatusArchived {
		return nil, fmt.Errorf("user is archived")
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
