package handler

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
)

func TestRobotsTxt(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	body := robotsTxt()
	for _, want := range []string{
		"User-agent: *",
		"Allow: /c/",
		"Disallow: /auth/",
		"Disallow: /admin/",
		"Disallow: /api/",
		"Disallow: /u/",
		"Sitemap: https://parts-pile.com/sitemap.xml",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("robots.txt missing %q\n%s", want, body)
		}
	}
}

func TestSitemapXML(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	body := sitemapXML()
	wants := []string{
		"https://parts-pile.com/",
		"https://parts-pile.com/about",
		"https://parts-pile.com/login",
	}
	for _, want := range wants {
		if !strings.Contains(body, "<loc>"+want+"</loc>") {
			t.Errorf("sitemap missing %q\n%s", want, body)
		}
	}
	for _, extra := range []string{"/faq", "/privacy", "/terms", "/register", "/u/"} {
		if strings.Contains(body, extra) {
			t.Errorf("sitemap should not include %q\n%s", extra, body)
		}
	}
}

func TestRobotsAndSitemapHandlers(t *testing.T) {
	prev := config.PublicSiteURL
	config.PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { config.PublicSiteURL = prev })

	app := fiber.New()
	app.Get("/robots.txt", RobotsHandler)
	app.Get("/sitemap.xml", SitemapHandler)

	resp, err := app.Test(httptest.NewRequest("GET", "/robots.txt", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("robots status %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "Sitemap:") {
		t.Fatalf("robots body: %s", b)
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/sitemap.xml", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("sitemap status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "xml") {
		t.Fatalf("sitemap content-type %q", ct)
	}
}
