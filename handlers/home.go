package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)
	view := cookie.GetLastView(c)
	adCategory := cookie.GetAdCategory(c)
	return render(c, ui.HomePage(userID, userName, c.Path(), view, adCategory))
}
