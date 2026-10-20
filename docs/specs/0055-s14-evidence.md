# SPEC-0055 / S1.4 evidence

Дата: 2026-10-20, Asia/Yekaterinburg. [Контракт](0055-s14-candidate-screening.md)
accepted commit `a94cbb4` предшествует коду. Baseline `7c4ef83` включает локально
проверенную S1.3. Ветка `codex/statement-candidate-smoke`.

## Локальная проверка

| ID | Исполняемое evidence |
| --- | --- |
| C1 | `TestScreeningCannotAuthorizeProductConsent`: product Policy.Valid/BuildPlan/CanonicalProposal и persisted-plan guard отвергают candidate. Только exact bounded pair работает в BuildScreeningPlan; stale/tampered plan не проходит |
| C2 | `TestScreeningTransportIsExplicitAndPinned`: fake HTTP проверяет model/only EU/ZDR/deny/no-fallback/reasoning disabled/16000 output. Malformed answer сохраняет known billing; отсутствие privacy membership останавливает POST. Обычный reasoner и provider по-прежнему отвергают candidate |
| C3 | `TestScreeningRealHelperThreeCases`: pinned Poppler → columns19/19 + cards20/20 + RU23/23. Whole-run reservation161700, один settle300 synthetic micros, 3 calls. RU omitted description FAIL, без retry. Это fake-provider evidence, не измерение Gemini |
| C4 | `TestScreeningStopSettleAndPreflight`, `TestScreeningReportNeverActivates`: wrong order/hash/policy до reservation, known error billed, unknown freezes; report с другим version/profile никогда не разрешает activation. `scripts/test-statement-screening.sh`: failed preflight без marker; failed run сохраняет report600/marker и не повторяется, включая другой SHA; billing не попадает в stdout |
| C5 | PASS: `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` (Go + 88 web files / 334 tests + lint/FSD/format/build/Jev); `scripts/test-bank-statements.sh`; `make test-production`; `go vet ./cmd/... ./internal/...`; `git diff --check`; YAML/exclusive-job guards |
| C6 | Live smoke NOT_RUN; разрешён ровно один, 3-call maximum / provider cap161700; production adoption не разрешено |

Remote harness добавлен как explicit `statement_screening_only` в существующий
workflow. Deploy/inspect jobs взаимоисключены с screening. Exact-source CI обязателен
до запуска; binary/fixtures manifest проверяется на VPS; прежний API container/image
и health проверяются после, config/flags/drafts не меняются. Authorization marker
`s14-gemini31-20261020` общий для разных workflow attempts/SHAs; предварительный
отказ до его создания не считается платным выполнением. После marker нет paid
replay даже при interruption. Приватный report остаётся в `/opt/grace/shared/`,
временный bundle и container directory очищаются. Public logs только fixed codes,
check counts и activationAuthorized=false; usage/generation data остаются приватны.

Прямой SSH из локальной сессии недоступен (`Permission denied`); используется
existing GitHub production environment/key. Новые credentials не запрашивались.

Локальные журналы: `/tmp/grace-s14-unit-final.log`, `/tmp/grace-s14-go-final.log`,
`/tmp/grace-s14-bank.log`, `/tmp/grace-s14-make.log`,
`/tmp/grace-s14-production-tests.log`, `/tmp/grace-s14-remote-test.log`,
`/tmp/grace-s14-vet.log`. Они не заменяют exact-source CI или live evidence.

Полная qualification, model registry и product adoption остаются вне этого
пакета; даже полный 19/20/23 PASS не гарантирует все банки. Никакого automatic
production switch или финансового подтверждения по результату screening.
