package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

// HandleAbout displays the About page
func HandleAbout(c *fiber.Ctx) error {
	userID := getUserID(c)
	userName := getUserName(c)
	return render(c, ui.AboutPage(userID, userName, c.Path()))
}
