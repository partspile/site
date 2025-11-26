package sms

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/verification"
	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/client"
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

var twilioClient = twilio.NewRestClientWithParams(twilio.ClientParams{
	Username: config.TwilioAccountSID,
	Password: config.TwilioAuthToken,
})

var tracker sync.Map

// smsAvailable tracks whether SMS functionality is available
// SMS is disabled if TWILIO_WEBHOOK_URL is localhost (webhooks won't work)
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
				logger.Info("Message delivered",
					"component", "SMS", "messageSid", messageSid,
					"phoneNumber", track.PhoneNumber)
			case SMSStatusFailed, SMSStatusUndelivered:
				logger.Warn("Message failed",
					"component", "SMS", "messageSid", messageSid,
					"phoneNumber", track.PhoneNumber, "status", status)
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
			logger.Warn("Message timed out",
				"component", "SMS", "messageSid", messageSid,
				"phoneNumber", track.PhoneNumber)
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
		return fmt.Errorf("SMS service not available - TWILIO_WEBHOOK_URL must be a public URL (not localhost)")
	}

	// Respect user opt-out - fail silently
	if IsOptedOut(phoneNumber) {
		logger.Debug("Message blocked: user opted out",
			"component", "SMS", "phoneNumber", phoneNumber)
		return nil
	}
	params := &Api.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(config.TwilioFromNumber)
	params.SetBody(message)
	params.SetStatusCallback(fmt.Sprintf("%s/api/sms/webhook", config.TwilioWebhookURL))

	result, err := twilioClient.Api.CreateMessage(params)
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

	logger.Info("Message sent",
		"component", "SMS", "messageSid", messageSid,
		"phoneNumber", phoneNumber)

	return nil
}

// HandleStopResponse processes when a user replies STOP to an SMS
func HandleStopResponse(phoneNumber string) error {
	logger.Info("STOP response received",
		"component", "SMS", "phoneNumber", phoneNumber)

	// Find user by phone and set opt-out flag
	u, err := user.GetUserByPhone(phoneNumber)
	if err == nil {
		// User exists, set opt-out flag
		if err := user.SetSMSOptOut(u.ID, true); err != nil {
			logger.Error("Failed to set opt-out for user",
				"error", err, "component", "SMS", "userID", u.ID)
		}
	}

	// Invalidate any pending verification codes for this phone
	err = verification.InvalidateVerificationCodes(phoneNumber)
	if err != nil {
		logger.Error("Failed to invalidate verification codes",
			"error", err, "component", "SMS", "phoneNumber", phoneNumber)
		return err
	}

	return nil
}

// HandleUnstopResponse processes when a user replies UNSTOP to opt back in
func HandleUnstopResponse(phoneNumber string) error {
	logger.Info("UNSTOP response received",
		"component", "SMS", "phoneNumber", phoneNumber)

	// Find user by phone and clear opt-out flag
	u, err := user.GetUserByPhone(phoneNumber)
	if err == nil {
		// User exists, clear opt-out flag
		if err := user.SetSMSOptOut(u.ID, false); err != nil {
			logger.Error("Failed to clear opt-out for user",
				"error", err, "component", "SMS", "userID", u.ID)
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
	logger.Warn("Delivery failure",
		"component", "SMS", "phoneNumber", phoneNumber,
		"errorMessage", errorMessage)

	// Invalidate verification codes for failed deliveries
	err := verification.InvalidateVerificationCodes(phoneNumber)
	if err != nil {
		logger.Error("Failed to invalidate verification codes",
			"error", err, "component", "SMS", "phoneNumber",
			phoneNumber)
		return err
	}

	return nil
}

// VerifyTwilioSignature verifies that the webhook request is authentic from Twilio
// using the official Twilio SDK's RequestValidator
func VerifyTwilioSignature(c *fiber.Ctx) bool {
	signature := c.Get("X-Twilio-Signature")
	if signature == "" || config.TwilioAuthToken == "" {
		return false
	}

	// Construct the full URL that Twilio called
	protocol := c.Get("X-Forwarded-Proto")
	if protocol == "" {
		protocol = c.Protocol()
	}
	if protocol == "" {
		protocol = "https"
	}

	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Hostname()
	}

	webhookURL := fmt.Sprintf("%s://%s%s", protocol, host, c.OriginalURL())

	// Extract all form parameters (POST body and query string)
	params := make(map[string]string)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if _, exists := params[keyStr]; !exists {
			params[keyStr] = string(value)
		}
	})

	// Validate using Twilio SDK
	validator := client.NewRequestValidator(config.TwilioAuthToken)
	return validator.Validate(webhookURL, params, signature)
}

// ParseWebhook parses webhook data from Fiber context
func ParseWebhook(c *fiber.Ctx) (*SMSWebhookData, error) {
	var webhookData SMSWebhookData
	if err := c.BodyParser(&webhookData); err != nil {
		logger.Error("Failed to parse webhook data",
			"error", err, "component", "SMS")
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

// InitSMS checks SMS configuration and availability, disabling SMS if TWILIO_WEBHOOK_URL is localhost
// This should be called at application startup
// Returns nil if SMS is ready, or an error if there's a configuration problem
func InitSMS() error {
	smsAvailable = false

	// Check if Twilio credentials are configured
	if config.TwilioAccountSID == "" || config.TwilioAuthToken == "" || config.TwilioFromNumber == "" {
		logger.Info("SMS disabled - Twilio credentials not configured",
			"component", "SMS")
		return nil // Not an error, just disabled
	}

	// Check if TWILIO_WEBHOOK_URL is localhost (webhooks won't work)
	if config.TwilioWebhookURL == "" {
		logger.Info("SMS disabled - TWILIO_WEBHOOK_URL is empty (webhooks required)",
			"component", "SMS")
		return nil // Not an error, just disabled
	}

	parsedURL, err := url.Parse(config.TwilioWebhookURL)
	if err != nil {
		logger.Warn("SMS disabled - Failed to parse TWILIO_WEBHOOK_URL",
			"error", err, "component", "SMS")
		return fmt.Errorf("invalid TWILIO_WEBHOOK_URL: %w", err) // This is an actual error
	}

	hostname := parsedURL.Hostname()
	if isLocalhost(hostname) {
		logger.Info("SMS disabled - TWILIO_WEBHOOK_URL is localhost (webhooks won't work). Use ngrok or set TWILIO_WEBHOOK_URL to a public URL.",
			"component", "SMS")
		return nil // Not an error, just disabled
	}

	// SMS is available
	smsAvailable = true
	logger.Info("SMS enabled",
		"component", "SMS", "twilioWebhookURL", config.TwilioWebhookURL)

	// Update Twilio phone number webhook URL on startup
	if err := UpdatePhoneNumberWebhook(); err != nil {
		logger.Warn("Failed to update phone number webhook",
			"error", err, "component", "SMS")
		logger.Info("You may need to manually configure the webhook in Twilio Console",
			"component", "SMS")
	}

	return nil
}

// IsSMSAvailable returns whether SMS functionality is available
// This is primarily used internally by the notification service
func IsSMSAvailable() bool {
	return smsAvailable
}

// UpdatePhoneNumberWebhook updates the Twilio phone number's incoming message webhook URL
// This should be called at startup to automatically configure the webhook based on TWILIO_WEBHOOK_URL
// Note: This is only called when SMS is available (checked by InitSMS)
func UpdatePhoneNumberWebhook() error {
	// Defensive check - SMS should already be available, but verify
	if !smsAvailable {
		return fmt.Errorf("SMS not available - cannot update webhook")
	}

	if config.TwilioAccountSID == "" || config.TwilioAuthToken == "" || config.TwilioFromNumber == "" {
		return fmt.Errorf("Twilio credentials not configured")
	}

	webhookURL := fmt.Sprintf("%s/api/sms/webhook", config.TwilioWebhookURL)

	// Get the phone number SID first
	params := &Api.ListIncomingPhoneNumberParams{}
	params.SetPhoneNumber(config.TwilioFromNumber)

	resp, err := twilioClient.Api.ListIncomingPhoneNumber(params)
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

	_, err = twilioClient.Api.UpdateIncomingPhoneNumber(phoneNumberSid, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update phone number webhook: %w", err)
	}

	logger.Info("Successfully updated phone number webhook",
		"component", "SMS", "phoneNumber", config.TwilioFromNumber,
		"webhookURL", webhookURL)
	return nil
}
