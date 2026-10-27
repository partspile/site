package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/partspile/site/internal/user"
)

func TestSharedUserProfile(t *testing.T) {
	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := http.Get(baseURL + "/u/not-valid")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("invalid token status %d", resp.StatusCode)
	}

	resp, err = http.Get(baseURL + "/u/AAAAAAAAAAAAAAAAAAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown token status %d", resp.StatusCode)
	}

	resp, err = noRedirect.Get(baseURL + "/auth/user/1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("logged-out profile status %d", resp.StatusCode)
	}

	client := getTestClient()
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	loginTestUser(t, client, baseURLParsed)

	resp, body := getRequestWithCookies(t, client, baseURL+"/auth/user/1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("own profile status %d", resp.StatusCode)
	}
	if !strings.Contains(body, "/auth/user/share") {
		t.Fatal("own profile missing share button")
	}

	resp, body = getRequestWithCookies(t, client, baseURL+"/auth/user/2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("other profile status %d", resp.StatusCode)
	}
	if strings.Contains(body, "/auth/user/share") {
		t.Fatal("other profile should not show share button")
	}

	resp, body = getRequestWithCookies(t, client, baseURL+"/auth/user/share")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("share modal status %d", resp.StatusCode)
	}
	token := shareTokenFromHTML(t, body)

	resp, err = noRedirect.Get(baseURL + "/u/" + token)
	if err != nil {
		t.Fatal(err)
	}
	guestBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shared profile status %d", resp.StatusCode)
	}
	if !strings.Contains(guestBody, "noindex, nofollow") {
		t.Fatal("shared profile missing noindex")
	}
	if !strings.Contains(guestBody, "Active Ads") {
		t.Fatal("shared profile missing ads")
	}
	if strings.Contains(guestBody, "/auth/user/share") {
		t.Fatal("guest shared profile should not show share button")
	}

	resp, err = noRedirect.Get(baseURL + "/u/" + token + "/view/2")
	if err != nil {
		t.Fatal(err)
	}
	viewBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shared view status %d", resp.StatusCode)
	}
	if !strings.Contains(viewBody, "user-profile-ads") {
		t.Fatal("shared view missing ads container")
	}
}

func shareTokenFromHTML(t *testing.T, body string) string {
	t.Helper()
	i := strings.Index(body, "/u/")
	if i < 0 {
		t.Fatalf("no /u/ in share modal: %s", body)
	}
	s := body[i+3:]
	end := strings.IndexAny(s, `"' <`)
	if end < 0 {
		t.Fatalf("unterminated /u/ in: %s", s)
	}
	tok := s[:end]
	if !user.ValidShareToken(tok) {
		t.Fatalf("extracted token %q", tok)
	}
	return tok
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			b = append(b, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(b)
}
