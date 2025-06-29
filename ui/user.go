package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

func UserNav(currentUser *user.User, currentPath string) g.Node {
	var navItems []g.Node
	if currentUser != nil {
		navItems = []g.Node{
			A(Href("/"), Class("text-xl font-bold"), g.Text("Parts Pile")),
			Span(Class("flex-grow")),
			Span(Class("font-semibold"), g.Text(currentUser.Name)),
			Span(Class("text-green-700 font-bold"), g.Text(fmt.Sprintf("%.2f tokens", currentUser.TokenBalance))),
		}
		if currentUser.IsAdmin {
			navItems = append(navItems,
				A(Href("/admin"), Class("text-blue-500 hover:underline"), g.Text("Admin")),
			)
		}
		if currentPath != "/settings" {
			navItems = append(navItems,
				A(Href("/settings"), Class("text-blue-500 hover:underline"), g.Text("Settings")),
			)
		}
		navItems = append(navItems,
			A(
				Class("text-blue-500 hover:underline"),
				Href("#"),
				hx.Post("/logout"),
				hx.Target("#result"),
				g.Text("Logout"),
			),
		)
	} else {
		navItems = []g.Node{
			A(Href("/"), Class("text-xl font-bold"), g.Text("Parts Pile")),
			Span(Class("flex-grow")),
		}
		if currentPath != "/login" {
			navItems = append(navItems,
				A(Href("/login"), Class("text-blue-500 hover:underline"), g.Text("Login")),
			)
		}
		if currentPath != "/register" {
			navItems = append(navItems,
				A(Href("/register"), Class("text-blue-500 hover:underline"), g.Text("Register")),
			)
		}
	}

	return Nav(
		Class("flex items-center space-x-4 w-full"),
		g.Group(navItems),
	)
}

func BookmarkedAdsSection(currentUser *user.User, ads []ad.Ad) g.Node {
	return Div(
		Class("mt-8"),
		SectionHeader("Bookmarked Ads", "Ads you have bookmarked for later."),
		g.If(len(ads) == 0,
			P(Class("text-gray-500"), g.Text("No bookmarked ads yet.")),
		),
		g.If(len(ads) > 0,
			Div(
				Class("space-y-4"),
				g.Group(BuildAdListNodesFromSlice(currentUser, ads, true)),
			),
		),
	)
}

func BuildAdListNodesFromSlice(currentUser *user.User, ads []ad.Ad, bookmarked bool) []g.Node {
	loc := time.Local
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	nodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		nodes = append(nodes, AdCardExpandable(ad, loc, bookmarked, userID))
	}
	return nodes
}

func SettingsPage(currentUser *user.User, currentPath string) g.Node {
	// Section navigation
	return Page(
		"Settings",
		currentUser,
		currentPath,
		[]g.Node{
			PageHeader("Settings"),
			Div(
				Class("flex space-x-4 mb-6"),
				A(Href("/settings"), Class("px-4 py-2 rounded "+sectionTabClass(currentPath, "/settings")), g.Text("General")),
				A(Href("/settings/bookmarked-ads"), hx.Get("/settings/bookmarked-ads"), hx.Target("#settings-section"), hx.PushURL("/settings/bookmarked-ads"), Class("px-4 py-2 rounded "+sectionTabClass(currentPath, "/settings/bookmarked-ads")), g.Text("Bookmarked Ads")),
			),
			Div(ID("settings-section"),
				g.If(currentPath == "/settings/bookmarked-ads",
					Div(g.Text("Loading bookmarked ads...")), // Will be replaced by HTMX
				),
				g.If(currentPath != "/settings/bookmarked-ads",
					ContentContainer(
						SectionHeader("Change Password", ""),
						FormContainer("changePasswordForm",
							FormGroup("Current Password", "currentPassword",
								PasswordInput("currentPassword", "currentPassword"),
							),
							FormGroup("New Password", "newPassword",
								PasswordInput("newPassword", "newPassword"),
							),
							FormGroup("Confirm New Password", "confirmNewPassword",
								PasswordInput("confirmNewPassword", "confirmNewPassword"),
							),
							ActionButtons(
								StyledButton("Change Password", ButtonPrimary,
									hx.Post("/api/change-password"),
									hx.Target("#result"),
									hx.Indicator("#changePasswordForm"),
								),
							),
						),
						Div(Class("mt-12"),
							SectionHeader("Delete Account", "This will permanently delete your account and all associated data. This action cannot be undone."),
							FormContainer("deleteAccountForm",
								FormGroup("Password", "deletePassword",
									PasswordInput("deletePassword", "password"),
								),
								ActionButtons(
									StyledButton("Delete My Account", ButtonDanger,
										hx.Post("/api/delete-account"),
										hx.Confirm("Are you sure you want to delete your account? This action is permanent."),
										hx.Target("#result"),
										hx.Indicator("#deleteAccountForm"),
									),
								),
							),
						),
						ResultContainer(),
					),
				),
			),
		},
	)
}

func sectionTabClass(currentPath, tabPath string) string {
	if currentPath == tabPath {
		return "bg-blue-600 text-white"
	}
	return "bg-gray-200 text-gray-700 hover:bg-gray-300"
}
