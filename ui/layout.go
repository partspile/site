package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/user"
)

// ---- Page Layout ----

func Page(title string, currentUser *user.User, currentPath string, content []g.Node) g.Node {
	return components.HTML5(components.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []g.Node{
			Link(Rel("icon"), Type("image/png"), Href("/favicon-32x32.png"), g.Attr("sizes", "32x32")),
			Link(Rel("icon"), Type("image/png"), Href("/favicon-16x16.png"), g.Attr("sizes", "16x16")),
			Link(
				Rel("stylesheet"),
				Href("https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css"),
			),
			Script(
				Type("text/javascript"),
				Src("https://unpkg.com/htmx.org@2.0.4"),
				Defer(),
			),
			Script(
				Type("text/javascript"),
				g.Raw(`
					document.addEventListener('htmx:load', function() {
						console.log('HTMX loaded');
					});
					document.addEventListener('htmx:beforeRequest', function(evt) {
						console.log('HTMX request:', evt.detail.path);
					});
				`),
			),
			// Script(
			// 	Type("text/javascript"),
			// 	g.Raw("if(window.htmx){htmx.logAll()} else {document.addEventListener('htmx:load',function(){htmx.logAll()})}"),
			// ),
		},
		Body: []g.Node{
			Div(
				Class("container mx-auto px-4 py-8"),
				hx.Headers(`js:{'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone}`),
				Div(
					Class("mb-8 border-b pb-4 flex items-center justify-between"),
					UserNav(currentUser, currentPath),
				),
				g.Group(content),
				ResultContainer(),
			),
		},
	})
}

func PageHeader(text string) g.Node {
	return H1(Class("text-4xl font-bold mb-8"), g.Text(text))
}
