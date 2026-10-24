# S1.22 V1–V3: offline evidence

- Принятая спецификация: [S1.22 V1–V3](0055-s122-direct-page-reading.md).
- Owner receipt: «Принять V1–V3 и реализовать offline», proposal PR120/head `fa18c519bf579776696b0bdf30c2ba0a62275416`; acceptance commit `e54b154370ad54a7387ec3caf47ae21426948362` создан до кода.
- Статус: **accepted implementation / integrated PR120**, main `e9d02b2`.
- Реальная модель, OpenAI account/key/region/privacy/funding: **UNVERIFIED**.
- V4/live provider calls, key read/setup, SSH/SQL/deploy/activation: **NOT_AUTHORIZED**; merge PR120 отдельно принят и выполнен. Здесь live calls0; реальные ключи/config не читались и не создавались. Ни research allowance для реального запуска, ни живой marker не создавались.

## Реализованный путь

`PNG всех страниц + обычный текст → Responses wire → строгие строки/quotes → unique source binding → прежние Go guards → recognition/judge/replay`.

Один документ — один запрос. В wire только prompt, пронумерованные PNG/detail=high и plain text при наличии; golden, span-ID aliases, feedback, tools и fallback отсутствуют. Candidate `gpt-6-astra`, medium, store=false/stream=false/service_tier=default, max_output_tokens16000. Отдельный research profile `statement.direct-page-reading.v1`; production transport/API/Web не подключались.

Каждое поле — raw value и ordered page/quote evidence либо null; Go разбирает money в minor units и даты с печатным временем. Строгий decoder отклоняет duplicate/case-alias/unknown/missing keys, неверные типы, trailing/malformed/truncated/oversize JSON. Source binding не получает golden: NFC и свёртывание только space/tab/CR/LF/VT/FF/NBSP/narrow-NBSP; пунктуация/знак/слова/время сохраняются. Row anchor задаёт область строки. Общая валютная колонка доступна только при отсутствии local match; неоднозначность не выбирается first-match. Направление должно ссылаться на signed account evidence.

`ValidateProposal(..., nil, false, "")` запускает прежние money/date/grounding/turnover-role/coverage проверки. Невыбранные spans не превращаются в service; owner flags не выставляются. Recognition, sourceBinding и importReady раздельны: в исследовании **importReady=false всегда**. Все blockers/checks публикуются; прежний ScreeningChecks используется без изменения для cards/columns и visual control. Двухстрочный legacy profile к full32 не применяется: все32 строки проходят отдельные297 checks и те же core guards.

## Зафиксированный public corpus

Все данные синтетические, из фиксированных литералов generator: ни человек, ни номер счёта, ни реальная/замаскированная выписка не использовались. Cards/columns исходные PDF не менялись. Full32 содержит русский текст, повтор заголовков, одинаковые суммы разных операций, own transfer, fee, FX и description через страницу. Просмотрены все6 уникальных PNG: текст читаем, строки не обрезаны. Image-only использует те же pixels cards, без extracted text; это control, а не независимый документ или OCR произвольной фотографии.

Renderer: immutable local image `sha256:5874a467b6446d165c933d99f65259c77c6db3dc9b2f9890ea96a1f37c7d97c3`, Poppler25.04.0; pinned DejaVuSans/font SHA в manifest, ReportLab4.4.9/invariant. Docker: pull=never/network=none/read-only/non-root, cap-drop/no-new-privileges, pids32/memory512MiB/cpu1, timeout90s, read-only PDF/font и только свой output rw. Без inherited env/secrets. Только служебный pdfinfo очищается от trailing spaces; source XML, PNG и plain text хранятся как получены. Два повторных prepare дали byte-equal manifest и все его PDF/PNG/text/source/golden hashes.

Manifest SHA256: `2df9b50fa82c6e1d8c598ca21728eebc238b670593e29034fbc4921ca6e60f0f`.
Prompt/schema hashes и полный список files/dimensions/order находятся в manifest/protocol.json. Все четыре запроса строятся до первого ответа; порядок campaign фиксирован twice(cards,columns,cards-image-only,full32).

| Case | Страниц | Строк | Recognition checks | Input token upper bound | Request SHA256 |
| --- | ---: | ---: | ---: | ---: | --- |
| cards | 1 | 2 | 27 | 8570 | `78644ba04d069533da37b57a074d435c62a71508d0df23b96f1caa8f73a2eb16` |
| columns | 2 | 2 | 27 | 11532 | `f15faa365f857ab496df8b6f0ea30c930b43abaad20167d803266736fae7f5fb` |
| cards-image-only | 1 | 2 | 27 | 8092 | `1126698a564e51f5929d145265af6766dd0458a8621d4fa0316db39a5ed663a0` |
| full32 | 3 | 32 | 297 | 19212 | `c336d09df82c580f5b1d43fbaa60197a8466c00098d2035c96292ba60eaf1fa1` |

Все4 positive fake controls: recognitionPassed=true, sourceBindingPassed=true, importReady=false. Binding checks соответственно23/20/23/234. У всех сохраняются `source_coverage_incomplete`, `owner_page_review_required`, `owner_field_role_review_required`; прежний full scorer честно не PASS. Это evidence реализации, не качества модели.

## Исполняемые AC

| AC принятого V1–V3 | Evidence |
| --- | --- |
| Visual wire | `TestDirectActualVisualHTTPWire`: fake RoundTripper читает реальный HTTP body и декодирует base64 PNG; byte/hash/order/detail/text/schema/model совпадают, image-only не содержит extracted text |
| Полный корпус | `TestDirectFourPositiveControls`, `scripts/research/test-statement-page-corpus.py`; full32=32 rows/3 pages,297 per-field/header/count/issues checks; actual sandbox repeat `cmp` manifest PASS |
| Семантика/binding | `TestDirectMeaningfulSemanticAndBindingNegatives`: пропуск/дубль строки/страницы, minor/currency/header currency/direction evidence/time precision/word/available-as-closing/false quote/ambiguous anchor/page-break/injection; прежние unresolved/owner blockers сохраняются |
| Safe defaults/caps | `TestCheckWithHostileEnvironmentNoKeysOrCommands`, `TestRunGateRefusalsConsumeWithoutKey`, `TestFakeCampaignReservationReplayAndNoRetry`, input/usage/identity/refusal/redirect/timeout tests: fake key getter только после durable reserve,≤8 POST,0 retry, one-shot также при отказе/concurrent attempt, unknown usage→uncertain |
| Replay/privacy | `TestDirectCampaignReplayAndWitness`, `TestDirectReplayRejectsTampering`, `TestDirectCredentialEchoNeverEntersPublicCapture`, `TestDirectNoRetryRedirectOrRawErrorLeak`: score/decision/hash/source/approval/cost/blockers нельзя подменить; есть concrete expected/observed/quote; keys/headers не сохраняются.104 исторических fixtures/evidence побайтно равны mainad1eca3 |
| Regression | `make test` PASS: Go,89 Web suites/336 tests, lint/FSD/format/build/Jev и corpus; повтор final Go PASS с TMPDIR в /var/tmp, `go vet ./cmd/... ./internal/...` PASS, staged/worktree `git diff --check` PASS. Exact-head CI5/5 публикуется в PR metadata после завершения; до этого остаётся pending |

Негативный public witness: cards row1/accountAmount expected `-1234.56`, observed `-1234.57`, quote `-1234.56` → REJECT; suffix not_sent. Даже при неизвестном usage известная смысловая ошибка остаётся REJECT, allowance uncertain. Timeout/unknown identity/usage без смыслового witness → INCONCLUSIVE; повтор не создаётся. Положительная fake campaign8/8 даёт только KEEP_FOR_FURTHER_REVIEW, importReady=false.

## Безопасное воспроизведение

```sh
go run ./cmd/statement-page-reading --mode check
go test ./cmd/statement-page-reading ./internal/statementextract ./internal/statementqualification -run 'Test(Check|RunGate|FakeCampaign|Direct)' -count=1
python3 -B scripts/research/test-statement-page-corpus.py
go vet ./cmd/... ./internal/...
git diff --check
```

Для полного `make test` в этом локальном окружении используется отдельный `TMPDIR` под `/var/tmp` (существующий import-review отвергает временный output, распознанный как внутри checkout). Standalone Go без такого TMPDIR выявил этот environment refusal; повтор с изолированным каталогом PASS. Код import-review не менялся.

Prepare требует явно установленных pinned ReportLab/font и cached renderer, сети не использует:

```sh
go run ./cmd/statement-page-reading --mode protocol > internal/statementextract/testdata/direct-page-reading/protocol.json
python3 scripts/research/prepare-statement-page-corpus.py --prepare
python3 scripts/research/prepare-statement-page-corpus.py --check
```

Эти команды готовят только публичные fixtures. Default check не читает env/key, не запускает subprocess и не делает network calls. Future run source существует, но здесь не выполнялся: в tests используются только dependency-injected key/HTTP и временные fake allowances.

## Future run source и оставшийся gate

Run требует execute flag, sourceSHA из build metadata, exact authorized-source и approval-id, explicit private local allowance. Atomic `.consumed` O_EXCL receipt создаётся до preflight; полный ceiling10400000 micros сохраняется+fsync файла/каталога до key getter/POST. Любой отказ расходует receipt, unknown result сохраняет uncertain; reset/release/retry source отсутствует. Key только `GRACE_STATEMENT_PAGE_OPENAI_KEY` после gates. HTTP endpoint фиксирован `/v1/responses`, timeout180s/POST и1500s/campaign, redirect/proxy env/fallback/retry0. Каждый capture сохраняет elapsed time и raw public response для независимого replay; elapsed включает decode/cleanup, deadline HTTP проверяется отдельно.

Input bound: UTF8 bytes prompt/text/schema+framing2048 +3001 tokens/page (high≤2500 patches×1.2, округление вверх); максимум30000 input/16000 output. До отдельного V4 ceiling предложен14/55 USD за1M input/output,≤1.30/request и10.40/campaign. `calculatedProviderCostMicros` явно **upper bound по price ceiling**, не invoice и не production settlement. Allowance требует подтверждённых funding/access/endpoint/region/privacy/pricing/no-extra-fees, но никакие такие подтверждения сейчас не заявляются.

Следующее решение — приёмка offline реализации/отдельное merge разрешение. V4 пока не запрашивается: account/model access, funding, privacy/region/актуальные rates и exact-source CI readiness ещё должны быть оформлены отдельным конкретным решением. Успех fake controls не означает, что исходная цель надёжного AI-импорта достигнута.

## Приёмка и post-merge

Owner: «Принять реализацию и слить №120», reviewed source
`52842439df5a3eb58df52b38690098a792f12602`. Merge
`e9d02b2bacad7fdcef919d28b44dd4817b721aae`, равный source tree
`4edda4974597227b7c0410182cc648f2cc8418a6`. Source CI40117756468/all5,
main CI40118781180/all5 и Delivery40119558599/all4 PASS. Main offline reader
check PASS, manifest `2df9b50fa82c6e1d8c598ca21728eebc238b670593e29034fbc4921ca6e60f0f`.
Реальные вызовы/ключи/deploy0. Новый OpenRouter product-контракт —
[SPEC-0056](0056-openrouter-document-reading.md), draft, реализация не начата.
