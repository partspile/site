package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	view := getCookieLastView(c)
	return render(c, ui.HomePage(currentUser, c.Path(), view))
}
