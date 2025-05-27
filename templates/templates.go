package templates

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func Page(title string, content []g.Node) g.Node {
	return HTML(
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Title(title),
			Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css")),
			Script(Src("https://unpkg.com/htmx.org@1.9.10")),
		),
		Body(
			Div(
				Class("container mx-auto px-4 py-8"),
				g.Group(content),
			),
		),
	)
}

func ValidationError(message string) g.Node {
	return Div(
		Class("bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded"),
		g.Text(message),
	)
}

func SuccessMessage(message string, redirectScript string) g.Node {
	return Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Text(message),
		Script(g.Raw(redirectScript)),
	)
}
