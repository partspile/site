package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/jwt"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

func logoutUser(c *fiber.Ctx) error {
	cookie.ClearJWT(c)
	return nil
}

// loginUser logs in a user by generating a JWT and setting it in the cookie
func loginUser(c *fiber.Ctx, u *user.User) error {
	token, err := jwt.GenerateToken(u)
	if err != nil {
		logger.Error("JWT generation error",
			"error", err, "userID", u.ID)
		return fmt.Errorf("failed to generate JWT: %w", err)
	}

	cookie.SetJWT(c, token)
	return nil
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

func HandleLogin(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.LoginPage(userID, userName, c.Path(), csrfToken))
}

func HandleLogout(c *fiber.Ctx) error {
	if err := logoutUser(c); err != nil {
		return err
	}
	return redirectToLogin(c)
}

func HandleLoginSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	userPassword := c.FormValue("password")

	logger.Info("Login attempt", "name", name)

	u, err := user.GetUserByName(name)
	userExists := err == nil

	// Always perform password verification to prevent timing attacks
	// If user doesn't exist, use a dummy hash/salt to maintain constant-time behavior
	// The dummy hash/salt are valid base64 strings that will cause verification to fail
	// but still perform the full Argon2id computation, preventing timing differences
	var passwordValid bool
	if userExists {
		passwordValid = password.VerifyPassword(userPassword, u.PasswordHash, u.PasswordSalt)
	} else {
		// Use dummy hash/salt with valid base64 encoding to ensure full Argon2id computation
		// This prevents timing attacks that could reveal user existence
		// The hash and salt are 32 and 16 bytes respectively (base64 RawStdEncoding, no padding)
		// They are intentionally invalid to ensure verification always fails
		dummyHash := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 32 bytes = 43 chars base64 RawStdEncoding
		dummySalt := "AAAAAAAAAAAAAAAAAAAAAA"                  // 16 bytes = 22 chars base64 RawStdEncoding
		password.VerifyPassword(userPassword, dummyHash, dummySalt)
		passwordValid = false
	}

	if !userExists || !passwordValid {
		logger.Warn("Login failed",
			"name", name, "userExists", userExists,
			"passwordValid", passwordValid)
		return ValidationErrorResponse(c, "Invalid username or password")
	}

	// Generate JWT token and log the user in
	if err := loginUser(c, &u); err != nil {
		logger.Error("Login failed", "error", err, "name", name)
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	logger.Info("Login successful", "userID", u.ID, "name", name)
	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	return render(c, ui.SuccessMessage("Login successful", "/", redirectDelayMs))
}
