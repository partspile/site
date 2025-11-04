package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	view := cookie.GetView(c)
	adCategory := cookie.GetAdCategory(c)
	params := extractSearchParams(c)
	path := c.Path()
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.HomePage(userID, userName, csrfToken, view, adCategory, path, params))
}
