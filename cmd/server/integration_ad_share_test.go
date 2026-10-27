package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/partspile/site/internal/config"
)

func TestAdShareCanonicalURLAndFlyer(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)
	loginTestUser(t, client, baseURLParsed)

	categoryCookie := &http.Cookie{
		Name: "category", Value: "6", Path: "/",
		HttpOnly: true, Secure: false,
	}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	formData := map[string]interface{}{
		"title":          "Flyer Share Ad",
		"description":    "Listing used to test share and flyer.",
		"year":           "2020",
		"price":          "15000",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
		"location":       "Los Angeles",
	}
	resp, _ := postFormRequest(t, baseURL+"/auth/ad/new", formData)
	adID := adIDFromCreateResponse(t, resp, "Flyer Share Ad")
	canonical := config.CanonicalURL("/ad/" + adID)
	flyerPath := fmt.Sprintf("/auth/ad/%s/flyer", adID)

	resp, body := getRequestWithCookies(t, client,
		fmt.Sprintf("%s/ad/%s", baseURL, adID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ad detail status %d", resp.StatusCode)
	}
	if strings.Contains(body, flyerPath) {
		t.Fatal("ad detail should not include flyer link")
	}

	shareURL := fmt.Sprintf("%s/api/ad/%s/share", baseURL, adID)
	resp, body = getRequestWithCookies(t, client, shareURL)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("share modal status %d", resp.StatusCode)
	}
	if !strings.Contains(body, canonical) {
		t.Fatalf("share modal missing CanonicalURL %q", canonical)
	}
	if !strings.Contains(body, flyerPath) {
		t.Fatal("owner share modal missing flyer link")
	}
	if !strings.Contains(body, "Print flyer") {
		t.Fatal("owner share modal missing print flyer button")
	}
	if strings.Contains(body, "flyer-thumb") {
		t.Fatal("share modal should not include flyer thumbnail")
	}

	resp, body = getRequestWithCookies(t, client, baseURL+flyerPath)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner flyer status %d", resp.StatusCode)
	}
	for _, want := range []string{
		"Flyer Share Ad",
		canonical,
		"data:image/png;base64,",
		"Print",
		"flyer-screen-only",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("flyer missing %q", want)
		}
	}
	if strings.Contains(body, `id="main-nav"`) {
		t.Fatal("flyer should not include site navigation")
	}

	resp, err := http.Get(shareURL)
	if err != nil {
		t.Fatal(err)
	}
	guestShare := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("guest share modal status %d", resp.StatusCode)
	}
	if strings.Contains(guestShare, flyerPath) ||
		strings.Contains(guestShare, "Print flyer") {
		t.Fatal("guest share modal should not include flyer")
	}

	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err = noRedirect.Get(baseURL + flyerPath)
	if err != nil {
		t.Fatal(err)
	}
	guestBody := readBody(t, resp)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("guest flyer status %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("guest flyer redirect %q, want /login", loc)
	}
	if strings.Contains(guestBody, "flyer-screen-only") {
		t.Fatal("guest should not receive flyer page")
	}

	resp, err = http.Get(baseURL + fmt.Sprintf("/ad/%s/flyer", adID))
	if err != nil {
		t.Fatal(err)
	}
	oldPathBody := readBody(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public flyer path status %d", resp.StatusCode)
	}
	if strings.Contains(oldPathBody, "flyer-screen-only") {
		t.Fatal("public flyer path should not serve flyer")
	}

	resp, loginBody := postFormRequest(t, baseURL+"/api/login",
		map[string]interface{}{
			"username": "admin",
			"password": "admin",
		})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login status %d body=%v", resp.StatusCode, loginBody)
	}
	resp, body = getRequestWithCookies(t, client, shareURL)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("other-user share modal status %d", resp.StatusCode)
	}
	if strings.Contains(body, flyerPath) ||
		strings.Contains(body, "Print flyer") {
		t.Fatal("other-user share modal should not include flyer")
	}
	resp, body = getRequestWithCookies(t, client, baseURL+flyerPath)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("other-user flyer status %d", resp.StatusCode)
	}
	if strings.Contains(body, "flyer-screen-only") {
		t.Fatal("other user should not receive flyer page")
	}
}
