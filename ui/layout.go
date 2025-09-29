package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/user"
)

// ---- Page Layout ----

// seoMetaTags returns SEO meta tags for the homepage
func seoMetaTags(title, currentPath string) []g.Node {
	return []g.Node{
		Meta(Name("description"), Content("Parts Pile - The ultimate automotive parts marketplace. Buy and sell car parts by make, model, year, and engine. Find exactly what you need for your vehicle.")),
		Meta(Name("keywords"), Content("auto parts, car parts, automotive parts, vehicle parts, parts marketplace, car parts for sale, auto parts classifieds")),
		Meta(Name("author"), Content("Parts Pile")),
		Meta(Name("robots"), Content("index, follow")),
		Meta(g.Attr("property", "og:title"), Content(title)),
		Meta(g.Attr("property", "og:description"), Content("Parts Pile - The ultimate automotive parts marketplace. Buy and sell car parts by make, model, year, and engine.")),
		Meta(g.Attr("property", "og:type"), Content("website")),
		Meta(g.Attr("property", "og:url"), Content("https://parts-pile.com"+currentPath)),
		Meta(g.Attr("property", "og:site_name"), Content("Parts Pile")),
		Meta(Name("twitter:card"), Content("summary")),
		Meta(Name("twitter:title"), Content(title)),
		Meta(Name("twitter:description"), Content("Parts Pile - The ultimate automotive parts marketplace. Buy and sell car parts by make, model, year, and engine.")),
	}
}

func Page(title string, currentUser *user.User, currentPath string, content []g.Node) g.Node {
	return components.HTML5(components.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []g.Node{
			// SEO Meta Tags (only on homepage)
			g.If(currentPath == "/", g.Group(seoMetaTags(title, currentPath))),

			// Favicons
			Link(Rel("icon"), Type("image/png"), Href("/images/favicon-32x32.png"), g.Attr("sizes", "32x32")),
			Link(Rel("icon"), Type("image/png"), Href("/images/favicon-16x16.png"), g.Attr("sizes", "16x16")),

			// Stylesheets
			Link(
				Rel("stylesheet"),
				Href(config.TailwindCSSURL),
			),
			Script(
				Type("text/javascript"),
				Src(config.HTMXURL),
				Defer(),
			),
			Script(
				Type("text/javascript"),
				Src(config.HTMXSSEURL),
				Defer(),
			),
			// Map resources
			Link(
				Rel("stylesheet"),
				Href(config.LeafletCSSURL),
			),
			Script(
				Type("text/javascript"),
				Src(config.LeafletJSURL),
				Defer(),
			),
			Script(
				Type("text/javascript"),
				Src("/js/map.js"),
				Defer(),
			),
			// Global loading indicator
			Script(
				Type("text/javascript"),
				Src("/js/global-indicator.js"),
				Defer(),
			),
			// Script(
			// 	Type("text/javascript"),
			// 	g.Raw("if(window.htmx){htmx.logAll()} else {document.addEventListener('htmx:load',function(){htmx.logAll()})}"),
			// ),
		},
		Body: []g.Node{
			Div(
				Class("container mx-auto px-4 py-8"),
				hx.Ext("global-indicator"),
				hx.Headers(`js:{'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone}`),
				navigation(currentUser, currentPath),
				g.Group(content),
			),
		},
	})
}

func pageHeader(text string) g.Node {
	return H1(Class("text-4xl font-bold mb-8"), g.Text(text))
}
