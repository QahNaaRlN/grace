# SPEC-0055 / S1.2 implementation evidence

- Дата актуализации: 2026-10-20 (Asia/Yekaterinburg).
- Accepted contract: [description separators](0055-s12-description-separators.md), прямое решение владельца «Утвердить S1.2, один smoke и включение при PASS». Принятие записано в `bf4b3c8` до production-кода.
- Branch: `codex/statement-description-separators`, integration base `ef782ec`.
- Production baseline: S1.1 `ef782ec`, deploy `39255669019` 2026-10-19, internal health PASS; smoke columns `record_description` FAIL, cards NOT_RUN, flag false. Это использованный S1.1 run, не новое разрешение S1.2.
- После восстановления quota автоматическая проверка разрешила запись в worktree. Ранее подготовленный `/tmp/grace-s12-review/S1.2.patch` применён через неё, без обхода. Новый S1.2 paid smoke ещё не выполнен.

## Изменение и проверка D1–D5

| ID | Evidence |
| --- | --- |
| D1 | [Public-source replay](../research/statement-quality/s11-description-offline.json), 0 provider calls / 0 SQL. Печатный `p2-b8-l1-w3` (`-`) пропущен из второй description в service; первая exact. Кандидат messages+schema 9060/9401 bytes, оба <90000. Исторические source hashes относятся к S1.1, не к текущему v11 |
| D2 | В `internal/statementextract/plan.go` добавлена ровно принятая инструкция; PromptVersion `statement-span-mapping.v11.description-separators`. Alias/planner/redaction/schema/scorer/report versions, tuples, лимиты, privacy и financial Go не меняются. Existing hashed-plan/profile validation требует fresh preview/consent; старый prompt несовместим |
| D3 | `TestActivationRealHelperSmoke/missing_description_separator`: actual pinned helper → delivered-tuples-only fake → Go/scorer. Negative переносит separator в service, сохраняя financial fields и coverage. Обязательный исход: 2 rows / 19 checks / 18 correct, единственный code `record_description`, без второго call, известный fake bill settled. Complete baseline PASS; остальные negative checks сохранены |
| D4 | [Actual planner proof](../research/statement-quality/s12-evidence/summary.json): **84 PASS**, pinned helper network-none/read-only UID10001, source hashes соответствуют текущему коду. `make test`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check` **PASS** (exit 0). Полный `make test-integration` ещё выполняется; real-helper bank suite, включая separator negative, уже PASS. Его итог и exact-head CI/Delivery фиксируются перед merge/release |
| D5 | **NOT_RUN**: отдельно утверждён один новый S1.2 columns/cards smoke ≤2 calls / $0.14 aggregate, прежний shared allowance/fees, no retry. Enable только оба полных 19/20 PASS, exact-release CI/Delivery/deploy и actual-image/runtime/policy/public checks |

Local logs: `/tmp/grace-s12-test.log`, `/tmp/grace-s12-integration.log`,
`/tmp/grace-s12-production.log`, `/tmp/grace-s12-vet.log`,
`/tmp/grace-s12-structural.log`. PASS определяется завершённым exit status.
При наличии постороннего read-only `/tmp/.git` CLI privacy guards сохраняются;
для финальных CLI gates использовать собственный TMPDIR вне этого repository.

Никакая автоматическая починка separator/нормализация description не добавлена.
Scorer остаётся строгим, 19/20. Hypothesis о поведении новой модели на v11
UNVERIFIED до live smoke. Ни successful fake, ни byte/structural PASS не являются
live quality evidence. Full AC12/Jev/cascade/corpus и owner private acceptance
остаются самостоятельными этапами.

Read-only baseline 2026-10-20: run `39393586879` PASS, fixed payload `statement_ai_runtime_disabled`, activation prerequisites passed; без inference/configuration writes.
