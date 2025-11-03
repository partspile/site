package handlers

import (
	"errors"
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

// CustomErrorHandler handles application errors with user context
// Sanitizes error messages to prevent information disclosure
func CustomErrorHandler(c *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Get user context using handler functions
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)

	// Log detailed error information server-side only
	log.Printf("[ERROR] %s %s - Status: %d, Error: %v, UserID: %d", c.Method(), c.Path(), code, err, userID)
	if code >= 500 {
		// For server errors, log stack trace server-side
		log.Printf("[ERROR] Stack trace: %s", string(debug.Stack()))
	}

	// Sanitize error message for client
	// Provide generic user-friendly messages instead of internal error details
	var userMessage string
	switch {
	case code >= 500:
		// Internal server errors - never expose details to users
		userMessage = "An internal server error occurred. Please try again later."
	case code == 404:
		// Not found errors
		userMessage = "The requested resource was not found."
	case code == 403:
		// Forbidden errors
		userMessage = "You do not have permission to access this resource."
	case code == 401:
		// Unauthorized errors
		userMessage = "Authentication required."
	case code >= 400:
		// Other client errors - use error message if it's from fiber.Error (controlled by us)
		if errors.As(err, &e) && e.Message != "" {
			// Only use fiber.Error messages (which are intentionally user-facing)
			userMessage = e.Message
		} else {
			userMessage = "Invalid request. Please check your input and try again."
		}
	default:
		userMessage = "An error occurred. Please try again."
	}

	// Send sanitized error page
	c.Status(code)
	return render(c, ui.ErrorPage(code, userMessage, userID, userName))
}
