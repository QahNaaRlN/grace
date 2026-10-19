# SPEC-0055 / S1.1 implementation evidence

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Accepted contract: [S1.1 lexical grounding](0055-s11-lexical-grounding.md); прямое решение владельца «Утвердить S1.1, один smoke и включение при PASS».
- Implementation branch: `codex/statement-lexical-grounding`, база accepted planning PR #72 / main `49152ad`.
- Production по-прежнему `711a13d`, Statement AI disabled: direct read-only runtime check `39250755164` PASS 2026-10-19 12:16:28 UTC / 17:16:28 +05:00. Нового платного S1.1 smoke ещё не было.

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
| L7 | **NOT_RUN** в этом implementation snapshot: новый один S1.1 smoke отдельно утверждён, но ещё не выполнялся. Обязательны оба full strengthened PASS, точный release CI/Delivery/deploy/runtime/public checks; до них flag false |

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
