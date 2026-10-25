# Q4: выбранный кандидат, тарифы и готовность доступа

Дата: 2026-10-25, Asia/Yekaterinburg. Основание: поручение владельца
«приступай» к выбору модели и подтверждению тарифов/готовности доступа.
Контракт: [accepted SPEC-0056 Q1–Q3](../../specs/0056-openrouter-document-qualification.md).
Это подготовка предложения; Q4 NOT_AUTHORIZED, Q4_READY=false,
качество реального распознавания UNVERIFIED.

## Выбор

Один кандидат для statement/receipt: `google/gemini-3.1-flash-lite`,
provider endpoint `google-vertex/eu`, display provider `Google`.
Canonical proposal: `google/gemini-3.1-flash-lite-20260507`.
Последний canonical известен из прежнего локального snapshot; его актуальность,
точный endpoint status, structured outputs, context и ZDR обязан подтвердить
новый Q4 preflight. Исторический snapshot не является свежим proof.

Причина выбора: публично объявленные image inputs/structured outputs,
EU endpoint и низкая ограниченная стоимость. Это кандидат для первого
измерения, не утверждение превосходства качества над другими моделями.
[OpenRouter model page](https://openrouter.ai/google/gemini-3.1-flash-lite)
и [Google model documentation](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite)
прочитаны в этой работе. Текст/изображения corpus не отправлялись.

`internal/documentqualification/testdata/candidate.json` содержит только
операторское предложение: qualification пустая, verifiedAt/qualifiedUntil
нулевые. Ни fresh proof, ни production qualification файл не создаёт.
Model/provider switch, fallback и автоматический выбор альтернативы исключены.

## Тарифные и image границы

Официальная страница OpenRouter публикует для Google Vertex Europe
**$0.275/M input, $1.65/M output**. Общая витринная цена $0.25/$1.50
не используется для расчёта EU endpoint. В candidate установлены ceilings
275000/1650000 USD micros за миллион токенов.

При максимальных 30000 input /16000 output:
`ceil((275000*30000 + 1650000*16000)/1000000) = 34650 micros`
= **$0.03465** за запрос; восемь таких резервов = **277200 micros ($0.2772)**.
Это верхняя оценка token-части, не фактический счёт и не проверка баланса.
Потолки предложения остаются $0.25/call, $2.00/campaign, ≤8POST/6GET.
Input включает native text, prompt/schema и image allowance; output включает
все учитываемые provider output tokens, в том числе reasoning.

Допустимые дополнительные request/image flat fees установлены **0**.
Публичная страница не доказывает отсутствие всех endpoint fees. Перед POST
существующий verifier должен получить явные неотрицательные request/image
rates и принять их только в пределах ceilings. Отсутствующие/ненулевые fees
при таком предложении дают отказ; автоматически увеличивать ceiling нельзя.
Tokenized image input не является дополнительным flat image fee.

Разрешено ≤4 изображения ≤2048px по каждой стороне; corpus имеет максимум3
страницы на документ. Операторский резерв **4200 tokens/image** превышает
опубликованные Google ориентиры Gemini3 (high1120, ultra_high2240), но таблица
Google содержит приблизительные значения и не доказывает отображение OpenRouter
`detail=high` в конкретную upstream media resolution. Это консервативная
плановая граница, а не измеренный расход или гарантия качества.
[Google media resolution](https://ai.google.dev/gemini-api/docs/media-resolution).
Фактические usage/identity/billing guards остаются обязательными.

## Что подтверждено и что ещё не проверено

| Проверка | Evidence / состояние |
| --- | --- |
| Offline implementation integrated | PR123 MERGED; merge `1bf79935d60c1be480d9e611e1fc7a09604407d2`, 2026-10-25 04:10:33 +05:00; `gh pr view 123` |
| CI исходной интегрированной базы | `gh run list --workflow ci.yml --commit 1bf79935d60c1be480d9e611e1fc7a09604407d2`: run40141643032 completed/success; это база, не CI текущего candidate diff |
| Q4 workflow зарегистрирован | `gh workflow list --all`: `.github/workflows/document-reading-eval.yml`, active; dispatch не выполнялся |
| Production secret names существуют | `gh secret list --env production`: dedicated statement SSH key и operator owner ID зарегистрированы; прочитаны только имена/даты, не значения |
| Точный canonical/endpoint/ZDR/fees | UNVERIFIED; свежие API metadata calls только в отдельно разрешённом Q4 preflight |
| SSH credential/host/container/marker readiness | UNVERIFIED; production соединений в этой работе0 |
| OpenRouter key/account entitlement/remaining shared budget | UNVERIFIED; ключи/UUID/DSN не читались, SQL/key endpoint не вызывались |
| Реальное качество | UNVERIFIED; provider inference POST0 |

Наличие secret name не доказывает работоспособность его значения. Публичная
доступность модели не доказывает доступ конкретного аккаунта.
OpenRouter отдельно описывает EU in-region gateway `eu.openrouter.ai` как
Business/Enterprise feature. Существующий adapter использует `openrouter.ai`
и pin `google-vertex/eu`; это не доказательство полной EU residency всей
цепочки. URL/план аккаунта этой подготовкой не меняются. ZDR проверяется
по конкретному endpoint и request policy, а не по бренду Google:
[OpenRouter ZDR](https://openrouter.ai/docs/guides/features/zdr).

## Что должно войти в отдельное решение Q4

После review/интеграции candidate diff нужны exact source SHA и CI5/5,
собранный из него manifest SHA256, owner receipt и один dispatch attempt1.
Manifest с UNBOUND либо с SHA базы без candidate diff не является допуском.
Состав восьми synthetic документов и prompt/schema/scorer не меняется.

Предлагаемые лимиты: ≤8POST/6metadataGET/$2.00, ≤$0.25/call; один dispatch,
≤2 root SSH command sessions и ≤1 SCP staging, pinned host key. Стандартный
remote preflight проверяет доступ, metadata и существующий shared budget;
отказ расходует попытку, без retry или пополнения allowance. Новых ключей,
бюджетных строк и проверочных платных ping не предлагается.

Пути существующего wrapper:
`/opt/grace/shared/document-reading-spec0056-q.started`,
`/opt/grace/shared/document-reading-spec0056-q.private.json`,
lock `/opt/grace/shared/statement-screening.lock`,
staging `/tmp/grace-document-reading-q-<exact-source-sha>`.
Marker/report создаются только разрешённым запуском. Их текущее отсутствие
не проверялось. Production activation/import/deploy в Q4 не входят.

## Локальная проверка предложения

`go run ./cmd/document-reading-eval -mode=check` PASS:8 cases,
page counts1/2/1/3/3/1/1/1, corpus digest
`268d0c2d49ea0803a2b9cffd619448ed9c9304f493d0db4e567a39472a3efa80`,
source UNBOUND, executionAuthorized=false, activationAuthorized=false.
Предварительный manifest: `/tmp/grace-q4-candidate-manifest.json`;
это временное evidence, не manifest одобренного exact source.

Временный offline helper загрузил actual corpus/candidate, установил только
локальную ModelSelection(DocumentReading,4,2048) и текущий VerifiedAt для
расчёта; вызвал existing DocumentRequest и ReservationMicros с actual
statement/receipt prompt/schema и16000 output. HTTP client, keys, env, SQL и
ExperimentGrant не использовались; helper удалён. Это проверка вместимости
предложения, не прохождение live metadata/preflight.

| Case | Input upper bound | Reserve, USD micros |
| --- | ---: | ---: |
| cards | 15667 | 30709 |
| columns | 19781 | 31840 |
| cards-photo | 15210 | 30583 |
| full32-native | 28663 | 34283 |
| full32-scan | 23622 | 32897 |
| receipt-discount | 8147 | 28641 |
| receipt-pdf | 8147 | 28641 |
| receipt-unreadable | 8147 | 28641 |

Суммарный консервативный резерв **246235 micros ($0.246235)**;
все requests≤30000 input, отдельные резервы≤$0.25, сумма≤$2.00.
Лог `/tmp/grace-q4-candidate-reserves.log`. Существующий реальный shared
allowance всё равно должен независимо подтвердить достаточный остаток.

Первый `make test` и первый fake workflow test выявили старые assertions
`candidate=null`; они обновлены для выбранного предложения. CLI test
проверяет отсутствие execution/activation и0 credential reads/run calls;
workflow test сравнивает exported candidate с actual check и сохраняет
оба authorization=false. Отдельный null-candidate refusal test сохранён.
Повторный bundle дал338/339 web PASS и timeout в неизменённом router test;
в обоих отказах DOM оставался пустым до завершения lazy route import.
Тесты router теперь явно ждут `router.state.initialized` до render/DOM assertions:
реальные lazy-модули сохранены, таймауты и product code не менялись.
После этой синхронизации полный `make test` PASS, web339/339.

Итог доступных проверок:

| Связь с AC принятого Q1–Q3 | Исполняемое evidence этой подготовки |
| --- | --- |
| Q1.1 / Q2.2 | Штатный `-mode=check` PASS; выбранный candidate, неизменный corpus/порядок; вместимость8actual DocumentRequest и reserve проверена offline |
| Q2.1 / Q3.2 | `go test ./cmd/... ./internal/...` PASS после обновления candidate assertion; `python3 -B scripts/research/test-document-reading-workflow.py` PASS: fake workflow/wrapper/CLI, actual SSH/provider calls0 |
| Q3.4 | `make test` PASS после синхронизации router test с реальной lazy initialization: Go, web339/339, lint, lint:fsd, format, build, Jev и corpus gates. Лог `/tmp/grace-q4-pr-make-test.log`; прежние timeout FAIL сохранены ниже |
| Q3.4, оставшиеся gates | `make test-jev-eval test-statement-page-corpus` PASS; web lint/lint:fsd/format PASS |

Логи: `/tmp/grace-q4-candidate-make-test.log` (старый null assertion),
`/tmp/grace-q4-candidate-make-test-final.log`,
`/tmp/grace-q4-candidate-make-test-retry.log`,
`/tmp/grace-q4-candidate-router-isolated.log`,
`/tmp/grace-q4-candidate-workflow-final.log`,
`/tmp/grace-q4-candidate-remaining.log`.
Новый integration/production live запуск не выполнялся. Содержимое production
и основной checkout не менялись. Ветка `codex/document-q4-candidate` содержит
review diff для публикации в PR; merge/dispatch не выполнялись.

`cd web && npm run build` PASS (лог `/tmp/grace-q4-candidate-web-build.log`);
`git diff --check` PASS; `go vet ./cmd/... ./internal/...` PASS
(`/tmp/grace-q4-pr-vet.log`). Remote exact-head CI будет зафиксирован в PR;
точный source/manifest для разрешённого запуска ещё не зафиксированы.
