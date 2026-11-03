# Flags at Parts Pile

## Scarce Peer Disputes, Visible Consequences, Provisional Assessment

**Document type:** Product design, threat, and legal analysis  
**Audience:** Product, engineering, trust & safety, and legal reviewers  
**Scope:** How Parts Pile flags work end-to-end after the throw-confirm UX; the philosophy behind scarcity-based, peer-visible disputes; dispute assessment (optional preview, on-demand store); known flaws; abuse paths; and legal tensions of self-policing with public flag signals and AI paraphrase.  
**Based on:** Parts Pile site implementation (Go / Fiber / Postgres / Grok flag opinions / encrypted conversation journals), August 2026.  
**Companions:** [DOC_PHONE_ONLY_IDENTITY_AND_SECURITY.md](DOC_PHONE_ONLY_IDENTITY_AND_SECURITY.md) (identity / SMS); [DOC_LOCAL_LAUNCH_CAR_TRUCK_PARTS.md](DOC_LOCAL_LAUNCH_CAR_TRUCK_PARTS.md) (go-to-market: flags as trust signal, not a substitute for liquidity).

---

## Abstract

Parts Pile treats classifieds trust as a **scarce, peer-enforced signal**, not a staffed ticket queue. Every member may keep at most **three outstanding flags**. Throwing one on a conversation marks a dispute, journals a canned reason, and increments a visible flag count on either the **ad** (inquirer threw) or the **inquirer’s profile** (owner threw). Flag icons on ads (and the **dispute assessment** behind them) are **public**—no login required. The assessment names parties only as **Owner** and **Inquirer**, never by username. Enough **ad-bound** flags remove the listing from search.

The throw path is deliberate, not a one-click flag: a confirm modal shows remaining flags, direction-specific caution, required reason radios (with examples; policy links to Terms), and an optional **Review assessment** that opens a stacked preview modal. Throw itself is fast—it does **not** call Grok. A stored dispute assessment is generated **on demand** when someone opens the opinion from the conversation or a flag icon. Pre-throw review is ephemeral and never written to `flag_opinions`.

The design is anti-bureaucratic. Terms and FAQ push ordinary listing problems into the product: throw, talk it out, lower when resolved. Ads are **assumed good until flaged**. That bet buys triage (“who cares enough to spend a flag?”) and a brand metaphor that matches the product. It also creates sharp edges: scarcity can be gamed or exhausted; publicity of chats conflicts with private-DM expectations; AI opinions leave the trust boundary; and self-policing is incomplete for statutory takedown and CSAM. This paper maps the current mechanism, philosophy, failure modes, and legal posture without pretending flags are a complete trust system.

---

## 1. Motivation: why flags exist

Classic newspaper classifieds had almost no platform moderation: the paper printed the ad; parties sorted risk in the real world. Modern marketplaces inverted that into report queues, classifiers, and opaque enforcement. Users learn that “Report” is slow, inconsistent, or captured by whoever can spam the button.

Parts Pile starts from a different thesis (also in the local-launch notes):

| Pain on FB / CL | Parts Pile angle |
|-----------------|-----------------|
| No shared trust signal | Flags mark disputed ads / actors; ads assumed good until flaged |
| Fake / throwaway reporters | Phone-gated accounts raise sockpuppet cost; flags themselves are scarce |
| Hidden numbers / deposit scams | Contact stays in-app; flags attach to the conversation |
| Flaky chat | Dedicated messaging + flag as a first-class dispute object |

Phone verification and flags are **complements**. Phone cost slows mass spam; flags make *individual* complaints expensive for the complainant and consequential for the listing. Neither invents liquidity. Flags are the retention and trust story once inventory exists—not a cold-start engine.

The brand metaphor is intentional: a flag is heavier than a flag emoji. You only get a few. Spending one should feel like staking reputation on a claim. The confirm modal exists so that feeling is explicit before the POST.

---

## 2. Philosophy

1. **Innocent until flaged.** Discovery does not require “verified seller” theater. Zero flags means fine for search. Distrust is earned by peers.

2. **Scarcity over infinite flags.** `MaxOutstandingFlags = 3`. You cannot carpet-bomb every ad you dislike.

3. **Dispute is a conversation, not a ticket.** A flag is bound to a buyer–seller thread. Resolution is working it out; the thrower lowers when done. The platform makes the dispute *visible and costly*, not judge of first resort.

4. **Binding depends on who throws.**  
   - **Inquirer throws** → **ad-bound** (`flag_raiser_id = inquirer_id`). Icons on the ad; counts toward `ads.flag_count` and search exclusion.  
   - **Owner throws** → **user-bound** (`flag_raiser_id = owner_id`). Icons on the inquirer’s profile. Does **not** bump the ad’s flag count.

5. **Publicity is the enforcement.** A flag is not a private DM to staff. Flag icons on the ad are a public signal; clicking them opens the **assessment** for anyone (including guests). The **message thread stays private** to the owner and inquirer (both logged in). Assessments do not disclose participant usernames—only Owner/Inquirer roles. Shame and search demotion do work a moderator queue would otherwise do.

6. **Intentional throw, optional AI.** Reasons are required. Assessment is optional before throw (preview) and lazy after throw (first opinion view). AI is provisional community guidance, not a Parts Pile ruling or legal advice.

7. **Operator reserve powers remain.** Terms still allow removal or account restriction for law or service protection. Flags are the *default* path for ordinary listing problems, not a waiver of emergency duties.

---

## 3. How flags work

### 3.1 Lifecycle

```text
Signup → conceptual allotment MaxOutstandingFlags (3)
                ↓
Participant opens / creates conversation on an active ad
                ↓
Throw control → confirm modal (GET …/flag/confirm)
                ↓
Confirm UI:
  • remaining flag icons (“You have N flags”)
  • caution (visibility, lower, return if other removed for abuse)
  • canned reason radios (ad- vs user-direction labels/examples)
  • optional Review assessment → stacked preview modal (ephemeral Grok)
                ↓
Throw (POST + reason) → journal "flag thrown" + reason body
                      → set flag_raiser_id / flag_raised_at
                      → if inquirer: SyncAdFlagCount
                      → SSE + flag event in conversation
                      → OOB close confirm/preview modals
                      → NO opinion generation
                ↓
Later: participant scales link or bystander flag icon
        → GetOrGenerate → store flag_opinions on cache miss
                ↓
Lower (DELETE, thrower only) → clear flag fields
                              → journal "flag lowern"
                              → Invalidate flag_opinions
                              → resync ad flag_count if ad-bound
```

Constraints:

| Rule | Behavior |
|------|----------|
| Participants only | Non-participants cannot throw |
| One flag per conversation | `ErrFlagAlreadyRaised` |
| Outstanding cap | `GetUserFlagCount` ≥ 3 → `ErrMaxFlagsReached` |
| Reason required | `policy` / `conduct` / `deal` only |
| Messaging open | Inactive/deleted ads or deleted counterparties block throw |
| No self-flag on own ad | Ad throw path forbids flaging your own listing |
| Lower authority | Only `flag_raiser_id` |

Force-lower paths clear flags on ad pause/delete and account delete so public counts do not dangle forever. See §5.10 for a sync gap on some force paths.

### 3.2 Reasons

Journal body stores a short code. UI shows the **direction-specific label** from the confirm modal (and on flag event bubbles).

| Code | At ad (inquirer → listing) | At user (owner → inquirer) |
|------|----------------------------|----------------------------|
| `policy` | Listing or content violates policies | Scam, spam, or prohibited requests |
| `conduct` | Seller harassment or bad-faith conduct | Harassment or bad-faith conduct |
| `deal` | Deal or transaction went wrong | Deal or meetup went wrong |

Policy radios link to `/terms#prohibited`. Examples under each radio are illustrative, not a second policy document. Conduct and deal cover harassment and transaction failures that Terms also prohibit but that are not “listing content” in the narrow sense.

### 3.3 Data model

**`conversations`**

- `flag_raiser_id` NULL = no active flag; NOT NULL = flaged. Public ad UX: `/ad/:id/flag/:ord` → anonymized assessment. Message thread: participants only (`CanViewConversation` / `GetConversation`).  
- Binding: owner throw → bound to inquirer; inquirer throw → bound to ad.  
- `flag_raised_at` timestamps the active flag.  
- Encrypted `journal` records throw/lower alongside messages; throw body = reason code.

**`ads`**

- `flag_count` = denormalized count of **ad-bound** flags only.

**`flag_opinions`**

- Cached LLM output keyed by `conversation_id`: summary, assessment 1–10, detail, resolution, reasoning.  
- Filled on first successful `GetOrGenerate`, not on throw.  
- Invalidated on lower, new message in that conversation, ad edit, image confirm, and ad-level cleanup.

### 3.4 Search and discovery

Metadata / vector search keeps ads with `flag_count <= MaxFlagCount` (`MaxFlagCount = 2`). An ad stays discoverable with 0–2 ad-bound flags and leaves search listings at **3+**. Icons communicate trouble before exclusion; exclusion is the hard consequence.

Embeddings include natural-language flag context so similarity search is not blind to dispute density. Throw/lower updates the live column used by the filter; embedding text can lag until the next re-index.

### 3.5 UX surfaces

| Surface | Behavior |
|---------|----------|
| **Confirm modal** | Remaining icons; caution; reason radios; Terms link on policy; Review + Throw enabled after reason |
| **Preview modal** | Stacked above confirm; provisional banner; tilted scale + summary; detail in `<details>`; does not block throw if unavailable |
| **Throw** | POST only; closes confirm (and preview) via OOB deletes; appends flag event |
| **Flag event bubble** | Label + full reason text; hint to open assessment when a flag is active |
| **Opinion link** | Scales control in the conversation; spins while HTMX loads |
| **Opinion modal** | Cached or freshly generated assessment; provisional footer |
| **Flag icons** | Ad ordinals → `/auth/ad/:id/flag/:ord`; user ordinals → `/auth/user/:id/flag/:ord` |
| **Bystander vs participant** | Ad flag icons → public `/ad/:id/flag/:ord` → **opinion modal** (no party usernames). Profile flag icons under `/auth/…`. Only owner and inquirer (logged in) can open the conversation thread. Direct `/auth/conversation/:id` is denied to everyone else |
| **Opinion modal** | Summary, ad facts, Owner↔Inquirer scale, resolution, reasoning; parties shown only as Owner and Inquirer |
| **Lower** | Direct control for the thrower in conversation actions |
| **Chrome badge** | Outstanding **thrown** count (hidden at zero)—not remaining allotment |
| **FAQ / Terms** | `/faq/flags`; Terms “Reporting and removal”; Privacy “Content reports” |

### 3.6 Dispute assessment (Grok)

**When the model runs**

| Path | Function | Stored? | Needs active flag? |
|------|----------|---------|--------------------|
| Review assessment (pre-throw) | `Preview` | No | No |
| Scales / flag-opinion / bystander icon | `GetOrGenerate` | Yes on miss | Yes |
| Throw | — | — | Never generates |

**Pipeline (preview and generate share `generate`)**

1. Load ad, tags, formal facets, journal messages, reason (selected or latest thrown).  
2. Redact phones, emails, street-like patterns, and participant display names from message text.  
3. Optionally attach listing images (see below).  
4. Call Grok as a neutral Owner/Inquirer arbitrator; JSON only.  
5. Parse; refuse if output still matches PII patterns (`ErrUnavailable`).  
6. Store only on `GetOrGenerate` miss.

**Image attach** (`shouldAttachImages`): only if the image store is configured and the ad has images, **and** either (a) messages mention image-ish words, or (b) reason is `policy` and there are **no** messages yet. Otherwise text-only. When attaching: indices `1..min(imageCount, MaxImagesPerAd)` at `480w`, base64 data-URI, `detail: "low"`. Cap is **20** (same as ad upload max)—not a separate opinion quota.

**Scale:** **1** = inquirer clearly in the right … **5** = balanced … **10** = owner clearly in the right. UI tilts a beam from the score. Copy: provisional community guidance, not a Parts Pile ruling.

**Invalidation → regenerate on next view** keeps opinions aligned with new messages, edits, and images—at the cost of possible score flips and repeat model spend.

---

## 4. What flags are not

- **Not KYC.** A flag does not prove who is right; phone OTP raised human cost at signup.  
- **Not a staffed trust & safety queue.** Default path is peer dispute; operator intervention is reserved.  
- **Not a court, insurance, or escrow.**  
- **Not a complete CSAM / illegal-goods pipeline.** Those require operator/legal removal regardless of flag count.  
- **Not a substitute for liquidity.** Launch docs warn against expecting flags + phone-only to invent supply.  
- **Not automatic AI judgment on every throw.** Assessment is opt-in preview or on-demand view.

---

## 5. Flaws and design tensions

### 5.1 Scarcity cuts both ways

Three outstanding flags limit flag spam. A good-faith user who hits four bad listings must triage—or wait for lowers. Coordinated actors can burn a victim’s budget with gray disputes.

### 5.2 Asymmetric binding

Buyers can hurt **listings** (search). Sellers can hurt **buyer profiles** without moving `ads.flag_count`. Useful (“warn others about this inquirer”) and abusable (smear while the listing stays searchable).

### 5.3 Publicity vs privacy

The **assessment** is public (and anonymized). The **chat transcript is not**. Only the ad owner and the inquirer, while logged in, can open the conversation. Guests and other members who click a flag icon see Owner/Inquirer paraphrase and scores—not meetup details, insults, or raw messages. Confirm-modal caution should stay aligned with that split (visible flag + assessment, not “entire transcript”).

Residual risk: opinion text can still leak what parties said if the model paraphrases too closely; redaction and prompt rules mitigate but do not eliminate that.

### 5.4 Thrower-only lower

Only the thrower reclaim. There is no accused-party unilateral return. FAQ/Terms match the code.

### 5.5 Soft threshold before hard exclusion

Two flags still leave an ad in search. Exclusion at `> MaxFlagCount` needs enough distinct inquirers to leave flags outstanding.

### 5.6 Assessment timing (intentional)

Throw does not call Grok. The assessment is generated the **first time someone opens it**; the UI shows a loading indicator while that runs.

**Review assessment** before throw is an ephemeral preview and is not stored. A later stored assessment can differ—messages may have changed, image attach rules may differ, or the model may answer differently. That is expected; the preview cannot know the future.

Each new message in a flaged thread invalidates the cached assessment so the next view regenerates from current evidence. Scores can change during an active dispute; that is preferred over a stale snapshot.

### 5.7 Conditional images

Policy cases with prior chat that never mentions photos may stay text-only even when listing images are the evidence. Conversely, a single “photo” mention attaches up to every listing image (≤20).

### 5.8 AI and redaction brittleness

Grok leaves the trust boundary. Opinions can be wrong or steered by theater prose. Regex redaction is best-effort. Cached bad opinions stick until invalidate. Opinion rows are plaintext in Postgres and backups.

### 5.9 Journal and backup footprint

Flag events live in encrypted journals; opinions are durable paraphrase. Account delete force-lowers; historical lines and prior opinion text may already have been seen or backed up.

### 5.10 Force-lower / count sync

Some lifecycle force-lowers do not always `SyncAdFlagCount` the same way as a manual lower. Stale high `ads.flag_count` (search over-exclusion) is a concrete engineering risk when accounts that threw on others’ ads are deleted. Orphan `flag_opinions` rows are possible if invalidation is skipped on a force path.

### 5.11 Concurrency

Outstanding-cap and one-flag-per-conversation checks are not a single transactional lock with the UPDATE (TOCTOU under concurrent throws).

### 5.12 Cold-start and capture

In a thin market, a few networked accounts can dominate flag outcomes. Phone friction raises cost; it does not equalize power.

---

## 6. Abuse and exploit narratives

Illustrative stories grounded in the running system—not claims of active CVEs.

### A — Brigading a competitor

Three phone-gated accounts open conversations and throw ad-bound flags. Target exits search once `flag_count` exceeds 2. Cost: three SIMs and three committed flags. Residual: real humans can still attack a niche listing.

### B — Flag-budget exhaustion

Gray disputes keep a target’s allotment stuck. Target cannot flag a clear scam. Scarcity becomes a DoS on the reporting tool.

### C — Seller smear without search impact

Owner flags inquisitive buyers. Profiles show user-bound flags; owner ads stay at `flag_count = 0`. Opinion scores may or may not correct the narrative.

### D — Opinion as disclosure theater

Parties chat privately; one throws after collecting sensitive details. Bystanders cannot open the transcript, but a poorly paraphrased assessment might still surface sensitive themes. Mitigations: Owner/Inquirer-only prompts, redaction, and refuse-to-store on PII patterns.

### E — Opinion theater

A party writes for the model. First `GetOrGenerate` caches the story for every flag-icon click until invalidate.

### F — Ordinal / ID enumeration

Authenticated or anonymous probing of flag ordinals harvests assessment text and ad titles—not the transcript. Conversation IDs no longer unlock the journal for non-participants.

### G — Stale flags after off-platform settle

Parties resolve elsewhere; thrower forgets or refuses to lower. Ad stays suppressed; flag stays locked.

### H — Account takeover

SIM-swap or password stuffing yields throw/lower power and journal participation. Flags inherit account-security posture (see phone-identity paper).

---

## 7. Legal issues

Internal analysis, **not legal advice**.

### 7.1 Self-policing vs platform duties

Terms: self-policed with flags; do not email for ordinary listing/copyright/content issues—use flags; operator may still remove or restrict.

Tension: some regimes expect designated intake for illegal content, copyright (e.g. DMCA), or child safety. Flags should stay the **preferred UX for ordinary disputes**, not the exclusive lawful intake where statute demands operator action. The reserve-power sentence is load-bearing.

### 7.2 Intermediary liability

Peer flags and search demotion are user-driven. **Platform-authored opinions** are closer to platform speech/tools. Keep them labeled provisional and non-binding. Counsel owns §230-style analysis.

### 7.3 Defamation and reputation

Public icons and user-bound flags are reputation statements. Mitigations: scarcity, phone cost, visible context, provisional AI framing, lower, operator bans for abuse.

### 7.4 Privacy and publicity

Publishing a flag assessment on the listing is a material change vs private messaging. Throw-time copy should stay honest: anyone can see that a flag was thrown and can open the assessment. Assessments must not name the parties.

### 7.5 Intellectual property

A flag is not a DMCA-compliant notice. If Terms funnel copyright into flags alone, confirm a parallel statutory intake or carve IP out of “don’t email.”

### 7.6 Illegal and high-risk content

Flags are too slow and peer-dependent for CSAM, imminent threats, and clear illegal goods. Operator tools and mandatory reporting must bypass flag thresholds.

### 7.7 “Assumed good”

Accurate as product default; not a safety guarantee. Keep copy descriptive (“peers marked a dispute”), not certifying (“verified safe”).

### 7.8 AI assessments

Accuracy, bias, transparency, cross-border model transfer, and retention after parties wanted the fight forgotten all remain. Invalidation helps; backups and screenshots do not forget.

---

## 8. Risk matrix (condensed)

| Threat / issue | Primary impact | Severity if unmitigated | Notes |
|----------------|----------------|-------------------------|--------|
| Brigading with phone-gated socks | Search exclusion | Medium–High | Costly but feasible in niches |
| Flag-budget exhaustion | Cannot report real scams | Medium | Scarcity DoS |
| Owner smear (user-bound) | Buyer reputation | Medium | No ad `flag_count` change |
| Publicity of private journal | Privacy / harassment | Low (thread) / Medium (opinion paraphrase) | Thread participants-only; assessment public |
| Opinion injection / wrong AI | Misleading bystanders | Medium | Cache amplifies; preview optional |
| Flags as sole CSAM/IP intake | Legal exposure | Critical if exclusive | Keep operator path |
| Account takeover → flag abuse | Trust-signal integrity | High | Inherited from auth |
| Force-lower count drift | Wrong search exclusion | Medium | Engineering hygiene |
| Thin-market capture | Local narrative monopoly | Medium | Cold start |

---

## 9. Recommendations

Status relative to the current codebase:

1. ~~**Thrower-only lower in copy.**~~ Done (FAQ/Terms).  
2. ~~**Throw-time caution + reasons + optional preview.**~~ Done (confirm + stacked preview).  
3. ~~**No assessment on throw.**~~ Done (on-demand `GetOrGenerate`).  
4. ~~**Direction-specific reason labels; Terms link on policy.**~~ Done.  
5. ~~**Conditional image attach for opinions.**~~ Done (mention or policy+empty thread; ≤ `MaxImagesPerAd`).  
6. ~~**Clarify bystander surface.**~~ **Done.** Assessment public (anonymized); message thread participants-only.  
7. **Carve statutory intakes.** Publish how CSAM, imminent harm, and copyright notices reach operators without flag scarcity. Soften “don’t email… copyright” if no compliant alternate exists.  
8. **Brigading signals for operators.** Multi-account flags on one ad in a short window—signal, not silent auto-pardon.  
9. **Preserve scarcity.** Resist unlimited flags; if raising caps, raise friction in tandem.  
10. **Force-lower hygiene.** Always `SyncAdFlagCount` and invalidate opinions on every clear path; document backup retention for `flag_opinions`.  
11. **Keep innocent-until-flaged.** Avoid pay-to-verify theater unless a separate honest product line needs it.

---

## 10. Conclusion

Flags answer a question most marketplaces dodge: **can peers enforce trust with a scarce, visible, conversation-bound signal instead of an infinite report button and a hidden queue?** The current implementation is coherent: confirm-before-throw, canned reasons, thrower-only lower, binding by thrower role, search exclusion past two ad-bound flags, and Grok assessment that is optional before throw and lazy after.

The philosophy—assumed good, scarcity, dispute-as-conversation, publicity-as-enforcement, AI as paraphrase not court—matches phone-minimal identity and the classifieds metaphor. The same philosophy creates the flaws that matter: gaming under scarcity, publicity of chats, asymmetric seller/buyer weapons, AI fallibility, and legal categories that cannot wait for three peers to spend flags.

The correct reading is not “flags replace trust & safety,” but **“flags replace *casual* flag spam with costly peer disputes, while the operator retains narrow duties for law and platform integrity.”** That bet pays off only if copy matches code, throwers understand publicity, statutory intakes stay open, and flag abuse is treated as seriously as listing fraud itself.

---

## Appendix A — Primary code map

| Area | Location |
|------|----------|
| Throw / lower / counts / ordinals | `internal/flag/flag.go` |
| Reason codes and labels | `internal/flag/reason.go` |
| Confirm / preview / throw HTTP | `internal/handler/flag.go` |
| Opinion HTTP / message invalidate | `internal/handler/message.go` |
| Ad / user flag ordinal entry | `internal/handler/ad.go` (`PublicAdFlagOpinionHandler`), `internal/handler/user.go` |
| Routes | `cmd/server/main.go` |
| Conversation access | `internal/message/view.go` (`CanViewConversation` = participants only) |
| Journal flag events | `internal/journal/journal.go`, `internal/message/journal.go` |
| Opinions (generate, cache, images, redact) | `internal/flagopinion/` |
| UI | `internal/ui/flag.go`, `flag_confirm.go`, `flag_opinion.go`, `message.go` |
| Scales spin CSS | `input.css` (`.flag-opinion-link` / `.flag-raise-review-btn` + `htmx-request`) |
| Search filter | `internal/search/metadata.go` |
| Embedding flag text | `internal/vector/ad.go` (`flagContext`) |
| Config | `internal/config/config.go` |
| Schema | `internal/db/schema.sql` |
| FAQ / Terms / Privacy | `internal/ui/faq.go`, `internal/ui/legal.go` |

## Appendix B — Related public claims

**FAQ `/faq/flags`:** N flags at join; throw on policy/problem ads; starts a conversation; excluded when more than `MaxFlagCount` flags; choose a reason; optional provisional assessment; thrower lowers; use wisely.  

**Terms “Reporting and removal”:** Self-policed with flags; visible; search exclusion; reason + optional assessment; thrower lowers; do not email for listing/copyright/content issues—use flags; operator may still act. Prohibited content lives under `/terms#prohibited`.  

**Privacy “Content reports”:** Listing problems handled in-product via flags.  

**Local launch doc:** Flags mark bad actors; ads assumed good until flaged; do not expect flags to substitute for supply.

## Appendix C — Parameter cheat sheet

| Constant | Value (Aug 2026) | Meaning |
|----------|------------------|---------|
| `MaxOutstandingFlags` | 3 | Max flags a user may have thrown at once |
| `MaxFlagCount` | 2 | Max `ads.flag_count` still in search (`<= 2`); excluded at 3+ |
| `MaxImagesPerAd` | 20 | Listing image cap; also upper bound when opinions attach images |
| Reason codes | `policy`, `conduct`, `deal` | Journal body; labels differ by throw direction |
| Assessment scale | 1–10 | 1 inquirer right … 5 balanced … 10 owner right |
