package ui

import (
	"github.com/partspile/site/internal/config"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func welcomeRocks(count int) g.Node {
	var rocks []g.Node
	for range count {
		rocks = append(rocks,
			Img(
				Src("/images/rock.svg"),
				Alt("Rock"),
				Class("w-full max-w-[80px]"),
			),
		)
	}
	return Div(
		Class("flex items-end justify-center gap-3"),
		g.Group(rocks),
	)
}

func WelcomePage(rockCount int) []g.Node {
	return []g.Node{
		Div(
			Class("bg-gradient-to-br from-blue-50 to-indigo-50 "+
				"dark:from-blue-900/30 dark:to-indigo-900/30 "+
				"border border-blue-200 dark:border-blue-800 "+
				"rounded-xl p-5 md:p-8 shadow-lg"),
			Div(
				Class("flex flex-col items-center text-center space-y-4"),
				Form(
					Class("flex flex-col items-center text-center space-y-4 w-full"),
					Method("get"),
					Action("/"),
					Div(
						Class("space-y-2"),
						H1(
							Class("text-2xl font-bold "+
								"text-zinc-900 dark:text-zinc-100"),
							g.Textf("Welcome to %s!", config.ServerName),
						),
						P(
							Class("text-base text-zinc-700 dark:text-zinc-300"),
							g.Textf("Here are your %d rocks", rockCount),
						),
					),
					welcomeRocks(rockCount),
					P(
						Class("text-base text-zinc-600 dark:text-zinc-400 leading-snug"),
						faqLink("/faq/rocks", "What are the rocks for?"),
					),
					standardButton(buttonProps{
						Type:  "submit",
						Text:  "Go To Ads",
						Class: "font-semibold py-3",
						Attrs: []g.Node{Autofocus()},
					}),
				),
			),
		),
	}
}
