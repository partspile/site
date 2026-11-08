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

func TestNavBackHome(t *testing.T) {
	home := renderPageHTML(t, "Car & Truck Parts", "/")
	if strings.Contains(home, `id="nav-back-home"`) {
		t.Error("home page should not show the back-to-home link")
	}

	about := renderPageHTML(t, "About", "/about")
	wants := []string{
		`id="nav-back-home"`,
		`href="/"`,
		`aria-label="Home"`,
		`src="/images/left.svg"`,
		`absolute left-4`,
		`-translate-x-full`,
	}
	for _, want := range wants {
		if !strings.Contains(about, want) {
			t.Errorf("about page missing %q", want)
		}
	}
}

func TestNavNewAdLink(t *testing.T) {
	loggedOut := renderPageHTML(t, "About", "/about")
	if strings.Contains(loggedOut, "/images/post_add.svg") {
		t.Error("new ad link should not appear when logged out")
	}

	var buf bytes.Buffer
	n := Page(7, false, "Sam", "About", "/about", "csrf", false, nil)
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	newAt := strings.Index(html, `/images/post_add.svg`)
	avatarAt := strings.Index(html, `id="user-avatar"`)
	if newAt < 0 || !strings.Contains(html, `href="/auth/ad/new"`) {
		t.Fatal("expected new ad link in the navbar when logged in")
	}
	if avatarAt < 0 || newAt > avatarAt {
		t.Fatal("new ad link should sit left of the user icon")
	}

	for _, path := range []string{"/auth/ad/new", "/auth/ad/12/copy"} {
		var hidden bytes.Buffer
		page := Page(7, false, "Sam", "New Ad", path, "csrf", false, nil)
		if err := page.Render(&hidden); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(hidden.String(), "/images/post_add.svg") {
			t.Errorf("new ad link should be hidden on %s", path)
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
