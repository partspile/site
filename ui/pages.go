package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/user"
)

func HomePage(currentUser *user.User, path string, view string) g.Node {
	return Page(
		"Parts Pile - Auto Parts and Sales",
		currentUser,
		path,
		[]g.Node{
			InitialSearchResults(view),
		},
	)
}

func RegisterPage(currentUser *user.User, path string) g.Node {
	return Page(
		"Register",
		currentUser,
		path,
		[]g.Node{
			PageHeader("Register"),
			ContentContainer(
				Div(
					Class("text-center mb-6"),
					P(
						Class("text-gray-600"),
						g.Text("Enter your information below. We'll send a verification code to your phone number to complete registration."),
					),
				),
				FormContainer("registerForm",
					FormGroup("Username", "name",
						Input(
							Type("text"),
							ID("name"),
							Name("name"),
							Class("w-full p-2 border rounded"),
							Required(),
						),
					),
					FormGroup("Phone Number", "phone",
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
						Div(
							Class("flex items-center space-x-2"),
							Input(
								Type("checkbox"),
								Name("terms"),
								Value("true"),
								ID("terms-true"),
								Required(),
							),
							Label(
								For("terms-true"),
								Class("text-sm"),
								g.Text("I accept the "),
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
						),
					),
					Div(
						Class("text-xs text-gray-600 bg-gray-50 p-3 rounded border"),
						g.Text("By providing your phone number you agree to receive informational text messages from Parts Pile. Consent is not a condition of purchase. Message frequency will vary. Msg & data rates may apply. Reply HELP for help or STOP to cancel."),
					),
					ActionButtons(
						StyledButton("Send Verification Code", ButtonPrimary,
							hx.Post("/api/register/step1"),
							hx.Target("#result"),
							hx.Indicator("#registerForm"),
						),
					),
					ResultContainer(),
				),
			),
		},
	)
}

func LoginPage(currentUser *user.User, path string) g.Node {
	return Page(
		"Login",
		currentUser,
		path,
		[]g.Node{
			PageHeader("Login"),
			ContentContainer(
				FormContainer("loginForm",
					FormGroup("Username", "name",
						TextInput("name", "name", ""),
					),
					FormGroup("Password", "password",
						PasswordInput("password", "password"),
					),
					ActionButtons(
						StyledButton("Login", ButtonPrimary,
							hx.Post("/api/login"),
							hx.Target("#result"),
							hx.Indicator("#loginForm"),
						),
					),
					ResultContainer(),
				),
			),
		},
	)
}

func VerificationPage(currentUser *user.User, path string) g.Node {
	return Page(
		"Verify Phone Number",
		currentUser,
		path,
		[]g.Node{
			PageHeader("Verify Your Phone Number"),
			ContentContainer(
				Div(
					Class("text-center mb-6"),
					P(
						Class("text-gray-600"),
						g.Text("We've sent a verification code to your phone number. "+
							"Please enter the code below to complete your registration."),
					),
				),
				FormContainer("verificationForm",
					FormGroup("Verification Code", "verification_code",
						TextInput("verification_code", "verification_code", ""),
					),
					FormGroup("Password", "password",
						PasswordInput("password", "password"),
					),
					FormGroup("Confirm Password", "password2",
						PasswordInput("password2", "password2"),
					),
					Div(
						Class("space-y-2"),
						Checkbox("terms", "accepted", "I accept the ", false, false),
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
					ActionButtons(
						StyledButton("Complete Registration", ButtonPrimary,
							hx.Post("/api/register/verify"),
							hx.Target("#result"),
							hx.Indicator("#verificationForm"),
						),
					),
					ResultContainer(),
				),
			),
		},
	)
}

func TermsOfServicePage(currentUser *user.User, path string) g.Node {
	return Page(
		"Terms of Service",
		currentUser,
		path,
		[]g.Node{
			PageHeader("Terms of Service"),
			ContentContainer(
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

func PrivacyPolicyPage(currentUser *user.User, path string) g.Node {
	return Page(
		"Privacy Policy",
		currentUser,
		path,
		[]g.Node{
			PageHeader("Privacy Policy"),
			ContentContainer(
				Div(
					Class("prose max-w-none"),
					H2(Class("text-xl font-semibold mb-4"), g.Text("Privacy Policy")),
					P(Class("mb-4"), g.Text("Last updated: December 2024")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("1. Information We Collect")),
					P(Class("mb-4"), g.Text("We collect information you provide directly to us, such as when you create an account, post ads, or contact us. This may include your name, phone number, email address, and any other information you choose to provide.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("2. How We Use Your Information")),
					P(Class("mb-4"), g.Text("We use the information we collect to provide, maintain, and improve our services, to communicate with you, and to develop new features.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("3. Information Sharing")),
					P(Class("mb-4"), g.Text("We do not sell, trade, or otherwise transfer your personal information to third parties without your consent, except as described in this policy.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("4. Data Security")),
					P(Class("mb-4"), g.Text("We implement appropriate security measures to protect your personal information against unauthorized access, alteration, disclosure, or destruction.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("5. SMS Communications")),
					P(Class("mb-4"), g.Text("By providing your phone number, you consent to receive informational text messages from Parts Pile. Message frequency will vary. You may opt out at any time by replying STOP.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("6. Cookies and Tracking")),
					P(Class("mb-4"), g.Text("We use cookies and similar tracking technologies to enhance your experience on our website and to analyze how our services are used.")),

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
