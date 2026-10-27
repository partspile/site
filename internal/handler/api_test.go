package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
)

func TestCategorySwitchRedirect(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(categorySwitchRedirect(c))
	})

	tests := []struct {
		query string
		want  string
	}{
		{"", "/"},
		{"return=/", "/"},
		{"return=/auth/ad/new", "/auth/ad/new"},
		{"return=//evil.com", "/"},
	}
	for _, tt := range tests {
		path := "/"
		if tt.query != "" {
			path += "?" + tt.query
		}
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body := make([]byte, 64)
		n, _ := resp.Body.Read(body)
		got := string(body[:n])
		if got != tt.want {
			t.Errorf("query %q: got %q, want %q", tt.query, got, tt.want)
		}
	}
}

func TestSwitchCategorySkipsSearchStateOffHome(t *testing.T) {
	want := cookie.SearchState{
		Location: "Denver, CO",
		Within:   25,
		Q:        "Honda",
	}
	app := fiber.New()
	app.Get("/seed", func(c *fiber.Ctx) error {
		cookie.SetSearchState(c, want)
		return c.SendStatus(fiber.StatusOK)
	})
	var saved cookie.SearchState
	app.Get("/api/category/:category/switch", func(c *fiber.Ctx) error {
		redirect := categorySwitchRedirect(c)
		if redirect == "/" {
			state := saveSearchStateFromRequest(c, nil, true)
			cookie.SetSearchState(c, state)
		}
		saved = cookie.GetSearchState(c)
		return c.SendStatus(fiber.StatusFound)
	})

	seedResp, err := app.Test(httptest.NewRequest("GET", "/seed", nil))
	if err != nil {
		t.Fatal(err)
	}
	var cookieVal string
	for _, c := range seedResp.Cookies() {
		if c.Name == "search" {
			cookieVal = c.Value
			break
		}
	}

	req := httptest.NewRequest("GET", "/api/category/5/switch?return=%2Fauth%2Fad%2Fnew", nil)
	req.Header.Set("Cookie", "search="+cookieVal)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	if saved.Location != "Denver, CO" || saved.Within != 25 || saved.Q != "Honda" {
		t.Fatalf("expected search cookie untouched, got %+v", saved)
	}
}

func TestShortCategoryHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/c/:category", ShortCategoryHandler)

	resp, err := app.Test(httptest.NewRequest("GET", "/c/4", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/" {
		t.Fatalf("Location = %q, want /", got)
	}
	// Cookie value needs loaded categories (see integration TestShortCategoryRoute).
	// Locals + Set-Cookie overwrite is covered by cookie.TestSetCategoryID*.
}
