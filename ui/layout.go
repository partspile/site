package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/user"
)

// ---- Page Layout ----

func Page(title string, currentUser *user.User, currentPath string, content []g.Node) g.Node {
	return HTML(
		Head(
			g.Raw(`<title>`+title+`</title>`),
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(
				Rel("stylesheet"),
				Href("https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css"),
			),
			Script(
				Type("text/javascript"),
				Src("https://unpkg.com/htmx.org@2.0.4"),
				Defer(),
			),
		),
		Body(
			Div(
				Class("container mx-auto px-4 py-8"),
				hx.Headers(`js:{'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone}`),
				Div(
					Class("mb-8 border-b pb-4 flex items-center justify-between"),
					UserNav(currentUser, currentPath),
				),
				g.Group(content),
			),
		),
	)
}

func PageHeader(text string) g.Node {
	return H1(Class("text-4xl font-bold mb-8"), g.Text(text))
}
