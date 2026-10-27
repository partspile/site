package turnstile

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

var (
	siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	httpClient    = &http.Client{Timeout: 10 * time.Second}
)

// Required reports whether Turnstile must be enforced.
func Required() bool {
	return !config.AllowTestRegistration
}

// Init validates Turnstile keys when Required.
func Init() error {
	if !Required() {
		logger.Warn("Turnstile validation skipped (ALLOW_TEST_REGISTRATION)",
			"component", "turnstile")
		return nil
	}
	if config.TurnstileSiteKey == "" {
		return fmt.Errorf("TURNSTILE_SITE_KEY is required")
	}
	if config.TurnstileSecretKey == "" {
		return fmt.Errorf("TURNSTILE_SECRET_KEY is required")
	}
	logger.Info("Turnstile configured", "component", "turnstile")
	return nil
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify checks a Turnstile response token with Cloudflare.
func Verify(token, remoteIP string) error {
	if !Required() {
		return nil
	}
	if token == "" {
		return fmt.Errorf("missing turnstile token")
	}

	form := url.Values{}
	form.Set("secret", config.TurnstileSecretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	resp, err := httpClient.PostForm(siteverifyURL, form)
	if err != nil {
		logger.Error("Turnstile siteverify request failed",
			"error", err, "component", "turnstile")
		return fmt.Errorf("turnstile verification unavailable")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("turnstile verification unavailable")
	}

	var result siteverifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("turnstile verification unavailable")
	}
	if !result.Success {
		logger.Warn("Turnstile verification failed",
			"component", "turnstile",
			"errorCodes", strings.Join(result.ErrorCodes, ","))
		return fmt.Errorf("turnstile verification failed")
	}
	return nil
}
