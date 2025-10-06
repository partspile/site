package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

// HandleAbout displays the About page
func HandleAbout(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.AboutPage(currentUser, c.Path()))
}
