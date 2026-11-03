package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// FlagOpinionModalData holds presentation fields for a dispute assessment modal.
// Party identities are never shown; roles are Owner / Inquirer only.
type FlagOpinionModalData struct {
	ConversationID   int
	AdID             int
	AdTitle          string
	Unavailable      bool
	Summary          string
	AssessmentScore  int
	AssessmentDetail string
	Resolution       string
	Reasoning        string
	AdFacts          []string
	GeneratedAt      *time.Time
}

func clampAssessmentScore(score int) int {
	if score < 1 {
		return 1
	}
	if score > 10 {
		return 10
	}
	return score
}

func assessmentTiltDeg(score int) float64 {
	return float64(score-5) * 3.6
}

func assessmentMarkerPct(score int) float64 {
	return float64(score-1) / 9.0 * 100.0
}

func assessmentScaleSection(score int, detail string) g.Node {
	score = clampAssessmentScore(score)
	tilt := assessmentTiltDeg(score)
	markerPct := assessmentMarkerPct(score)

	return Div(
		Class("mb-4"),
		H3(
			Class("text-sm font-semibold text-zinc-800 dark:text-zinc-200 mb-2"),
			g.Text("Assessment"),
		),
		Div(
			Class("flex justify-between text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-3"),
			Span(g.Text("Inquirer")),
			Span(
				Class("text-lg font-bold text-zinc-800 dark:text-zinc-200"),
				g.Text(fmt.Sprintf("%d", score)),
			),
			Span(g.Text("Owner")),
		),
		Div(
			Class("relative h-20 flex items-end justify-center mb-3"),
			Div(
				Class("absolute bottom-0 w-0 h-0"),
				Style("border-left: 10px solid transparent; "+
					"border-right: 10px solid transparent; "+
					"border-bottom: 16px solid #a1a1aa"),
			),
			Div(
				Class("absolute bottom-4 w-44 h-1 bg-zinc-600 dark:bg-zinc-400 rounded"),
				Style(fmt.Sprintf("transform: rotate(%.1fdeg)", tilt)),
				Div(
					Class("absolute -left-1 -top-7 w-9 h-9 rounded-full border-2 "+
						"border-zinc-600 dark:border-zinc-400 bg-zinc-100 dark:bg-zinc-700"),
				),
				Div(
					Class("absolute -right-1 -top-7 w-9 h-9 rounded-full border-2 "+
						"border-zinc-600 dark:border-zinc-400 bg-zinc-100 dark:bg-zinc-700"),
				),
			),
		),
		Div(
			Class("relative h-2 bg-zinc-200 dark:bg-zinc-700 rounded-full mx-2 mb-3"),
			Div(
				Class("absolute -top-1 w-3 h-3 bg-orange-500 rounded-full"),
				Style(fmt.Sprintf("left: calc(%.1f%% - 6px)", markerPct)),
			),
		),
		P(
			Class("text-xs text-zinc-500 dark:text-zinc-400 text-center mb-2"),
			g.Text("1 = inquirer in the right · 5 = balanced · 10 = owner in the right"),
		),
		P(
			Class("text-sm text-zinc-700 dark:text-zinc-300 whitespace-pre-wrap"),
			g.Text(detail),
		),
	)
}

func opinionSection(title, body string) g.Node {
	return Div(
		Class("mb-4"),
		H3(
			Class("text-sm font-semibold text-zinc-800 dark:text-zinc-200 mb-1"),
			g.Text(title),
		),
		P(
			Class("text-sm text-zinc-700 dark:text-zinc-300 whitespace-pre-wrap"),
			g.Text(body),
		),
	)
}

// FlagOpinionModal renders the cached or generated dispute assessment.
func FlagOpinionModal(d FlagOpinionModalData) g.Node {
	modalName := fmt.Sprintf("flag-opinion-%d", d.ConversationID)

	return g.Group([]g.Node{
		modalBackdrop(modalName),
		Div(
			ID(modalName+"-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-lg shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto"),
				Style("max-height: min(80vh, 600px)"),
				Div(
					Class("flex items-start justify-between p-4 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					Div(
						Class("flex-1 min-w-0 pr-4"),
						Div(
							Class("text-sm text-zinc-600 dark:text-zinc-400 mb-1"),
							Span(Class("font-semibold"), g.Text("Subject: ")),
							g.Text(d.AdTitle),
						),
						Div(
							Class("text-sm font-semibold text-zinc-800 dark:text-zinc-200"),
							g.Text("Dispute assessment"),
						),
						P(
							Class("text-xs text-zinc-500 dark:text-zinc-400 mt-1"),
							g.Text("Parties are shown only as Owner and Inquirer."),
						),
					),
					modalClose(modalName),
				),
				Div(
					Class("flex-1 overflow-y-auto p-4"),
					g.If(d.Unavailable,
						P(
							Class("text-sm text-zinc-600 dark:text-zinc-400 text-center py-8"),
							g.Text("Assessment unavailable — try again later."),
						),
					),
					g.If(!d.Unavailable,
						g.Group([]g.Node{
							opinionSection("Summary", d.Summary),
							g.If(len(d.AdFacts) > 0,
								Div(
									Class("mb-4"),
									H3(
										Class("text-sm font-semibold text-zinc-800 dark:text-zinc-200 mb-1"),
										g.Text("Relevant ad facts"),
									),
									Ul(
										Class("text-sm text-zinc-700 dark:text-zinc-300 list-disc pl-5 space-y-1"),
										g.Group(adFactItems(d.AdFacts)),
									),
								),
							),
							assessmentScaleSection(
								d.AssessmentScore, d.AssessmentDetail,
							),
							opinionSection("Recommended resolution", d.Resolution),
							opinionSection("Reasoning", d.Reasoning),
							g.If(d.GeneratedAt != nil,
								P(
									Class("text-xs text-zinc-500 dark:text-zinc-400 mt-4 pt-2 border-t border-zinc-200 dark:border-zinc-700"),
									g.Text(fmt.Sprintf(
										"Generated %s · Provisional — may update if parties continue messaging",
										d.GeneratedAt.Format("Jan 2, 2006 3:04 PM"),
									)),
								),
							),
						}),
					),
				),
			),
		),
	})
}

func adFactItems(facts []string) []g.Node {
	items := make([]g.Node, len(facts))
	for i, f := range facts {
		items[i] = Li(g.Text(f))
	}
	return items
}

// FlagOpinionLink renders a link for participants to view the assessment.
func FlagOpinionLink(conversationID int) g.Node {
	const label = "View dispute assessment"
	return A(
		Href("#"),
		ID(fmt.Sprintf("flag-opinion-link-%d", conversationID)),
		Class("flex-shrink-0 cursor-pointer flag-opinion-link"),
		hx.Get(fmt.Sprintf("/auth/conversation/%d/flag-opinion", conversationID)),
		hx.Target("body"),
		hx.Swap("beforeend"),
		hx.Indicator("this"),
		Title(label),
		g.Attr("aria-label", label),
		Img(
			Src("/images/balance.svg"),
			Alt(label),
			Class("w-6 h-6 dark:invert dark:opacity-80"),
		),
	)
}
