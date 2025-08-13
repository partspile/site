package sms

import (
	"fmt"
	"log"
	"strings"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/verification"
	"github.com/twilio/twilio-go"
	Api "github.com/twilio/twilio-go/rest/api/v2010"
)

type SMSService struct {
	client *twilio.RestClient
	from   string
}

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

// SMSStatusTracker tracks the status of SMS messages
type SMSStatusTracker struct {
	statuses map[string]SMSStatus
}

var globalStatusTracker = &SMSStatusTracker{
	statuses: make(map[string]SMSStatus),
}

// GetMessageStatus returns the current status of a message
func (s *SMSStatusTracker) GetMessageStatus(messageSid string) SMSStatus {
	if status, exists := s.statuses[messageSid]; exists {
		return status
	}
	return ""
}

// SetMessageStatus sets the status of a message
func (s *SMSStatusTracker) SetMessageStatus(messageSid string, status SMSStatus) {
	s.statuses[messageSid] = status
}

// GetGlobalStatusTracker returns the global SMS status tracker
func GetGlobalStatusTracker() *SMSStatusTracker {
	return globalStatusTracker
}

// NewSMSService creates a new SMS service instance
func NewSMSService() (*SMSService, error) {
	accountSid := config.TwilioAccountSID
	authToken := config.TwilioAuthToken
	fromNumber := config.TwilioFromNumber

	if accountSid == "" || authToken == "" || fromNumber == "" {
		return nil, fmt.Errorf("missing Twilio configuration")
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	return &SMSService{
		client: client,
		from:   fromNumber,
	}, nil
}

// SendVerificationCode sends a verification code via SMS
func (s *SMSService) SendVerificationCode(phoneNumber, code string) (string, error) {
	message := fmt.Sprintf("Your Parts Pile verification code is: %s. "+
		"This code expires in 10 minutes. Reply STOP to unsubscribe.", code)

	params := &Api.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(s.from)
	params.SetBody(message)
	params.SetStatusCallback(fmt.Sprintf("%s/api/sms/webhook", config.BaseURL))

	result, err := s.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("[SMS] Failed to send verification code to %s: %v", phoneNumber, err)
		return "", fmt.Errorf("failed to send SMS: %w", err)
	}

	log.Printf("[SMS] Verification code sent to %s", phoneNumber)
	return *result.Sid, nil
}

// SendGeneralMessage sends a general SMS message
func (s *SMSService) SendGeneralMessage(phoneNumber, message string) (string, error) {
	params := &Api.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(s.from)
	params.SetBody(message)
	params.SetStatusCallback(fmt.Sprintf("%s/api/sms/webhook", config.BaseURL))

	result, err := s.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("[SMS] Failed to send general message to %s: %v", phoneNumber, err)
		return "", fmt.Errorf("failed to send SMS: %w", err)
	}

	log.Printf("[SMS] General message sent to %s", phoneNumber)
	return *result.Sid, nil
}

// HandleWebhook processes Twilio webhook callbacks for SMS status updates
func (s *SMSService) HandleWebhook(data SMSWebhookData) error {
	log.Printf("[SMS] Webhook received: MessageSid=%s, Status=%s, To=%s, From=%s",
		data.MessageSid, data.MessageStatus, data.To, data.From)

	// Update the global status tracker
	status := SMSStatus(data.MessageStatus)
	GetGlobalStatusTracker().SetMessageStatus(data.MessageSid, status)

	// Handle STOP responses
	if strings.ToUpper(strings.TrimSpace(data.Body)) == "STOP" {
		return s.handleStopResponse(data.To)
	}

	// Handle delivery status updates
	switch status {
	case SMSStatusDelivered:
		log.Printf("[SMS] Message delivered successfully to %s", data.To)
	case SMSStatusFailed, SMSStatusUndelivered:
		log.Printf("[SMS] Message failed to deliver to %s: %s", data.To, data.ErrorMessage)
		return s.handleDeliveryFailure(data.To, data.ErrorMessage)
	case SMSStatusSent:
		log.Printf("[SMS] Message sent to %s", data.To)
	default:
		log.Printf("[SMS] Unknown message status: %s for %s", data.MessageStatus, data.To)
	}

	return nil
}

// handleStopResponse processes when a user replies STOP to an SMS
func (s *SMSService) handleStopResponse(phoneNumber string) error {
	log.Printf("[SMS] STOP response received from %s", phoneNumber)

	// Invalidate any pending verification codes for this phone
	err := verification.InvalidateVerificationCodes(phoneNumber)
	if err != nil {
		log.Printf("[SMS] Failed to invalidate verification codes for %s: %v", phoneNumber, err)
		return err
	}

	// Note: In a production system, you might also want to:
	// 1. Add the phone number to a "do not contact" list
	// 2. Cancel any pending registration processes
	// 3. Send a confirmation message that they've been unsubscribed

	return nil
}

// handleDeliveryFailure processes when an SMS fails to deliver
func (s *SMSService) handleDeliveryFailure(phoneNumber, errorMessage string) error {
	log.Printf("[SMS] Delivery failure for %s: %s", phoneNumber, errorMessage)

	// Invalidate verification codes for failed deliveries
	err := verification.InvalidateVerificationCodes(phoneNumber)
	if err != nil {
		log.Printf("[SMS] Failed to invalidate verification codes for %s: %v", phoneNumber, err)
		return err
	}

	return nil
}

// MockSMSService is used for testing without sending actual SMS
type MockSMSService struct {
	sentCodes map[string]string
}

func NewMockSMSService() *MockSMSService {
	return &MockSMSService{
		sentCodes: make(map[string]string),
	}
}

func (m *MockSMSService) SendVerificationCode(phoneNumber, code string) error {
	m.sentCodes[phoneNumber] = code
	log.Printf("[MOCK SMS] Verification code %s sent to %s", code, phoneNumber)
	return nil
}

func (m *MockSMSService) GetSentCode(phoneNumber string) (string, bool) {
	code, exists := m.sentCodes[phoneNumber]
	return code, exists
}

// Mock webhook handling for testing
func (m *MockSMSService) HandleWebhook(data SMSWebhookData) error {
	log.Printf("[MOCK SMS] Webhook received: %+v", data)
	return nil
}
