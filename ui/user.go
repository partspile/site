package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

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

func SettingsPage(currentUser *user.User, currentPath string) g.Node {
	return Page(
		"Settings",
		currentUser,
		currentPath,
		[]g.Node{
			PageHeader("Settings"),
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
		},
	)
}
