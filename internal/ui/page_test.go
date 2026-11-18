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
		`name="description" content="Parts Pile is a marketplace for things.`,
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

func TestAboutIntro(t *testing.T) {
	var buf bytes.Buffer
	n := Page(0, false, "", "About", "/about", "csrf", false, AboutPage())
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	wants := []string{
		"is an online marketplace to buy, sell, and trade.",
		"brand-metal brand-metal-control mx-6 space-y-3 rounded-lg",
		"Cars &amp; Trucks and Parts",
		"/images/category/car.svg",
		"Motorcycles and Parts",
		"Bicycles and Parts",
		"Ag Equipment and Parts",
		"Posting is free.",
		"don&#39;t sell your information or share it with " +
			"third parties for their own use.",
		"A phone number is all we ask for.",
		"We keep your number private.",
		"Parts Pile, LLC",
		"Get Started",
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("about page missing %q", want)
		}
	}
	intro := strings.Index(html, "buy, sell, and trade.")
	cats := strings.Index(html, "Cars &amp; Trucks and Parts")
	free := strings.Index(html, "Posting is free.")
	sell := strings.Index(html, "third parties for their own use.")
	phone := strings.Index(html, "A phone number is all we ask for.")
	start := strings.Index(html, "Get Started")
	if intro < 0 || cats < intro || free < cats || sell < free ||
		phone < sell || start < phone {
		t.Fatal("about intro is out of order")
	}
	for _, gone := range []string{
		"online marketplace for",
		"Buy - Sell - Trade",
		"marketplace for things",
		"Here&#39;s how it works",
		"Bob posts an ad",
		"Buyers and sellers message here.",
		"share or sell your information.",
		"stays private, and we don&#39;t sell it.",
		"that number stays with us.",
	} {
		if strings.Contains(html, gone) {
			t.Errorf("about page still has %q", gone)
		}
	}
}

func TestLegalEntityNamed(t *testing.T) {
	const want = "Parts Pile, LLC"
	for _, page := range []struct {
		name string
		html string
	}{
		{"privacy", renderNodes(t, PrivacyPolicyPage())},
		{"terms", renderNodes(t, TermsOfServicePage())},
	} {
		if !strings.Contains(page.html, want) {
			t.Errorf("%s missing %q", page.name, want)
		}
	}
	privacy := renderNodes(t, PrivacyPolicyPage())
	if !strings.Contains(privacy, "operates") {
		t.Fatal("privacy should name the operator")
	}
	if strings.Contains(privacy, "under 13") {
		t.Fatal("privacy should not set a lower age than the terms")
	}
	if !strings.Contains(privacy, "18 and older") {
		t.Fatal("privacy should match the 18-year minimum")
	}
	terms := renderNodes(t, TermsOfServicePage())
	if !strings.Contains(terms, "at least 18 years old") {
		t.Fatal("terms should require 18")
	}
}
