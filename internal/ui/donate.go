package ui

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func DonatePage() []g.Node {
	return []g.Node{
		pageTitle("Donate"),
		P(
			Class("mt-4 text-xl font-medium text-zinc-800 dark:text-zinc-200"),
			g.Textf("%s is publicly funded", config.ServerName),
		),
		Div(
			Class("mt-10 space-y-6 text-zinc-700 dark:text-zinc-300"),
			P(
				Class("text-lg leading-relaxed text-zinc-800 "+
					"dark:text-zinc-200"),
				g.Text("The site runs on donations to cover hosting, "+
					"SMS, and other operating costs."),
			),
		),
		bitcoinDonateCard(config.BitcoinDonateAddress),
		P(
			Class("mt-6 text-lg leading-relaxed text-zinc-800 "+
				"dark:text-zinc-200"),
			g.Text("Thank you for your donation."),
		),
	}
}

func bitcoinDonateCard(addr string) g.Node {
	uri := "bitcoin:" + addr
	qrSrc := ""
	if addr != "" {
		qrSrc = bitcoinQRSrc(uri)
	}
	return Div(
		Class("mt-8 p-4 border border-zinc-200 dark:border-zinc-700 "+
			"rounded-lg space-y-4"),
		Div(
			Class("flex items-center gap-3"),
			Img(
				Class("w-5 h-5 dark:invert dark:opacity-80"),
				Src("/images/money.svg"),
				Alt(""),
			),
			H2(
				Class("text-lg font-semibold text-zinc-900 "+
					"dark:text-zinc-100"),
				g.Text("Bitcoin"),
			),
		),
		P(
			Class("text-base leading-relaxed text-zinc-700 "+
				"dark:text-zinc-300"),
			g.Text("Bitcoin is the only payment we accept."),
		),
		g.If(addr != "", bitcoinAddressBlock(addr, qrSrc)),
	)
}

func bitcoinAddressBlock(addr, qrSrc string) g.Node {
	return Div(
		Class("space-y-3"),
		P(
			Class("text-base leading-relaxed text-zinc-700 "+
				"dark:text-zinc-300"),
			g.Text("Send BTC to this address:"),
		),
		Div(
			Class("flex flex-col sm:flex-row sm:items-start gap-4"),
			Div(
				Class("min-w-0 flex-1 space-y-3"),
				Div(
					Class("font-mono text-sm break-all p-3 "+
						"bg-zinc-100 dark:bg-zinc-800 "+
						"border border-zinc-300 "+
						"dark:border-zinc-600 rounded-md "+
						"text-zinc-900 dark:text-zinc-100"),
					g.Text(addr),
				),
				donateCopyButton(addr),
			),
			g.If(qrSrc != "",
				Img(
					Class("w-40 h-40 shrink-0 bg-white p-2 "+
						"rounded-md border border-zinc-300 "+
						"dark:border-zinc-600"),
					Src(qrSrc),
					Alt("Bitcoin address QR code"),
				),
			),
		),
	)
}

func donateCopyButton(addr string) g.Node {
	return Button(
		Type("button"),
		Class("px-3 py-1.5 text-sm border border-zinc-300 "+
			"dark:border-zinc-600 rounded-md "+
			"hover:bg-zinc-50 dark:hover:bg-zinc-800 "+
			"transition-colors"),
		g.Attr("onclick", fmt.Sprintf(
			`navigator.clipboard.writeText(%q)`, addr)),
		g.Text("Copy address"),
	)
}

func bitcoinQRSrc(content string) string {
	return qrPNGDataURI(content, 192)
}
