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
	return turnstileWidget(false, "")
}

// TurnstileProbeWidget renders the widget even when test registration skips
// the register and change-phone gates. action is sent as data-action.
func TurnstileProbeWidget(action string) g.Node {
	return turnstileWidget(true, action)
}

func turnstileWidget(always bool, action string) g.Node {
	if config.TurnstileSiteKey == "" {
		return nil
	}
	if !always && !turnstile.Required() {
		return nil
	}
	widget := []g.Node{
		Class("cf-turnstile"),
		g.Attr("data-sitekey", config.TurnstileSiteKey),
		g.Attr("data-theme", "auto"),
		g.Attr("data-size", "flexible"),
	}
	if action != "" {
		widget = append(widget, g.Attr("data-action", action))
	}
	return Div(
		Class("my-4"),
		Script(
			Src("https://challenges.cloudflare.com/turnstile/v0/api.js"),
			Async(),
			Defer(),
		),
		Div(widget...),
	)
}
