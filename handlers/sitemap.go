package handlers

import (
	"encoding/xml"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SitemapURL struct {
	Loc        string    `xml:"loc"`
	LastMod    time.Time `xml:"lastmod"`
	ChangeFreq string    `xml:"changefreq"`
	Priority   string    `xml:"priority"`
}

type Sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

func HandleSitemap(c *fiber.Ctx) error {
	baseURL := "https://parts-pile.com"

	sitemap := Sitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []SitemapURL{
			{
				Loc:        baseURL + "/",
				LastMod:    time.Now(),
				ChangeFreq: "daily",
				Priority:   "1.0",
			},
			{
				Loc:        baseURL + "/register",
				LastMod:    time.Now(),
				ChangeFreq: "monthly",
				Priority:   "0.6",
			},
			{
				Loc:        baseURL + "/login",
				LastMod:    time.Now(),
				ChangeFreq: "monthly",
				Priority:   "0.5",
			},
			{
				Loc:        baseURL + "/terms",
				LastMod:    time.Now(),
				ChangeFreq: "yearly",
				Priority:   "0.3",
			},
			{
				Loc:        baseURL + "/privacy",
				LastMod:    time.Now(),
				ChangeFreq: "yearly",
				Priority:   "0.3",
			},
		},
	}

	c.Set("Content-Type", "application/xml")
	return c.XML(sitemap)
}
