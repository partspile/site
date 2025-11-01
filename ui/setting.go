package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func SettingsPage(userID int, userName string, currentPath string, notificationMethod string, emailAddress *string, phoneNumber string, smsOptedOut bool) g.Node {
	return Page(
		"Settings",
		userID,
		userName,
		currentPath,
		[]g.Node{
			pageHeader("Settings"),
			contentContainer(
				sectionHeader("Notification Preferences", "Choose how you'd like to receive notifications."),
				formContainer("notificationForm",
					Div(ID("notificationMethodGroup"),
						NotificationMethodRadioGroup(notificationMethod, emailAddress, phoneNumber, smsOptedOut),
					),
					actionButtons(
						button("Update Preferences",
							withAttributes(
								hx.Post("/api/update-notification-method"),
								hx.Target("#notificationPreferencesResults"),
								hx.Indicator("#notificationForm"),
							),
						),
					),
				),
				Div(ID("notificationPreferencesResults"), Class("mt-2")),
				Div(Class("mt-12"),
					sectionHeader("Change Password", ""),
					formContainer("changePasswordForm",
						formGroup("Current Password", "currentPassword",
							passwordInput("currentPassword", "currentPassword"),
						),
						formGroup("New Password", "newPassword",
							passwordInput("newPassword", "newPassword"),
						),
						formGroup("Confirm New Password", "confirmNewPassword",
							passwordInput("confirmNewPassword", "confirmNewPassword"),
						),
						actionButtons(
							button("Change Password",
								withAttributes(
									hx.Post("/api/change-password"),
									hx.Target("#result"),
									hx.Indicator("#changePasswordForm"),
								),
							),
						),
					),
				),
				Div(Class("mt-12"),
					sectionHeader("Delete Account", "This will permanently delete your account and all associated data. This action cannot be undone."),
					formContainer("deleteAccountForm",
						formGroup("Password", "deletePassword",
							passwordInput("deletePassword", "password"),
						),
						actionButtons(
							buttonDanger("Delete My Account",
								withAttributes(
									hx.Post("/api/delete-account"),
									hx.Confirm("Are you sure you want to delete your account? This action is permanent."),
									hx.Target("#result"),
									hx.Indicator("#deleteAccountForm"),
								),
							),
						),
					),
				),
				resultContainer(),
			),
		},
	)
}
