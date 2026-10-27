package cookie

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/location"
)

func TestGetDistanceUnit(t *testing.T) {
	app := fiber.New()
	var got string

	app.Get("/", func(c *fiber.Ctx) error {
		got = GetDistanceUnit(c)
		return nil
	})

	tests := []struct {
		name    string
		cookies map[string]string
		want    string
	}{
		{
			name:    "cookie mi",
			cookies: map[string]string{"distance_unit": "mi"},
			want:    location.UnitMiles,
		},
		{
			name:    "cookie km",
			cookies: map[string]string{"distance_unit": "km"},
			want:    location.UnitKm,
		},
		{
			name:    "invalid cookie falls back to timezone",
			cookies: map[string]string{"distance_unit": "bad", "timezone": "America%2FNew_York"},
			want:    location.UnitMiles,
		},
		{
			name:    "missing cookie falls back to timezone",
			cookies: map[string]string{"timezone": "Europe%2FBerlin"},
			want:    location.UnitKm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: k, Value: v})
			}
			if _, err := app.Test(req); err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("GetDistanceUnit() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetDistanceUnitForUser(t *testing.T) {
	app := fiber.New()
	var setCookie string

	app.Post("/", func(c *fiber.Ctx) error {
		SetDistanceUnitForUser(c, "+14155552671")
		setCookie = string(c.Response().Header.Peek("Set-Cookie"))
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: "timezone", Value: "Europe%2FBerlin"})

	if _, err := app.Test(req); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !strings.Contains(setCookie, "distance_unit=mi") {
		t.Errorf("Set-Cookie = %q, want distance_unit=mi", setCookie)
	}
}
