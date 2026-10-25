# SPEC-0056 S: evidence alignment и ограниченная серия до PASS

- Статус: **accepted** до production code.
- Implementation status: **not_started**.
- Дата:2026-10-25, Asia/Yekaterinburg.
- База: main5b10b2c, интегрированный Q4.2 evidence PR127; diagnosis [PR128](https://github.com/QahNaaRlN/grace/pull/128).
- Родители: [accepted G1–G4](0056-openrouter-document-reading.md), [Q](0056-openrouter-document-qualification.md), [R](0056-q4-pricing-repair.md).

## Приёмка до кода

Владелец ответил «согласуй, затем прогон до положительного результата» на
предложение согласовать Schema с валидатором и уточнить «нет цитаты → null».
Затем явно выбрал **«До3 попыток / $2 суммарно»** в ответ на фиксированные
условия: та же модель,8 документов и критерии качества, каждая попытка≤8POST/
6GET/$2, после REJECT — разбор и исправление только контракта evidence.
Это принятие указанного bounded repair и полного пути offline → CI → merge →
до3 новых exact-source кампаний без повторного запроса тех же полномочий.
Этот receipt commit должен предшествовать implementation diff.

## S1. Согласование evidence

VisualSchema сохраняет существующие поля/nullable structure/shape и явно
передаёт box items minimum0/maximum1, quote/evidence/anchor minItems1/maxItems8.
Box остаётся массивом ровно4numbers. Go Validate, DirectJSON и scorer не
ослабляются; actual page bounds, положительная площадь, byte/whitespace limits
по-прежнему проверяются кодом. Не все эти отношения выразимы portable JSON Schema.

DocumentPrompt/его версия получают явные fractional normalized координаты
[left,top,right,bottom] (left<right,top<bottom), не pixels/0..1000.
Непустое поле должно иметь1..8 настоящих page/quote/box evidence; нет цитаты →null.
Запрещены преобразование ответа по эвристической шкале, выдумывание цитат,
удаление model issues ради PASS или перенос значений из golden.
Содержательные правила суммы/даты/строк не меняются; receipts protocol прежний.

## S2. Проверка и границы файлов

- internal/statementdata/visual.go: только VisualSchema evidence constraints.
- internal/statementextract/document_reading.go: только DocumentPrompt/version evidence semantics.
- Причинные tests рядом: actual grouped wire содержит schema; 0..1000/empty
  evidence отрицательные controls; fractional/null/cited положительные; no fallback.
- internal/documentqualification/pricing_repair_test.go: пересчитать actual
  request reserves после роста prompt/schema, сохранив30k/16k/per-call/total caps.
- deployments/production/document-reading-eval.sh и fake workflow controls:
  новая fixed identity q3 (далее q4/q5 только если нужна разрешённая итерация),
  общий lock/no-clobber/cleanup/сохранность старых q/q2 и предыдущих files.
- .github/workflows/document-reading-eval.yml: название текущей попытки;
  source/main ancestry/CI5/5/receipt/manifest/attempt1/count1/default-false gates прежние.
- docs: acceptance, implementation evidence, series journal, immutable receipts/results.

Quality oracle/goldens/corpus/order/scorer/financial core/API/OpenAPI/web,
model/provider/canonical/caps/production groups/deployment config вне scope.
Итерации после REJECT могут менять только этот evidence contract и operational
identity. Если причина требует другой модели, иных criteria или соседнего
product repair — серия останавливается с evidence, не расширяется молча.

## S3. Серия и денежная граница

Новые slots Q4.3/Q4.4/Q4.5, максимум3 accepted dispatch, attempt1 каждый;
все прежние grants/markers/report остаются CONSUMED. Один run на один source;
source меняется только через проверенную итерацию/identity, без rerun workflow.
Каждая попытка:≤8POST/6GET/$0.25call/$2 campaign, HTTP180s;
та же Gemini3.1FlashLite / google-vertex/eu / canonical20260507;
input275000/output1650000micros/M, request1000/image1micros, image bound4200,
≤4images/2048px, input30000/output16000, router_ceiling_only только research.

Максимальный разрешённый reserve35654micros/call,8max285232;
3*8max=855696micros ($0.855696) ниже совокупных $2. Новые prompt/schema
должны укладываться в прежний token plan; фактические request reserves проверяются.
Это верхняя плановая стоимость при соблюдении rate/token caps, не гарантия
корректности стороннего счёта. Pricing/usage/billing guards остаются обязательными.

Перед каждым dispatch immutable receipt связывает source/manifest/CI/model/slot,
все предыдущие validated summaries серии, known cost и remaining $2.
Условие dispatch: нет unknown/failed settlements, предыдущие bills settled,
spent+next maximal reserve≤2000000micros и remaining slots>0.
После accepted dispatch slot CONSUMED независимо от preflight/transport результата.
Known spend суммируется по всем runs, unknown bill замораживает всю серию.
Первая price/token overrun, identity/privacy/billing/settlement failure закрывает
серию; нет reserve top-up, новых budget rows или off-ledger calls.
После quality REJECT допустим read-only разбор сохранённого report и bounded
repair evidence, затем все gates заново. Все неудачные результаты сохраняются.

Стоп на первом actual PASS_SYNTHETIC всех8cases либо при исчерпании3/$2,
инфраструктурном/финансовом отказе или выходе за evidence scope. Успех заранее
не гарантирован. Legacy readiness не пересчитывается; importReady=false.

## S4. Transport, результаты и приёмка

На run≤2 pinned root SSH command sessions +≤1SCP через существующие secrets,
те же timeouts40min/1810s/30min/grant35min/summary60s; staging source-specific.
Fixed new files `/opt/grace/shared/document-reading-spec0056-q3.{started,private.json}`
(аналогично q4/q5 при следующей итерации), shared statement-screening.lock.
Ни новые ключи, ни deploy/config/container images/restart/migrations не выполняются.
Private raw report/replay остаются VPS; наружу только validated public summary.
Для разбора после отказа≤2 дополнительных pinned read-only SSH sessions на
точно указанные сохранённые marker/report, bounded reader из stdin, без remote
persistent writes/SQL/key/env probes/API/model calls; только whitelist diagnostics.

Перед merge: make test, test-production, test-integration, go vet, diff check,
exact head CI5/5. Перед каждым live: clean integrated source CI5/5,
source-bound exported manifest и опубликованный receipt. Владелец уже разрешил
этот путь; дополнительные подтверждения тех же операций не нужны.

| AC | Evidence |
| --- | --- |
| S1 | Actual schema/wire содержит0..1 и quote count1..8; validator/scorer unchanged; обе причины Q4.2 сохраняют causal negative controls |
| S2 | Новый prompt/version, actual reserves≤plans; full fake8→PASS_SYNTHETIC, full32по297; stop/replay/forgery/privacy/billing controls прежние |
| S3 | Fake wrapper сохраняет старые consumed files; slot no-clobber и все source gates PASS; реальные SSH/provider в tests0 |
| S4 | make test/production/integration, go vet, diff check, PR/main CI5/5; exact source/manifest receipts |
| S5 | Series journal:≤3 attempts/$2, все summaries/verdicts/bills; PASS только actual8/8, иначе честный stop без автоматического расширения |

## Официальные основания

Проверено2026-10-25: [OpenRouter structured outputs](https://openrouter.ai/docs/guides/features/structured-outputs)
передаёт strict json_schema и требует endpoint support; [Gemini schema support](https://ai.google.dev/gemini-api/docs/generate-content/structured-output)
поддерживает number minimum/maximum и array minItems/maxItems. Это не заменяет
проверку actual wire/ответа и не доказывает заранее качество endpoint.
