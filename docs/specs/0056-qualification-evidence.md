# SPEC-0056 Q1–Q3: offline-реализация и evidence

Дата: 2026-10-25, Asia/Yekaterinburg. Исполнитель: Codex.
[Контракт Q1–Q3](0056-openrouter-document-qualification.md) **accepted** до кода:
владелец ответил «Принять Q1–Q3 и реализовать offline» для PR123/head
`ab2f10bf8f08ffc9b6ea4598ba27bf708f96d2e2`. Receipt commit
`8ec6fe4d7d4089aec77bf578a289e28ca39cad6b` предшествует implementation diff.
Implementation acceptance/merge: **PENDING**; Q4 **NOT_AUTHORIZED**.

## Результат

Подготовлен один исследовательский путь через тот же OpenRouter gateway,
DocumentReasoner и ReceiptExtractor, без нового product route. Experiment grant
связан с source/approval/manifest/сроком; production ModelGroups отклоняет
reserved experiment qualification. Промпты/schema, privacy/identity/fees guards
общие с G1–G4. Никакого переключения действующих production групп не было.

Реальный локальный helper подготовил5 PDF/image cases выписок и3 чека.
Полный judge сравнивает строки, поля и позиции; сохранены старые legacy checks
и их отказы readiness. Положительный fake control даёт PASS_SYNTHETIC, full32
native и scan по297 recognition checks. Наличие matching golden не снимает
owner review; `importReady=false`, native binding не заявляется для сканов.

Campaign preflight проверяет все requests и консервативную сумму резервов;
лимиты8POST/6metadataGET/$2.00 total/$0.25 call не означают разрешения запуска.
Общий SQLBudget вызывает только существующие reserve/guard/settle/remaining.
Known bills сохраняются и при отвергнутом content/overrun; unknown или ошибка
settlement останавливает кампанию. Ни один финансовый writer runner не вызывает.

Raw HTTP и budget evidence сохраняются для replay: повторно проходят именно
actual gateway/decoder/scorer, а не сохранённый score. Research artifact decoder
имеет отдельный32MiB предел для base64/captures; provider content limits G1–G4
не увеличены. Replay не открывает сеть и не читает ключи. Envelope hash и
producer linkage не являются криптографической аттестацией происхождения.
Временное продление in-memory grant при replay действует только с закрытым
replay HTTP client; исходный execution interval проверяется по исходному grant.

CI projection содержит только фиксированные case/check codes, proof hashes,
finite timing/tokens/cost/settlement; literals/raw/errors/credentials не выходят.
Она сохраняет также все старые legacy checks, включая coverage/blockers FAIL.
PASS_SYNTHETIC относится к распознаванию синтетического набора; legacy financial
readiness остаётся false, её ошибки не пересчитываются в PASS.

## AC и исполняемые проверки

| AC | Команда / actual результат |
| --- | --- |
| Q1.1 | `go run ./cmd/document-reading-eval -mode=check`:8 cases, public source/страницы/goldens hashes, limits8/6/$2.00/$0.25, candidate=null, executionAuthorized=false. `TestQActualRendererCorpus` на настоящем helper: committed corpus совпадает; image-only не получает text/spans. PDF-чек отрендерен и визуально просмотрен. |
| Q1.2 | `go test ./internal/documentqualification -run TestQQualityNegativesStopAndSettle`: потеря строки32/страничной quote, extra row, currency, time/seconds, item/name/discount/null →REJECT, known bill settled, следующие not_sent. `TestQActualGatewayEightCasesAndReplay`:8 correct fake replies →PASS_SYNTHETIC; full32 по297, owner readiness=false. |
| Q2.1 | `go test ./internal/assistant -run TestQExperimentNeverBecomesProductionQualification`: обычная config отвергает experiment qualification; expired grant/proof не refresh и не исполняются. `TestQGoldenNeverEntersModelInput`: изменение golden marker не меняет изображения/text запроса и marker в wire отсутствует. |
| Q2.2 | `TestQProtocolBudgetFailuresNeverBecomeQualityPass`: text-only/schema/ZDR/missing fees/price, timeout, wrong model, unknown cost, cost overrun и credential echo; максимум1 fake POST до остановки, всего≤6GET. `TestQUnknownReceiptTransportFailureIsNotRecognition`: timeout последнего нечитаемого case →INCONCLUSIVE, не успешный unknown. Старые group/wire/stale/source guards G1–G4 также проходят. |
| Q3.1 | `./scripts/test-document-reading.sh`, `TestQSQLBudget`: новая одноразовая БД, настоящий shared reserve/guard/settle;8 fake bills=80micros только в тестовом ledger, финансовая delta0. Два owner конкурируют за ограниченный fixture budget — только один reserve; отменённый context не теряет settlement; unknown freeze блокирует другого owner. Production allowance не читался и не менялся. |
| Q3.2 | `python3 -B scripts/research/test-document-reading-workflow.py`: actual guard/build/transport blocks + реальный CLI, fake git/gh/SSH/SCP/Docker. Проверены source/attempt/ancestor/all5CI/duplicate dispatch, approved manifest, symlink/tamper/spent marker/lock, auth/SCP/loss, container identity, cleanup и stderr/model-text/overflow. Реальные SSH/provider calls0. Default workflow execute=false; candidate=null закрывает run до credentials. |
| Q3.3 | `TestQReplayRecomputesInsteadOfTrustingSavedResult`: forged score с пересчитанным seal и missing bill отвергаются. `TestQSummaryRejectsPayloadAndSemanticForgeries`: произвольные keys/codes/case/reason, bool/alias/amount/readiness не попадают в projection. `go test ./cmd/document-reading-eval`: default и source/execute/manifest/candidate отказы дают0 credential reads/0 execution calls. |
| Q3.4 | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`. Предметные Q проверки PASS; окончательные результаты общего локального bundle и exact-head CI фиксируются в [PR123](https://github.com/QahNaaRlN/grace/pull/123) перед решением владельца. Product API/OpenAPI/Web и migrations в Q не менялись. |

Ранняя SQL fixture проверка отказала при allowance=0: это подтвердило
preflight0POST, а не дефект production budget. После явной настройки только
одноразовой тестовой БД весь QSQL сценарий PASS. Guard не ослаблен.
Дополнительный fake build test сначала встретил fake git без VCS output;
stub теперь делегирует реальные read-only VCS команды, production go build
не получает `-buildvcs=false` или другого обхода stamping.

## Готовность и следующий шаг

`candidate.json=null`: настоящий model/provider, актуальные rates/flat fees,
доступ существующего ключа и runtime readiness **UNVERIFIED**. Поэтому
**Q4_READY=false**, live quality UNVERIFIED. Никаких paid/metadata calls,
чтения/настройки secrets, production SSH/SQL, dispatch/deploy/activation не было.
Стоимость настоящих provider calls этой работы —0;80micros выше синтетические.

После приёмки offline diff нужен отдельный merge. Далее можно выбрать
конкретный endpoint и представить Q4 по готовому source/manifest/тарифам;
этот PR не заменяет такой допуск. Ни source ref, ни операционные файлы сами
по себе не разрешают запуск. Старые grants остаются CONSUMED.

## Проверяемые артефакты и локальные логи

Corpus file SHA256: `8b6417266c3d6c7b7fc1e43c870b757313411c74f10ace806576605548b4e615`.
PDF-чек SHA256: `4650a27fc30fdab5a159d7199e9ade5b8912b889382b08736b629ca5d493c137`.
Manifest дополнительно раскрывает каждый source hash, страницы/их размеры и
hashes, native/text/golden digest, ожидаемые строки/позиции и режим источника.
Данные модели и полный ожидаемый ответ туда не копируются.

Локальные логи: `/tmp/grace-q-make-test-final.log`,
`/tmp/grace-q-make-integration-final.log`, `/tmp/grace-q-make-production-final.log`;
заключительный review/manifest repeat: `/tmp/grace-q-make-test-reviewed.log`,
`/tmp/grace-q-document-reviewed.log`, `/tmp/grace-q-workflow-reviewed.log`,
`/tmp/grace-q-vet-reviewed.log`. Команды/tests в repo воспроизводимы; `/tmp` —
временное execution evidence. Feature worktree сохраняется для review, основной
checkout чистый на интегрированном G1–G4.

## Проверка воспроизводимости в CI

Первый implementation CI40138581313 на7c866ed: **4/5 PASS**, integration FAIL
на `TestQActualRendererCorpus`. Это не model quality результат.
Локальный Go1.27.3 и CI Go1.24.0 выдавали разные JPEG байты трёх фото.
Повторная локальная генерация с Go1.24.0 показала: native PDF/scan/PDF-чек,
текст и goldens совпадают; изменились только normalized photo pages.

Prepare и manifest теперь явно фиксируют `normalizer=go1.24.0`; подготовка
с иным compiler отказывает. Integration script компилирует Q parity/SQL test
с `GOTOOLCHAIN=go1.24.0`, совпадающим с go.mod/CI. Corpus заново получен
через actual helper этим compiler; сравнение полных SHA не ослаблено.
Default check/run/replay используют сохранённые page bytes и могут проверяться
другим поддерживаемым Go без повторного кодирования изображений.
Это изменение исследовательской воспроизводимости, product normalizer не менялся.
Исходный FAIL сохранён; итог нового exact-head CI публикуется в PR123.

Дополнительно воспроизведён именно CI Alpine digest
`sha256:2e0b81d60597f449baa56dc25d355367869d2dca4cd57e866f1e2b52af3eb55f`
в отдельном локальном helper. Вместе с Go1.24.0 полный сохранённый corpus
совпал byte-for-byte (`cmp` PASS), включая все PDF/фото. Product Dockerfile,
текущий Docker tag и исходники normalizer не менялись. Логи:
`/tmp/grace-q-ci-base-parity.log`, `/tmp/grace-q-pinned-corpus.log`,
`/tmp/grace-q-make-test-compatible.log`, `/tmp/grace-q-integration-compatible.log`,
`/tmp/grace-q-workflow-compatible.log`, `/tmp/grace-q-vet-compatible.log`.
