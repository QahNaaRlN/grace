# SPEC-0059 R implementation evidence

Accepted contract: [SPEC-0059 R](0059-receipt-normalization-candidate.md), owner
acceptance recorded and integrated in PR144 before implementation. This report
contains aggregate verification only; the real document, raw response, reference,
generation identifiers and draft identifier remain outside Git.

## Implemented behavior

A strictly valid local clock plus calendar date becomes a date-only AI candidate.
The original extraction and source text remain immutable. New extractions store
this candidate separately; untouched legacy extracted drafts receive a read-only
projection with no SQL/revision update. Owner-edited and confirmed fields are not
normalized. Invalid dates/clocks and conflicting timestamps remain invalid.

Draft, core command, direct receipt HTTP validation and SQL accept nonnegative
child unit/line amounts, with strictly positive quantity and purchase total.
Existing exact draft arithmetic, currency, overflow and confirmation guards stay
in force. Migration32 down refuses to discard retained free items. The Web shows
the original text in a read-only disclosure and labels the date without claiming
that the printed time is unknown. OpenAPI and generated TypeScript are synchronized.

## Acceptance evidence

| Criterion | Executable evidence | Result |
| --- | --- | --- |
| R1 strict clock/date and original retention | `go test ./internal/receiptdrafts -run 'TestExtractedReceiptClockProjection|TestReceiptProjectionOnlyForUntouchedExtractedCandidate'` | PASS |
| R1 SQL new/legacy candidate, GET without write, manual edit | `./scripts/test-document-reading.sh`: `TestS56ReceiptNormalizationOriginalRetentionAndFreeConfirmation` | PASS |
| R2 free quantity2 row, positive purchase, replay, missing/foreign/stale confirmation, safe downgrade | Same SQL test and `TestS56ReceiptDocumentLifecycle/image_local_clock_free`; `go test ./internal/receiptdrafts ./internal/receipts` | PASS |
| R2 empty migration up/down/up | `./scripts/test-document-reading.sh` migration32 | PASS |
| R3 original text after local and saved edits, date label | `cd web && npm test -- --run src/features/recognize-receipt-photo/ui/ReceiptPhoto.test.tsx` | PASS |
| R4 saved real response offline replay using the Go projection | Private `r-replay.sh`, sealed original reference/comparator, private `r-comparison.json` | Candidate80/80; original79/80 unchanged; all9rows; providerPOST0 |
| R5 general gates | `make test`, `go vet ./cmd/... ./internal/...`, `make test-production`, `./scripts/test-bank-statements.sh`, `git diff --check` | PASS |
| R5 remote CI / exact-SHA Delivery/deploy | Implementation PR and main workflows | PASS |
| R6 existing production draft reload and counters | Read-only browser + before/after SQL snapshots | PASS |

Private replay uses temporary synthetic account/action validator inputs only in
memory; it performs no SQL or provider calls and does not assign a real account
or category. The real draft remains unconfirmed. No claim of general OCR quality
or a new model inference is made. The previous known settlement is unchanged.

Local gates completed 2026-10-26: all unit/Web/lint/build gates, Go vet,
document SQL integration, bank SQL retention/restore controls and production
deployment/backup checks PASS. The three-step empty migration replay and the
retained-free-row refusal both passed.

## Integration and production verification — 2026-10-26

Implementation source `3f2a5e7f9f2f33256acdf143ff67a36ee5e04dba` merged in
[PR145](https://github.com/QahNaaRlN/grace/pull/145). Exact runtime main SHA:
`da621d94de1c956ce03ddfff4081cbbfb06dee52`.

- [PR CI](https://github.com/QahNaaRlN/grace/actions/runs/40369280891): all5 PASS.
- [Main CI](https://github.com/QahNaaRlN/grace/actions/runs/40370888189): all5 PASS.
- Post-merge `make test`, `go vet ./cmd/... ./internal/...`, `git diff --check`:
  PASS; primary checkout clean.
- [Delivery](https://github.com/QahNaaRlN/grace/actions/runs/40372594200): all4 PASS.
- [Deploy](https://github.com/QahNaaRlN/grace/actions/runs/40372795492): success;
  verified pre-deploy PostgreSQL backup, migration32 clean, exact API/Web/PDF
  images healthy. Public shell/login/meta200, protected API401, no Basic challenge.

R6: one authenticated reload of the existing saved production draft showed the
correct calendar date and label, all9rows, a quantity2 free row with zero price
and line total, and original source text with its printed clock. No reading plan,
provider POST, real account/category assignment or financial confirmation was
performed. Read-only SQL snapshots before/after are identical for the complete
stored draft, including original fields/extractions, status and revision2.
AI usage/event/posting/receipt/unsettled deltas are all0; unsettled remains0 and
the prior known settlement remains unchanged. The original reference and
comparator seal are intact. This UI/SQL verification is distinct from the private
offline candidate80/80 replay; it is not a second OCR call.

Only receipt membership changed to revision `spec0059-r1`; same model, canonical,
provider, single-image bound and positive fee ceilings. Fresh product capability
resolution succeeded through the shared three-GET resolver; no standalone probe
or repeated inference was used. Qualification expires **2026-11-02 17:55:14 UTC
(22:55:14 Asia/Yekaterinburg)**, seven days after application. Analysis/statement
memberships, auth/AI flags and purchased allowance were verified unchanged.
Protected pre-change configuration remains available for rollback. New OpenRouter
cost for R: $0; no automatic qualification renewal or new agent inference grant.

## Rollout boundary

Deploy with the receipt qualification initially closed. After the local replay
and SQL gates, reopen only the same accepted receipt/model/provider for at most
seven days, with the existing ceilings and a fresh three-GET capability proof.
Preserve the other groups, allowance and financial data. No paid POST, automatic
renewal, real purchase confirmation or account/category choice is authorized.
If free items have been confirmed by an owner, retain the compatible schema/API
instead of forcing a destructive downgrade.
