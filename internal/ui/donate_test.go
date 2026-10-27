package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/partspile/site/internal/config"
	g "maragu.dev/gomponents"
)

func renderNodes(t *testing.T, nodes []g.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := g.Group(nodes).Render(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestAboutPageDonateItem(t *testing.T) {
	html := renderNodes(t, AboutPage())
	idxSource := strings.Index(html, "Source code")
	idxFunStats := strings.Index(html, "Fun Stats")
	idxDonate := strings.Index(html, "Donate")
	if idxSource < 0 || idxFunStats < 0 || idxDonate < 0 {
		t.Fatal("expected Source code, Fun Stats, and Donate on about page")
	}
	if idxFunStats < idxSource {
		t.Fatal("Fun Stats should follow Source code")
	}
	if idxDonate < idxFunStats {
		t.Fatal("Donate should follow Fun Stats")
	}
	for _, want := range []string{
		`href="/funstats"`,
		"/images/bar_chart.svg",
		"Users and ads over time",
		`href="/donate"`,
		"/images/money.svg",
		"Help with operating costs",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("about missing %q", want)
		}
	}
}

func TestDonatePageBitcoin(t *testing.T) {
	prev := config.BitcoinDonateAddress
	addr := "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
	config.BitcoinDonateAddress = addr
	t.Cleanup(func() { config.BitcoinDonateAddress = prev })

	html := renderNodes(t, DonatePage())
	for _, want := range []string{
		"publicly funded",
		"The site runs on donations to cover hosting, SMS, and other " +
			"operating costs.",
		"Thank you for your donation.",
		"Bitcoin",
		"only payment we accept",
		addr,
		"/images/money.svg",
		"Copy address",
		"data:image/png;base64,",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("donate missing %q", want)
		}
	}
}

func TestDonatePageOmitsAddressWhenUnset(t *testing.T) {
	prev := config.BitcoinDonateAddress
	config.BitcoinDonateAddress = ""
	t.Cleanup(func() { config.BitcoinDonateAddress = prev })

	html := renderNodes(t, DonatePage())
	if !strings.Contains(html, "publicly funded") {
		t.Fatal("expected funding copy")
	}
	if !strings.Contains(html, "Bitcoin") {
		t.Fatal("expected Bitcoin as the donation method")
	}
	if strings.Contains(html, "Copy address") {
		t.Fatal("should omit copy controls without an address")
	}
	if strings.Contains(html, "data:image/png;base64,") {
		t.Fatal("should omit QR without an address")
	}
}
