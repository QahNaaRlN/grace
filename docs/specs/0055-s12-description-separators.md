# SPEC-0055 / S1.2: полное описание с печатными разделителями

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Статус: **accepted**, 2026-10-19; прямое решение владельца «Утвердить S1.2, один smoke и включение при PASS». Implementation **in_progress**, `codex/statement-description-separators`; принятие записано в `bf4b3c8` до production-кода.
- Основание: [S1.1 evidence](0055-s11-evidence.md), release `ef782ec`, один smoke `39255669019` FAIL. Columns: две строки, финансовые поля, обороты и coverage PASS; единственный code `record_description`. Cards NOT_RUN. AI false, разрешение на этот smoke использовано.
- Scope: только prompt clarification + negative regression; затем **один отдельно разрешённый** новый columns/cards smoke. Не разрешает повторов, model switching или расширение corpus.

## Подтверждённая причина и пределы вывода

Read-only `39255943568` вернул безопасные canonical locators без usage/values.
Настоящий pinned helper + source-hashed [offline replay](../research/statement-quality/s11-description-offline.json)
подтвердил: description первой строки exact; во второй отсутствует только
`p2-b8-l1-w3`, напечатанный дефис между словами. Этот span выбран в service.
Ничего не исправлялось в ответе модели для получения PASS. Scorer правильно
отклонил неполное описание. Numeric lexical eligibility не определяет роль;
дефис допустим и внутри описания. Почему модель сделала такой выбор, неизвестно.

Предлагаемая инструкция — проверяемая гипотеза, **semantic quality UNVERIFIED**.
Offline PASS доказывает диагноз и byte bounds, не поведение модели. Cards в S1.1
не вызывался; нельзя утверждать улучшение или regression этого формата.

## Конкретная реализация при принятии

В конец текущего SystemPrompt добавить ровно этот текст:

> Preserve all printed fragments of each transaction description in source order, including standalone punctuation separators such as plus or minus signs inside that description. Numeric lexical eligibility is not a financial role: a separator printed within description text belongs to that row.description, not service or a monetary field. Do not include neighboring amounts, dates, currency tokens or page titles in the description.

PromptVersion → `statement-span-mapping.v11.description-separators`.
Остальные planner/alias/redaction/schema/scorer/report versions сохраняются.
PlanHash/profile сравнения требуют нового preview и consent; старый report не
разрешает enable с новым prompt. Alias/tuple transport, visibility, money eligibility,
financial Go validation, description oracle и counts **19/20** не изменяются.
Нельзя автоматически дописывать пропущенную пунктуацию, ослаблять scorer,
нормализовывать дефис в отсутствие или собирать description за модель.

Добавить negative real-helper fake-provider regression: description пропускает
печатный separator и помещает его в service, полная source coverage и финансовые
поля сохранены; этот ответ обязан FAIL по `record_description` без второго call.
Существующие successful, title/swap/text-as-money/legacy/budget/confirmation
проверки сохраняются. Повторить actual planner proof на 84 cases с новым prompt
и хешами, `make test`, `make test-integration`, `make test-production`, `go vet
./cmd/... ./internal/...`, `git diff --check`; CI/Delivery точного release.

## Лимиты и единичный live gate

Прежние S1 bounds: 2000 spans / 90000 bytes; 100000 input / 16000 output;
120 s/call, 150 s/product attempt, HTTP 180 s. Offline candidate bytes:
columns **9060**, cards **9401** — измерены messages + current schema, не actual
token usage. Shared allowance, fees и pinned model/endpoint policy сохраняются.

Только при явном решении владельца разрешить **один новый** columns → cards
smoke: максимум **2 calls / $0.14 aggregate provider ceiling**, 300 s/run,
whole-run reserve, без retry/repair/fallback/личных документов/imports/events.
Enable только оба полных strengthened PASS и exact-image/runtime/policy/public
checks. Любой FAIL оставляет AI false и требует нового решения перед следующим
paid run. Прежние FAIL и private audit сохраняются; полная AC12/Jev/corpus
qualification и owner functional acceptance не объявляются выполненными.

## Критерии

| ID | Исполняемая проверка |
| --- | --- |
| D1 | Public-source replay подтверждает exact omitted separator в service; 0 provider calls/0 SQL, source hashes и обе byte bounds |
| D2 | Точное prompt amendment/version; preview/consent/profile stale rejection; без transport/privacy/financial/scorer changes |
| D3 | Real-helper missing-separator negative FAIL, complete baseline PASS; никакой автоматической починки |
| D4 | 84 actual planner cases, все локальные обязательные команды, exact-head CI/Delivery PASS |
| D5 | Отдельно разрешённый один smoke: оба 19/20 PASS и runtime/image/public enable evidence; иначе disabled |
