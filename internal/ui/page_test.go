package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/partspile/site/internal/config"
)

func renderPageHTML(t *testing.T, title, path string) string {
	t.Helper()
	var buf bytes.Buffer
	n := Page(0, false, "", title, path, "csrf", false, nil)
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestPageHomeSEO(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	html := renderPageHTML(t, "Car & Truck Parts", "/")
	wants := []string{
		"<title>Parts Pile - Car &amp; Truck Parts</title>",
		`rel="canonical" href="https://parts-pile.com/"`,
		`name="description" content="Parts Pile - Classified Ads.`,
		`property="og:title" content="Parts Pile - Car &amp; Truck Parts"`,
		`"@type":"WebSite"`,
		`"url":"https://parts-pile.com/"`,
		`name="robots" content="index, follow"`,
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("homepage missing %q", want)
		}
	}
}

func TestPageLoginSEO(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	html := renderPageHTML(t, "Login", "/login")
	wants := []string{
		"<title>Parts Pile - Login</title>",
		`rel="canonical" href="https://parts-pile.com/login"`,
		`name="description" content="Log in to Parts Pile`,
		`name="robots" content="index, follow"`,
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("login missing %q", want)
		}
	}
	if strings.Contains(html, "application/ld+json") {
		t.Error("WebSite JSON-LD should only be on the homepage")
	}
}

func TestPageCategoryLandingSEO(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	html := renderPageHTML(t, "Cars & Trucks", "/c/6")
	wants := []string{
		"<title>Parts Pile - Cars &amp; Trucks</title>",
		`rel="canonical" href="https://parts-pile.com/c/6"`,
		`property="og:url" content="https://parts-pile.com/c/6"`,
		`name="robots" content="index, follow"`,
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("category landing missing %q", want)
		}
	}
	if strings.Contains(html, `"@type":"WebSite"`) {
		t.Error("WebSite JSON-LD should only be on /")
	}
}

func TestPageSharedProfileSEO(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	html := renderPageHTML(t, "test", "/u/AAAAAAAAAAAAAAAAAAAAAA")
	wants := []string{
		`name="robots" content="noindex, nofollow"`,
		`rel="canonical" href="https://parts-pile.com/u/AAAAAAAAAAAAAAAAAAAAAA"`,
		`property="og:url" content="https://parts-pile.com/u/AAAAAAAAAAAAAAAAAAAAAA"`,
		`property="og:type" content="profile"`,
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("shared profile missing %q", want)
		}
	}
}

func TestRobotsForPath(t *testing.T) {
	if got := robotsForPath("/auth/user/settings"); got != "noindex, nofollow" {
		t.Fatalf("auth: %q", got)
	}
	if got := robotsForPath("/u/abc"); got != "noindex, nofollow" {
		t.Fatalf("share profile: %q", got)
	}
	if got := robotsForPath("/about"); got != "index, follow" {
		t.Fatalf("about: %q", got)
	}
	if got := robotsForPath("/faq"); got != "index, follow" {
		t.Fatalf("faq: %q", got)
	}
	if got := robotsForPath("/donate"); got != "index, follow" {
		t.Fatalf("donate: %q", got)
	}
	if got := robotsForPath("/funstats"); got != "index, follow" {
		t.Fatalf("funstats: %q", got)
	}
}
