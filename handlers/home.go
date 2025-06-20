package handlers

import (
	"net/http"

	g "maragu.dev/gomponents"

	"github.com/parts-pile/site/templates"
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
		newAdButton = templates.StyledLink("New Ad", "/new-ad", templates.ButtonPrimary)
	} else {
		newAdButton = templates.StyledLinkDisabled("New Ad", templates.ButtonPrimary)
	}

	_ = templates.Page(
		"Parts Pile - Auto Parts and Sales",
		currentUser,
		r.URL.Path,
		[]g.Node{
			templates.SearchWidget(newAdButton),
			templates.InitialSearchResults(),
		},
	).Render(w)
}
