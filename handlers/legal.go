package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
)

// HandleTermsOfService displays the Terms of Service page
func HandleTermsOfService(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	return render(c, ui.TermsOfServicePage(userID, userName, c.Path()))
}

// HandlePrivacyPolicy displays the Privacy Policy page
func HandlePrivacyPolicy(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	return render(c, ui.PrivacyPolicyPage(userID, userName, c.Path()))
}
