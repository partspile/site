package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/location"
	"github.com/partspile/site/internal/search"
	"github.com/partspile/site/internal/vector"
)

func TestSearchPageHandler(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     int
		query          string
		expectContains []string
		expectAbsent   []string
	}{
		{
			name:           "text query matches title",
			categoryID:     6,
			query:          "?q=Honda",
			expectContains: []string{"search-results"},
		},
		{
			name:           "price min filter",
			categoryID:     6,
			query:          "?price_min=10000",
			expectContains: []string{"search-results"},
		},
		{
			name:           "location and within",
			categoryID:     6,
			query:          "?location=Denver&within=50",
			expectContains: []string{"search-results"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := getClientWithCategoryCookie(tt.categoryID)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			if tt.query != "?q=Honda" && tt.name != "location and within" {
				if err := setSearchCookieOnClient(client, cookie.SearchState{Expanded: true}); err != nil {
					t.Fatalf("set search cookie: %v", err)
				}
			}
			url := baseURL + "/api/search/" + tt.query
			resp, body := getRequestWithCookies(t, client, url)
			if resp.StatusCode != http.StatusOK {
				snippet := body
				if len(snippet) > 200 {
					snippet = snippet[:200]
				}
				t.Fatalf("Expected status 200, got %d body=%s", resp.StatusCode, snippet)
			}
			for _, s := range tt.expectContains {
				if !strings.Contains(body, s) {
					t.Errorf("expected body to contain %q", s)
				}
			}
			for _, s := range tt.expectAbsent {
				if strings.Contains(body, s) {
					t.Errorf("expected body not to contain %q", s)
				}
			}
		})
	}
}

func TestSearchPageHandlerMileageDirect(t *testing.T) {
	data, err := json.Marshal(cookie.SearchState{Expanded: true})
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	req := httptest.NewRequest("GET", "/api/search/?mileage_min=40000&mileage_max=50000", nil)
	req.AddCookie(&http.Cookie{Name: "category", Value: "6"})
	req.AddCookie(&http.Cookie{Name: "search", Value: encoded})
	resp, err := testServer.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if !strings.Contains(body, "2020 Honda Civic") {
		t.Fatalf("expected Honda in body, got %q", body)
	}
	if strings.Contains(body, "Ford F-150") {
		t.Fatal("expected Ford excluded")
	}
}

func TestMileageSearchFilter(t *testing.T) {
	var withMileage int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ad_facets f JOIN ads a ON a.id = f.ad_id WHERE a.category_id = 6 AND f.key = 'mileage'`).Scan(&withMileage); err != nil {
		t.Fatalf("query seed mileage: %v", err)
	}
	if withMileage == 0 {
		t.Fatal("expected seeded car ads with mileage")
	}

	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if err := setSearchCookieOnClient(client, cookie.SearchState{Expanded: true}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/api/search/?mileage_min=40000&mileage_max=50000")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "2020 Honda Civic") {
		t.Error("expected mileage filter to match Honda Civic ad")
	}
	if strings.Contains(body, "Ford F-150") {
		t.Error("expected Ford F-150 to be excluded by mileage filter")
	}
}

func TestShowFiltersCategoryFacets(t *testing.T) {
	t.Run("cars show mileage filter", func(t *testing.T) {
		client, err := getClientWithCategoryCookie(6)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		if err := setSearchCookieOnClient(client, cookie.SearchState{Expanded: true}); err != nil {
			t.Fatalf("set search cookie: %v", err)
		}
		resp, body := getRequestWithCookies(t, client, baseURL+"/api/show-filters")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "filter-mileage-min") {
			t.Error("expected mileage filter for Cars & Trucks category")
		}
	})
	t.Run("parts hide mileage filter", func(t *testing.T) {
		client, err := getClientWithCategoryCookie(5)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		if err := setSearchCookieOnClient(client, cookie.SearchState{Expanded: true}); err != nil {
			t.Fatalf("set search cookie: %v", err)
		}
		resp, body := getRequestWithCookies(t, client, baseURL+"/api/show-filters")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		if strings.Contains(body, "filter-mileage-min") {
			t.Error("expected no mileage filter for parts category")
		}
	})
}

func TestSwitchCategoryClearsMileageFilter(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	min := 10000
	if err := setSearchCookieOnClient(client, cookie.SearchState{
		Q: "Honda", Facets: map[string]facet.Filter{"mileage": {Min: &min}}, Expanded: true,
	}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}

	switchURL := baseURL + "/api/category/5/switch?q=Honda&mileage_min=10000"
	resp, _ := getRequestWithCookies(t, client, switchURL)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("switch: expected 302, got %d", resp.StatusCode)
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/api/show-filters")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show-filters: expected 200, got %d", resp.StatusCode)
	}
	if strings.Contains(body, `name="mileage_min"`) && strings.Contains(body, `value="10000"`) {
		t.Error("expected mileage filter cleared after switching to parts category")
	}
}

func TestIntegrationSearchGeoAndFacetFilters(t *testing.T) {
	lat, lon, ok, err := location.ResolveLocation("Los Angeles, CA, US")
	if err != nil || !ok {
		t.Fatalf("resolve Los Angeles: ok=%v err=%v", ok, err)
	}

	p := search.Params{
		CategoryID: integrationCarsCategory,
		Expanded:   true,
		Limit:      50,
		CenterLat:  lat,
		CenterLon:  lon,
		WithinKm:   location.MilesToKm(50),
		HasGeo:     true,
	}
	result, err := search.Search(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.IDs) == 0 {
		t.Fatal("expected geo search to match seeded Los Angeles car ads")
	}

	t.Run("condition checkboxes", func(t *testing.T) {
		insertIntegrationAdWithCondition(t, "New bike", "New")
		insertIntegrationAdWithCondition(t, "Fair bike", "Used - Fair")
		insertIntegrationAdWithCondition(t, "Poor bike", "Used - Poor")
		result, err := search.Search(search.Params{
			CategoryID: integrationCarsCategory,
			Expanded:   true,
			Limit:      10,
			FacetFilters: map[string]facet.Filter{"condition": {
				Values: []string{"New", "Used - Fair"},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.IDs) < 2 {
			t.Fatalf("expected at least 2 ads, got %v", result.IDs)
		}
	})
	t.Run("mileage range", func(t *testing.T) {
		insertIntegrationAdWithMileage(t, "Low miles car", 28000)
		insertIntegrationAdWithMileage(t, "Higher miles car", 45000)
		min := 40000
		max := 50000
		result, err := search.Search(search.Params{
			CategoryID:   integrationCarsCategory,
			Expanded:     true,
			Limit:        10,
			FacetFilters: map[string]facet.Filter{"mileage": {Min: &min, Max: &max}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.IDs) == 0 {
			t.Fatalf("expected ads in mileage range, got %v", result.IDs)
		}
	})
	t.Run("sale start date range", func(t *testing.T) {
		insertIntegrationAdWithDate(t, "Early sale", "2026-06-01")
		insertIntegrationAdWithDate(t, "Late sale", "2026-06-20")
		min := "2026-06-10"
		result, err := search.Search(search.Params{
			CategoryID: integrationGarageCategory,
			Expanded:   true,
			Limit:      10,
			FacetFilters: map[string]facet.Filter{"sale_start_date": {
				TextMin: &min,
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.IDs) == 0 {
			t.Fatalf("expected ads after sale date min, got %v", result.IDs)
		}
	})
	t.Run("pricing style multi enum", func(t *testing.T) {
		insertIntegrationAdWithPricingStyle(t, "Priced sale", `["Everything Priced"]`)
		insertIntegrationAdWithPricingStyle(t, "Negotiable sale", `["Negotiable / Make Offer"]`)
		result, err := search.Search(search.Params{
			CategoryID: integrationGarageCategory,
			Expanded:   true,
			Limit:      10,
			FacetFilters: map[string]facet.Filter{"pricing_style": {
				Values: []string{"Everything Priced"},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.IDs) == 0 {
			t.Fatalf("expected priced sale ads, got %v", result.IDs)
		}
	})
}

func insertIntegrationAdWithCondition(t *testing.T, title, condition string) {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at)
		VALUES ($1, $2, 'desc', $3,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationCarsCategory, title, integrationTestUserID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, 'condition', NULL, $2)`, id, condition); err != nil {
		t.Fatal(err)
	}
	rebuildIntegrationAdVector(t, id)
}

func rebuildIntegrationAdVector(t *testing.T, adID int) {
	t.Helper()
	in, err := ad.GetForEmbedding(adID)
	if err != nil {
		t.Fatal(err)
	}
	if err := vector.BuildAdEmbedding(in); err != nil {
		t.Fatal(err)
	}
}

func insertIntegrationAdWithMileage(t *testing.T, title string, mileage int) {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at)
		VALUES ($1, $2, 'desc', $3,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationCarsCategory, title, integrationTestUserID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, 'mileage', $2, 'mi')`, id, mileage); err != nil {
		t.Fatal(err)
	}
	rebuildIntegrationAdVector(t, id)
}

func insertIntegrationAdWithDate(t *testing.T, title, date string) {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at)
		VALUES ($1, $2, 'desc', $3,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationGarageCategory, title, integrationTestUserID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, 'sale_start_date', NULL, $2)`, id, date); err != nil {
		t.Fatal(err)
	}
	rebuildIntegrationAdVector(t, id)
}

func insertIntegrationAdWithPricingStyle(t *testing.T, title, jsonVal string) {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at)
		VALUES ($1, $2, 'desc', $3,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationGarageCategory, title, integrationTestUserID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, 'pricing_style', NULL, $2)`, id, jsonVal); err != nil {
		t.Fatal(err)
	}
	rebuildIntegrationAdVector(t, id)
}
