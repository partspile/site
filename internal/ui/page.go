package ui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/local"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func pageDescription(path string) string {
	name := config.ServerName
	switch {
	case path == "/":
		return name + " - Classified Ads. Buy and sell vehicles, " +
			"parts, jobs, pets, housing, and more."
	case path == "/about":
		return name + " is classified ads for the Internet. Post with " +
			"your phone number; it stays hidden."
	case path == "/login":
		return "Log in to " + name + " to post ads, message sellers, " +
			"and manage your account."
	case strings.HasPrefix(path, "/c/"):
		return name + " classified ads by category."
	default:
		return ""
	}
}

func robotsForPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/auth/"),
		strings.HasPrefix(path, "/admin/"),
		strings.HasPrefix(path, "/u/"),
		path == "/recover",
		path == "/error",
		path == "/health",
		path == "/logout":
		return "noindex, nofollow"
	default:
		return "index, follow"
	}
}

func homepageSocialMeta(docTitle, desc, path string) []g.Node {
	return []g.Node{
		Meta(
			Name("keywords"),
			Content(
				"classifieds, marketplace, buy and sell, car parts, "+
					"vehicles, jobs, pets, housing, automotive parts, "+
					"bicycles, motorcycles, agricultural equipment",
			),
		),
		Meta(Name("author"), Content(config.ServerName)),
		Meta(g.Attr("property", "og:title"), Content(docTitle)),
		Meta(g.Attr("property", "og:description"), Content(desc)),
		Meta(g.Attr("property", "og:type"), Content("website")),
		Meta(
			g.Attr("property", "og:url"),
			Content(config.CanonicalURL(path)),
		),
		Meta(g.Attr("property", "og:site_name"), Content(config.ServerName)),
	}
}

func websiteJSONLD() g.Node {
	payload := map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     config.ServerName,
		"alternateName": []string{
			config.ServerName + " - Classified Ads",
			config.PublicHost(),
		},
		"url": config.CanonicalURL("/"),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return Script(
		Type("application/ld+json"),
		g.Raw(string(b)),
	)
}

func seoHead(docTitle, currentPath string) []g.Node {
	nodes := []g.Node{
		Link(Rel("canonical"), Href(config.CanonicalURL(currentPath))),
		Meta(Name("robots"), Content(robotsForPath(currentPath))),
	}
	desc := pageDescription(currentPath)
	if desc != "" {
		nodes = append(nodes, Meta(Name("description"), Content(desc)))
	}
	if currentPath == "/" || strings.HasPrefix(currentPath, "/c/") {
		nodes = append(nodes, homepageSocialMeta(docTitle, desc, currentPath)...)
		if currentPath == "/" {
			nodes = append(nodes, websiteJSONLD())
		}
	}
	if strings.HasPrefix(currentPath, "/u/") {
		nodes = append(nodes,
			sharedProfileSocialMeta(docTitle, desc, currentPath)...)
	}
	return nodes
}

func sharedProfileSocialMeta(docTitle, desc, path string) []g.Node {
	if desc == "" {
		desc = docTitle
	}
	return []g.Node{
		Meta(g.Attr("property", "og:title"), Content(docTitle)),
		Meta(g.Attr("property", "og:description"), Content(desc)),
		Meta(g.Attr("property", "og:type"), Content("profile")),
		Meta(
			g.Attr("property", "og:url"),
			Content(config.CanonicalURL(path)),
		),
		Meta(g.Attr("property", "og:site_name"), Content(config.ServerName)),
	}
}

func getUserInitial(userName string) string {
	return strings.ToUpper(string([]rune(userName)[0]))
}

func avatar(userName string) g.Node {
	return Span(
		ID("user-avatar"),
		Class("bg-red-500 text-white rounded-full w-8 h-8 flex items-center justify-center font-semibold text-sm cursor-pointer hover:bg-red-600"),
		hx.Get("/auth/user/menu"),
		hx.Target("body"),
		hx.Swap("beforeend"),
		g.Text(getUserInitial(userName)),
	)
}

func hasUnreadIndicator() g.Node {
	return Div(
		Class("bg-green-500 rounded-full w-3 h-3"),
	)
}

func UnreadIndicatorSwapOOB(hasUnread bool) g.Node {
	return Div(
		hx.SwapOOB("true"),
		ID("message-unread-indicator"),
		g.If(hasUnread, Div(Class("absolute -top-1 -right-3"), hasUnreadIndicator())),
	)
}

func navLoggedIn(userName string, hasUnread bool) g.Node {
	return Div(
		ID("user-avatar-container"),
		Class("relative"),
		avatar(userName),
		UnreadIndicatorSwapOOB(hasUnread),
	)
}

func loginNode(returnPath string) g.Node {
	href := "/login"
	if returnPath != "" && returnPath != "/login" &&
		returnPath != "/register" &&
		!strings.HasPrefix(returnPath, "/register/") {
		href = "/login?return=" + url.QueryEscape(returnPath)
	}
	return A(
		Href(href),
		Class("inline-flex items-center gap-1.5 text-blue-600 dark:text-blue-400 hover:underline"),
		Img(
			Src("/images/login.svg"),
			Alt(""),
			Class("w-5 h-5 dark:invert dark:opacity-80"),
		),
		g.Text("Login"),
	)
}

func navLoggedOut(currentPath string) g.Node {
	switch currentPath {
	case "/login", "/register/verify", "/recover":
		return nil
	default:
		return loginNode(currentPath)
	}
}

func indicator() g.Node {
	return Div(
		ID("indicator"),
		Class("htmx-indicator nav-loading-bar"),
		Div(Class("nav-loading-bar-glow")),
	)
}

func RemoveIntroBanner() g.Node {
	return Div(
		ID("intro-banner"),
		hx.SwapOOB("delete"),
	)
}

func introBanner() g.Node {
	return Div(
		ID("intro-banner"),
		Class("relative -mx-4 px-4 h-8 flex items-center "+
			"justify-center bg-yellow-400 "+
			"border-b border-yellow-400 "+
			"text-zinc-700"),
		A(
			Class("font-medium text-zinc-900 "+
				"hover:underline cursor-pointer"),
			hx.Get("/api/intro-banner/dismiss?redirect=/about"),
			hx.Swap("none"),
			g.Textf("What is %s?", config.ServerName),
		),
		Button(
			Type("button"),
			g.Attr("aria-label", "Close"),
			Class("absolute right-4 top-1/2 -translate-y-1/2 "+
				"w-6 h-6 flex items-center justify-center "+
				"rounded-full hover:bg-amber-200/60 "+
				"dark:hover:bg-amber-900/50 cursor-pointer "+
				"text-lg leading-none"),
			hx.Get("/api/intro-banner/dismiss"),
			hx.Swap("none"),
			g.Text("×"),
		),
	)
}

func navigation(userID int, userName, currentPath string,
	hasUnread bool) g.Node {
	return Nav(
		ID("main-nav"),
		Class("sticky top-0 z-10 relative "+
			"bg-white/75 dark:bg-zinc-900/75 backdrop-blur-xl "+
			"border-b border-zinc-200 dark:border-zinc-700 "+
			"flex items-center justify-between mb-4 py-4 -mx-4 px-4"),
		A(
			Href("/"),
			Class("flex items-center gap-2 shrink-0 whitespace-nowrap"),
			Img(
				Src("/images/rock.svg"),
				Alt(""),
				Class("w-9 h-9 flex-shrink-0"),
			),
			Span(Class("text-xl font-bold"), g.Text(config.ServerName)),
		),
		Div(
			Class("flex items-center gap-3 shrink-0"),
			g.Iff(local.IsLoggedIn(userID), func() g.Node { return navLoggedIn(userName, hasUnread) }),
			g.Iff(!local.IsLoggedIn(userID), func() g.Node { return navLoggedOut(currentPath) }),
		),
		indicator(),
	)
}

func SiteFooter() g.Node {
	linkClass := "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 " +
		"dark:hover:text-zinc-200 hover:underline"
	return Footer(
		Class("mt-16 pt-6 border-t border-zinc-200 dark:border-zinc-700 "+
			"flex flex-wrap gap-x-4 gap-y-2 text-sm"),
		A(Href("/about"), Class(linkClass), g.Text("About")),
		A(Href("/privacy"), Class(linkClass), g.Text("Privacy")),
		A(Href("/terms"), Class(linkClass), g.Text("Terms")),
		A(Href("/faq"), Class(linkClass), g.Text("FAQ")),
		A(
			Href("mailto:"+config.ContactEmail),
			Class(linkClass),
			g.Text("Contact"),
		),
	)
}

func Page(userID int, hasUnread bool, userName, title, currentPath,
	csrfToken string, showIntroBanner bool, body []g.Node) g.Node {
	docTitle := config.ServerName
	if title != "" {
		docTitle = config.ServerName + " - " + title
	}

	var headNodes []g.Node
	headNodes = append(headNodes, seoHead(docTitle, currentPath)...)

	// Favicon
	headNodes = append(headNodes,
		Link(
			Rel("icon"),
			Type("image/svg+xml"),
			Href("/images/rock.svg"),
		),
	)

	// Stylesheets
	headNodes = append(headNodes,
		Link(
			Rel("stylesheet"),
			Href("/css/output.css"),
		),
		Script(
			Type("text/javascript"),
			Src(config.HTMXURL),
			Defer(),
		),
		Script(
			Type("text/javascript"),
			Src(config.HTMXSSEURL),
			Defer(),
		),
		Script(
			Type("text/javascript"),
			Src("/js/timezone.js"),
			Defer(),
		),
		Script(
			Type("text/javascript"),
			Src("/js/theme.js"),
			Defer(),
		),
	)

	// Properly escape the CSRF token for JSON
	headersJSON := fmt.Sprintf(`{"X-Csrf-Token": %q}`, csrfToken)

	contentClass := "w-full md:max-w-3xl md:mx-auto pb-8 px-6 " +
		"text-zinc-900 dark:text-zinc-200 bg-white dark:bg-zinc-900"

	bodyNodes := []g.Node{
		Class("min-h-screen bg-white dark:bg-zinc-900"),
		Div(
			ID("page-content"),
			g.If(local.IsLoggedIn(userID), hx.Ext("sse")),
			g.If(local.IsLoggedIn(userID), g.Attr("sse-connect", "/auth/sse")),
			g.If(local.IsLoggedIn(userID), g.Attr("sse-close", "close")),
			g.If(local.IsLoggedIn(userID), UnreadSSESink()),
			Div(
				Class(contentClass),
				hx.Headers(headersJSON),
				hx.Indicator("#indicator"),
				g.If(showIntroBanner, introBanner()),
				navigation(userID, userName, currentPath, hasUnread),
				g.Group(body),
				SiteFooter(),
			),
		),
	}

	return components.HTML5(components.HTML5Props{
		Title:    docTitle,
		Language: "en",
		Head:     headNodes,
		Body:     bodyNodes,
	})
}
