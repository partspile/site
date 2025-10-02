package ui

import (
	"strings"

	"github.com/parts-pile/site/user"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func getUserInitial(currentUser *user.User) string {
	return strings.ToUpper(string([]rune(currentUser.Name)[0]))
}

func indicator() g.Node {
	return Div(
		ID("indicator"),
		Class("htmx-indicator flex items-center gap-2 text-blue-600"),
		Div(
			Class("w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"),
		),
		g.Text("Loading..."),
	)
}

func navLoggedIn(currentUser *user.User) g.Node {
	return Div(
		Span(
			Class("bg-red-500 text-white rounded-full w-8 h-8 flex items-center justify-center font-semibold text-sm cursor-pointer hover:bg-red-600"),
			hx.Get("/user-menu"),
			hx.Target("body"),
			hx.Swap("beforeend"),
			hx.Headers(`js:{'X-Requested-With': 'XMLHttpRequest'}`),
			g.Text(getUserInitial(currentUser)),
		),
	)
}

func loginNode() g.Node {
	return A(Href("/login"), Class("text-blue-500 hover:underline"), g.Text("Login"))
}

func registerNode() g.Node {
	return A(Href("/register"), Class("text-blue-500 hover:underline"), g.Text("Register"))
}

func navLoggedOut(currentPath string) g.Node {
	switch currentPath {
	case "/login":
		return registerNode()
	case "/register", "/rocks":
		return loginNode()
	case "/register/verify":
		return nil
	default:
		return Div(
			Class("flex items-center space-x-4"),
			loginNode(),
			registerNode(),
		)
	}
}

func navigation(currentUser *user.User, currentPath string) g.Node {
	return Nav(
		Class("mb-8 border-b pb-4 flex items-center justify-between w-full"),
		A(Href("/"), Class("text-xl font-bold"), g.Text("Parts Pile")),
		indicator(),
		g.Iff(currentUser != nil, func() g.Node { return navLoggedIn(currentUser) }),
		g.Iff(currentUser == nil, func() g.Node { return navLoggedOut(currentPath) }),
	)
}

func menuHeader(currentUser *user.User) g.Node {
	return Div(
		Class("px-4 py-3 border-b border-gray-100 text-center"),
		Div(
			Class("w-12 h-12 bg-red-500 text-white rounded-full flex items-center justify-center font-semibold text-lg mx-auto mb-2"),
			g.Text(getUserInitial(currentUser)),
		),
		Div(
			Class("text-sm font-medium text-gray-900"),
			g.Text(currentUser.Name),
		),
		Div(
			Class("text-xs text-gray-500"),
			g.Text("Logged in"),
		),
	)
}

func UserMenuPopup(currentUser *user.User, currentPath string) g.Node {
	var menuItems []g.Node

	if currentUser.IsAdmin {
		menuItems = append(menuItems,
			A(
				Href("/admin"),
				Class("block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"),
				g.Text("🛠️ Admin"),
			),
		)
	}

	menuItems = append(menuItems,
		A(
			Href("/bookmarks"),
			Class("block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 flex items-center"),
			Img(
				Src("/images/bookmark-true.svg"),
				Alt("Bookmarks"),
				Class("w-4 h-4 mr-2"),
			),
			g.Text("Bookmarks"),
		),
	)

	menuItems = append(menuItems,
		A(
			Href("/messages"),
			Class("block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 flex items-center"),
			Img(
				Src("/images/message.svg"),
				Alt("Messages"),
				Class("w-4 h-4 mr-2"),
			),
			g.Text("Messages"),
		),
	)

	menuItems = append(menuItems,
		A(
			Href("/settings"),
			Class("block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 flex items-center"),
			Img(
				Src("/images/settings.svg"),
				Alt("Settings"),
				Class("w-4 h-4 mr-2"),
			),
			g.Text("Settings"),
		),
	)

	menuItems = append(menuItems,
		A(
			Href("#"),
			Class("block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 flex items-center"),
			hx.Post("/logout"),
			hx.Target("body"),
			hx.Swap("outerHTML"),
			hx.Indicator("#htmx-indicator"),
			Img(
				Src("/images/logout.svg"),
				Alt("Logout"),
				Class("w-4 h-4 mr-2"),
			),
			g.Text("Logout"),
		),
	)

	return Div(
		ID("user-menu-popup"),
		Class("fixed inset-0 bg-black bg-opacity-30 z-50"),
		g.Attr("onclick", "this.remove()"),
		Div(
			Class("fixed top-16 right-4 pointer-events-none"),
			Div(
				Class("bg-white rounded-lg shadow-lg border border-gray-200 w-40 pointer-events-auto"),
				menuHeader(currentUser),
				Div(
					Class("py-1"),
					g.Group(menuItems),
				),
			),
		),
	)
}
