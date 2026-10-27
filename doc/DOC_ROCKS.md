# Rocks at Parts Pile

## Scarce Peer Disputes, Visible Consequences, Provisional Assessment

**Document type:** Product design, threat, and legal analysis  
**Audience:** Product, engineering, trust & safety, and legal reviewers  
**Scope:** How Parts Pile rocks work end-to-end after the throw-confirm UX; the philosophy behind scarcity-based, peer-visible disputes; dispute assessment (optional preview, on-demand store); known flaws; abuse paths; and legal tensions of self-policing with public rock signals and AI paraphrase.  
**Based on:** Parts Pile site implementation (Go / Fiber / Postgres / Grok rock opinions / encrypted conversation journals), August 2026.  
**Companions:** [DOC_PHONE_ONLY_IDENTITY_AND_SECURITY.md](DOC_PHONE_ONLY_IDENTITY_AND_SECURITY.md) (identity / SMS); [DOC_LOCAL_LAUNCH_CAR_TRUCK_PARTS.md](DOC_LOCAL_LAUNCH_CAR_TRUCK_PARTS.md) (go-to-market: rocks as trust signal, not a substitute for liquidity).

---

## Abstract

Parts Pile treats classifieds trust as a **scarce, peer-enforced signal**, not a staffed ticket queue. Every member may keep at most **three outstanding rocks**. Throwing one on a conversation marks a dispute, journals a canned reason, and increments a visible rock count on either the **ad** (inquirer threw) or the **inquirer’s profile** (owner threw). Rock icons on ads (and the **dispute assessment** behind them) are **public**—no login required. The assessment names parties only as **Owner** and **Inquirer**, never by username. Enough **ad-bound** rocks remove the listing from search.

The throw path is deliberate, not a one-click flag: a confirm modal shows remaining rocks, direction-specific caution, required reason radios (with examples; policy links to Terms), and an optional **Review assessment** that opens a stacked preview modal. Throw itself is fast—it does **not** call Grok. A stored dispute assessment is generated **on demand** when someone opens the opinion from the conversation or a rock icon. Pre-throw review is ephemeral and never written to `rock_opinions`.

The design is anti-bureaucratic. Terms and FAQ push ordinary listing problems into the product: throw, talk it out, unthrow when resolved. Ads are **assumed good until rocked**. That bet buys triage (“who cares enough to spend a rock?”) and a brand metaphor that matches the product. It also creates sharp edges: scarcity can be gamed or exhausted; publicity of chats conflicts with private-DM expectations; AI opinions leave the trust boundary; and self-policing is incomplete for statutory takedown and CSAM. This paper maps the current mechanism, philosophy, failure modes, and legal posture without pretending rocks are a complete trust system.

---

## 1. Motivation: why rocks exist

Classic newspaper classifieds had almost no platform moderation: the paper printed the ad; parties sorted risk in the real world. Modern marketplaces inverted that into report queues, classifiers, and opaque enforcement. Users learn that “Report” is slow, inconsistent, or captured by whoever can spam the button.

Parts Pile starts from a different thesis (also in the local-launch notes):

| Pain on FB / CL | Parts Pile angle |
|-----------------|-----------------|
| No shared trust signal | Rocks mark disputed ads / actors; ads assumed good until rocked |
| Fake / throwaway reporters | Phone-gated accounts raise sockpuppet cost; rocks themselves are scarce |
| Hidden numbers / deposit scams | Contact stays in-app; rocks attach to the conversation |
| Flaky chat | Dedicated messaging + rock as a first-class dispute object |

Phone verification and rocks are **complements**. Phone cost slows mass spam; rocks make *individual* complaints expensive for the complainant and consequential for the listing. Neither invents liquidity. Rocks are the retention and trust story once inventory exists—not a cold-start engine.

The brand metaphor is intentional: a rock is heavier than a flag emoji. You only get a few. Spending one should feel like staking reputation on a claim. The confirm modal exists so that feeling is explicit before the POST.

---

## 2. Philosophy

1. **Innocent until rocked.** Discovery does not require “verified seller” theater. Zero rocks means fine for search. Distrust is earned by peers.

2. **Scarcity over infinite flags.** `MaxOutstandingRocks = 3`. You cannot carpet-bomb every ad you dislike.

3. **Dispute is a conversation, not a ticket.** A rock is bound to a buyer–seller thread. Resolution is working it out; the thrower unthrows when done. The platform makes the dispute *visible and costly*, not judge of first resort.

4. **Binding depends on who throws.**  
   - **Inquirer throws** → **ad-bound** (`rock_thrower_id = inquirer_id`). Icons on the ad; counts toward `ads.rock_count` and search exclusion.  
   - **Owner throws** → **user-bound** (`rock_thrower_id = owner_id`). Icons on the inquirer’s profile. Does **not** bump the ad’s rock count.

5. **Publicity is the enforcement.** A rock is not a private DM to staff. Rock icons on the ad are a public signal; clicking them opens the **assessment** for anyone (including guests). The **message thread stays private** to the owner and inquirer (both logged in). Assessments do not disclose participant usernames—only Owner/Inquirer roles. Shame and search demotion do work a moderator queue would otherwise do.

6. **Intentional throw, optional AI.** Reasons are required. Assessment is optional before throw (preview) and lazy after throw (first opinion view). AI is provisional community guidance, not a Parts Pile ruling or legal advice.

7. **Operator reserve powers remain.** Terms still allow removal or account restriction for law or service protection. Rocks are the *default* path for ordinary listing problems, not a waiver of emergency duties.

---

## 3. How rocks work

### 3.1 Lifecycle

```text
Signup → conceptual allotment MaxOutstandingRocks (3)
                ↓
Participant opens / creates conversation on an active ad
                ↓
Throw control → confirm modal (GET …/rock/confirm)
                ↓
Confirm UI:
  • remaining rock icons (“You have N rocks”)
  • caution (visibility, unthrow, return if other removed for abuse)
  • canned reason radios (ad- vs user-direction labels/examples)
  • optional Review assessment → stacked preview modal (ephemeral Grok)
                ↓
Throw (POST + reason) → journal "rock thrown" + reason body
                      → set rock_thrower_id / rock_thrown_at
                      → if inquirer: SyncAdRockCount
                      → SSE + rock event in conversation
                      → OOB close confirm/preview modals
                      → NO opinion generation
                ↓
Later: participant scales link or bystander rock icon
        → GetOrGenerate → store rock_opinions on cache miss
                ↓
Unthrow (DELETE, thrower only) → clear rock fields
                              → journal "rock unthrown"
                              → Invalidate rock_opinions
                              → resync ad rock_count if ad-bound
```

Constraints:

| Rule | Behavior |
|------|----------|
| Participants only | Non-participants cannot throw |
| One rock per conversation | `ErrRockAlreadyThrown` |
| Outstanding cap | `GetUserRockCount` ≥ 3 → `ErrMaxRocksReached` |
| Reason required | `policy` / `conduct` / `deal` only |
| Messaging open | Inactive/deleted ads or deleted counterparties block throw |
| No self-rock on own ad | Ad throw path forbids rocking your own listing |
| Unthrow authority | Only `rock_thrower_id` |

Force-unthrow paths clear rocks on ad pause/delete and account delete so public counts do not dangle forever. See §5.10 for a sync gap on some force paths.

### 3.2 Reasons

Journal body stores a short code. UI shows the **direction-specific label** from the confirm modal (and on rock event bubbles).

| Code | At ad (inquirer → listing) | At user (owner → inquirer) |
|------|----------------------------|----------------------------|
| `policy` | Listing or content violates policies | Scam, spam, or prohibited requests |
| `conduct` | Seller harassment or bad-faith conduct | Harassment or bad-faith conduct |
| `deal` | Deal or transaction went wrong | Deal or meetup went wrong |

Policy radios link to `/terms#prohibited`. Examples under each radio are illustrative, not a second policy document. Conduct and deal cover harassment and transaction failures that Terms also prohibit but that are not “listing content” in the narrow sense.

### 3.3 Data model

**`conversations`**

- `rock_thrower_id` NULL = no active rock; NOT NULL = rocked. Public ad UX: `/ad/:id/rock/:ord` → anonymized assessment. Message thread: participants only (`CanViewConversation` / `GetConversation`).  
- Binding: owner throw → bound to inquirer; inquirer throw → bound to ad.  
- `rock_thrown_at` timestamps the active rock.  
- Encrypted `journal` records throw/unthrow alongside messages; throw body = reason code.

**`ads`**

- `rock_count` = denormalized count of **ad-bound** rocks only.

**`rock_opinions`**

- Cached LLM output keyed by `conversation_id`: summary, assessment 1–10, detail, resolution, reasoning.  
- Filled on first successful `GetOrGenerate`, not on throw.  
- Invalidated on unthrow, new message in that conversation, ad edit, image confirm, and ad-level cleanup.

### 3.4 Search and discovery

Metadata / vector search keeps ads with `rock_count <= MaxRockCount` (`MaxRockCount = 2`). An ad stays discoverable with 0–2 ad-bound rocks and leaves search listings at **3+**. Icons communicate trouble before exclusion; exclusion is the hard consequence.

Embeddings include natural-language rock context so similarity search is not blind to dispute density. Throw/unthrow updates the live column used by the filter; embedding text can lag until the next re-index.

### 3.5 UX surfaces

| Surface | Behavior |
|---------|----------|
| **Confirm modal** | Remaining icons; caution; reason radios; Terms link on policy; Review + Throw enabled after reason |
| **Preview modal** | Stacked above confirm; provisional banner; tilted scale + summary; detail in `<details>`; does not block throw if unavailable |
| **Throw** | POST only; closes confirm (and preview) via OOB deletes; appends rock event |
| **Rock event bubble** | Label + full reason text; hint to open assessment when a rock is active |
| **Opinion link** | Scales control in the conversation; spins while HTMX loads |
| **Opinion modal** | Cached or freshly generated assessment; provisional footer |
| **Rock icons** | Ad ordinals → `/auth/ad/:id/rock/:ord`; user ordinals → `/auth/user/:id/rock/:ord` |
| **Bystander vs participant** | Ad rock icons → public `/ad/:id/rock/:ord` → **opinion modal** (no party usernames). Profile rock icons under `/auth/…`. Only owner and inquirer (logged in) can open the conversation thread. Direct `/auth/conversation/:id` is denied to everyone else |
| **Opinion modal** | Summary, ad facts, Owner↔Inquirer scale, resolution, reasoning; parties shown only as Owner and Inquirer |
| **Unthrow** | Direct control for the thrower in conversation actions |
| **Chrome badge** | Outstanding **thrown** count (hidden at zero)—not remaining allotment |
| **FAQ / Terms** | `/faq/rocks`; Terms “Reporting and removal”; Privacy “Content reports” |

### 3.6 Dispute assessment (Grok)

**When the model runs**

| Path | Function | Stored? | Needs active rock? |
|------|----------|---------|--------------------|
| Review assessment (pre-throw) | `Preview` | No | No |
| Scales / rock-opinion / bystander icon | `GetOrGenerate` | Yes on miss | Yes |
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

## 4. What rocks are not

- **Not KYC.** A rock does not prove who is right; phone OTP raised human cost at signup.  
- **Not a staffed trust & safety queue.** Default path is peer dispute; operator intervention is reserved.  
- **Not a court, insurance, or escrow.**  
- **Not a complete CSAM / illegal-goods pipeline.** Those require operator/legal removal regardless of rock count.  
- **Not a substitute for liquidity.** Launch docs warn against expecting rocks + phone-only to invent supply.  
- **Not automatic AI judgment on every throw.** Assessment is opt-in preview or on-demand view.

---

## 5. Flaws and design tensions

### 5.1 Scarcity cuts both ways

Three outstanding rocks limit flag spam. A good-faith user who hits four bad listings must triage—or wait for unthrows. Coordinated actors can burn a victim’s budget with gray disputes.

### 5.2 Asymmetric binding

Buyers can hurt **listings** (search). Sellers can hurt **buyer profiles** without moving `ads.rock_count`. Useful (“warn others about this inquirer”) and abusable (smear while the listing stays searchable).

### 5.3 Publicity vs privacy

The **assessment** is public (and anonymized). The **chat transcript is not**. Only the ad owner and the inquirer, while logged in, can open the conversation. Guests and other members who click a rock icon see Owner/Inquirer paraphrase and scores—not meetup details, insults, or raw messages. Confirm-modal caution should stay aligned with that split (visible rock + assessment, not “entire transcript”).

Residual risk: opinion text can still leak what parties said if the model paraphrases too closely; redaction and prompt rules mitigate but do not eliminate that.

### 5.4 Thrower-only unthrow

Only the thrower reclaim. There is no accused-party unilateral return. FAQ/Terms match the code.

### 5.5 Soft threshold before hard exclusion

Two rocks still leave an ad in search. Exclusion at `> MaxRockCount` needs enough distinct inquirers to leave rocks outstanding.

### 5.6 Assessment timing (intentional)

Throw does not call Grok. The assessment is generated the **first time someone opens it**; the UI shows a loading indicator while that runs.

**Review assessment** before throw is an ephemeral preview and is not stored. A later stored assessment can differ—messages may have changed, image attach rules may differ, or the model may answer differently. That is expected; the preview cannot know the future.

Each new message in a rocked thread invalidates the cached assessment so the next view regenerates from current evidence. Scores can change during an active dispute; that is preferred over a stale snapshot.

### 5.7 Conditional images

Policy cases with prior chat that never mentions photos may stay text-only even when listing images are the evidence. Conversely, a single “photo” mention attaches up to every listing image (≤20).

### 5.8 AI and redaction brittleness

Grok leaves the trust boundary. Opinions can be wrong or steered by theater prose. Regex redaction is best-effort. Cached bad opinions stick until invalidate. Opinion rows are plaintext in Postgres and backups.

### 5.9 Journal and backup footprint

Rock events live in encrypted journals; opinions are durable paraphrase. Account delete force-unthrows; historical lines and prior opinion text may already have been seen or backed up.

### 5.10 Force-unthrow / count sync

Some lifecycle force-unthrows do not always `SyncAdRockCount` the same way as a manual unthrow. Stale high `ads.rock_count` (search over-exclusion) is a concrete engineering risk when accounts that threw on others’ ads are deleted. Orphan `rock_opinions` rows are possible if invalidation is skipped on a force path.

### 5.11 Concurrency

Outstanding-cap and one-rock-per-conversation checks are not a single transactional lock with the UPDATE (TOCTOU under concurrent throws).

### 5.12 Cold-start and capture

In a thin market, a few networked accounts can dominate rock outcomes. Phone friction raises cost; it does not equalize power.

---

## 6. Abuse and exploit narratives

Illustrative stories grounded in the running system—not claims of active CVEs.

### A — Brigading a competitor

Three phone-gated accounts open conversations and throw ad-bound rocks. Target exits search once `rock_count` exceeds 2. Cost: three SIMs and three committed rocks. Residual: real humans can still attack a niche listing.

### B — Rock-budget exhaustion

Gray disputes keep a target’s allotment stuck. Target cannot rock a clear scam. Scarcity becomes a DoS on the reporting tool.

### C — Seller smear without search impact

Owner rocks inquisitive buyers. Profiles show user-bound rocks; owner ads stay at `rock_count = 0`. Opinion scores may or may not correct the narrative.

### D — Opinion as disclosure theater

Parties chat privately; one throws after collecting sensitive details. Bystanders cannot open the transcript, but a poorly paraphrased assessment might still surface sensitive themes. Mitigations: Owner/Inquirer-only prompts, redaction, and refuse-to-store on PII patterns.

### E — Opinion theater

A party writes for the model. First `GetOrGenerate` caches the story for every rock-icon click until invalidate.

### F — Ordinal / ID enumeration

Authenticated or anonymous probing of rock ordinals harvests assessment text and ad titles—not the transcript. Conversation IDs no longer unlock the journal for non-participants.

### G — Stale rocks after off-platform settle

Parties resolve elsewhere; thrower forgets or refuses to unthrow. Ad stays suppressed; rock stays locked.

### H — Account takeover

SIM-swap or password stuffing yields throw/unthrow power and journal participation. Rocks inherit account-security posture (see phone-identity paper).

---

## 7. Legal issues

Internal analysis, **not legal advice**.

### 7.1 Self-policing vs platform duties

Terms: self-policed with rocks; do not email for ordinary listing/copyright/content issues—use rocks; operator may still remove or restrict.

Tension: some regimes expect designated intake for illegal content, copyright (e.g. DMCA), or child safety. Rocks should stay the **preferred UX for ordinary disputes**, not the exclusive lawful intake where statute demands operator action. The reserve-power sentence is load-bearing.

### 7.2 Intermediary liability

Peer rocks and search demotion are user-driven. **Platform-authored opinions** are closer to platform speech/tools. Keep them labeled provisional and non-binding. Counsel owns §230-style analysis.

### 7.3 Defamation and reputation

Public icons and user-bound rocks are reputation statements. Mitigations: scarcity, phone cost, visible context, provisional AI framing, unthrow, operator bans for abuse.

### 7.4 Privacy and publicity

Publishing a rock assessment on the listing is a material change vs private messaging. Throw-time copy should stay honest: anyone can see that a rock was thrown and can open the assessment. Assessments must not name the parties.

### 7.5 Intellectual property

A rock is not a DMCA-compliant notice. If Terms funnel copyright into rocks alone, confirm a parallel statutory intake or carve IP out of “don’t email.”

### 7.6 Illegal and high-risk content

Rocks are too slow and peer-dependent for CSAM, imminent threats, and clear illegal goods. Operator tools and mandatory reporting must bypass rock thresholds.

### 7.7 “Assumed good”

Accurate as product default; not a safety guarantee. Keep copy descriptive (“peers marked a dispute”), not certifying (“verified safe”).

### 7.8 AI assessments

Accuracy, bias, transparency, cross-border model transfer, and retention after parties wanted the fight forgotten all remain. Invalidation helps; backups and screenshots do not forget.

---

## 8. Risk matrix (condensed)

| Threat / issue | Primary impact | Severity if unmitigated | Notes |
|----------------|----------------|-------------------------|--------|
| Brigading with phone-gated socks | Search exclusion | Medium–High | Costly but feasible in niches |
| Rock-budget exhaustion | Cannot report real scams | Medium | Scarcity DoS |
| Owner smear (user-bound) | Buyer reputation | Medium | No ad `rock_count` change |
| Publicity of private journal | Privacy / harassment | Low (thread) / Medium (opinion paraphrase) | Thread participants-only; assessment public |
| Opinion injection / wrong AI | Misleading bystanders | Medium | Cache amplifies; preview optional |
| Rocks as sole CSAM/IP intake | Legal exposure | Critical if exclusive | Keep operator path |
| Account takeover → rock abuse | Trust-signal integrity | High | Inherited from auth |
| Force-unthrow count drift | Wrong search exclusion | Medium | Engineering hygiene |
| Thin-market capture | Local narrative monopoly | Medium | Cold start |

---

## 9. Recommendations

Status relative to the current codebase:

1. ~~**Thrower-only unthrow in copy.**~~ Done (FAQ/Terms).  
2. ~~**Throw-time caution + reasons + optional preview.**~~ Done (confirm + stacked preview).  
3. ~~**No assessment on throw.**~~ Done (on-demand `GetOrGenerate`).  
4. ~~**Direction-specific reason labels; Terms link on policy.**~~ Done.  
5. ~~**Conditional image attach for opinions.**~~ Done (mention or policy+empty thread; ≤ `MaxImagesPerAd`).  
6. ~~**Clarify bystander surface.**~~ **Done.** Assessment public (anonymized); message thread participants-only.  
7. **Carve statutory intakes.** Publish how CSAM, imminent harm, and copyright notices reach operators without rock scarcity. Soften “don’t email… copyright” if no compliant alternate exists.  
8. **Brigading signals for operators.** Multi-account rocks on one ad in a short window—signal, not silent auto-pardon.  
9. **Preserve scarcity.** Resist unlimited rocks; if raising caps, raise friction in tandem.  
10. **Force-unthrow hygiene.** Always `SyncAdRockCount` and invalidate opinions on every clear path; document backup retention for `rock_opinions`.  
11. **Keep innocent-until-rocked.** Avoid pay-to-verify theater unless a separate honest product line needs it.

---

## 10. Conclusion

Rocks answer a question most marketplaces dodge: **can peers enforce trust with a scarce, visible, conversation-bound signal instead of an infinite report button and a hidden queue?** The current implementation is coherent: confirm-before-throw, canned reasons, thrower-only unthrow, binding by thrower role, search exclusion past two ad-bound rocks, and Grok assessment that is optional before throw and lazy after.

The philosophy—assumed good, scarcity, dispute-as-conversation, publicity-as-enforcement, AI as paraphrase not court—matches phone-minimal identity and the classifieds metaphor. The same philosophy creates the flaws that matter: gaming under scarcity, publicity of chats, asymmetric seller/buyer weapons, AI fallibility, and legal categories that cannot wait for three peers to spend rocks.

The correct reading is not “rocks replace trust & safety,” but **“rocks replace *casual* flag spam with costly peer disputes, while the operator retains narrow duties for law and platform integrity.”** That bet pays off only if copy matches code, throwers understand publicity, statutory intakes stay open, and rock abuse is treated as seriously as listing fraud itself.

---

## Appendix A — Primary code map

| Area | Location |
|------|----------|
| Throw / unthrow / counts / ordinals | `internal/rock/rock.go` |
| Reason codes and labels | `internal/rock/reason.go` |
| Confirm / preview / throw HTTP | `internal/handler/rock.go` |
| Opinion HTTP / message invalidate | `internal/handler/message.go` |
| Ad / user rock ordinal entry | `internal/handler/ad.go` (`PublicAdRockOpinionHandler`), `internal/handler/user.go` |
| Routes | `cmd/server/main.go` |
| Conversation access | `internal/message/view.go` (`CanViewConversation` = participants only) |
| Journal rock events | `internal/journal/journal.go`, `internal/message/journal.go` |
| Opinions (generate, cache, images, redact) | `internal/rockopinion/` |
| UI | `internal/ui/rock.go`, `rock_confirm.go`, `rock_opinion.go`, `message.go` |
| Scales spin CSS | `input.css` (`.rock-opinion-link` / `.rock-throw-review-btn` + `htmx-request`) |
| Search filter | `internal/search/metadata.go` |
| Embedding rock text | `internal/vector/ad.go` (`rockContext`) |
| Config | `internal/config/config.go` |
| Schema | `internal/db/schema.sql` |
| FAQ / Terms / Privacy | `internal/ui/faq.go`, `internal/ui/legal.go` |

## Appendix B — Related public claims

**FAQ `/faq/rocks`:** N rocks at join; throw on policy/problem ads; starts a conversation; excluded when more than `MaxRockCount` rocks; choose a reason; optional provisional assessment; thrower unthrows; use wisely.  

**Terms “Reporting and removal”:** Self-policed with rocks; visible; search exclusion; reason + optional assessment; thrower unthrows; do not email for listing/copyright/content issues—use rocks; operator may still act. Prohibited content lives under `/terms#prohibited`.  

**Privacy “Content reports”:** Listing problems handled in-product via rocks.  

**Local launch doc:** Rocks mark bad actors; ads assumed good until rocked; do not expect rocks to substitute for supply.

## Appendix C — Parameter cheat sheet

| Constant | Value (Aug 2026) | Meaning |
|----------|------------------|---------|
| `MaxOutstandingRocks` | 3 | Max rocks a user may have thrown at once |
| `MaxRockCount` | 2 | Max `ads.rock_count` still in search (`<= 2`); excluded at 3+ |
| `MaxImagesPerAd` | 20 | Listing image cap; also upper bound when opinions attach images |
| Reason codes | `policy`, `conduct`, `deal` | Journal body; labels differ by throw direction |
| Assessment scale | 1–10 | 1 inquirer right … 5 balanced … 10 owner right |
