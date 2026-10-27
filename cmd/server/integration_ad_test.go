package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
)

func TestCreateAdWithMileage(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)

	clearCookie := &http.Cookie{Name: "auth_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: false}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{clearCookie})

	resp, _ := postFormRequest(t, baseURL+"/api/login", map[string]interface{}{
		"username": "test",
		"password": "test",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed with status %d", resp.StatusCode)
	}

	categoryCookie := &http.Cookie{Name: "category", Value: "6", Path: "/", HttpOnly: true, Secure: false}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	formData := map[string]interface{}{
		"title":          "Test Car Ad",
		"description":    "A test vehicle listing.",
		"year":           "2020",
		"price":          "15000",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
		"location":       "Los Angeles",
	}

	resp, result := postFormRequest(t, baseURL+"/auth/ad/new", formData)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected redirect or success, got %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusOK {
		raw, _ := result["raw"].(string)
		if !strings.Contains(raw, "Test Car Ad") {
			t.Fatalf("expected created ad page, body=%q", raw)
		}
		return
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/ad/") {
		t.Fatalf("expected redirect to ad page, got %q", loc)
	}
}

func TestCreateAdWithImages(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)
	loginTestUser(t, client, baseURLParsed)

	categoryCookie := &http.Cookie{
		Name: "category", Value: "6", Path: "/",
		HttpOnly: true, Secure: false,
	}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	fields := map[string]interface{}{
		"title":          "Ad With Images",
		"description":    "Listing with uploaded photos.",
		"year":           "2020",
		"price":          "15000",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
		"location":       "Los Angeles",
	}
	resp, result := postAdUploadForm(t, baseURL+"/auth/ad/new", fields)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d result=%v", resp.StatusCode, result)
	}
	adIDFloat, ok := result["adId"].(float64)
	if !ok {
		t.Fatalf("missing adId: %v", result)
	}
	adID := int(adIDFloat)

	presignResp, presign := postJSONRequest(t,
		fmt.Sprintf("%s/auth/ad/%d/presign-images", baseURL, adID),
		map[string]any{"count": 2})
	if presignResp.StatusCode != http.StatusOK {
		t.Fatalf("presign: %d %v", presignResp.StatusCode, presign)
	}
	uploads, _ := presign["uploads"].([]interface{})
	if len(uploads) != 6 {
		t.Fatalf("expected 6 upload URLs, got %d", len(uploads))
	}

	putLocalAdImages(t, adID, 1, 2)

	confirmResp, confirm := postJSONRequest(t,
		fmt.Sprintf("%s/auth/ad/%d/confirm-images", baseURL, adID),
		map[string]any{"imageCount": 2})
	if confirmResp.StatusCode != http.StatusOK {
		t.Fatalf("confirm: %d %v", confirmResp.StatusCode, confirm)
	}

	var imageCount int
	if err := db.QueryRow(
		"SELECT image_count FROM ads WHERE id = $1", adID,
	).Scan(&imageCount); err != nil {
		t.Fatal(err)
	}
	if imageCount != 2 {
		t.Fatalf("expected image_count 2, got %d", imageCount)
	}

	for _, size := range []string{"160w", "480w", "1200w"} {
		for idx := 1; idx <= 2; idx++ {
			path := filepath.Join(
				testImageDir, fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.jpg", idx, size),
			)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("missing image file %s: %v", path, err)
			}
		}
	}

	detail, err := http.Get(baseURL + fmt.Sprintf("/ad/%d", adID))
	if err != nil {
		t.Fatal(err)
	}
	defer detail.Body.Close()
	body, _ := io.ReadAll(detail.Body)
	html := string(body)
	if strings.Contains(html, "/ad/") && strings.Contains(html, "/image/") {
		// proxy path should be gone from img src
		if strings.Contains(html, fmt.Sprintf("/ad/%d/image/", adID)) {
			t.Fatalf("expected no proxy image URLs, body snippet has /ad/%d/image/", adID)
		}
	}
	if !strings.Contains(html, "127.0.0.1/local/") {
		t.Fatalf("expected LocalStore PresignGet URL in HTML")
	}
}

func TestCreateAdImageValidation(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)
	loginTestUser(t, client, baseURLParsed)

	categoryCookie := &http.Cookie{
		Name: "category", Value: "6", Path: "/",
		HttpOnly: true, Secure: false,
	}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	fields := map[string]interface{}{
		"title":          "Presign Limit Ad",
		"description":    "Too many images.",
		"year":           "2020",
		"price":          "15000",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
	}
	resp, result := postAdUploadForm(t, baseURL+"/auth/ad/new", fields)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, result)
	}
	adID := int(result["adId"].(float64))

	presignResp, presign := postJSONRequest(t,
		fmt.Sprintf("%s/auth/ad/%d/presign-images", baseURL, adID),
		map[string]any{"count": config.MaxImagesPerAd + 1})
	if presignResp.StatusCode == http.StatusOK {
		t.Fatal("expected too-many-images failure")
	}
	errMsg, _ := presign["error"].(string)
	raw, _ := presign["raw"].(string)
	if !strings.Contains(errMsg+raw, "too many") {
		t.Fatalf("expected too many images, got %v", presign)
	}
}

func TestUpdateAd(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)

	clearCookie := &http.Cookie{Name: "auth_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: false}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{clearCookie})

	resp, _ := postFormRequest(t, baseURL+"/api/login", map[string]interface{}{
		"username": "test",
		"password": "test",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed with status %d", resp.StatusCode)
	}

	categoryCookie := &http.Cookie{Name: "category", Value: "6", Path: "/", HttpOnly: true, Secure: false}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	createData := map[string]interface{}{
		"title":          "Before edit",
		"description":    "Original listing text.",
		"year":           "2020",
		"price":          "3400",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
	}
	resp, _ = postFormRequest(t, baseURL+"/auth/ad/new", createData)
	adID := adIDFromCreateResponse(t, resp, "Before edit")

	editData := map[string]interface{}{
		"title":          "After edit",
		"description":    "Has the 2.0L engine.",
		"price":          "3000",
		"price_currency": "USD",
		"year":           "2020",
		"mileage":        "12000",
		"mileage_unit":   "mi",
	}
	resp, _ = postFormRequest(t, baseURL+"/auth/ad/"+adID+"/edit", editData)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		t.Fatalf("update ad: expected redirect or success, got %d", resp.StatusCode)
	}

	var desc string
	if err := db.QueryRow(
		"SELECT description FROM ads WHERE id = $1", adID,
	).Scan(&desc); err != nil {
		t.Fatal(err)
	}
	parts := ad.ParseDescriptionForDisplay(desc)
	if parts.Original != "Has the 2.0L engine." {
		t.Errorf("original = %q, want rewritten description", parts.Original)
	}
	if parts.Body != "Has the 2.0L engine." {
		t.Errorf("body = %q, want rewritten description", parts.Body)
	}
	display := ad.DisplayDescription(desc)
	for _, want := range []string{
		"Has the 2.0L engine.",
		ad.DescriptionChangeLabel,
		"Title change",
		"Price change",
		"Price dropped",
	} {
		if !strings.Contains(display, want) {
			t.Errorf("description missing %q: %q", want, display)
		}
	}
	if strings.Contains(display, ad.DescriptionAdditionLabel) {
		t.Errorf("legacy addition label in rewritten description: %q", display)
	}
}

func TestUpdateAdAppendImages(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)
	loginTestUser(t, client, baseURLParsed)

	categoryCookie := &http.Cookie{
		Name: "category", Value: "6", Path: "/",
		HttpOnly: true, Secure: false,
	}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	createFields := map[string]interface{}{
		"title":          "Edit Append Images",
		"description":    "Original listing text.",
		"year":           "2020",
		"price":          "3400",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
	}
	resp, result := postAdUploadForm(t, baseURL+"/auth/ad/new", createFields)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, result)
	}
	adID := int(result["adId"].(float64))
	putLocalAdImages(t, adID, 1, 1)
	confirmResp, _ := postJSONRequest(t,
		fmt.Sprintf("%s/auth/ad/%d/confirm-images", baseURL, adID),
		map[string]any{"imageCount": 1})
	if confirmResp.StatusCode != http.StatusOK {
		t.Fatal("confirm first image")
	}

	editFields := map[string]interface{}{
		"title":          "Edit Append Images",
		"description":    "Original listing text.",
		"year":           "2020",
		"price":          "3400",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
	}
	resp, result = postAdUploadForm(t,
		fmt.Sprintf("%s/auth/ad/%d/edit", baseURL, adID), editFields)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit: %d %v", resp.StatusCode, result)
	}

	presignResp, presign := postJSONRequest(t,
		fmt.Sprintf("%s/auth/ad/%d/presign-images", baseURL, adID),
		map[string]any{"count": 1})
	if presignResp.StatusCode != http.StatusOK {
		t.Fatalf("presign: %d %v", presignResp.StatusCode, presign)
	}
	putLocalAdImages(t, adID, 2, 1)
	confirmResp, _ = postJSONRequest(t,
		fmt.Sprintf("%s/auth/ad/%d/confirm-images", baseURL, adID),
		map[string]any{"imageCount": 2})
	if confirmResp.StatusCode != http.StatusOK {
		t.Fatal("confirm second image")
	}

	var imageCount int
	var desc string
	if err := db.QueryRow(
		"SELECT image_count, description FROM ads WHERE id = $1", adID,
	).Scan(&imageCount, &desc); err != nil {
		t.Fatal(err)
	}
	if imageCount != 2 {
		t.Fatalf("expected image_count 2, got %d", imageCount)
	}
	display := ad.DisplayDescription(desc)
	if !strings.Contains(display, "Images Added") {
		t.Errorf("description missing Images Added: %q", display)
	}
	parts := ad.ParseDescriptionForDisplay(desc)
	if len(parts.History) == 0 ||
		len(parts.History[0].ImageIndices) != 1 ||
		parts.History[0].ImageIndices[0] != 2 {
		t.Fatalf("history indices = %v", parts.History)
	}

	path := filepath.Join(
		testImageDir, fmt.Sprintf("%d", adID), "2-160w.jpg",
	)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("missing appended image file %s: %v", path, err)
	}
}

func TestAdTagsPersistViaMultipartForm(t *testing.T) {
	client := getTestClient()
	baseURLParsed, _ := url.Parse(baseURL)
	loginTestUser(t, client, baseURLParsed)

	categoryCookie := &http.Cookie{Name: "category", Value: "6", Path: "/", HttpOnly: true, Secure: false}
	client.Jar.SetCookies(baseURLParsed, []*http.Cookie{categoryCookie})

	createData := map[string]interface{}{
		"title":          "Tagged Ad",
		"description":    "A test vehicle listing.",
		"year":           "2020",
		"price":          "15000",
		"price_currency": "USD",
		"mileage":        "12000",
		"mileage_unit":   "mi",
		"location":       "Los Angeles",
		"suggestion": []string{
			ad.EncodeSuggestionFormValue("fuel", "diesel"),
			ad.EncodeSuggestionFormValue("clean title", "yes"),
		},
	}

	resp, _ := postFormRequest(t, baseURL+"/auth/ad/new", createData)
	adID := adIDFromCreateResponse(t, resp, "Tagged Ad")

	if tags := adTagsJSON(t, adID); tags == "[]" || tags == "" {
		t.Fatalf("tags not saved on create, got %q", tags)
	}

	editData := map[string]interface{}{
		"title":          "Tagged Ad",
		"description":    "A test vehicle listing.",
		"price":          "15000",
		"price_currency": "USD",
		"year":           "2020",
		"mileage":        "12000",
		"mileage_unit":   "mi",
		"suggestion": []string{
			ad.EncodeSuggestionFormValue("fuel", "diesel"),
		},
	}
	resp, _ = postFormRequest(t, baseURL+"/auth/ad/"+adID+"/edit", editData)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		t.Fatalf("update ad: expected redirect or success, got %d", resp.StatusCode)
	}
	if tags := adTagsJSON(t, adID); tags == "[]" || tags == "" {
		t.Fatalf("tags not preserved on edit, got %q", tags)
	}
}

func TestIntegrationGetAdNullLocationID(t *testing.T) {
	var adID int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id, location_id,
		 expires_at)
		VALUES ($1, 'No location ad', 'desc', $2, NULL,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationCarsCategory, integrationTestUserID).Scan(&adID)
	if err != nil {
		t.Fatal(err)
	}

	loc, _ := time.LoadLocation("America/Los_Angeles")
	got, err := ad.GetAd(0, adID, loc)
	if err != nil {
		t.Fatalf("GetAd(%d) failed: %v", adID, err)
	}
	if got.LocationID != nil {
		t.Fatalf("expected nil location_id, got %v", *got.LocationID)
	}
}
