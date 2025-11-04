package handlers

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/jwt"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

// regenerateSession regenerates the session to invalidate old CSRF tokens
// This MUST be called when authorization status changes (login/logout)
func regenerateSession(c *fiber.Ctx) error {
	sess, err := sessionStore.Get(c)
	if err != nil {
		return err
	}
	// Regenerate session - this will delete the old CSRF token and create a new session
	return sess.Regenerate()
}

func logoutUser(c *fiber.Ctx) error {
	// Regenerate session to invalidate CSRF token (per Fiber CSRF docs)
	if err := regenerateSession(c); err != nil {
		log.Printf("[AUTH] Error regenerating session on logout: %v", err)
	}
	cookie.ClearJWT(c)
	return nil
}

// loginUser logs in a user by generating a JWT and setting it in the cookie
func loginUser(c *fiber.Ctx, u *user.User) error {
	token, err := jwt.GenerateToken(u)
	if err != nil {
		log.Printf("[AUTH] JWT generation error: %v", err)
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

	// Regenerate session to invalidate old CSRF token and create new one (per Fiber CSRF docs)
	if err := regenerateSession(c); err != nil {
		log.Printf("[AUTH] Error regenerating session on login: %v", err)
		// Continue with login even if session regeneration fails
	}

	// Generate JWT token and log the user in
	if err := loginUser(c, &u); err != nil {
		log.Printf("[AUTH] Login failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	log.Printf("[AUTH] Login successful: userID=%d, name=%s", u.ID, name)
	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	return render(c, ui.SuccessMessage("Login successful", "/", redirectDelayMs))
}
