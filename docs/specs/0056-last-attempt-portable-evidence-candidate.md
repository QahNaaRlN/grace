# SPEC-0056 V: portable evidence schema и последняя попытка S

- Статус: **accepted V1–V3**; implementation NOT_STARTED.
- Приёмка владельца 2026-10-26: «Принять V и выполнить последнюю попытку (рекомендую)».
- Прежние [S](0056-evidence-alignment-series.md)/[U](0056-q43-owner-activity-reconciliation-candidate.md) не расширяются молча.
- Основание: [Q4.4 evidence](0056-q44-execution-evidence.md),consumed2/3,remaining1,$2 total прежний.

## Что установлено

Q4.2 получил200/model output и protocol REJECT. После S1 min/max evidence bounds
Q4.3 получил400/INVALID_ARGUMENT; Q4.4 после удаления $defs/$ref снова400 с тем же
generic invalid argument, без fieldViolationPaths/generation/usage. Это доказывает,
что удаления refs недостаточно. Конкретный keyword/предел compiler не установлен.
V проверяет одну следующую compatibility гипотезу; PASS не обещается.

## V1. Shape в запросе модели, семантические bounds в Go

Из VisualSchema удалить **только добавленные S1** box-item minimum0/maximum1 и
quote/evidence/anchor minItems1/maxItems8. Сохранить исходные required/nullable/
field names/strict/additionalProperties=false и box ровно4numbers. Не возвращать
$defs/$ref. Сохранить DocumentPrompt v3 с fractional0..1/1..8quotes/нет цитаты→null.

Go Validate, scorer/oracle/goldens/corpus, actual page bounds, положительная
площадь, byte limits/1..8quotes/0..1coordinates, financial core/API/model/rates/
порядок8cases/критерии качества **не меняются**. Никакого рескейла/синтетических
quotes/переноса golden/эвристического ремонта ответа или удаления checks/issues.
Это изменение уровня generative schema, а не ослабление acceptance criteria.

Обновить causal tests: actual grouped wire несёт portable shape/strict/no fallback
и прежний semantic prompt; unchanged Go validator по-прежнему отвергает0..1000/
empty evidence. Recompute actual reservations после уменьшения schema, сохранив
прежние30k/16k/rates/35654call/285232campaign/общий$2. Не менять цены или лимиты.
Новая fixed identity q5/workflow Q4.5; q/q2/q3/q4 markers/reports сохраняются.

## V2. Только две заранее ограниченные policy reconciliations

Владелец отдельно допускает закрытие текущего **Q4.4** reserve32023micros с actual0
по documented HTTP400 policy, без individual metadata proof. Нужны sealed report/
marker/response hashes и exact one usage row, unchanged owner/policy/time/reserve,
HTTP400,error status INVALID_ARGUMENT, no generation/choices/usage/cost и source
без auxiliary plugins/fallback. Основание `owner_accepted_http400_policy`,
directMetadataVerified=false; не выдавать это за подтверждённый API bill0.

Q4.4 identities: sourcea8752028463d19351c9eddcb0ee40b71fe398b47,
manifest1639957c4398c15733643b315b7a185c82fc9fdfad2c981517c1cf2ad012f3fc,
approval1c4e940a90ed1b4586d0dea14d16312a4c060a6b,
report21d4d71e413bc3f8631d369db1ed344e7f9fc6f13c210cc58a80f08a2a3b7963,
report filecb657a28762be9c9338c5f7871172d8f0a7d91470de22ebbba4d2098b070842d,
marker42e4e23a7c70d18cac6dd870b52d540a6041ce3d48e7941d2ab559e449c726df,
responsec876b2c8e82268919c4060e1384c37f0a94af976a28f6039bc8cdb67dc71c13e,
window2026-10-25T19:15:05Z..19:16:01Z. Reservation UUID только из этого sealed report.

Разрешить≤1pinned root SSH для Q4.4 settlement. Если **последний Q4.5** вернёт
ровно тот же400/INVALID_ARGUMENT/no generation/no choices/no usage/cost/noaux
паттерн — разрешить ещё≤1SSH **только** для его единственного uncertain reserve,
source/manifest/approval строго из отдельно опубликованного immutable Q4.5
receipt, report digest из validated public summary, один unknown reserve из sealed report. Другие errors (timeout/5xx/200unknown и т.п.)
не получают policy0; при них stop/unknown без settlement. Новых metadata GET0.

Каждая операция: fixed files, operator+budget+one-row locks, tenant context owner
только из bound row, existing grace_ai_settle0 с отдельным policy/report marker,
precommit snapshot/row checks/rollback, idempotency/conflict guards/read-after-write,
отдельный private0600 audit before/after/committed/basis. Только эта/эти usage row;
никаких functions/migrations/allowance/другихusage/financial/config/key/deploy writes.
Private raw/IDs/UUID не выгружаются; наружу closed status/cost/hash/flags.

Original unknown summaries неизменны. Administrative settlement помечается отдельно;
**вся освобождённая сумма остаётся contingency в прежнем$2**:30329+32023=62352micros
после Q4.4. Последний next max285232:0+62352+285232=347584micros<$2.
Если policy0 нужен и Q4.5, его reserve тоже добавляется в contingency, не исчезает
из conservative accounting. Прежний Q4.2 bill не переучитывается.

## V3. Только последний слот

После VERIFIED_OPERATION Q4.4 reconciliation и всех gates — ровно один Q4.5,
**попытка3из3**, та же Gemini/canonical/provider/8 документов/критерии;≤8POST/6GET,
прежние $2 и все S time/privacy/source/manifest/receipt/count1/attempt1 gates.
Acceptance commit V до production/ops code; local make test/production/integration,
vet/diff, exact PR headCI5/5, clean integrated source/mainCI5/5 перед dispatch.
Offline real SQL/owner/conflict/replay/rollback/privacy tests обязательны до ops.
Не менять corpus/oracle/scorer/financial state. No deploy/activation/import.

Стоп на actual PASS_SYNTHETIC8/8 либо после единственного принятого dispatch/
инфраструктурного/финансового отказа. После Q4.5 слотов0 независимо от результата;
V не разрешает новую серию, другой model, retry/rerun или четвёртую попытку.
Полный результат и accounting публикуются честно; неудача не пересчитывается в PASS.

## Принятое решение

Владелец принял V1–V3 целиком: один bounded compatibility repair, указанный policy accounting
текущего400/при точно таком же исходе последнего run и последний Q4.5 в прежних3/$2.
Новое принятие требуется, потому что S1 требовал эти bounds именно в generative
schema, а U разрешал только reconciliation Q4.3. V принят до production/ops implementation; повторное разрешение этого пути не требуется.
