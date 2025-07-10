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
				FormContainer("registerForm",
					FormGroup("Username", "name",
						TextInput("name", "name", ""),
					),
					FormGroup("Phone Number", "phone",
						Div(
							Input(
								Type("text"),
								ID("phone"),
								Name("phone"),
								Class("w-full p-2 border rounded"),
								g.Attr("placeholder", "+12025550123 or 202-555-0123"),
							),
							Span(
								Class("text-xs text-gray-500 mt-1"),
								g.Text("Enter your phone in international format (e.g. +12025550123) or US/Canada format (e.g. 503-523-8780)."),
							),
						),
					),
					FormGroup("Password", "password",
						PasswordInput("password", "password"),
					),
					FormGroup("Confirm Password", "password2",
						PasswordInput("password2", "password2"),
					),
					ActionButtons(
						StyledButton("Register", ButtonPrimary,
							hx.Post("/api/register"),
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
