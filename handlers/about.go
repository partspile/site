package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

func HandleAbout(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.AboutPage(userID, userName, c.Path(), csrfToken))
}
