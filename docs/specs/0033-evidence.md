# SPEC-0033: implementation and review evidence

Date: 2026-10-10. Accepted specification: [SPEC-0033](0033-ux-refinement.md).
Branch: `codex/ux-refinement`, combining the existing UI/password/statistics work
with accepted SPEC-0026. Local implementation; owner review precedes deployment.

## Acceptance criteria

| Criterion | Implementation | Executable evidence |
| --- | --- | --- |
| AC1: card identification and type | Shared `accountLabel` and `AccountIcon`; bank + last four digits in lists, forms, imports and assistant; leading zeroes retained; accounts without card details keep their name | `make test`: HomePage account/create-account tests; browser creates a preview of Т-Банк *0012, without submitting a financial write |
| AC2: personal expense categories and selections | Create/rename dialog; global categories read-only; API validation; owner RLS and cross-owner relation guard; server filters category/date before the 100-event limit | `scripts/test-ux-refinement.sh`: `TestPersonalCategoriesAndFilteredHistory`, including 105 newer events, retained purchase ID/amount on rename, two owners, duplicate/invalid names, inaccessible category and local date boundaries; `Categories.test.tsx` checks explicit submission and retained form on conflict |
| AC3: consistent loading, file input and disclosure | Shared skeletons, Radix Collapsible/ChevronDown, preserved mounted form state, inert closed content, keyboard interaction and reduced motion; file padding 4px on every side | `make test`: `disclosure.test.tsx`; browser loading fixture shows 11 skeleton elements and aria-busy=true; file input computed height 44px, button 34px, padding 4px; disclosure keyboard toggle and 220ms grid transition inspected |
| AC4: understandable AI and health | OpenRouter analysis/receipt sections, server capabilities and separate consent; old read tools below disclosure; [user guide](../ai-user-guide.md); health uses existing Go cash-position/payment data, explicitly flags unknown balances and incomplete history | `make test`: AssistantPage, ReceiptPhoto and FinancialHealth tests; `scripts/test-ai-experiment.sh` and consolidated integration; browser desktop/mobile displays guide, consent, debit/credit labels and unknown balance |
| AC5: regression and review readiness | Authoritative OpenAPI regenerated, consolidated migrations, preview from current branch | `make test`, `git diff --check`, all integration scripts below, browser checks in both themes |

## Why financial health changed

The previous screen resembled a spreadsheet report, with unavailable fields and
no clear next action. It now answers three concrete questions: what remains after
known required payments, which payments need attention, and whether the available
data is complete enough to trust the result. Links lead to accounts, import,
payments and statistics. No score, assumed salary, credit limit as spendable cash,
or client-side financial arithmetic was introduced. The disclosure identifies
counts as loaded records, not a complete history.

## Verification

- `make test`: PASS; Go packages, 66 web tests in 18 files, lint, production web
  build and offline Jev datasets v1=49/v2=72.
- `git diff --check`: PASS.
- Every script in `make test-integration` passed against isolated PostgreSQL:
  `test-ux-refinement`, `test-password-authentication`, `test-ai-experiment`,
  `test-imports`, `test-authentication`, `test-financial-events`, `test-receipts`,
  `test-obligations`, `test-analytics`, `test-assistant-tools`,
  `test-income-relative-periods`, `test-assistant-router`.
- The initial aggregate integration invocation stopped on Windows path conversion
  at `/dev/null`. Remaining scripts were run separately with Git Bash and
  `MSYS2_ARG_CONV_EXCL='-path='`. A second receipt test identified native curl's
  Cyrillic argument encoding; JSON Unicode escapes now preserve the same Russian
  fixture values, and the unchanged `Молоко` round-trip assertion passes.
  These were test-harness problems; no receipt business validation was weakened.
- Local ignored logs: `web/node_modules/ux-final-check.log`,
  `ux-all-integration.log`, `ux-finance-integration.log`,
  `ux-remaining-integration.log`, `ux-receipts-final.log`.
- Browser: desktop light/dark, mobile 390px and 320px; cards, category create and
  rename, operations filter, health, assistant, import and loading states.
  At 320px the document scroll width is 305px with a 15px vertical scrollbar,
  so it has no horizontal overflow. Closed disclosures preserve entered values.

## Review and rollout boundaries

Preview: `http://127.0.0.1:6224/node_modules/.grace-ui-qa/ui-preview.html`.
It uses demonstration data. Category changes are in memory and reset on reload;
external AI, mail and financial writes are blocked. The `loading=1` query shows
skeletons; `screen=/health` opens the reviewed health screen directly.

Migration numbering on this unpublished consolidated branch: password 000012,
AI usage 000013, conversations 000014, receipt date precision 000015, receipt
drafts 000016, personal categories 000017. Existing historical reports retain
their original branch numbering. Rollback guards refuse deletion of password
credentials and personal-category ownership; validated in consolidated tests.

Categories are selected when creating purchases or reviewing import; rename
keeps existing links. Deletion and arbitrary editing of published purchases are
outside this scope. Operation selection returns up to 100 matching recent events;
the UI states that limit. AI analysis remains a monthly expense explanation,
with receipt recognition as a separate scenario, rather than an unrestricted
banking agent. Fresh financial calculations remain in Go.

No production migration, deployment, live mail or paid model request was performed.
Production rollout and real-provider smoke checks remain pending owner review.
The build passes with a bundle-size advisory; code splitting is future performance
work and does not affect the checked interaction behavior.
