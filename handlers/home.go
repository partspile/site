package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	u := getUser(c)
	view := getCookieLastView(c)
	adCategory := getCookieAdCategory(c)
	return render(c, ui.HomePage(u, c.Path(), view, adCategory))
}
