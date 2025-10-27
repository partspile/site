package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

func HandleAbout(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	return render(c, ui.AboutPage(userID, userName, c.Path()))
}
