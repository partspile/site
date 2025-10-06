package ui

import (
	"github.com/parts-pile/site/user"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func AboutPage(currentUser *user.User, path string) g.Node {
	return Page(
		"About",
		currentUser,
		path,
		[]g.Node{
			pageHeader("About Parts Pile"),
			contentContainer(
				Div(
					Class("prose max-w-none"),
					H2(Class("text-xl font-semibold mb-4"), g.Text("About Parts Pile")),
					P(Class("mb-4"), g.Text("Parts Pile is a marketplace platform designed to help users buy and sell automotive parts and accessories.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("What We Do")),
					P(Class("mb-4"), g.Text("We provide a simple, secure platform where users can:")),
					Ul(Class("ml-4 mb-4 space-y-2"),
						Li(g.Text("• List automotive parts and accessories for sale")),
						Li(g.Text("• Search for specific parts by make, model, and category")),
						Li(g.Text("• Connect with other users through our messaging system")),
						Li(g.Text("• Bookmark interesting listings")),
						Li(g.Text("• View listings in multiple formats: grid, list, map, and tree views")),
					),

					H3(Class("text-lg font-semibold mb-2"), g.Text("Our Mission")),
					P(Class("mb-4"), g.Text("We believe in creating a community-driven marketplace that makes it easy to find and sell automotive parts while maintaining user privacy and security.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("Privacy First")),
					P(Class("mb-4"), g.Text("We collect only the minimal information necessary to operate our service. We don't require real names, addresses, or payment information. ")),
					A(
						Href("/privacy"),
						Class("text-blue-600 hover:text-blue-800 underline"),
						g.Text("Learn more about our privacy practices"),
					),
					g.Text("."),

					H3(Class("text-lg font-semibold mb-2"), g.Text("Getting Started")),
					P(Class("mb-4"), g.Text("To get started, simply register with a username and phone number. Once verified, you can start browsing listings or create your own.")),

					H3(Class("text-lg font-semibold mb-2"), g.Text("Contact Us")),
					P(Class("mb-4"), g.Text("If you have any questions or need help, please contact us through our website or reach out to our support team.")),
				),
			),
		},
	)
}
