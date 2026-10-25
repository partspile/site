# Rename: Rocky Ads → Parts Pile

Checklist for renaming the Oregon LLC, brand, domains, site, and external services from **Rocky Ads** to **Parts Pile**.

**Legal path chosen:** amend `Rocky Ads, LLC` → `Parts Pile, LLC` (Oregon SOS Articles of Amendment, $100). Same EIN; notify IRS after the state filing.

**Domains:** Owned — `parts-pile.com` (canonical target). Not owned — `partspile.com` (parked elsewhere; typo/spoof risk). `rockyads.com` remains during transition via redirects.

---

## P0 — Legal & brand lock-in

1. SOS name check — confirm “Parts Pile, LLC” is available in Oregon.
2. File Articles of Amendment ($100) — Rocky Ads, LLC → Parts Pile, LLC.
3. IRS name change — same EIN; notify IRS of new legal name.
4. Bank / Stripe / tax docs — update legal name (keep same EIN).
5. Claim socials — `@partspile` on X, Instagram, TikTok, YouTube, Facebook; bio → `parts-pile.com`.
6. Watch / buy `partspile.com` if listed — only clean fix for no-hyphen type-in and spoof risk.

## P1 — Domains & email (before public flip)

7. Decide primary host — e.g. `parts-pile.com` canonical; `rockyads.com` / `www` 301 → it (keep rockyads as alias during transition).
8. DNS — point `parts-pile.com` (and www) at Render; keep rockyads redirects working.
9. TLS / Render custom domains — add `parts-pile.com` + www; verify certs before cutting SEO.
10. Email — `contact@parts-pile.com` (forward `contact@rockyads.com` for a while).
11. Defensive domains if cheap — `parts-pile.net` / `.org`, `partspile.net`.

## P2 — Site config & copy

12. Render env — `APP_NAME=Parts Pile`, `CONTACT_EMAIL=contact@parts-pile.com`, `PUBLIC_SITE_URL=https://parts-pile.com` (or chosen primary).
13. UI strings — titles, footer, FAQ, about, OG/meta, schema.org `WebSite` name, SMS/email copy that say “Rocky Ads”.
14. Legal pages — Terms / Privacy entity → Parts Pile, LLC; update contact.
15. Favicon / logo — Parts Pile mark; retire Rocky branding from first viewport.
16. Canonical + redirects — old rockyads URLs 301 to new host; sitemap / robots.
17. Tests — hardcoded `rockyads.com` / `Rocky Ads` (e.g. `page_test.go`).

## P3 — External services

18. Twilio — friendly name / Messaging Service / Verify templates / webhook URLs if host changes; A2P brand/campaign legal name if filed under Rocky Ads.
19. Cloudflare Turnstile — add `parts-pile.com` (keep rockyads during transition).
20. MinIO / CORS — allow PUT/GET from new origin; update `MINIO_PUBLIC_URL` if hostname is branded.
21. Geoapify / x.ai (Grok) — update billing/account name if tied to old LLC (optional).
22. Domain registrar / Cloudflare — registrant org → Parts Pile, LLC after SOS amend.
23. Google Search Console / Analytics / Bing (if any) — new property + verify; leave old with redirects.
24. App stores — N/A until apps ship; reserve names later if needed.

## P4 — Infra identifiers (can lag; higher churn)

25. GitHub — display names; optional rename `rockyads` → `partspile` (breaks clone URLs — do carefully or leave technical names).
26. Docker Hub `rockyads/*` — optional retag/new repos; update `render.yaml` / GitHub Actions.
27. Render service names — cosmetic; update when convenient.
28. Postgres DB name / MinIO bucket `rockyads` — **optional**; renaming prod is disruptive — keep internal IDs unless you want purity.
29. Docker Compose local names — rename when touching local DX.
30. Docs — README, `doc/DOC_*`, comments: Rocky Ads → Parts Pile.

## P5 — Trademark & go-live

31. USPTO Class 35 DIY (or attorney) — “Parts Pile” for classified advertising / online marketplace after real use or ITU.
32. Announce — banner / in-product notice: “Rocky Ads is now Parts Pile”; update socials.
33. Monitor — spoof of `partspile.com`, fake `@partspile`, Search Console indexing.
34. Retire Rocky marketing — keep redirects 12+ months after cutover.

---

## Suggested timeline

| When | Focus |
|------|--------|
| Week 1 | SOS amend → IRS/bank → claim socials → email + DNS plan |
| Week 2 | Render env + UI/legal copy + Turnstile/Twilio/MinIO CORS + redirect cutover |
| Week 3+ | Search Console, TM filing, optional Docker/GitHub renames, typo-domain watch |

## Blockers vs can-wait

**Block public flip on:** primary domain TLS, `APP_NAME` / contact, Turnstile domains, Twilio webhooks if host changes, legal name on payment/tax accounts if required.

**Do not block on:** renaming Postgres/MinIO bucket, Docker Hub org, or GitHub repo paths — those are internal.

---

## Oregon SOS fee reference

| Action | Fee |
|--------|-----|
| Articles of Amendment (rename) | $100 |
| Articles of Organization (new LLC) | $100 |
| Articles of Dissolution | $100 |
| Assumed business name (DBA) only | $50 |

Rename path chosen to keep the same LLC registry number and EIN with less migration mess than forming a new entity.

## Related notes

- GitHub org `partspile` already exists and points at `https://parts-pile.com`.
- `parts-pile.com` currently redirects to `rockyads.com`.
- Knock-out search found no exact “Parts Pile” USPTO mark; name is descriptive (weaker mark, clearer product). Not a clearance opinion.
