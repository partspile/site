package handler

import (
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ui"
)

func showError(c *fiber.Ctx, errMsg string) error {
	if wantsJSON(c) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": errMsg,
		})
	}
	return showErrorTo(c, "error", errMsg)
}

func showErrorTo(c *fiber.Ctx, elementID, errMsg string) error {
	return render(c, ui.ErrorDivWithID(elementID, errMsg))
}

// ErrorPageHandler displays an error page based on query parameters
func ErrorPageHandler(c *fiber.Ctx) error {
	code := c.QueryInt("code", fiber.StatusInternalServerError)
	message := c.Query("message", "Internal Server Error")

	// Decode URL-encoded message
	decodedMessage, err := url.QueryUnescape(message)
	if err == nil {
		message = decodedMessage
	}

	c.Status(code)
	title := fmt.Sprintf("%d - %s", code, message)
	return renderPage(c, title, ui.ErrorPageContent(code, message))
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Build error page URL with query parameters
	errorURL := fmt.Sprintf("/error?code=%d&message=%s", code, url.QueryEscape(message))

	// For HTMX requests, use HX-Redirect header to trigger full page redirect
	if c.Get("HX-Request") != "" {
		c.Set("HX-Redirect", errorURL)
		return c.SendStatus(code)
	}

	// For regular requests, use standard redirect with 302 Found status
	// (browsers require 3xx status codes for redirects, not error codes)
	return c.Redirect(errorURL, fiber.StatusFound)
}
