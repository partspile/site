package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/user"
)

func TestLoginHandler(t *testing.T) {
	resp, err := http.Get(baseURL + "/login")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html" && contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	bodyBytes := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	body := string(bodyBytes)
	if len(body) == 0 {
		t.Error("Expected non-empty HTML response")
	}
}

func TestLoginSubmitHandler(t *testing.T) {
	if len(config.DBEncryptionKey) == 0 {
		t.Fatal("DB_ENCRYPTION_KEY environment variable not set. This is required for user decryption.")
	}

	testUser, err := user.GetByName("test")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("Test user not found in database. Database may need to be seeded (Admin TUI → Init with seed)")
		}
		t.Fatalf("Failed to retrieve test user (error: %v). This might be a decryption error if DB_ENCRYPTION_KEY doesn't match the key used to seed the database.", err)
	}

	if testUser.ID == 0 {
		t.Fatal("Test user found but invalid (ID is 0)")
	}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
		expectJWT      bool
	}{
		{"Valid login", "test", "test", 200, true},
		{"Valid login - admin", "admin", "admin", 200, true},
		{"Invalid username", "nonexistent", "test", 200, false},
		{"Invalid password", "test", "wrong", 200, false},
		{"Empty username", "", "test", 200, false},
		{"Empty password", "test", "", 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := getTestClient()
			baseURLParsed, _ := url.Parse(baseURL)

			clearCookie := &http.Cookie{
				Name:     "auth_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteStrictMode,
			}
			client.Jar.SetCookies(baseURLParsed, []*http.Cookie{clearCookie})

			formData := map[string]interface{}{
				"username": tt.username,
				"password": tt.password,
			}

			resp, result := postFormRequest(t, baseURL+"/api/login", formData)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				if len(result) > 0 {
					t.Logf("Response: %+v", result)
				}
			}

			hasJWT := false
			var jwtCookie *http.Cookie
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "auth_token" {
					hasJWT = true
					jwtCookie = cookie
					if cookie.Secure && tt.expectJWT {
						testCookie := &http.Cookie{
							Name:     cookie.Name,
							Value:    cookie.Value,
							Path:     cookie.Path,
							Domain:   cookie.Domain,
							HttpOnly: cookie.HttpOnly,
							SameSite: cookie.SameSite,
							Secure:   false,
						}
						client.Jar.SetCookies(baseURLParsed, []*http.Cookie{testCookie})
					}
					break
				}
			}

			if tt.expectJWT {
				if !hasJWT {
					t.Error("Expected JWT cookie but it was not set")
					t.Logf("Response cookies: %v", resp.Cookies())
				} else if jwtCookie != nil && jwtCookie.Value == "" {
					t.Error("JWT cookie value is empty")
				}
			} else if hasJWT {
				t.Error("Unexpected JWT cookie for failed login")
			}
		})
	}
}
