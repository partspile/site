package ui

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

const maxFlyerImages = 4

const flyerSheetCSS = `
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
`

const flyerPrintCSS = flyerSheetCSS + `
@page {
  size: letter portrait;
  margin: 0.5in;
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
    padding: 0;
    height: 10in;
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
				Type("image/svg+xml"),
				Href("/images/rock.svg"),
			),
			Link(Rel("stylesheet"), Href("/css/output.css")),
			g.El("style", g.Raw(flyerPrintCSS)),
		},
		Body: []g.Node{
			Class("bg-white text-zinc-900 min-h-screen"),
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
				H1(
					Class("text-3xl font-bold"),
					adCardTitle(d.Title, d.FacetLabel),
				),
				Div(
					Class("flex flex-wrap items-baseline "+
						"gap-x-2 gap-y-0.5"),
					priceSpan(d.PriceDisplay, d.HasPrice),
					g.If(d.Location != "",
						Span(Class("text-sm text-zinc-500"),
							g.Text(d.Location))),
				),
				flyerDetails(d),
			),
			g.If(qrSrc != "", flyerQR(d.AdURL, qrSrc)),
		),
		flyerImages(d.ID, d.ImageCount),
	)
}

func flyerQR(adURL, qrSrc string) g.Node {
	return Div(
		Class("shrink-0 flex flex-col items-center gap-2"),
		Img(
			Class("w-40 h-40 bg-white p-2 rounded-md "+
				"border border-zinc-300"),
			Src(qrSrc),
			Alt("QR code for this ad"),
		),
		Span(
			Class("w-40 text-xs text-zinc-500 break-all text-center"),
			g.Text(adURL),
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
