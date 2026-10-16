# SPEC-0027 — evidence

Accepted specification: [0027-date-only-history.md](0027-date-only-history.md).
Acceptance: owner explicitly approved calendar dates with unknown time on
2026-10-08. Implementation: `597cfcc`, branch `codex/history-release`.

| Criterion | Executable evidence |
| --- | --- |
| AC1: nullable timestamp, date and idempotency | `TestDateOnlyHistoryPublicationAndCalendarAnalytics`: database NULL timestamp, exact calendar date, JSON null, repeated confirmation does not double the calendar total. Existing import lifecycle tests cover retained source/provenance and atomic publication. |
| AC2: validation and reviewed matching | `internal/financialevents/occurrence_test.go`: valid leap day; invalid/nonexistent day; neither/both forms rejected. Import matching and source-month correction use the calendar date. Existing candidate tests retain explicit owner review. |
| AC3: calendar totals and ambiguous chronology | PostgreSQL test above verifies whole-day total, `ErrTimePrecision` for partial-day history and `income_time_unknown` for date-only income. Existing analytics/income-relative integration checks cover exact-time behavior. |
| AC4: honest UI and contract | `finance-display.test.ts` rejects invented hours/epoch. `ImportsPage.test.tsx` saves only occurredDate, leaves occurredAt undefined, and never publishes from the edit action. Regenerated OpenAPI types, TypeScript build and lint pass. Responsive date fields retain the existing single-column mobile layout; production visual smoke remains pending. |
| AC5: migrations and required gates | `scripts/test-imports.sh` verifies migrations 1–10, down/up before publication and deliberate rollback refusal after date-only history. `bash scripts/test-windows.sh unit` passes: make test, 39 Web tests/10 files, lint/build, offline Jev datasets and go vet. `bash scripts/test-windows.sh integration` completes all nine integration scripts successfully, including exact-time analytics, authentication/RLS, receipts, obligations and assistant regressions. Staged `git diff --check` passes. |

## Release boundary

The release also includes verified SPEC-0017, SPEC-0023 and SPEC-0024. Their
existing evidence is retained. No Gemini/LLM experiment code is included.

The owner's complete spreadsheet snapshot is retained outside Git. Owner
deferred account mapping, opening balances and candidate verification until
after release. Production upload must save a draft only: no invented account
baseline, automatic reconciliation or ledger confirmation. Financial summaries
continue to describe confirmed events; draft rows are reviewed separately.

Remote CI, delivery, deployment and real-source draft upload are separate gates
and remain unverified until their actual results are recorded.

## Rollback

Migration 000010 does not alter existing timestamp values. Its down path refuses
to discard date-only events. After first date-only publication, use a compatible
application release or the established backup/restore procedure. Production
deploy creates a pre-release backup and never automatically downgrades the DB.

Обновление 2026-10-16 (сверка roadmap): CI `main` и Deploy production №12 для `f257f8e` успешны ([0028-evidence](0028-evidence.md)); статус `integrated`, работает в production.
