# SMS OTP and Pumping Defenses

Parts Pile uses **Twilio Verify** (with Fraud Guard) for registration and
change-phone one-time codes, and **Programmable Messaging** for unread-message
alerts and inbound SMS (`STOP`, `RECOVER`).

This document is the ops and architecture runbook for SMS pumping defenses.
Product/privacy context: [DOC_PHONE_ONLY_IDENTITY_AND_SECURITY.md](DOC_PHONE_ONLY_IDENTITY_AND_SECURITY.md).

## Threat

**SMS pumping** (artificially inflated traffic): bots submit phone numbers that
route to destinations where the attacker earns a share of SMS fees. The app
pays Twilio; fraudsters farm volume through “send me a code” forms.

## Architecture

| Flow | Channel | Protection |
|------|---------|------------|
| Register OTP | Twilio Verify | Fraud Guard; Verify geo US+CA allow / elsewhere monitor+block fraud; Turnstile; rate limits |
| Change-phone OTP | Twilio Verify | Same |
| Unread alerts | Programmable Messaging | Messaging geo US+CA only; optional SMS Pumping Protection |
| Inbound STOP / RECOVER | Programmable Messaging webhook | Twilio signature verification |

Registration still issues a short-lived HttpOnly `reg_ticket` JWT after a
successful Verify check; password submission consumes that cookie (no raw OTP
in HTML).

```text
Browser → Turnstile → rate limits → Verify create
                              ↓
          Fraud Guard + Verify geo (US/CA allow; else monitor+block fraud)
                              ↓
                         SMS OTP (or block)
Browser → Verify check → reg_ticket cookie → CreateUser
```

There is **no app-level country allowlist** yet (see Deferred). Twilio Console
geo settings are the source of truth for which destinations can receive SMS.

## Console checklist (production)

Current intended settings (keep Console aligned with this doc):

1. Create a **Verify Service**; copy the Service SID (`VA…`).
2. Enable **Fraud Guard** on that Verify Service.
3. **Verify Geographic Permissions:**
   - **US** and **CA:** Allow all traffic
   - **All other countries:** Monitor all traffic and block fraud
4. **Messaging Geographic Permissions:** enable **US** and **CA** only
   (notification SMS and inbound to the Messaging number).
5. Optional: enable Programmable Messaging **SMS Pumping Protection** for
   non-OTP traffic.
6. Configure **usage / spend alerts** on the Twilio account (daily/monthly
   thresholds you will notice).

### Spend spike runbook

1. Confirm spike in Twilio Console (Messaging Intelligence / usage).
2. Temporarily disable public registration if needed (feature flag / deploy).
3. Tighten or re-check Verify and Messaging geo permissions (US+CA; consider
   Verify non-US/CA → block all if pumping continues).
4. Review Turnstile and rate-limit logs for bot traffic.
5. Rotate Twilio auth token if compromise is suspected.
6. Re-enable registration after traffic normalizes.

## Environment variables

| Variable | Required | Notes |
|----------|----------|-------|
| `TWILIO_ACCOUNT_SID` | Yes* | Shared by Verify and Messaging |
| `TWILIO_AUTH_TOKEN` | Yes* | Shared |
| `TWILIO_FROM_NUMBER` | Yes* | Messaging sender (alerts, inbound) |
| `PUBLIC_SITE_URL` | Yes* | Public site base for Twilio webhooks and SMS deep links |
| `TWILIO_VERIFY_SERVICE_SID` | Yes* | Verify Service SID (`VA…`) |
| `TURNSTILE_SITE_KEY` | Yes* | Cloudflare Turnstile site key |
| `TURNSTILE_SECRET_KEY` | Yes* | Cloudflare Turnstile secret |
| `REDIS_URL` | Yes | Shared rate-limit store (`redis://…:6379`) |
| `ALLOW_TEST_REGISTRATION` | Dev only | Skips Verify, Turnstile, and Twilio startup checks for `+1555010xxxx` |

\*Required at server start unless `ALLOW_TEST_REGISTRATION=true` (Twilio/Turnstile
only; `REDIS_URL` is always required).

### OTP start rate limits (code constants)

- Per IP: registration limiter (20 / 15 min; relaxed under test allow).
- Per phone: 1 OTP start / 60s and max 5 / hour (register and change-phone).

Counters live in **Redis** (`REDIS_URL`) so limits are shared across app
instances.

## Test / local development

With `ALLOW_TEST_REGISTRATION=true`:

- Phones matching `+1555010xxxx` skip Turnstile and Verify.
- Registration still sets `reg_ticket` before password step.
- Twilio Verify Service SID and Turnstile keys are not required at startup.

Never enable `ALLOW_TEST_REGISTRATION` in production.

## Deferred (not in this phase)

- App-level `GEO_ALLOWLIST` (Console geo is primary)
- Twilio Lookup pumping risk score before Verify create
- Carrier / prefix denylist

## Manual test checklist

- [ ] Register with real US or CA number: Turnstile → SMS → code → password → welcome
- [ ] Wrong OTP rejected; expired/missing `reg_ticket` rejected
- [ ] Change phone with Turnstile + Verify
- [ ] Test phone path with `ALLOW_TEST_REGISTRATION`
- [ ] Unread-message SMS still delivered via Messaging
- [ ] `RECOVER` inbound SMS still works
- [ ] Outside US/CA: Messaging blocked; Verify may still attempt under monitor+block-fraud (confirm Console behavior)
