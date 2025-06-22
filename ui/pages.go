package ui

import (
	g "maragu.dev/gomponents"

	"github.com/parts-pile/site/user"
)

func HomePage(currentUser *user.User, path string) g.Node {
	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = StyledLink("New Ad", "/new-ad", ButtonPrimary)
	} else {
		newAdButton = StyledLinkDisabled("New Ad", ButtonPrimary)
	}

	return Page(
		"Parts Pile - Auto Parts and Sales",
		currentUser,
		path,
		[]g.Node{
			SearchWidget(newAdButton),
			InitialSearchResults(),
		},
	)
}
