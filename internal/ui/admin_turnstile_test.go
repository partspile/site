package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/partspile/site/internal/config"
)

func TestAdminTurnstilePageRendersProbe(t *testing.T) {
	origAllow := config.AllowTestRegistration
	origKey := config.TurnstileSiteKey
	config.AllowTestRegistration = true
	config.TurnstileSiteKey = "0x4AAAAAAEFpEa4IkHW7HWg3"
	t.Cleanup(func() {
		config.AllowTestRegistration = origAllow
		config.TurnstileSiteKey = origKey
	})

	var buf bytes.Buffer
	for _, n := range AdminTurnstilePage() {
		if err := n.Render(&buf); err != nil {
			t.Fatal(err)
		}
	}
	html := buf.String()
	for _, want := range []string{
		`hx-post="/admin/turnstile"`,
		`class="cf-turnstile"`,
		`data-sitekey="0x4AAAAAAEFpEa4IkHW7HWg3"`,
		`data-action="admin-check"`,
		`id="turnstile-probe-result"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("page missing %q", want)
		}
	}
}
