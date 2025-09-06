package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func HandleHome(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	view := getCookieLastView(c)
	return render(c, ui.HomePage(currentUser, c.Path(), view))
}

func HandleSettings(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}

// HandleTermsOfService displays the Terms of Service page
func HandleTermsOfService(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.TermsOfServicePage(currentUser, c.Path()))
}

// HandlePrivacyPolicy displays the Privacy Policy page
func HandlePrivacyPolicy(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.PrivacyPolicyPage(currentUser, c.Path()))
}
