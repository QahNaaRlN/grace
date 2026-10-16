# SPEC-0023: Implementation evidence

Accepted scope: [SPEC-0023](0023-monthly-expense-migration.md), acceptance
commit 20900ba, direct owner instruction on 2026-10-07.
Branch: `codex/monthly-expense-migration`. Implementation commit: b6ec8bc.
Dependency integration: 76a7a49; current email/profile authentication,
navigation and payment instruments remain compatible. Import migration is
000009; existing 000007/000008 are unchanged.

Status: locally verified. Main integration, remote CI, production rollout and
publication of the owner's real financial history have not occurred. Every
fixture and confirmed event used here is synthetic.

Обновление 2026-10-16 (сверка roadmap): CI `main` и Deploy production №12 для `f257f8e` успешны ([0028-evidence](0028-evidence.md)); статус `integrated`, работает в production.

## Delivered behavior

CSV uploads preserve original bytes, source coordinates and text amounts;
UTF-8/BOM, comma/semicolon and quoted multiline fields are supported. Invalid
encoding, extra columns, malformed CSV and incompatible headers fail before
writing a batch. JSON snapshot remains available for full formula/sheet evidence.

Month reconciliation reads the entire batch in one repeatable-read transaction.
Source sums include possible duplicates; prepared and confirmed purchase totals
count unique target events, with separate currencies and null for unknown or
overflow. Rows retain original/target fields, status and reasons. Missing dates
require explicit reviewed evidence or confirmed exclusion. Main-block and
corrected-row duplicate candidates require review, never automatic merging.

The UI supports CSV/JSON, month selection, all source rows and separate
confirmation. Confirmation retains the selected month and refreshes financial
and analytics caches. SPEC-0017 supplies audited drafts, RLS, CSRF/Origin,
provenance and atomic/idempotent publication.

## Acceptance criteria and executable evidence

| Criterion | Executable evidence |
| --- | --- |
| AC1: CSV preserves all rows and exact values | `TestCSVPreservesAllRowsAndExactAmounts`: 301 rows through row 305, BOM/both delimiters, quoted multiline cells, 762.75 RUB = 76275 minor units, no invented time. `TestCSVBlankRowsAndInvalidStructure`: blank coordinates and rejected malformed/encoding/header/extra-column inputs. |
| AC2: entire month and separate exact totals | `TestMonthCoversEntireBatchAndUnknowns`: 191 rows, unknown/outside dates and missing money. `TestMonthCurrenciesOverflowAndCalendar`: int64 overflow, RUB/USD separation and Yekaterinburg midnight. `TestCSVMonthLifecycleAndHistoricalAnalytics`: PostgreSQL page returns 100 but report contains all 191. |
| AC3: decisions, duplicates and completion | `TestMonthDecisionsUniqueEventsAndCorrections`: source 5000, prepared 900, confirmed 1000; repeated link, exclusions and date changes. `TestMonthMissingDateNeedsExplicitReviewedDate`: explicit date evidence retains unknown original date; only published reasoned exclusion resolves an excluded undated row. `TestCSVMainBlockDuplicateNeedsReview`: original and corrected main-block matches block unreviewed preview. |
| AC4: historical ledger, provenance and repeat | `TestCSVMonthLifecycleAndHistoricalAnalytics`: original CSV bytes retained; upload/edit/report do not affect September analytics; confirmation adds exactly 76275 RUB with September date; replay adds nothing. Existing `TestPersistentImportLifecycle`, `TestConcurrentConfirmation`, `TestFiveKindsAndAtomicRollback` cover provenance, lost response, concurrent publication and rollback. |
| AC5: isolation and migrations | Lifecycle: foreign admin gets repository not_found and HTTP 404. `TestImportSessionCSRFAndOrigin`: CSV/JSON 401/403/403/201 matrix. `TestIsolationAndHTTP`: RLS/FK/event protections. `scripts/test-imports.sh`: migrations 1–9 up, import down/up before publication and protected down refusal afterward. |
| AC6: UI and generated contract | `MonthPanel.test.tsx`: 191 rows, unknown values, explicit month, error handling and source opening. `ImportsPage.test.tsx`: original CSV File without numeric parsing, month retained after confirmation, caches refreshed and delayed detail reload. `PreviewPanel.test.tsx`: separate click and blockers. Current mobile layout/menu and existing ≥44px components retained. OpenAPI generation and TypeScript build PASS. |
| AC7: required checks | `bash scripts/test-windows.sh unit`: make test + go vet PASS; 37 Web tests in 9 files, lint, build and offline Jev fixtures. `bash scripts/test-windows.sh integration`: full make test-integration PASS. Import integration repeated for expanded matching via `bash scripts/test-imports.sh`: PASS. Final make test includes reviewed-date/exclusion unit cases. Staged `git diff --check`: PASS. |

## Environment and limitations

Windows checks used Git Bash and Docker Desktop/PostgreSQL. The Windows test
adapter uses a common temp directory and UTF-8 stdin for curl. An ignored local
npm wrapper invokes bundled Node 24.19.0. Installed Node 23 produced dependency
engine warnings; initial sandbox-isolated Vitest workers could not share temp
module files. The common-temp adapter resolved the environmental failure;
package/lockfile changes were unnecessary.

Build succeeds with the existing large-bundle warning. Fresh visual browser QA
and the real-file owner smoke test remain for release review. Date-only history
requires explicit time review under SPEC-0017; CSV cannot preserve formulas
absent from the export. Opening balances are not adjusted automatically. No
private source, token or screenshot was committed. Docker test projects are
cleaned by script traps. Review and release remain separate from local verification.
