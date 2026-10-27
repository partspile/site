package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/partspile/site/internal/accountrecovery"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/user"
)

func TestRecoverAccountFlow(t *testing.T) {
	u, err := user.GetByName("test")
	if err != nil {
		t.Fatalf("get test user: %v", err)
	}
	oldSalt := u.PasswordSalt

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}
	baseParsed, _ := url.Parse(baseURL)

	resp, err := client.Get(baseURL + "/recover")
	if err != nil {
		t.Fatalf("GET /recover: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	storeHTTPCookies(jar, baseParsed, resp.Cookies())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /recover status %d", resp.StatusCode)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("want Cache-Control no-store, got %q",
			resp.Header.Get("Cache-Control"))
	}
	html := string(body)
	if !strings.Contains(html, "Recover account") {
		t.Fatal("missing recover page title")
	}

	re := regexp.MustCompile(`RECOVER (\d{6})`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		snippet := html
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		t.Fatalf("recovery code not found in page: %s", snippet)
	}
	code := m[1]
	reload, err := client.Get(baseURL + "/recover")
	if err != nil {
		t.Fatalf("GET /recover reload: %v", err)
	}
	reloadBody, _ := io.ReadAll(reload.Body)
	reload.Body.Close()
	storeHTTPCookies(jar, baseParsed, reload.Cookies())
	if reload.StatusCode != http.StatusOK {
		t.Fatalf("GET /recover reload status %d", reload.StatusCode)
	}
	reloadCode := re.FindStringSubmatch(string(reloadBody))
	if reloadCode == nil || reloadCode[1] != code {
		t.Fatalf("reload should reuse code %s, got %q",
			code, string(reloadBody))
	}
	if !strings.Contains(html, "Text exactly") {
		t.Fatal("recover page should emphasize exact message to text")
	}
	if !strings.Contains(html, "recover-countdown") {
		t.Fatal("recover page should include live countdown")
	}

	// Pending status must not replace the waiting panel.
	pendingResp, err := client.Get(baseURL + "/api/recover/status")
	if err != nil {
		t.Fatalf("GET pending status: %v", err)
	}
	pendingResp.Body.Close()
	if pendingResp.StatusCode != http.StatusNoContent {
		t.Fatalf("pending status: want 204, got %d", pendingResp.StatusCode)
	}

	var hasRecoverCookie bool
	for _, c := range jar.Cookies(baseParsed) {
		if c.Name == "recover_session" && c.Value != "" {
			hasRecoverCookie = true
			break
		}
	}
	if !hasRecoverCookie {
		t.Fatal("missing recover_session cookie")
	}

	err = accountrecovery.CompleteFromSMS(u.PhoneE64, "RECOVER "+code)
	if err != nil {
		t.Fatalf("CompleteFromSMS: %v", err)
	}

	statusResp, err := client.Get(baseURL + "/api/recover/status")
	if err != nil {
		t.Fatalf("GET status: %v", err)
	}
	statusBody, _ := io.ReadAll(statusResp.Body)
	statusResp.Body.Close()
	storeHTTPCookies(jar, baseParsed, statusResp.Cookies())
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("status status %d", statusResp.StatusCode)
	}
	statusHTML := string(statusBody)
	if !strings.Contains(statusHTML, "test") {
		t.Fatalf("status should reveal username; got: %s", statusHTML)
	}
	if !strings.Contains(statusHTML, `name="username"`) ||
		!strings.Contains(statusHTML, "readonly") {
		t.Fatalf("status should include readonly username field; got: %s", statusHTML)
	}
	if !strings.Contains(statusHTML, "Update password") {
		t.Fatalf("status should show password form; got: %s", statusHTML)
	}
	if strings.Contains(statusHTML, "Text exactly") ||
		strings.Contains(statusHTML, "RECOVER "+code) {
		t.Fatal("verified panel should hide recovery code instructions")
	}

	const newPassword = "recovered-password-99"
	resetResp := postRecoverForm(t, client, baseURL+"/api/recover/password",
		map[string]string{
			"password":  newPassword,
			"password2": newPassword,
		})
	resetResp.Body.Close()
	storeHTTPCookies(jar, baseParsed, resetResp.Cookies())
	if resetResp.StatusCode != http.StatusOK {
		t.Fatalf("password reset status %d", resetResp.StatusCode)
	}
	if resetResp.Header.Get("HX-Redirect") != "/login" {
		t.Fatalf("want HX-Redirect /login, got %q",
			resetResp.Header.Get("HX-Redirect"))
	}

	updated, err := user.GetByName("test")
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if updated.PasswordSalt == oldSalt {
		t.Fatal("password salt unchanged after recovery reset")
	}

	loginResp := postRecoverForm(t, client, baseURL+"/api/login", map[string]string{
		"username": "test",
		"password": newPassword,
	})
	loginResp.Body.Close()
	storeHTTPCookies(jar, baseParsed, loginResp.Cookies())
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login with new password: %d", loginResp.StatusCode)
	}
	var hasJWT bool
	for _, c := range jar.Cookies(baseParsed) {
		if c.Name == "auth_token" && c.Value != "" {
			hasJWT = true
		}
	}
	if !hasJWT {
		t.Fatal("expected auth_token after login with new password")
	}

	t.Cleanup(func() {
		hash, salt, err := password.HashPassword("test")
		if err != nil {
			t.Logf("restore password hash: %v", err)
			return
		}
		if err := user.UpdatePassword(updated.ID, hash, salt, "argon2id"); err != nil {
			t.Logf("restore password: %v", err)
		}
	})
}

func TestJWTSaltRevokeAfterPasswordChange(t *testing.T) {
	loginClient := getTestClient()
	baseParsed, _ := url.Parse(baseURL)

	resp, _ := postFormRequest(t, baseURL+"/api/login", map[string]interface{}{
		"username": "test",
		"password": "test",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d", resp.StatusCode)
	}

	var authToken string
	for _, c := range loginClient.Jar.Cookies(baseParsed) {
		if c.Name == "auth_token" {
			authToken = c.Value
		}
	}
	if authToken == "" {
		t.Fatal("no auth token after login")
	}

	u, err := user.GetByName("test")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	hash, salt, err := password.HashPassword("test-changed-temp")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := user.UpdatePassword(u.ID, hash, salt, "argon2id"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}
	t.Cleanup(func() {
		h, s, err := password.HashPassword("test")
		if err != nil {
			return
		}
		_ = user.UpdatePassword(u.ID, h, s, "argon2id")
	})

	settingsReq, err := http.NewRequest("GET", baseURL+"/auth/user/settings", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	settingsResp, err := loginClient.Do(settingsReq)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	defer settingsResp.Body.Close()
	if settingsResp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(settingsResp.Body)
		if strings.Contains(string(body), "Change Password") {
			t.Fatal("old JWT still authorized after password salt change")
		}
	}
}

func TestRecoverLoginLink(t *testing.T) {
	resp, err := http.Get(baseURL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `href="/recover"`) {
		t.Fatal("login page missing Recover account link")
	}
}

func postRecoverForm(t *testing.T, client *http.Client, requestURL string,
	fields map[string]string) *http.Response {
	t.Helper()
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}

	var csrfToken string
	for _, cookie := range client.Jar.Cookies(baseURLParsed) {
		if cookie.Name == "_csrf" {
			csrfToken = cookie.Value
			break
		}
	}
	if csrfToken == "" {
		getResp, err := client.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("get CSRF: %v", err)
		}
		storeHTTPCookies(client.Jar, baseURLParsed, getResp.Cookies())
		getResp.Body.Close()
		for _, cookie := range client.Jar.Cookies(baseURLParsed) {
			if cookie.Name == "_csrf" {
				csrfToken = cookie.Value
				break
			}
		}
	}
	if csrfToken == "" {
		t.Fatal("no CSRF token")
	}

	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	req, err := http.NewRequest("POST", requestURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Csrf-Token", csrfToken)
	req.Header.Set("HX-Request", "true")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	return resp
}

// storeHTTPCookies copies response cookies into the jar with Secure cleared so
// they work over HTTP (CSRF middleware may bake CookieSecure at package init).
func storeHTTPCookies(jar http.CookieJar, u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		jar.SetCookies(u, []*http.Cookie{{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			HttpOnly: cookie.HttpOnly,
			SameSite: cookie.SameSite,
			Secure:   false,
		}})
	}
}
