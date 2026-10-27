package ui

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/rock"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// RockThrowConfirmData drives the pre-throw confirmation modal.
type RockThrowConfirmData struct {
	ConversationID int
	AdID           int
	CSRFToken      string
	Remaining      int
	ThrowLabel     string
	// AtAd is true when the inquirer is throwing at the listing.
	AtAd bool
	// OtherUserID / OtherName are the counterparty (ad owner when AtAd, else inquirer).
	OtherUserID int
	OtherName   string
}

// RockThrowPreviewData is the inline assessment preview inside the confirm modal.
type RockThrowPreviewData struct {
	Unavailable      bool
	Summary          string
	AssessmentScore  int
	AssessmentDetail string
	Resolution       string
	Reasoning        string
}

const rockThrowConfirmModalName = "rock-throw-confirm"
const rockThrowPreviewModalName = "rock-throw-preview"
const rockThrowUserPopupID = "rock-throw-confirm-user-popup"

// CloseRockThrowModalsOOB removes confirm/preview modals after a successful throw.
func CloseRockThrowModalsOOB() []g.Node {
	return append(
		RemoveModal(rockThrowConfirmModalName),
		RemoveModal(rockThrowPreviewModalName)...,
	)
}

// RockThrowConfirmModal is a single scrollable confirm before throwing a rock.
func RockThrowConfirmModal(d RockThrowConfirmData) g.Node {
	csrfHeader := fmt.Sprintf(`{"X-Csrf-Token": %q}`, d.CSRFToken)
	previewURL := fmt.Sprintf(
		"/auth/conversation/%d/rock/preview", d.ConversationID)
	throwURL := fmt.Sprintf(
		"/auth/conversation/%d/rock/throw", d.ConversationID)
	msgTarget := ConversationMessagesSelector(d.ConversationID)
	if d.ConversationID == 0 {
		previewURL = fmt.Sprintf("/auth/ad/%d/rock/preview", d.AdID)
		throwURL = fmt.Sprintf("/auth/ad/%d/rock/throw", d.AdID)
		msgTarget = "#conversation-0-messages"
	}

	throwLabel := d.ThrowLabel
	if throwLabel == "" {
		throwLabel = "Throw rock"
	}

	return g.Group([]g.Node{
		modalBackdrop(rockThrowConfirmModalName),
		Div(
			ID(rockThrowConfirmModalName+"-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-2 sm:p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-lg shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto max-h-[90vh]"),
				Div(
					Class("flex items-center justify-between p-4 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(
						Class("text-lg font-bold text-zinc-900 dark:text-zinc-200"),
						g.Text("Throw a rock?"),
					),
					modalClose(rockThrowConfirmModalName),
				),
				Form(
					ID("rock-throw-confirm-form"),
					Class("flex flex-col flex-1 min-h-0"),
					Div(
						Class("flex-1 overflow-y-auto p-4 space-y-4"),
						remainingRocksSection(d.Remaining),
						rockThrowCautionBullets(d),
						g.If(d.Remaining <= 0,
							P(
								Class("text-sm text-red-600 dark:text-red-400"),
								g.Text("You cannot throw until you unthrow an outstanding rock."),
							),
						),
						g.If(d.Remaining > 0, rockReasonRadios(d)),
					),
					Div(
						Class("flex flex-wrap gap-2 justify-end p-4 border-t border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
						Button(
							Type("button"),
							Class("px-4 py-2 rounded-md border border-zinc-300 dark:border-zinc-600 text-zinc-800 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-700"),
							hx.Get(fmt.Sprintf("/api/modal-remove/%s",
								rockThrowConfirmModalName)),
							hx.Swap("none"),
							g.Text("Cancel"),
						),
						g.If(d.Remaining > 0, g.Group([]g.Node{
							Button(
								Type("button"),
								ID("rock-throw-review-btn"),
								Class("rock-throw-review-btn inline-flex items-center gap-2 px-4 py-2 rounded-md bg-zinc-700 text-white hover:bg-zinc-800 disabled:opacity-40 disabled:cursor-not-allowed"),
								Disabled(),
								hx.Post(previewURL),
								hx.Headers(csrfHeader),
								hx.Include("#rock-throw-confirm-form"),
								hx.Target("body"),
								hx.Swap("beforeend"),
								hx.Indicator("this"),
								g.Attr("hx-on::before-request",
									"if(document.getElementById('"+
										rockThrowPreviewModalName+
										"-modal')){htmx.ajax('GET','/api/modal-remove/"+
										rockThrowPreviewModalName+
										"',{swap:'none'});}"),
								Img(
									Src("/images/balance.svg"),
									Alt(""),
									Class("w-5 h-5 flex-shrink-0 invert"),
								),
								g.Text("Review assessment"),
							),
							Button(
								Type("button"),
								ID("rock-throw-submit-btn"),
								Class("px-4 py-2 rounded-md bg-red-500 text-white hover:bg-red-600 disabled:opacity-40 disabled:cursor-not-allowed"),
								Disabled(),
								hx.Post(throwURL),
								hx.Headers(csrfHeader),
								hx.Include("#rock-throw-confirm-form"),
								hx.Target(msgTarget),
								hx.Swap(conversationMessagesAppendSwap()),
								g.Text(throwLabel),
							),
						})),
					),
				),
			),
			// Viewport-fixed host (sibling of the card) so summaries aren't
			// clipped by the modal body's overflow-y-auto.
			Div(
				ID(rockThrowUserPopupID),
				Class("fixed z-[100] hidden w-64 p-3 bg-white dark:bg-zinc-800 rounded shadow-lg border border-zinc-200 dark:border-zinc-700 text-sm text-zinc-700 dark:text-zinc-300 pointer-events-auto"),
				g.Attr("onmouseenter",
					"this.dataset.hover='1'"),
				g.Attr("onmouseleave",
					"this.dataset.hover='';this.classList.add('hidden')"),
			),
		),
	})
}

func remainingRocksSection(remaining int) g.Node {
	msg := fmt.Sprintf("You have %d rocks", remaining)
	if remaining == 1 {
		msg = "You have 1 rock"
	}
	if remaining <= 0 {
		msg = "You have no rocks"
	}
	return Div(
		Class("flex flex-col items-center gap-2 py-2"),
		g.If(remaining > 0, confirmRockIcons(remaining)),
		P(
			Class("text-sm font-medium text-zinc-800 dark:text-zinc-200"),
			g.Text(msg),
		),
	)
}

func confirmRockIcons(rockCount int) g.Node {
	icons := make([]g.Node, 0, rockCount)
	for range rockCount {
		icons = append(icons, Img(
			Src("/images/rock.svg"),
			Alt("Rock"),
			Class("w-10 h-10 flex-shrink-0"),
		))
	}
	return Span(
		Class("inline-flex items-center gap-1.5"),
		g.Group(icons),
	)
}

func rockThrowCautionBullets(d RockThrowConfirmData) g.Node {
	left := d.Remaining - 1
	if left < 0 {
		left = 0
	}
	leftMsg := fmt.Sprintf("You'll have %d rocks left.", left)
	if left == 1 {
		leftMsg = "You'll have 1 rock left."
	}
	if left == 0 {
		leftMsg = "You'll have no rocks left."
	}

	reviewLink := rockConfirmReviewLink()

	var markLine, resolveLine g.Node
	if d.AtAd {
		markLine = Li(
			g.Text("This ad will show a rock anyone can see, with a dispute assessment (use "),
			reviewLink,
			g.Text(" for a neutral take first)."),
		)
		resolveLine = Li(
			g.Text("If you resolve the dispute with "),
			rockConfirmUserLinkID(d.OtherUserID, d.OtherName, "resolve"),
			g.Text(", you can Unthrow and get your rock back."),
		)
	} else {
		markLine = Li(
			rockConfirmUserLinkID(d.OtherUserID, d.OtherName, "mark"),
			g.Text(" will have a rock on their profile that anyone can see, with a dispute assessment (use "),
			reviewLink,
			g.Text(" for a neutral take first)."),
		)
		resolveLine = Li(
			g.Text("If you resolve the dispute with "),
			rockConfirmUserLinkID(d.OtherUserID, d.OtherName, "resolve"),
			g.Text(", you can Unthrow and get your rock back."),
		)
	}

	return Div(
		Class("space-y-1.5"),
		P(
			Class("text-sm font-semibold text-zinc-800 dark:text-zinc-200"),
			g.Text("If you throw a rock…"),
		),
		Ul(
			Class("text-xs text-zinc-600 dark:text-zinc-400 list-disc pl-4 space-y-1"),
			Li(g.Text(leftMsg)),
			markLine,
			resolveLine,
			Li(g.Text(
				"If they are removed for repeated abuse, your rock is returned.",
			)),
		),
	)
}

func rockConfirmUserLinkID(userID int, name, _ string) g.Node {
	if name == "" {
		name = "them"
	}
	if userID <= 0 {
		return Span(Class("font-medium"), g.Text(name))
	}
	href := fmt.Sprintf("/auth/user/%d", userID)
	summaryURL := fmt.Sprintf("/auth/user/%d/summary", userID)
	// Fixed popup host avoids clipping by the modal's overflow-y-auto body.
	// Desktop: hover loads/shows summary. Mobile: tap follows href to profile.
	showJS := "var p=document.getElementById('" + rockThrowUserPopupID + "');" +
		"if(!p)return;" +
		"var r=this.getBoundingClientRect();" +
		"var w=256;" +
		"var left=Math.min(Math.max(8,r.left),window.innerWidth-w-8);" +
		"var top=r.bottom+6;" +
		"if(top+180>window.innerHeight){top=Math.max(8,r.top-186);}" +
		"p.style.left=left+'px';" +
		"p.style.top=top+'px';" +
		"p.classList.remove('hidden');"
	hideJS := "var p=document.getElementById('" + rockThrowUserPopupID + "');" +
		"if(!p)return;" +
		"setTimeout(function(){if(p.dataset.hover!=='1')p.classList.add('hidden');},120);"
	return A(
		Href(href),
		Class("text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline"),
		hx.Get(summaryURL),
		hx.Trigger("mouseenter delay:200ms"),
		hx.Target("#"+rockThrowUserPopupID),
		hx.Swap("innerHTML"),
		g.Attr("onmouseenter", showJS),
		g.Attr("onmouseleave", hideJS),
		g.Text(name),
	)
}

func rockConfirmReviewLink() g.Node {
	return Button(
		Type("button"),
		ID("rock-throw-review-link"),
		Class("underline font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 p-0 bg-transparent border-0 cursor-pointer inline text-xs disabled:opacity-40 disabled:cursor-not-allowed disabled:no-underline disabled:hover:text-blue-600 dark:disabled:hover:text-blue-400"),
		Disabled(),
		g.Attr("onclick",
			"var b=document.getElementById('rock-throw-review-btn');"+
				"if(b&&!b.disabled){b.click();}"),
		g.Text("Review assessment"),
	)
}

func rockThrowEnableActionsJS() string {
	return "document.getElementById('rock-throw-review-btn').disabled=false;" +
		"document.getElementById('rock-throw-review-link').disabled=false;" +
		"document.getElementById('rock-throw-submit-btn').disabled=false;" +
		"if(document.getElementById('" + rockThrowPreviewModalName +
		"-modal')){htmx.ajax('GET','/api/modal-remove/" +
		rockThrowPreviewModalName + "',{swap:'none'});}"
}

func rockReasonRadios(d RockThrowConfirmData) g.Node {
	reasons := rock.ReasonsForTarget(d.AtAd)
	var options []g.Node
	for _, r := range reasons {
		code := r.Code
		exID := fmt.Sprintf("rock-reason-ex-%s", code)
		var exItems []g.Node
		for _, ex := range r.Examples {
			exItems = append(exItems, Li(g.Text(ex)))
		}
		labelBody := []g.Node{
			Span(
				Class("text-sm font-medium text-zinc-900 dark:text-zinc-200"),
				g.Text(r.Label),
			),
		}
		if code == rock.ReasonPolicy {
			labelBody = append(labelBody, P(
				Class("text-xs mt-0.5"),
				A(
					Href("/terms#prohibited"),
					Target("_blank"),
					Rel("noopener noreferrer"),
					Class("text-blue-600 dark:text-blue-400 hover:underline"),
					g.Attr("onclick", "event.stopPropagation()"),
					g.Text("See Terms"),
				),
			))
		}
		options = append(options,
			Label(
				Class("block border border-zinc-200 dark:border-zinc-600 rounded-md p-3 cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"),
				Div(
					Class("flex items-start gap-2"),
					Input(
						Type("radio"),
						Name("reason"),
						Value(code),
						Class("mt-1"),
						Required(),
						g.Attr("onchange",
							"document.querySelectorAll('.rock-reason-ex').forEach(function(e){e.classList.add('hidden')});"+
								"document.getElementById('"+exID+"').classList.remove('hidden');"+
								rockThrowEnableActionsJS()),
					),
					Div(Class("min-w-0"), g.Group(labelBody)),
				),
				Ul(
					ID(exID),
					Class("rock-reason-ex hidden text-sm text-zinc-600 dark:text-zinc-400 list-disc pl-8 mt-2 space-y-0.5"),
					g.Group(exItems),
				),
			),
		)
	}

	var why g.Node
	if d.AtAd {
		why = g.Text("Why are you throwing this rock at the ad?")
	} else {
		why = g.Group([]g.Node{
			g.Text("Why are you throwing this rock at "),
			rockConfirmUserLinkID(d.OtherUserID, d.OtherName, "why"),
			g.Text("?"),
		})
	}

	return Div(
		Class("space-y-2"),
		P(
			Class("text-sm font-semibold text-zinc-800 dark:text-zinc-200"),
			why,
		),
		Div(Class("space-y-2"), g.Group(options)),
	)
}

// RockThrowPreviewModal is a stacked modal for pre-throw dispute assessment.
func RockThrowPreviewModal(d RockThrowPreviewData) g.Node {
	name := rockThrowPreviewModalName
	var body g.Node
	if d.Unavailable {
		body = Div(
			Class("space-y-2 py-4"),
			P(
				Class("text-sm font-medium text-zinc-800 dark:text-zinc-200 text-center"),
				g.Text("Assessment unavailable"),
			),
			P(
				Class("text-sm text-zinc-600 dark:text-zinc-400 text-center"),
				g.Text("You can still throw if you are sure. Prefer waiting and trying Review again."),
			),
		)
	} else {
		body = g.Group([]g.Node{
			P(
				Class("text-xs font-semibold uppercase tracking-wide text-zinc-500 dark:text-zinc-400 mb-3"),
				g.Textf("Provisional — not a %s ruling",
					config.ServerName),
			),
			assessmentScaleSection(d.AssessmentScore, d.AssessmentDetail),
			opinionSection("Summary", d.Summary),
			Details(
				Class("text-sm text-zinc-700 dark:text-zinc-300"),
				Summary(
					Class("cursor-pointer font-medium text-zinc-800 dark:text-zinc-200"),
					g.Text("More detail"),
				),
				Div(
					Class("mt-2 space-y-2"),
					opinionSection("Recommended resolution", d.Resolution),
					opinionSection("Reasoning", d.Reasoning),
				),
			),
		})
	}

	return g.Group([]g.Node{
		Div(
			ID(name+"-modal-backdrop"),
			Class("fixed inset-0 bg-black/40 z-[55]"),
			hx.Get(fmt.Sprintf("/api/modal-remove/%s", name)),
			hx.Swap("none"),
			hx.Trigger("click"),
		),
		Div(
			ID(name+"-modal"),
			Class("fixed inset-0 flex items-center justify-center z-[60] p-2 sm:p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-lg shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto max-h-[85vh]"),
				Div(
					Class("flex items-center justify-between p-4 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					Div(
						Class("flex items-center gap-2 min-w-0"),
						Img(
							Src("/images/balance.svg"),
							Alt(""),
							Class("w-5 h-5 flex-shrink-0 dark:invert"),
						),
						H3(
							Class("text-lg font-bold text-zinc-900 dark:text-zinc-200"),
							g.Text("Dispute assessment"),
						),
					),
					modalClose(name),
				),
				Div(
					Class("flex-1 overflow-y-auto p-4"),
					body,
				),
				Div(
					Class("flex justify-end p-4 border-t border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					Button(
						Type("button"),
						Class("px-4 py-2 rounded-md bg-zinc-700 text-white hover:bg-zinc-800"),
						hx.Get(fmt.Sprintf("/api/modal-remove/%s", name)),
						hx.Swap("none"),
						g.Text("Close"),
					),
				),
			),
		),
	})
}
