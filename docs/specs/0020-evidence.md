# SPEC-0020 implementation evidence

Specification: [0020-home-redesign.md](0020-home-redesign.md), accepted 2026-10-06.
Implementation branch: `codex/home-design-audit`.
Status: local verification complete; remote CI/release pending.

| Criterion | Executable evidence |
| --- | --- |
| AC1 | Chrome real Go API/PostgreSQL at localhost:18280; indigo light/dark; responsive 320/390px; saved screenshot `bin/spec0020-mobile-card.png` (ignored, synthetic records only) |
| AC2 | Menu → accounts changes URL; mobile bottom nav; details Esc closes; explicit trigger focus restoration added |
| AC3 | `HomePage.test.tsx`: stale calculated amount hidden on analytics failure, accounts remain visible; browser recovered queries via Repeat |
| AC4 | Go `accounts_test.go`: 0012, missing bank, ASCII validation, reject full card number/cash card; `authtest/integration_test.go`: card round trip, RLS, cross-owner FK, rollback without leaked account |
| AC5 | Browser: create test card → review → POST → 1000.01 RUB balance; purchase 10.01 RUB → review → POST → server returns card balance 990.00 RUB, cash 125990.00 RUB; web test proves no POST before confirmation and same retry idempotency key |
| AC6 | At 320px card dialog clientWidth = scrollWidth = 288; operation review 273; footer margin 24px; pointer cursor; two action buttons wrap; body clientWidth = scrollWidth = 305 |
| AC7 | `make test`: Go, 23 web tests, lint, build, offline Jev 49/72; `git diff --check`; generated contract updated; PostgreSQL auth suite passed on isolated test database |

Production-bundle test was attempted on Windows: Git Bash path conversion and
symlink limitations prevent the full deployment fixture here. Linux CI is the
required release gate; its actual run/result is recorded below after completion.

Migration 000007 adds tenant-owned payment instruments only. Creation with an
account is atomic. It does not migrate financial records or duplicate account
balances. SPEC-0017 draft PR 7 also has an unintegrated migration number 000007:
when that work resumes its migration must be rebased/renumbered; this release
does not publish import drafts or edit that unrelated PR.

The financial-health screen preserves the ten-sheet field structure and shows
existing accounts/events/categories/fixed obligations/receipts. Missing spreadsheet
fields remain unknown. Full editable credit facilities/statements, planned salary,
budgets and reconciliation/document management remain separate roadmap work;
they are not simulated as saved financial data. No production test writes planned.
