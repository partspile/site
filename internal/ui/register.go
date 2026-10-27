package ui

import (
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/password"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func phoneInput() g.Node {
	return Div(
		Div(
			Class("flex items-baseline justify-between mb-1"),
			Label(
				Class("text-base font-medium"),
				For("phone"),
				g.Text("Phone Number"),
			),
			A(
				Href("/faq/phone-number"),
				Class("text-sm text-blue-600 dark:text-blue-400 hover:underline shrink-0"),
				g.Text("Why do you need my phone number?"),
			),
		),
		Input(
			Class(textFieldClass),
			Type("tel"),
			Name("phone"),
			ID("phone"),
			MinLength("10"),
			MaxLength("20"),
			g.Attr("placeholder", "+12025550123 or 202-555-0123"),
			g.Attr("pattern", "^\\+?[\\d\\s\\-\\(\\)\\.]{10,20}$"),
			g.Attr("autocomplete", "tel"),
			Required(),
		),
		Span(
			Class("text-xs text-zinc-500 dark:text-zinc-400 mt-1"),
			g.Text("Enter your phone in international format (e.g. +12025550123) or US/Canada format (e.g. 503-523-8780)."),
		),
	)
}

func offers() g.Node {
	inputAttrs := []g.Node{
		Type("checkbox"),
		Name("offers"),
		Value("true"),
		ID("offers-true"),
		Class("mt-0.5 h-5 w-5 shrink-0 accent-blue-600 cursor-pointer"),
		Required(),
	}
	return Div(
		Class("flex items-start gap-3"),
		Input(inputAttrs...),
		Label(
			For("offers-true"),
			Class("text-sm leading-snug text-zinc-600 dark:text-zinc-300 "+
				"cursor-pointer"),
			g.Text("I agree to the "),
			faqLink("/terms", "Terms of Service"),
			g.Text(" and "),
			faqLink("/privacy", "Privacy Policy"),
			g.Text(", and to receive informational text messages from "+
				config.ServerName+
				". Message frequency varies. Message and data rates "+
				"may apply. Reply STOP to cancel."),
		),
	)
}

func RegisterForm(username string) g.Node {
	return Form(
		Class("space-y-8 mt-8"),
		ID("registerForm"),
		hx.Post("/api/register/step1"),
		hx.Swap("none"),
		userNameInput(true, "username", true, username),
		phoneInput(),
		offers(),
		TurnstileWidget(),
		Div(
			Class("flex items-center gap-4"),
			standardButton(buttonProps{
				Type: "submit",
				Text: "Register",
			}),
			ErrorDiv(""),
		),
	)
}

func RegisterPage(username string) []g.Node {
	return []g.Node{
		pageTitle("Register"),
		RegisterForm(username),
	}
}

func hiddenUserName(username string) g.Node {
	return Input(
		Type("hidden"),
		Name("username"),
		Value(username),
	)
}

func hiddenPhone(phone string) g.Node {
	return Input(
		Type("hidden"),
		Name("phone"),
		Value(phone),
	)
}

func verificationCodeInput() g.Node {
	return Div(
		Class("text-center mb-6 space-y-6 py-4"),
		P(
			Class("text-base leading-relaxed"),
			g.Text("We've sent a verification code to your phone number. "+
				"Please enter the code below to continue."),
		),
		Label(
			Class("block text-base font-medium mb-2"),
			g.Attr("for", "verification-code"),
			g.Text("Verification Code"),
		),
		Input(
			ID("verification-code"),
			Class("max-w-xs w-full p-4 border-2 border-zinc-300 dark:border-zinc-600 rounded-md text-center text-2xl font-mono tracking-widest focus:border-blue-500 dark:focus:border-blue-400 focus:outline-none dark:bg-zinc-800 dark:text-zinc-200 mx-auto block my-2"),
			Type("text"),
			Name("code"),
			g.Attr("autocomplete", "one-time-code"),
			g.Attr("inputmode", "numeric"),
			g.Attr("pattern", "[0-9]*"),
			g.Attr("maxlength", "6"),
			g.Attr("placeholder", "000000"),
			Autofocus(),
		),
	)
}

func passwordInput2() g.Node {
	return labeledPasswordField("Confirm Password", "password2", "new-password", false)
}

func terms() g.Node {
	linkClass := "text-blue-600 dark:text-blue-400 " +
		"hover:text-blue-800 dark:hover:text-blue-300 underline"
	return Div(
		Class("flex items-start gap-3"),
		Input(
			Type("checkbox"),
			Name("terms"),
			Value("accepted"),
			ID("terms-accepted"),
			Class("mt-0.5 h-5 w-5 shrink-0 accent-blue-600 cursor-pointer"),
			Required(),
		),
		Label(
			For("terms-accepted"),
			Class("text-sm leading-snug text-zinc-600 dark:text-zinc-300 cursor-pointer"),
			g.Text("I accept the "),
			A(
				Href("/terms"),
				Class(linkClass),
				g.Text("Terms of Service"),
			),
			g.Text(" & "),
			A(
				Href("/privacy"),
				Class(linkClass),
				g.Text("Privacy Policy"),
			),
			g.Text("."),
		),
	)
}

func RegisterVerify(username, phoneE64 string) g.Node {
	return Form(
		Class("space-y-12"),
		ID("registerForm"),
		hx.Post("/api/register/step2"),
		hx.Swap("none"),
		hx.SwapOOB("true"),
		hiddenUserName(username),
		hiddenPhone(phoneE64),
		verificationCodeInput(),
		Div(
			Class("flex flex-col items-center gap-6"),
			standardButton(buttonProps{
				Type: "submit",
				Text: "Verify Code",
			}),
			ErrorDiv(""),
		),
	)
}

func RegisterPassword(username, phoneE64 string) g.Node {
	return Form(
		Class("space-y-8 mt-8"),
		ID("registerForm"),
		hx.Post("/api/register/step3"),
		hx.Swap("none"),
		hx.SwapOOB("true"),
		hiddenPhone(phoneE64),
		passwordManagerUsername(username),
		Div(
			labeledPasswordField("Password", "password", "new-password", true),
			Span(
				Class("text-xs text-zinc-500 dark:text-zinc-400 mt-1 block"),
				g.Text(password.StrengthRequirements),
			),
		),
		passwordInput2(),
		terms(),
		Div(
			Class("flex items-center gap-4"),
			standardButton(buttonProps{
				Type: "submit",
				Text: "Complete Registration",
			}),
			ErrorDiv(""),
		),
	)
}
