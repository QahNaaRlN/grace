# SPEC-0055 / S1.10: проверки исходной разметки

Дата: 2026-10-21, Asia/Yekaterinburg.
[Принятый контракт](0055-s110-grounding-guards.md), acceptance `2a21d4f`
предшествует production-коду. [PR №82](https://github.com/QahNaaRlN/grace/pull/82).
Implementation status: local_verified; интеграция и deployment не выполнялись.

| ID | Исполняемое подтверждение |
| --- | --- |
| G1 | `go test ./internal/statementdata` PASS: пропуск `-`, `+`, слова и placeholder внутри одной линии даёт description_evidence_gap при service и unmapped; исходные proposal/literal/money/debit totals сохранены. Line/block/page transitions, исходный Word-порядок без монотонности X и отдельный итоговый merchant text допустимы, unknown не заполняется. NaN/нулевая/повторная/обратная/несовпадающая metadata и missing retained evidence => description_evidence_unverified. Дедупликация fresh/read checks PASS |
| G2 | Та же unit suite: обе роли, все пять EN/RU phrases для каждой, регистр/пунктуация, opposite/unknown/available/bank/turnover/digit-bearing labels, соседняя/удалённая/чужая страница/колонка/обратная/перенесённая подпись, отсутствие/дубли evidence, противоречие literal/value/layout. BIC/KPP/INN/SWIFT и БИК/КПП/ИНН в контексте отклонены. Parsed values/evidence не меняются, nil parsing и прежний ambiguous_money сохранены. Split signed row/book money `-1 234.56` PASS |
| G3 | `scripts/test-bank-statements.sh` PASS: TestStatementGroundingLegacyPreviewAndConfirmation, 9 сценариев gap/book_role/missing_evidence × full/deleted/expired. Fresh и forged legacy review, rolesConfirmed=true, cleared blockers/matched, actual HTTP read/preview/confirm. Старый good preview =>409 preview_changed, новый blocked preview =>422 preview_blocked. Postings/events count неизменен, JSON review и revision не меняются при чтении/preview/failed confirm. Owner isolation/revision conflict/owner-renamed import description PASS |
| G4 | `go test ./internal/statementqualification ./cmd/statement-admin` и real-helper qualification PASS: старые scorer tags отвергаются при правильном заново вычисленном digest, current profiles принимаются. Closed new diagnostics и row_1..row_5000 prefix не пропускают private text/ID. Полные fake-provider columns19/cards20/RU23 проходят прежние сильные golden assertions. Historical replay v1 =>UNVERIFIED при текущем профиле, а current validator на фактическом S1.8 trace добавляет row_2:description_evidence_gap: current16/19 при historical17/19. Исходные projection/replay сохранены |
| G5 | `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go, 88 файлов/334 Web tests, lint/FSD/Prettier/build/Jev PASS. `make test-production`, финальные `make test-go`, `scripts/test-bank-statements.sh`, `go vet ./cmd/... ./internal/... ./scripts/research/s19-replay`, vet с integration tags и `git diff --check` PASS. External model requests=0, production workflow dispatch=0, deployment/model switch=0 |

## Что изменилось

`ApplyGroundingChecks` подключён в конце ValidateProposal и в applyStatementFlags:
новые proposals и cached/retained reviews получают одинаковые diagnostics.
Общая helper-функция проверяет порядок и геометрию retained spans; при наличии
layout сверяет каждый span с оригиналом через один индекс, без копирования layout.

Проверки добавляют row issues/blockers и BookBalance incomplete, сохраняя
значения, описания, selections и evidence. Денежные итоги не исправляются по
результату описательной проверки. Прежние source/row/header guards сохранены.
Человеческие подписи новых кодов добавлены в существующий Web issues dictionary;
бизнес-правил в React нет. Публичный API/OpenAPI и generated DTO не менялись.

Scorer tags: statement-fixed-corpus.v3.grounding и
statement-screening.v2.grounding19-20-23. Старый qualification PASS нельзя
переиспользовать как доказательство нового профиля. Текущий prompt/schema,
redaction/aliases/planner, model/endpoint/цены/лимиты/бюджет/fees не менялись.
Новых внешних запросов и production-активации пакет не содержит.

## Исторические результаты и границы

S1.8 остаётся historical FAIL17/19, S1.9 D5 PASS воспроизводим на своём source
ecfa850. Сохранённые projection/replay и private artifacts не переписываются.
На текущем новом validator фактический trace имеет дополнительный row issue,
что даёт 16/19, а не улучшение качества модели. Replay v1 отказывает до выдачи
нового PASS; новый remote diagnostic dispatch не выполнялся.

Обнаруживаются только внутренние пропуски в выбранном сегменте одной печатной
линии. Пропуск первого/последнего слова или целой линии этим invariant не доказан:
отдельный тест фиксирует эту границу; golden description equality не ослаблена.
Словарь book-balance labels конечный; unknown требует ручной разметки/unknown,
а не принятия по confidence. При недостаточной retained evidence остаётся
unverified; исходный PDF не восстанавливается догадкой.

## Исполнение и журналы

В первом SQL fixture позитивная mapping проверялась до owner page/role review;
исправлена фаза fixture, production guard не ослаблялся. В финальном проходе
все девять сценариев и остальные SQL/restore/down-guard regressions PASS.
После перехода helper на один индекс и удаления лишнего предположения о возрастании X повторены Go и bank suites. Финальное
vet выполнено на обновлённом коде; изменения Web ограничены подписью issues.

Журналы `/tmp/grace-s110-make.log`, `/tmp/grace-s110-go-final.log`,
`/tmp/grace-s110-production.log`, `/tmp/grace-s110-bank-final.log`,
`/tmp/grace-s110-vet-final.log`, `/tmp/grace-s110-integration-vet.log`,
`/tmp/grace-s110-helper.log`, `/tmp/grace-s110-guard-final.log`.

Scope check: `git diff --exit-code 5981449 -- api migrations internal/statementextract internal/assistant deployments/production docs/research/statement-quality/s18-saved-trace-projection.json docs/research/statement-quality/s18-saved-trace-replay.json scripts/research/diagnose-statement-s19.py scripts/research/inspect-statement-screening-s18.py` exit0.
CI текущего implementation/head отображается в PR №82; полный CI PASS отдельно
подтверждается после завершения всех jobs, локальные PASS не подменяют CI.
