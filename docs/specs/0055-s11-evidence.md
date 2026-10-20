# SPEC-0055 / S1.1 implementation evidence

Текущий статус на 2026-10-20: **S1.2 owner-test enabled**, `7b2deea`, run `39396162864`; [actual evidence](0055-s12-evidence.md). Ниже сохранён исторический snapshot и его FAIL, не текущий runtime статус.

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Accepted contract: [S1.1 lexical grounding](0055-s11-lexical-grounding.md); прямое решение владельца «Утвердить S1.1, один smoke и включение при PASS».
- Implementation branch: `codex/statement-lexical-grounding`, база accepted planning PR #72 / main `49152ad`.
- Implementation integrated: [PR #73](https://github.com/QahNaaRlN/grace/pull/73), merge `ef782ec3d3ad2772aa758856744763c288b7a99d`, 2026-10-19 12:44:28 UTC / 17:44:28 +05:00. PR CI `39253076844` — все 5 jobs PASS. Main CI `39254418192` и exact-release Delivery/runtime проверяются отдельно.
- Production `ef782ec`: main CI `39254418192` (5/5), Delivery `39255451276` (4/4) PASS. Post-merge `make test` / `go vet` / `git diff --check` PASS, primary clean.
- Один разрешённый S1.1 smoke использован: deploy `39255669019`, internal health PASS 2026-10-19 **12:55:14 UTC / 17:55:14 +05:00**, activation FAIL 12:55:19 UTC. Columns 2 rows, единственный scorer code `record_description`; cards NOT_RUN. AI false.
- Read-only `39255943568` PASS: direct `statement_ai_runtime_disabled` 12:56:16 UTC, readiness без unknown usage blocker. Standalone public/auth smoke PASS: `/`, `/login`, `/api/meta` 200; protected routes 401. Никаких платных повторов.


## Изменения

Server-owned `lexicalClass` вычисляется до redaction, входит в hashed plan и
OpenAPI/TS preview, но не добавляется в tuple messages. Hidden numeric fragments
получают `[number_<alias>]`, other — `[text_<alias>]`. Старые privacy deny rules
сохранены и распространяются на Unicode-space integer groups; значения не
раскрываются даже при visibility. Дополнительный numeric shape hint принят
владельцем явно, требует нового preview/consent.

Money enum и Go canonicalizer используют только numeric_fragment metadata,
никогда prefix source text. Source-order aliases сохраняют один complete atom,
исходные IDs/values/geometry и exact-once membership. Planner/alias/redaction/
prompt/schema versions обновлены. Private administrative report v3 включает
ScorerVersion; summary требует 19/20 checks, включая обе exact descriptions.
S1 policies/token/byte/call/timeout/allowance/fees и explicit confirmation сохранены.

## L1–L7 → executable evidence

| ID | Проверка / доказательство |
| --- | --- |
| L1 | `TestLexicalShapeIsNotFinancialMeaningOrPlaceholderText`, `TestSourceOrderAliasesAndHiddenClassesRemainAuthoritative`: grouped integers/Unicode spaces/decimals/signs, hidden IDs even with visibility, fake literal `[number_s0]` имеет other и не попадает в money enum. Private values/canonical map и JSON-поле lexicalClass не добавляются в tuples; принятый shape hint виден через placeholder/enum |
| L2 | `TestActivationRealHelperSmoke/recorded_S1_failure`: реальный retained model mapping S1 перенумерован только по новому alias bijection; три ordinary-text amount refs отвергаются до merge/draft (`qualification_protocol_failed`), known bill settled, второго call нет. Sign остаётся eligible, value проверяется штатным money parser |
| L3 | `TestDocumentAliasesRoundTripAllTypedSelections`, plan mutation/legacy/exact byte/span tests и [actual planner report](../research/statement-quality/s11-evidence/summary.json): **84 PASS**, v7/v8/current prompt sizes, two real public PDF + geometry stress, exact-once/full-record membership. Current source/profile hashes в report |
| L4 | `TestActivationRealHelperSmoke`: actual pinned helper → delivered-tuples-only positive fake → canonicalizer → Go → strengthened scorer/report roundtrip. Оба baseline PASS; missing coverage, wrong FX, swapped record descriptions FAIL. SQL `TestStatementRealPDFFakeProviderLifecycle` сохраняет отсутствие finance writes до confirmation и replay safety |
| L5 | `TestScoreCatchesLostClockAndAvailableAsClosing` также отвергает title/foreign-record descriptions; требуемые totals 19/20. `TestActivationReportRejectsLegacyIncompleteChangedEvidence` отвергает scorer-version/profile/report mutations; summary не принимает старый version. Diagnostic сохраняет возможность read-only historical reports |
| L6 | Команды `make test`, `make test-integration`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check`. Generated TS получен штатным `npm run generate:api`, DTO вручную не дублировались. Focused Go, real-helper/bank, 84 structural gates и все пять перечисленных обязательных команд **PASS** локально (завершённые exit 0). Exact-head CI проверяется перед merge |
| L7 | **FAIL**: live run `39255669019` columns `record_description`; cards NOT_RUN, enable/runtime post-check skipped. CI/Delivery/internal health PASS, direct read-only flag false и public/auth PASS. Один разрешённый run использован, нового S1.1 paid retry нет |

Real-helper test/probe runtime network=none/read-only UID10001 для provider-free
квалификатора и structural cases; disposable SQL suite имеет только свою network.
Source goldens и saved failure refs относятся только к fixed public corpus,
никогда не к личной выписке. Privacy diagnostic tests — отдельные negative checks.

Local logs: `/tmp/grace-s11-focused.log`, `/tmp/grace-s11-bank.log`,
`/tmp/grace-s11-structural.log`, `/tmp/grace-s11-test.log`,
`/tmp/grace-s11-integration.log`, `/tmp/grace-s11-production.log`,
`/tmp/grace-s11-vet.log`. PASS определяется завершённым exit status, не наличием
частичного log. Exact current source hashes structural report проверяются до
публикации. CI/integration/production completion не выводятся из intention.

## Ограничения и следующие gates

Source-order alias effect на реальную модель остаётся UNVERIFIED; структурный
PASS не предсказывает model semantic PASS. Устранён конкретный impossible
text-as-money path, а description scorer усилен, поэтому старый ошибочный ответ
не может стать «успехом» от ослабления проверки. S2/expanded corpus/AC12/Jev и
SPEC-0056 model registry по-прежнему не qualified. При live FAIL нового
разрешённого run — никакого ещё одного paid retry без нового решения владельца.

Исторический S1.1 prototype comparison воспроизводится из planning revision
`aeac32d` (production S1 sources без S1.1), как указано в его source hashes.
Для текущего product использовать actual planner runner и `s11-evidence`,
не смешивать baseline prototype со snapshot новой реализации.

## Диагноз единственного live FAIL

[Sanitized locators](../research/statement-quality/s11-smoke-2026-10-19-locators.json)
и [source-hashed replay](../research/statement-quality/s11-description-offline.json),
0 provider calls / 0 SQL: первая description exact; во второй только дефис
`p2-b8-l1-w3` помещён в service. Финансовые поля и coverage columns прошли,
но полное описание — самостоятельный обязательный критерий. Это улучшение
наблюдаемого случая, не доказательство устойчивости модели или cards quality.
Никакой автоматической починки/ослабления scorer. [S1.2](0055-s12-description-separators.md)
отдельно принят владельцем для узкого prompt amendment, regression и одного
нового bounded smoke; его семантический результат пока UNVERIFIED.

## Subsequent S1.2 activation — 2026-10-20

**Statement AI включён для owner production testing**: release `7b2deea`,
2026-10-20 **13:51:07 +05:00 / 08:51:07 UTC**. PR #74, main CI `39395041781`,
Delivery `39396018990`, deploy `39396162864` — PASS. Columns **19/19** и cards
**20/20** прошли строгий scorer; exact API image/runtime flag/current policy и
public/auth smoke подтверждены. [Evidence D1–D5](0055-s12-evidence.md).

OA7/OA8 достигнуты для отдельно принятого S1.2, а не ретроактивно для неуспешного S1/S1.1 run. Owner private functional acceptance остаётся pending.
