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
# Base URL for notification links and webhooks
# For local development with ngrok:
BASE_URL=https://your-ngrok-url.ngrok-free.app
# For production:
# BASE_URL=https://yourdomain.com
# For localhost only (webhooks won't work):
# BASE_URL=http://localhost:8000
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

## Local Development Setup

### Using ngrok for Webhook Testing

When developing locally, Twilio needs a public URL to send webhooks. Use ngrok to expose your local server:

1. **Install ngrok**
   ```bash
   # macOS
   brew install ngrok
   
   # Or download from https://ngrok.com/download
   ```

2. **Start ngrok tunnel**
   ```bash
   ngrok http 8000
   ```
   This will give you a URL like: `https://abc123.ngrok-free.app`

3. **Configure BASE_URL**
   ```bash
   export BASE_URL=https://abc123.ngrok-free.app
   ```

4. **Automatic Webhook Configuration**
   - The application automatically configures the Twilio phone number webhook on startup
   - When the app starts, it uses `BASE_URL` to automatically update the webhook URL in Twilio
   - No manual configuration needed! Just set `BASE_URL` and the webhook will be updated automatically
   - You'll see a log message: `[SMS] Successfully updated phone number X webhook to https://...`
   - **Note**: If automatic configuration fails, you'll see a warning and can manually configure it in Twilio Console

5. **Start your application**
   ```bash
   go run ./
   ```

6. **Test STOP/UNSTOP**
   - Send SMS to your Twilio number
   - Reply with "STOP" or "UNSTOP"
   - Check your application logs for `[SMS] STOP response received` or `[SMS] UNSTOP response received`
   - The webhook should appear in ngrok's web interface at http://127.0.0.1:4040

**Note**: The ngrok URL changes each time you restart ngrok (unless you have a paid plan with a fixed URL). You'll need to update `BASE_URL` each time.

## Testing

### Test SMS
- Use a real phone number (not +15551234567)
- Ensure your Twilio account has credits
- Check Twilio logs for delivery status
- For local testing: Use ngrok to expose your server for webhook callbacks

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

4. **"Webhook not receiving STOP/UNSTOP responses"**
   - **Check startup logs**: Look for `[SMS] Successfully updated phone number X webhook to...` on app startup
   - If you see a warning about failing to update webhook, check:
     - Twilio credentials are correct (TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN)
     - TWILIO_FROM_NUMBER matches the phone number in your Twilio account
     - BASE_URL is set and accessible (use ngrok for local development)
   - Ensure BASE_URL is set to a publicly accessible URL (use ngrok for local development)
   - Check ngrok web interface (http://127.0.0.1:4040) to see if requests are reaching your server
   - Check application logs for `[SMS] Webhook received` messages - if you don't see these, Twilio isn't reaching your server
   - Check application logs for `[SMS] STOP response received` or `[SMS] Processing STOP request` messages
   - Verify phone number format matches what's stored in database (E.164 format)
   - **Manual fallback**: If automatic configuration fails, manually set webhook in Twilio Console → Phone Numbers → Your Number → "A MESSAGE COMES IN"

5. **Email not delivered**
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
