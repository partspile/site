package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func userNameInput(
	showHelp bool,
	autocomplete string,
	autofocus bool,
	value string,
) g.Node {
	attrs := []g.Node{
		Type("text"),
		Name("username"),
		MinLength("3"),
		MaxLength("20"),
		g.Attr("pattern", "^[a-zA-Z][a-zA-Z0-9]{2,19}$"),
		g.Attr("autocomplete", autocomplete),
		Required(),
	}
	if autofocus {
		attrs = append(attrs, Autofocus())
	}
	if value != "" {
		attrs = append(attrs, Value(value))
	}
	return Div(
		labeledTextInput("Username", "username", attrs...),
		g.If(showHelp, Span(
			Class("text-xs text-zinc-500 dark:text-zinc-400 mt-1"),
			g.Text("3-20 characters, letters and digits only. Must start with a letter."),
		)),
	)
}

func passwordInput(autocomplete string) g.Node {
	return labeledPasswordField("Password", "password", autocomplete, false)
}

// passwordManagerUsername shows the username in a disabled-looking field and
// includes a readonly text input (not type=hidden) so password managers can
// associate it. Readonly (not disabled) so the value still submits.
func passwordManagerUsername(username string) g.Node {
	return Div(
		label("Username"),
		Input(
			Class(textFieldClass+
				" bg-zinc-100 dark:bg-zinc-800 "+
				"text-zinc-500 dark:text-zinc-400 cursor-not-allowed"),
			Type("text"),
			Name("username"),
			ID("username"),
			Value(username),
			g.Attr("autocomplete", "username"),
			g.Attr("readonly", "readonly"),
			g.Attr("tabindex", "-1"),
		),
	)
}

func LoginForm(returnPath string) g.Node {
	nodes := []g.Node{
		Class("space-y-8 mt-8"),
		hx.Post("/api/login"),
		hx.Swap("none"),
	}
	if returnPath != "" {
		nodes = append(nodes, Input(
			Type("hidden"),
			Name("return"),
			Value(returnPath),
		))
	}
	nodes = append(nodes,
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Don't have an account? "),
			A(
				Href("/register"),
				Class("text-blue-600 dark:text-blue-400 hover:underline"),
				g.Attr("onclick",
					"event.preventDefault();"+
						"var u=document.getElementById('username').value.trim();"+
						"location.href=u"+
						"?('/register?username='+encodeURIComponent(u))"+
						":'/register';"),
				g.Text("Sign up"),
			),
		),
		userNameInput(false, "username", true, ""),
		passwordInput("current-password"),
		Div(
			Class("space-y-3"),
			Div(
				Class("flex items-center gap-4"),
				standardButton(buttonProps{
					Type: "submit",
					Text: "Login",
				}),
				ErrorDiv(""),
			),
			P(
				Class("text-sm text-zinc-600 dark:text-zinc-400"),
				A(
					Href("/recover"),
					Class("text-blue-600 dark:text-blue-400 hover:underline"),
					g.Text("Recover account"),
				),
			),
		),
	)
	return Form(nodes...)
}

func LoginPage(returnPath string) []g.Node {
	return []g.Node{
		pageTitle("Login"),
		LoginForm(returnPath),
	}
}
