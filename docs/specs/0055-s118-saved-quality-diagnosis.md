# SPEC-0055 / S1.18: диагностика качества сохранённого E6

- Статус: accepted (Q1–Q5 offline).
- Автор: Codex; дата: 2026-10-23, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: прямое «принимаю»,
  2026-10-23, после предъявления [PR №103](https://github.com/QahNaaRlN/grace/pull/103)
  на review head `2f05dca7f64085bb5cc80c8e6424f77dda9c44fb` и границы Q1–Q5/Q6.
- Основание подготовки: «влил PR, продолжай работу» после предложения подготовить S1.18.
  Контракт Q1–Q5 впоследствии явно принят указанным выше решением.
  Q6 требует отдельного exact-source решения после реализации Q1–Q5.
- Implementation status: not_started; Q6 execution status: NOT_AUTHORIZED / NOT_DISPATCHED.
- База: main `2236c73c54058123de800f624f7cebc3a95bf41f`, интегрированный PR №102.
- Ветка спецификации: `codex/statement-s118-diagnosis-spec`.
- Implementation branch / commit: отсутствуют.
- Зависимости: [SPEC-0055](0055-bank-statement-pdf-import.md),
  [S1.17](0055-s117-alias-comparison.md), [E6 evidence](0055-s117-e6-evidence.md).

## Проблема и пользовательский результат

Единственный E6 завершился `qualification_quality_failed`: canonicalization,
capture и replay прошли6/6, качество0/3 у обеих моделей. Из публичной проекции
известны coverage, clocks, turnover и `unknown` validation, но конкретные
структурные причины не установлены. Повторный inference для их разбора не нужен.

Результат пакета — воспроизводимая связь каждого failed check с actual validation
и выбранными source positions либо честное UNVERIFIED. Это даст основание выбрать
следующее минимальное исправление; само исправление в пакет не входит.
Диагноз описывает ответ и проверяющий predicate, а не рассуждение модели.

## Scope

**Q1–Q5 offline приняты**, реализация после интеграции спецификации в main: pure
диагностика6 сохранённых tuples на fixed v3 context; закрытый verifier/output;
отдельный read-only runner и default-off workflow mode; causal/privacy/real-helper
проверки. Допустимые области будущего diff: `internal/statementqualification`,
узкие helpers существующих predicates в `statementdata` при доказанной неизменности
поведения, отдельные `scripts/research`, workflow isolation, tests/gates/docs.

**Q6 execution** — один отдельно разрешаемый read-only запуск на VPS после
реализации, review/integration и exact-source CI5/5. До этого SSH/VPS=0.
Принятие Q1–Q5, слияние №102 и старое E6 permission не разрешают Q6.
В текущем документационном пакете Q1–Q6 не исполняются.

## Non-goals

Inference/metadata/catalog/generation GET, новые модели/fixtures, retry/repair,
SQL/reconciliation/reserve/settle, бюджет/fees/allowance, production policy/flags/
image/health/config, deployment, финансовые записи, личные PDF и owner drafts.
Prompt/schema/planner/redaction/alias allocation, golden/scorer/guards/admission
не меняются. Не назначать unmapped spans в service автоматически, не исправлять
saved proposal, не переписывать report hashes/profile и не расширять legacy readers.
Старые P6/D5 artifacts не читать. SPEC-0056 registry остаётся draft/not_started.

## Текущее поведение и доказательства

[E6 run39925874197](https://github.com/QahNaaRlN/grace/actions/runs/39925874197)
на immutable source `24f73223b9b3370ca78c41d34e89edca9a276488`, ref
`codex/statement-s117-e6-source`, permission CONSUMED. Расход10897 USD micros
known/settled; эта диагностика не перечитывает ledger и не переоценивает billing.

| Tuple | Candidate / fixture | Quality | Наблюдаемые failed checks |
| --- | --- | --- | --- |
| 1 | A / columns | 17/19 | coverage, blockers |
| 2 | B / columns | 17/19 | coverage, blockers |
| 3 | A / cards | 14/20 | operation_clock и booking_clock по обеим строкам, coverage, blockers |
| 4 | B / cards | 18/20 | coverage, blockers |
| 5 | A / turnover-roles-ru | 17/23 | row_issues по обеим строкам, printed_debit_turnover, printed_credit_turnover, turnover, blockers |
| 6 | B / turnover-roles-ru | 19/23 | row_issues по обеим строкам, turnover, blockers |

[Closed E6 projection](0055-s117-e6-public-summary.json) сохраняет порядок и
повторения codes. `unknown` в validation/coverage ещё не объяснён.
`ReplayComparisonObservation` подтверждает canonicalization, hashes и diagnostic;
он не заменяет полную проверку review/scorer equivalence для нового диагноза.
`Score`/`ScoreScreening` уже считают19/20/23 checks; новая шкала не нужна.
`ValidateProposal` и `NewMappingSnapshot` — источники blockers и unmapped membership.
`DiagnoseSavedProposals` закреплён за S1.14/v2 и другими3 наблюдениями: менять его
source/context или использовать его output как E6 diagnosis запрещено.
Существующий S1.17 inspector читает SQL/Docker и потому не является Q6 runner.

## Предметная модель и инварианты

### Q1. Fixed E6 context и полное воспроизведение

Вход — только outer `statement.comparison.observed.s117.v1`, authorization
`s117-comparison-20261023`, marker source24f7322, exact6 tuples в исходном порядке,
прежние A/B policies,3 fixture hashes и profile:
`statement-document-atom.v3.alias-enum`, schema
`1a6e755395da69883666f3eb61471ad8c4257f7d465451a4ad0050fbd253c3c7`.
До witnesses проверить strict JSON, outer/inner digests, version/auth/profile,
source/marker, bounds и document/plan/request hashes. Наблюдаемый baseline —
captured6/6; availability каждого snapshot проверяется отдельно без восстановления.
Локальные public contexts строятся pinned Poppler25.04.0 в network=none.

Для каждого tuple воспроизвести canonicalizer→MergeProposals→ValidateProposal→
ScoreScreening с прежними reviewedPages/rolesConfirmed/zone. Сверить canonical
proposal, replay stage/diagnostic, весь Result (codes/validation/coverage, их
порядок и повторения, checks/correct/pass/rows) и falseAcceptance с saved report.
Latency брать только из recorded Call, не из времени диагностического replay.
Проверять настоящий v3 request; JSON repair, rehash или подмена PROFILE запрещены.

Context/integrity failure закрывает весь output без witnesses. При верном общем
context missing snapshot/replay mismatch отдельного tuple даёт ему UNVERIFIED;
не выдавать его diagnosis за VERIFIED. Ограниченность3 fixtures и исторический
quality FAIL сохраняются при любом результате диагностики.

### Q2. Coverage и объяснение sanitization unknown

Для tuples1–4 вывести actual unmapped membership через `NewMappingSnapshot`,
общее число и positions в порядке `Document.Spans`. Position: SourceOrdinal с1,
Page/Block/Line/Word, closed lexical class `numeric`/`other` из verified plan.
Membership не означает service/financial role и не разрешает игнорировать span.

Для всех6 tuples сопоставить каждый actual row issue, reconciliation blocker и
coverage reason с сохранённой проекцией, включая случаи, где sanitizer дал
`unknown`. Только перечисленный ниже finite vocabulary существующих Go predicates допускается
в новом versioned allowlist; implementation review подтверждает scopes и tests.
Не выводить произвольный string из report/error/model.
В output сохраняются scope (`row`/`blocker`/`coverage`/`turnover`/`book_balance`), row ordinal при наличии,
порядковый номер reason, прежний public code и разрешённый actual predicate code.
Если row issue/turnover reason не имеет собственного элемента в старом public output,
указать closed relation `not_exposed` без public code; не придумывать ему `unknown`.
Связь с соответствующим blocker проверяется отдельно.
Сам по себе повторённый `unknown` не доказывает одну общую причину.

Unknown actual code остаётся UNVERIFIED/unsupported_reason_code; никаких догадок
о том, что hidden code означает. Старые public sanitizer и E6 projection не менять.

Нормализованный reason vocabulary Q2 (не предположение о фактическом E6):

- Evidence: `missing_evidence`, `invalid_span_reference`, `noncontiguous_evidence`,
  `noncontiguous_date_evidence`, `field_evidence_limit`, `overlapping_evidence`.
- Dates/money: `operation_date_required`, `invalid_date`, `invalid_timezone`,
  `ambiguous_local_time`, `account_amount_required`, `currency_required`,
  `currency_conflict`, `invalid_currency`, `ambiguous_money`, `amount_overflow`,
  `unsupported_account_currency`, `zero_posting`.
- Direction/description: `invalid_direction`, `direction_conflict`,
  `direction_required`, `transaction_direction_conflict`, `description_evidence_limit`,
  `financial_candidate_in_description`, `description_evidence_gap`,
  `description_evidence_unverified`.
- Header/roles: `field_label_required`, `available_is_not_book_balance`,
  `negative_turnover`, `turnover_role_unverified`, `book_balance_role_unverified`.
- Reconciliation/coverage: `aggregate_overflow`, `invalid_source_rows`,
  `printed_turnover_mismatch`, `book_balance_mismatch`, `book_balance_period_not_proven`,
  `source_coverage_incomplete`, `unmapped_spans`, `invalid_page_review`,
  `owner_page_review_required`, `owner_field_role_review_required`, `service_reason_required`.

`row_N:code` разбирается только как bounded ordinal + code из этого списка;
header/service prefixes — только enum `openingBookDate`, `closingBookDate`,
`openingBookBalance`, `closingBookBalance`, `availableBalanceObserved`,
`debitTurnover`, `creditTurnover`, `service`. `unmapped_spans:N` превращается в
`unmapped_spans` + integer count0..2000 и сверяется с actual membership; raw suffix
не копируется. Неизвестный/лишний prefix, неоднозначный разбор или несовпадение
count — UNVERIFIED. Новые predicates вне этого перечня требуют review контракта.
Показать causal связь с actual production predicate либо его проверяемым
результатом; самостоятельную расходящуюся копию validation не вводить.

### Q3. Clocks и turnover witnesses

Для clock failures tuple3 показать row ordinal, поле operationDate/bookingDate,
selected source positions и сравнение с независимым golden соответствующего
public fixture. Только closed отношения: `missing`, `matches`, `differs`,
`unavailable`; отдельно date/time/precision/UTC parts, без literal dates/text.
`matches` для selected part не объявлять PASS всего clock check. Перепутанные часы
или потерянное время можно утверждать только при доказанном exact relation;
иначе фиксировать наблюдаемый mismatch, не его предполагаемую причину.

Для tuples5–6 показать связи row_issues, printed debit/credit turnover и turnover
reconciliation с actual issues/predicates и selected source positions. Для money
только наличие/отсутствие и сравнение amount/currency с pinned public golden;
для role guard — разрешённый predicate code и структурные evidence positions.
Не выводить суммы, currency/source literals, BIC/KPP/account digits или labels.
Не считать отсутствующий header равным0. Не переносить суммы из golden в proposal.
При отсутствии точного witness — UNVERIFIED, без вывода о «галлюцинации» модели.
Все failed checks таблицы должны иметь verified explanation либо явное limitation.

### Q4. Closed output и fixed read-only verifier

Новая проекция: `statement.saved-quality-diagnosis.s118.v1`, fixed E6 context,
отдельные artifactSource24f7322 и verifierSource exact40hexSHA,6 tuple rows,
contextVerified/artifactsUnchanged, providerCalls=0, persistentWritesRequested=0.
Для каждого tuple и issue отдельный VERIFIED/UNVERIFIED; причины только
snapshot_unavailable/context_mismatch/replay_mismatch/guard_mismatch/
unsupported_reason_code/diagnosis_limit_exceeded/unavailable_runtime.
Witnesses отсутствуют у UNVERIFIED issue; verified соседние issues сохраняются.
Context/file failure выдаёт только closed diagnostic_unavailable без частичных данных.

Не более128 positions на family/tuple,128 reason entries/tuple и65536bytes output.
Превышение — bounded UNVERIFIED без усечённого VERIFIED списка; output overflow
закрывает всю проекцию. Exact totalUnmapped допустим отдельно при его проверке.
Строгие keys/types/enums/bounds на Go boundary и Python wrapper; duplicate/case-
ambiguous keys, unknown fields, trailing JSON, overflow, free text отвергать.
No raw source/alias/model/service text, hashes отдельных selections, raw proposal,
owner/reservation/generation/provider IDs, credentials или free-form exceptions.

Verifier читает только fixed `/opt/grace/shared/statement-comparison-s117-comparison-20261023.json`
и `.started`: O_NOFOLLOW, regular0600, report<=524288bytes, marker<=128bytes,
no arbitrary path/auth flags. Before/after bytes hash/size/mode/mtime совпадают;
atime не критерий. Проверять identity открытого файла и path при замене/гонке.
Никаких SQL/Docker/API/provider/key constructors и новых health probes на VPS.

Отдельные bootstrap/wrapper по bounded S1.15 memfd pattern: pinned public contexts
готовятся локально; на VPS sealed memfd binary, bounded stdin, env={}, stderr
не публикуется. Binary<=16MiB, public context<=524288bytes, envelope<=32MiB,
remote timeout45s, SSH timeout75s. Persistent writes requested0 относится к
verifier/bootstrap, не отрицает обычные OS/SSH audit logs. Никаких установок,
изменений permissions, новых report/marker или persistent runtime files.

### Q5. Offline gates и изоляция запуска

Отдельные default-false `diagnose_statement_s118_only` и exact verifier SHA input.
Mixed modes выбирают0 execution jobs; все старые live/diagnose/inspect/deploy/
reconcile jobs исключают новый mode. YAML parser всех workflow и полная матрица
режимов обязательны. Не переиспользовать paid S1.17 job или его permission marker.

Causal tests: настоящие delivered-alias proposals для coverage/clocks/turnover,
реальные canonicalize/validate/scorer результаты, repeated unknown и near misses,
независимый golden, limit128/129, missing snapshot, mutated source/profile/hashes/
scorer mismatch, файл/symlink/path race, env poison, malformed JSON и sentinel
на success/error/timeout. Real pinned helper проверяет actual document/plan hashes.
Fake-provider tests не объявляются E6 evidence. Legacy decisions/output/profile,
scorer count19/20/23 и financial guards остаются неизменными.

### Q6. Отдельно разрешаемое read-only исполнение

После Q1–Q5 acceptance, implementation review/integration и полного exact verifier
CI5/5 предъявить владельцу один fixed-host run, SHA/ref и hash manifest binary/
public contexts/bootstrap/wrapper. Q6 остаётся NOT_AUTHORIZED до отдельного
решения по этому источнику. До dispatch записать receipt; после него permission
consumed независимо от успеха. Никакого автоматического retry/repair после отказа.

Допустимы только fixed artifact read, verification/replay и closed stdout.
Provider/metadata/SQL/metering calls0; artifacts/permissions/runtime unchanged.
В evidence отдельно verifier/source/run/job/CI,6 tuple statuses, ограничения,
artifactsUnchanged. Payload с witnesses не загружать в Git/PR/Actions artifacts:
использовать закрытый stdout в рамках отдельно разрешённого read, публично
публиковать только safe status/counts и ссылки. Доступность saved bytes до Q6
UNVERIFIED; отсутствие данных не разрешает повторить платный E6.

## API и OpenAPI

Product API/DTO/OpenAPI/Web не меняются; TypeScript regeneration не требуется.
Новая версия — только operator projection; read-only diagnosis не import consent.

## Данные, миграция и rollback

Миграций/финансовых/ledger writes нет. Fixed E6 source/ref/report/marker immutable;
старый результат quality FAIL и CONSUMED не изменяются. Remote memfd ephemeral;
нового persistent storage нет. Rollback выключает новый diagnostic mode и не
удаляет исторические artifacts/evidence. Гипотеза будущего ремонта оформляется
отдельным принятым контрактом, не патчем поверх saved proposal.

## UI

Пользовательский UI и owner drafts не меняются.

## Ошибки и негативные сценарии

Wrong campaign/source/v2 profile, missing/oversized snapshot, stale/changed code
semantics, invalid indices, malformed/ambiguous JSON, symlink/mutation/path race,
unavailable memfd/read permission, timeout, unknown predicate и limits дают
closed failure или локальный UNVERIFIED согласно Q1/Q4. Никаких chmod, fallback,
установок или восстановления отсутствующего snapshot по golden/другой модели.

## Критерии приёмки

| ID | Исполняемые доказательства будущей реализации |
| --- | --- |
| Q1 | Strict v3 context/digests/hashes и6 tuple replay; actual review/scorer equivalence, tuple-local negatives и общий fail-closed integrity |
| Q2 | Actual unmapped positions и reason→public-code mapping для всех6 tuples, full finite allowlist, duplicate-order/sanitizer/unsupported-code tests; no role inference |
| Q3 | Actual clock/row/printed-turnover/reconciliation witnesses и causal near misses, независимый golden, без literals/repair или новой шкалы |
| Q4 | Writer→strict reader→pure diagnosis→closed wrapper, privacy/sentinel/file-race/limits/timeouts PASS, неизменные legacy artifacts/readers |
| Q5 | Real helper, full mode matrix/YAML parser, exact-source/manifest gate, make test/test-production/integration/vet/diff checks и CI5/5 до Q6 review |
| Q6 | Только после отдельного approval один read-only dispatch, artifact/marker unchanged,6 tuples VERIFIED либо явные UNVERIFIED; provider/SQL/writes0, historical E6 FAIL/CONSUMED сохранены |

## Исполняемые проверки

Будущая реализация добавляет свои causal/strict-reader/runner тесты в обязательные
Go/production gates, точные команды и результаты перечисляет в evidence Q1–Q5.
Общие проверки:

```sh
TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test
TMPDIR=/tmp make test-production
make test-integration
go vet ./cmd/... ./internal/...
git diff --check
```

Это offline/local gates без Q6 dispatch или paid calls. В текущем docs-only пакете
проверяются документные ссылки, отсутствие runtime diff, make test и diff check;
это не evidence реализации Q1–Q5 и не live quality PASS.

## Открытые решения

1. Q1–Q5 приняты. Интеграция принятой спецификации PR №103 в main ещё
   не выполнена; implementation not_started до неё. Затем отдельный implementation
   worktree от актуального main по [spec-first процессу](README.md#жизненный-цикл).
2. Q6 exact source/ref/manifest и одно разрешение чтения — после реализации;
   сейчас отсутствуют и не выводятся из «продолжай работу».
3. После диагноза — отдельное решение о минимальном ремонте. Новая paid campaign,
   production switch и SPEC-0056 не становятся разрешёнными автоматически.
