# Owner-test activation: implementation evidence

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Контракт: [accepted owner-test activation amendment](0055-owner-test-activation.md), прямое решение владельца «Утвердить реализацию, один smoke и включение при PASS».
- Branch: `codex/statement-owner-activation`, база main `8ed8e80` (S1 PR #70); S1 main CI `39243697470` и Delivery `39244588305` PASS.
- Этот отчёт фиксирует проверку реализации до live smoke. Runtime completion требует отдельного evidence OA7/OA8 ниже; зелёные offline тесты не включают AI.

## Реализовано

Exact two-fixture smoke использует S1 whole-document wire, 120 s / call,
300 s total, aggregate provider ceiling 140000 micros. Оба plans и pinned
fixture hashes проверяются до whole-run reservation. Arbitrary corpus/caps,
retry, repair и fallback не добавлены. Allowance/fees не менялись.

Protected report v2 связывает fixtures, document/plan hashes, весь current
profile, policy, результаты scorer и verified calls. Digest выявляет случайную
подмену файла; это **не подпись от злонамеренного оператора**. Файл остаётся в
доверенной operator boundary mode 0600. Summary отвергает старый version,
неполный/несвежий report, duplicate JSON keys (включая case variants), trailing
JSON и несовместимые profile/fixture/check/call/billing evidence.

Production API/OpenAPI/финансовые правила не менялись. Публичный `/api/meta`
показывает доступность сервиса; факт включения подтверждается отдельно через
runtime `grace-statement-admin -mode=check`, image SHA и deployment checks.
Новый report называется activation smoke, не qualified registry entry.

## Acceptance criteria → executable evidence

| ID | Evidence / результат до production |
| --- | --- |
| OA1 | `TestNewProfileReservesApprovedAggregateBeforeProvider`, `TestSmokePreflightAndCancellationNeverRetry`: aggregate 140000, bad fixture/oversize second plan/denied budget → 0 calls; real helper complete path ровно 2 calls |
| OA2 | `TestQualificationBillsMalformedAndNeverRetries`, `TestSmokePreflightAndCancellationNeverRetry`: call deadline 120 s, cancel/first FAIL → second call не начинается, known cost settled, invalid/unknown usage uncertain; complete path settles обеих calls |
| OA3 | `TestActivationRealHelperSmoke`: real pinned helper → actual S1 tuples-only fake → canonical → unchanged Go validator/Score → report roundtrip. Columns/cards PASS; missing coverage и wrong FX FAIL без повторов. Binary runtime network=none/read-only/UID10001, нет DB/financial writer |
| OA4 | `TestActivationReportRejectsLegacyIncompleteChangedEvidence`, `TestDecodeReportRejectsAmbiguousInput`: legacy, partial, changed profile/fixture/plan/checks/calls/billing/time/digest, duplicate/trailing/unknown/oversize JSON rejects; real successful report accepts |
| OA5 | `scripts/test-statement-activation.sh`: quality FAIL и summary FAIL не меняют flag; deploy FAIL восстанавливает config; replay при valid runtime не делает inference; disable без inference |
| OA6 | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`; результаты полного exact-head CI tracked в implementation PR. Focused real-helper/bank suite и final Go package tests пройдены локально |
| OA7 | До публикации: readiness inspect `39245147523` PASS, no configuration/inference writes. Implementation image/production smoke **NOT_RUN** на момент этого snapshot |
| OA8 | **NOT_RUN** на момент snapshot: включение подтверждается только live runtime/image/public checks после обоих PASS; текущая production настройка не изменялась этим report |

Локальные logs: `/tmp/grace-owner-activation-test.log`,
`/tmp/grace-owner-activation-focused.log`, `/tmp/grace-owner-activation-bank.log`,
`/tmp/grace-owner-activation-integration.log`,
`/tmp/grace-owner-activation-production.log`, `/tmp/grace-owner-activation-vet.log`.
Неполный log не считается PASS; окончательные результаты перед merge проверяются
по exit status и exact-head CI. Исходные private PDF/тексты/ключи/generation IDs/
production cost/remaining не публикуются.

## Граница release

Ровно один live smoke разрешён этим решением. При FAIL новый платный прогон
требует нового решения; оставшиеся calls не используются как «повтор до PASS».
При PASS возможен ранний owner-test release по явно принятому amendment.
S2, S3/S4, AC12/Jev и draft SPEC-0056 не получают qualification/acceptance от
этого двухисходного smoke. Исторические FAIL и их protected audit сохраняются.
