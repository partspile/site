package cookie

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/facet"
)

func TestSearchStateRoundTrip(t *testing.T) {
	min := 100
	max := 5000
	want := SearchState{
		Q:        "Honda",
		Facets:   map[string]facet.Filter{"price": {Min: &min, Max: &max}},
		Location: "Denver",
		Within:   25,
		Expanded: true,
	}
	app := fiber.New()
	var got SearchState
	app.Get("/set", func(c *fiber.Ctx) error {
		SetSearchState(c, want)
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/get", func(c *fiber.Ctx) error {
		got = GetSearchState(c)
		return c.SendStatus(fiber.StatusOK)
	})

	setResp, err := app.Test(httptest.NewRequest("GET", "/set", nil))
	if err != nil {
		t.Fatal(err)
	}
	var cookieVal string
	for _, c := range setResp.Cookies() {
		if c.Name == searchCookieName {
			cookieVal = c.Value
			break
		}
	}
	if cookieVal == "" {
		t.Fatal("expected search cookie in response")
	}

	getReq := httptest.NewRequest("GET", "/get", nil)
	getReq.Header.Set("Cookie", searchCookieName+"="+cookieVal)
	if _, err := app.Test(getReq); err != nil {
		t.Fatal(err)
	}
	if got.Q != want.Q || got.Location != want.Location ||
		got.Within != want.Within || got.Expanded != want.Expanded {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	pf, ok := got.Facets["price"]
	if !ok || pf.Min == nil || *pf.Min != min {
		t.Fatalf("price filter: got %+v", got.Facets)
	}
}

func TestGetSearchStateInvalidCookie(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		state := GetSearchState(c)
		if state.Q != "" || state.Expanded {
			t.Fatalf("expected zero state, got %+v", state)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", searchCookieName+"=not-valid-base64!!!")
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
}

func TestClearSearchState(t *testing.T) {
	app := fiber.New()
	app.Get("/set", func(c *fiber.Ctx) error {
		SetSearchState(c, SearchState{Q: "Honda", Location: "Denver"})
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/clear", func(c *fiber.Ctx) error {
		ClearSearchState(c)
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/get", func(c *fiber.Ctx) error {
		state := GetSearchState(c)
		if state.Q != "" || state.Location != "" {
			t.Fatalf("expected empty state, got %+v", state)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	setResp, err := app.Test(httptest.NewRequest("GET", "/set", nil))
	if err != nil {
		t.Fatal(err)
	}
	var cookieVal string
	for _, c := range setResp.Cookies() {
		if c.Name == searchCookieName {
			cookieVal = c.Value
			break
		}
	}
	if cookieVal == "" {
		t.Fatal("expected search cookie in response")
	}

	clearReq := httptest.NewRequest("GET", "/clear", nil)
	clearReq.Header.Set("Cookie", searchCookieName+"="+cookieVal)
	if _, err := app.Test(clearReq); err != nil {
		t.Fatal(err)
	}

	if _, err := app.Test(httptest.NewRequest("GET", "/get", nil)); err != nil {
		t.Fatal(err)
	}
}

func TestSearchStateJSON(t *testing.T) {
	min := 10
	state := SearchState{Q: "test", Facets: map[string]facet.Filter{"price": {Min: &min}}, Expanded: true}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	if _, err := base64.RawURLEncoding.DecodeString(encoded); err != nil {
		t.Fatal(err)
	}
}
