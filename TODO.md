# TODO List for Parts Pile

## 1. Ad Management
- [x] Implement create, edit, and delete for ads (CRUD)
- [x] Ensure each ad includes description, price, vehicle fitment (make, year(s), model(s), engine(s)), part category, and subcategory
- [x] Timestamp ads
- [x] Ad editing is now inline via the ad detail view, with an edit icon for owners only. No standalone edit page.
- [x] Add optional location field to Ad and ArchivedAd tables in schema
- [x] Add Location field to Ad struct and update all relevant Go code
- [x] Update ad creation/edit forms and details UI to support/display location

## 2. Vehicle Data Integration
- [x] Maintain comprehensive, normalized vehicle database (make, year, model, engine)
- [x] Support accurate fitment and filtering

## 3. Part Categorization
- [x] Organize parts by category and subcategory (e.g., "Electrical" > "Alternator")
- [x] Store categories and subcategories in the database; allow extension

## 4. Search & Filtering
- [x] Implement free-text search parsed by LLM into structured query
- [x] Support filtering by vehicle, category, subcategory
- [x] Support pagination (cursor-based)

## 5. User Management
- [x] User registration, login, logout
- [x] User authentication for ad creation/edit/delete
- [x] User settings page (change password, delete account)

## 6. Token Economy (Partial)
- [ ] TokenTransaction and PayoutFund logic (pending)

## 7. Legal/Compliance
- [ ] Terms of service, privacy policy, and compliance (pending)

## 8. Paid Advertising & Promotion
- [ ] Allow advertisers to pay for prominent placement in search results
- [ ] Visually distinguish paid ads
- [ ] Place first paid ad at top, others interleaved at pagination boundaries
- [ ] Charge advertisers per click; contribute portion to payout fund
- [ ] Intermix paid and regular ads, but clearly mark paid ads

## 9. Non-Functional Requirements
- [x] Fast search and filtering, efficient DB queries, indexed tables (indexes and efficient queries present)
- [x] Extensible for user accounts, more part attributes, etc.
- [x] Input validation and confirmation for destructive actions
- [x] Transactional DB operations for ad CRUD
- [x] Modular Go codebase, clear separation of concerns
- [x] Responsive, accessible UI components

## 10. Open Questions / Future Work
- [x] User authentication and account management (basic registration/login/logout present, but no account management UI)
- [ ] Messaging between buyers and sellers
- [ ] Image uploads for ads
- [ ] Advanced analytics and reporting
- [ ] Internationalization/localization
- [ ] Admin dashboard for moderation
- [ ] Ongoing review of the sustainability model for the payout fund
- [ ] Ongoing legal and compliance review for token use, exchange, and peer-to-peer payments

## 11. Ad Cost Calculation & Payout Fund Drawdown
- [ ] Implement formula: ad cost = base_payout × exp(-λ × total_ads) × fund_factor × weight_factor
- [ ] Cap payout/expense as specified
- [ ] Drawdown and sustainability logic

## 12. Legal & Compliance
- [ ] Legal review for token use, exchange, and peer-to-peer payments
- [ ] Implement KYC/AML and tax reporting if required
- [ ] Record all token transactions for transparency and auditability

## 13. Token and Fund Flow Diagram
- [ ] Ensure system logic matches flow described in diagram

## Implementation Gaps / Codebase TODOs
- [x] Implement LLM-based search parsing (see PRD 3.4)
- [x] Implement all placeholder UI elements (e.g., search bar placeholder)
- [x] Ensure all SQL query placeholders are safe and correct
- [x] Review all code for incomplete, stub, or placeholder logic

## Miscellaneous
- [x] Quieted server logs by handling Chrome DevTools requests for `/.well-known/appspecific/com.chrome.devtools.json`.
- [x] Remove expand/collapse icons and in-place ad detail expansion from ad cards
- [x] Ad cards now only have a bookmark icon (always visible and clickable)
- [x] Clicking anywhere else on the ad card navigates to the ad detail page

### 17. Create Authentication Utilities ✅ COMPLETED
- **Issue**: Repeated user extraction and permission checking patterns
- **Action**: Centralized user context handling and permission checks
- **Implementation**: Added `CurrentUser`, `RequireAdmin`, and `RequireOwnership` helpers to `handlers/auth.go`. All handlers now use these utilities for extracting the current user, checking admin status, and verifying resource ownership. Removed the old `ValidateOwnership` function and updated all usages 