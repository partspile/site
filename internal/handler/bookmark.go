package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/bookmark"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/ui"
)

func BookmarkHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	csrfToken := local.GetCSRFToken(c)

	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	var bookmarked bool
	var handlerErr error

	switch c.Method() {
	case fiber.MethodPost:
		bookmarked = true
		handlerErr = bookmark.Add(userID, adID)
	case fiber.MethodDelete:
		bookmarked = false
		handlerErr = bookmark.Remove(userID, adID)
	default:
		return fiber.NewError(fiber.StatusMethodNotAllowed, "Method not allowed")
	}

	if handlerErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, handlerErr.Error())
	}

	return render(c, ui.BookmarkButton(adID, bookmarked, csrfToken))
}
