package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

// HandleTermsOfService displays the Terms of Service page
func HandleTermsOfService(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.TermsOfServicePage(currentUser, c.Path()))
}

// HandlePrivacyPolicy displays the Privacy Policy page
func HandlePrivacyPolicy(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	return render(c, ui.PrivacyPolicyPage(currentUser, c.Path()))
}
