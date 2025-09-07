package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func HandleSettings(c *fiber.Ctx) error {
	currentUser, _ := getUser(c)
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}
