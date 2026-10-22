# S1.16 — offline implementation evidence

- Accepted contract: [S1.16 A1–A6](0055-s116-alias-schema.md), владелец принял
  2026-10-23 прямым «принимаю», интеграция [PR №97](0055-pr97-postmerge.md).
- Base: `a1cc07218da1aa9302f600d24c135aa67e09a1d6`.
- Branch: `codex/statement-alias-schema`, implementation commit: `e44963915d1e1c976af0b5dbbe30e84415846086`.
- Status: integrated offline, PR №98 / post-merge verification PASS.
  Владелец подтвердил слияние; financial acceptance и live qualification не объявляются.

## Результат и границы

`RequestSchema` использует shared `$defs.alias`, `$defs.selection` и
`$defs.currencySelection` для exact delivered IDs и отдельный прежний numeric
enum. Все22 selection paths ограничены; пустые множества дают arrays maxItems0,
currency maxItems1 сохраняется. Новый planner `statement-document-atom.v3.alias-enum`,
schema hash `1a6e755395da69883666f3eb61471ad8c4257f7d465451a4ad0050fbd253c3c7`.
Новый plan/consent/profile/request identity несовместим со старым v2.
Prompt, aliases allocation, redaction, scorer, guards, limits и financial writer
не менялись. OpenAPI/TS и SQL schema не менялись.

## Критерии

| ID | Текущее evidence | Статус |
| --- | --- | --- |
| A1 | `go test ./internal/statementextract -run TestAliasSchema -count=1`: independent explicit oracle22 paths, positive/negative instances с `$ref`, required/object/cardinality, empty D/N и placeholder spoof | PASS local |
| A2 | Та же suite: shared enum, deterministic order/dedup, input/template isolation, schema-ignoring unknown/duplicate/non-numeric proposals на всех paths; `go test -race ./internal/statementextract ./internal/assistant` | PASS local |
| A3 | `TestStatementRealPDFFakeProviderLifecycle`: реальные helper/SQL/HTTP, persisted v2 preview+consent→409 до reserve/Complete, новый v3 preview+consent→extraction, historical completed attempt читается; `TestAliasSchemaProfileRejectsPreviousQualification`, `TestAliasSchemaComparisonRequestIdentity` | PASS local |
| A4 | `TestDocumentPlanExactByteAndSpanBoundaries`:89999/90000/90001 bytes,2000/2001 spans; actual serialized HTTP schema/messages/bytes в `TestStatementWireProfileRoundTripAboveLegacyTokenBounds`; public size capture ниже | PASS local |
| A5 | Real network-none PDF helper + fake aliases + canonical/merge/unchanged Go/scorer: columns19/19, cards20/20, RU23/23, negative description/coverage/role cases FAIL; SQL lifecycle: финансовых writes0 до confirmation | PASS local |
| A6 | `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check` | PASS local: все команды exit0 |

## Public corpus: actual messages + schema bytes

Источник — `TestScreeningRealHelperThreeCases` в `make test-integration`.
[Машиночитаемый capture и source hashes](0055-s116-public-sizes.json).
Только shipped synthetic PDF; не D5/private-derived данные и не token count.

| Fixture | Spans | Messages bytes | Schema bytes | Total bytes / 90000 |
| --- | --- | --- | --- | --- |
| columns | 46 | 6024 | 3299 | 9323 |
| cards | 57 | 6369 | 3361 | 9730 |
| turnover-roles-ru | 51 | 6270 | 3359 | 9629 |

## Историческая совместимость тестов

Первый SQL run выявил missing owner_id у synthetic old-consent fixture;
fixture исправлен явным BootstrapOwnerID, production/RLS код не менялся.
Первый `make test-production` выявил смешение current Go v3 с frozen Python
v2 inspector. Runtime inspectors и их PROFILE не изменены. Тесты S1.13/S1.14
сначала проверяют, что frozen inspector отвергает current report, затем
компилируют test-only v2 overlay из exact public schema source snapshot
и прежней planner version для положительных и corruption/privacy/immutability
cases старого wire. Отчёты не relabel/rehash; current Go profile gates не ослаблены.
Новые тесты current Go отдельно отвергают старые planner/schema profiles.

## Исполнение

Logs: `/tmp/grace-s116-make.log`, `/tmp/grace-s116-vet.log`,
`/tmp/grace-s116-race.log`, `/tmp/grace-s116-integration.log`,
`/tmp/grace-s116-production-tests-final.log` (итоговый полный exit0).
Первый полный production gate завершился ошибкой последнего off-site backup fixture
(`/tmp/grace-s116-production-tests.log`); изолированный повтор backup exit0
(`/tmp/grace-s116-backup-rerun.log`) и последующий полный `make test-production`
exit0 без изменения backup-кода. Причина первого upload failure не установлена.

Эти результаты offline/fake,
не доказательство model compliance, completeness или live quality.
Внешних inference/metadata calls0; production/SSH/private artifact reads0.
Нового live permission/marker нет. Старый P6 FAIL и consumed D5 сохранены.


## Итоговая проверка границы

Implementation `e44963915d1e1c976af0b5dbbe30e84415846086`: production diff
только `schema.go` и PlannerVersion в `plan.go`; остальные code changes — tests
и frozen test-only source. Primary maina1cc072 clean. API/Web/migrations/financial
validators/operational inspectors/workflow/deployment diff отсутствует.
Обязательные local gates завершены exit0, включая полный integration и полный
повтор production gate. Новый PR/head CI проверяется отдельно после публикации;
его SUCCESS заранее не заявлялся. Последующая интеграция PR №98 и post-merge
проверки — в [отдельном receipt](0055-pr98-postmerge.md). Следующий контракт
[S1.17](0055-s117-alias-comparison.md) пока draft.
