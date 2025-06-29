package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	view := c.Cookies("last_view", "list") // default to list
	return render(c, ui.HomePage(currentUser, c.Path(), view))
}

func HandleSettings(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}
