package ui

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const (
	aboutSkin = "#f0c7a4"
	aboutHair = "#3a312b"
	aboutInk  = "#1c1917"
)

func aboutHowItWorks() g.Node {
	return Div(
		Class("space-y-3"),
		H2(
			Class("text-xl font-semibold text-zinc-900 dark:text-zinc-100"),
			g.Text("Here's how it works"),
		),
		Div(
			Class("rounded-lg border border-zinc-200 dark:border-zinc-700 "+
				"p-4 sm:p-5 space-y-5"),
			Div(
				Class("grid gap-6 sm:grid-cols-2"),
				aboutHowPost(),
				aboutHowInquire(),
			),
			aboutHowThread(),
		),
	)
}

func aboutHowPost() g.Node {
	return Div(
		Class("flex items-center gap-3"),
		aboutCartoon(false, "#c49a4a"),
		Div(
			Class("min-w-0 space-y-2"),
			P(
				Class("text-sm font-medium text-zinc-800 dark:text-zinc-200"),
				g.Text("Bob posts an ad"),
			),
			Div(
				Class("rounded-md border border-zinc-200 "+
					"dark:border-zinc-600 bg-white dark:bg-zinc-900 "+
					"px-3 py-2"),
				P(
					Class("font-medium text-zinc-900 dark:text-zinc-100"),
					g.Text("Truck tailgate"),
				),
				P(
					Class("text-sm text-zinc-600 dark:text-zinc-300"),
					g.Text("$120"),
				),
			),
		),
	)
}

func aboutHowInquire() g.Node {
	return Div(
		Class("flex items-center gap-3"),
		aboutCartoon(true, "#3b82f6"),
		Div(
			Class("min-w-0 space-y-2"),
			P(
				Class("text-sm font-medium text-zinc-800 dark:text-zinc-200"),
				g.Text("Alice messages Bob"),
			),
			Div(
				Class("rounded-lg rounded-tl-none bg-blue-500 text-white "+
					"px-3 py-2 text-sm"),
				g.Text("Is the tailgate still available?"),
			),
		),
	)
}

func aboutHowThread() g.Node {
	return Div(
		Class("space-y-3 border-t border-zinc-200 dark:border-zinc-700 pt-4"),
		aboutHowBubble("Alice", "Is the tailgate still available?", true),
		aboutHowBubble("Bob", "Yes. Saturday morning works.", false),
		aboutHiddenNumbers(),
	)
}

func aboutHowBubble(name, text string, fromAlice bool) g.Node {
	row := "flex flex-col items-end gap-1"
	bubble := "bg-zinc-200 dark:bg-zinc-700 text-zinc-900 " +
		"dark:text-zinc-100 rounded-lg rounded-tr-none"
	if fromAlice {
		row = "flex flex-col items-start gap-1"
		bubble = "bg-blue-500 text-white rounded-lg rounded-tl-none"
	}
	return Div(
		Class(row),
		Span(
			Class("text-xs font-medium text-zinc-500 dark:text-zinc-400 px-1"),
			g.Text(name),
		),
		Div(
			Class("max-w-[85%] px-3 py-2 text-sm "+bubble),
			g.Text(text),
		),
	)
}

func aboutHiddenNumbers() g.Node {
	return Div(
		Class("flex flex-col sm:flex-row sm:items-center gap-3 "+
			"rounded-md bg-zinc-100 dark:bg-zinc-800 px-3 py-3"),
		Div(
			Class("shrink-0 space-y-1 font-mono text-sm "+
				"text-zinc-800 dark:text-zinc-200"),
			aboutMaskedPhone("Bob"),
			aboutMaskedPhone("Alice"),
		),
		P(
			Class("text-sm text-zinc-700 dark:text-zinc-300"),
			g.Text("Phone numbers stay hidden. They message here, "+
				"and neither one sees the other's number."),
		),
	)
}

func aboutMaskedPhone(name string) g.Node {
	return Div(
		Class("flex items-center gap-2"),
		aboutPhoneSlash(),
		Span(g.Text(name)),
		Span(
			Class("tracking-wider text-zinc-500 dark:text-zinc-400"),
			g.Text("••• ••• ••••"),
		),
	)
}

func aboutPhoneSlash() g.Node {
	return g.El("svg",
		g.Attr("viewBox", "0 0 24 24"),
		g.Attr("aria-hidden", "true"),
		Class("w-4 h-4 shrink-0"),
		g.El("rect",
			g.Attr("x", "7"),
			g.Attr("y", "2"),
			g.Attr("width", "10"),
			g.Attr("height", "20"),
			g.Attr("rx", "2"),
			g.Attr("fill", "none"),
			g.Attr("stroke", "currentColor"),
			g.Attr("stroke-width", "1.6"),
		),
		g.El("line",
			g.Attr("x1", "5"),
			g.Attr("y1", "19"),
			g.Attr("x2", "19"),
			g.Attr("y2", "5"),
			g.Attr("stroke", "currentColor"),
			g.Attr("stroke-width", "1.6"),
			g.Attr("stroke-linecap", "round"),
		),
	)
}

func aboutCartoon(longHair bool, shirt string) g.Node {
	nodes := []g.Node{
		g.Attr("viewBox", "0 0 96 112"),
		g.Attr("aria-hidden", "true"),
		Class("w-16 h-auto shrink-0"),
	}
	if longHair {
		nodes = append(nodes, aboutShape("path", aboutHair,
			g.Attr("d", "M30 50 C14 46 16 18 48 16 "+
				"C80 18 82 46 66 50 L70 96 "+
				"C60 104 36 104 26 96 Z"),
		))
	}
	nodes = append(nodes,
		aboutShape("path", shirt,
			g.Attr("d", "M18 112 L18 80 Q18 70 34 68 "+
				"L62 68 Q78 70 78 80 L78 112 Z"),
		),
		aboutShape("rect", aboutSkin,
			g.Attr("x", "42"),
			g.Attr("y", "62"),
			g.Attr("width", "12"),
			g.Attr("height", "14"),
			g.Attr("rx", "4"),
		),
		aboutShape("circle", aboutSkin,
			g.Attr("cx", "48"),
			g.Attr("cy", "42"),
			g.Attr("r", "22"),
		),
	)
	if longHair {
		nodes = append(nodes, aboutShape("path", aboutHair,
			g.Attr("d", "M28 42 C30 22 66 22 68 42 "+
				"C62 34 34 34 28 42 Z"),
		))
	} else {
		nodes = append(nodes, aboutShape("path", aboutHair,
			g.Attr("d", "M28 40 C28 18 68 18 68 40 "+
				"C64 32 32 32 28 40 Z"),
		))
	}
	nodes = append(nodes,
		aboutShape("circle", aboutInk,
			g.Attr("cx", "40"),
			g.Attr("cy", "44"),
			g.Attr("r", "2.2"),
		),
		aboutShape("circle", aboutInk,
			g.Attr("cx", "56"),
			g.Attr("cy", "44"),
			g.Attr("r", "2.2"),
		),
		g.El("path",
			g.Attr("d", "M40 52 Q48 58 56 52"),
			g.Attr("fill", "none"),
			g.Attr("stroke", aboutInk),
			g.Attr("stroke-width", "1.6"),
			g.Attr("stroke-linecap", "round"),
		),
	)
	return g.El("svg", nodes...)
}

func aboutShape(tag, fill string, attrs ...g.Node) g.Node {
	nodes := []g.Node{g.Attr("fill", fill)}
	nodes = append(nodes, attrs...)
	return g.El(tag, nodes...)
}
