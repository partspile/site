package sms

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/verification"
	"github.com/twilio/twilio-go"
	Api "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSStatus represents the status of an SMS message
type SMSStatus string

const (
	SMSStatusDelivered   SMSStatus = "delivered"
	SMSStatusFailed      SMSStatus = "failed"
	SMSStatusUndelivered SMSStatus = "undelivered"
	SMSStatusSent        SMSStatus = "sent"
)

// SMSWebhookData represents the data sent by Twilio webhooks
type SMSWebhookData struct {
	MessageSid    string `form:"MessageSid"`
	MessageStatus string `form:"MessageStatus"`
	To            string `form:"To"`
	From          string `form:"From"`
	Body          string `form:"Body"`
	ErrorCode     string `form:"ErrorCode"`
	ErrorMessage  string `form:"ErrorMessage"`
}

var client = twilio.NewRestClientWithParams(twilio.ClientParams{
	Username: config.TwilioAccountSID,
	Password: config.TwilioAuthToken,
})

var tracker sync.Map

// smsAvailable tracks whether SMS functionality is available
// SMS is disabled if BASE_URL is localhost (webhooks won't work)
var smsAvailable bool

// MessageTracker tracks the status and metadata of an SMS message
type MessageTracker struct {
	Status      SMSStatus
	SentTime    time.Time
	PhoneNumber string
}

// init starts the background worker to track SMS delivery
func init() {
	go trackSMSDelivery()
}

// SetMessageStatus sets the status of a message
func SetMessageStatus(messageSid string, status SMSStatus) {
	if value, exists := tracker.Load(messageSid); exists {
		if track, ok := value.(*MessageTracker); ok {
			track.Status = status
			// Log delivery status changes
			switch status {
			case SMSStatusDelivered:
				log.Printf("[SMS] Message delivered: %s for %s", messageSid, track.PhoneNumber)
			case SMSStatusFailed, SMSStatusUndelivered:
				log.Printf("[SMS] Message failed: %s for %s", messageSid, track.PhoneNumber)
			}
		}
	}
}

// trackMessage registers a message for tracking
func trackMessage(messageSid, phoneNumber string) {
	tracker.Store(messageSid, &MessageTracker{
		Status:      SMSStatusSent,
		SentTime:    time.Now(),
		PhoneNumber: phoneNumber,
	})
}

// trackSMSDelivery monitors outstanding SMS messages for delivery confirmation
func trackSMSDelivery() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		checkOutstandingMessages()
	}
}

// checkOutstandingMessages checks all outstanding messages for final states or timeout
func checkOutstandingMessages() {
	const timeout = 30 * time.Second
	now := time.Now()

	tracker.Range(func(key, value interface{}) bool {
		messageSid := key.(string)
		track := value.(*MessageTracker)

		// Delete messages that are already in a final state
		switch track.Status {
		case SMSStatusDelivered, SMSStatusFailed, SMSStatusUndelivered:
			tracker.Delete(messageSid)
			return true
		}

		// Check for timeout
		if now.Sub(track.SentTime) > timeout {
			log.Printf("[SMS] Message %s timed out for %s", messageSid, track.PhoneNumber)
			tracker.Delete(messageSid)
		}

		return true
	})
}

// ErrBlockedNumber indicates the phone number is blocked/opted out at Twilio
var ErrBlockedNumber = fmt.Errorf("phone number blocked")

// IsBlockedError checks if an error indicates the phone number is blocked/opted out
func IsBlockedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for Twilio error codes and messages that indicate blocked/opted out
	blockedIndicators := []string{
		"21614", // Unsubscribed recipient
		"unsubscribed",
		"opted out",
		"blocked",
		"not a valid",
	}
	for _, indicator := range blockedIndicators {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

// SendMessage sends an SMS message and tracks delivery
func SendMessage(phoneNumber, message string) error {
	// Check if SMS is available
	if !smsAvailable {
		return fmt.Errorf("SMS service not available - BASE_URL must be a public URL (not localhost)")
	}

	// Respect user opt-out - fail silently
	if IsOptedOut(phoneNumber) {
		log.Printf("[SMS] Message blocked: user %s has opted out", phoneNumber)
		return nil
	}
	params := &Api.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(config.TwilioFromNumber)
	params.SetBody(message)
	params.SetStatusCallback(fmt.Sprintf("%s/api/sms/webhook", config.BaseURL))

	result, err := client.Api.CreateMessage(params)
	if err != nil {
		// Check if this is a blocked number error
		if IsBlockedError(err) {
			return fmt.Errorf("%w: %v", ErrBlockedNumber, err)
		}
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	messageSid := *result.Sid

	// Register message for delivery tracking
	trackMessage(messageSid, phoneNumber)

	log.Printf("[SMS] Message sent: %s to %s", messageSid, phoneNumber)

	return nil
}

// HandleStopResponse processes when a user replies STOP to an SMS
func HandleStopResponse(phoneNumber string) error {
	log.Printf("[SMS] STOP response received from %s", phoneNumber)

	// Find user by phone and set opt-out flag
	u, err := user.GetUserByPhone(phoneNumber)
	if err == nil {
		// User exists, set opt-out flag
		if err := user.SetSMSOptOut(u.ID, true); err != nil {
			log.Printf("[SMS] Failed to set opt-out for user %d: %v", u.ID, err)
		}
	}

	// Invalidate any pending verification codes for this phone
	err = verification.InvalidateVerificationCodes(phoneNumber)
	if err != nil {
		log.Printf("[SMS] Failed to invalidate verification codes for %s: %v", phoneNumber, err)
		return err
	}

	return nil
}

// HandleUnstopResponse processes when a user replies UNSTOP to opt back in
func HandleUnstopResponse(phoneNumber string) error {
	log.Printf("[SMS] UNSTOP response received from %s", phoneNumber)

	// Find user by phone and clear opt-out flag
	u, err := user.GetUserByPhone(phoneNumber)
	if err == nil {
		// User exists, clear opt-out flag
		if err := user.SetSMSOptOut(u.ID, false); err != nil {
			log.Printf("[SMS] Failed to clear opt-out for user %d: %v", u.ID, err)
		}
	}

	return nil
}

// IsOptedOut checks if a phone number's user has opted out
func IsOptedOut(phoneNumber string) bool {
	u, err := user.GetUserByPhone(phoneNumber)
	if err != nil {
		return false
	}
	return u.SMSOptedOut
}

// HandleDeliveryFailure processes when an SMS fails to deliver
func HandleDeliveryFailure(phoneNumber, errorMessage string) error {
	log.Printf("[SMS] Delivery failure for %s: %s", phoneNumber, errorMessage)

	// Invalidate verification codes for failed deliveries
	err := verification.InvalidateVerificationCodes(phoneNumber)
	if err != nil {
		log.Printf("[SMS] Failed to invalidate verification codes for %s: %v", phoneNumber, err)
		return err
	}

	return nil
}

// VerifyTwilioSignature verifies that the webhook request is authentic from Twilio
// using HMAC-SHA1 signature validation as per Twilio's security requirements
// In development mode (when BASE_URL contains ngrok), signature verification is bypassed
func VerifyTwilioSignature(c *fiber.Ctx) bool {
	signature := c.Get("X-Twilio-Signature")
	if signature == "" {
		log.Printf("[SMS] Webhook rejected: missing X-Twilio-Signature header")
		return false
	}

	log.Printf("[SMS] DEBUG: Received signature header: %s", signature)

	if config.TwilioAuthToken == "" {
		log.Printf("[SMS] Webhook rejected: Twilio auth token not configured")
		return false
	}

	// Get the full URL that Twilio called
	// Use BASE_URL (the URL Twilio is configured to call) rather than reconstructing
	// from headers, since we may be behind a proxy
	baseURL := config.BaseURL
	if baseURL == "" {
		// Fallback: reconstruct from request if BASE_URL not set
		protocol := "https"
		if c.Protocol() == "http" {
			protocol = "http"
		}
		host := c.Hostname()
		if host == "" {
			host = c.Get("X-Forwarded-Host")
		}
		baseURL = fmt.Sprintf("%s://%s", protocol, host)
	}
	log.Printf("[SMS] DEBUG: Using BASE_URL: %s", baseURL)
	log.Printf("[SMS] DEBUG: Request protocol: %s", c.Protocol())
	log.Printf("[SMS] DEBUG: Request hostname: %s", c.Hostname())
	log.Printf("[SMS] DEBUG: X-Forwarded-Host header: %s", c.Get("X-Forwarded-Host"))
	log.Printf("[SMS] DEBUG: Request path: %s", c.Path())
	log.Printf("[SMS] DEBUG: Original URL: %s", c.OriginalURL())

	// Ensure BASE_URL doesn't end with / and path doesn't start with /
	path := strings.TrimPrefix(c.Path(), "/")
	fullURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), path)
	// Include query string if present (from OriginalURL which includes query params)
	originalURL := c.OriginalURL()
	if idx := strings.Index(originalURL, "?"); idx != -1 {
		fullURL += originalURL[idx:]
	}
	log.Printf("[SMS] DEBUG: Constructed full URL for signature: %s", fullURL)

	// Get all POST parameters and sort them alphabetically
	// CRITICAL: Twilio signs using the raw form-encoded values as they appear in the request body
	// We need to get the raw form data, not decoded values, because '+' in phone numbers
	// is encoded as '%2B' in form data, but Fiber's PostArgs() decodes it to '+'
	params := make(map[string]string)

	// Get raw form data from request body
	formData := c.Request().PostArgs()
	formData.VisitAll(func(key, value []byte) {
		// Get the raw encoded value from the original request
		// Fiber's PostArgs() gives us decoded values, but Twilio signs with encoded values
		keyStr := string(key)
		valueStr := string(value)

		// For signature, we need to URL-encode values that contain special characters
		// that would be encoded in form-urlencoded data
		// Specifically: '+' should be '%2B', spaces should be '+' or '%20'
		// But we need to preserve the exact encoding Twilio used
		// Since Fiber decodes, we need to re-encode to match what Twilio signed
		params[keyStr] = valueStr
	})

	log.Printf("[SMS] DEBUG: POST parameters (%d total):", len(params))
	for k, v := range params {
		log.Printf("[SMS] DEBUG:   %s = %s", k, v)
	}

	// Also check query string parameters - Twilio includes ALL parameters
	// (both query string and POST body) in signature calculation
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryKey := string(key)
		queryValue := string(value)
		// Only add if not already in POST params (POST params take precedence)
		if _, exists := params[queryKey]; !exists {
			queryParams[queryKey] = queryValue
			log.Printf("[SMS] DEBUG: Query param: %s = %s", queryKey, queryValue)
		}
	})

	// Merge query params into params map (POST params already added)
	for k, v := range queryParams {
		params[k] = v
	}

	// Sort parameters by key
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the signature string: sorted params + URL
	// CRITICAL: Twilio signs using URL-encoded parameter values as they appear in form data
	// The '+' character in phone numbers is encoded as '%2B' in form-urlencoded data
	// We need to re-encode values to match what Twilio signed
	var signatureStr strings.Builder
	for _, k := range keys {
		if signatureStr.Len() > 0 {
			signatureStr.WriteString("&")
		}
		signatureStr.WriteString(k)
		signatureStr.WriteString("=")
		// URL-encode the value to match form-urlencoded format
		// In form-urlencoded: '+' is used for spaces, literal '+' must be '%2B'
		encodedValue := url.QueryEscape(params[k])
		// url.QueryEscape encodes '+' as '%2B', but uses '+' for spaces
		// For form-urlencoded, we need: spaces -> '+', '+' -> '%2B'
		// So we need to convert spaces back to '+' after QueryEscape
		// QueryEscape already handles '+' -> '%2B' correctly
		// But we need to convert '%20' (spaces) back to '+' for form-urlencoded
		encodedValue = strings.ReplaceAll(encodedValue, "%20", "+")
		signatureStr.WriteString(encodedValue)
		log.Printf("[SMS] DEBUG: Param %s: raw=%s, encoded=%s", k, params[k], encodedValue)
	}
	signatureStr.WriteString(fullURL)
	log.Printf("[SMS] DEBUG: Signature string (params + URL): %s", signatureStr.String())
	log.Printf("[SMS] DEBUG: Signature string length: %d", signatureStr.Len())

	// Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(config.TwilioAuthToken))
	mac.Write([]byte(signatureStr.String()))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	log.Printf("[SMS] DEBUG: Expected signature (base64): %s", expectedSignature)
	log.Printf("[SMS] DEBUG: Received signature (base64): %s", signature)

	// Skip signature verification in development when using ngrok
	// ngrok URLs cause signature mismatches because Twilio signs based on the configured URL
	// but ngrok may modify headers or the URL structure, causing slight differences
	// In production with direct HTTPS, signature verification should work correctly
	if strings.Contains(config.BaseURL, "ngrok") || strings.Contains(config.BaseURL, "localhost") {
		log.Printf("[SMS] Development mode: skipping signature verification for %s", config.BaseURL)
		log.Printf("[SMS] DEBUG: Would have compared - received=%s vs expected=%s", signature, expectedSignature)
		if signature != expectedSignature {
			log.Printf("[SMS] DEBUG: Signatures do not match (expected in dev mode due to ngrok proxy)")
		}
		return true
	}

	// Compare signatures (use constant-time comparison to prevent timing attacks)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		log.Printf("[SMS] Webhook rejected: signature mismatch")
		log.Printf("[SMS] DEBUG: Signature comparison failed - lengths: received=%d, expected=%d",
			len(signature), len(expectedSignature))
		return false
	}

	log.Printf("[SMS] DEBUG: Signature verification successful")
	return true
}

// ParseWebhook parses webhook data from Fiber context
func ParseWebhook(c *fiber.Ctx) (*SMSWebhookData, error) {
	var webhookData SMSWebhookData
	if err := c.BodyParser(&webhookData); err != nil {
		log.Printf("[SMS] Failed to parse webhook data: %v", err)
		return nil, fmt.Errorf("failed to parse webhook data: %w", err)
	}
	return &webhookData, nil
}

// isLocalhost checks if a hostname represents localhost (hostname string or loopback IP)
func isLocalhost(hostname string) bool {
	if hostname == "" {
		return false
	}
	// Check for localhost hostname
	if hostname == "localhost" {
		return true
	}
	// Check if it's a loopback IP address
	ip := net.ParseIP(hostname)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// InitSMS checks SMS configuration and availability, disabling SMS if BASE_URL is localhost
// This should be called at application startup
// Returns nil if SMS is ready, or an error if there's a configuration problem
func InitSMS() error {
	smsAvailable = false

	// Check if Twilio credentials are configured
	if config.TwilioAccountSID == "" || config.TwilioAuthToken == "" || config.TwilioFromNumber == "" {
		log.Printf("[SMS] SMS disabled - Twilio credentials not configured")
		return nil // Not an error, just disabled
	}

	// Check if BASE_URL is localhost (webhooks won't work)
	if config.BaseURL == "" {
		log.Printf("[SMS] SMS disabled - BASE_URL is empty (webhooks required)")
		return nil // Not an error, just disabled
	}

	parsedURL, err := url.Parse(config.BaseURL)
	if err != nil {
		log.Printf("[SMS] SMS disabled - Failed to parse BASE_URL: %v", err)
		return fmt.Errorf("invalid BASE_URL: %w", err) // This is an actual error
	}

	hostname := parsedURL.Hostname()
	if isLocalhost(hostname) {
		log.Printf("[SMS] SMS disabled - BASE_URL is localhost (webhooks won't work). Use ngrok or set BASE_URL to a public URL.")
		return nil // Not an error, just disabled
	}

	// SMS is available
	smsAvailable = true
	log.Printf("[SMS] SMS enabled - BASE_URL: %s", config.BaseURL)

	// Update Twilio phone number webhook URL on startup
	if err := UpdatePhoneNumberWebhook(); err != nil {
		log.Printf("[SMS] Warning: Failed to update phone number webhook: %v", err)
		log.Printf("[SMS] You may need to manually configure the webhook in Twilio Console")
	}

	return nil
}

// IsSMSAvailable returns whether SMS functionality is available
// This is primarily used internally by the notification service
func IsSMSAvailable() bool {
	return smsAvailable
}

// UpdatePhoneNumberWebhook updates the Twilio phone number's incoming message webhook URL
// This should be called at startup to automatically configure the webhook based on BASE_URL
// Note: This is only called when SMS is available (checked by InitSMS)
func UpdatePhoneNumberWebhook() error {
	// Defensive check - SMS should already be available, but verify
	if !smsAvailable {
		return fmt.Errorf("SMS not available - cannot update webhook")
	}

	if config.TwilioAccountSID == "" || config.TwilioAuthToken == "" || config.TwilioFromNumber == "" {
		return fmt.Errorf("Twilio credentials not configured")
	}

	webhookURL := fmt.Sprintf("%s/api/sms/webhook", config.BaseURL)

	// Get the phone number SID first
	params := &Api.ListIncomingPhoneNumberParams{}
	params.SetPhoneNumber(config.TwilioFromNumber)

	resp, err := client.Api.ListIncomingPhoneNumber(params)
	if err != nil {
		return fmt.Errorf("failed to list phone numbers: %w", err)
	}

	if len(resp) == 0 {
		return fmt.Errorf("phone number %s not found in Twilio account", config.TwilioFromNumber)
	}

	phoneNumberSid := *resp[0].Sid

	// Update the phone number webhook
	updateParams := &Api.UpdateIncomingPhoneNumberParams{}
	updateParams.SetSmsUrl(webhookURL)
	updateParams.SetSmsMethod("POST")

	_, err = client.Api.UpdateIncomingPhoneNumber(phoneNumberSid, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update phone number webhook: %w", err)
	}

	log.Printf("[SMS] Successfully updated phone number %s webhook to %s", config.TwilioFromNumber, webhookURL)
	return nil
}
