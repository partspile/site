package handlers

import (
	"net/http"

	g "maragu.dev/gomponents"

	"github.com/parts-pile/site/components"
	"github.com/parts-pile/site/user"
)

func HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var currentUser *user.User
	currentUser, _ = GetCurrentUser(r)

	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = components.StyledLink("New Ad", "/new-ad", components.ButtonPrimary)
	} else {
		newAdButton = components.StyledLinkDisabled("New Ad", components.ButtonPrimary)
	}

	_ = components.Page(
		"Parts Pile - Auto Parts and Sales",
		currentUser,
		r.URL.Path,
		[]g.Node{
			components.SearchWidget(newAdButton),
			components.InitialSearchResults(),
		},
	).Render(w)
}
