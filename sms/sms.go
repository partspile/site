package sms

import (
	"fmt"
	"log"

	"github.com/parts-pile/site/config"
	"github.com/twilio/twilio-go"
	Api "github.com/twilio/twilio-go/rest/api/v2010"
)

type SMSService struct {
	client *twilio.RestClient
	from   string
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
func (s *SMSService) SendVerificationCode(phoneNumber, code string) error {
	message := fmt.Sprintf("Your Parts Pile verification code is: %s. "+
		"This code expires in 10 minutes.", code)

	params := &Api.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(s.from)
	params.SetBody(message)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("[SMS] Failed to send verification code to %s: %v", phoneNumber, err)
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	log.Printf("[SMS] Verification code sent to %s", phoneNumber)
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
