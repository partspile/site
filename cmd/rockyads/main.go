// rockyads is the rockyads.com stand-in. Rocky Ads was renamed to
// Parts Pile; this serves a short landing page with a link to
// PUBLIC_SITE_URL (no auto-redirect).
package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	base, err := parseBase(os.Getenv("PUBLIC_SITE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	addr := ":" + port
	log.Printf("listening on %s, landing → %s", addr, homeURL(base))
	srv := &http.Server{
		Addr:              addr,
		Handler:           newMux(base),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func newMux(base *url.URL) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok\n")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, landingHTML(homeURL(base)))
	})
	return mux
}

func parseBase(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil ||
		(u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf(
			"PUBLIC_SITE_URL must be an absolute http(s) URL")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

func homeURL(base *url.URL) string {
	u := *base
	u.Path = "/"
	return u.String()
}

func landingHTML(partsPileURL string) string {
	href := html.EscapeString(partsPileURL)
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Rocky Ads is now Parts Pile</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,600;9..144,700&family=Source+Sans+3:wght@400;600&display=swap" rel="stylesheet">
<style>
:root {
  --ink: #0f1c2e;
  --muted: #4a5a6d;
  --band: #16324f;
  --accent: #1f7a6c;
  --accent-ink: #f4fffc;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 1.25rem;
  font-family: "Source Sans 3", "Segoe UI", sans-serif;
  color: var(--ink);
  background:
    radial-gradient(ellipse 90% 70% at 15% 0%,
      rgba(31, 122, 108, 0.18), transparent 55%),
    radial-gradient(ellipse 80% 60% at 100% 100%,
      rgba(22, 50, 79, 0.16), transparent 50%),
    linear-gradient(160deg, #c9d6e3 0%, #dbe4ee 40%, #b7c7d8 100%);
}
main {
  width: min(34rem, 100%);
  text-align: center;
}
.brand {
  font-family: Fraunces, Georgia, serif;
  font-size: clamp(2.4rem, 8vw, 3.5rem);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.05;
  margin: 0 0 1.25rem;
  color: var(--band);
  animation: rise 520ms ease-out both;
}
.note {
  margin: 0 auto 1.75rem;
  max-width: 28rem;
  font-size: 1.125rem;
  line-height: 1.55;
  color: var(--muted);
  animation: rise 520ms ease-out 80ms both;
}
.note strong {
  color: var(--ink);
  font-weight: 600;
}
.cta {
  display: inline-block;
  padding: 0.85rem 1.6rem;
  border-radius: 0.4rem;
  background: var(--accent);
  color: var(--accent-ink);
  font-size: 1.05rem;
  font-weight: 600;
  text-decoration: none;
  letter-spacing: 0.01em;
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.08);
  transition: transform 160ms ease, box-shadow 160ms ease,
    background 160ms ease;
  animation: rise 520ms ease-out 160ms both;
}
.cta:hover {
  background: #18675c;
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(31, 122, 108, 0.3);
}
.cta:focus-visible {
  outline: 3px solid var(--band);
  outline-offset: 3px;
}
@keyframes rise {
  from { opacity: 0; transform: translateY(0.6rem); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
</head>
<body>
<main>
  <h1 class="brand">Rocky Ads</h1>
  <p class="note">Rocky Ads has been renamed to
  <strong>Parts Pile</strong>. Same classifieds, new name —
  find us at parts-pile.com.</p>
  <a class="cta" href="` + href + `">Go To Parts Pile</a>
</main>
</body>
</html>`
}
