package verify

import (
	"errors"
	"fmt"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/verify/v2"
)

const (
	PurposeRegister    = "register"
	PurposeChangePhone = "change_phone"
)

var (
	ErrInvalidCode = errors.New("invalid verification code")
	ErrBlocked     = errors.New("phone number blocked or opted out")
	ErrRejected    = errors.New("verification rejected")
)

var client *twilio.RestClient

// Init validates Verify configuration. Skipped when ALLOW_TEST_REGISTRATION.
func Init() error {
	if config.AllowTestRegistration {
		logger.Warn("Twilio Verify validation skipped (ALLOW_TEST_REGISTRATION)",
			"component", "verify")
		return nil
	}
	if config.TwilioVerifyServiceSID == "" {
		return fmt.Errorf("TWILIO_VERIFY_SERVICE_SID is required")
	}
	if !strings.HasPrefix(config.TwilioVerifyServiceSID, "VA") {
		return fmt.Errorf("TWILIO_VERIFY_SERVICE_SID must start with 'VA'")
	}
	client = twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: config.TwilioAccountSID,
		Password: config.TwilioAuthToken,
	})
	if client == nil {
		return fmt.Errorf("failed to create Twilio Verify client")
	}
	logger.Info("Twilio Verify configured",
		"component", "verify",
		"serviceSID", config.TwilioVerifyServiceSID)
	return nil
}

// StartSMS begins an SMS verification for phoneE64.
func StartSMS(phoneE64, purpose string) error {
	params := &openapi.CreateVerificationParams{}
	params.SetTo(phoneE64)
	params.SetChannel("sms")

	_, err := client.VerifyV2.CreateVerification(
		config.TwilioVerifyServiceSID, params)
	if err != nil {
		logger.Warn("Verify start failed",
			"component", "verify", "phone", phoneE64,
			"purpose", purpose, "error", err)
		return mapTwilioErr(err)
	}
	logger.Info("Verify SMS started",
		"component", "verify", "phoneE64", phoneE64, "purpose", purpose)
	return nil
}

// Check validates an OTP for phoneE64. Returns nil when approved.
func Check(phoneE64, code string) error {
	params := &openapi.CreateVerificationCheckParams{}
	params.SetTo(phoneE64)
	params.SetCode(code)

	resp, err := client.VerifyV2.CreateVerificationCheck(
		config.TwilioVerifyServiceSID, params)
	if err != nil {
		logger.Warn("Verify check failed",
			"component", "verify", "phone", phoneE64, "error", err)
		return mapTwilioErr(err)
	}
	if resp.Status == nil || *resp.Status != "approved" {
		status := ""
		if resp.Status != nil {
			status = *resp.Status
		}
		logger.Warn("Verify check not approved",
			"component", "verify", "phone", phoneE64, "status", status)
		return ErrInvalidCode
	}
	logger.Info("Verify check approved",
		"component", "verify", "phoneE64", phoneE64)
	return nil
}

func mapTwilioErr(err error) error {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "60200"),
		strings.Contains(msg, "invalid parameter"),
		strings.Contains(msg, "was not found"):
		return ErrInvalidCode
	case strings.Contains(msg, "21610"),
		strings.Contains(msg, "blacklisted"),
		strings.Contains(msg, "opted out"),
		strings.Contains(msg, "blocked"):
		return fmt.Errorf("%w: %v", ErrBlocked, err)
	case strings.Contains(msg, "60410"),
		strings.Contains(msg, "geo"),
		strings.Contains(msg, "permission"),
		strings.Contains(msg, "fraud"):
		return fmt.Errorf("%w: %v", ErrRejected, err)
	default:
		return fmt.Errorf("%w: %v", ErrRejected, err)
	}
}
