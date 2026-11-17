package ui

import (
	"github.com/partspile/site/internal/config"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

const turnstileProbeResultID = "turnstile-probe-result"

func AdminTurnstileTab() g.Node {
	return Div(
		ID("turnstile-tab"),
		Class("mt-4 space-y-4"),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Submits a Turnstile token to Cloudflare and nothing else. "+
				"No SMS is sent and no account data changes."),
		),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Open this tab on a hostname listed on the widget. "+
				"The check still runs when test registration hides the "+
				"register and change-phone widgets."),
		),
		Form(
			ID("turnstile-probe-form"),
			Class("space-y-4"),
			hx.Post("/admin/turnstile"),
			hx.Swap("none"),
			hx.On("after-request",
				"try { if (window.turnstile) turnstile.reset() } catch (e) {}"),
			turnstileProbeWidget(),
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
		turnstileProbeMountScript(),
	)
}

func turnstileProbeWidget() g.Node {
	if config.TurnstileSiteKey == "" {
		return nil
	}
	return Div(
		Class("my-4"),
		Div(
			ID("admin-turnstile-widget"),
			Class("cf-turnstile"),
			g.Attr("data-sitekey", config.TurnstileSiteKey),
			g.Attr("data-action", "admin-check"),
			g.Attr("data-theme", "auto"),
			g.Attr("data-size", "flexible"),
		),
	)
}

// The tab is inserted by HTMX, so the widget is rendered explicitly after
// the swap instead of waiting for a full page load.
func turnstileProbeMountScript() g.Node {
	if config.TurnstileSiteKey == "" {
		return nil
	}
	return Script(g.Raw(`(function () {
  var el = document.getElementById("admin-turnstile-widget");
  if (!el) return;
  function mount() {
    if (!window.turnstile || el.dataset.rendered || !el.isConnected) return;
    el.dataset.rendered = "1";
    turnstile.render(el, {
      sitekey: el.getAttribute("data-sitekey"),
      action: el.getAttribute("data-action"),
      theme: "auto",
      size: "flexible"
    });
  }
  if (window.turnstile) { mount(); return; }
  var s = document.createElement("script");
  s.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
  s.async = true;
  s.onload = mount;
  document.head.appendChild(s);
})();`))
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
