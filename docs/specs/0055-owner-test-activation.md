# SPEC-0055: S1 activation smoke для тестов владельца

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Статус: **accepted**, 2026-10-19; прямой ответ владельца «Утвердить реализацию, один smoke и включение при PASS».
- Implementation: **in_progress**. Разрешён ровно один smoke по указанным ниже границам; повтор после FAIL требует нового решения.
- Основание: цель владельца — «Достигнуть включения Statement AI для моих собственных тестов в production среде»; [accepted S1](0055-record-atomic-planning.md), [S1 implementation](0055-s1-evidence.md), [existing activation gate](0055-activation-evidence.md).
- Baseline: main `8ed8e80a591c0696fd79e91de7a1d91ed9979917`, integrated PR #70. S1 не менял старый qualification cap; два новых plans сейчас получают `qualification_cap_exceeded` до reservation/call.

## Решение, предлагаемое владельцу

Утвердить ограниченное обновление существующего activation smoke под S1 и
**один** production smoke из двух поставляемых synthetic PDF `columns` и `cards`.
При полном PASS обоих cases, успешных CI/Delivery/deploy/runtime checks включить
существующий pinned Statement AI для собственных тестов владельца. При любом
FAIL флаг остаётся false, reports сохраняются, новый paid run автоматически не
начинается. Разрешение не переносится на повтор после исправления ошибки модели.

Это release для owner testing по существующему SPEC-0055 activation gate,
не `qualified` запись для draft SPEC-0056. AC12 representative benchmark,
Jev comparison, S2 coverage protocol и S3/S4 corpus/campaign сохраняют свои
отдельные статусы и не объявляются пройденными. Порядок S1–S5 из design study
остаётся планом полной qualification; владелец отдельно разрешает ранний
owner-test release после обоих обязательных synthetic checks. Ранее известный
cards FAIL сохраняется как исторический результат, новый PASS его не стирает.

## Точный scope и бюджет

| Параметр | Контракт одного разрешаемого запуска |
| --- | --- |
| Corpus / порядок | Только image-shipped public synthetic `columns.pdf`, затем `cards.pdf`; exact hashes входят в report |
| Model / endpoint | `google/gemini-2.5-flash` / `google-vertex/eu`, Google; без альтернативных routes |
| Attempts | Один проход по каждому case; **максимум 2 model calls всего**, без retry/repair/fallback |
| Input / output / bytes | S1 100000 / 16000 tokens; 90000 messages+schema bytes, 2000 spans; preflight обоих до reservation |
| Provider ceiling | **70000 micros ($0.07) / call; 140000 micros ($0.14) на весь smoke** при прежних price ceilings |
| Общий бюджет | Прежний allowance/fees; whole-smoke reservation до первого call, отказ при недостатке остатка или любой uncertain usage |
| Time bounds | 120 s / model call; 300 s / весь CLI run, включая helper/metadata/scoring/settlement; bounded persistence после cancel |
| Private input | Личные документы/PDF/text/IDs владельца не используются в smoke |
| Writes | Только существующий usage/reservation audit и после полного PASS operator flag/policy; нет imports/drafts/financial events от smoke |

Суммы выше — публичные теоретические ceilings; фактический production расход,
remaining, token counts и generation IDs остаются в protected VPS report mode
0600. В Git/CI только fixed error codes, case names и PASS/FAIL. Shared ledger
сам применяет существующие fees; allowance и fee policy не изменяются.

## Изменения реализации после принятия

1. `statementqualification.Run` сохраняет exact two-case allowlist, preflights
   весь smoke и использует S1 bounds: ≤2 calls / ≤140000 micros вместо старых
   ≤4 / ≤32000. Ни параметра с произвольным cap, ни произвольного corpus нет.
   `context.WithTimeout` вызова — 120 s; административный CLI — 300 s.
2. Report следующей версии связывает exact fixture hashes, current planner /
   prompt/schema/redaction/alias versions, limits и policy. Summary отказывает
   старому, неполному, подменённому или несовместимому report. Только два PASS
   с обязательными scorer checks допускают operator activation. Report не
   объявляет общую model qualification и не сохраняется в registry.
3. Для каждого response те же pinned privacy/no-thinking/text-only checks,
   canonicalization и Go validation. Known billing сохраняется при content/
   coverage FAIL; unknown/invalid billing settles uncertain и останавливает run.
   Отсутствующий, дублированный или неверный source alias не материализует draft.
4. Существующий independent scorer обязателен: rows/count, operation/booking
   clocks, account/FX amounts/currencies, direction, printed turnovers, complete
   coverage и distinction available/book balance. Нельзя объявить остаток service
   или ослабить критерии ради PASS. Частичный PASS не включает flag.
5. `statement-ai.sh enable` после successful smoke/summary получает проверенную
   policy и применяет конфигурацию существующим transactional deploy/rollback.
   Idempotent enable при уже валидном runtime не делает новых calls. При отказе
   deploy сохраняет прежние runtime/persisted flags. Disable не требует inference.
6. Production prerequisite check использует ceiling всего smoke, а не одного
   call; вывод остаётся boolean/fixed codes. Политика и capability повторно
   проверяются перед запросом, без предположения о свежести операторской даты.

## Проверки и completion evidence

| ID | Обязательное доказательство |
| --- | --- |
| OA1 | Fake-provider tests: два plans / одна полная reservation / ≤2 calls; input/call/aggregate overflow и недостаток бюджета → ноль model calls |
| OA2 | Timeout tests используют bounded contexts; second call не стартует после cancel/first FAIL; known/unknown billing корректно settle без скрытого retry |
| OA3 | Exact baseline scorer обоих fixtures: PASS; deliberate missing coverage, wrong row/currency/FX/date/available-book, unknown alias → FAIL, ноль financial writes |
| OA4 | Report/summary schema/version/hash/missing-case/tampering negatives; старый report не разрешает новое enable |
| OA5 | Shell activation mocks: quality FAIL не включает flag; deploy FAIL восстанавливает конфигурацию; repeat enable не расходует; disable работает |
| OA6 | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`; generated API unchanged unless contract changes |
| OA7 | Exact implementation SHA: PR/main CI и Delivery PASS; deploy с preserve false; один отдельно разрешённый live smoke обоих cases PASS; protected audit retained, no private output |
| OA8 | Runtime `statementAI:true` + `policyVerified:true`, exact deployed image SHA и публичный `/api/meta` подтверждают включение; protected routes 401 без session; manual import доступен |

Completion цели требует OA8 в реальном production. Код, зелёный CI, только
один case PASS или сохранённый flag без runtime проверки completion не доказывают.
Личная выписка остаётся действием владельца через новый preview/consent, а любая
финансовая публикация требует отдельного explicit confirmation в интерфейсе.

## Первичные источники / ограничения

[Gemini structured outputs](https://ai.google.dev/gemini-api/docs/structured-output)
и [OpenRouter structured outputs](https://openrouter.ai/docs/guides/features/structured-outputs)
проверены 2026-10-19: schema support не доказывает семантическую правильность
ответа exact endpoint. Поэтому runtime smoke/scorer остаются обязательными.
Документ не меняет response schema S1, не добавляет S2 или новые providers.
