package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/facet"
)

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestChromeDevToolsEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/.well-known/appspecific/com.chrome.devtools.json")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     int
		expectedStatus int
	}{
		{"Valid category", 6, 200},
		{"Another valid category", 5, 200},
		{"Invalid category", 999, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := getClientWithCategoryCookie(tt.categoryID)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			resp, body := getRequestWithCookies(t, client, baseURL+"/")

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				if len(body) > 500 {
					t.Logf("Response body (truncated): %s...", body[:500])
				} else {
					t.Logf("Response body: %s", body)
				}
			}

			if tt.expectedStatus == 200 {
				contentType := resp.Header.Get("Content-Type")
				if contentType != "text/html" && contentType != "text/html; charset=utf-8" {
					t.Errorf("Expected Content-Type text/html, got %s", contentType)
				}
				if len(body) == 0 {
					t.Error("Expected non-empty HTML response")
				}

				if tt.categoryID == 999 {
					if !strings.Contains(body, "Car &amp; Truck Parts") && !strings.Contains(body, "Car & Truck Parts") {
						t.Errorf("Expected HTML to contain default category name 'Car & Truck Parts' (or HTML entity version)")
					}
				}
			}
		})
	}

	t.Run("No category cookie", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 without category cookie (should default to default category), got %d", resp.StatusCode)
		}

		hasCategoryCookie := false
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "category" {
				hasCategoryCookie = true
				break
			}
		}
		if !hasCategoryCookie {
			t.Error("Expected category cookie to be set when accessing home page without cookie")
		}
	})
}

func TestHomePageSEO(t *testing.T) {
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	if !strings.Contains(html, "Parts Pile - Car") ||
		!strings.Contains(html, "Truck Parts") {
		t.Error("expected homepage title Parts Pile - Car & Truck Parts")
	}
	if !strings.Contains(html, `rel="canonical"`) {
		t.Error("expected canonical link")
	}
	if !strings.Contains(html, "application/ld+json") {
		t.Error("expected WebSite JSON-LD")
	}
}

func TestRobotsAndSitemap(t *testing.T) {
	resp, err := http.Get(baseURL + "/robots.txt")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("robots.txt status %d", resp.StatusCode)
	}
	robots := string(body)
	if !strings.Contains(robots, "Disallow: /auth/") {
		t.Errorf("robots.txt: %s", robots)
	}
	if !strings.Contains(robots, "Disallow: /u/") {
		t.Errorf("robots.txt missing /u/: %s", robots)
	}
	if !strings.Contains(robots, "Sitemap:") {
		t.Errorf("robots.txt missing Sitemap: %s", robots)
	}

	resp, err = http.Get(baseURL + "/sitemap.xml")
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sitemap.xml status %d", resp.StatusCode)
	}
	sm := string(body)
	for _, path := range []string{"/about", "/login", "/c/6"} {
		if !strings.Contains(sm, path+"</loc>") {
			t.Errorf("sitemap missing %s\n%s", path, sm)
		}
	}
	if strings.Contains(sm, "/c/5") {
		t.Errorf("sitemap should not include default /c/5\n%s", sm)
	}
	if strings.Contains(sm, "/faq") {
		t.Errorf("sitemap should not include /faq\n%s", sm)
	}
}

func TestHomeHandlerSearchHiddenByDefault(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `id="search-toggle"`) {
		t.Error("Expected search toggle on category row")
	}
	if !strings.Contains(body, `/images/search.svg`) {
		t.Error("Expected search icon on category row")
	}
	if !strings.Contains(body, `id="search-area" class="flex flex-col hidden"`) {
		t.Error("Expected hidden #search-area on fresh home page")
	}
	if strings.Contains(body, `id="category-modal"`) ||
		strings.Contains(body, `id="category-modal-backdrop"`) {
		t.Error("home page should not embed category modal")
	}
	if !strings.Contains(body, `aria-haspopup="listbox"`) {
		t.Error("expected category picker to be a listbox dropdown")
	}
	if !strings.Contains(body, "category-select-dismiss") {
		t.Error("expected category picker outside-click dismiss")
	}
	if !strings.Contains(body, "Bicycles") {
		t.Error("expected category dropdown to list other categories")
	}
	if strings.Contains(body, `/images/post_add.svg`) {
		t.Error("new ad link should not appear when logged out")
	}
}

func TestHomeHandlerNewAdLinkWhenLoggedIn(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)
	loginTestUser(t, client, baseURLParsed)
	categoryCookie := &http.Cookie{
		Name: "category", Value: "6", Path: "/", HttpOnly: true, Secure: false,
	}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})
	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `/images/post_add.svg`) {
		t.Error("Expected new ad icon on category row when logged in")
	}
	if !strings.Contains(body, `href="/auth/ad/new"`) {
		t.Error("Expected new ad link to /auth/ad/new when logged in")
	}
}

func TestToggleSearchHandler(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	resp, body := getRequestWithCookies(t, client, baseURL+"/api/toggle-search")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("toggle-search: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `id="searchBox"`) {
		t.Error("toggle-search should reveal search box")
	}
	if !strings.Contains(body, `id="search-toggle"`) || !strings.Contains(body, `id="search-toggle" class="`) || !strings.Contains(body, " hidden") {
		t.Error("toggle-search should hide search icon when search is open")
	}
	if !strings.Contains(body, `hx-swap-oob="outerHTML"`) {
		t.Error("toggle-search should OOB-swap search area and toggle")
	}
}

func TestHomeHandlerFiltersExpanded(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if err := setSearchCookieOnClient(client, cookie.SearchState{
		Q: "Honda", Expanded: true,
	}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}
	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "filter-price-min") {
		t.Error("Expected expanded filter panel with price field")
	}
	if !strings.Contains(body, `id="search-area"`) {
		t.Error("Expected #search-area on home page")
	}
	if !strings.Contains(body, `id="search-location"`) {
		t.Error("Expected #search-location on home page")
	}
	if !strings.Contains(body, `id="filter-toggle"`) {
		t.Error("Expected filter toggle on home page")
	}
	if !strings.Contains(body, `title="Collapse filters"`) {
		t.Error("Expected expanded filter toggle (^) when search cookie expanded")
	}
}

func TestSwitchCategoryHandler(t *testing.T) {
	tests := []struct {
		name                   string
		categoryID             string
		returnParam            string
		queryParams            string
		expectedStatus         int
		expectRedirect         string
		expectRedirectContains []string
	}{
		{"Valid category", "6", "", "", 302, "/", nil},
		{"Valid category with return", "5", "/auth/ad/new", "", 302, "/auth/ad/new", nil},
		{"Invalid category ID defaults to default category", "999", "", "", 302, "/", nil},
		{
			name:           "Redirect URL has no filter query params",
			categoryID:     "6",
			queryParams:    "q=Honda&price_min=10000",
			expectedStatus: 302,
			expectRedirect: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatalf("Failed to create cookie jar: %v", err)
			}
			client := &http.Client{
				Jar: jar,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			requestURL := fmt.Sprintf("%s/api/category/%s/switch", baseURL, tt.categoryID)
			q := url.Values{}
			if tt.returnParam != "" {
				q.Set("return", tt.returnParam)
			}
			if tt.queryParams != "" {
				for _, part := range strings.Split(tt.queryParams, "&") {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) == 2 {
						q.Set(kv[0], kv[1])
					}
				}
			}
			if encoded := q.Encode(); encoded != "" {
				requestURL += "?" + encoded
			}
			resp, _ := getRequestWithCookies(t, client, requestURL)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedStatus == 302 {
				redirect := resp.Header.Get("Location")
				if tt.expectRedirect != "" && redirect != tt.expectRedirect {
					t.Errorf("Expected Location %q, got %q", tt.expectRedirect, redirect)
				}
				for _, s := range tt.expectRedirectContains {
					if !strings.Contains(redirect, s) {
						t.Errorf("Expected Location to contain %q, got %q", s, redirect)
					}
				}

				hasCategoryCookie := false
				var cookieValue string
				for _, c := range resp.Cookies() {
					if c.Name == "category" {
						hasCategoryCookie = true
						cookieValue = c.Value
						break
					}
				}
				if !hasCategoryCookie {
					t.Error("Expected category cookie to be set")
				}
				expectedCookie := tt.categoryID
				if tt.categoryID == "999" {
					expectedCookie = "5"
				}
				if cookieValue != expectedCookie {
					t.Errorf("Expected category cookie value %s, got %s", expectedCookie, cookieValue)
				}
			}
		})
	}
}

func TestShortCategoryRoute(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("Failed to create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Prior navigation left a different category cookie.
	u, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse baseURL: %v", err)
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "category", Value: "5"}})

	resp, body := getRequestWithCookies(t, client, baseURL+"/c/6")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Cars") || !strings.Contains(body, "Trucks") {
		t.Fatalf("expected Cars & Trucks title, body snippet missing")
	}

	var categoryVal string
	for _, c := range resp.Cookies() {
		if c.Name == "category" {
			categoryVal = c.Value
		}
	}
	if categoryVal != "6" {
		t.Fatalf("expected category cookie 6 (path wins over prior 5), got %q", categoryVal)
	}
}

func TestSwitchCategoryPreservesSearchCookie(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	switchURL := baseURL + "/api/category/5/switch?q=Honda&price_min=10000"
	resp, _ := getRequestWithCookies(t, client, switchURL)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("switch: expected 302, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/" {
		t.Fatalf("expected redirect /, got %q", got)
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Honda") {
		t.Error("expected home page to show search query from cookie")
	}
}

func TestSwitchCategoryFromNewAdPreservesLocation(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if err := setSearchCookieOnClient(client, cookie.SearchState{
		Location: "Denver, CO",
		Within:   25,
	}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	switchURL := baseURL + "/api/category/5/switch?return=" + url.QueryEscape("/auth/ad/new")
	resp, _ := getRequestWithCookies(t, client, switchURL)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("switch: expected 302, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/auth/ad/new" {
		t.Fatalf("expected redirect /auth/ad/new, got %q", got)
	}

	state, err := getSearchStateFromClient(client)
	if err != nil {
		t.Fatalf("read search cookie: %v", err)
	}
	if state.Location != "Denver, CO" || state.Within != 25 {
		t.Fatalf("expected location preserved, got %+v", state)
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home: expected 200, got %d", resp.StatusCode)
	}
	if strings.Contains(body, "No location set") {
		t.Error("expected home page to keep saved location after new-ad category switch")
	}
}

func TestSwitchCategoryPreservesSearchOpen(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if err := setSearchCookieOnClient(client, cookie.SearchState{
		SearchOpen: true,
	}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}

	switchURL := baseURL + "/api/category/5/switch"
	resp, _ := getRequestWithCookies(t, client, switchURL)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch: expected 200, got %d", resp.StatusCode)
	}

	state, err := getSearchStateFromClient(client)
	if err != nil {
		t.Fatalf("read search cookie: %v", err)
	}
	if !state.SearchOpen {
		t.Fatal("expected SearchOpen preserved after category switch")
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home: expected 200, got %d", resp.StatusCode)
	}
	if strings.Contains(body, `id="search-area" class="flex flex-col hidden"`) {
		t.Error("expected search bar to stay visible after category switch")
	}
}

func TestSwitchCategoryPreservesExpanded(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if err := setSearchCookieOnClient(client, cookie.SearchState{
		SearchOpen: true,
		Expanded:   true,
	}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}

	switchURL := baseURL + "/api/category/5/switch"
	resp, _ := getRequestWithCookies(t, client, switchURL)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch: expected 200, got %d", resp.StatusCode)
	}

	state, err := getSearchStateFromClient(client)
	if err != nil {
		t.Fatalf("read search cookie: %v", err)
	}
	if !state.Expanded {
		t.Fatal("expected Expanded preserved after category switch")
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `title="Collapse filters"`) {
		t.Error("expected expanded filter panel after category switch")
	}
}

func TestSearchPaginationKeepsSearchHidden(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	resp, _ := getRequestWithCookies(t, client, baseURL+"/api/search/?page=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search pagination: expected 200, got %d", resp.StatusCode)
	}
	state, err := getSearchStateFromClient(client)
	if err != nil {
		t.Fatalf("read search cookie: %v", err)
	}
	if state.SearchOpen {
		t.Fatal("scroll pagination should not open search bar")
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `id="search-area" class="flex flex-col hidden"`) {
		t.Error("expected search area hidden after scroll pagination request")
	}
}

func TestMultipleCategorySwitchesKeepSearchHidden(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Simulate scroll sentinel firing while browsing.
	resp, _ := getRequestWithCookies(t, client, baseURL+"/api/search/?page=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search pagination: expected 200, got %d", resp.StatusCode)
	}

	categories := []int{5, 4, 3}
	for _, id := range categories {
		switchURL := fmt.Sprintf("%s/api/category/%d/switch", baseURL, id)
		resp, _ = getRequestWithCookies(t, client, switchURL)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("switch to category %d: expected 200, got %d", id, resp.StatusCode)
		}
		state, err := getSearchStateFromClient(client)
		if err != nil {
			t.Fatalf("read search cookie: %v", err)
		}
		if state.SearchOpen {
			t.Fatalf("category %d switch left SearchOpen true", id)
		}
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `id="search-area" class="flex flex-col hidden"`) {
		t.Error("expected search area hidden after multiple category switches")
	}
}

func TestHideFiltersHandler(t *testing.T) {
	client, err := getClientWithCategoryCookie(6)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	min := 10000
	if err := setSearchCookieOnClient(client, cookie.SearchState{
		Q: "Honda", Facets: map[string]facet.Filter{"price": {Min: &min}}, Expanded: true,
	}); err != nil {
		t.Fatalf("set search cookie: %v", err)
	}

	resp, body := getRequestWithCookies(t, client, baseURL+"/api/show-filters")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show-filters: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "filter-price-min") {
		t.Error("show-filters should render filter panel fragment")
	}
	if !strings.Contains(body, `id="filter-toggle"`) {
		t.Error("show-filters should OOB-swap filter toggle")
	}
	if !strings.Contains(body, `title="Collapse filters"`) {
		t.Error("show-filters should swap toggle to collapse (^)")
	}
	if !strings.Contains(body, `id="search-filters"`) {
		t.Error("show-filters should render filter panel with search triggers")
	}
	if !strings.Contains(body, `hx-get="/api/search/"`) ||
		!strings.Contains(body, "change delay:300ms") {
		t.Error("show-filters filter panel should trigger /api/search on input change")
	}

	hideURL := baseURL + "/api/hide-filters?q=Honda&price_min=10000"
	resp, body = getRequestWithCookies(t, client, hideURL)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hide-filters: expected 200, got %d", resp.StatusCode)
	}
	if strings.Contains(body, "filter-price-min") {
		t.Error("hide-filters should clear filter panel")
	}
	if !strings.Contains(body, `title="Expand filters"`) {
		t.Error("hide-filters should swap toggle to expand (v)")
	}
	if !strings.Contains(body, `id="search-results"`) {
		t.Error("hide-filters should OOB-swap search results")
	}
}
