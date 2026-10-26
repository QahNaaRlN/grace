# SPEC-0058 P: production verification

Текущее состояние: **deployed / product PDF flow verified with one manual correction**.
Release `db1d63ddc664cd11626ec71a9bf843aba55ab0c5`, [Deploy40311400541](https://github.com/QahNaaRlN/grace/actions/runs/40311400541).
Один разрешённый paid provider POST выполнен, grant **CONSUMED 1/1**. Raw OCR767/768, все даты/время/суммы/валюты/карты верны; одна пунктуационная правка через UI дала candidate768/768. Это не заявление о безошибочном OCR или photo readiness.

Принятый контракт: [SPEC-0058 P](0058-production-price-ceiling-candidate.md), acceptance `d89199b`, интеграция PR136 `d1c4adb` до изменения production code. Владелец отменил первоначальный отказ и явно выбрал принятие P/продолжение. Основная реализация SPEC-0058 интегрирована PR135 `c09027c`; rollout и один новый product call отдельно разрешены владельцем.

## P: исполнение критериев

- Явный plain opt-in, полный bounded request reserve и wire ceiling: `TestPlainProductRequestCeilingAdmissionAndWire` PASS. Один POST / 3 GET в fixture, request/image caps, fallback disabled, schema absent. Пропущенное поле с opt-in допускается; null/empty/number/negative/malformed/over-cap request и missing image отвергаются до POST.
- Другие профили/границы: `TestPlainRequestCeilingCannotExpandOtherProfiles`, `TestPlainMissingRequestPriceStillRequiresOperatorOptIn` и прежние `TestR*` PASS. Research mode не входит в product; product mode не входит в research, analysis, receipts или strict statement. Нулевой/превышенный ceiling и отсутствие operator opt-in отвергаются.
- `go test ./internal/assistant -count=1` PASS.
- `TMPDIR=/var/tmp/grace0058-tests make test` PASS (Go, 341 Web / 91 files, lint/FSD/format/generated API/build/Jev/corpus); внешний TMPDIR сохраняет прежние privacy guards.
- `git diff --check` PASS. CI/integration/Delivery/deploy/live result ещё pending; offline wire не называется production вызовом.

## Readiness до rollout

Текущий VPS использует release `ec1586100819d1bd77902e5a5d94d508cdc984af`, AI/statement flags true, password auth; grouped profile ещё не установлен. Опубликованное endpoint metadata выбранного Google Vertex EU: token rates $0.275/$1.65 per million, image $0.000000275, request key отсутствует. Положительный request ceiling $0.001 не является подтверждением request rate и целиком включается в reservation.

Read-only ledger snapshot: allowance 4900000 micros, usageCount 27, committed 145998 micros, unsettledCount 0. Allowance и прежние settlements не менялись. Подготовлена операторская конфигурация с сохранением прежних analysis/receipt memberships, statement only / plain / один member, 5 страниц, reserve 35655 micros, qualification на семь дней на основании прежнего PDF/replay и разрешённой product acceptance. Реальные фото не квалифицированы. До live вызова нужны exact image deploy, backup/readiness и обычный browser consent.

Приватные PDF, эталон, ответы, реквизиты и session/key material в отчёт не входят. Сохранение реальных финансовых операций не разрешено этим прогоном.

## A8 readiness repair

P implementation PR137 merged `aefed4f`; source CI40300480625/all5, exact-main CI40302245700/all5 and Delivery40303592691/all4 PASS. Local post-merge `make test`/diff-check PASS. Deploy40303810120 FAILED at old `grace-statement-admin -mode=check`: it checked retired text policy despite product using groups. Pre-deploy dump verified and encrypted off-site copy uploaded; migrations30/31 applied. API/Web rolled back to previous `ec158610`; healthy new PDF helper retained (no destructive schema rollback).

Accepted SPEC-0058 A8 requires readiness of the actual configured profile. Repair changes only check-mode dispatch when groups are explicitly configured: the same `NewModelGroups/Resolve(statement)` and reservation arithmetic as product; no provider, inference, consent, budget or database writes. Legacy no-groups check and qualify/policy/research modes retain their behavior. Fixed diagnostic codes distinguish group failures.

`TestGroupedReadinessUsesProductProfileWithoutInference` PASS: fresh metadata and positive ceiling accepted for plain; strict missing request price, missing image modality, no ZDR and expired qualification rejected, POST=0. `go test ./cmd/statement-admin ./internal/assistant -count=1` and full `make test` PASS; diff-check PASS. A separate read-only/non-root/cap-drop verifier on VPS with actual operator groups returned `plain_document_reading`, `spec0058-p-v1`, policyVerified=true, maximumMicros35655; 3 metadata GET/0 POST. Initial isolated probes used the internal backend network without egress; switching the probe to API's existing edge network resolved the probe's network error, no production network protections changed. Runtime performs its own fresh proof; the per-proof bound remains3 GET.

No paid product call yet. Deploy retry follows exact repaired commit's CI/Delivery; one PDF call permission remains unused.

## A6: настоящий product consent выявил JSONB/hash дефект до inference

Readiness repair PR138 merged `b59ceb6`; exact main CI40305434447/all5 и Delivery40307005484/all4 PASS. Deploy40307175906 SUCCESS: API/Web/PDF-helper exact `b59ceb6`, healthy, grouped readiness/public auth smoke PASS. Локальный post-merge make test/diff-check PASS.

Через production browser в действующей owner session загружен предоставленный пятистраничный PDF на соответствующий счёт. Все страницы визуально содержат текст. Product preview выбрал ту же модель/Google Vertex EU, 1 call/$0.0357 maximum, images-only. После просмотра всех страниц и согласия обычный UI получил отказ: source revision1, preview1, consent0, attempts0, calls0, usage0 у нового источника; глобальный usageCount27 и financial event/posting counters unchanged не изменились. Это NOT_SENT, не model quality REJECT и не потраченное разрешение на provider POST.

Локальное read-only воспроизведение сохранённого preview подтвердило причину: после JSONB persisted plan hash не совпадает; восстановление typed ModelPolicy serialization возвращает исходный hash. PostgreSQL меняет порядок ключей вложенного RawMessage. Однопольный SQL mock `{fixture:true}` скрывал дефект реального полного policy snapshot.

Исправление в принятом A2/A6 scope: только plain snapshot канонизируется перед preview/сравнением актуальной policy и после чтения из SQL. UseNumber сохраняет точные целые. Новый planner `document-pages.0058.v2` требует свежего плана/согласия; global Hash, strict legacy plans, старые записи и verdicts не переписываются. Реальные изменения содержимого по-прежнему отклоняются. Stale/changed preview сообщает о необходимости подготовить новый план.

`TestPlainSnapshotHashIgnoresJSONBKeyOrderWithoutRounding` PASS: перестановка nested keys не меняет canonical hash, число9007199254740993 не округлено, изменение значения меняет hash, invalid/non-object snapshot отклоняется. Усиленный `TestS58FivePageProductAttempt` с полным nested policy snapshot прошёл PostgreSQL: changed-price negative consent до POST, восстановленный неизменный план → consent → один fake call → idempotent replay; остальные S58 lifecycle guards сохранены. Полный `./scripts/test-document-reading.sh` PASS, live provider calls0. Private diagnosis содержит только локальные данные, они не публикуются.

Окончательные make test/CI/Delivery/deploy этого repair отмечаются далее. Разрешённый один реальный provider POST ещё не выполнен.

## Итог нового product run — 2026-10-26

Consent repair PR139 integrated `db1d63d`. Exact-main [CI40309616197](https://github.com/QahNaaRlN/grace/actions/runs/40309616197)/all5 и [Delivery40311263833](https://github.com/QahNaaRlN/grace/actions/runs/40311263833)/all4 PASS; local `TMPDIR=/var/tmp/grace0058-tests make test`/diff-check и полный document-reading SQL PASS. Deploy40311400541 SUCCESS, API/Web/PDF-helper работают на exact release, healthy; grouped readiness и public authentication smoke PASS.

Новый browser flow: тот же сохранённый PDF → новый planner0058.v2 → отображены пять страниц и model/provider/cost → явное согласие → один product inference → сохранённая таблица96 строк. Передавались изображения всех страниц, без native text/эталона/schema. Предыдущий отказ на v1 был до consent и provider call; новой серии или второго inference не было.

Billing: одна completed attempt, один completed call, одна settled AI-usage row. Reserve35655 micros, actual13741 micros ($0.013741 в точности ledger, округление вверх до integer USDmicros), input5595/output7395; неизвестных списаний0. Исходный response сохранён отдельно; budget allowance, ключ и предыдущие settlements не менялись. Разрешение P **израсходовано1/1**, автоматический повтор не открыт.

Private Go comparison нового неизменённого ответа с зафиксированным независимым local reference: 96 строк, **767/768 полей совпало**, header6/6. Единственное расхождение — пропущенный trailing ASCII hyphen в описании строки17. Даты и оба времени, обе signed amounts/currencies и все card suffixes совпали. Raw exact-match критерий не объявляется PASS: private `TestPlainPrivateSavedAnswer` честно сообщил одну description mismatch. Старый сохранённый research answer не использовался вместо нового ответа.

Дефис подтверждён визуально на исходной странице2 read-only PDF crop. В обычном UI исправлено одно описание и нажато сохранение; server candidate помечен manual. Deep comparison подтвердил только эту правку (raw+typed representation одного поля) и manual marker: остальные значения, header и immutable original model answer не изменились. После правки current candidate **768/768 + header6/6**. Reload сохранил96 строк и исправление. Затем Codex через делегированную owner session просмотрел5/5 страниц, сохранил техническую сверку текущего digest и проверил её после reload. Это browser execution агента, не независимое принятие эталона человеком.

Финансовый результат: publishedRows0, financialEventDelta0/postingDelta0. Семантические решения строк и запись реальных операций не выполнялись. Соответствующая часть A6 подтверждалась ранее только изолированным public fixture, не этим production документом. Реальная photo/scan quality A7 и независимое owner approval эталона остаются UNVERIFIED. Закрытая серия SPEC-0056 не переименована и не возобновлена.

Операторская доступность: statement/plain, тот же Gemini/Google Vertex EU, отдельный явный P request ceiling, срок квалификации **02.11.2026 14:59 Asia/Yekaterinburg**; дальнейшее продление требует новой операторской проверки, не происходит автоматически. Fresh proof по-прежнему не дольше24ч, per-proof metadata≤3GET; inference без fallback/retry. Analysis/receipt memberships сохранены, их новые live quality tests не выполнялись.

Rollback: предыдущий автоматический rollback был до появления plain data; после этого документа нужны сохранение schema/evidence, отключение Statement AI либо совместимая версия plain API. Migration31 down намеренно отвергает существующие plain sources; destructive rollback на production не выполнялся. Private config backup сохранён; временный metadata verifier удалён. PDF, response, etalon, generation/session/key data и реальные строки в GitHub не включены.
