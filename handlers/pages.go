package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	g "maragu.dev/gomponents"
)

func HandleHome(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c) // this might return an error, but we ignore it, same as original.

	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	} else {
		newAdButton = ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
	}

	return render(c, ui.Page(
		"Parts Pile - Auto Parts and Sales",
		currentUser,
		c.Path(),
		[]g.Node{
			ui.SearchWidget(newAdButton),
			ui.InitialSearchResults(),
		},
	))
}

func HandleSettings(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, ui.SettingsPage(currentUser, c.Path()))
}
