package ui

import (
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/service/turnstile"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TurnstileWidget renders Cloudflare Turnstile when keys are configured and
// enforcement is required (not under ALLOW_TEST_REGISTRATION).
func TurnstileWidget() g.Node {
	if !turnstile.Required() || config.TurnstileSiteKey == "" {
		return nil
	}
	return Div(
		Class("my-4"),
		Script(
			Src("https://challenges.cloudflare.com/turnstile/v0/api.js"),
			Async(),
			Defer(),
		),
		Div(
			Class("cf-turnstile"),
			g.Attr("data-sitekey", config.TurnstileSiteKey),
			g.Attr("data-theme", "auto"),
			g.Attr("data-size", "flexible"),
		),
	)
}
