# S1.14 / P6: проект решения об одном observed synthetic comparison

- Статус: draft execution decision; acceptance record: отсутствует.
- Дата: 2026-10-21, Asia/Yekaterinburg; acceptance owner: владелец Grace.
- Live status: NOT_AUTHORIZED / NOT_STARTED. Ни metadata GET, ни SSH/VPS,
  workflow dispatch, provider inference или deployment при подготовке не выполнялись.
- Основание подготовки: «идем дальше» после offline P1–P5 и PR №91.
  Это открывает интеграцию и подготовку следующего решения, не paid execution.
- Основание реализации: [accepted S1.14](0055-s114-observed-comparison.md),
  [P1–P5 evidence](0055-s114-evidence.md), [post-merge №91](0055-pr91-postmerge.md).
- **Предложенный exact source SHA:** `301afd6bedc4a19559e731471336bbc4e2f74075`,
  implementation head [PR №91](https://github.com/QahNaaRlN/grace/pull/91), уже достижимый из main.
- Source ref: `codex/statement-observed-comparison`, закреплён на SHA выше;
  source worktree `statement-observed-comparison` сохраняется чистым.
- Это кандидат source, **не** already-authorized source. Принятие P1–P5,
  merge PR №91, публикация ref и слияние этого документа не разрешают P6.

## Предлагаемое решение

Разрешить **одну** новую campaign authorization `s114-comparison-20261021`
на exact source выше, только при успешных post-merge/local/CI gates и runtime
preflight. Общий предел: **до6POST / 208500 USD micros ($0.2085 provider)**.
Одно owner approval покрывает один dispatch с preflight и возможной campaign;
preflight refusal завершает эту попытку, автоматического повторного dispatch нет.

| ID | Request alias → exact canonical | Exact endpoint / display provider | Provider max/document | Provider max/3 documents |
| --- | --- | --- | --- | --- |
| A | google/gemini-3.1-flash-lite → google/gemini-3.1-flash-lite-20260507 | google-vertex/eu / Google | 53900 micros | 161700 micros |
| B | inception/mercury-3 → inception/mercury-3-20261019 | inception / Inception | 15600 micros | 46800 micros |

Это существующие source pins/ceilings; свежая доступность и реальные цены
**UNVERIFIED** до отдельно разрешённого preflight. Никакой модели/route замены
или повышения caps при несовместимом metadata proof. Inception допускается только
для public synthetic fixtures; EU residency всей цепочки не заявляется.

Порядок фиксирован: A-columns → B-columns → A-cards → B-cards →
A-turnover-roles-ru → B-turnover-roles-ru. На tuple не более1POST;
input100000/output16000,120s/call,900s inference context, settlement5s.
До запуска проверяются все6plans и общий reserve208500. Общий прежний $10 ledger,
remaining, historical costs и fees сохраняются. Provider cap не выдаётся за
fee-inclusive цену; бюджет не увеличивается и не сбрасывается.

Только3 прежних публичных synthetic PDFs с exact hashes S1.11. Planner, prompt,
schema, redaction, aliases, scorer19/20/23 и S1.10 guards не меняются. Личные PDF,
owner drafts, финансовые записи и product import этому процессу не передаются.

## Что уже проверено и что не известно

P1–P5 реализованы source code9db80af; final implementation head301afd6 прошёл
[CI39612267914](https://github.com/QahNaaRlN/grace/actions/runs/39612267914)5/5.
Private snapshot/diagnostics/context, legacy compatibility, one-shot lifecycle,
real helper/stdin CLI replay, privacy и disposable SQL проверены offline.
Fake scores и test costs не являются live quality/billing.

[Post-merge evidence](0055-pr91-postmerge.md): make test/production/integration,
vet/diff PASS на merge tree68ee7ee; этот tree byte-identical выбранному301afd6.
Exact source301afd6 имеет настоящий CI39612267914: 5/5 PASS и уже интегрирован
в main. На новый source не выдаётся CI другого SHA.

GitHub не зарегистрировал push-CI/Delivery для main merge68ee7ee и двух пустых
integration commits c23f4c1/fd68235; их состояние остаётся UNVERIFIED. Их trees
идентичны301afd6; это проверено git diff и ancestry. Поэтому **кандидатом P6 выбран
ранее проверенный integrated301afd6**, а не непроверенный новый main SHA.
Source301afd6/CI5/5 и отсутствие новых main runs — разные утверждения. Это явный
выбор кандидата до owner approval, не перенос старого C6 permission или замена
уже одобренного source. Primary main сейчасfd68235; source ref301afd6 frozen.

Actual runtime image/health, global AI/key, helper, один active admin, marker/
report absence, shared remaining/uncertain и fresh catalog/endpoint/ZDR proofs
обеих пар **UNVERIFIED** до P6 preflight. Новый source/authorization не снимает
shared block и не разрешает reconciliation, изменение environment или budget.

Старый C6 остаётся FAIL canonicalization, exact guard UNVERIFIED, cost1995micros
accounted/settled и authorization consumed. Старые reports/markers/ledger не
перезаписываются, не удаляются и не становятся разрешением на новую попытку.

## Обязательный preflight после отдельного approval

До любого POST: exact ref/SHA и CI5/5, полный SHA256SUMS operator bundle
(admin/replay binaries,3PDFs,SOURCE_SHA,runner,оба inspector scripts), прежний
shared exclusive lock, отсутствие новых marker/report/preflight/exit artifacts
включая symlinks/orphans, readiness image/health/helper/global AI/key и единственный
active admin. Fresh metadata обязана подтвердить exact пары, ZDR/profile и цены
в пределах pins. Все6plans и reserve должны помещаться в существующий budget.
`grace_statement_call_allowed` проверяется перед каждым Complete.

Неисправность/несовпадение даёт закрытый отказ без обхода guards, другой модели,
смены source или повторного smoke. Если source ref сдвинулся — вернуть source
на owner review, не заменять SHA в input молча. Proof GET/SSH входят только в
отдельно разрешённый P6, не в подготовку этого документа.

## Остановки, accounting и сохранность

Known quality/grounding FAIL позволяет закончить оставшиеся заранее разрешённые
tuples. Protocol/transport/identity/privacy/cost/timeout, unknown billing,
settlement failure или shared block останавливают всю campaign. В частности,
canonicalization FAIL A/columns не разрешает отдельно вызвать B. Not_sent tuples
не переносятся; retry/fallback/repair и дополнительные пробные POST запрещены.

Фиксированный marker `statement-comparison-s114-comparison-20261021.started`
создаётся no-clobber до первой возможной отправки, report с таким же identity
пишется0600. Private envelope `statement.comparison.observed.s114.v1` сохраняет
прежний report и ограниченный typed alias-proposal до canonicalization,
closed diagnostic и исходные inner/outer digest. Snapshot<=32768bytes/tuple,
artifact<=524288bytes; overflow отбрасывает snapshot целиком и не меняет billing.
Capture не crash-durable: убийство процесса не гарантирует полный artifact.
Отсутствие snapshot/invalid artifact означает UNVERIFIED, не право повторить POST.

Одна новая reservation model=`statement-comparison.s114`, provider=`fixed-A-B`,
purpose=`statement`; exact UUID берётся только из verified report. Known costs
всех attempts учитываются точно, unknown не становится0 и блокирует дальнейшие
paid actions. Automatic generation metadata GET/recovery/reconciliation не входят.
Preflight failure, consumed marker или accounting failure не дают права очистить
старые/новые artifacts или увеличить budget.

После run входит только read-only inspection **нового fixed artifact**: оба
digests/context, exact unique ledger correlation/status/amount/time и offline
replay captured proposal по bundled public PDFs/pinned Unix helper. Replay не
вызывает модель, не читает key/config и не выполняет SQL. Bytes/mode/mtime
report/marker должны совпасть до/после inspection. Unavailable/mismatch остаётся
UNVERIFIED, без guessed guard или нового recovery run.

Public stdout/stderr/Actions/evidence содержат только closed enums/bools/counts
и принятую numeric cost/latency проекцию. Alias strings, raw proposal/service/
error/model text, generation/reservation/owner IDs и ключи не выводятся и не
публикуются Actions artifacts. Private artifact остаётся на прежнем host storage.

## Параметры будущего исполнения — не команда запуска

Только после прямого owner approval оператор выбирает существующий workflow
`Deploy production` с **ref `codex/statement-observed-comparison`**:

| Input | Единственное допустимое значение |
| --- | --- |
| statement_comparison_s114_only | true |
| comparison_s114_authorized_sha | 301afd6bedc4a19559e731471336bbc4e2f74075 |
| statement_action | preserve |
| release_sha | empty |
| Все другие screening/comparison/diagnose/inspect modes | false |
| Оба reconcile modes | none |

Выбирается только S1.14 comparison job. Default deploy, old C6, owner-test enable,
reconcile и diagnostic jobs не входят. Название workflow не является разрешением
на deployment. Source pin не должен сдвигаться после approval; будущий docs merge
не изменяет этот executable source. До owner approval dispatch отсутствует.

## Результат для владельца

После единственного run или preflight refusal: exact source/run/job IDs,
количество отправок и not_sent, tuple quality/false acceptance, closed stage/
guard/capture/replay, known cost и Complete latency, reservation known/uncertain,
image/health до/после. Complete latency включает proof/network/decode и не является
pure generation; при n<=3 не заявлять p95. При incomplete — comparison/winner
UNVERIFIED. Даже PASS3/3 обеих пар не даёт production qualification/activation
или подтверждения финансовых данных пользователя. Production adoption отдельно.

## Требуемое решение владельца

Acceptance record отсутствует. Конкретный предлагаемый ответ:

> Разрешаю один P6 на 301afd6bedc4a19559e731471336bbc4e2f74075,
> authorization s114-comparison-20261021, до6POST/$0.2085 provider,
> при обязательном preflight и прежнем shared budget, без retry/deploy/model switch.

До такого отдельного решения live_status остаётся NOT_AUTHORIZED.
