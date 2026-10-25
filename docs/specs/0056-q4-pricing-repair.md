# SPEC-0056 R: ограниченный pricing repair и один Q4.2

- Статус: **accepted** до production code; условное разрешение на полный путь до Q4.2.
- Implementation status: **integrated_runtime_measured**; [evidence](0056-q4-pricing-repair-evidence.md).
- Дата: 2026-10-25, Asia/Yekaterinburg.
- Основание: «приступай, мне нужен прогон» после VERIFIED read-only диагноза Q4.
- База: integrated main `9200b731a0dea8f171e31aaf988d038073ad577f`.
- Родитель: [accepted SPEC-0056 Q1–Q3](0056-openrouter-document-qualification.md).
- [Q4 и read-only диагноз](https://github.com/QahNaaRlN/grace/pull/125):
  run40186488958/attempt1 INCONCLUSIVE / CONSUMED,0POST/3GET/$0.

## Результат и граница

Довести тот же bounded synthetic campaign до попытки настоящего распознавания,
не ослабляя верхние расходы и не выдавая metadata omission за известную цену0.
Старые marker/report сохраняются; новый запуск имеет собственную identity.
Успешный transport не равен quality PASS; даже PASS_SYNTHETIC не активирует
production профиль и не разрешает финансовый импорт.

## R1. Явный research-only request cap при отсутствующей цене

Добавляется optional GroupMembership `requestFeeEvidence`:
- пустое/отсутствующее значение — прежний strict режим, требуется явный request rate;
- `router_ceiling_only` — только isolated experiment groups с действующим grant,
  при положительном RequestFeeMicros; обычный production NewModelGroups отвергает режим.
- неизвестные значения и cap≤0 отвергаются до secret access.

При `router_ceiling_only` допускается **только отсутствие JSON ключа request**.
Это означает «цена не подтверждена metadata; принят операторский потолок,
обеспечиваемый max_price», а не «комиссия равна0». Нулём поле не заполняется.
Explicit null/empty/nonstring/negative/nonfinite/malformed rates отвергаются.
Если request rate присутствует, он обязан пройти обычную numeric ceiling проверку.

RequestFeeMicros всегда целиком добавляется в каждый reserve независимо от
наличия metadata rate и передаётся в `provider.max_price.request`.
Image pricing остаётся явным/проверяемым: missing/null/invalid image price
по-прежнему запрещены, цена обязана укладываться в ImageFeeMicros и в wire cap.
Default strict и production admission поведение не меняются.

Это явное изменение evidence-требования только для research. Оно полагается
на документированный server-side max_price; оно не является независимым
подтверждением отсутствующего request fee. Выход за cap/unknown billing по-прежнему
останавливает кампанию и сохраняется в shared ledger.

## R2. Candidate и консервативные резервы

Model `google/gemini-3.1-flash-lite`, canonical proposal
`google/gemini-3.1-flash-lite-20260507`, endpoint `google-vertex/eu`, provider Google.
Token ceilings прежние: input275000/output1650000 USD micros/M ($0.275/$1.65).
Новые явные ceilings:
- RequestFeeMicros=1000 ($0.001/request), целиком резервируется даже при omission;
- ImageFeeMicros=1 ($0.000001/image), до4 изображений, дополнительно к image token bound.

Из сохранённого неизменного report, отдельным read-only вычислением:
image rate — валидная positive decimal string, округление вверх до USD micros=1;
request key отсутствует. Это snapshot старого запуска, не fresh metadata нового.
Семантика image по official schema — USD/input image; image_token — отдельное поле.
Консервативный double reservation image tokens и image price допустим,
недоучёт либо молчаливая подмена отсутствующего rate запрещены.

При30000 input/16000 output:
`34650 + 1000 + 4*1 = 35654 USD micros/call` ($0.035654).
Восемь maximal reserves≤285232micros ($0.285232), что ниже прежних $0.25/call и $2 total.
Actual request/purpose reservations нужно воспроизвести offline на всех8cases.
Доступность реального shared allowance проверяется штатным preflight;
новых budget rows/пополнений/parallel off-ledger расходов нет.

## R3. Новая one-shot identity

Сохраняются тот же corpus/order8, source bytes, golden, scorer (full32297checks),
prompts/schema, resolver identity/ZDR/region/context guards, transport и ledger.
До live source получает CI5/5; новый receipt связывает exact source/manifest,
данный accepted контракт и условное разрешение владельца на Q4.2.

Новый fixed marker `/opt/grace/shared/document-reading-spec0056-q2.started`;
новый private report `/opt/grace/shared/document-reading-spec0056-q2.private.json`.
Общий lock прежний `/opt/grace/shared/statement-screening.lock`.
Staging source-specific `/tmp/grace-document-reading-q-<new-exact-source>`.
Старые Q4 marker/report не читаются для reset и не удаляются.

Один фактический workflow_dispatch через branch ref, закреплённый на approved SHA,
attempt1. Никаких rerun, повторов POST, provider/model switch или fallback.
Лимиты≤8POST/6metadataGET/$2.00 total/$0.25 call, HTTP180s остаются прежними.
Proof≤24h, default execute=false; source ancestry main, CI5/5, manifest/receipt binding.
После первого decisive quality REJECT/timeout/unknown bill/identity/privacy/price/
settlement failure — stop, остальные not_sent. Отказ preflight расходует допуск.

Один dispatch использует прежние dedicated production secrets только внутри CI:
≤2 pinned root SSH command sessions и≤1 SCP staging; ConnectTimeout15s,
ServerAliveInterval30s/CountMax60, job40min/remote1810s/campaign30min/grant35min.
Нет новых keys/secrets, deploy, image/config/service changes, рестартов, migrations,
финансовых/пользовательских SQL чтений или writers. Raw report/replay остаются VPS;
наружу только bounded public summary, staging удаляет cleanup, marker сохраняется.

## Затрагиваемые файлы после принятия

- internal/assistant/model_groups.go и причинные tests: evidence mode,
  request field presence/type и admission; общий wire/ReservationMicros контракт прежний.
- internal/documentqualification/{corpus.go,bundle.go,campaign_test.go} и
  testdata/candidate.json: новый optional mode, caps и offline/negative controls.
- deployments/production/document-reading-eval.sh и соответствующие fake controls:
  только новый q2 marker/report basename, общий lock и no-clobber сохранены.
- .github/workflows/document-reading-eval.yml: обозначение Q4.2, default/gates прежние.
- Документы accepted receipt, evidence и source/manifest/rate/transport record.

Product API/OpenAPI/web/financial core, prompts/schema/scorer/golden, существующие
production groups и deployment config вне scope. Никаких побочных repairs.

## Исполняемые критерии приёмки

| AC | Evidence |
| --- | --- |
| R1 | Strict missing request отказывает; positive research cap-only + omitted key допускается; null/empty/nonstring/negative/over-cap/unknown-mode дают0POST; ordinary production config отвергает mode |
| R2 | Actual grouped wire передаёт request0.001/image0.000001, без fallback; reserve включает1000+4micros; image missing/overcap отказывает; old replay без нового mode сохраняет INCONCLUSIVE |
| R3 | Fake8 correct replies →PASS_SYNTHETIC; полный quality-negative/identity/privacy/billing stop сохраняется, snapshots/manifest/approval несовпадения дают0secrets/0POST |
| R4 | Fake wrapper/workflow подтверждает q2 no-clobber/общий lock/source attempt1/cleanup и сохранность старых marker/report; no real SSH/model calls в tests |
| R5 | make test, make test-integration, make test-production, go vet, git diff --check; local/source CI5/5; новый main exact-source manifest и receipt до dispatch |
| R6 | Один Q4.2 с public counters/cost/settlement/verdict; live quality честно PASS_SYNTHETIC/REJECT/INCONCLUSIVE, grant затем CONSUMED независимо от результата |

## Публичные основания семантики

Проверены2026-10-25:
- [Models pricing object](https://openrouter.ai/docs/guides/overview/models):
  request — fixed API request cost, image — input image cost.
- [Official Terraform schema](https://github.com/OpenRouterTeam/terraform-provider-openrouter/blob/main/docs/data-sources/model.md):
  image и image_token различаются; ставки строковые.
- [Provider selection / max_price](https://openrouter.ai/docs/guides/routing/provider-selection):
  request и image имеют собственные max_price ceilings.
- [Provider integration](https://openrouter.ai/docs/guides/providers/integration):
  provider pricing допускает omission неприменимых SKU; это не переносится
  автоматически на доказательство0 в legacy endpoint metadata.

## Принятое решение владельца

Принять этот конкретный R-контракт **до production code** и разрешить связанный
путь: ограниченная реализация → offline/local+CI gates → merge прошедшего repair →
новый exact-source/manifest receipt → один Q4.2 в указанных лимитах.
Если принято условное разрешение на весь путь, оператор фиксирует source/manifest
после CI без повторного запроса тех же полномочий. Изменение модели, caps,
corpus/prompts/scorer, лимитов либо расширение scope потребует нового решения.

## Приёмка владельцем до кода

2026-10-25 владелец выбрал **«Принять и довести до прогона»** в ответ на
конкретный контракт и вопрос о полном пути repair → тесты/CI → merge → один
Q4.2 на новом source/manifest, при прежней модели/8 документах/≤8POST/6GET/$2
и явных ceilings $0.001/request, $0.000001/image, включённых в reserve.
Это разрешает перечисленные файлы и операции после прохождения gates.
Receipt commit предшествует implementation diff; старый grant CONSUMED.
Повторные запросы тех же полномочий не нужны; изменение scope/модели/лимитов
или повтор Q4.2 после расходования потребует нового решения.
