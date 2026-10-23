# SPEC-0055 / S1.20: объяснить сохранённый REJECT без новых model calls

- Статус: **accepted R1–R3 offline**; implementation_status: **integrated offline (PR110)**.
- Автор: Codex; дата: 2026-10-23, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: прямое решение владельца ниже.
- Proposal branch: `codex/statement-s119-pilot-evidence`, тот же PR108.
- Implementation branch: `codex/statement-clock-reject-diagnosis`; base main44e6413 после PR108.
  Код принят владельцем и интегрирован PR110/main2979280; exact source/CI и R4 receipt ниже.
- Offline R1–R3 приняты и контракт интегрирован PR108 до реализации. Live R4 отдельно:
  **AUTHORIZED / NOT_RUN** по отдельному receipt ниже; E6/Q6/S1.19 permissions остаются consumed.

## Проблема и пользовательский результат

Конечная цель SPEC0055 — надёжный проверяемый черновик банковской PDF-выписки
с исправлениями и явным подтверждением владельца. Для следующего ремонта нужно
знать, какие именно критерии качества не прошли; новый prompt не выбирается наугад.

Единственный S1.19 [run39983522098](https://github.com/QahNaaRlN/grace/actions/runs/39983522098)
дал6/6 completed, clocks `[0,4,4,0,0,4]`, known/settled и **REJECT**.
Старый публичный summary не экспортирует precision и non-clock checks или actual
сумму. По коду judge известен класс возможных причин, но конкретные проверки
не установлены. Private typed report сохранён; заново вызывать модель для
восстановления уже существующих данных не требуется.

Полезный результат — точные закрытые коды отказа, все checks/precision по всем
шести ответам и проверяемая стоимость. Это основание следующего решения,
не новая qualification, принятие candidate или разрешение на его ремонт.

## Scope

Один пакет R1–R4: чистая проекция уже сохранённого ClockReport, фиксированный
read-only reader, закрытый output и один отдельно разрешённый запуск.
Результат/approval/source receipt добавляются в тот же исследовательский пакет;
отдельные readiness/result спецификации и PR для каждой записи не нужны.

R1–R3 — offline implementation/verification после принятия. До этого разрешены
только проектирование и чтение кода/public fixtures. R4 не открывается принятием
offline контракта: exact reader source/ref/CI/manifest/inputs предъявляются позже.

## Non-goals

Новые inference/metadata calls, новый pilot/candidate/model, prompt repair,
финансовые записи, SQL/ledger reads или reserve/settle, deploy/flags/activation,
API/UI/schema/migrations. E6/P6/D5/Q6 private artifacts и их локальные копии
закрыты. Не читаются env/config/keys, исходные пользовательские выписки или
произвольные пути. Нельзя менять S1.19 report/marker, frozen sources, decoder,
scorer/guards или интерпретировать прежний REJECT как PASS.

## Текущее поведение и доказательства

Producer source: `8da7f45b105105d6596032850893bb690febebd8`,
ref `codex/statement-s119-pilot-source`, версия `statement.clock-pilot.s119.v1`.
Public summary/manifest/run receipt — [S1.19 evidence](0055-s119-evidence.md).
`DecodeClockReport` проверяет JSON shape/digest/context и заново считает все6
completed snapshots через canonicalization → merge → validation → scorer.
`DecideClock` учитывает candidate target, precision и base-PASS→candidate-FAIL.
Новые правила качества не нужны. Существующий S1.18 sealed-memfd transport и
O_NOFOLLOW/stable-read guards служат образцом; его identity/режимы не расширяются.

Известные runtime файлы на Grace VPS:
`/opt/grace/shared/statement-s119-clock-pilot.started` и
`/opt/grace/shared/statement-s119-clock-pilot.json`.
Их наличие/состояние сейчас не опрашивалось. Marker содержит producer SHA,
а report не содержит подписанного GitHub run id. Связь с run подтверждается
fixed source/marker/context и совпадением с опубликованным summary; это не
криптографическая аттестация происхождения файла.

## Предметная модель и инварианты

### R1. Полное детерминированное объяснение

Добавляется чистая проекция в `internal/statementqualification/clock_diagnosis.go`.
Вход проходит прежний `DecodeClockReport`, producer Source обязан совпасть с
фиксированным8da7f45, document — с публичным s119-cards-document.json и его hash.
Ожидаются6 completed, error пуст, REJECT, counts `[0,4,4,0,0,4]`, known/settled,
как в опубликованном summary. Любое несовпадение — UNVERIFIED, не новая история.

Output version: `statement.clock-diagnosis.s120.v1`; shape закрытая:

- readerSource/producerSource (40 lowercase hex), reportSha256 и reportDigest
  (64 lowercase hex), producerRun=39983522098, status=VERIFIED, replayVerified=6,
  decision REJECT, activationAuthorized=false;
- attempts: ровно6, исходный ordinal/variant, clockCount, precision[4],
  qualityPassed/correct/checksCount и ровно20 checks `{ordinal,rowOrdinal,code,passed}`;
- pairs: ровно3 с pairOrdinal, baseOrdinal/candidateOrdinal, полный список
  nonClockRegressions `{ordinal,rowOrdinal,code,basePassed:true,candidatePassed:false}`
  по20-check trace, precisionFailures `{rowOrdinal,field}` (field=operation/booking)
  и boolean candidateClockTargetFailed;
- rejectReasons: исчерпывающий набор из `candidate_clock_target_failed`,
  `candidate_precision_failed`, `non_clock_regression`, вычисленный из полных данных;
- accounting: callCostsMicros[6] и actualMicros из report, сумма actual проверена
  по6 bills, maximumMicros=323400, billingKnown/settlementRecorded=true.
  Это сохранённый accounting, не свежая проверка SQL ledger/fees/allowance;
- providerCalls=0, sqlCalls=0, persistentWritesRequested=0.

Code — только фиксированный whitelist нынешних scorer codes; ordinals1..20,
rowOrdinal0..2, precision booleans в operation/booking порядке двух строк.
Неизвестный code не скрывается и не преобразуется в объяснённый unknown:
весь diagnostic UNVERIFIED/unsupported_check_code. Все120 checks сохраняются,
включая failures, общие для base и candidate; список регрессий не заменяет trace.
Raw values/text/spans/service reasons/proposals, identities/reservation/generation
IDs, routes/provider error strings не экспортируются. Никаких обрезанных массивов.

На CI runner закрытая схема повторно проверяется и решение заново вычисляется
по опубликованным booleans/ordinals, включая равенство rejectReasons и pair diffs.
Финансовые values не нужны для этой независимой проверки C4.

### R2. Фиксированное чтение с прежней защитой

Новый CLI `scripts/research/s120-diagnose` читает только два fixed S1.19 файла
выше. CLI modes: `-mode=check` (default), `-mode=read`, `-mode=validate`.
Check не открывает private files. Read/validate требуют `-authorized-source`,
равный compiled readerSource (`-ldflags -X main.readerSource=<SHA>`);
producer source не параметр. Read принимает public context из stdin,
validate — только закрытую projection из stdin и не читает private файлы.
Public document передаётся в bounded envelope, hash проверяется до report decode.
Нет generic path/source/report overrides. Старые reader/CLI не меняются.

O_RDONLY/O_NOFOLLOW/O_NONBLOCK, regular files0600; marker ≤41 bytes,
report ≤1048576 bytes. Stable reads до/после сравнивают inode/device/size/mode/
mtime/hash; несовпадение или отсутствие — закрытый отказ. Atime/OS SSH logging
не выдаются за прикладные записи; содержимое/mode/mtime файлов не меняются.
Ни PDF helper, ни SQL, ни provider/config clients для reader не нужны.
Private bytes обрабатываются только в памяти VPS и не идут в stdout/artifacts.

### R3. Один bounded operator path, без нового framework

Один отдельный workflow `.github/workflows/statement-clock-diagnosis.yml`,
один research job, environment/concurrency production; default false.
Inputs: authorized_source (40 SHA), execute_approved_diagnosis (boolean).
Gate: reader SHA==GITHUB_SHA, source ancestor main, exact CI5/5, source manifest.
Default false не вызывает SSH. Monolithic deploy workflow не расширяется.

Переиспользуется паттерн S1.18: pinned SSH, передача compiled static binary и
public context через sealed memfd bootstrap, Python -B, без дисковых bundle/
marker/report writes, без Docker/container/image changes. Fixed host/principal
прежние Grace VPS/grace-deploy. Одна SSH subprocess, timeout75s; binary≤16MiB,
envelope≤32MiB, context≤32768 bytes, закрытый stdout≤65536 bytes.
Capture/schema/limit failure — закрытый отказ, без усечения или повторов.
Typed UNVERIFIED/cause сохраняется при ненулевом exit status; произвольный stdout,
неверная schema/source или отсутствие typed отказа дают unavailable_runtime,
а не вывод private error. Nonzero status не стирает корректный закрытый cause.
Нет private witness/raw-response artifact. В Git/CI только source/manifest,
закрытый diagnostic и его digest; stderr не переносит произвольные строки.

Минимальные seams/functions для текущего report; не создавать generic evaluator,
второй scorer, универсальный replay framework или режим следующего pilot.

### R4. Один отдельный live допуск и остановка

После integration и exact source CI5/5 предъявляются reader source/ref,
manifest и inputs. Только затем отдельное прямое разрешение на один
workflow_dispatch и read-only SSH-разбор сохранённого S1.19. Ни платных вызовов,
ни SQL, ни новых metering writes. До этого live NOT_AUTHORIZED / NOT_RUN.

Разрешение учитывается pre-dispatch receipt и одним CI run/attempt1; нового
persistent one-shot marker на VPS не создаётся. Отказ тоже расходует один допуск.
После VERIFIED или UNVERIFIED — итог в этом пакете и остановка. Никаких rerun,
скрытого второго SSH, самостоятельного ремонта candidate или другого report.

## API, OpenAPI и UI

Публичный продуктовый контракт и generated types не меняются. Только закрытая
operator projection с отдельной версией; это не product endpoint/confirmation.
Manual path/owner review продолжают работать независимо от AI.

## Данные, миграция и rollback

Новых таблиц, financial/budget writes, private copies или retention изменений нет.
Rollback — удалить/отключить reader path; исходный S1.19 report/marker сохраняются.
R1–R3 acceptance не разрешает читать их до R4 exact-source approval.

## Ошибки и негативные сценарии

Внешний отказ имеет только поля version, readerSource, status=UNVERIFIED,
reason и activationAuthorized=false. Version фиксирована statement.clock-diagnosis.s120.v1, readerSource
равен compiled SHA. Конечный reason enum: snapshot_unavailable, source_mismatch, context_mismatch,
report_invalid, snapshot_changed, summary_mismatch, unsupported_check_code,
diagnosis_limit_exceeded, output_mismatch, unavailable_runtime.
Это диагностический UNVERIFIED; исторический REJECT не меняется.
Нет fake zero cost/checks/precision на ошибке. Никаких private strings в error.

## Критерии приёмки и исполняемые проверки

| AC | Требуемое доказательство |
| --- | --- |
| R1 | Actual pipeline synthetic controls для target/precision failure, non-clock regression и комбинации; все120 checks и3 pair diffs; одинаковые input → bytes; forged score/precision/digest/context/summary rejection |
| R2 | Only fixed marker/report; missing/symlink/FIFO/mode/oversize/mutation/replacement tests; bytes/inode/mode/mtime до/после; no private/config/network/SQL access в default mode |
| R3 | SSH stub ровно1 subprocess, no persistent writes; default false/unauthorized/CI/source/limit отказ до SSH; private sentinel отсутствует в stdout/artifacts; closed schema и независимый C4 recompute |
| R4 | Только отдельный exact-source grant: один run, status VERIFIED либо честный UNVERIFIED; report сохранён, provider/SQL/writes0, вывод без payload, остановка |

Обязательные offline gates: `go test ./internal/statementqualification ./scripts/research/s120-diagnose`,
hermetic Python bootstrap/SSH tests, existing pinned-helper causal gate,
`make test`, `make test-production`, `make test-integration`, `go vet` и
`git diff --check`. Evidence реализации приведено в конце этого документа. Offline fakes не заполняют R4 и не устанавливают причину
реального REJECT. Новые assertions используют actual report decode/replay,
не фабрикуют Review или итоговый score. Недостижимые для текущего pipeline
комбинации не превращаются в искусственный positive report ради теста ветки:
поддельная cached precision/score — отрицательный replay-mismatch case.

## Решение владельца

2026-10-23 владелец ответил **«Принять, интегрировать и продолжить offline»**
на вопрос о принятии R1–R3 в PR108, review head
`db1f8642b4d634ffd489346fa4e20fb880f7f281`, записи acceptance, слиянии PR108 и
PR109 после exact-head CI5/5 и последующей offline-реализации reader.
Это acceptance контракта; на момент решения implementation_status оставался **not_started**.
Ход последующей offline-реализации зафиксирован ниже.
R4 — **NOT_AUTHORIZED / NOT_RUN**: чтение закрытого отчёта VPS требует
отдельного разрешения на готовый exact source. Новый prompt/model pilot,
повтор S1.19 или автоматическое слияние будущего implementation PR не разрешены.


## Проверка подготовки draft (не R1–R4 implementation evidence)

Существующий `go test ./internal/statementqualification -run '^TestClockCausalAndJudge$' -count=1 -v`
— PASS, `/tmp/grace-s120-design-baseline.log`: подтверждены нынешний pipeline,
clock gain/REJECT/non-clock regression/incomplete/N/A branches. На момент подготовки draft новый reader ещё не был создан.
`TMPDIR=<private /var/tmp> NODE_DISABLE_COMPILE_CACHE=1 VITEST_MAX_WORKERS=1 make test`
— PASS, `/tmp/grace-s120-proposal-make-test.log`,88 Web files/334 tests плюс Go,
lint/FSD/Prettier/build/offline Jev; `git diff --check` PASS. Executable sources
не менялись. Private TMPDIR удалён; private report/VPS/provider не опрашивались.

## Интеграция принятого контракта и offline evidence R1–R3

Это отчёт исполнителя для review, не owner acceptance реализации и не R4.
Спецификация принята до кода на review head `db1f8642b4d634ffd489346fa4e20fb880f7f281`.
Запись acceptance — `603e112e340b65b9a1aabecaa8d26b0bb26510f7`,
[CI39993532400](https://github.com/QahNaaRlN/grace/actions/runs/39993532400) 5/5.
PR108 интегрирован merge `44e641369da55e8ae0b2e374370707cd14710e72`;
[main CI39995240441](https://github.com/QahNaaRlN/grace/actions/runs/39995240441) 5/5,
[Delivery39996404889](https://github.com/QahNaaRlN/grace/actions/runs/39996404889) 4/4.
Implementation worktree/ветка `statement-clock-reject-diagnosis` /
`codex/statement-clock-reject-diagnosis` создана из main44e6413 после интеграции
принятого контракта и до кода. Main CI/Delivery выше подтверждены позднее.

Разрешённый PR109 интегрирован отдельно: head `e0acf8ea8cc9e5a8b681fa0e2a2107ef331a1ca1`,
[CI39990561103](https://github.com/QahNaaRlN/grace/actions/runs/39990561103) 5/5,
merge `c09308920f29787d7812b9118292c0fbf0f4259b`,
[main CI39993577486](https://github.com/QahNaaRlN/grace/actions/runs/39993577486) 5/5,
[Delivery39994741222](https://github.com/QahNaaRlN/grace/actions/runs/39994741222) 4/4.
Локальные post-merge `make test`, `make test-production`, `make test-integration`,
`go vet`, diff-check на c093089 PASS (`/tmp/grace-s120-postmerge109-*.log`).
PR108 добавил только документы; exact merge CI повторил полные gates.
Delivery публикует образы; нового deploy/activation не было.

| AC | Реализация и исполняемое evidence |
| --- | --- |
| R1 | `internal/statementqualification/clock_diagnosis.go`; прежние decoder/scorer/judge не изменены. `go test ./internal/statementqualification -run '^TestClockDiagnosis' -count=1 -v` PASS, `/tmp/grace-s120-r1.log`. Fake reasoner проходит настоящий RunClockPilot → Encode/Decode/replay; полные120 checks, shared failures, три пары включая reversed order, суммы bills и повторяемые bytes. Forged precision/score/digest/context/source/decision/unknown code отклонены |
| R2 | `scripts/research/s120-diagnose`; `go test ./scripts/research/s120-diagnose` PASS, `/tmp/grace-s120-r1-r2.log`. Синтетические файлы0600: успешное чтение и неизменность bytes/inode/mode/mtime; missing/symlink/FIFO/directory/mode/oversize/partial/marker source; mutation/mtime/mode/replacement/symlink/remove между reads. Default/check и validate не вызывают private reader; неверный source/context останавливаются до него |
| R3 | `python3 -B scripts/research/test-statement-s120-diagnosis.py` PASS, `/tmp/grace-s120-r3-final.log`:31 Go/Python parity cases, настоящий sealed memfd/пустой env, nonzero typed cause, stdout overflow/timeout, один SSH stub, отказы до SSH. Исполнен actual YAML guard с hermetic gh/git для source/ancestor/CI SHA/skipped job/rerun. `node scripts/research/check-workflow-syntax.mjs` PASS. Live SSH0 |
| R4 | **NOT_AUTHORIZED / NOT_RUN**. Истинная причина S1.19 REJECT, фактическая сумма и наличие/состояние fixed VPS files остаются UNVERIFIED. Ни synthetic result, ни успешно собранный reader их не заменяют |

`go test -race ./internal/statementqualification ./scripts/research/s120-diagnose`
PASS (`/tmp/grace-s120-race.log`). Actual pinned helper gate в `make test-production`
подтвердил тождество public JSON настоящему Poppler25.04.0/bbox-v1: good20/20,
clocks4; date-only14/20, clocks0, coverage failure сохранён.

Target/precision controls получены из настоящего pipeline и корректно дают
`summary_mismatch`, когда отличаются от исторических `[0,4,4,0,0,4]`.
Их реальные измерения отдельно проверяют вычисление нескольких reject causes.
Precision-only failure при четырёх верных clocks текущий pipeline не производит;
поддельная cached precision проверяется как отрицательный replay mismatch.
Валидный synthetic REJECT с историческими clock counts возникает от non-clock
регрессии. Это тест reader, не утверждение о причинах сохранённого live report.

При review дополнительно закрыты два случая: Python больше не принимает UTF-16/BOM,
отклоняемые Go; comparison manifest различает bool/int. Workflow явно запрещает
`GITHUB_RUN_ATTEMPT != 1` до SSH. Финальные narrow parity/YAML gates повторены
после этих изменений; ошибок полноценного gate не скрыто.

Полные offline gates реализации: `make test` PASS (89 Web files/336 tests, Go,
lint/FSD/format/build/offline Jev), `make test-production` PASS,
`make test-integration` PASS, `go vet ./cmd/... ./internal/... ./scripts/research/s120-diagnose`
PASS, `git diff --check` PASS. Логи: `/tmp/grace-s120-make-test.log`,
`/tmp/grace-s120-production.log`, `/tmp/grace-s120-integration.log`, `/tmp/grace-s120-vet.log`.
`make test` использовал private `/var/tmp` TMPDIR с удалением после завершения:
унаследованный `/tmp/.git` не менялся. Disposable Docker suites завершены с cleanup.

Reader не создаёт marker/report на VPS и не имеет path/producer override.
Единственный workflow default false; вывод — закрытый diagnostic или enum failure.
Реальный запуск потребует integration реализации, exact-source CI5/5,
manifest/ref/inputs и отдельный R4 grant. Ни новой версии prompt, ни model calls,
SQL/ledger/reserve/settle, финансовых записей или deploy этот пакет не выполнял.
Исходный S1.19 **REJECT** сохранён; общая цель надёжного AI-импорта не закрыта.


## Принятие реализации, интеграция и R4 pre-dispatch receipt

Владелец принял offline-реализацию прямым **«Принять реализацию и слить №110»**
на review source `1a6b74201db92ba37be750411ff7883d8216b602`.
Принятие сохранено в [PR110](https://github.com/QahNaaRlN/grace/pull/110)
и merge commit `29792809f5e6435c93fe07a5170e7ed7135d38ca`.
Source [CI39998179098](https://github.com/QahNaaRlN/grace/actions/runs/39998179098)
и merge [CI39999830331](https://github.com/QahNaaRlN/grace/actions/runs/39999830331)
SUCCESS5/5; [Delivery40001135152](https://github.com/QahNaaRlN/grace/actions/runs/40001135152)
SUCCESS4/4. Canonical verifier и все jobs проверены отдельно по exact SHA.
Локальные post-merge make test (89/336), test-production (31 parity cases/helper),
test-integration, vet/diff-check PASS; `/tmp/grace-s120-postmerge-*.log`.
Деревья merge/source совпадают: `701cc3b2b2a73af8ebe411c0433d18f5701b0472`.
Main чистый0700; frozen E6/Q6/S1.19 refs сохранены.

До этого receipt R4 не разрешался и не выполнялся. Далее владелец ответил
**«Разрешаю один R4»** на exact-source вопрос:

> Разрешаете один R4 на source 1a6b74201db92ba37be750411ff7883d8216b602,
> ref codex/statement-s120-reader-source: один workflow_dispatch
> statement-clock-diagnosis.yml с authorized_source=этот SHA и
> execute_approved_diagnosis=true, один read-only SSH-разбор сохранённого
> S1.19 на Grace VPS? Provider/metadata/SQL calls, новые прикладные VPS writes
> и deploy — 0; в CI только закрытый diagnostic. Отказ тоже расходует допуск,
> повторов нет. Отдельного разрешения требует принятый S1.20 R4.

Receipt подготовлен 2026-10-23T18:16:50.556017+00:00, **до dispatch**. Статус: **AUTHORIZED / NOT_RUN**.

- Frozen reader source `1a6b74201db92ba37be750411ff7883d8216b602`;
  ref `codex/statement-s120-reader-source`, remote ref повторно проверен.
  Source ancestor origin/main; exact-source CI5/5 повторно проверен.
- Manifest: `docs/research/statement-quality/s120-reader-manifest.json`,
  SHA256 `6c4b72a138fcee3a7ee9dcaed8d390ec0f03a43240f91a9f81aa167bf4f21be7`.
  `-mode=check` не открывает private files.
- Workflow `statement-clock-diagnosis.yml`;
  `authorized_source=1a6b74201db92ba37be750411ff7883d8216b602`,
  `execute_approved_diagnosis=true`; ровно один dispatch/run, attempt1.
- Один SSH subprocess с pinned host, timeout75s. Только fixed S1.19 marker/report
  `/opt/grace/shared/statement-s119-clock-pilot.started` и
  `/opt/grace/shared/statement-s119-clock-pilot.json`, stable reads/0600,
  без произвольных путей и private witness downloads. Их runtime состояние
  до разрешённого запуска не опрашивалось.
- Дополнительные provider/metadata/SQL/reserve/settle calls и новые прикладные
  persistent VPS writes0. Никаких deploy/flags/prompt/model changes.
  Только закрытая projection/digest либо enum UNVERIFIED; отказ расходует grant.
- На момент подготовки receipt Actions metadata содержит **0 прежних R4 runs**.
  После одного результата — запись outcome в этом же пакете и остановка.
  Receipt не разрешает второй SSH/dispatch, новый report, repair или pilot.

Это один пакет acceptance/source/approval/result. Отдельного readiness PR
или новой спецификации для этой записи не создаётся; outcome добавляется сюда.
