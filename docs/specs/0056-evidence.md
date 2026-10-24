# SPEC-0056 — отчёт локальной реализации G1–G4

Дата: 2026-10-25, Asia/Yekaterinburg. Исполнитель: Codex.
Принятый контракт: [SPEC-0056 G1–G4](0056-openrouter-document-reading.md).
Implementation acceptance владельцем: **ACCEPTED**; merge: **INTEGRATED PR122**.

Владелец 2026-10-25: «Принять реализацию и слить №122», exact head
`b160af182760fbfde9e38da059f1f185f11c76dd`; source CI40131646353/5 PASS.
Merge/main `10a0059bb90265ea1b39a51f6f84037ebc4dd9c8`, tree равен принятому.
[Main CI40132897197](https://github.com/QahNaaRlN/grace/actions/runs/40132897197)
**5/5 PASS**, [Delivery40133674626](https://github.com/QahNaaRlN/grace/actions/runs/40133674626)
**4/4 PASS**, оба exact merge SHA. Delivery не является deploy.
Receipt и post-merge evidence: [PR122](https://github.com/QahNaaRlN/grace/pull/122).

Контракт принят прямым ответом «Принять контракт, интегрировать и реализовать»
до production-кода. PR121 интегрирован в main
`1fa1d93bc83e63ff8ddb0143bdf5849e32e6d98d`:
[main CI40124177563](https://github.com/QahNaaRlN/grace/actions/runs/40124177563)
5/5 PASS; [Delivery40124951783](https://github.com/QahNaaRlN/grace/actions/runs/40124951783)
4/4 PASS. Delivery не является production deploy.

Implementation commit: `32b0ae33ff53ede5a5fd551a68e9802ca9e885b1`
(основная реализация `f3476c759aba09e5dd04af9061cf5be8432e8354` и review fix).
Ветка/worktree: `codex/openrouter-document-reading-implementation` /
`grace-worktrees/openrouter-document-reading-implementation`.
Документационные уточнения после этого commit не меняют проверенный контракт;
remote CI оценивается по фактическому последнему head implementation PR.

## Полученный результат

Один production gateway OpenRouter для generative reasoning, receipt, statement
и quick router. Resolver выбирает конкретный квалифицированный endpoint из
версионированной группы по свежим capabilities/privacy/price доказательствам.
Выбор фиксируется до согласия и резервирования. Старые прямые OpenAI/TypeSafe
live entrypoints закрыты, offline history/replay сохранены.

Локальный PDF helper отдаёт изображения всех страниц и настоящий текст, если
он есть. Фото нормализуются локально. Выписки и чеки читаются в редактируемый
черновик с приватным источником рядом. Native PDF сохраняет spans/grounding;
фото/сканы имеют отдельный image_review без заявленного независимого OCR.
Сервер проверяет source/revision/digests, страницы, роли и прежние blockers.
Финансовые записи создаёт только прежний confirmation после решений владельца.

Все перечисленные model outputs — контролируемые fake-ответы. Реальное
качество модели, начальные production group members и rollout **UNVERIFIED**.
Новых provider/metadata live calls, чтения/настройки ключей, production
SSH/SQL/deploy, включения flags и изменения реального allowance не было.

## AC и исполняемое evidence

| AC | Проверка и actual результат |
| --- | --- |
| G1.1 | `go test ./internal/platform/config ./internal/platform/httpserver ./cmd/jev-eval ./cmd/statement-page-reading ./internal/assistant`: PASS. API composition использует grouped OpenRouter; TypeSafe adapters доступны только в `_test.go`; `jev-eval run` отказывает remote endpoint до key, page-reading main run закрыт до key. Endpoint inventory: `rg -n 'NewTypeSafeProvider|NewTypeSafeGroupSuggester|api.openai.com|typesafe.ai' cmd internal` различает закрытые entrypoints и исторические fixtures. |
| G1.2 | `go test ./internal/assistant -run 'TestDocumentGroups|TestModelGroups|TestGroupedTransport|TestGroupedPrePOST'`: PASS. Две eligible модели, deterministic выбор/qualification expiry, stale/forged proof, text-only/missing schema/wrong provider/ZDR/price/context, routing aliases и скрытый file input дают 0 POST. Pre-POST refusal settles0, без ложного unknown bill. |
| G1.3 | `go test ./internal/assistant -run 'TestGroupedDocumentWireAndIdentity|TestDocumentReservationIncludesFlatFees|TestRejectedOutputRetainsVerifiedBilling'`: PASS. Concrete/canonical/provider/attempt/pipeline/fees/token/cost guards; wire deny/ZDR/no-fallback. `TestS56StatementDocumentLifecycle`, `TestS56UnknownBillingFreezesAndDeletedSourceCannotRun` — реальная PostgreSQL: один call/replay, known settlement, unknown uncertain и 0 повторных вызовов/проводок. |
| G1.4 | `./scripts/test-document-reading.sh`: PASS для receipt grouped adapter+meter и `TestS56QuickRouterUsesConsentAndOneMeteredGateway`: до consent0 POST; после consent1 POST/1 read-only tool/1 settlement, confidence не выдумывается. Старый анализ и read-only tools проходят `make test`/`make test-integration`; shared wire тот же. |
| G2.1 | `TestS56RealRenderNativeAndScan` в `./scripts/test-document-reading.sh`: PASS для actual cards, columns, full32 и full32-scan. Scan: 3 PNG, native text отсутствует; legacy text extractor по-прежнему отказывает. Hashes/order/size проверяются `Document.Validate`; helper реально network none/read-only/512 MiB/pids16, версии всех трёх Poppler binaries 25.04.0. Tmpfs32 MiB ограничивает original10 MiB+PNG8 MiB. |
| G2.2 | `go test ./internal/documentdata ./internal/assistant -run 'TestPhotoFormats|TestDocumentRejects|TestImageFormats|TestGroupedTransport'`: PASS. Реальные PNG/JPEG/WebP, EXIF6 поворачивает32×48→48×32 и удаляется; malformed/oversize/hash tampering/5pages отклонены. `TestS56RealRendererRefusesFifthPage`: actual pdfinfo отклоняет настоящий 5-page PDF целиком. Прежние `TestHelperBusyAndCancellation`, `TestHelperRejectsOversizeAndInvalid`, `TestProcessGroupCancellation` PASS. |
| G3.1 | `TestS56StatementDocumentLifecycle` photo/scan/native и `TestS56ReceiptDocumentLifecycle` image/pdf в `./scripts/test-document-reading.sh`: PASS. Реальные файлы/helper/HTTP upload, preview/consent/fake reading, owner edit/review, confirm/reload/replay; до confirmation0 postings. |
| G3.2 | `go test ./internal/assistant -run TestGroupedReaderFull32ThroughActualAdapter`: PASS. Три настоящих PNG проходят новый OpenRouter wire→strict decoder→VisualReading→полный прежний MeasureDirect: 32 строки, **297/297 checks**, source binding PASS на native corpus, importReady=false. Независимый scan binding этим не заявляется. В PostgreSQL full32-scan сохраняет все32 операции, local precision, FX/описания; owner выбирает семантику own transfer/fees и точные postings. |
| G3.3 | `TestS56StatementDocumentLifecycle`: foreign owner, missing consent, stale revision, before-review, forged digest — отказ/0 postings; правка строки снимает только её proof, Original сохранён; unchanged page/header review сохраняются. `TestVisualRefusesTruncationMissingCurrencyAndInvalidRegions`: null/currency/sign/region blockers PASS. `./scripts/test-bank-statements.sh`: прежние duplicate/late-stale/atomic/concurrent/confirmation controls PASS; новый `TestStatementVisualSourceExpiry` очищает actual image source/render и закрывает page/original. |
| G3.4 | `make test-integration`: все17 scripts PASS, включая прежние JSON/CSV/receipts/native mapping, financial core и assistant tools. `TestS56StatementDocumentLifecycle`: exact delta/post count, confirmation replay, repeated file dedup, original/render удалены, минимальное evidence сохранено. Migration30 up/down/up на пустой БД PASS; downgrade с document evidence намеренно отклонён. |
| G4 | `make test` PASS: Go, **90 Web files / 339 tests**, ESLint/FSD/Prettier/build, generated OpenAPI types, Jev offline/corpus. `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check` PASS. После последних Go/image/compose additions отдельно повторены Go/vet, document suite и fake production-deployment checks; exact final-head CI публикуется в implementation PR. |

Локальные полные логи: `/tmp/grace-0056-make-test-final.log`,
`/tmp/grace-0056-all-integration.log`, `/tmp/grace-0056-production-tests.log`.
Последние предметные повторы: `/tmp/grace-0056-document-final.log`,
`/tmp/grace-0056-bank-final.log`, `/tmp/grace-0056-go-final.log`,
`/tmp/grace-0056-vet-final.log`, `/tmp/grace-0056-deployment-source.log`.
После review повторён `/tmp/grace-0056-document-review-final.log`: отсутствие
private storage даёт безопасный503/0POST; page error содержит private/no-store.
Подпись image batch в истории исправлена на «Фото-выписка».
Это временные локальные evidence; воспроизводимые commands/tests находятся в repo.

## UI evidence

Браузер Codex, только `127.0.0.1` + временный synthetic API, реальные React
компоненты. Проверены default940px и responsive1440px: изображение/поля,
увеличение, правка описания, dirty запрет row review, сохранение значений,
проверка строк/страницы/ролей; receipt upload→preview модели/ceiling→disabled
без consent→один fake reading, изображение остаётся после reading.
Это проверка UI, не human acceptance и не доказательство OCR-качества.

![Источник рядом с редактируемыми полями](../review/0056-statement-image-review.png)

SHA256 скриншота: `9b5c59d2ca523456030fbe9c36f207288293061289339f5bdef74dc64128575f`.
Receipt snapshot `/tmp/grace-0056-receipt-ui.png` — локальные вручную заданные
fake values, намеренно не используется как evidence точности распознавания.
Временный Vite QA entry удалён. В CI воспроизводятся `ReceiptReading.test.tsx`
и `ReceiptPhoto.test.tsx`: explicit plan consent, stale plan, сохранённый source,
ручные поля без доступной модели. Финансовые решения остаются на сервере.

## Ограничения и дальнейший gate

До реальной проверки не выбран и не квалифицирован новый production endpoint.
Существующий visual fake PASS подтверждает протокол/guards, не точность LLM.
G1–G4 приняты и интегрированы. Следующее предложение —
[Q1–Q3: offline-подготовка измерения](0056-openrouter-document-qualification.md),
**accepted Q1–Q3 / offline implementation pending acceptance**;
[AC/evidence](0056-qualification-evidence.md). Q4 требует отдельного допуска на
exact source/endpoint/данные/стоимость; production rollout отдельно.
Гайд оператора: [OpenRouter и чтение документов](../openrouter-document-reading.md).

## Post-merge G1–G4

На merge10a0059: `make test` (90 Web files/339 tests),
`make test-integration` (17 suites), `make test-production`,
`go vet ./cmd/... ./internal/...`, `git diff --check` — **PASS**.
Exact-main CI проверен независимым `verify-screening-source-ci.py`, Delivery
проверен по SHA и четырём обязательным jobs. Основной checkout чистый.

Первый запуск Docker gates из main остановился из-за недоступности bind-mount
UID10001 при правах700 на основной каталог. Чистый выделенный implementation
worktree fast-forward обновлён до merge SHA; полные integration/production
gates там PASS, права основного каталога не менялись. Worktree сохранён для
воспроизводимости sandbox-проверок. `make test` выполнялся с TMPDIR вне checkout;
защитные output_inside_repository проверки не изменялись.

Логи: `/tmp/grace-0056-merged-test.log`,
`/tmp/grace-0056-merged-integration-worktree.log`,
`/tmp/grace-0056-merged-production-worktree.log`, `/tmp/grace-0056-merged-vet.log`.
Live provider/metadata calls, key setup, production SSH/SQL/deploy и расходы
по merge не выполнялись. Q1–Q3 отдельно приняты и реализованы offline в PR123, приёмка/merge pending;
[AC/evidence](0056-qualification-evidence.md). Q4 NOT_AUTHORIZED.
