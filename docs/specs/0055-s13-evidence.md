# SPEC-0055 / S1.3 — implementation evidence

Дата: 2026-10-20 (Asia/Yekaterinburg). Контракт:
[accepted S1.3](0055-s13-turnover-roles.md), acceptance commit `b94395e`
предшествует production-коду. Ветка `codex/statement-quality-next`, baseline
`d538b78`. Это локальная реализация, не deployment/owner financial acceptance.

## Изменение и критерии

| ID | Evidence |
| --- | --- |
| T1 | `TestTurnoverRolesPublicLayout`, `TestTurnoverRolesRealHelper`: публичный двухстраничный RU fixture → real Poppler layout → source-authored golden. 2 строки, точные descriptions/amounts/currency, turnover и coverage matched; book balances absent. 51 spans, S1.3 request 10057 bytes <90000, 51 aliases. PDF отрендерен, обе страницы просмотрены |
| T2 | Семь negative cases: identifier label, same-number identifier, borrowed label, reversed label, unknown label, missing label, other-page label. Полное coverage сохранено во всех; same-number case сохраняет точное равенство денег и не имеет printed_turnover_mismatch. FAIL требуется именно по turnover_role_unverified. Геометрия и explicit bank context дополнительно проверены в statementdata unit tests |
| T3 | `TestTurnoverRoleLabelsStayVisibleWithoutIdentifierValues`: новые фиксированные подписи видимы, numeric identifiers скрыты даже при explicit visibility; старые v11/v3 profiles invalid. Prompt v12 / redaction v4, без смены schema/planner/alias/limits/policy. Existing stale-consent/lexical/fake transport checks сохранены |
| T4 | `TestStatementTurnoverLegacyGuard`: новый manual draft reviewable с blocker; симуляция старого persisted review с matched totals и подтверждёнными ролями/страницами не проходит read/financial preview. Read guard не переписывает review в SQL, postings count не меняется. Retained-evidence unit tests работают без layout |
| T5 | [Metadata snapshot и shortlist](../research/statement-quality/model-shortlist-2026-10-20.md): baseline + Gemini 3.1 Flash Lite / Google EU + Mercury 3 / Inception. Decimal ceiling 53900 / 15600 micros; provider calls 0. Ни одна новая пара не qualified/allowlisted, actual quality/latency UNVERIFIED |
| T6 | Команды и итоговые результаты ниже; изменения не создают финансовые writes и не вызывают провайдер |

Роль оборота проверяется через поддержанные подписи, геометрию и явные подписи
реквизитов рядом с выбранным числом. Guard сохраняет исходные evidence и parsed
values, добавляя blocker и incomplete turnover status. Он не выбирает другой
источник и не исправляет ответ модели. Поддержанные RU/EN labels — ограниченный
набор, неизвестные layouts могут потребовать нового контракта. После удаления
полного layout невыбранный банковский контекст восстановить нельзя; retained
label/geometry checks сохраняются. Отсутствующий оборот не вычисляется из строк.
Web показывает понятное пояснение через прежний словарь issues; API не менялся.

## Команды и результаты

- `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test`:
  **PASS**: Go, Web 88 files / 334 tests, lint/FSD/format/build и Jev offline.
- `TMPDIR=/var/tmp/grace-quality-next-tests ./scripts/test-bank-statements.sh`:
  **PASS**: финальный повтор с усиленным coverage assertion, 84 planner/layout
  cases, columns/cards activation offline fakes, новый real-helper RU regression,
  lifecycle/SQL/restore и legacy role guard.
- `./scripts/test-statement-activation.sh`: PASS, offline kill-switch/rollback/
  replay/failed-qualification guards, без production действий.
- `go test ./internal/statementdata ./internal/statementextract ./internal/statementqualification`:
  PASS после финального Go review; `go vet ./cmd/... ./internal/...`: PASS.
- `git diff --check` и локальные documentation links: PASS.

Два обычных параллельных `make test` завершились FAIL на неизменённом
`web/src/app/router.test.tsx`: ожидания lazy headings с default Testing Library
wait (сначала 1, затем 2 assertions). Тесты/таймауты/маршруты не изменялись.
Последовательный запуск Vitest проверяет весь прежний набор, не исключает
упавший тест; его 334 assertions прошли. Эти FAIL не объявляются исправленными
S1.3. Начальный Go test получил output_inside_repository из-за `/tmp/.git`
окружения; TMPDIR вынесен в `/var/tmp`, production guard не ослаблялся.

Локальные журналы: `/tmp/grace-s13-make-test.log`,
`/tmp/grace-s13-make-test-final.log` (параллельные FAIL),
`/tmp/grace-s13-make-test-serial.log`, `/tmp/grace-s13-bank-tests-final.log`,
`/tmp/grace-s13-regression-final.log`, `/tmp/grace-s13-activation-tests.log`,
`/tmp/grace-s13-vet.log`. Они не являются GitHub CI или production evidence.

## Источники fixture и пределы

PDF SHA256 `648ac4477b9a78a7a1bb43eb9ae918fd5b6a2eb338e2ee07038146141781686f`.
Golden SHA256 `bd613c97dfc7b8b552fafad38d0a146fb0949e67712eb5724e9be99ce78f57c5`.
Генератор: `scripts/fixtures/generate-turnover-roles-pdf.py` (ReportLab, Noto
Sans, invariant output). `turnover-roles-ru.xml` — host Poppler offline snapshot;
отдельный integration test извлекает исходный PDF настоящим pinned helper,
не доверяя сохранённому XML как доказательству версии helper. Golden source
задан независимо от модели; service selections перечислены явно, не по остатку.
24-case исторический manifest не выдаётся за новый полный корпус; этот fixture
— дополнительный development regression, не holdout и не model qualification.

Новых платных calls/private inference нет, личный PDF не использован,
production release остаётся `ec15861`. Новая модель/новый prompt ещё не прошли
live smoke. Следующий run и его candidate policy должны быть отдельно согласованы;
[предложение](../research/statement-quality/model-shortlist-2026-10-20.md) содержит
3 public cases и provider ceiling $0.1617 для Gemini 3.1 Flash Lite / Google EU.
Никакого automatic enable/registry/financial acceptance по fake-provider PASS.
