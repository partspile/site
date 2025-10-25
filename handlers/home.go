package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)
	view := cookie.GetView(c)
	adCategory := cookie.GetAdCategory(c)
	params := extractSearchParams(c)
	path := c.Path()
	return render(c, ui.HomePage(userID, view, adCategory, userName, path, params))
}
