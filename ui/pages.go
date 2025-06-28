package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"

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
						TextInput("phone", "phone", ""),
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
