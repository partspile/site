package ui

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/phoneformat"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// NavigationIcon creates an icon for navigation menu items
func menuIcon(iconSrc, alt string) g.Node {
	return Img(
		Src(iconSrc),
		Alt(alt),
		Class("w-6 h-6 mr-2"),
	)
}

func menuHeader(userID int, userName, memberSince string, rockCount int,
	userRockIcons g.Node) g.Node {
	return Div(
		Class("px-4 py-3 border-b border-zinc-100 text-center"),
		Div(
			Class("w-12 h-12 bg-red-500 text-white rounded-full flex items-center justify-center font-semibold text-lg mx-auto mb-2"),
			g.Text(getUserInitial(userName)),
		),
		Div(
			Class("flex items-center justify-center gap-2 mb-1"),
			Div(
				Class("text-sm font-medium text-zinc-900 flex items-center gap-1"),
				UserNameLink(userID, userName, userRockIcons),
			),
			RockCountBadge(rockCount),
		),
		Div(
			Class("text-[10px] text-zinc-500"),
			g.Text("Member since "+memberSince),
		),
	)
}

// UserNameLink renders a link to the user's profile with a hover popup summary
func UserNameLink(userID int, userName string, extra ...g.Node) g.Node {
	href := fmt.Sprintf("/auth/user/%d", userID)
	summaryURL := fmt.Sprintf("/auth/user/%d/summary", userID)
	popupID := fmt.Sprintf("user-summary-%d", userID)
	return Span(
		Class("relative group inline-flex items-center gap-1"),
		A(
			Href(href),
			Class("text-blue-600 dark:text-blue-400 hover:underline"),
			hx.Get(summaryURL),
			hx.Trigger("mouseenter delay:200ms once"),
			hx.Target("next .user-summary-popup"),
			hx.Swap("innerHTML"),
			g.Group(extra),
			g.Text(userName),
		),
		Span(
			Class("user-summary-popup absolute left-1/2 -translate-x-1/2 top-full mt-1 hidden group-hover:block z-50 w-64 p-3 bg-white dark:bg-zinc-800 rounded shadow-lg border border-zinc-200 dark:border-zinc-700 text-sm text-zinc-700 dark:text-zinc-300"),
			ID(popupID),
			g.Text("Loading…"),
		),
	)
}

func userProfileStats(d UserProfileData, memberSinceClass,
	statClass string) []g.Node {
	nodes := []g.Node{
		Div(Class(memberSinceClass), g.Text("Member since "+d.MemberSince)),
		Div(Class(statClass),
			g.Text(fmt.Sprintf("%d active ad(s)", d.ActiveAdCount)),
		),
	}
	if d.UserRockCount > 0 {
		nodes = append(nodes,
			Div(Class(statClass),
				g.Text(fmt.Sprintf("%d rock(s)", d.UserRockCount)),
			),
		)
	}
	return nodes
}

// UserSummaryFragment is the popup content loaded on hover
func UserSummaryFragment(d UserProfileData) g.Node {
	statClass := "text-xs text-zinc-500"
	return Div(
		Class("space-y-1"),
		Div(Class("font-semibold text-zinc-900 dark:text-zinc-100"), g.Text(d.Name)),
		g.Group(userProfileStats(d, statClass, statClass)),
	)
}

// UserProfilePage renders the user profile page body
func UserProfilePage(d UserProfileData, view int, adNodes []g.Node) []g.Node {
	title := pageTitle(d.Name)
	if d.ShowShare {
		title = Div(
			Class("flex items-center justify-between gap-4"),
			pageTitle(d.Name),
			shareProfileButton(),
		)
	}
	return []g.Node{
		title,
		Div(
			Class("mt-8 space-y-4 max-w-md"),
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("w-16 h-16 bg-red-500 text-white rounded-full flex items-center justify-center font-semibold text-2xl"),
					g.Text(getUserInitial(d.Name)),
				),
				Div(
					Class("text-zinc-600 dark:text-zinc-400"),
					g.Group(userProfileStats(d, "text-sm", "text-sm mt-1")),
				),
			),
		),
		UserAccountPictureBanner(d),
		UserProfileAds(d.AdsViewPathPrefix, view, adNodes),
	}
}

func shareProfileButton() g.Node {
	return iconButton(buttonProps{
		ImageSrc: "/images/share.svg",
		Alt:      "Share profile",
		Class:    "dark:invert dark:opacity-80",
		Attrs: []g.Node{
			hx.Get("/auth/user/share"),
			hx.Target("body"),
			hx.Swap("beforeend"),
		},
	})
}

// UserAccountPictureBanner shows the optional full-width account picture.
func UserAccountPictureBanner(d UserProfileData) g.Node {
	if !d.HasAccountPicture {
		return nil
	}
	src := UserAccountPictureSrc(d.ID)
	if src == "" {
		return nil
	}
	img := Img(
		Src(src),
		Alt(d.Name+" account picture"),
		Class("w-full max-h-[300px] object-cover"),
	)
	if d.AccountPictureURL == "" {
		return Div(Class("mt-8"), img)
	}
	return Div(
		Class("mt-8"),
		A(
			Href(d.AccountPictureURL),
			Target("_blank"),
			Rel("noopener noreferrer"),
			img,
		),
	)
}

// UserProfileAds is the active-ads list with grid/list toggles.
func UserProfileAds(pathPrefix string, view int, adNodes []g.Node) g.Node {
	return Div(
		ID("user-profile-ads"),
		Class("space-y-4 mt-8"),
		Div(
			Class("flex items-center justify-between gap-4"),
			H2(
				Class("text-lg font-semibold text-zinc-900 dark:text-zinc-100"),
				g.Text("Active Ads"),
			),
			viewTogglesTargeting(view, pathPrefix, "#user-profile-ads"),
		),
		AdsContent(view, adNodes),
	)
}

func UserShareModal(path string) g.Node {
	return g.Group([]g.Node{
		modalBackdrop("user-share"),
		Div(
			ID("user-share-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-md shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto"),
				Div(
					Class("flex items-center justify-between p-6 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(Class("text-xl font-bold text-zinc-900 dark:text-zinc-200"),
						g.Text("Share Profile")),
					modalClose("user-share"),
				),
				Div(
					Class("p-6 flex flex-col gap-4"),
					Div(
						Class("flex flex-col gap-2"),
						Label(
							Class("text-sm font-medium text-zinc-700 dark:text-zinc-300"),
							For("user-link-input"),
							g.Text("Profile Link"),
						),
						Div(
							Class("flex items-center gap-2"),
							Input(
								ID("user-link-input"),
								Type("text"),
								Value(path),
								g.Attr("readonly", ""),
								g.Attr("onfocus", "this.select();"),
								Class("flex-1 px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-zinc-50 dark:bg-zinc-700 text-sm text-zinc-900 dark:text-zinc-200"),
							),
							copyButton(path, false),
						),
					),
					P(
						Class("text-sm text-zinc-600 dark:text-zinc-400"),
						g.Text("Anyone with this link can see your "+
							"public profile and active ads."),
					),
				),
			),
		),
	})
}

func UserMenu(userName, memberSince string, userID int, isAdmin bool,
	hasUnread bool, rockCount int, userRockCount int) g.Node {
	var menuItems []g.Node

	if isAdmin {
		menuItems = append(menuItems,
			A(
				Href("/admin/dashboard"),
				Class("block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-50 flex items-center"),
				menuIcon("/images/tools.svg", "Admin"),
				g.Text("Admin"),
			),
		)
	}

	menuItems = append(menuItems,
		A(
			Href("/auth/user/myads"),
			Class("block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-50 flex items-center"),
			menuIcon("/images/bookmark-true.svg", "My Ads"),
			g.Text("My Ads"),
		),
		A(
			Href("/auth/user/messages"),
			Class("block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-50 flex items-center justify-between relative"),
			Div(
				Class("flex items-center"),
				menuIcon("/images/message.svg", "Messages"),
				g.Text("Messages"),
			),
			Div(
				ID("message-unread-indicator"),
				g.If(hasUnread, hasUnreadIndicator()),
			),
		),
		A(
			Href("/auth/user/settings"),
			Class("block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-50 flex items-center"),
			menuIcon("/images/settings.svg", "Settings"),
			g.Text("Settings"),
		),
		A(
			Href("/about"),
			Class("block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-50 flex items-center"),
			menuIcon("/images/info.svg", "About"),
			g.Text("About"),
		),
		A(
			Href("/logout"),
			Class("block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-50 flex items-center"),
			menuIcon("/images/logout.svg", "Logout"),
			g.Text("Logout"),
		),
	)

	return g.Group([]g.Node{
		modalBackdrop("user-menu"),
		Div(
			ID("user-menu-modal"),
			Class("fixed inset-x-0 top-16 z-50 pointer-events-none"),
			Div(
				Class("w-full md:max-w-3xl md:mx-auto px-6 flex justify-end"),
				Div(
					Class("pointer-events-auto bg-white rounded-lg shadow-lg border border-zinc-200 w-40"),
					menuHeader(userID, userName, memberSince, rockCount, UserRockIcons(userID, userRockCount)),
					Div(
						Class("py-1"),
						g.Group(menuItems),
					),
				),
			),
		),
	})
}

func MyAdsPage(activeTab string, view int, adNodes []g.Node) []g.Node {
	return []g.Node{
		Div(
			Class("flex items-center justify-between gap-4"),
			pageTitle("My Ads"),
			standardButton(buttonProps{
				Href: "/auth/ad/new",
				Text: "New Ad",
			}),
		),
		MyAdsContainer(activeTab, view, adNodes),
	}
}

func MyAdsContainer(activeTab string, view int, adNodes []g.Node) g.Node {
	pathPrefix := fmt.Sprintf(
		"/auth/user/myads/tab/%s/view/", activeTab)
	return Div(
		ID("my-ads-container"),
		Class("space-y-4 mt-6"),
		MyAdsTabs(activeTab),
		adsViewToggles(view, pathPrefix, "#my-ads-container"),
		AdsContent(view, adNodes),
	)
}

func MyAdsTabs(activeTab string) g.Node {
	return Div(
		ID("my-ads-tabs"),
		Class("border-b border-zinc-200 dark:border-zinc-700"),
		Div(
			Class("flex space-x-8"),
			myAdsTab("Bookmarked", "bookmarked", activeTab == "bookmarked"),
			myAdsTab("Active", "active", activeTab == "active"),
			myAdsTab("Paused", "inactive", activeTab == "inactive"),
			myAdsTab("Deleted", "deleted", activeTab == "deleted"),
		),
	)
}

func myAdsTab(name, tabID string, active bool) g.Node {
	var classes string
	if active {
		classes = "border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 py-4 px-1 text-sm font-medium"
	} else {
		classes = "border-b-2 border-transparent text-zinc-500 hover:text-zinc-700 hover:border-zinc-300 dark:text-zinc-400 dark:hover:text-zinc-300 py-4 px-1 text-sm font-medium"
	}

	href := fmt.Sprintf("/auth/user/myads/tab/%s", tabID)

	return A(
		Href(href),
		hx.Get(href),
		hx.Target("#my-ads-container"),
		hx.Swap("outerHTML"),
		Class(classes),
		g.Text(name),
	)
}

func AdsContent(view int, adNodes []g.Node) g.Node {
	var content g.Node
	if len(adNodes) == 0 {
		content = Div(
			Class("col-span-full text-center text-zinc-600 "+
				"dark:text-zinc-400 py-8"),
			P(g.Text("No ads found.")),
		)
	} else {
		content = g.Group(adNodes)
	}
	return Div(
		Class(resultsContainerClass(view)),
		content,
	)
}

const (
	SettingsChangePasswordErrorID = "change-password-error"
	SettingsChangePhoneErrorID    = "change-phone-error"
	SettingsDeleteAccountErrorID  = "delete-account-error"
	SettingsAccountPictureErrorID = "account-picture-error"
)

func settingsFormActions(button buttonProps, errorID string) g.Node {
	return Div(
		Class("flex items-center gap-4"),
		standardButton(button),
		ErrorDivWithID(errorID, ""),
	)
}

func settingsSection(title string, children ...g.Node) g.Node {
	nodes := []g.Node{
		H2(Class("text-xl font-semibold mb-4"), g.Text(title)),
	}
	nodes = append(nodes, children...)
	return Div(
		Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
		g.Group(nodes),
	)
}

func settingsAccountRow(labelText string, value g.Node) g.Node {
	return Div(
		Class("flex items-center gap-6"),
		Span(
			Class("w-24 shrink-0 text-sm font-medium text-zinc-500 dark:text-zinc-400"),
			g.Text(labelText),
		),
		value,
	)
}

func settingsUserAvatar(userName string) g.Node {
	return Span(
		Class("bg-red-500 text-white rounded-full w-8 h-8 flex items-center justify-center font-semibold text-sm shrink-0"),
		g.Text(getUserInitial(userName)),
	)
}

func settingsAccountSection(userName, phoneE64 string) g.Node {
	return Div(
		ID("settings-account"),
		Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
		H2(Class("text-xl font-semibold mb-4"), g.Text("Account")),
		Div(
			Class("space-y-4"),
			settingsAccountRow("Username",
				Div(
					Class("flex items-center gap-2 min-w-0"),
					settingsUserAvatar(userName),
					Span(
						Class("text-zinc-900 dark:text-zinc-100 font-medium"),
						g.Text(userName),
					),
				),
			),
			settingsAccountRow("Phone",
				Span(
					Class("text-zinc-900 dark:text-zinc-100"),
					g.Text(phoneformat.Display(phoneE64)),
				),
			),
		),
	)
}

func NotificationsSection(smsOptedOut bool) g.Node {
	return notificationsSection(smsOptedOut, false)
}

func NotificationsSectionSwapOOB(smsOptedOut bool) g.Node {
	return notificationsSection(smsOptedOut, true)
}

func notificationsSection(smsOptedOut bool, oob bool) g.Node {
	enabled := !smsOptedOut
	statusClass := "text-green-600 dark:text-green-400"
	statusText := "ON"
	if smsOptedOut {
		statusClass = "text-red-600 dark:text-red-400"
		statusText = "OFF"
	}

	attrs := []g.Node{
		ID("sms-notifications"),
		Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
	}
	if oob {
		attrs = append(attrs, hx.SwapOOB("outerHTML"))
	}

	return Div(
		append(attrs,
			H2(Class("text-xl font-semibold mb-4"), g.Text("Notifications")),
			Div(
				Class("flex items-center justify-between gap-4"),
				Div(
					Class("flex items-center gap-3"),
					Label(
						Class("relative inline-flex items-center cursor-pointer"),
						Input(
							Type("checkbox"),
							Name("enabled"),
							Value("true"),
							Class("sr-only peer"),
							g.If(enabled, Checked()),
							hx.Post("/auth/user/settings/notifications"),
							hx.Target("#sms-notifications"),
							hx.Swap("outerHTML"),
							hx.Trigger("change"),
						),
						Span(
							Class("w-11 h-6 bg-zinc-300 dark:bg-zinc-600 rounded-full peer peer-checked:bg-green-500 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all"),
						),
					),
					Span(Class("text-sm font-medium"), g.Text("Text messages")),
				),
				Span(
					Class("text-sm font-semibold "+statusClass),
					g.Text("Text messages: "+statusText),
				),
			),
			g.If(enabled,
				P(
					Class("mt-4 text-sm"),
					A(
						Href("/faq/sms-notifications"),
						Class("text-blue-600 dark:text-blue-400 hover:underline"),
						g.Text("Why am I not getting text messages?"),
					),
				),
			),
		)...,
	)
}

func SettingsPage(name, phoneE64 string, smsOptedOut bool,
	userID int, hasAccountPicture bool, accountPictureURL string) []g.Node {
	return []g.Node{
		pageTitle("Settings"),
		settingsAccountSection(name, phoneE64),
		AccountPictureSection(userID, hasAccountPicture, accountPictureURL, ""),
		NotificationsSSESink(),
		NotificationsSection(smsOptedOut),
		ChangePhoneSection(),
		settingsSection("Change Password",
			Form(
				Class("space-y-4"),
				hx.Post("/auth/user/settings/password"),
				hx.Swap("none"),
				passwordManagerUsername(name),
				labeledPasswordFieldID("Current Password", "current_password",
					"settings_current_password", "current-password",
					false, true),
				Div(
					labeledPasswordField("New Password", "new_password", "new-password", false),
					Span(
						Class("text-xs text-zinc-500 dark:text-zinc-400 mt-1 block"),
						g.Text(password.StrengthRequirements),
					),
				),
				labeledPasswordField("Confirm New Password", "confirm_password", "new-password", false),
				settingsFormActions(buttonProps{
					Type: "submit",
					Text: "Change Password",
				}, SettingsChangePasswordErrorID),
			),
		),
		settingsSection("Delete Account",
			P(
				Class("text-sm text-zinc-600 dark:text-zinc-400 mb-4"),
				g.Text("Permanently delete your account. Your username cannot be "+
					"reused. Your ads are permanently deleted. Your phone number "+
					"will be held for 10 days before it can be registered again."),
			),
			Form(
				ID("delete-account-form"),
				Class("space-y-4"),
				labeledPasswordField("Password", "password", "current-password", false),
				settingsFormActions(buttonProps{
					Type:  "button",
					Text:  "Delete My Account",
					Class: "bg-red-600 hover:bg-red-700",
					Attrs: []g.Node{
						hx.Get("/auth/user/settings/delete-confirm"),
						hx.Target("body"),
						hx.Swap("beforeend"),
					},
				}, SettingsDeleteAccountErrorID),
			),
		),
		g.Raw(`<script src="/js/account-picture.js" defer></script>`),
	}
}

// AccountPictureSection is the settings block for upload/URL/remove.
func AccountPictureSection(userID int, hasPicture bool,
	pictureURL, errMsg string) g.Node {
	previewSrc := ""
	if hasPicture {
		previewSrc = UserAccountPictureSrc(userID)
	}
	uploadLabel := "Upload picture"
	if hasPicture {
		uploadLabel = "Replace picture"
	}

	nodes := []g.Node{
		H2(Class("text-xl font-semibold mb-4"), g.Text("Account Picture")),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400 mb-4"),
			g.Text("Shown full-width on your public profile below your "+
				"member info. Optional link opens in a new tab when the "+
				"picture is clicked."),
		),
	}

	if previewSrc != "" {
		nodes = append(nodes,
			Div(
				Class("mb-4"),
				Img(
					Src(previewSrc),
					Alt("Account picture preview"),
					Class("w-full max-h-[300px] object-cover rounded"),
				),
			),
		)
	}

	nodes = append(nodes,
		Div(
			Class("space-y-4"),
			Div(
				Label(
					Class("block text-sm font-medium mb-1"),
					g.Text(uploadLabel),
				),
				Input(
					Type("file"),
					ID("account-picture-file"),
					Accept("image/*"),
					Class("block w-full text-sm text-zinc-600 dark:text-zinc-400 "+
						"file:mr-4 file:py-2 file:px-4 file:rounded "+
						"file:border-0 file:bg-zinc-200 dark:file:bg-zinc-700 "+
						"file:text-zinc-900 dark:file:text-zinc-100"),
				),
				Div(
					Class("mt-2 flex items-center gap-4"),
					standardButton(buttonProps{
						Type: "button",
						Text: uploadLabel,
						Attrs: []g.Node{
							ID("account-picture-upload-btn"),
						},
					}),
					Span(
						ID("account-picture-status"),
						Class("text-sm text-zinc-500 dark:text-zinc-400 hidden"),
					),
				),
			),
			Form(
				Class("space-y-4"),
				hx.Post("/auth/user/settings/account-picture"),
				hx.Target("#account-picture-section"),
				hx.Swap("outerHTML"),
				labeledTextInput("Link URL (optional)", "account_picture_url",
					Type("url"),
					Name("account_picture_url"),
					Value(pictureURL),
					Placeholder("https://example.com"),
				),
				Div(
					Class("flex items-center gap-4"),
					standardButton(buttonProps{
						Type: "submit",
						Text: "Save link",
					}),
					Div(
						ID(SettingsAccountPictureErrorID),
						Class("text-red-500 text-lg"),
						g.If(errMsg != "", g.Text(errMsg)),
					),
				),
			),
		),
	)

	if hasPicture {
		nodes = append(nodes,
			Form(
				Class("mt-4"),
				hx.Post("/auth/user/settings/account-picture/remove"),
				hx.Target("#account-picture-section"),
				hx.Swap("outerHTML"),
				standardButton(buttonProps{
					Type:  "submit",
					Text:  "Remove picture",
					Class: "bg-zinc-600 hover:bg-zinc-700",
				}),
			),
		)
	}

	return Div(
		ID("account-picture-section"),
		Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
		g.Group(nodes),
	)
}

func ChangePhoneSection() g.Node {
	return Div(
		ID("change-phone-section"),
		Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
		H2(Class("text-xl font-semibold mb-4"), g.Text("Change Phone Number")),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400 mb-4"),
			g.Text("Enter your current password and a new phone number. "+
				"We will send a verification code to the new number."),
		),
		ChangePhoneRequestForm(),
	)
}

func ChangePhoneRequestForm() g.Node {
	return Form(
		Class("space-y-4"),
		hx.Post("/auth/user/settings/phone"),
		hx.Swap("none"),
		labeledPasswordFieldID("Current Password", "current_password",
			"phone_current_password", "current-password", false, false),
		Div(
			Div(
				Class("flex items-baseline justify-between mb-1"),
				Label(
					Class("text-base font-medium"),
					For("new-phone"),
					g.Text("New Phone Number"),
				),
			),
			Input(
				Class(textFieldClass),
				Type("tel"),
				Name("phone"),
				ID("new-phone"),
				MinLength("10"),
				MaxLength("20"),
				g.Attr("placeholder", "+12025550123 or 202-555-0123"),
				g.Attr("pattern", "^\\+?[\\d\\s\\-\\(\\)\\.]{10,20}$"),
				g.Attr("autocomplete", "tel"),
				Required(),
			),
		),
		TurnstileWidget(),
		settingsFormActions(buttonProps{
			Type: "submit",
			Text: "Send Verification Code",
		}, SettingsChangePhoneErrorID),
	)
}

func ChangePhoneVerifySection(phoneE64 string) g.Node {
	return Div(
		ID("change-phone-section"),
		Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
		hx.SwapOOB("outerHTML"),
		H2(Class("text-xl font-semibold mb-4"), g.Text("Change Phone Number")),
		Form(
			Class("space-y-4"),
			hx.Post("/auth/user/settings/phone/verify"),
			hx.Swap("none"),
			Input(Type("hidden"), Name("phone"), Value(phoneE64)),
			P(
				Class("text-sm text-zinc-600 dark:text-zinc-400"),
				g.Text("We've sent a verification code to your new phone number. "+
					"Enter it below to finish changing your number."),
			),
			Label(
				Class("block text-base font-medium mb-2"),
				For("change-phone-code"),
				g.Text("Verification Code"),
			),
			Input(
				ID("change-phone-code"),
				Class("max-w-xs w-full p-4 border-2 border-zinc-300 "+
					"dark:border-zinc-600 rounded-md text-center text-2xl "+
					"font-mono tracking-widest focus:border-blue-500 "+
					"dark:focus:border-blue-400 focus:outline-none "+
					"dark:bg-zinc-800 dark:text-zinc-200 block"),
				Type("text"),
				Name("code"),
				g.Attr("autocomplete", "one-time-code"),
				g.Attr("inputmode", "numeric"),
				g.Attr("pattern", "[0-9]*"),
				g.Attr("maxlength", "6"),
				g.Attr("placeholder", "000000"),
				Autofocus(),
				Required(),
			),
			settingsFormActions(buttonProps{
				Type: "submit",
				Text: "Verify and Update Phone",
			}, SettingsChangePhoneErrorID),
		),
	)
}

func ChangePhoneSuccess(userName, phoneE64 string) g.Node {
	return g.Group([]g.Node{
		Div(
			ID("settings-account"),
			Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
			hx.SwapOOB("outerHTML"),
			H2(Class("text-xl font-semibold mb-4"), g.Text("Account")),
			Div(
				Class("space-y-4"),
				settingsAccountRow("Username",
					Div(
						Class("flex items-center gap-2 min-w-0"),
						settingsUserAvatar(userName),
						Span(
							Class("text-zinc-900 dark:text-zinc-100 font-medium"),
							g.Text(userName),
						),
					),
				),
				settingsAccountRow("Phone",
					Span(
						Class("text-zinc-900 dark:text-zinc-100"),
						g.Text(phoneformat.Display(phoneE64)),
					),
				),
			),
		),
		Div(
			ID("change-phone-section"),
			Class("mt-8 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
			hx.SwapOOB("outerHTML"),
			H2(Class("text-xl font-semibold mb-4"), g.Text("Change Phone Number")),
			P(
				Class("text-sm text-green-600 dark:text-green-400 mb-4"),
				g.Text("Phone number updated to "+
					phoneformat.Display(phoneE64)+"."),
			),
			ChangePhoneRequestForm(),
		),
	})
}

func DeleteAccountConfirmModal(csrfToken string) g.Node {
	csrfHeader := fmt.Sprintf(`{"X-Csrf-Token": %q}`, csrfToken)
	return g.Group([]g.Node{
		modalBackdrop("delete-account"),
		Div(
			ID("delete-account-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-md shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto"),
				Div(
					Class("flex items-center justify-between p-6 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(Class("text-xl font-bold text-zinc-900 dark:text-zinc-200"),
						g.Text("Delete account?")),
					modalClose("delete-account"),
				),
				Div(
					Class("p-6 flex flex-col gap-4"),
					P(
						Class("text-sm text-zinc-600 dark:text-zinc-400"),
						g.Text("Are you sure you want to permanently delete your account? Your username cannot be reused. Your ads will be permanently deleted. Your phone number will be held for 10 days before it can be registered again."),
					),
					Div(
						Class("flex flex-col gap-2 sm:flex-row sm:justify-end"),
						standardButton(buttonProps{
							Type:  "button",
							Text:  "Cancel",
							Class: "bg-zinc-500 hover:bg-zinc-600 w-full sm:w-auto text-center",
							Attrs: []g.Node{
								hx.Get("/api/modal-remove/delete-account"),
								hx.Swap("none"),
							},
						}),
						standardButton(buttonProps{
							Type:  "button",
							Text:  "Delete Forever",
							Class: "bg-red-600 hover:bg-red-700 w-full sm:w-auto text-center",
							Attrs: []g.Node{
								hx.Post("/auth/user/settings/delete"),
								hx.Headers(csrfHeader),
								hx.Include("#delete-account-form"),
								hx.Swap("none"),
							},
						}),
					),
				),
			),
		),
	})
}

func aboutIconLink(href, icon, alt string, external bool) g.Node {
	attrs := []g.Node{
		Href(href),
		Class("p-2 border border-zinc-300 dark:border-zinc-600 rounded-md shrink-0 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"),
		Img(
			Class("w-5 h-5 dark:invert dark:opacity-80"),
			Src(icon),
			Alt(alt),
		),
	}
	if external {
		attrs = append(attrs, Target("_blank"), Rel("noopener noreferrer"))
	}
	return A(attrs...)
}

func aboutItem(icon, alt, label, href string, external bool,
	value g.Node) g.Node {
	return Div(
		Class("flex items-center gap-4 p-4 border border-zinc-200 dark:border-zinc-700 rounded-lg"),
		aboutIconLink(href, icon, alt, external),
		Div(
			Class("min-w-0"),
			Div(
				Class("text-sm font-medium text-zinc-500 dark:text-zinc-400"),
				g.Text(label),
			),
			Div(Class("mt-0.5"), value),
		),
	)
}

func aboutNoItem(text string) g.Node {
	return Li(
		Class("flex items-center gap-2"),
		Img(
			Class("w-5 h-5 shrink-0"),
			Src("/images/block.svg"),
			Alt(""),
			Style("filter: invert(27%) sepia(94%) saturate(2878%) "+
				"hue-rotate(346deg) brightness(95%) contrast(92%)"),
		),
		g.Text(text),
	)
}

func AboutPage() []g.Node {
	return []g.Node{
		pageTitle("About"),
		P(
			Class("mt-4 text-xl font-medium text-zinc-800 dark:text-zinc-200"),
			g.Textf("%s is classified ads for the Internet",
				config.ServerName),
		),
		Div(
			Class("mt-8 p-2 sm:p-3 bg-zinc-100 dark:bg-zinc-800 "+
				"border border-zinc-300 dark:border-zinc-600 shadow-md"),
			Img(
				Src("/images/classifieds.jpg"),
				Alt("Newspaper classifieds"),
				Class("w-full max-h-[480px] object-cover"),
			),
		),
		Div(
			Class("mt-10 space-y-8 text-zinc-700 dark:text-zinc-300"),
			Div(
				Class("space-y-6"),
				P(
					Class("text-lg leading-relaxed text-zinc-800 "+
						"dark:text-zinc-200"),
					g.Textf(
						"Remember classified ads? Easy, simple. "+
							"Post an ad with your phone number and "+
							"folks contact you. %s works the same "+
							"way, except your phone number stays "+
							"hidden.",
						config.ServerName,
					),
				),
				Ul(
					Class("pl-4 space-y-2 text-base font-medium "+
						"text-zinc-800 dark:text-zinc-200"),
					aboutNoItem("No email"),
					aboutNoItem("No Facebook friends"),
					aboutNoItem("No posting fees"),
					aboutNoItem("No credit cards"),
				),
				P(
					Class("text-lg"),
					g.Text("All you need is your phone number to "),
					faqLink("/register", "get started"),
				),
				P(
					Class("text-lg inline-flex items-center flex-wrap"),
					g.Text("And the fun part? You get to "),
					faqLink("/faq/rocks", "throw rocks"),
					Img(
						Class("w-9 h-9 ml-1"),
						Src("/images/rock.svg"),
						Alt(""),
					),
				),
			),
			Div(
				Class("space-y-3"),
				aboutItem(
					"/images/question_mark.svg",
					"FAQ",
					"FAQ",
					"/faq",
					false,
					faqLink("/faq", "Frequently asked questions"),
				),
				aboutItem(
					"/images/info.svg",
					"Privacy",
					"Privacy",
					"/privacy",
					false,
					faqLink("/privacy", "Privacy Policy"),
				),
				aboutItem(
					"/images/balance.svg",
					"Terms",
					"Terms",
					"/terms",
					false,
					faqLink("/terms", "Terms of Service"),
				),
				aboutItem(
					"/images/contact_mail.svg",
					"Contact",
					"Contact",
					"mailto:"+config.ContactEmail,
					false,
					A(
						Href("mailto:"+config.ContactEmail),
						Class("text-blue-600 dark:text-blue-400 hover:underline"),
						g.Text(config.ContactEmail),
					),
				),
				aboutItem(
					"/images/code.svg",
					"Source code",
					"Source code",
					config.GitHubRepoURL,
					true,
					A(
						Href(config.GitHubRepoURL),
						Target("_blank"),
						Rel("noopener noreferrer"),
						Class("text-blue-600 dark:text-blue-400 hover:underline break-all"),
						g.Text(config.GitHubRepoURL),
					),
				),
				aboutItem(
					"/images/bar_chart.svg",
					"Fun Stats",
					"Fun Stats",
					"/funstats",
					false,
					faqLink("/funstats", "Users and ads over time"),
				),
				aboutItem(
					"/images/money.svg",
					"Donate",
					"Donate",
					"/donate",
					false,
					faqLink("/donate", "Help with operating costs"),
				),
			),
			P(
				Class("text-sm text-zinc-500 dark:text-zinc-400"),
				g.Text("Location data powered by "),
				A(
					Href("https://www.geoapify.com/"),
					Target("_blank"),
					Rel("noopener noreferrer"),
					Class("text-blue-600 dark:text-blue-400 hover:underline"),
					g.Text("Geoapify"),
				),
				g.Text(". © "),
				A(
					Href("https://www.openstreetmap.org/copyright"),
					Target("_blank"),
					Rel("noopener noreferrer"),
					Class("text-blue-600 dark:text-blue-400 hover:underline"),
					g.Text("OpenStreetMap contributors"),
				),
				g.Text("."),
			),
		),
	}
}
