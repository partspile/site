package ui

import (
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/phoneformat"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

const RecoverPasswordErrorID = "recover-password-error"

func RecoverPage(code, twilioFrom string, expiresAt time.Time) []g.Node {
	return []g.Node{
		pageTitle("Recover account"),
		RecoverWaitingPanel(code, twilioFrom, expiresAt),
	}
}

func recoverBackToLogin() g.Node {
	return P(
		Class("text-sm text-zinc-600 dark:text-zinc-400"),
		A(
			Href("/login"),
			Class("text-blue-600 dark:text-blue-400 hover:underline"),
			g.Text("Back to login"),
		),
	)
}

func RecoverWaitingPanel(code, twilioFrom string, expiresAt time.Time) g.Node {
	displayPhone := phoneformat.Display(twilioFrom)
	if displayPhone == "" {
		displayPhone = twilioFrom
	}
	message := "RECOVER " + code
	expiresUnix := expiresAt.UTC().Unix()

	return Div(
		ID("recover-panel"),
		Class("space-y-8 mt-8"),
		hx.Get("/api/recover/status"),
		hx.Trigger("every 3s"),
		hx.Swap("outerHTML"),
		Div(
			Class("space-y-4"),
			P(
				Class("text-base leading-relaxed"),
				g.Text("From your registered phone, text this exact message to "),
				Span(Class("font-semibold"), g.Text(displayPhone)),
				g.Text(":"),
			),
			Div(
				Class("text-center py-6 space-y-3"),
				P(
					Class("text-sm text-zinc-500 dark:text-zinc-400"),
					g.Text("Text exactly"),
				),
				P(
					ID("recover-message"),
					Class("text-3xl sm:text-4xl font-mono font-semibold tracking-wide break-all"),
					g.Text(message),
				),
				P(
					Class("text-sm text-zinc-600 dark:text-zinc-400"),
					g.Text("Expires in "),
					Span(
						ID("recover-countdown"),
						Class("font-mono font-semibold tabular-nums"),
						g.Attr("data-expires", fmt.Sprintf("%d", expiresUnix)),
						g.Text(formatCountdown(expiresAt)),
					),
				),
			),
			recoverCountdownScript(),
		),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Waiting for your text message…"),
		),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text(
				"Anyone who can send texts from your registered "+
					"number can complete recovery and set a new "+
					"password. Protect your carrier account "+
					"(account PIN, port-out or SIM locks where "+
					"available). ",
			),
			A(
				Href("/faq/account-recovery"),
				Class("text-blue-600 dark:text-blue-400 hover:underline"),
				g.Text("See FAQ"),
			),
			g.Text("."),
		),
		recoverBackToLogin(),
	)
}

func formatCountdown(expiresAt time.Time) string {
	left := int(time.Until(expiresAt).Seconds())
	if left < 0 {
		left = 0
	}
	return fmt.Sprintf("%d:%02d", left/60, left%60)
}

func recoverCountdownScript() g.Node {
	// Live M:SS countdown; on expiry, refresh panel via status endpoint.
	return Script(g.Raw(`(function(){
var el=document.getElementById('recover-countdown');
if(!el)return;
var exp=parseInt(el.getAttribute('data-expires'),10);
if(!exp)return;
function tick(){
  var left=Math.max(0,exp-Math.floor(Date.now()/1000));
  el.textContent=Math.floor(left/60)+':'+(left%60<10?'0':'')+(left%60);
  if(left<=0){
    if(window.htmx){
      htmx.ajax('GET','/api/recover/status',{target:'#recover-panel',swap:'outerHTML'});
    }
    return;
  }
  setTimeout(tick,1000);
}
tick();
})();`))
}

func RecoverExpiredPanel() g.Node {
	return Div(
		ID("recover-panel"),
		Class("space-y-4 mt-8"),
		P(
			Class("text-base text-red-600 dark:text-red-400"),
			g.Text("This recovery code has expired."),
		),
		A(
			Href("/recover"),
			Class("inline-block text-blue-600 dark:text-blue-400 hover:underline"),
			g.Text("Get a new code"),
		),
		recoverBackToLogin(),
	)
}

func RecoverFailedPanel(message string) g.Node {
	return Div(
		ID("recover-panel"),
		Class("space-y-4 mt-8"),
		P(
			Class("text-base text-red-600 dark:text-red-400"),
			g.Text(message),
		),
		A(
			Href("/recover"),
			Class("inline-block text-blue-600 dark:text-blue-400 hover:underline"),
			g.Text("Try again"),
		),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Need an account? "),
			A(
				Href("/register"),
				Class("text-blue-600 dark:text-blue-400 hover:underline"),
				g.Text("Sign up"),
			),
		),
		recoverBackToLogin(),
	)
}

func RecoverResetForm(username string) g.Node {
	return Div(
		ID("recover-panel"),
		Class("space-y-6 mt-8"),
		P(
			Class("text-base"),
			g.Text("Phone verified. Set a new password for your account."),
		),
		Form(
			Class("space-y-4"),
			hx.Post("/api/recover/password"),
			hx.Swap("none"),
			passwordManagerUsername(username),
			Div(
				labeledPasswordField("New Password", "password", "new-password", true),
				Span(
					Class("text-xs text-zinc-500 dark:text-zinc-400 mt-1 block"),
					g.Text(password.StrengthRequirements),
				),
			),
			labeledPasswordField("Confirm Password", "password2", "new-password", false),
			Div(
				Class("flex items-center gap-4"),
				standardButton(buttonProps{
					Type: "submit",
					Text: "Update password",
				}),
				ErrorDivWithID(RecoverPasswordErrorID, ""),
			),
		),
		recoverBackToLogin(),
	)
}

func RecoverUnavailablePage() []g.Node {
	return []g.Node{
		pageTitle("Recover account"),
		P(
			Class("mt-8 text-base text-zinc-600 dark:text-zinc-400"),
			g.Text("Account recovery is temporarily unavailable. "),
			g.Text("Please try again later or contact "),
			A(
				Href("mailto:"+config.ContactEmail),
				Class("text-blue-600 dark:text-blue-400 hover:underline"),
				g.Text(config.ContactEmail),
			),
			g.Text("."),
		),
		P(
			Class("mt-4 text-sm"),
			A(
				Href("/login"),
				Class("text-blue-600 dark:text-blue-400 hover:underline"),
				g.Text("Back to login"),
			),
		),
	}
}
