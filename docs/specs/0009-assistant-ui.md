# SPEC-0009: single-turn интерфейс read-only ассистента

- Статус: draft
- Автор: Codex
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: отсутствует
- Specification branch: `codex/spec-0009-assistant-ui`
- Implementation branch: отсутствует
- Implementation commit: отсутствует

## Проблема и пользовательский результат

SPEC-0008 интегрировала безопасный single-turn endpoint
`POST /api/v1/assistant/query`, но обычный пользователь может вызвать его
только вручную через HTTP. В Web нет поля вопроса, понятного результата и
объяснения состояний `clarify`, `unsupported` и `unavailable`.

Пользовательский результат SPEC-0009 — отдельная Web-страница «Ассистент», на
которой пользователь задаёт один вопрос на русском языке и получает
структурированный read-only результат Grace. Интерфейс не изображает свободный
чат: он не хранит историю, не генерирует финансовые советы и не позволяет
изменять данные.

## Scope

- добавить маршрут `/assistant` и видимую навигацию между финансовой картиной
  `/` и ассистентом;
- добавить single-turn форму с textarea, submit по кнопке и ограничением
  принятого API `1..2000` UTF-8 bytes;
- отправлять `language: "ru"` и текущий browser-local `referenceTime` в RFC
  3339 с явным UTC offset;
- вызывать только существующий `POST /api/v1/assistant/query` через
  `openapi-fetch` и generated TypeScript contract;
- отображать четыре варианта `completed`: accounts, obligations, receipts и
  period since latest confirmed income;
- отображать локализованные, детерминированные состояния `clarify`,
  `unsupported`, `unavailable`, validation и network failure;
- показывать рядом с формой постоянное privacy disclosure: текст текущего
  вопроса, язык и reference time передаются TypeSafe, финансовые записи и
  результаты tools — нет;
- исправить OpenAPI discriminator mappings, чтобы generated TypeScript
  literals совпадали с фактическими wire values;
- добавить Web interaction tests и включить их в обязательный `make test`;
- выполнить визуальную проверку desktop и mobile layout.

## Non-goals

- изменение Go router, question set v3, dataset v2, model, threshold или
  compatibility matrix SPEC-0008;
- model-generated текст, советы, прогнозы или пересчёт финансовых значений в
  Web;
- история диалога, threads, follow-up state, streaming или Markdown output;
- сохранение utterance/result в PostgreSQL, localStorage, sessionStorage или
  URL;
- отправка предыдущих вопросов или финансовых данных provider;
- write tools, подтверждение AI-originated writes и любые формы изменения
  данных на странице ассистента;
- извлечение произвольных дат, receipt limits или income category из текста;
- автоматическая повторная отправка после `clarify`;
- английский UI или выбор языка в первом срезе;
- auth, multi-user isolation, внешний rollout или юридическое утверждение о
  zero-data-retention;
- iOS/SwiftUI клиент;
- изменение PostgreSQL schema или migrations.

## Текущее поведение и доказательства

- SPEC-0008 имеет статус `integrated` и endpoint default-off;
- OpenAPI уже описывает request и все runtime response shapes;
- generated TypeScript сейчас содержит неверные discriminator literals:
  `"AssistantCompleted"`, `"AssistantClarify"`, `"AssistantUnsupported"` и
  имена schema для `data.name`, тогда как runtime возвращает `completed`,
  `clarify`, `unsupported` и реальные tool names;
- причина — `discriminator` без explicit `mapping`: установленный
  `openapi-typescript` выводит значение из имени referenced schema;
- Web имеет только маршрут `/`; `AppLayout` не содержит навигации;
- в Web нет test runner и interaction tests, `make test` выполняет только
  ESLint и production build для клиента;
- existing UI уже имеет presentation patterns для money, accounts, receipts,
  obligations и period summaries, но они локальны для `HomePage`.

## Предметная модель и инварианты

`AssistantSubmission` существует только в памяти текущей страницы:

- `utterance`: trimmed UTF-8 string, 1..2000 bytes;
- `language`: константа `ru`;
- `referenceTime`: момент submit в локальной временной зоне браузера с явным
  offset.

`AssistantViewState`:

- `idle` — вопрос ещё не отправлялся;
- `submitting` — один request выполняется, повторный submit заблокирован;
- `completed` — отображается typed financial result;
- `clarify` — отображается локальный текст по `clarificationCode` и список
  требуемых аргументов, если он есть;
- `unsupported` — объясняется read-only граница;
- `unavailable` — ассистент выключен или временно недоступен, финансовое ядро
  продолжает работать;
- `invalid`/`network_error` — client validation или transport failure.

Инварианты:

1. Web не принимает решений о маршруте и не повторяет confidence threshold.
2. Web ветвится только по generated discriminators `status` и `data.name`.
3. Денежные значения форматируются из integer minor units и явной currency;
   арифметика финансового смысла не переносится в React.
4. Новый submit заменяет предыдущий результат; история не накапливается.
5. Utterance отсутствует в query keys, URL, storage, console и telemetry.
6. Компонент не выполняет background refetch, retry или submit при mount.
7. В каждый момент активен не более чем один assistant request.
8. Unmount/navigation отменяет request через `AbortSignal`, если transport это
   поддерживает; поздний response не меняет размонтированный UI.
9. `clarify`, `unsupported` и `unavailable` никогда не рендерятся как
   `completed` и не инициируют второй API call.
10. UI не изменяет и не скрывает `route`/`confidence` contract, но не использует
    их как основание для исполнения; compact technical details могут показать
    их пользователю справочно.
11. Пустые result arrays имеют явное нейтральное empty state, а не ошибку.
12. Страница `/` и все существующие формы продолжают работать без assistant и
    без TypeSafe key.

## API и OpenAPI

Новый HTTP endpoint не добавляется. Request остаётся:

```json
{
  "utterance": "Покажи мои счета",
  "language": "ru",
  "referenceTime": "2026-10-05T18:30:00+05:00"
}
```

В оба существующих `oneOf` discriminator добавляется explicit mapping:

- `status`: `completed`, `clarify`, `unsupported` → соответствующие schemas;
- `name`: `get_accounts`, `get_obligations`, `get_receipts`,
  `get_period_since_income` → соответствующие result schemas.

Wire contract и Go JSON не меняются. После `make generate-api` generated types
обязаны содержать фактические literals. Ручные дублирующие DTO, type assertions
для обхода generated contract и `any` запрещены.

Для удобства импорта `web/src/shared/api/types.ts` может экспортировать aliases
на generated `AssistantQuery`, response variants и tool results; источником
контракта остаётся `api/openapi.yaml`.

## Данные, миграция и rollback

PostgreSQL schema, migrations и server-side storage не меняются. React state
живёт только до reload/navigation.

Rollback удаляет `/assistant`, навигацию, Web tests и explicit discriminator
mappings. Endpoint SPEC-0008 и financial data остаются без изменений. Если UI
нужно временно скрыть, router по-прежнему может быть server-side default-off;
отдельный клиентский feature flag в этом work item не вводится.

## UI

`AppLayout` получает простую доступную навигацию:

- «Финансы» → `/`;
- «Ассистент» → `/assistant`;
- активный пункт определяется по текущему route и имеет не только цветовой
  признак, но и `aria-current="page"`.

Страница ассистента содержит:

1. Заголовок «Финансовый ассистент» и короткое read-only пояснение.
2. Постоянный privacy disclosure до submit.
3. Labelled textarea с примерами допустимых вопросов, счётчиком UTF-8 bytes и
   кнопкой «Спросить Grace».
4. Pending state с заблокированной кнопкой и доступным `aria-live` статусом.
5. Один result region, который заменяется следующим ответом.

Presentation completed results:

- accounts — имя, тип, currency и balance;
- obligations — дата состояния, имя, status, paid/remaining и ближайшие
  payments;
- receipts — seller, issued time, total/currency и позиции;
- period since income — anchor date/category/postings и canonical period
  summary по currency, включая opening/closing, income, purchases, refunds,
  debt payments и net external cash flow.

Тексты `clarificationCode` задаются закрытой таблицей в Web. Required arguments
переводятся на пользовательские названия (`from`, `to`, `through`, `asOf`,
`limit`, `categoryId`). Неизвестное значение невозможно по generated enum, но
runtime fallback обязан показать безопасное общее уточнение, а не упасть.

Страница должна работать при ширине 390 px без горизонтального scroll и при
1280 px без чрезмерно растянутых строк. Keyboard submit поддерживает
`Ctrl+Enter`/`Cmd+Enter`; простой Enter создаёт новую строку. После response
focus не перемещается неожиданно, result announcement доступен screen reader.

## Ошибки и негативные сценарии

- blank/whitespace-only или >2000 UTF-8 bytes: локальная ошибка, API не
  вызывается;
- двойной submit во время pending: только один request;
- `422 validation_error`: «Проверьте текст вопроса и повторите»;
- `503 assistant_unavailable`: объяснить, что ассистент выключен или временно
  недоступен; не объявлять недоступным весь Grace;
- `503 dependency_unavailable`: сообщить, что финансовые данные временно не
  удалось получить;
- network/cancel: безопасное локальное сообщение без raw response;
- `clarify/low_confidence`: предложить сформулировать один конкретный вопрос;
- `clarify/arguments_required`: перечислить необходимые даты/аргументы;
- `clarify/income_category_required`: объяснить, что named income category пока
  нельзя безопасно сопоставить;
- `clarify/no_income_anchor`: сообщить, что подтверждённый доход не найден;
- `clarify/composition_mismatch`: общее безопасное уточнение;
- `unsupported`: объяснить, что этот срез только читает поддерживаемые
  финансовые данные и ничего не изменяет;
- empty accounts/obligations/receipts/period currencies: typed empty state;
- malformed unexpected response: общее «Не удалось обработать ответ» без
  отображения raw body;
- TypeSafe key никогда не появляется в Vite env или browser bundle.

## Критерии приёмки

- [ ] `/assistant` доступен из общей навигации; `/` не регрессирует.
- [ ] Форма отправляет trimmed utterance, `ru` и browser-local RFC 3339
      reference time с explicit offset.
- [ ] Blank, whitespace-only и >2000-byte input не вызывают API.
- [ ] Pending state исключает двойной submit; navigation отменяет request.
- [ ] Все четыре completed tool results имеют typed presentation и empty
      states без финансовой арифметики в React.
- [ ] Все принятые clarify codes, unsupported, оба unavailable codes, 422,
      network и malformed response имеют безопасный локализованный UI.
- [ ] Новый submit заменяет результат; utterance/history не попадают в URL,
      storage, logs или query cache.
- [ ] Privacy disclosure виден до отправки и точно описывает data flow
      SPEC-0008.
- [ ] OpenAPI discriminator mappings генерируют реальные wire literals для
      `status` и `data.name`; ручных DTO и `any` нет.
- [ ] Assistant UI не содержит write controls и вызывает только один
      allowlisted endpoint.
- [ ] Interaction tests доказывают request payload, completed variants,
      clarify/unsupported/unavailable/error states, validation, double-submit
      protection и отсутствие automatic request on mount.
- [ ] `make test` включает Web tests, ESLint, TypeScript и production build.
- [ ] Visual evidence подтверждает layout и keyboard flow при 390 px и 1280
      px.
- [ ] `make test`, `make test-integration` и `git diff --check` проходят до и
      после merge.

## Исполняемые проверки

```bash
make generate-api
rg -n 'status: "completed"|status: "clarify"|status: "unsupported"' \
  web/src/shared/api/generated/schema.d.ts
rg -n 'name: "get_accounts"|name: "get_obligations"|name: "get_receipts"|name: "get_period_since_income"' \
  web/src/shared/api/generated/schema.d.ts
cd web && npm test -- --run
cd web && npm run lint
cd web && npm run build
make test
make test-integration
git diff --check
```

При implementation создаётся deterministic fake submit adapter для component
tests; live TypeSafe key и network для Web tests не требуются.

## Открытые решения

Для принятия owner подтверждает весь пакет следующих решений:

1. отдельная страница `/assistant`, а не ещё одна большая карточка на `/`;
2. single-turn replacement UI без визуальной истории сообщений;
3. русский язык без language selector в первом срезе;
4. постоянное privacy disclosure без отдельного consent checkbox для текущего
   owner-only local deployment;
5. добавление Vitest, Testing Library и jsdom как dev dependencies и запуск
   Web tests из `make test`;
6. исправление только OpenAPI discriminator mappings без изменения wire API;
7. compact route/confidence details являются справочной presentation и не
   повторяют backend policy;
8. внешний/multi-user rollout остаётся заблокирован отдельным privacy/legal и
   auth work item.
