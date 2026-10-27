# Phone-Only Identity at Parts Pile

## Privacy by Minimal Collection, and the Security Questions That Remain

**Document type:** Design and security analysis  
**Audience:** Product, engineering, and security reviewers  
**Scope:** How Parts Pile uses a phone number as the sole contact identifier for registration, account management, and communications; why other personal data is intentionally omitted; and how the system can still be attacked despite a small PII footprint.  
**Based on:** Parts Pile site implementation (Go / Fiber / Postgres / Redis / Twilio Verify + Messaging / Turnstile / JWT / Geoapify), August 2026.  
**Companion runbook:** [DOC_SMS_OTP_AND_PUMPING_DEFENSES.md](DOC_SMS_OTP_AND_PUMPING_DEFENSES.md) (OTP pumping, Console geo, spend alerts).

---

## Abstract

Parts Pile is classified advertising rebuilt for the web: post an ad, get contacted through in-app messaging, and keep your phone number hidden from other users. The product thesis is deliberately austere: **no email, no social graph, no payment card on file for posting, and no profile dossier.** The one piece of personal contact information the system requires is a text-capable phone number in E.164 form.

That choice is not a gimmick. It is a privacy architecture. Collecting less personal data reduces the blast radius of a breach, simplifies compliance narratives, and matches the mental model of classic newspaper classifieds—where a phone number was enough to complete a transaction. At the same time, **low PII is not the same as low risk.** Phone numbers are high-value identifiers. SMS channels can be abused. Sessions, passwords, message content, uploaded media, and operational secrets remain attractive targets. This paper explains the product rationale, maps how phone numbers are used in the running system, and explores realistic exploit paths that survive even when the user record is intentionally thin.

Recent hardening (Twilio Verify, registration tickets, Turnstile, Redis OTP/IP limits) closes several earlier gaps—especially plaintext in-app OTPs and unbound registration step 3—without changing the phone-only product model.

---

## 1. Product thesis: one identifier, many jobs

Traditional consumer marketplaces accumulate identity over time: email for login and receipts, phone for two-factor auth, address for shipping, payment methods for fees, and often a social login for convenience. Each field expands the attack surface and the privacy liability.

Parts Pile reverses that default. The FAQ states the ideal plainly: the service would prefer to collect no personal information at all, but still needs a reliable way to reach a real person. A text-capable phone number fills three roles at once:

1. **Proof of a reachable human** during registration (one-time SMS verification via Twilio Verify).
2. **Account recovery channel** when a user loses password access (inbound SMS proof of possession on Programmable Messaging).
3. **Operational notification channel** when unread messages arrive (outbound Programmable Messaging with a link into the authenticated messages UI).

Everything else is optional or absent by design. Login itself is username plus password, not phone-as-password. Other users never see the number. Marketing SMS unrelated to the account is out of scope. The public surface of a member is closer to a classified-ad persona than to a social-network profile: username, ads, optional account picture, and activity signals—not a contact card.

The economic and UX claims in the product README reinforce the same boundary: no Facebook friends graph, no posting fees, no credit cards required to participate. Phone verification substitutes for the weak “anyone can invent an email” barrier without importing an entire CRM stack.

---

## 2. What counts as PII here—and what Parts Pile refuses to collect

For this paper, **PII** means data that identifies or can reasonably identify a natural person. Phone numbers clearly qualify. Usernames may or may not, depending on whether they encode a real name; Parts Pile encrypts them at rest anyway. Content users voluntarily type into ads or messages can contain arbitrary sensitive data; that is user-generated content, not a registration requirement.

### Intentionally not collected as account fields

| Category | Status |
|----------|--------|
| Email address | Not collected |
| Legal / government ID | Not collected |
| Mailing / home address | Not collected (ads may have location facets) |
| Payment card / bank account | Not required for posting |
| Social login / friends graph | Not used |
| Date of birth, gender, employer | Not collected |
| Device advertising IDs as identity | Not used as the account key |

### What is stored for an account

At the user-record level, the durable personal contact field is the phone number (E.164), stored encrypted with AES-GCM under a server-held key, with an HMAC-SHA256 lookup hash (peppered with `DB_HASH_PEPPER`) for uniqueness checks. The username is similarly encrypted and hashed. Passwords are stored as Argon2id hashes. Soft-delete leaves a time-bounded hold so a phone cannot be immediately reused after account deletion.

Ephemeral or operational stores also touch identity briefly:

| Store | Contents |
|-------|----------|
| **Twilio Verify** | Register / change-phone OTPs (out of process; not in app DB) |
| **Redis** (`REDIS_URL`) | Registration/recovery IP counters; per-phone OTP start keys (`otp:cd:…`, `otp:hr:…`) that include E.164 |
| **`reg_ticket` cookie** | Short-lived JWT binding verified username + phone after OTP success |
| **`account_recovery`** | HMAC’d session/code; bound `user_id` after inbound SMS |
| **`sms_notification_queue`** | User/conversation IDs for unread alerts |
| **`locations`** | Cached geocodes (city/admin/country/lat/lon) from **Geoapify** on cache miss |
| **Server logs** | IP, UA, path; some SMS logs may include phone E.164 |
| **Admin UI / backups** | Web admin is insights-only (no usernames/phones); jump-server `cmd/admin` lists users and reveals phones / user writes; ciphertext in backups |

There is **no** application `phone_verification` table storing plaintext OTPs (removed when OTP moved to Twilio Verify).

The privacy claim is therefore precise: **Parts Pile does not need other contact PII to operate the product**, not that the system stores zero information about people. Ads, images, message journals, click history, and geo facets still create a behavioral and content footprint. Minimizing *required contact PII* is the design win; it does not erase all privacy risk.

---

## 3. How the phone number is used end-to-end

### 3.1 Registration

Registration is a three-step HTMX flow:

1. **Step 1 — claim username and phone.** The user consents to SMS (`offers=true`), passes **Cloudflare Turnstile** (unless test registration), the system checks availability, enforces **per-IP registration limits** and **per-phone OTP start limits** (both in Redis), screens the username via Grok, and starts a **Twilio Verify** SMS (Fraud Guard + Verify geographic permissions; intended Console: US+CA allow, elsewhere monitor+block fraud).
2. **Step 2 — prove possession.** The user submits the OTP. The server runs a Verify check; on success it sets a short-lived HttpOnly **`reg_ticket` cookie**: a JWT (HS256 with `JWT_SECRET`) bound to that username and phone (~10 minute TTL). The raw OTP is not placed in the password form.
3. **Step 3 — set password and accept terms.** Before `CreateUser`, the handler **consumes** the cookie (`registrationticket.Consume`): verify signature and expiry, check username/phone match the form, clear the cookie. Only then is the user row created, a JWT session cookie issued, and the browser redirected to the authenticated welcome path. Replay of a consumed flow is blocked in practice by username/phone uniqueness on create.

Test environments may allow a reserved phone range (`+1555010xxxx`) to skip Turnstile and Verify when `ALLOW_TEST_REGISTRATION=true`; those paths still issue and consume the registration cookie. That flag must never be on in production. `REDIS_URL` is always required at startup.

### 3.2 Day-to-day account management

Once registered, the phone number remains the user’s contact anchor:

- Settings displays the current number and supports **change phone** (password + Turnstile + Verify OTP to the new number; same per-phone OTP start limits).
- Users can toggle SMS notifications (`sms_opted_out`).
- Password changes invalidate sessions by rotating a salt bound into the JWT claims.
- Account deletion is soft-delete with a phone reuse hold (on the order of ten days).
- Distance-unit defaults can be inferred from phone region without asking for a home address.

Login remains username/password. The phone is not a daily credential; it is the recovery and notification backbone.

### 3.3 Communications

| Channel | Use |
|---------|-----|
| **Twilio Verify** | Registration and change-phone OTP |
| **Programmable Messaging** | Unread-message alerts; inbound `STOP` / `RECOVER` |

A background worker drains a notification queue, applies suppression (for example, avoiding SMS spam when the user was recently texted or has no unread messages), decrypts the phone, and sends via Messaging. Intended Messaging geo: **US + CA only**.

**STOP vs in-app opt-out:** inbound `STOP` / `START` (and common Twilio opt keywords) sync `sms_opted_out` so Settings matches what the user texted. Carrier-level blocking may still prevent Messaging delivery until the user texts `START`; FAQ covers that residual case. OTP codes remain in Twilio Verify (short TTL) and are not cleared by STOP.

Account recovery is possession-based rather than email-based: the browser starts a short-lived recovery session and shows a code; the user texts that code from their registered number; the webhook, after Twilio signature verification, binds the session to the matching user; the browser then reveals the username and allows password reset. Recovery starts are rate-limited per IP in Redis.

### 3.4 What other users see

Other members interact through usernames, ads, and the messaging product. The phone stays server-side. That preserves the classifieds metaphor—people call *the system’s messaging layer*, not a publicly printed mobile number—while still letting Parts Pile nudge the seller when activity happens.

### 3.5 Location (related contact-adjacent data)

Ad location text is resolved via **Geoapify** on cache miss and stored in `locations` (city, admin area, country, lat/lon). That allows durable caching under Geoapify’s terms (with attribution). It is not phone PII, but it is another third party that sees free-text location queries.

---

## 4. Why “phone only” is a real privacy improvement

### 4.1 Smaller breach catalog

If an attacker obtains a user table, the absence of emails, payment instruments, and home addresses means there is less immediately monetizable identity material of those types. There is no email list to dump into spam corpora, no card PANs to sell, no mailing list to dox. For many breach narratives, that is a meaningful reduction in harm categories.

### 4.2 Smaller third-party fan-out

Email-centric products often share addresses with ESP vendors, support desks, analytics, and CRM tools. Parts Pile’s contact channel is SMS via Twilio (Verify + Messaging), plus Turnstile for bots and Geoapify for geocoding. That is still a concentrated set of third parties—but it avoids the sprawling email marketing stack that usually accumulates shadow copies of identity.

### 4.3 Better match to threat of over-collection

Many privacy failures are not exotic zero-days; they are products that asked for too much “just in case.” Phone-only registration is a forcing function against that habit. Features that would normally demand email (receipts, newsletters, magic links) must either be redesigned around SMS/in-app UX or rejected.

### 4.4 Human-cost reduction for casual users

Classifieds users often want transactional anonymity relative to strangers. Hiding the number from other users while still verifying it for the platform is a concrete improvement over printing a cell number in an ad.

---

## 5. Security properties that already support the model

Parts Pile is not relying on obscurity alone. Relevant controls in the current stack include:

- **Password hashing** with Argon2id.
- **Field encryption** for username and phone (AES-GCM with per-user key derivation from `DB_ENCRYPTION_KEY`).
- **Peppered lookup hashes** for username/phone (`HMAC-SHA256` with `DB_HASH_PEPPER`) so a DB dump alone cannot offline-dictionary E.164 / username space.
- **Message journal encryption** at rest.
- **JWT session cookies** (HttpOnly, SameSite=Strict, Secure outside local development), with password-salt binding so password changes invalidate tokens.
- **CSRF** protection (double-submit), with a deliberate exemption for the Twilio webhook path.
- **Helmet / CSP** and related browser hardening headers.
- **Redis-backed rate limits:** registration IP (20 / 15 min), recovery start IP (20 / 15 min; reloads of an active session do not count), login IP (20 / 15 min), per-username login failures (10 / 15 min, cleared on success), per-phone OTP starts (1 / 60s and max 5 / hour); shared across app instances via `REDIS_URL`.
- **Twilio Verify** for register/change-phone OTP (Fraud Guard + Console geo), then a signed HttpOnly **`reg_ticket` cookie** for step 3 so the password form never round-trips the raw OTP.
- **Cloudflare Turnstile** before OTP start (register and change-phone).
- **Recovery secrets** stored as HMAC digests rather than raw tokens.
- **Twilio request signature verification** on inbound webhooks.
- **Soft-delete phone hold** to slow account recycling abuse.
- **Geoapify** for forward geocoding with durable `locations` cache (allowed storage model).

Ops detail for pumping and Console settings: [DOC_SMS_OTP_AND_PUMPING_DEFENSES.md](DOC_SMS_OTP_AND_PUMPING_DEFENSES.md).

These controls matter because the phone-only model concentrates trust in SMS possession, password strength, session integrity, and the secrecy of encryption/JWT/Twilio/Redis credentials. The following sections assume those controls exist and ask what still goes wrong.

---

## 6. Low PII is not low attack surface

A useful mental model: **privacy minimization shrinks the value of some data dumps; it does not shrink the number of ways to hijack an account, spam a channel, or extract user-generated content.** In some cases it even intensifies pressure on the remaining identifier.

### 6.1 The phone number is a high-value singleton

On the **carrier / SMS-possession** axis, Parts Pile is in the same club as WhatsApp (and Signal, iMessage registration, and many “phone-only” messengers): whoever can receive SMS for the number can prove possession to the service. Parts Pile does not escape SIM swap, SS7 abuse, malicious MVNO apps, shared family plans, stolen handsets, or social-engineered ports. That weakness lives at the phone network, not in how little other PII is stored.

Where Parts Pile **differs** from WhatsApp is how the number is *used* after proof:

| | WhatsApp-style messenger | Parts Pile |
|--|--------------------------|-----------|
| Daily auth | Phone (re)registration / device link | Username + password |
| Public / peer identity | Number is the account handle; contacts can discover you | Number is hidden; peers see a username |
| How strangers reach you | Message the number (or a linked identity) | In-app messaging to the username / ad |
| Phone’s jobs here | Identity + delivery address | Verify, recover, and notify only |

So “phone only” means different things. WhatsApp makes the number the *social* identity. Parts Pile makes it a *platform-side contact key* and deliberately keeps a second channel (password) for day-to-day login. That split helps in one direction and hurts in another:

- **Helps:** a SIM swap alone does not equal an already-logged-in browser session the way a fresh WhatsApp registration on a stolen number can. The attacker still needs recovery (SMS) *or* the password (stuffing / phishing).
- **Hurts:** password stuffing and weak passwords become first-class risks WhatsApp largely sidesteps by not using passwords. And for **registration, change-phone, and recovery**, SMS possession remains the lifeline—closer to WhatsApp’s model than to “email + SMS 2FA,” where those are separate channels.

**Implication:** privacy minimization does not buy Parts Pile a stronger phone channel than WhatsApp’s. It buys a thinner account dossier and a hidden number. Carrier-layer SMS identity risk still applies wherever the product gates on SMS.

### 6.2 Registration and OTP economics

Even with current controls, SMS OTP systems face familiar abuses:

- **SMS pumping / toll fraud:** bots submit numbers that route SMS to premium or incentivized destinations, burning Twilio spend. Mitigations in place: Twilio Verify + Fraud Guard, Verify/Messaging geo (US+CA posture), Turnstile, per-IP/per-phone OTP start limits, spend-alert runbook — see [DOC_SMS_OTP_AND_PUMPING_DEFENSES.md](DOC_SMS_OTP_AND_PUMPING_DEFENSES.md). Deferred: app-level geo allowlist, Lookup risk scores, prefix denylist.
- **OTP flooding / harassment:** mitigated by Turnstile, Redis per-phone cooldown/hourly caps, and registration IP limits; change-phone shares OTP-start limits.
- **Step binding (mitigated):** registration previously trusted step 2 alone at password submission. That gap is closed: step 2 completes Twilio Verify and sets `reg_ticket`; step 3 must present that cookie matching the form.
- **Redis phone keys:** OTP limit keys embed E.164. A Redis dump does not yield OTPs or passwords, but it can reveal which numbers recently requested codes—treat Redis as sensitive alongside Postgres.

OTP codes for register/change-phone are owned by Twilio Verify and are not stored in the application database.

### 6.3 Password and session attacks still dominate day-to-day risk

Because daily login is username/password:

- credential stuffing and password spraying remain relevant;
- XSS that steals nothing “PII-like” can still steal a session cookie if script injection ever appears;
- CSRF protections must stay correct for state-changing routes;
- a leaked `JWT_SECRET` is catastrophic regardless of how little PII is stored.

Low PII does not help a user whose password is `Summer2024!` and whose username is guessable from their public ads. Login is throttled per IP (Fiber) and per username (Redis failure lockout); residual risk is distributed stuffing below those caps and weak passwords.

### 6.4 Account recovery as a target

The recovery design is elegant for a no-email product, but it creates a crisp attack graph:

1. Attacker starts many recovery sessions (mitigated by per-IP Redis rate limits).
2. Attacker tries to guess or intercept the on-screen code path, or socially engineer the victim to text a code.
3. If the attacker can send SMS *from* the victim’s number (compromised phone, spoofing depending on carrier path—Twilio’s `From` is the authentic inbound peer for webhook handling), they bind recovery and reset the password.

Webhook security is load-bearing. Signature verification must use the correct public URL; trusting mis-set `X-Forwarded-*` headers in reverse-proxy configurations can weaken or break that validation. The recovery endpoint is CSRF-exempt by necessity for Twilio, which makes signature checks non-optional.

### 6.5 Lookup hashes and offline correlation

Encrypting phone and name at rest is strong against casual database reads **if** `DB_ENCRYPTION_KEY` remains secret. Lookup columns (`name_hash`, `phone_hash`) use **HMAC-SHA256 with `DB_HASH_PEPPER`**, so an attacker with only the database (no pepper) cannot dictionary-match E.164 numbers or usernames against those hashes. A dump that also includes the pepper (or both secrets) still allows recomputing hashes; the encryption key separately unlocks ciphertext. Keep `DB_HASH_PEPPER` distinct from `DB_ENCRYPTION_KEY` and treat both as production secrets.

### 6.6 Content and metadata are still PII-rich

Users will put phone numbers, meetup addresses, and workplace details into ad text and messages. Image uploads may contain faces, license plates, or document photos. Geo facets and the `locations` table describe where goods are. Click and bookmark tables describe interest graphs. Encrypted journals protect disks and backups better than plaintext, but participants and anyone who can act as a participant (account takeover) can read them.

In other words: **the registration form is minimal; the product corpus is not.** A breach of message stores or object storage is a privacy incident even if the `users` table lacks email.

### 6.7 Admin and insider risk

Web admin is insights-only (SMS queue, embeddings, clicks aggregates)—no usernames or phones. User PII and write ops (list, show phone, promote/demote/delete, DB tools, embedding backfill) live on jump-server `cmd/admin`. Compromised admin JWTs or jump-server access to production keys remain privileged-path risks; promote/demote takes effect on the next request because `JWTMiddleware` refreshes `is_admin` from the DB into locals (it is not trusted from the JWT claim). Low external PII does not remove insider risk.

### 6.8 Third-party and configuration failures

| Dependency | Failure mode |
|------------|----------------|
| Twilio (Verify + Messaging) | Account takeover → SMS redirect, outbound spam as Parts Pile, inbound webhook forgery if signatures fail; Verify Console misconfig reopens pumping |
| `PUBLIC_SITE_URL` misconfig | Webhooks and SMS deep links both use this base; wrong host breaks Twilio and/or phishes users |
| Cloudflare Turnstile | Misconfig / downtime can block registration or (if bypassed in code paths) weaken bot friction |
| Redis | Compromise exposes rate-limit keys (including recent E.164s) and can disable OTP/IP throttles |
| Grok | Usernames (and other AI features: rock opinions, suggestions, compress) leave the trust boundary |
| Geoapify | Location query text leaves the trust boundary; durable cache in Postgres is intentional |
| MinIO / presigned URLs | Stolen long-lived GET URLs leak images; PUT URL abuse if minting is too loose |
| `ALLOW_TEST_REGISTRATION` | Skips Turnstile and Verify for allowlisted numbers if enabled in prod |
| Ollama | Embeddings only today; expands operational attack surface |

### 6.9 Product abuse that is not “hacking the crypto”

Even with perfect cryptography, a classifieds site faces:

- scam ads and advance-fee fraud,
- spam accounts (phone verification raises cost but does not eliminate VoIP/SMS farms),
- harassment via messaging (SMS notifications become part of the harassment loop if not rate-limited and easy to disable),
- enumeration side channels (phone-hold messages, timing of availability checks, distinct error paths).

These harms do not require a large PII database. They require a working marketplace.

---

## 7. Example exploit narratives (even with phone-only accounts)

The following scenarios are illustrative threat stories grounded in how the system works. They are not claims of currently exploited CVEs.

### Narrative A — SIM swap recovery takeover

An attacker social-engineers the victim’s carrier, ports the number, starts recovery on Parts Pile, texts `RECOVER <code>` from the newly controlled line, resets the password, and reads message history about local meetups. No email inbox was needed. The “low PII” database still yielded conversation content and ad activity once the account was owned.

### Narrative B — Database / Redis backup without the encryption key

An attacker steals a logical Postgres backup. AES-GCM phone ciphertext resists direct read. Register/change-phone OTPs are not in the app database (Twilio Verify). Peppered phone hashes resist offline E.164 dictionaries without `DB_HASH_PEPPER`. A separate Redis dump may list E.164s that recently hit OTP start limits. The attacker may then phishing-SMS those numbers with a lookalike “verify your Parts Pile phone” link.

### Narrative C — Registration step confusion / race *(mitigated)*

**Previously:** an attacker who knew an in-flight username + phone could POST step 3 without the OTP, because account creation trusted that step 2 had already succeeded. The victim’s verified phone could become an ownership-transfer mechanism. An earlier interim also kept OTP material in the password form / app store.

**Now:** step 2 completes Twilio Verify and sets a signed `reg_ticket` cookie; step 3 verifies and clears that cookie. Without the cookie, enrollment fails. Remaining related risks are cookie theft from the browser and SMS/Verify-channel attacks—not username/phone guessing alone. See [DOC_SMS_OTP_AND_PUMPING_DEFENSES.md](DOC_SMS_OTP_AND_PUMPING_DEFENSES.md).

### Narrative D — SMS brand phishing via config foot-gun

A deployment sets `PUBLIC_SITE_URL` to a wrong host. Legitimate unread-message texts and Twilio webhooks both use that host. Users, trained by Parts Pile to trust SMS links into `/auth/user/messages`, may land on an attacker site. The application’s cryptography never failed; the operational URL did. Mitigation is ops discipline (set `PUBLIC_SITE_URL` to the real public site), not a second base URL.

### Narrative E — Credential stuffing without any phone involvement *(mitigated)*

Public usernames from ads are tried with passwords from unrelated breaches against `/api/login`. Per-IP login ceilings and per-username failure lockouts slow spraying; residual risk is slow distributed stuffing and weak reused passwords. On success, the attacker can still disable SMS notifications or change the phone after password entry, isolating the victim from alerts. Phone-only registration does not slow this path because login does not require the phone.

### Narrative F — Marketplace trust exploits

A scammer verifies a disposable SMS number (harder with Fraud Guard / geo / Turnstile, not impossible), posts attractive ads, and social-engineers buyers inside encrypted message journals. When reports arrive, the operator has a phone hash and ciphertext—not a rich KYC file. The low PII posture that protects honest users also limits investigative breadcrumbs.

### Narrative G — SMS pumping *(partially mitigated)*

Bots drive Verify creates toward premium destinations. Current stack raises cost (Turnstile, Redis limits, Fraud Guard, geo). Residual risk is Console drift, new pumping routes, and notification Messaging paths—hence the spend-alert runbook.

---

## 8. Risk matrix (condensed)

| Threat | Needs large PII store? | Primary impact | Severity if unmitigated | Notes |
|--------|------------------------|----------------|-------------------------|--------|
| SIM swap / SMS interception | No | Account recovery / phone change | High | Carrier-layer |
| Password stuffing / weak passwords | No | Full account takeover | High | Login IP + per-username lockout; weak passwords remain |
| JWT or encryption key leak | No | Mass session or field decrypt | Critical | |
| Twilio console compromise | No | SMS redirect, spam, trust break | Critical | |
| SMS pumping | No | Direct financial loss | Medium–High | Mitigated; ops-dependent |
| Registration step-3 without OTP | No | Account hijack at signup | ~~High~~ Mitigated | `reg_ticket` + Verify |
| In-app plaintext OTP table | No | Signup hijack + phone harvest | ~~High~~ Mitigated | Moved to Verify |
| Redis compromise | No | Recent E.164s; throttle bypass | Medium | Treat as sensitive |
| Presigned media URL leak | No | Image/PII-in-photo exposure | Medium | |
| Admin token abuse | No | Bulk phone disclosure | High | |
| Content scams / harassment | No | User harm, brand trust | High | |
| Offline phone-hash correlation | No | Re-identification of members | ~~Medium~~ Mitigated | Needs `DB_HASH_PEPPER` + DB |
| Geoapify / Grok third-party leak | No | Location text / usernames leave boundary | Low–Medium | By design for those features |

The pattern is consistent: **almost none of the serious threats require Parts Pile to have collected email or credit cards.**

---

## 9. Recommendations aligned with the phone-only philosophy

Status relative to the current codebase:

1. ~~**Treat registration as a single server-side capability token.**~~ **Done.** Step 2 completes Twilio Verify and sets a signed HttpOnly `reg_ticket` JWT cookie; step 3 verifies binding and clears it.
2. ~~**Keep register/change-phone OTPs out of the app DB.**~~ **Done.** Twilio Verify owns codes.
3. ~~**Pepper phone/name lookup hashes**~~ **Done.** `db.HashString` is HMAC-SHA256 with required `DB_HASH_PEPPER`; startup `user.RehashLookupHashes` upgrades legacy unsalted rows; restore recomputes hashes.
4. ~~**Add login-specific throttling and lockouts** (per username and per IP), not only global request ceilings.~~ **Done.** Fiber login IP limiter (20 / 15 min) plus Redis per-username failure lockout (10 / 15 min, cleared on success).
5. ~~**Rate-limit OTP starts and monitor Twilio spend.**~~ **Done (OTP path):** Verify + Fraud Guard, Turnstile, Redis per-phone/IP limits; spend runbook in [DOC_SMS_OTP_AND_PUMPING_DEFENSES.md](DOC_SMS_OTP_AND_PUMPING_DEFENSES.md). Notification SMS remain on Programmable Messaging.
6. ~~**Make STOP and in-app opt-out consistent**~~ **Done.** Inbound STOP/START (and common Twilio opt keywords) set/clear `sms_opted_out`; FAQ notes residual carrier blocking until START.
7. ~~**Bound notification link bases** to a first-party canonical site URL~~ **Superseded.** Single `PUBLIC_SITE_URL` is the public base for both SMS deep links and Twilio webhooks (replaces `TWILIO_WEBHOOK_URL`). No provider-specific URL fallbacks.
8. ~~**Shorten admin phone exposure**~~ **Partial.** Web admin shows no user PII (insights only). Jump-server `cmd/admin` Users list reveals a phone on demand and owns promote/demote/delete. ~~Privilege changes take effect on the next request~~ (`JWTMiddleware` loads `is_admin` from DB into locals; not a JWT claim). Still open: optional audit logs for phone reveal.
9. ~~**Document SIM-swap reality**~~ **Done.** Recovery waiting panel and `/faq/account-recovery` explain that phone possession can reset the password and that users should protect carrier accounts.
10. **Keep `ALLOW_TEST_REGISTRATION` impossible in production** via startup refusal when release mode is live. *(Still open — policy + env discipline today.)*
11. **Preserve the non-collection stance** when new features are proposed—receipts, digests, and “magic links” should not silently reintroduce email as a second identity without a deliberate privacy review.
12. **Treat Redis as PII-adjacent** (ACLs, no public exposure, backups/retention policy) because OTP keys contain E.164. *(Operational — enforce in deploy.)*
13. **Optional later:** run username screening and other chat completions on local Ollama. *(Geocoding already uses Geoapify; keep it that way — don’t use LLMs for coordinates. Self-hosted Nominatim is an optional later alternative to Geoapify.)*

---

## 10. Conclusion

Parts Pile’s phone-only approach is a coherent privacy strategy: one practical contact channel, hidden from other users, reused for verification, recovery, and message notifications, with a deliberate refusal to assemble an email-and-profile dossier. That stance reduces certain breach harms and keeps the product honest about classifieds-scale identity.

Recent work moved OTP ownership to Twilio Verify, bound registration completion to a signed `reg_ticket`, added Turnstile and Redis-backed OTP/IP/login limits, and documented pumping defenses. Those changes shrink several of the highest-leverage *application* bugs and cost attacks. They do not shrink carrier-layer SIM risk, weak passwords under the login caps, JWT/key compromise, or content scams.

The correct reading of the architecture remains: not “little PII, therefore safe,” but **“little PII, therefore the remaining controls on phone proof, sessions, secrets, and content access must be excellent.”** Privacy minimization is necessary and valuable. For Parts Pile, it is also a bet that engineering discipline around a single contact identifier can outperform the false comfort of collecting more personal data “for security.” That bet only pays off if the phone channel, the password channel, and the operational envelope keep hardening with the same seriousness that motivated collecting so little in the first place.

---

## Appendix A — Primary code map

| Area | Location |
|------|----------|
| Registration / OTP handlers | `internal/handler/register.go` (Turnstile + Verify + `reg_ticket`) |
| Twilio Verify | `internal/service/verify/` |
| Turnstile | `internal/service/turnstile/` |
| Registration ticket cookie | `internal/registrationticket/`, `internal/cookie/register_ticket.go` |
| OTP start limits (Redis) | `internal/otplimit/`, `internal/kv/` |
| IP registration/recovery/login limiters | `internal/handler/ratelimit.go` |
| Login username failure lockout | `internal/loginlimit/` |
| SMS webhook (STOP/START → opt-out) | `internal/handler/sms.go` |
| Login / JWT | `internal/handler/login.go`, `internal/handler/jwt.go`, `internal/cookie/jwt.go` |
| Recovery | `internal/handler/recover.go`, `internal/accountrecovery/` |
| SMS send/queue/worker/webhook | `internal/service/sms/`, `internal/handler/sms.go` |
| Settings / change phone | `internal/handler/user.go`, `internal/user/` |
| Geocoding | `internal/service/geoapify/`, `internal/location/resolve.go` |
| Encryption helpers | `internal/encryption/`, `internal/user/encryption.go` |
| Lookup hashes | `internal/db/db.go` (`HashString` + `SetHashPepper`) |
| Hash rehash migration | `internal/user/rehash.go` |
| Schema | `internal/db/schema.sql` |
| Privacy / FAQ copy | `internal/ui/legal.go`, `internal/ui/faq.go` |
| Env and dependency overview | `README.md`, `doc/DOC_TECH_STACK.md` |
| SMS pumping runbook | `doc/DOC_SMS_OTP_AND_PUMPING_DEFENSES.md` |

## Appendix B — Related public claims

Product README: phone number is enough to get started; no email; number stays hidden from the classifieds-style public view.  
In-app FAQ (`/faq/phone-number`): phone is collected mainly for message notifications and reachability; email, mailing address, and payment details are not requested as contact profile data; numbers are not sold or shown to other users.  
In-app FAQ (`/faq/account-recovery`): recovery is phone-possession based; SIM swap / carrier takeover can reset the password; carrier-account protections recommended.

## Appendix C — Change log (security-relevant)

| Change | Effect on this analysis |
|--------|-------------------------|
| Twilio Verify + Fraud Guard / geo | OTPs leave app DB; pumping mitigations; Console becomes load-bearing |
| `reg_ticket` cookie | Closes unbound registration step 3 |
| Turnstile | Bot friction before OTP start |
| Redis (`REDIS_URL`) rate limits | Shared OTP/IP/login throttles; E.164 in OTP Redis keys |
| Login IP + per-username lockout | Slows credential stuffing on `/api/login` |
| STOP/START syncs `sms_opted_out` | Carrier unsubscribe matches Settings preference |
| `PUBLIC_SITE_URL` | Single public base for SMS links + Twilio webhooks (no `TWILIO_WEBHOOK_URL`) |
| Geoapify for locations | Replaces LLM geocoding; durable cache OK under Geoapify terms |
| `DB_HASH_PEPPER` + HMAC lookup hashes | Offline dictionary of `phone_hash` / `name_hash` needs pepper |
| Companion SMS pumping doc | Ops runbook split from this privacy/threat paper |
| Live `is_admin` via `SessionAuth` | Promote/demote effective next request; not a JWT claim |
| Recovery + FAQ SIM-swap copy | Users warned phone possession can reset password |
