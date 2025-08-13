# Messaging System Setup Guide

## Overview
The Parts Pile messaging system uses Twilio for both SMS and email notifications. This guide explains how to configure the system.

## Environment Variables

### Required for SMS Notifications
```bash
# Twilio Account Configuration
TWILIO_ACCOUNT_SID=your_twilio_account_sid
TWILIO_AUTH_TOKEN=your_twilio_auth_token
TWILIO_FROM_NUMBER=your_twilio_phone_number
```

### Required for Email Notifications
```bash
# Twilio SendGrid Configuration
TWILIO_SENDGRID_API_KEY=your_sendgrid_api_key
TWILIO_FROM_EMAIL=your_verified_sender_email
```

### Server Configuration
```bash
# Base URL for notification links
BASE_URL=http://localhost:8000
```

## Setup Steps

### 1. Twilio Account Setup
1. Create a Twilio account at [twilio.com](https://twilio.com)
2. Get your Account SID and Auth Token from the Twilio Console
3. Purchase a phone number for SMS (or use a trial number)

### 2. SendGrid Email Setup
1. In your Twilio Console, go to "Email" section
2. Set up SendGrid integration
3. Get your SendGrid API key
4. Verify your sender email address

### 3. Phone Number Verification
- For production: Use a verified Twilio phone number
- For testing: Use Twilio trial numbers (limited functionality)

### 4. Email Verification
- The `TWILIO_FROM_EMAIL` must be verified with SendGrid
- This prevents email delivery issues

## Testing

### Test SMS
- Use a real phone number (not +15551234567)
- Ensure your Twilio account has credits
- Check Twilio logs for delivery status

### Test Email
- Use a real email address
- Check spam folders
- Monitor SendGrid delivery logs

## Troubleshooting

### Common Issues

1. **"SMS service not available"**
   - Check TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER
   - Ensure Twilio account is active and has credits

2. **"Email service not available"**
   - Check TWILIO_SENDGRID_API_KEY, TWILIO_FROM_EMAIL
   - Verify sender email is verified with SendGrid

3. **"Invalid phone number"**
   - Use E.164 format (+1234567890)
   - Ensure number is valid and not a test number

4. **Email not delivered**
   - Check spam folders
   - Verify sender email domain reputation
   - Check SendGrid delivery logs

### Logs
- SMS errors: Look for `[SMS]` in application logs
- Email errors: Look for `[EMAIL]` in application logs
- Notification errors: Look for `Failed to send notification` messages

## Security Notes
- Never commit API keys to version control
- Use environment variables for all sensitive configuration
- Regularly rotate API keys
- Monitor Twilio usage and costs
