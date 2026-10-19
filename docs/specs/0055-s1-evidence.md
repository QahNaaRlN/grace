# SPEC-0055 / S1 implementation evidence

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Принятый контракт: [S1 document-atom.v1](0055-record-atomic-planning.md), принят прямым ответом владельца «Утвердить S1 с новыми лимитами»; specification PR #69 integrated main `dcd2bb8`.
- Implementation branch: `codex/statement-document-plan`; локальные проверки выполнены, implementation опубликован в [PR #70](https://github.com/QahNaaRlN/grace/pull/70); live CI/integration status — в PR. До интеграции результат не называется deployed/production accepted.
- Scope: whole-document plan, aliases, exact limits/consent, pinned text-only transport profile, HTTP/cancellation и существующий Go→review→confirmation path.
- External LLM calls: **0**. Production/key/flags/allowance не менялись. В fixtures использованы synthetic requests и disposable PostgreSQL.

## Результат

Один atom / один request, максимум 2000 source spans и 90000 messages+schema
bytes. Alias lex-base36 map сохраняется и связывается plan hash вместе с
версиями и limits. Typed selections переводятся в canonical IDs до merge и Go
validation; source values и descriptions не подменяются. Нет автоматического
service для остатка, нет повторного model call по использованному preview.
Повтор того же idempotency key читает уже существующий attempt.

Отдельный statement transport profile имеет 100000 input / 16000 output,
120 s call, 150 s request/attempt, 180 s HTTP write deadline. Остальные
assistant/receipt callers сохраняют 10000/2000 и 30 s. Profile не принимает
image messages или включённый reasoning; required exact route/privacy flags
и bounded provider envelope проверены локальным mock gateway.

Старый `statementqualification.Run` smoke cap **32000 micros** сохранён.
Два plans нового профиля требуют theoretical reservation 140000 micros и
получают `qualification_cap_exceeded` **до reserve/provider call**. Утверждение
S1 не превращается в разрешение на новый paid campaign и не включает AI.
Read-only production readiness threshold соответствует одному 70000-micros
call; это не qualification и не вывод реального remaining.

## Acceptance criteria → executable evidence

| Критерий | Результат | Evidence |
| --- | --- | --- |
| S1-AC1 | LOCAL PASS | `TestStatementDocumentLayouts` real pinned helper: columns/cards, один complete chunk, exact-once source IDs, full-record oracle |
| S1-AC2 | LOCAL PASS | [Actual planner report](../research/statement-quality/s1-evidence/summary.json): 28 layouts × v7/v8/current prompt = **84 PASS**; 56 historical-prompt cases и 28 current v9. Compile overlays не меняют production files; каждый input содержит все source IDs |
| S1-AC3 | LOCAL PASS | `TestDocumentAliasesRoundTripAllTypedSelections`, SQL subtest `bad-alias`; known alias reuse сохраняется, canonical/foreign/unknown/duplicate wire selections отвергаются; reason/direction/source неизменны |
| S1-AC4 | LOCAL PASS | `TestDocumentPlanExactByteAndSpanBoundaries`: 89999/90000/90001, 2000/2001. `TestStatementExtractionConsentMeteringReplay`: oversize preview до reserve/call. Capability/envelope negative tests до HTTP |
| S1-AC5 | LOCAL PASS | Redaction regression tests integer fragments/phone/email даже с visible IDs; `TestStatementWireProfileRoundTripAboveLegacyTokenBounds`: gateway не получает canonical map, only pinned ZDR/deny/no-fallback/no-thinking text payload |
| S1-AC6 | LOCAL PASS | `TestDocumentPlanContractRejectsMutationsAndLegacyConsent` и SQL limit/map/prompt mutation tests с пересчитанным hash: 409 до calls/reservation; реальный SQL roundtrip plan/map/limits, owner/RLS/stale/idempotency guards |
| S1-AC7 | LOCAL PASS | `TestStatementRealPDFFakeProviderLifecycle`: helper → actual HTTP handler/CORS → fake только по delivered tuples → canonicalization → SQL; даты, money/currency/FX, totals/coverage проверены. `missing-coverage` остаётся source blocker |
| S1-AC8 | LOCAL PASS | До confirmation неизменны counts financial events/postings/receipts/settlements; owner создаёт одну purchase, явно исключает own-transfer row; confirm replay не создаёт второй posting. Manual suites проходят без Reasoner |
| S1-AC9 | LOCAL PASS (contract/runtime fixtures) | Profile default isolation, exact verified billing выше прежних token bounds, budget reserve 70000/settle, real HTTP delayed fake/global timeout и browser cancellation с persisted known bill; unknown bill freeze, restore/downgrade preservation, generated OpenAPI types/build |

Rollback check проверяет serialization/version contract: старый typed decoder
не восстанавливает новый hash, новая версия отвергает legacy/mutated plan;
история attempt после удаления full plan остаётся читаемой. Полный rollback
production release и owner session acceptance **не выполнялись**.

CORS и principal-bearing request context проверены в реальном HTTP test;
production `auth.Require` inspected: writer не подменяет, request cancellation
сохраняет. Web API/mutation inspected: short timeout не задан, paid mutations
имеют explicit `retry:false`. Caddy config не задаёт отдельный short upstream
response-header timeout. Production proxy/browser end-to-end не запускался;
это не утверждение owner acceptance.

## Команды и результаты

- `make test` — PASS: Go, **87 Web suites / 330 tests**, lint/FSD/format/build, offline Jev datasets.
- `make test-integration` — PASS: полный repository suite с isolated containers/DB. После новых negative/cancellation assertions дополнительно выполнен final `./scripts/test-bank-statements.sh` — PASS.
- `make test-production` — PASS: local mock deployment/activation/backup gates; production deploy не запускался.
- `go vet ./cmd/... ./internal/...`, `git diff --check` — PASS.
- `./scripts/research/verify-statement-document-plan.sh docs/research/statement-quality/s1-evidence` — PASS, **84 structural cases**, actual Poppler 25.04.0; network=none/read-only/UID10001 runtime, no model/SQL.
- Final focused Go tests и Web AI consent tests проверяют последние transport/retry guards; generated TypeScript получен через `npm run generate:api`, ручных DTO нет.

Local logs: `/tmp/grace-s1-freeze-all.log`, `/tmp/grace-s1-integration.log`,
`/tmp/grace-s1-bank-complete.log`, `/tmp/grace-s1-production-gates.log`,
`/tmp/grace-s1-production-planner-proof.log`. Reports содержат только public
synthetic case names, source-code/fixture hashes и software versions.

## Оставшиеся gates к включению

S1 не доказывает качество Gemini на новом wire payload и не снимает прежний
cards quality FAIL. Реальный new-profile schema compatibility, token use,
latency/billing и повторяемость остаются `UNVERIFIED`. S2 explicit coverage
protocol, S3 expanded corpus/golden review и отдельно утверждённая bounded
qualification сохраняют свои этапы. SPEC-0056 registry/model auto-selection
остаётся draft/not_started. Ручной импорт и financial confirmation сохраняются.

Датированное продолжение 2026-10-19: старый 32000-micros smoke cap был сохранён
именно в S1. Позже владелец отдельно принял [owner-test activation amendment](0055-owner-test-activation.md)
с двумя calls / aggregate 140000 micros и единственным разрешённым smoke.
Это самостоятельное решение, а не ретроспективное расширение разрешения S1.
