package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/parts-pile/site/config"
)

// EmailService handles sending emails via Twilio SendGrid
type EmailService struct {
	apiKey string
	from   string
}

// Email represents an email message
type Email struct {
	To      string    `json:"to"`
	From    string    `json:"from"`
	Subject string    `json:"subject"`
	Content []Content `json:"content"`
}

// Content represents the content of an email
type Content struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewEmailService creates a new email service instance
func NewEmailService() (*EmailService, error) {
	apiKey := config.TwilioSendGridAPIKey
	fromEmail := config.TwilioFromEmail

	if apiKey == "" || fromEmail == "" {
		return nil, fmt.Errorf("missing SendGrid configuration")
	}

	return &EmailService{
		apiKey: apiKey,
		from:   fromEmail,
	}, nil
}

// SendEmail sends an email via SendGrid
func (s *EmailService) SendEmail(to, subject, htmlBody string) error {
	email := Email{
		To:      to,
		From:    s.from,
		Subject: subject,
		Content: []Content{
			{
				Type:  "text/html",
				Value: htmlBody,
			},
		},
	}

	jsonData, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: %d", resp.StatusCode)
	}

	log.Printf("[EMAIL] Email sent successfully to %s", to)
	return nil
}

// SendNotificationEmail sends a notification email about a new message
func (s *EmailService) SendNotificationEmail(to, senderName, adTitle, messageContent string, conversationID int) error {
	subject := fmt.Sprintf("New message from %s about '%s'", senderName, adTitle)

	htmlBody := fmt.Sprintf(`
<html>
<head>
    <title>New Message</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .message-box { background-color: #f1f3f4; padding: 15px; border-radius: 6px; margin: 20px 0; }
        .button { display: inline-block; background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2 style="margin: 0; color: #495057;">New Message on Parts Pile</h2>
        </div>
        
        <p><strong>From:</strong> %s</p>
        <p><strong>About:</strong> %s</p>
        
        <div class="message-box">
            <p><strong>Message:</strong></p>
            <p style="margin: 0; font-style: italic;">%s</p>
        </div>
        
        <a href="%s/messages/%d" class="button">View Conversation</a>
        
        <div class="footer">
            <p>This is an automated notification from Parts Pile.</p>
            <p>You can manage your notification preferences in your account settings.</p>
        </div>
    </div>
</body>
</html>`, senderName, adTitle, messageContent, config.BaseURL, conversationID)

	return s.SendEmail(to, subject, htmlBody)
}

// MockEmailService is used for testing without sending actual emails
type MockEmailService struct {
	sentEmails []struct {
		to      string
		subject string
		body    string
	}
}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		sentEmails: make([]struct {
			to      string
			subject string
			body    string
		}, 0),
	}
}

func (m *MockEmailService) SendEmail(to, subject, htmlBody string) error {
	m.sentEmails = append(m.sentEmails, struct {
		to      string
		subject string
		body    string
	}{to, subject, htmlBody})
	log.Printf("[MOCK EMAIL] Email sent to %s: %s", to, subject)
	return nil
}

func (m *MockEmailService) SendNotificationEmail(to, senderName, adTitle, messageContent string, conversationID int) error {
	return m.SendEmail(to, fmt.Sprintf("New message from %s about '%s'", senderName, adTitle), messageContent)
}

func (m *MockEmailService) GetSentEmails() []struct {
	to      string
	subject string
	body    string
} {
	return m.sentEmails
}
