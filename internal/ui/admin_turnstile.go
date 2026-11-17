package ui

import (
	"github.com/partspile/site/internal/config"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

const turnstileProbeResultID = "turnstile-probe-result"

func AdminTurnstilePage() []g.Node {
	return []g.Node{
		pageTitle("Turnstile check"),
		P(
			Class("mt-4 text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Submits a Turnstile token to Cloudflare and nothing else. "+
				"No SMS is sent and no account data changes."),
		),
		P(
			Class("mt-2 text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Open this page on a hostname listed on the widget. "+
				"The check still runs when test registration hides the "+
				"register and change-phone widgets."),
		),
		Form(
			ID("turnstile-probe-form"),
			Class("space-y-4 mt-6"),
			hx.Post("/admin/turnstile"),
			hx.Swap("none"),
			hx.On("after-request",
				"try { if (window.turnstile) turnstile.reset() } catch (e) {}"),
			TurnstileProbeWidget("admin-check"),
			g.If(config.TurnstileSiteKey == "",
				P(
					Class("text-red-500"),
					g.Text("TURNSTILE_SITE_KEY is not set."),
				),
			),
			Div(
				Class("flex items-center gap-4"),
				standardButton(buttonProps{
					Type: "submit",
					Text: "Check",
				}),
				turnstileProbeResult("", false),
			),
		),
	}
}

func AdminTurnstileResult(ok bool, msg string) g.Node {
	return turnstileProbeResult(msg, ok)
}

func turnstileProbeResult(msg string, ok bool) g.Node {
	class := "text-red-500 text-lg"
	if ok {
		class = "text-green-600 dark:text-green-400 text-lg"
	}
	return Div(
		ID(turnstileProbeResultID),
		Class(class),
		g.If(msg != "", hx.SwapOOB("true")),
		g.If(msg != "", g.Text(msg)),
	)
}
