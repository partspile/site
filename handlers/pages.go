package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

func HandleHome(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c) // this might return an error, but we ignore it, same as original.
	return render(c, ui.HomePage(currentUser, c.Path()))
}

func HandleSettings(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}
