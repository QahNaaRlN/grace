# История разработки Statement AI до пересмотра S1.19

Архив с main1a5bbfb, 2026-10-23. Формулировки «текущий» ниже относятся к
историческим срезам, а не к состоянию runtime сегодня. Отрицательные результаты
и использованные разрешения сохранены. Актуальное состояние —
[одна status-страница](../statement-ai-status.md).

## Исторический индекс спецификаций

Актуализация разработки: **2026-10-23**, Asia/Yekaterinburg.

Текущий пакет — [S1.18: диагностика качества сохранённого E6](0055-s118-saved-quality-diagnosis.md),
**accepted Q1–Q5 / integrated offline, PR №104; Q6 EXECUTED / SUCCESS**.
[Post-merge №104](0055-pr104-postmerge.md): main438256b clean, head/merge CI5/5
и Delivery4/4 PASS. [Q1–Q5 evidence](0055-s118-evidence.md): все local gates PASS.
После отдельного «Разрешаю один Q6» на frozen5fdd0f9/ref/CI5/5 опубликован
[receipt](0055-s118-q6-execution-receipt.md), затем один
[run39949603261](https://github.com/QahNaaRlN/grace/actions/runs/39949603261),
только Q6, остальные12 jobs skipped. [Q6 evidence](0055-s118-q6-evidence.md):
6/6 tuples,22 failed checks и23 reasons VERIFIED,15 public unknown сопоставлены;
artifactsUnchanged=true, providerCalls0, persistentWritesRequested0.
Публично только safe status/counts, closed witnesses не переносятся в Git/PR/artifacts.
Permission CONSUMED; исторический E6 quality FAIL сохранён. Ремонт, новый paid
run, deployment/model/guard switch и SPEC-0056 этим результатом не открыты.

Завершённый пакет — [S1.17: контролируемое сравнение после ограничения alias](0055-s117-alias-comparison.md),
**accepted / integrated offline, PR №100/№101; E6 EXECUTED / FAIL quality**.
[YAML blocker](0055-s117-e6-readiness.md) устранён №101 по прямому разрешению
владельца после CI5/5; [post-merge](0055-pr101-postmerge.md).
[E6 receipt](0055-s117-e6-execution-receipt.md) опубликован до одного
[run39925874197](https://github.com/QahNaaRlN/grace/actions/runs/39925874197) на frozen24f7322/CI5/5.
[Результат](0055-s117-e6-evidence.md):6POST, canonicalization/capture/replay6/6,
качество0/3 у обоих, A17/19–14/20–17/23, B17/19–18/20–19/23.
10897 USD micros ($0.010897 provider) known/settled; shared uncertain=false,
image/health preserved. Permission CONSUMED; повтор/deploy/model switch не открыты.
Read-only диагностика saved E6 выполнена S1.18 Q6 (см. выше); исходный quality FAIL не исправлялся.

Интегрированный offline пакет — [S1.16: допустимые alias во всех selections](0055-s116-alias-schema.md),
**accepted / integrated offline, PR №98**. Владелец принял A1–A6 прямым «принимаю»
2026-10-23: exact delivered-alias enum,
прежняя numeric eligibility, wire/bytes/consent проверки только offline.
Принятая спека интегрирована PR №97 (`a1cc072`), реализация — PR №98 (`81ef7e5`).
[Post-merge №98](0055-pr98-postmerge.md): local/head/merge gates PASS; CI head и merge оба5/5. Delivery отдельно.
[Evidence A1–A6](0055-s116-evidence.md), [post-merge №97](0055-pr97-postmerge.md).
Scope S1.16 не включал inference/SSH/deploy или новую campaign.
[PR №96 post-merge](0055-pr96-postmerge.md): receipt интегрирован в main `5b6ed20`.

Текущий пакет — [S1.15](0055-s115-saved-proposal-diagnosis.md), **accepted D1–D4 / integrated offline**:
[PR №94](https://github.com/QahNaaRlN/grace/pull/94) слит, merge `17e3493`.
Source `703b6c6`: CI 5/5; merge: CI 5/5 и Delivery 4/4 PASS;
[post-merge evidence](0055-pr94-postmerge.md), [offline AC evidence](0055-s115-evidence.md).
[D5 operational receipt](0055-s115-d5-execution-receipt.md): один явно разрешённый
[run39860669842](https://github.com/QahNaaRlN/grace/actions/runs/39860669842) на `703b6c6`
завершён SUCCESS, только diagnostic job; остальные10 jobs skipped.
Closed результат публикуется только в разрешённом CI artifact, private-derived payload в Git не публикуется.
Permission CONSUMED; retry/repair/new paid run не открыты.
[PR №95 post-merge](0055-pr95-postmerge.md): main403f69d clean.
Текущий этап — фиксация SUCCESS диагностики Q6; quality FAIL E6 сохранён, новый run/repair не открыты. Закрытый evidence D5 остаётся вне Git.

Текущий пакет — [S1.14](0055-s114-observed-comparison.md), **accepted / integrated offline**:
P1–P5 интегрированы PR №91 (`68ee7ee`); [post-merge evidence](0055-pr91-postmerge.md).
Offline acceptance «принимаю»; [implementation evidence](0055-s114-evidence.md), implementation CI5/5 PASS.
[P6 execution decision](0055-s114-p6-run-decision.md) отдельно принят прямым «+»
на integrated source301afd6/CI5/5; **EXECUTED / FAIL protocol, authorization consumed**.
[Live evidence](0055-s114-p6-evidence.md):3POST, A/columns16/19, B/columns17/19,
A/cards unknown_alias(operationDate,row1), captured+matched replay; остальные3 not_sent.
5320micros ($0.00532 provider) known/settled, shared uncertain block=false,
image/health preserved; no retry/deploy/model switch. Main CI/Delivery уточнены в [post-merge №94](0055-pr94-postmerge.md); P6 FAIL сохранён.

Завершённый пакет — [S1.13](0055-s113-protocol-observability.md), **accepted / integrated offline**:
PR №89, merge `d63c16f`; [post-merge evidence](0055-pr89-postmerge.md):
local gates, merge CI5/5 и Delivery PASS. Новых paid/metadata/SSH/deploy calls нет.
O1–O5 приняты прямым «одобряю, приступай»; [implementation evidence](0055-s113-evidence.md).

Завершённый пакет — [S1.12](0055-s112-saved-comparison-diagnosis.md), **accepted / integrated**:
read-only диагностика сохранённого C6, без inference. R1–R5 приняты прямым «утверждаю»;
точный guard canonicalization может оказаться невосстановимым из-за отсутствия raw proposal.
[R1–R5 evidence](0055-s112-evidence.md): local/CI5/5 PASS; R5 SUCCESS, providerCalls0.
C6 stage canonicalization; raw proposal missing, exact guard UNVERIFIED;
recorded provider cost1995micros ($0.001995), Complete latency2931ms.

Следующий контракт — [S1.11](0055-s111-model-comparison.md), **accepted / integrated**, 2026-10-21:
предложено сравнение Gemini 3.1 Flash Lite / Google EU и Mercury 3 / Inception
на одинаковых columns/cards/RU с S1.10 guards. Offline C1–C5 приняты прямым «одобряю» и реализованы;
[C1–C5 evidence](0055-s111-evidence.md): PR №84 слит (`012f604`);
[post-merge evidence](0055-pr84-postmerge.md): local/CI5/5/Delivery4/4 PASS;
[C6 decision](0055-s111-c6-run-decision.md) отдельно принят и исполнен: один вызов A/columns,
FAIL `qualification_protocol_failed` до score; остальные5 не отправлены.
[Live evidence](0055-s111-c6-evidence.md): known billing/settled, image/health preserved;
quality/latency UNVERIFIED, authorization consumed, без повтора.
[Свежие metadata](../research/statement-quality/model-shortlist-2026-10-21.md) прочитаны без inference; обе пары unqualified,
прежняя baseline EU-пара отсутствует в публичном snapshot. Deployment/model switch не выполнялись.

Текущий пакет — [S1.10](0055-s110-grounding-guards.md), **accepted / integrated**, 2026-10-21:
общие Go-проверки внутренних пропусков description и ролей opening/closing
book balance, актуальная проверка retained reviews перед confirmation и новая
qualification/scorer version. Inference=0, без prompt/model/deployment изменений.
Владелец принял G1–G5; реализация и offline/SQL/HTTP проверки PASS.
[Подтверждения](0055-s110-evidence.md). PR №82 слит (`2fe0d79`); post-merge local gates, merge CI5/5 и Delivery4/4 PASS.
[Post-merge evidence](0055-pr82-postmerge.md).
PR №81 слит (`5981449`, 04:04:03 +05:00); [post-merge evidence](0055-pr81-postmerge.md):
local gates PASS, source/head/main CI все5/5 PASS, Delivery39513380470 PASS.

[S1.9 — бесплатная диагностика сохранённого S1.8](0055-s19-saved-trace-diagnosis.md) —
**accepted / integrated, diagnosis PASS**, 2026-10-21 03:48:32 +05:00.
[D1–D5 evidence](0055-s19-evidence.md): read-only39510628212 SUCCESS, artifactsUnchanged=true,
replayMatchesSaved=true. Причины FAIL17/19 установлены: пропущен дефис строки2;
два дефиса заголовков выбраны как отсутствующие book balances и отклонены
Go ambiguous_money. Новый inference=0; scorer/prompt/model/flags не менялись.
PR №81 слит; S1.9 integrated, post-merge local/CI5/5/Delivery PASS.
PR №80 интегрирован (`5e9116e`), post-merge local/CI5/5/Delivery PASS.

Последний live результат — **2026-10-21 03:10:03 +05:00**:
[S1.8](0055-s18-evidence.md) выполнен один раз после CI5/5: **FAIL17/19 columns**,
record_description/blockers. Exact canonical прошёл; cards/RU не отправлены.
Известный расход учтён, reservation settled, shared uncertain block=false.
Image/health сохранены; marker consumed, новых попыток и model/flag switch нет.
Offline-разбор выполнен S1.9 (см. выше); исправления prompt/scorer/transport
и новый paid run остаются отдельным scope.

Интеграция №79: `938b370`, 2026-10-21 02:43:30 +05:00;
[post-merge gates](0055-pr79-postmerge.md) завершены: main CI5/5 и Delivery PASS,
локальные make test/go vet/diff check PASS. S1.7/S1.8 integrated (PR №79/№80); исторические live FAIL сохранены.

Исторический срез до S1.8 — **2026-10-21 02:13:09 +05:00**:
расход единственного S1.6 подтверждён и учтён, shared uncertain block снят.
[Сверка и settlement](0055-s16-reconciliation-evidence.md) выполнены
без нового inference; одна reservation settled, остальные строки не изменялись.
[S1.6](0055-s16-evidence.md) остаётся **FAIL selected_model** до
первой оценки, quality UNVERIFIED; cards/RU не отправлены, разрешение израсходовано.
Production policy/model/flags сохранены, деплоя и paid retry нет.
Следующий [S1.7 — точная идентичность модели в screening](0055-s17-selected-model-identity.md)
— **accepted/integrated**, 2026-10-21. Владелец утвердил offline-реализацию;
I1–I4 PASS: точная пара, свежий catalog proof, report v3, product isolation,
реальные PDF со scorer19/20/23, make test/production/race/vet.
[Подтверждения](0055-s17-evidence.md). Внешних inference=0; source в main №79, post-merge gates PASS.
Повторные платные smoke, деплой и переключение модели не разрешены.

Интеграция — **2026-10-21**: PR №78 слит, `29a28a7`; S1.3/S1.4/S1.5 в main.
[Post-merge evidence](0055-pr78-postmerge.md): main CI 5/5 PASS, Delivery PASS,
локальные make test/go vet/diff check PASS; deployment не выполнялся. [S1.6](0055-s16-diagnostic-screening.md) — accepted/local_verified (N1–N4 PASS), live FAIL selected_model;
владелец утвердил реализацию и один запуск, разрешение израсходовано.

Исторический срез, 2026-10-20 17:10:51 UTC: S1.4 screening остановился на
provider stage до первого score; quality UNVERIFIED. Расход подтверждён через
GET generation metadata и учтён; shared uncertain block снят.
Runtime Statement AI flag enabled сохранён, production-модель не менялась.
Ручной импорт доступен. [Reconciliation/evidence](0055-s14-reconciliation.md);
новых inference, deployment или повторов smoke не было.

[SPEC-0055 / S1.4 — candidate screening](0055-s14-candidate-screening.md) — **accepted**,
2026-10-20, прямое «работаем»: реализация и один 3-call screening Gemini 3.1
Flash Lite / Google EU, provider cap $0.1617. Не разрешает production model switch
или enable; implementation integrated (PR №78, `29a28a7`; C1–C5 PASS), live C6 FAIL provider-stage, quality UNVERIFIED;
[evidence](0055-s14-evidence.md).

[Диагностика исходного S1.4 failure](../research/statement-quality/s14-provider-failure-diagnosis.md),
2026-10-20 17:18:36 UTC: read-only saved-metadata audit PASS, runtime/shared-ledger
readiness PASS, 14 synthetic transport counterexamples PASS. Доказан пробел
observability, точный исходный guard UNVERIFIED: chat envelope не сохранён.
[S1.5 screening diagnostics](0055-s15-screening-diagnostics.md) — **accepted**,
2026-10-20, прямое «Утверждаю»;
implementation **integrated** (PR №78, `29a28a7`; D1–D5 PASS); [evidence](0055-s15-evidence.md).
Закрытые diagnostics, private unverified correlation и новый screening report v2;
без нового inference, policy switch или deployment. Current-head CI отдельно.

[SPEC-0055 / S1.3 — роли оборотов и подбор моделей](0055-s13-turnover-roles.md)
— **accepted**, 2026-10-20, прямое поручение «оформи спеку и вперед»;
implementation **integrated** (PR №78, `29a28a7`), [T1–T6 evidence](0055-s13-evidence.md).
Go role guard + RU regression + financial-label
redaction + публичная metadata современных кандидатов. Без paid calls,
production policy switch или deployment; SPEC-0056 registry остаётся draft.

[SPEC-0055 — исправление review](0055-review-repair.md) — **accepted**, 2026-10-20,
implementation **deployed** (`ec15861`, PR #76), R1–R7 PASS; R8 agent browser PASS. Пакет после owner-test: наблюдаемая сверка
итогов, сохранение canonical mapping/coverage и редактирование текущих строк;
legacy service не восстанавливается догадкой. Владелец утвердил конкретный контракт («да, утверждаю»); [release/R8 evidence](0055-review-repair-evidence.md). Release/deploy пройдены; owner financial acceptance остаётся отдельно.

[S1.2 description separators](0055-s12-description-separators.md) — accepted, **deployed_owner_test** (PR #74 / `7b2deea`), 2026-10-20. Один smoke `39396162864`: columns 19/19, cards 20/20; actual image/runtime/current policy/public/auth PASS. **Statement AI включён** 13:51:07 +05:00 / 08:51:07 UTC. [D1–D5 evidence](0055-s12-evidence.md); owner private test и full AC12 остаются открыты. Исторические S1/S1.1 FAIL сохранены.

[S1 owner-test activation amendment](0055-owner-test-activation.md) — accepted, 2026-10-19: владелец утвердил реализацию, один smoke из двух synthetic calls / theoretical $0.14 total и включение только при PASS; без повторов, прежний shared budget. Implementation integrated PR #71; S1.1 PR #73 был выпущен `ef782ec` с AI false; первичная активация S1.2 `7b2deea` enabled; текущий release review repair `ec15861` сохраняет эту policy. Полная qualification остаётся отдельно.

# Spec-first процесс Grace

[SPEC-0056 — автоматический выбор модели для банковской выписки](0056-statement-model-selection.md)
— `draft`, 2026-10-19; направление согласовано владельцем («Хорошо»),
конкретный контракт ожидает review; implementation `not_started`.
Предложен выбор из квалифицированных model/endpoint пар до consent, закрепление
одной policy на attempt и прежний общий бюджет. Это не fallback после отправки
данных и не обход unknown billing SPEC-0055; платные calls/production changes
в документационном пакете не выполняются. База кандидата — PR #66, теперь integrated (`06169e5`); SPEC-0056 остаётся draft.
[Проработка качества](../research/statement-quality/README.md), 2026-10-19:
принята владельцем («Принимаю»), 32 offline probes, план planner/coverage/corpus;
модельных вызовов нет, реализация not_started.
[Accepted S1 contract](0055-record-atomic-planning.md) фиксирует границы
первого пакета: [offline comparison](../research/statement-quality/atoms.md),
280 cases, геометрические кандидаты REJECTED. Whole-document/aliases и новые
caps 90000 bytes / 100000 input / 16000 output приняты владельцем 2026-10-19
(«Утвердить S1 с новыми лимитами»), theoretical provider ceiling 0.07 USD/call;
общий бюджет сохраняется, paid campaigns отдельно. [S1 implementation](0055-s1-evidence.md)
интегрирован PR #70 (`8ed8e80`), включён в текущий production: 84 layout cases, HTTP/SQL/confirmation/cancellation, no external
model calls; опубликовано в [PR #70](https://github.com/QahNaaRlN/grace/pull/70); CI/Delivery пройдены; последующие S1.1/S1.2 изменяют aliases/prompt отдельными accepted amendments. Полная qualification не сводится к smoke.

[SPEC-0055 — банковские PDF разных форматов](0055-bank-statement-pdf-import.md)
— `integrated` (PR #64, `b51b74f`, main CI `39167930999` PASS), технический вариант явно утверждён владельцем 2026-10-19:
локальный текст/страницы → AI-предложение с source spans → Go validation →
сохраняемый review и подтверждаемый импорт. LLM — для структуры, Jev — кандидат
для узких выборов/проверок после измерения. Доступный остаток не подменяет book
balance. [Технический контракт](0055-technical-contract.md), включая новый provider tag,
принят. Первый ручной production release `b51b74f` подтверждён 2026-10-19 02:37:31 +05:00.
Текущий production **`ec15861`** (review repair PR #76): [release/R8 evidence](0055-review-repair-evidence.md),
main CI `39423345963`, Delivery `39424344690`, deploy `39424505774` PASS;
R8 review repair agent browser PASS, owner financial confirmation pending.
Statement AI enabled сохранён, новых provider calls нет. Исторический S1.2 (`7b2deea`, PR #74): main CI `39395041781`, Delivery
`39396018990`, deploy `39396162864` PASS. Columns 19/19 и cards 20/20,
exact-image/runtime/current policy/public/auth PASS; Statement AI enabled
2026-10-20 13:51:07 +05:00 / 08:51:07 UTC. Один S1.2 smoke использован,
не personal inference. Исторические FAIL сохранены; [D1–D5](0055-s12-evidence.md).
[Implementation evidence](0055-evidence.md), [активация и актуальные статусы](0055-activation-evidence.md).
Полная AC12 qualification, Jev comparison и owner functional acceptance остаются
открытыми; private inference не выполнялся.


## Исторические записи roadmap

Актуализация разработки: **2026-10-23**, Asia/Yekaterinburg.

Текущий пакет — [S1.18: диагностика качества сохранённого E6](../specs/0055-s118-saved-quality-diagnosis.md),
**accepted Q1–Q5 / integrated offline, PR №104; Q6 EXECUTED / SUCCESS**.
[Post-merge №104](../specs/0055-pr104-postmerge.md): main438256b clean, head/merge CI5/5
и Delivery4/4 PASS. [Q1–Q5 evidence](../specs/0055-s118-evidence.md): все local gates PASS.
После отдельного «Разрешаю один Q6» на frozen5fdd0f9/ref/CI5/5 опубликован
[receipt](../specs/0055-s118-q6-execution-receipt.md), затем один
[run39949603261](https://github.com/QahNaaRlN/grace/actions/runs/39949603261),
только Q6, остальные12 jobs skipped. [Q6 evidence](../specs/0055-s118-q6-evidence.md):
6/6 tuples,22 failed checks и23 reasons VERIFIED,15 public unknown сопоставлены;
artifactsUnchanged=true, providerCalls0, persistentWritesRequested0.
Публично только safe status/counts, closed witnesses не переносятся в Git/PR/artifacts.
Permission CONSUMED; исторический E6 quality FAIL сохранён. Ремонт, новый paid
run, deployment/model/guard switch и SPEC-0056 этим результатом не открыты.

Завершённый пакет — [S1.17: контролируемое сравнение после ограничения alias](../specs/0055-s117-alias-comparison.md),
**accepted / integrated offline, PR №100/№101; E6 EXECUTED / FAIL quality**.
[YAML blocker](../specs/0055-s117-e6-readiness.md) устранён №101 по прямому разрешению
владельца после CI5/5; [post-merge](../specs/0055-pr101-postmerge.md).
[E6 receipt](../specs/0055-s117-e6-execution-receipt.md) опубликован до одного
[run39925874197](https://github.com/QahNaaRlN/grace/actions/runs/39925874197) на frozen24f7322/CI5/5.
[Результат](../specs/0055-s117-e6-evidence.md):6POST, canonicalization/capture/replay6/6,
качество0/3 у обоих, A17/19–14/20–17/23, B17/19–18/20–19/23.
10897 USD micros ($0.010897 provider) known/settled; shared uncertain=false,
image/health preserved. Permission CONSUMED; повтор/deploy/model switch не открыты.
Read-only диагностика saved E6 выполнена S1.18 Q6 (см. выше); исходный quality FAIL не исправлялся.

Интегрированный offline пакет — [S1.16: допустимые alias во всех selections](../specs/0055-s116-alias-schema.md),
**accepted / integrated offline, PR №98**. Владелец принял A1–A6 прямым «принимаю»
2026-10-23: exact delivered-alias enum,
прежняя numeric eligibility, wire/bytes/consent проверки только offline.
Принятая спека интегрирована PR №97 (`a1cc072`), реализация — PR №98 (`81ef7e5`).
[Post-merge №98](../specs/0055-pr98-postmerge.md): local/head/merge gates PASS; CI head и merge оба5/5. Delivery отдельно.
[Evidence A1–A6](../specs/0055-s116-evidence.md), [post-merge №97](../specs/0055-pr97-postmerge.md).
Scope S1.16 не включал inference/SSH/deploy или новую campaign.
[PR №96 post-merge](../specs/0055-pr96-postmerge.md): receipt интегрирован в main `5b6ed20`.

Текущий пакет — [S1.15](../specs/0055-s115-saved-proposal-diagnosis.md), **accepted D1–D4 / integrated offline**:
[PR №94](https://github.com/QahNaaRlN/grace/pull/94) слит, merge `17e3493`.
Source `703b6c6`: CI 5/5; merge: CI 5/5 и Delivery 4/4 PASS;
[post-merge evidence](../specs/0055-pr94-postmerge.md), [offline AC evidence](../specs/0055-s115-evidence.md).
[D5 operational receipt](../specs/0055-s115-d5-execution-receipt.md): один явно разрешённый
[run39860669842](https://github.com/QahNaaRlN/grace/actions/runs/39860669842) на `703b6c6`
завершён SUCCESS, только diagnostic job; остальные10 jobs skipped.
Closed результат публикуется только в разрешённом CI artifact, private-derived payload в Git не публикуется.
Permission CONSUMED; retry/repair/new paid run не открыты.
[PR №95 post-merge](../specs/0055-pr95-postmerge.md): main403f69d clean.
Текущий этап — фиксация SUCCESS диагностики Q6; quality FAIL E6 сохранён, новый run/repair не открыты. Закрытый evidence D5 остаётся вне Git.

Текущий пакет — [S1.14](../specs/0055-s114-observed-comparison.md), **accepted / integrated offline**:
P1–P5 интегрированы PR №91 (`68ee7ee`); [post-merge evidence](../specs/0055-pr91-postmerge.md).
Offline acceptance «принимаю»; [implementation evidence](../specs/0055-s114-evidence.md), implementation CI5/5 PASS.
[P6 execution decision](../specs/0055-s114-p6-run-decision.md) отдельно принят прямым «+»
на integrated source301afd6/CI5/5; **EXECUTED / FAIL protocol, authorization consumed**.
[Live evidence](../specs/0055-s114-p6-evidence.md):3POST, A/columns16/19, B/columns17/19,
A/cards unknown_alias(operationDate,row1), captured+matched replay; остальные3 not_sent.
5320micros ($0.00532 provider) known/settled, shared uncertain block=false,
image/health preserved; no retry/deploy/model switch. Main CI/Delivery уточнены в [post-merge №94](../specs/0055-pr94-postmerge.md); P6 FAIL сохранён.

Завершённый пакет — [S1.13](../specs/0055-s113-protocol-observability.md), **accepted / integrated offline**:
PR №89, merge `d63c16f`; [post-merge evidence](../specs/0055-pr89-postmerge.md):
local gates, merge CI5/5 и Delivery PASS. Новых paid/metadata/SSH/deploy calls нет.
O1–O5 приняты прямым «одобряю, приступай»; [implementation evidence](../specs/0055-s113-evidence.md).

Завершённый пакет — [S1.12](../specs/0055-s112-saved-comparison-diagnosis.md), **accepted / integrated**:
read-only диагностика сохранённого C6, без inference. R1–R5 приняты прямым «утверждаю»;
точный guard canonicalization может оказаться невосстановимым из-за отсутствия raw proposal.
[R1–R5 evidence](../specs/0055-s112-evidence.md): local/CI5/5 PASS; R5 SUCCESS, providerCalls0.
C6 stage canonicalization; raw proposal missing, exact guard UNVERIFIED;
recorded provider cost1995micros ($0.001995), Complete latency2931ms.

Следующий контракт — [S1.11](../specs/0055-s111-model-comparison.md), **accepted / integrated**, 2026-10-21:
предложено сравнение Gemini 3.1 Flash Lite / Google EU и Mercury 3 / Inception
на одинаковых columns/cards/RU с S1.10 guards. Offline C1–C5 приняты прямым «одобряю» и реализованы;
[C1–C5 evidence](../specs/0055-s111-evidence.md): PR №84 слит (`012f604`);
[post-merge evidence](../specs/0055-pr84-postmerge.md): local/CI5/5/Delivery4/4 PASS;
[C6 decision](../specs/0055-s111-c6-run-decision.md) отдельно принят и исполнен: один вызов A/columns,
FAIL `qualification_protocol_failed` до score; остальные5 не отправлены.
[Live evidence](../specs/0055-s111-c6-evidence.md): known billing/settled, image/health preserved;
quality/latency UNVERIFIED, authorization consumed, без повтора.
[Свежие metadata](../research/statement-quality/model-shortlist-2026-10-21.md) прочитаны без inference; обе пары unqualified,
прежняя baseline EU-пара отсутствует в публичном snapshot. Deployment/model switch не выполнялись.

Текущий пакет — [S1.10](../specs/0055-s110-grounding-guards.md), **accepted / integrated**, 2026-10-21:
общие Go-проверки внутренних пропусков description и ролей opening/closing
book balance, актуальная проверка retained reviews перед confirmation и новая
qualification/scorer version. Inference=0, без prompt/model/deployment изменений.
Владелец принял G1–G5; реализация и offline/SQL/HTTP проверки PASS.
[Подтверждения](../specs/0055-s110-evidence.md). PR №82 слит (`2fe0d79`); post-merge local gates, merge CI5/5 и Delivery4/4 PASS.
[Post-merge evidence](../specs/0055-pr82-postmerge.md).
PR №81 слит (`5981449`, 04:04:03 +05:00); [post-merge evidence](../specs/0055-pr81-postmerge.md):
local gates PASS, source/head/main CI все5/5 PASS, Delivery39513380470 PASS.

[S1.9 — бесплатная диагностика сохранённого S1.8](../specs/0055-s19-saved-trace-diagnosis.md) —
**accepted / integrated, diagnosis PASS**, 2026-10-21 03:48:32 +05:00.
[D1–D5 evidence](../specs/0055-s19-evidence.md): read-only39510628212 SUCCESS, artifactsUnchanged=true,
replayMatchesSaved=true. Причины FAIL17/19 установлены: пропущен дефис строки2;
два дефиса заголовков выбраны как отсутствующие book balances и отклонены
Go ambiguous_money. Новый inference=0; scorer/prompt/model/flags не менялись.
PR №81 слит; S1.9 integrated, post-merge local/CI5/5/Delivery PASS.
PR №80 интегрирован (`5e9116e`), post-merge local/CI5/5/Delivery PASS.

Последний live результат — **2026-10-21 03:10:03 +05:00**:
[S1.8](../specs/0055-s18-evidence.md) выполнен один раз после CI5/5: **FAIL17/19 columns**,
record_description/blockers. Exact canonical прошёл; cards/RU не отправлены.
Известный расход учтён, reservation settled, shared uncertain block=false.
Image/health сохранены; marker consumed, новых попыток и model/flag switch нет.
Offline-разбор выполнен S1.9 (см. выше); исправления prompt/scorer/transport
и новый paid run остаются отдельным scope.

Интеграция №79: `938b370`, 2026-10-21 02:43:30 +05:00;
[post-merge gates](../specs/0055-pr79-postmerge.md) завершены: main CI5/5 и Delivery PASS,
локальные make test/go vet/diff check PASS. S1.7 integrated; PR №80 слит в main (`5e9116e`) 2026-10-21 03:24:55 +05:00.
[Post-merge проверки №80](../specs/0055-pr80-postmerge.md): local make test/vet/diff PASS;
merge CI5/5 PASS, S1.8 integrated; Delivery39508797773 PASS; production deployment не выполнялся.

Исторический срез до S1.8 — **2026-10-21 02:13:09 +05:00**:
расход единственного S1.6 подтверждён и учтён, shared uncertain block снят.
[Сверка и settlement](../specs/0055-s16-reconciliation-evidence.md) выполнены
без нового inference; одна reservation settled, остальные строки не изменялись.
[S1.6](../specs/0055-s16-evidence.md) остаётся **FAIL selected_model** до
первой оценки, quality UNVERIFIED; cards/RU не отправлены, разрешение израсходовано.
Production policy/model/flags сохранены, деплоя и paid retry нет.
Следующий [S1.7 — точная идентичность модели в screening](../specs/0055-s17-selected-model-identity.md)
— **accepted/integrated**, 2026-10-21. Владелец утвердил offline-реализацию;
I1–I4 PASS: точная пара, свежий catalog proof, report v3, product isolation,
реальные PDF со scorer19/20/23, make test/production/race/vet.
[Подтверждения](../specs/0055-s17-evidence.md). Внешних inference=0; source в main №79, post-merge gates PASS.
Повторные платные smoke, деплой и переключение модели не разрешены.

Интеграция — **2026-10-21**: [PR №78](https://github.com/QahNaaRlN/grace/pull/78)
слит в main (`29a28a7`); S1.3/S1.4/S1.5 интегрированы, не deployed.
[Проверки после слияния](../specs/0055-pr78-postmerge.md): main CI 5/5 PASS,
локальные make test/go vet/diff check PASS; Delivery `39490817275` PASS.
Образы опубликованы, деплой приложения не выполнялся.
Следующий [S1.6](../specs/0055-s16-diagnostic-screening.md) — accepted/local_verified, live FAIL selected_model:
один новый smoke с диагностикой, не более 3 вызовов / $0.1617 provider cap,
без переключения модели; владелец утвердил реализацию и этот единственный запуск; он выполнен, повтор не разрешён.

Состояние до S1.6 — 2026-10-20 17:10:51 UTC / 22:10:51 +05:00:
[единственный S1.4 screening](../specs/0055-s14-evidence.md) завершился provider-stage
FAIL до первого score; quality UNVERIFIED. Предоставленный владельцем generation
ID сверён через API, подтверждённый расход учтён, shared uncertain block снят.
[Reconciliation](../specs/0055-s14-reconciliation.md) завершён без нового inference.
Statement AI runtime flag enabled и production-модель сохранены; deployment
не выполнялся. Ручной импорт работает; автоматического paid retry нет.

