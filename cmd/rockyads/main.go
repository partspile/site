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

	_ "embed"
)

//go:embed rock.svg
var rockSVG []byte

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
	mux.HandleFunc("/images/rock.svg", func(
		w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		w.Write(rockSVG)
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
<link rel="icon" type="image/svg+xml" href="/images/rock.svg">
<style>
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 1.5rem;
  font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto,
    Helvetica, Arial, sans-serif;
  color: #18181b;
  background: #fff;
}
@media (prefers-color-scheme: dark) {
  body {
    color: #e4e4e7;
    background: #18181b;
  }
}
main {
  width: min(28rem, 100%);
  text-align: center;
}
.brand {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin: 0 0 1.5rem;
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.2;
  color: inherit;
}
.brand img {
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
}
.note {
  margin: 0 auto 1.5rem;
  font-size: 1.05rem;
  line-height: 1.55;
  color: #3f3f46;
}
@media (prefers-color-scheme: dark) {
  .note { color: #a1a1aa; }
}
.note strong {
  color: #18181b;
  font-weight: 600;
}
@media (prefers-color-scheme: dark) {
  .note strong { color: #f4f4f5; }
}
.cta {
  display: inline-block;
  padding: 0.5rem 1.5rem;
  border-radius: 0.5rem;
  background: #3b82f6;
  color: #fff;
  font-size: 1rem;
  font-weight: 600;
  text-decoration: none;
  transition: background-color 150ms ease;
}
.cta:hover { background: #2563eb; }
.cta:focus-visible {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
}
.tm {
  margin: 2rem auto 0;
  max-width: 24rem;
  font-size: 0.8rem;
  line-height: 1.45;
  color: #71717a;
  text-align: left;
}
</style>
</head>
<body>
<main>
  <div class="brand">
    <img src="/images/rock.svg" alt="">
    <span>Rocky Ads</span>
  </div>
  <p class="note">Rocky Ads<sup>*</sup> has been renamed to
  <strong>Parts Pile</strong>. Same classifieds, new name —
  find us at parts-pile.com.</p>
  <a class="cta" href="` + href + `">Go To Parts Pile</a>
  <p class="tm">* "Rocky" is a trademark of others; we are switching
  to a new name.</p>
</main>
</body>
</html>`
}
