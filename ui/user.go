package ui

import (
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

func BookmarkedAdsSection(currentUser *user.User, ads []ad.Ad) g.Node {
	return Div(
		Class("mt-8"),
		SectionHeader("Bookmarked Ads", "Ads you have bookmarked for later."),
		g.If(len(ads) == 0,
			P(Class("text-gray-500"), g.Text("No bookmarked ads yet.")),
		),
		g.If(len(ads) > 0,
			AdCompactListContainer(
				g.Group(BuildAdListNodesFromSlice(currentUser, ads)),
			),
		),
	)
}

func BuildAdListNodesFromSlice(currentUser *user.User, ads []ad.Ad) []g.Node {
	loc := time.Local
	nodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		nodes = append(nodes, AdListNode(ad, loc, currentUser.ID))
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
						SectionHeader("Notification Preferences", "Choose how you'd like to receive notifications."),
						FormContainer("notificationForm",
							NotificationMethodRadioGroup(currentUser.NotificationMethod, currentUser.EmailAddress, currentUser.Phone),
							ActionButtons(
								StyledButton("Update Preferences", ButtonPrimary,
									hx.Post("/api/update-notification-method"),
									hx.Target("#notificationPreferencesResults"),
									hx.Indicator("#notificationForm"),
								),
							),
						),
						Div(ID("notificationPreferencesResults"), Class("mt-2")),
						Div(Class("mt-12"),
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
