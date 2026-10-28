package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseBase(t *testing.T) {
	u, err := parseBase("https://parts-pile.com/")
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "https://parts-pile.com" {
		t.Fatalf("parseBase = %q", u.String())
	}
	if got := homeURL(u); got != "https://parts-pile.com/" {
		t.Fatalf("homeURL = %q", got)
	}

	for _, raw := range []string{
		"",
		"parts-pile.com",
		"javascript:alert(1)",
		"https://user:pass@parts-pile.com",
	} {
		if _, err := parseBase(raw); err == nil {
			t.Errorf("parseBase(%q) succeeded", raw)
		}
	}
}

func TestLandingAndHealth(t *testing.T) {
	base, err := parseBase("https://parts-pile.com")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newMux(base))
	t.Cleanup(srv.Close)

	client := srv.Client()
	client.CheckRedirect = func(
		req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	for _, path := range []string{"/", "/ads/1", "/about"} {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d", path, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "" {
			t.Fatalf("%s redirected to %q", path, loc)
		}
		html := string(body)
		if !strings.Contains(html, "has been renamed") {
			t.Fatalf("%s missing rename note", path)
		}
		if !strings.Contains(html, `href="https://parts-pile.com/"`) {
			t.Fatalf("%s missing Parts Pile link", path)
		}
		if !strings.Contains(html, "Go To Parts Pile") {
			t.Fatalf("%s missing CTA label", path)
		}
		if !strings.Contains(html, `/images/rock.svg`) {
			t.Fatalf("%s missing rock icon", path)
		}
		if !strings.Contains(html, `"Rocky" is a trademark`) {
			t.Fatalf("%s missing trademark note", path)
		}
	}

	rock, err := client.Get(srv.URL + "/images/rock.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer rock.Body.Close()
	if rock.StatusCode != http.StatusOK {
		t.Fatalf("rock.svg status = %d", rock.StatusCode)
	}
	if ct := rock.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("rock.svg content-type = %q", ct)
	}

	health, err := client.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", health.StatusCode)
	}
	body, err := io.ReadAll(health.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok\n" {
		t.Fatalf("health body = %q", body)
	}
}

func TestLandingHTMLEscapesHref(t *testing.T) {
	out := landingHTML(`https://parts-pile.com/" onclick="alert(1)`)
	if strings.Contains(out, `" onclick="`) {
		t.Fatal("href quote was not escaped")
	}
	if !strings.Contains(out, "&#34;") &&
		!strings.Contains(out, "&quot;") {
		t.Fatal("expected escaped quote in href")
	}
}
