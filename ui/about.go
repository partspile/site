package ui

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func AboutPage(userID int, userName string, path string) g.Node {
	return Page(
		"About",
		userID,
		userName,
		path,
		[]g.Node{
			pageHeader("About Parts Pile"),
			contentContainer(
				Div(
					Class("prose max-w-none"),
					H2(Class("text-xl font-semibold mb-4"), g.Text("About Parts Pile")),
					P(Class("mb-4"), g.Text("Parts Pile is for buying and selling automotive parts.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("Privacy First")),
					P(Class("mb-4"), g.Text("We collect only the minimal information necessary to operate our service. We don't require real names, addresses, or payment information. ")),
					A(
						Href("/privacy"),
						Class("text-blue-600 hover:text-blue-800 underline"),
						g.Text("Learn more about our privacy practices"),
					),
					g.Text("."),
				),
			),
		},
	)
}
