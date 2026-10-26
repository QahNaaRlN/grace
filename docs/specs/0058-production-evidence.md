# SPEC-0058 P: production verification

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
