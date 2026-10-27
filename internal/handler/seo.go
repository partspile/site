package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
)

func RobotsHandler(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return c.SendString(robotsTxt())
}

func SitemapHandler(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "application/xml; charset=utf-8")
	return c.SendString(sitemapXML())
}

func robotsTxt() string {
	return "User-agent: *\n" +
		"Allow: /\n" +
		"Allow: /c/\n" +
		"Disallow: /auth/\n" +
		"Disallow: /admin/\n" +
		"Disallow: /api/\n" +
		"Disallow: /u/\n" +
		"\n" +
		"Sitemap: " + config.CanonicalURL("/sitemap.xml") + "\n"
}

func sitemapPaths() []string {
	paths := []string{"/", "/about", "/login"}
	cats, err := ad.GetCategories()
	if err != nil {
		return paths
	}
	defaultID := ad.DefaultCategoryID()
	for _, cat := range cats {
		if cat.ID == defaultID {
			continue
		}
		paths = append(paths, "/c/"+strconv.Itoa(cat.ID))
	}
	return paths
}

func sitemapXML() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteByte('\n')
	b.WriteString(
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	b.WriteByte('\n')
	for _, path := range sitemapPaths() {
		b.WriteString("  <url><loc>")
		b.WriteString(xmlEscape(config.CanonicalURL(path)))
		b.WriteString("</loc></url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
