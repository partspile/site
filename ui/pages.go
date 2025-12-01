package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/config"
)

func HomePage(userID int, userName, csrfToken string, view, adCategory int, path string, params map[string]string) g.Node {
	return Page(
		"Parts Pile",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			SearchContainer(userID, view, adCategory, params),
		},
	)
}

func RegisterPage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Register",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Register"),
			contentContainer(
				Div(
					ID("register-content"),
					Div(
						Class("text-center mb-6"),
						P(
							Class("text-gray-600"),
							g.Text("Enter your information below. We'll send a verification code to your phone number to complete registration."),
						),
					),

					formContainer("registerForm",
						formGroup("Username", "name",
							Div(
								Input(
									Type("text"),
									ID("name"),
									Name("name"),
									Class("w-full p-2 border rounded"),
									g.Attr("pattern", "^[a-zA-Z][a-zA-Z0-9]{2,19}$"),
									g.Attr("maxlength", "20"),
									g.Attr("placeholder", "JohnDoe"),
									Required(),
								),
								Span(
									Class("text-xs text-gray-500 mt-1"),
									g.Text("3-20 characters, letters and digits only. Must start with a letter."),
								),
							),
						),
						formGroup("Phone Number", "phone",
							Div(
								Input(
									Type("text"),
									ID("phone"),
									Name("phone"),
									Class("w-full p-2 border rounded"),
									g.Attr("placeholder", "+12025550123 or 202-555-0123"),
									Required(),
								),
								Span(
									Class("text-xs text-gray-500 mt-1"),
									g.Text("Enter your phone in international format (e.g. +12025550123) or US/Canada format (e.g. 503-523-8780)."),
								),
							),
						),
						Div(
							Class("space-y-3"),
							Checkbox("offers", "true", "I agree to receive informational text messages", false, false, Required()),
						),
						Div(
							Class("text-xs text-gray-600 bg-gray-50 p-3 rounded border"),
							g.Text("By providing your phone number you agree to receive informational text messages from Parts Pile. Message frequency will vary. Msg & data rates may apply. Reply HELP for help or STOP to cancel. We only use your phone for essential communications and verification."),
						),
						actionButtons(
							button("Send Verification Code",
								withAttributes(
									hx.Post("/api/register/step1"),
									hx.Target("#register-content"),
									hx.Swap("outerHTML"),
									hx.Indicator("#registerForm"),
								),
							),
						),
					),
					Div(
						Class("bg-blue-50 border border-blue-200 rounded-lg p-4 mt-6"),
						H3(
							Class("text-lg font-semibold text-blue-900 mb-2"),
							g.Text("Privacy First"),
						),
						P(
							Class("text-blue-800 text-sm leading-relaxed"),
							g.Text("At parts-pile, we believe your privacy is important. We collect only the minimal personal information needed to operate our service:"),
						),
						Ul(
							Class("text-blue-800 text-sm mt-2 space-y-1"),
							Li(g.Text("• Phone number (required) - for verification and notifications")),
							Li(g.Text("• Username (required) - to identify you on the platform")),
							Li(g.Text("• Email (optional) - only if you choose email notifications in settings")),
						),
						P(
							Class("text-blue-800 text-sm mt-2 leading-relaxed"),
							g.Text("We don't collect real names, addresses, credit card information, or any other personal details. Your phone number requires verification to prevent abuse and ensure legitimate users. "),
							A(
								Href("/privacy"),
								Class("text-blue-600 hover:text-blue-800 underline font-medium"),
								g.Text("Learn more in our Privacy Policy"),
							),
							g.Text("."),
						),
					),
				),
			),
		},
	)
}

func LoginPage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Login",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Login"),
			contentContainer(
				formContainer("loginForm",
					formGroup("Username", "name",
						TextInput("name", "name", ""),
					),
					formGroup("Password", "password",
						passwordInput("password", "password"),
					),
					Div(
						Class("text-right mb-4"),
						A(
							Href("/password-recovery"),
							Class("text-blue-500 hover:underline text-sm"),
							g.Text("Forgot password?"),
						),
					),
					actionButtons(
						button("Login",
							withAttributes(
								hx.Post("/api/login"),
								hx.Target("#result"),
								hx.Indicator("#loginForm"),
							),
						),
					),
					resultContainer(),
				),
			),
		},
	)
}

func PasswordRecoveryPage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Password Recovery",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Password Recovery"),
			contentContainer(
				Div(
					ID("password-recovery-content"),
					Div(
						Class("text-center mb-6"),
						P(
							Class("text-gray-600"),
							g.Text("Enter your username to recover your password. "+
								"We'll send a verification code to your registered phone number."),
						),
					),
					formContainer("passwordRecoveryForm",
						formGroup("Username", "username",
							TextInput("username", "username", ""),
						),
						actionButtons(
							button("Send verification code",
								withAttributes(
									hx.Post("/api/password-recovery/step1"),
									hx.Target("#result"),
									hx.Indicator("#passwordRecoveryForm"),
								),
							),
						),
						resultContainer(),
					),
				),
			),
		},
	)
}

// PasswordRecoveryVerifyPage returns the step 2 content (verification code + password form)
func PasswordRecoveryVerifyPage(username string, phone string) g.Node {
	return Div(
		ID("password-recovery-content"),
		Div(
			Class("text-center mb-6"),
			P(
				Class("text-gray-600"),
				g.Text("We've sent a verification code to your phone number. "+
					"Please enter the code below along with your new password."),
			),
		),
		formContainer("verifyForm",
			// Hidden fields
			Input(
				Type("hidden"),
				Name("username"),
				Value(username),
			),
			Input(
				Type("hidden"),
				Name("recovery_phone"),
				Value(phone),
			),
			formGroup("Verification Code", "verification_code",
				TextInput("verification_code", "verification_code", ""),
			),
			formGroup("New Password", "password",
				passwordInput("password", "password"),
			),
			formGroup("Confirm Password", "password2",
				passwordInput("password2", "password2"),
			),
			actionButtons(
				button("Reset Password",
					withAttributes(
						hx.Post("/api/password-recovery/step2"),
						hx.Target("#result"),
						hx.Indicator("#verifyForm"),
					),
				),
			),
			resultContainer(),
		),
	)
}

// PasswordRecoveryVerifyPageWithSwapOob returns the verification form with out-of-band swap wrapper
// This includes an empty result div to clear errors and the verification form with hx-swap-oob
func PasswordRecoveryVerifyPageWithSwapOob(username string, smsOptedOut bool) g.Node {
	return Div(
		Div(ID("result")), // Empty result div to clear errors
		Div(
			g.Attr("hx-swap-oob", "true"),
			ID("password-recovery-content"),
			Div(
				Class("text-center mb-6"),
				P(
					Class("text-gray-600"),
					g.Text("We've sent a verification code to your phone number. "+
						"Please enter the code below along with your new password."),
				),
			),
			formContainer("verifyForm",
				Input(Type("hidden"), Name("username"), Value(username)),
				formGroup("Verification Code", "verification_code",
					TextInput("verification_code", "verification_code", ""),
				),
				formGroup("New Password", "password",
					passwordInput("password", "password"),
				),
				formGroup("Confirm Password", "password2",
					passwordInput("password2", "password2"),
				),
				Div(
					Class("text-center mb-4"),
					Button(
						Type("button"),
						Class("text-blue-500 hover:underline text-sm"),
						g.Attr("onclick", "document.getElementById('troubleshoot-section').classList.toggle('hidden')"),
						g.Text("Didn't receive code?"),
					),
				),
				Div(
					ID("troubleshoot-section"),
					Class(func() string {
						if smsOptedOut {
							return "bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6"
						}
						return "bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6 hidden"
					}()),
					Div(
						Class("space-y-4"),
						H4(
							Class("text-sm font-semibold text-blue-900 mb-2"),
							g.Text("Troubleshooting"),
						),
						Div(
							Class("text-sm text-blue-800 mb-3"),
							P(g.Text("If you didn't receive the code, try:")),
							Ul(
								Class("list-disc list-outside ml-5 mt-2 space-y-2"),
								Li(Class("break-words pl-1"), g.Text("Check your phone's messages")),
								Li(Class("break-words pl-1"), g.Text("Wait a few moments - messages can be delayed")),
								Li(
									Class("break-words pl-1"),
									g.Text(fmt.Sprintf("SMS may be disabled on your phone number. To enable receiving SMS messages, send UNSTOP to %s from your phone", config.TwilioFromNumber)),
								),
								Li(Class("break-words pl-1"), g.Text("Click 'Resend Code' below to request a new code")),
							),
						),
						Div(
							Class("flex justify-center"),
							button("Resend Code",
								withAttributes(
									hx.Post("/api/password-recovery/resend"),
									hx.Target("#result"),
									hx.Include("#verifyForm"),
									hx.Indicator("#verifyForm"),
								),
							),
						),
					),
				),
				actionButtons(
					button("Reset Password",
						withAttributes(
							hx.Post("/api/password-recovery/step2"),
							hx.Target("#result"),
							hx.Indicator("#verifyForm"),
						),
					),
				),
				resultContainer(),
			),
		),
	)
}

// VerificationPageContent returns just the verification form content without the Page wrapper
func VerificationPageContent(username string) g.Node {
	return Div(
		ID("register-content"),
		Div(
			Class("text-center mb-6"),
			P(
				Class("text-gray-600"),
				g.Text("We've sent a verification code to your phone number. "+
					"Please enter the code below to complete your registration."),
			),
		),
		formContainer("verificationForm",
			// Hidden username field for password managers
			Input(
				Type("hidden"),
				Name("username"),
				Value(username),
				g.Attr("autocomplete", "username"),
			),
			formGroup("Verification Code", "verification_code",
				TextInput("verification_code", "verification_code", ""),
			),
			formGroup("Password", "password",
				passwordInput("password", "password"),
			),
			formGroup("Confirm Password", "password2",
				passwordInput("password2", "password2"),
			),
			Div(
				Class("flex items-center gap-2"),
				Checkbox("terms", "accepted", "I accept the ", false, false, Required()),
				A(
					Href("/terms"),
					Class("text-blue-600 hover:text-blue-800 underline"),
					g.Text("Terms of Service"),
				),
				g.Text(" & "),
				A(
					Href("/privacy"),
					Class("text-blue-600 hover:text-blue-800 underline"),
					g.Text("Privacy Policy"),
				),
				g.Text("."),
			),
			actionButtons(
				button("Complete Registration",
					withAttributes(
						hx.Post("/api/register/step2"),
						hx.Target("#result"),
						hx.Indicator("#verificationForm"),
					),
				),
			),
			resultContainer(),
		),
	)
}

func TermsOfServicePage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Terms of Service",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Terms of Service"),
			contentContainer(
				Div(
					Class("prose max-w-none"),
					H2(Class("text-xl font-semibold mb-4"), g.Text("Terms of Service")),
					P(Class("mb-4"), g.Text("Last updated: December 2024")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("1. Acceptance of Terms")),
					P(Class("mb-4"), g.Text("By accessing and using Parts Pile, you accept and agree to be bound by the terms and provision of this agreement.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("2. Use License")),
					P(Class("mb-4"), g.Text("Permission is granted to temporarily download one copy of the materials on Parts Pile for personal, non-commercial transitory viewing only.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("3. Disclaimer")),
					P(Class("mb-4"), g.Text("The materials on Parts Pile are provided on an 'as is' basis. Parts Pile makes no warranties, expressed or implied, and hereby disclaims and negates all other warranties including without limitation, implied warranties or conditions of merchantability, fitness for a particular purpose, or non-infringement of intellectual property or other violation of rights.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("4. Limitations")),
					P(Class("mb-4"), g.Text("In no event shall Parts Pile or its suppliers be liable for any damages (including, without limitation, damages for loss of data or profit, or due to business interruption) arising out of the use or inability to use the materials on Parts Pile.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("5. Revisions and Errata")),
					P(Class("mb-4"), g.Text("The materials appearing on Parts Pile could include technical, typographical, or photographic errors. Parts Pile does not warrant that any of the materials on its website are accurate, complete or current.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("6. Links")),
					P(Class("mb-4"), g.Text("Parts Pile has not reviewed all of the sites linked to its website and is not responsible for the contents of any such linked site. The inclusion of any link does not imply endorsement by Parts Pile of the site.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("7. Site Terms of Use Modifications")),
					P(Class("mb-4"), g.Text("Parts Pile may revise these terms of use for its website at any time without notice. By using this website you are agreeing to be bound by the then current version of these Terms and Conditions of Use.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("8. Governing Law")),
					P(Class("mb-4"), g.Text("Any claim relating to Parts Pile shall be governed by the laws of the United States without regard to its conflict of law provisions.")),
				),
			),
		},
	)
}

func PrivacyPolicyPage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Privacy Policy",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Privacy Policy"),
			contentContainer(
				Div(
					Class("prose max-w-none"),
					H2(Class("text-xl font-semibold mb-4"), g.Text("Privacy Policy")),
					P(Class("mb-4"), g.Text("Last updated: December 2024")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("1. Information We Collect")),
					P(Class("mb-4"), g.Text("We collect only the minimal personal information necessary to operate our service. This includes:")),
					Ul(Class("ml-4 mb-4 space-y-2"),
						Li(g.Text("• Username (required) - to identify you on the platform")),
						Li(g.Text("• Phone number (required) - for verification and essential communications")),
						Li(g.Text("• Email address (optional) - only if you choose email notifications in settings")),
					),
					P(Class("mb-4"), g.Text("We do not collect real names, addresses, credit card information, or any other personal details. We believe in collecting as little personal information as possible while still providing a secure and functional service.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("2. How We Use Your Information")),
					P(Class("mb-4"), g.Text("We use the minimal information we collect solely for:")),
					Ul(Class("ml-4 mb-4 space-y-2"),
						Li(g.Text("• Account verification and security")),
						Li(g.Text("• Essential service communications")),
						Li(g.Text("• Platform functionality and user identification")),
					),
					P(Class("mb-4"), g.Text("We do not use your information for marketing, advertising, or any other purposes beyond what is necessary to operate the service.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("3. Information Sharing")),
					P(Class("mb-4"), g.Text("We do not sell, trade, or otherwise transfer your personal information to third parties without your consent, except as described in this policy.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("4. Data Security")),
					P(Class("mb-4"), g.Text("We implement appropriate security measures to protect your personal information against unauthorized access, alteration, disclosure, or destruction.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("5. SMS Communications")),
					P(Class("mb-4"), g.Text("By providing your phone number, you consent to receive informational text messages from Parts Pile. Message frequency will vary. You may opt out at any time by replying STOP.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("6. Cookies and Tracking")),
					P(Class("mb-4"), g.Text("We use a minimal set of cookies solely for essential website functionality. We do not use cookies for analytics, advertising, or tracking user behavior beyond what is necessary for the service to operate.")),

					H4(Class("text-md font-semibold mb-2"), g.Text("Cookies We Use:")),

					Div(Class("ml-4 mb-4"),
						H5(Class("text-sm font-semibold mb-1"), g.Text("last_view")),
						P(Class("text-sm mb-2"), g.Text("Purpose: Saves your preferred view layout (list, grid, or tree) for a seamless browsing experience.")),
						P(Class("text-sm mb-2"), g.Text("Data collected: Single view preference value (e.g., 'list', 'grid', 'tree').")),
						P(Class("text-sm mb-2"), g.Text("Retention: Expires after 30 days.")),
						P(Class("text-sm mb-2"), g.Text("Third parties: None.")),
					),

					Div(Class("ml-4 mb-4"),
						H5(Class("text-sm font-semibold mb-1"), g.Text("ad_category")),
						P(Class("text-sm mb-2"), g.Text("Purpose: Saves your preferred ad category (CarPart, MotorcyclePart, BicyclePart, AgPart) to remember your browsing preference.")),
						P(Class("text-sm mb-2"), g.Text("Data collected: Single category preference value (e.g., 'CarPart', 'MotorcyclePart').")),
						P(Class("text-sm mb-2"), g.Text("Retention: Expires after 30 days.")),
						P(Class("text-sm mb-2"), g.Text("Third parties: None.")),
					),

					Div(Class("ml-4 mb-4"),
						H5(Class("text-sm font-semibold mb-1"), g.Text("Session Cookies")),
						P(Class("text-sm mb-2"), g.Text("Purpose: Maintains your login session and authentication state while using the website.")),
						P(Class("text-sm mb-2"), g.Text("Data collected: User ID and session identifier for authentication.")),
						P(Class("text-sm mb-2"), g.Text("Retention: Expires when you close your browser or log out.")),
						P(Class("text-sm mb-2"), g.Text("Third parties: None.")),
					),

					H3(Class("text-lg font-semibold mb-2"), g.Text("7. Your Rights")),
					P(Class("mb-4"), g.Text("You have the right to access, update, or delete your personal information. You may also opt out of certain communications or request that we restrict the processing of your information.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("8. Changes to This Policy")),
					P(Class("mb-4"), g.Text("We may update this privacy policy from time to time. We will notify you of any changes by posting the new policy on this page.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("9. Contact Us")),
					P(Class("mb-4"), g.Text("If you have any questions about this privacy policy, please contact us through our website.")),
				),
			),
		},
	)
}

func RocksPage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Your Rocks - Parts Pile",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Your Rocks"),
			contentContainer(
				Div(
					Class("text-center mb-8"),
					H2(
						Class("text-3xl font-bold text-gray-900 mb-4"),
						g.Text("🎯 Welcome to Parts Pile!"),
					),
					P(
						Class("text-lg text-gray-600 mb-6"),
						g.Text("You've been given 3 rocks to help maintain quality on our platform."),
					),
					Div(
						Class("bg-blue-50 border border-blue-200 rounded-lg p-6 mb-6"),
						H3(
							Class("text-xl font-semibold text-blue-900 mb-3"),
							g.Text("How Rocks Work"),
						),
						Ul(
							Class("text-blue-800 space-y-2 text-left"),
							Li(g.Text("• Throw rocks at ads that violate our policies or have issues")),
							Li(g.Text("• Each rock creates a conversation with the ad owner")),
							Li(g.Text("• Work together to resolve the dispute")),
							Li(g.Text("• Once resolved, the seller can return your rock")),
							Li(g.Text("• Rocks are limited - use them wisely!")),
						),
					),
					Div(
						Class("bg-green-50 border border-green-200 rounded-lg p-6 mb-8"),
						H3(
							Class("text-xl font-semibold text-green-900 mb-3"),
							g.Text("Your Rock Inventory"),
						),
						Div(
							Class("flex justify-center items-center space-x-4 text-2xl font-bold text-green-800"),
							Span(g.Text("🪨")),
							Span(g.Text("🪨")),
							Span(g.Text("🪨")),
						),
						P(
							Class("text-green-700 mt-2"),
							g.Text("You have 3 rocks available"),
						),
					),
					Div(
						Class("text-center"),
						button("Continue to Login", withHref("/login"), withClass("text-lg px-8 py-3")),
					),
				),
			),
		},
	)
}

func AdDeletedPage(userID int, userName, path, csrfToken string) g.Node {
	return Page(
		"Ad Deleted - Parts Pile",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			contentContainer(
				Div(
					Class("text-center py-16"),
					Div(
						Class("mb-6 flex justify-center"),
						largeIcon("/images/trashcan.svg", "Deleted", "24"),
					),
					H2(
						Class("text-3xl font-bold text-gray-900 mb-4"),
						g.Text("Ad Deleted"),
					),
					P(
						Class("text-lg text-gray-600 mb-8"),
						g.Text("This ad has been deleted by the owner and is no longer available."),
					),
					Div(
						Class("text-center"),
						button("Back to Home", withHref("/"), withClass("text-lg px-8 py-3")),
					),
				),
			),
		},
	)
}
