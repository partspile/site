package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func HandleSettings(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}

func HandleUserMenu(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.UserMenuPopup(currentUser, c.Path()))
}
