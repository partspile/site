package ui

import (
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

const maxFlyerImages = 4

const flyerSheetCSS = `
.flyer-page {
  --content-ink: #1c1917;
  --content-ink-soft: #57534e;
  --content-ink-muted: #78716c;
  color: #1c1917;
  background: #fff;
}
.flyer-page a {
  color: #2563eb;
}
.flyer-sheet {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  height: 10in;
  max-height: 10in;
  overflow: hidden;
  box-sizing: border-box;
}
.flyer-images {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}
.flyer-image {
  aspect-ratio: 4 / 3;
  min-width: 0;
  overflow: hidden;
}
.flyer-images img,
.flyer-images svg {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.flyer-desc {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 6;
}
.flyer-logo {
  width: 10rem;
  height: auto;
  display: block;
  margin-bottom: 2rem;
}
.flyer-price {
  color: #1c1917;
  font-size: 3rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.02em;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}
.flyer-qr {
  position: relative;
  width: 10rem;
  min-width: 10rem;
  max-width: 10rem;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}
.flyer-qr img {
  border: 2px solid #1c1917;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}
/* Cloud sits up-left of the QR. Origin is the leader tip on the code's left edge. */
.flyer-scan {
  position: absolute;
  width: 12rem;
  height: auto;
  left: -11.15rem;
  top: -1.15rem;
  overflow: visible;
  transform: rotate(-4deg);
  transform-origin: 94% 89%;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}
.flyer-foot {
  margin-top: auto;
  display: flex;
  flex-wrap: nowrap;
  gap: 1.25rem;
  align-items: baseline;
  font-size: 0.7rem;
  line-height: 1.2;
  color: #78716c;
  white-space: nowrap;
}
`

const flyerPrintCSS = flyerSheetCSS + `
@page {
  size: letter portrait;
  /* No margin: the browser draws the date, title, and URL there. */
  margin: 0;
}
@media print {
  .flyer-screen-only { display: none !important; }
  html, body {
    margin: 0;
    overflow: hidden;
    background: white;
  }
  .flyer-sheet {
    max-width: none;
    margin: 0;
    padding: 0.5in 0.5in 0.4in;
    height: 11in;
    max-height: 11in;
  }
}
`

func AdFlyerPage(d AdFlyerData) g.Node {
	qrSrc := qrPNGDataURI(d.AdURL, 256)
	docTitle := config.ServerName
	if d.Title != "" {
		docTitle = config.ServerName + " - " + d.Title
	}
	return components.HTML5(components.HTML5Props{
		Title:    docTitle,
		Language: "en",
		Head: []g.Node{
			Meta(Name("robots"), Content("noindex, nofollow")),
			Meta(Name("color-scheme"), Content("light")),
			Link(
				Rel("icon"),
				Type("image/png"),
				Href("/images/favicon.png"),
			),
			Link(Rel("stylesheet"), Href("/css/output.css")),
			g.El("style", g.Raw(flyerPrintCSS)),
		},
		Body: []g.Node{
			Class("flyer-page bg-white text-zinc-900 min-h-screen"),
			flyerToolbar(d.ID),
			flyerSheet(d, qrSrc),
		},
	})
}

func flyerToolbar(adID int) g.Node {
	return Div(
		Class("flyer-screen-only flex items-center justify-between "+
			"gap-4 px-6 py-4 border-b border-zinc-200"),
		A(
			Href(fmt.Sprintf("/ad/%d", adID)),
			Class("text-blue-600 hover:underline"),
			g.Text("Back to ad"),
		),
		standardButton(buttonProps{
			Type: "button",
			Text: "Print",
			Attrs: []g.Node{
				g.Attr("onclick", "window.print()"),
			},
		}),
	)
}

func flyerSheet(d AdFlyerData, qrSrc string) g.Node {
	return Div(
		Class("flyer-sheet max-w-3xl mx-auto p-6"),
		Div(
			Class("flex items-start gap-6 shrink-0"),
			Div(
				Class("min-w-0 flex-1 flex flex-col gap-3"),
				Img(
					Class("flyer-logo"),
					Src("/images/parts-pile-logo-light.png"),
					Alt("Parts Pile"),
				),
				H1(
					Class("text-3xl font-bold"),
					adCardTitle(d.Title, d.FacetLabel),
				),
				Div(
					Class("flex flex-wrap items-baseline "+
						"gap-x-3 gap-y-0.5"),
					g.If(d.HasPrice,
						Span(Class("flyer-price shrink-0 whitespace-nowrap"),
							g.Text(d.PriceDisplay))),
					g.If(d.Location != "",
						Span(Class("text-sm text-zinc-500"),
							g.Text(d.Location))),
				),
				flyerDetails(d),
			),
			g.If(qrSrc != "", flyerQR(qrSrc)),
		),
		flyerImages(d.ID, d.ImageCount),
		flyerFoot(d),
	)
}

func flyerFoot(d AdFlyerData) g.Node {
	when := d.PrintedAt
	if when.IsZero() {
		when = time.Now()
	}
	return Div(
		Class("flyer-foot"),
		g.If(d.AdURL != "", Span(g.Text(d.AdURL))),
		Span(g.Text("printed: "+when.Format("Jan 2, 2006"))),
	)
}

func flyerQR(qrSrc string) g.Node {
	return Div(
		Class("flyer-qr"),
		flyerScanBubble(),
		Img(
			Class("w-40 h-40 bg-white p-2 rounded-md"),
			Src(qrSrc),
			Alt("QR code for this ad"),
		),
	)
}

func flyerScanBubble() g.Node {
	hand := "Chalkboard SE, Marker Felt, Comic Sans MS, cursive"
	return g.El("svg",
		Class("flyer-scan"),
		g.Attr("viewBox", "0 0 270 170"),
		g.Attr("role", "img"),
		g.Attr("aria-label", "Scan to inquire"),
		g.El("path",
			g.Attr("fill", "#fffef8"),
			g.Attr("stroke", "#1c1917"),
			g.Attr("stroke-width", "3.5"),
			g.Attr("stroke-linejoin", "round"),
			g.Attr("d", "M40 108C12 108 8 74 36 64C28 38 60 16 84 40"+
				"C96 14 138 12 150 40C168 16 210 30 196 64"+
				"C222 74 214 114 174 110C180 140 128 156 96 132"+
				"C74 152 32 144 36 120C12 130 8 118 40 108Z"),
		),
		g.El("path",
			g.Attr("fill", "none"),
			g.Attr("stroke", "#1c1917"),
			g.Attr("stroke-width", "3"),
			g.Attr("stroke-linecap", "round"),
			g.Attr("d", "M210 92Q232 120 252 150"),
		),
		g.El("circle",
			g.Attr("cx", "254"),
			g.Attr("cy", "152"),
			g.Attr("r", "4.5"),
			g.Attr("fill", "#1c1917"),
		),
		g.El("text",
			g.Attr("x", "104"),
			g.Attr("y", "78"),
			g.Attr("text-anchor", "middle"),
			g.Attr("font-size", "28"),
			g.Attr("font-weight", "700"),
			g.Attr("font-family", hand),
			g.Attr("fill", "#1c1917"),
			g.Text("Scan to"),
		),
		g.El("text",
			g.Attr("x", "104"),
			g.Attr("y", "110"),
			g.Attr("text-anchor", "middle"),
			g.Attr("font-size", "28"),
			g.Attr("font-weight", "700"),
			g.Attr("font-family", hand),
			g.Attr("fill", "#1c1917"),
			g.Text("inquire"),
		),
	)
}

func flyerDetails(d AdFlyerData) g.Node {
	var nodes []g.Node
	if len(d.FacetDetails) > 0 {
		items := make([]g.Node, len(d.FacetDetails))
		for i, s := range d.FacetDetails {
			items[i] = Li(g.Text(s))
		}
		nodes = append(nodes, Ul(
			Class("list-disc list-inside mb-3 text-sm text-zinc-700"),
			g.Group(items),
		))
	}
	if d.Description != "" {
		nodes = append(nodes, Div(
			Class("flyer-desc whitespace-pre-wrap"),
			uiads.DescriptionTextWithLinks(d.Description),
		))
	}
	if len(d.Tags) > 0 {
		pills := make([]g.Node, len(d.Tags))
		for i, s := range d.Tags {
			pills[i] = Span(
				Class("inline-block px-3 py-1 rounded-full "+
					"border border-zinc-300 text-sm text-zinc-700"),
				g.Text(s),
			)
		}
		nodes = append(nodes, Div(
			Class("flex flex-wrap gap-2 mt-3"),
			g.Group(pills),
		))
	}
	if len(nodes) == 0 {
		return nil
	}
	return Div(Class("text-base"), g.Group(nodes))
}

func flyerImages(adID, count int) g.Node {
	if count < 1 {
		return nil
	}
	if count > maxFlyerImages {
		count = maxFlyerImages
	}
	imgs := make([]g.Node, count)
	for i := 1; i <= count; i++ {
		imgs[i-1] = flyerImg(adID, i)
	}
	return Div(Class("flyer-images"), g.Group(imgs))
}

func flyerImg(adID, index int) g.Node {
	class := "rounded border border-zinc-200"
	src := AdImageSrc(adID, index, "480w")
	var img g.Node
	if src == "" {
		img = GenerateSVG(adID, index, "480w", class)
	} else {
		img = Img(
			Class(class),
			Src(src),
			Alt(fmt.Sprintf("Ad image %d", index)),
		)
	}
	return Div(Class("flyer-image"), img)
}
