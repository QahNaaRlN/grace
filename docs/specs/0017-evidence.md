# SPEC-0017: Implementation evidence

Accepted scope: [SPEC-0017](0017-persistent-import-drafts.md), acceptance commit
deba6ae, local integration 2cfbca3. Branch: `codex/persistent-import-drafts`.
Implementation commit: a1ea19d. All local mandatory gates passed on 2026-10-06.
Review branch is published in [draft PR #7](https://github.com/QahNaaRlN/grace/pull/7).
GitHub CI, main integration and production rollout are pending. After merging
current main's SPEC-0018 repair into the review branch, make test/go vet passed
again (24 Web tests); the preserved production entrypoint is included.

## Delivered behavior

The owner-scoped `internal/imports` capability retains original snapshot bytes,
the complete SPEC-0016 report and immutable source records. Audited decisions
use batch revisions. Preview reuses the deterministic financial writer, makes
no ledger writes, expires after 15 minutes and binds the selected decisions,
postings and candidate comparisons to a hash. Confirmation recomputes the plan
under shared financial-owner and batch locks; one transaction writes events,
provenance, published decisions and an immutable receipt. Idempotent replay is
checked before expiry or revision so a lost response cannot duplicate facts.

Five existing event kinds use the same financial core. The core preserves
manual idempotency behavior and takes the shared owner lock. Group sums reject
int64 overflow; aggregate refunds cannot exceed the original purchase. Import
source is explicit. Links/duplicates/exclusions have provenance without extra
postings. Unsupported domains and semantics remain evidence. All seven new
tables have FORCE RLS, limited grants and tenant-aware composite foreign keys.

OpenAPI defines all nine routes, discriminated decisions, source/report DTOs,
preview and receipt. Generated Web types are committed. `/imports` provides
upload, paginated resume/review, source/audit/candidates, existing selectors,
status/domain filters, preview and a separate confirmation action. Confirmation
retains its idempotency key across retry; a saved receipt URL survives reload.

## Acceptance evidence

Run `./scripts/test-imports.sh` for isolated real PostgreSQL, or on Windows run
`bash scripts/test-windows.sh integration`. The latter invokes the same Make
target; its adapter sends JSON via UTF-8 stdin instead of Windows curl argv.
Only committed synthetic fixtures and generated synthetic data are used.

| AC | Executable evidence |
| --- | --- |
| AC1 | `TestPersistentImportLifecycle`: full snapshot bytes, both operation blocks, raw cells, new Repository reload, audit, zero postings before confirmation. Manual browser upload/edit/reload on synthetic ten-sheet fixture. |
| AC2 | Same lifecycle: same-hash batch reuse, changed-hash published evidence candidates, explicit link without extra posting; no candidate auto-merges. |
| AC3 | Lifecycle: revision conflict, audit, invalid self/chain duplicate, immutable published edit. DB triggers additionally prevent published-record/receipt updates. |
| AC4 | `TestIsolationAndHTTP`: every new table hides another owner's rows and has FORCE RLS, cross-owner record FK rejected, other admin gets 404, foreign financial event cannot be linked. `TestImportSessionCSRFAndOrigin`: 401/403/403/201 matrix for session, Origin and CSRF. |
| AC5 | Lifecycle exact deltas; `TestExpiredAndChangedPreview`: expired preview, changed draft and new manual event candidate prevent first publication. |
| AC6 | Lifecycle lost-response replay even after expiry; `TestConcurrentConfirmation` different keys; `TestFiveKindsAndAtomicRollback` injected last-event failure rolls back all events, provenance and published flags. |
| AC7 | Five kinds test includes balanced transfer and original purchase refund; lifecycle covers link/duplicate; excluded row has no posting. `TestBlockedDateAndFinancialGroup` and `TestUnsupportedSemanticsAndSourceFilters` cover explicit time, borrowed income, HOLD and unsupported source sheets. `preview_test.go` checks integer overflow and malformed decisions/selection. |
| AC8 | `ImportsPage.test.tsx`: delayed record refresh cannot reset a saved decision or publish it. `PreviewPanel.test.tsx`: separate click, server blockers and pending state. Browser: saved exclude/create decisions survive reload; account/category/time form; preview of 14497 minor units; 390×844 viewport (375 content width) and 1280×900 (1265 content width), no horizontal overflow, keyboard focus visible. Screenshots retained locally in ignored `bin/spec0017-{mobile,desktop}.png`; no real ledger confirmation in browser. |
| AC9 | Migration up/down/up before publication; down refusal after confirmation. Generated contract, make test, go vet, full make test-integration and whitespace gates recorded below. |

## Final gates and limitations

- `bash scripts/test-windows.sh unit` invokes `make test` and `go vet ./...`: PASS, including 24 Web tests, lint, production build and offline Jev datasets (49/72 cases).
- `bash scripts/test-windows.sh integration` invokes full `make test-integration`: PASS for imports, authentication, financial events, receipts, obligations, analytics, assistant tools, income-relative periods and production read-only router.
- Isolated import integration includes eight real-PostgreSQL tests, migration up/down/up before publication and expected protected rollback refusal after publication: PASS.
- Windows path conversion and curl argv encoding were diagnosed; UTF-8 stdin adapter preserves the existing assertions.
- OpenAPI regeneration followed by SHA-256 equality check: PASS, no generated drift.
- `git diff --check`: PASS.
- Post-merge/CI: not run; this implementation is not in remote main or production.

Original int64 source bytes are retained without JavaScript parsing on upload.
Web editing/confirmation only supports safe integer values; exact display beyond
that range is intentionally blocked. No source-private data or invitation token
is committed. Publication of real financial data requires the user's separate
confirmation through the product.

Обновление 2026-10-16 (сверка roadmap): CI `main` и Deploy production №12 для `f257f8e` успешны ([0028-evidence](0028-evidence.md)); статус `integrated`, работает в production.
