Следующий кандидат — [S1.15](0055-s115-saved-proposal-diagnosis.md), **draft / not_started**:
закрытая read-only диагностика saved P6 proposals: alias relation, description-gap
и unmapped witnesses. D1–D4 offline ожидают принятия; D5 — отдельного exact-source
read-only approval после реализации/CI. Новых model/metadata/SQL/SSH calls нет.

Текущий пакет — [S1.14](0055-s114-observed-comparison.md), **accepted / integrated offline**:
P1–P5 интегрированы PR №91 (`68ee7ee`); [post-merge evidence](0055-pr91-postmerge.md).
Offline acceptance «принимаю»; [implementation evidence](0055-s114-evidence.md), implementation CI5/5 PASS.
[P6 execution decision](0055-s114-p6-run-decision.md) отдельно принят прямым «+»
на integrated source301afd6/CI5/5; **EXECUTED / FAIL protocol, authorization consumed**.
[Live evidence](0055-s114-p6-evidence.md):3POST, A/columns16/19, B/columns17/19,
A/cards unknown_alias(operationDate,row1), captured+matched replay; остальные3 not_sent.
5320micros ($0.00532 provider) known/settled, shared uncertain block=false,
image/health preserved; no retry/deploy/model switch. Main push-CI/Delivery отдельно UNVERIFIED.

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

Принята прямым поручением владельца 2026-10-17 «Приступай к разработке», статус `integrated` (PR #60, merge `a89a855`; локальные post-merge gates пройдены 2026-10-18; main CI run 39056633511 attempt 2 и Delivery run 39151285103 пройдены; production release `a89a855` подтверждён Deploy production run 39151600396, 2026-10-18; [evidence](0054-evidence.md)):
[SPEC-0054 — рассрочки и BNPL](0054-installments-bnpl.md)
— связь покупки, долга и погашений; рассрочки внутри карты и банковское
требование для грейса. Принят предложенный v1 без процентов, капитализации
комиссий и возвратов; банковское требование подтверждается вручную.

Принята владельцем 2026-10-17 (вид кредита списком, разбивка в статистике, деление «строка + досрочно», ручная привязка), статус `integrated` (PR #57, merge `95c5fa8`; CI `main` run 39032209158 пройден; в production с выпуском `95c5fa8` (Deploy production run 39034114035, 2026-10-17); [evidence](0053-evidence.md)):
[SPEC-0053 — кредиты с графиком: тело, проценты и снимки долга](0053-loan-principal-interest.md)
— вид кредита и расчёт частей строк, распределение платежей, досрочное погашение,
остаток основного долга и «Стоимость кредитов» в статистике.

Принята владельцем 2026-10-16 (`lazy` маршрутов, одна перезагрузка при ошибке чанка, все страницы), статус `integrated` (локальные post-merge gates на `a89a855` пройдены 2026-10-18; PR #56; в production с выпуском `95c5fa8` (Deploy production run 39034114035, 2026-10-17); [evidence](0052-evidence.md)):
[SPEC-0052 — разделение кода Web по маршрутам](0052-route-code-splitting.md)
— страницы `pages/*` загружаются по маршруту, основной чанк меньше 500 кБ.

Принята владельцем 2026-10-16 (сначала кредитные карты; ручной ввод условий, напоминания, общая таблица документов), статус `integrated` (локальные post-merge gates на `a89a855` пройдены 2026-10-18; PR #55; в production с выпуском `95c5fa8` (Deploy production run 39034114035, 2026-10-17); [evidence](0051-evidence.md)):
[SPEC-0051 — кредитные карты: условия, долг и льготный период](0051-credit-cards.md).

Принята владельцем 2026-10-16 (Yandex Object Storage, ключ только на загрузку, письмо через Postbox, хранение 35/180 дней), статус `integrated` (локальные post-merge gates на `a89a855` пройдены 2026-10-18; PR #52–#54; production-проверки AC1–AC7 пройдены на `fc19d32`; [evidence](0050-evidence.md)):
[SPEC-0050 — зашифрованная копия backup вне VPS](0050-offsite-backup.md).

Принята владельцем 2026-10-16 (без инвалидации обязательств), статус `integrated` (PR #50, merge `cb4a432`; CI `main` run 38811547370 пройден; в production с выпуском `822b8f8`; [evidence](0049-evidence.md)):
[SPEC-0049 — обновление данных после подтверждения чека](0049-receipt-confirmation-refresh.md).

Принята владельцем 2026-10-15 (Steiger, Prettier без `;` и с одинарными кавычками), статус `integrated` (фазы 0–5, PR #42–#48; post-merge проверки на `d486582` пройдены):
[SPEC-0048 — архитектура Web по Feature-Sliced Design](0048-frontend-fsd-architecture.md)
— аудит слоёв, правила срезов, серверное состояние, автоматические проверки и
поэтапная миграция без изменения поведения.

Принята владельцем 2026-10-14 (валюта списком, категория подписки, вёрстка графика), статус `integrated` (PR #41, merge `e623775`; CI `main` run 38573760071 пройден). Production развёрнут 2026-10-14 (release `e623775`, workflow run 38575191780):
[SPEC-0047 — валюта списком, категория подписки, вёрстка графика](0047-subscription-form-and-schedule-layout.md).

Принята владельцем 2026-10-14 (порог «за 3 дня и за 1 день», issue #23), статус `integrated` (PR #36):
[SPEC-0046 — статус «скоро платёж»](0046-payment-due-soon.md).

Приняты владельцем 2026-10-13 (вторая часть ревью), статус `integrated` (PR #29–#34;
post-merge проверки на `1339eea` пройдены). Production развёрнут 2026-10-14 (release `8ad9705`, workflow run 38560119989):
[SPEC-0040 — счета, карты и сверка остатка](0040-account-reconciliation.md),
[SPEC-0041 — исправление и отмена операций](0041-operation-corrections.md),
[SPEC-0042 — категории: изменение, удаление, иконки](0042-category-management.md),
[SPEC-0043 — список операций и иконка профиля в сайдбаре](0043-operations-list.md),
[SPEC-0044 — история импортов и проверка сотен строк](0044-import-review-at-scale.md),
[SPEC-0045 — график по числу месяца и сроку](0045-schedule-by-period.md).

SPEC-0035–0038 приняты владельцем 2026-10-13, статус `integrated` (PR #22,
merge `ed641ac`; post-merge проверки пройдены). Production развёрнут 2026-10-14
в составе release `8ad9705`. Продолжения #23 и #24 закрыты SPEC-0046 и SPEC-0039:
[SPEC-0035 — ревизия дизайн-системы](0035-design-system-revision.md)
(сравнение — тема «proposed» в артефакте Grace Design System),
[SPEC-0036 — ввод сумм с разделителями и обратная связь профиля](0036-amount-input-profile-feedback.md),
[SPEC-0037 — ввод длинных графиков платежей](0037-long-schedule-entry.md),
[SPEC-0038 — обратная связь при выборе и сохранении файла импорта](0038-import-file-feedback.md)
(GitHub #14; #15 устранён SPEC-0032 и закрыт).

[SPEC-0039 — компактный график обязательств на «Платежах»](0039-compact-payment-schedules.md)
— принята владельцем 2026-10-14 (issue #24), статус `integrated` (PR #37), в production: просрочка и ближайшие платежи
сверху, оплаченная история и дальние платежи свёрнуты «Показать все N».

SPEC-0026 и SPEC-0029–0033 интегрированы объединённым выпуском `d475cca`
(CI run 38038836282, Deploy production run 38039374895, 2026-10-10), SPEC-0034 —
PR #21, `75c8fbc` (CI run 38045618255, Deploy production run 38045963849,
2026-10-10); статус всех семи — `integrated`. Абзацы ниже сохраняют сведения
о локальных контрольных точках до этих выпусков. Включение OpenRouter в
production остаётся отдельным gate SPEC-0026/0034.

[SPEC-0034 — подписки, кредиты и напоминания](0034-commitments-ai-reminders.md)
принята прямым поручением владельца 2026-10-10; копия графика — черновик внутри
Grace. Включение OpenRouter ограничено прежним бюджетом и отдельным согласием
на передачу личных данных.

[SPEC-0033 — понятные счета, категории, здоровье и ИИ](0033-ux-refinement.md)
принята прямым поручением владельца 2026-10-10. Работа объединяет локальные
SPEC-0026/0031/0032. Деплой только после пользовательской проверки.
[Гайд ИИ](../ai-user-guide.md), [отчёт и проверки](0033-evidence.md).

[SPEC-0032 — единый полный интерфейс](0032-ui-redesign.md) принят прямым
поручением владельца; все действующие экраны переработаны локально.
[Система компонентов](../design-system.md), [evidence](0032-evidence.md).

[SPEC-0031 — почта, пароль и приглашения](0031-password-registration.md) —
accepted прямым поручением и выбором password-only login / email-bound invites.
Реализация локально; [evidence](0031-evidence.md), [rollout](../password-authentication-rollout.md).

[SPEC-0030 — графики статистики](0030-statistics-charts.md) — accepted прямым
поручением владельца после обсуждения трёх графиков. Реализация в отдельной
ветке codex/statistics-charts, проверки: [evidence](0030-evidence.md).

[SPEC-0029 — проработка палитры и единое оформление](0029-design-foundations.md)
— accepted по поручению владельца 2026-10-09: три palette previews,
прозрачная favicon и согласованные общие controls. Выбрана петрольная палитра
и знак ₽ с пульсом; применены локально, production rollout ещё не выполнен.
[Проработка](../design-direction.md), [evidence](0029-evidence.md).

Выпуск 2026-10-09: SPEC-0017/0023/0024/0027 и
[SPEC-0028 — история с несверенными остатками](0028-unreconciled-history-accounts.md)
интегрированы и работают в production f257f8e. Подтверждённый перенос истории
завершён; исходники, связи дублей и исключения сохранены, неизвестные остатки
не заменены нулями. [Результаты проверки](0028-evidence.md).
SPEC-0026 остаётся отдельным принятым направлением; этот выпуск не включает
эксперимент с LLM и распознаванием фото чеков. Ниже сохранены сведения о
предыдущих контрольных точках; их этапы интеграции заменены этим выпуском.

[SPEC-0026 — встроенный ассистент и фото чеков](0026-ai-assistant-receipts.md)
— accepted: архитектура ядра/Jev/LLM, серверный OpenRouter adapter, бюджет $10,
временные фото и отдельное подтверждение AI-originated expense. Реализация
ведётся в `codex/ai-assistant-receipts`; live experiment и rollout ещё не выполнены.
Scope и политика записи приняты владельцем 2026-10-08. Проработка:
[архитектура взаимодействия](../ai-interaction-architecture.md).

[SPEC-0027 — исторические операции с неизвестным временем](0027-date-only-history.md)
принята владельцем 2026-10-08. Реализация `597cfcc` проверена (`verified`),
включая полный набор интеграционных проверок, и входит в выпуск истории вместе
с SPEC-0017/0023/0024. [Evidence](0027-evidence.md). Owner отложил сверку
счетов/остатков/дублей; реальная книга переносится только в сохраняемый черновик.

Реестр реализованных этапов и оставшихся направлений: [roadmap](../roadmap.md).

[SPEC-0024 — favicon и плавность интерфейса](0024-interface-motion.md)
принята прямым поручением владельца 2026-10-07 и проверена локально в ветке
`codex/interface-motion`. Критерии и результаты: [evidence](0024-evidence.md).
В main и production визуальные изменения ещё не включены.
На 2026-10-07 SPEC-0001–0016 интегрированы; основной production-вход по почте
и перенос прежнего профиля владельца подтверждены по SPEC-0022. Telegram временно отключён.

SPEC-0016 реализует офлайн-нормализацию таблицы и отчёт сверки, без записи в
ledger. Следующий этап roadmap — сохраняемые черновики, происхождение данных
и подтверждаемый импорт; он требует отдельной принятой спецификации.
Scope [SPEC-0017 — сохраняемые черновики и подтверждение
импорта](0017-persistent-import-drafts.md) принят 2026-10-06; реализация a1ea19d
проверена локально (`verified`) в `codex/persistent-import-drafts`.
Открыт [draft PR #7](https://github.com/QahNaaRlN/grace/pull/7);
интеграция в main и post-merge CI ещё не выполнены.
Результаты: [evidence](0017-evidence.md), [инструкция](../imports.md).

[SPEC-0023 — перенос месяца расходов](0023-monthly-expense-migration.md)
принята прямым поручением владельца 2026-10-07. Ветка
`codex/monthly-expense-migration` совмещает основу SPEC-0017 с текущим main,
использует миграцию 000009 и добавляет CSV, полный месячный отчёт и проверку
дублей основного блока. Проверки и ограничения: [evidence](0023-evidence.md).
Интеграция в main и production rollout — отдельные последующие действия.
Банковские PDF, фото чеков, кредитная модель и LLM API ещё не реализованы.

[SPEC-0018](0018-public-telegram-entrypoint.md) интегрирована и развёрнута:
старый Basic Auth перед Telegram-входом удалён, выпуск f9875f5 проверен.
Результаты: [evidence](0018-evidence.md).

[SPEC-0019](0019-google-apple-mobile-login.md) — draft проработки входа через
Google и Apple с сохранением приглашений, явной привязкой к существующему
аккаунту и мобильным возвратом. Реализация не начата; отдельно проверяются
условия Apple для Web-only Grace. [Исследование](../authentication-options.md).

Раздел «Текущее поведение» внутри конкретной спецификации описывает состояние
на момент её подготовки, до реализации. Он сохраняется как исторический
контекст. Итоговый статус следует читать вместе с соответствующим
`NNNN-evidence.md`; ранние записи в отчёте могут предшествовать его
заключительной post-merge проверке.

## Статусы

Каждая спецификация содержит один из статусов:

- `draft` — обсуждается; production-код по ней запрещён;
- `accepted` — scope, контракты и критерии приёмки явно приняты владельцем;
- `implementing` — реализация ведётся в выделенном worktree;
- `verified` — implementation commit существует, все обязательные проверки
  выполнены, но изменение ещё не интегрировано;
- `integrated` — implementation commit достижим из `main`, выполнены
  post-merge проверки.

Статусы не взаимозаменяемы. В частности, `accepted` не означает
`implemented`, а `verified` не означает `integrated`.

## Обязательные разделы спецификации

Спецификация должна определять:

1. проблему и пользовательский результат;
2. scope и non-goals;
3. наблюдаемое текущее поведение;
4. предметную модель и инварианты;
5. изменения API/OpenAPI;
6. изменения данных, миграцию и rollback;
7. ошибки и негативные сценарии;
8. критерии приёмки;
9. исполняемые проверки;
10. открытые решения.

## Жизненный цикл

1. Создать worktree для спецификации:

       git worktree add -b codex/<spec-slug> \
         ../grace-worktrees/<spec-slug> main

2. Добавить draft по [template.md](template.md). Код продукта не менять.
3. Получить review владельца. Неясные решения остаются явно открытыми.
4. После явного принятия записать дату и основание, изменить статус на
   `accepted` и интегрировать спецификацию в `main`.
5. Создать новый implementation worktree от актуального `main`.
6. Реализовать только принятый scope. Изменение scope сначала возвращает
   спецификацию на review.
7. Выполнить все критерии приёмки и общие gates.
8. После интеграции повторить gates в основной копии.

## Правила worktree

- Корень: `/home/krmiftakhov/PersonalProjects/grace-worktrees`.
- Один work item — один каталог и одна ветка.
- Перед созданием проверить чистоту основной копии, отсутствие ветки и пути.
- Во время review и исправлений переиспользовать тот же worktree.
- Перед удалением выполнить `git status --porcelain`.
- Чистый завершённый worktree удалять только командой:

      git worktree remove <exact-path>
      git worktree prune
      git worktree list

- Dirty, неизвестный или ещё нужный worktree сохранять с указанием причины и
  следующего условия очистки.
- Не использовать `--force`, `rm -rf` или широкую очистку
  `grace-worktrees`.

## Общие gates

Минимальный набор:

    make test
    git diff --check

Спецификация добавляет предметные и интеграционные проверки, необходимые для
своего риска.

## SPEC-0020 — реализация принятого макета

[Спецификация](0020-home-redesign.md) принята владельцем 2026-10-06 вместе
с палитрой индиго; публикация и развёртывание разрешены. Реализация интегрирована и развёрнута на graceapp.ru (release 7b108df). Проверки:
[0020-evidence.md](0020-evidence.md). Данные карты хранятся как отдельный
платёжный инструмент счёта; структура таблицы сохранена, неподдерживаемые
сведения обозначены неизвестными. Полная кредитная модель остаётся отдельным
этапом. Remote CI и deployment имеют самостоятельные gates.

[SPEC-0021](0021-explicit-salary-data.md) интегрирована и развёрнута: общий доход не классифицируется как зарплата. Окончательный smoke-release интерфейса: 11f6b2e; [проверки](0021-evidence.md).

[SPEC-0022](0022-email-authentication-profile.md) — integrated: основной вход по
почтовому коду, персональный профиль и сохранение существующего владельца.
Интегрирована в PR #10 и развёрнута в release 5405363. Postbox/DNS настроены,
вход владельца подтверждён, финансовые данные сохранены. Telegram временно
отключён. [Evidence](0022-evidence.md), [порядок подключения](../email-authentication-rollout.md).
