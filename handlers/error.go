package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

// CustomErrorHandler handles application errors with user context
func CustomErrorHandler(ctx *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Get user context using handler functions
	userID := getUserID(ctx)
	userName := getUserName(ctx)

	// Send custom error page
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return ui.ErrorPage(code, err.Error(), userID, userName).Render(ctx)
}
