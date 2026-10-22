Актуализация разработки: **2026-10-23**, Asia/Yekaterinburg.

Следующий кандидат — [S1.16: допустимые alias во всех selections](specs/0055-s116-alias-schema.md),
**draft / implementation not_started**. Предложены exact delivered-alias enum,
прежняя numeric eligibility, wire/bytes/consent проверки только offline.
Владелец ещё не принял контракт; inference/SSH/deploy и новая campaign не разрешены.
[PR №96 post-merge](specs/0055-pr96-postmerge.md): receipt интегрирован в main `5b6ed20`.

Текущий пакет — [S1.15](specs/0055-s115-saved-proposal-diagnosis.md), **accepted D1–D4 / integrated offline**:
[PR №94](https://github.com/QahNaaRlN/grace/pull/94) слит, merge `17e3493`.
Source `703b6c6`: CI 5/5; merge: CI 5/5 и Delivery 4/4 PASS;
[post-merge evidence](specs/0055-pr94-postmerge.md), [offline AC evidence](specs/0055-s115-evidence.md).
[D5 operational receipt](specs/0055-s115-d5-execution-receipt.md): один явно разрешённый
[run39860669842](https://github.com/QahNaaRlN/grace/actions/runs/39860669842) на `703b6c6`
завершён SUCCESS, только diagnostic job; остальные10 jobs skipped.
Closed результат публикуется только в разрешённом CI artifact, private-derived payload в Git не публикуется.
Permission CONSUMED; retry/repair/new paid run не открыты.
[PR №95 post-merge](specs/0055-pr95-postmerge.md): main403f69d clean.
Следующий этап — review draft S1.16; новая реализация не начата. Закрытый evidence D5 остаётся вне Git.

Текущий пакет — [S1.14](specs/0055-s114-observed-comparison.md), **accepted / integrated offline**:
P1–P5 интегрированы PR №91 (`68ee7ee`); [post-merge evidence](specs/0055-pr91-postmerge.md).
Offline acceptance «принимаю»; [implementation evidence](specs/0055-s114-evidence.md), implementation CI5/5 PASS.
[P6 execution decision](specs/0055-s114-p6-run-decision.md) отдельно принят прямым «+»
на integrated source301afd6/CI5/5; **EXECUTED / FAIL protocol, authorization consumed**.
[Live evidence](specs/0055-s114-p6-evidence.md):3POST, A/columns16/19, B/columns17/19,
A/cards unknown_alias(operationDate,row1), captured+matched replay; остальные3 not_sent.
5320micros ($0.00532 provider) known/settled, shared uncertain block=false,
image/health preserved; no retry/deploy/model switch. Main CI/Delivery уточнены в [post-merge №94](specs/0055-pr94-postmerge.md); P6 FAIL сохранён.

Завершённый пакет — [S1.13](specs/0055-s113-protocol-observability.md), **accepted / integrated offline**:
PR №89, merge `d63c16f`; [post-merge evidence](specs/0055-pr89-postmerge.md):
local gates, merge CI5/5 и Delivery PASS. Новых paid/metadata/SSH/deploy calls нет.
O1–O5 приняты прямым «одобряю, приступай»; [implementation evidence](specs/0055-s113-evidence.md).

Завершённый пакет — [S1.12](specs/0055-s112-saved-comparison-diagnosis.md), **accepted / integrated**:
read-only диагностика сохранённого C6, без inference. R1–R5 приняты прямым «утверждаю»;
точный guard canonicalization может оказаться невосстановимым из-за отсутствия raw proposal.
[R1–R5 evidence](specs/0055-s112-evidence.md): local/CI5/5 PASS; R5 SUCCESS, providerCalls0.
C6 stage canonicalization; raw proposal missing, exact guard UNVERIFIED;
recorded provider cost1995micros ($0.001995), Complete latency2931ms.

Следующий контракт — [S1.11](specs/0055-s111-model-comparison.md), **accepted / integrated**, 2026-10-21:
предложено сравнение Gemini 3.1 Flash Lite / Google EU и Mercury 3 / Inception
на одинаковых columns/cards/RU с S1.10 guards. Offline C1–C5 приняты прямым «одобряю» и реализованы;
[C1–C5 evidence](specs/0055-s111-evidence.md): PR №84 слит (`012f604`);
[post-merge evidence](specs/0055-pr84-postmerge.md): local/CI5/5/Delivery4/4 PASS;
[C6 decision](specs/0055-s111-c6-run-decision.md) отдельно принят и исполнен: один вызов A/columns,
FAIL `qualification_protocol_failed` до score; остальные5 не отправлены.
[Live evidence](specs/0055-s111-c6-evidence.md): known billing/settled, image/health preserved;
quality/latency UNVERIFIED, authorization consumed, без повтора.
[Свежие metadata](research/statement-quality/model-shortlist-2026-10-21.md) прочитаны без inference; обе пары unqualified,
прежняя baseline EU-пара отсутствует в публичном snapshot. Deployment/model switch не выполнялись.

Текущий пакет — [S1.10](specs/0055-s110-grounding-guards.md), **accepted / integrated**, 2026-10-21:
общие Go-проверки внутренних пропусков description и ролей opening/closing
book balance, актуальная проверка retained reviews перед confirmation и новая
qualification/scorer version. Inference=0, без prompt/model/deployment изменений.
Владелец принял G1–G5; реализация и offline/SQL/HTTP проверки PASS.
[Подтверждения](specs/0055-s110-evidence.md). PR №82 слит (`2fe0d79`); post-merge local gates, merge CI5/5 и Delivery4/4 PASS.
[Post-merge evidence](specs/0055-pr82-postmerge.md).
PR №81 слит (`5981449`, 04:04:03 +05:00); [post-merge evidence](specs/0055-pr81-postmerge.md):
local gates PASS, source/head/main CI все5/5 PASS, Delivery39513380470 PASS.

[S1.9 — бесплатная диагностика сохранённого S1.8](specs/0055-s19-saved-trace-diagnosis.md) —
**accepted / integrated, diagnosis PASS**, 2026-10-21 03:48:32 +05:00.
[D1–D5 evidence](specs/0055-s19-evidence.md): read-only39510628212 SUCCESS, artifactsUnchanged=true,
replayMatchesSaved=true. Причины FAIL17/19 установлены: пропущен дефис строки2;
два дефиса заголовков выбраны как отсутствующие book balances и отклонены
Go ambiguous_money. Новый inference=0; scorer/prompt/model/flags не менялись.
PR №81 слит; S1.9 integrated, post-merge local/CI5/5/Delivery PASS.
PR №80 интегрирован (`5e9116e`), post-merge local/CI5/5/Delivery PASS.

Последний live результат — **2026-10-21 03:10:03 +05:00**:
[S1.8](specs/0055-s18-evidence.md) выполнен один раз после CI5/5: **FAIL17/19 columns**,
record_description/blockers. Exact canonical прошёл; cards/RU не отправлены.
Известный расход учтён, reservation settled, shared uncertain block=false.
Image/health сохранены; marker consumed, новых попыток и model/flag switch нет.
Offline-разбор выполнен S1.9 (см. выше); исправления prompt/scorer/transport
и новый paid run остаются отдельным scope.

Интеграция №79: `938b370`, 2026-10-21 02:43:30 +05:00;
[post-merge gates](specs/0055-pr79-postmerge.md) завершены: main CI5/5 и Delivery PASS,
локальные make test/go vet/diff check PASS. S1.7 integrated; PR №80 слит в main (`5e9116e`) 2026-10-21 03:24:55 +05:00.
[Post-merge проверки №80](specs/0055-pr80-postmerge.md): local make test/vet/diff PASS;
merge CI5/5 PASS, S1.8 integrated; Delivery39508797773 PASS; production deployment не выполнялся.

Исторический срез до S1.8 — **2026-10-21 02:13:09 +05:00**:
расход единственного S1.6 подтверждён и учтён, shared uncertain block снят.
[Сверка и settlement](specs/0055-s16-reconciliation-evidence.md) выполнены
без нового inference; одна reservation settled, остальные строки не изменялись.
[S1.6](specs/0055-s16-evidence.md) остаётся **FAIL selected_model** до
первой оценки, quality UNVERIFIED; cards/RU не отправлены, разрешение израсходовано.
Production policy/model/flags сохранены, деплоя и paid retry нет.
Следующий [S1.7 — точная идентичность модели в screening](specs/0055-s17-selected-model-identity.md)
— **accepted/integrated**, 2026-10-21. Владелец утвердил offline-реализацию;
I1–I4 PASS: точная пара, свежий catalog proof, report v3, product isolation,
реальные PDF со scorer19/20/23, make test/production/race/vet.
[Подтверждения](specs/0055-s17-evidence.md). Внешних inference=0; source в main №79, post-merge gates PASS.
Повторные платные smoke, деплой и переключение модели не разрешены.

Интеграция — **2026-10-21**: [PR №78](https://github.com/QahNaaRlN/grace/pull/78)
слит в main (`29a28a7`); S1.3/S1.4/S1.5 интегрированы, не deployed.
[Проверки после слияния](specs/0055-pr78-postmerge.md): main CI 5/5 PASS,
локальные make test/go vet/diff check PASS; Delivery `39490817275` PASS.
Образы опубликованы, деплой приложения не выполнялся.
Следующий [S1.6](specs/0055-s16-diagnostic-screening.md) — accepted/local_verified, live FAIL selected_model:
один новый smoke с диагностикой, не более 3 вызовов / $0.1617 provider cap,
без переключения модели; владелец утвердил реализацию и этот единственный запуск; он выполнен, повтор не разрешён.

Состояние до S1.6 — 2026-10-20 17:10:51 UTC / 22:10:51 +05:00:
[единственный S1.4 screening](specs/0055-s14-evidence.md) завершился provider-stage
FAIL до первого score; quality UNVERIFIED. Предоставленный владельцем generation
ID сверён через API, подтверждённый расход учтён, shared uncertain block снят.
[Reconciliation](specs/0055-s14-reconciliation.md) завершён без нового inference.
Statement AI runtime flag enabled и production-модель сохранены; deployment
не выполнялся. Ручной импорт работает; автоматического paid retry нет.

# Roadmap Grace

Актуализация: **2026-10-21** (Asia/Yekaterinburg; UTC отдельно в evidence).
**Production `ec15861`: review repair PR #76 deployed** в 17:44:58 +05:00 /
12:44:58 UTC. Main CI `39423345963`, Delivery `39424344690`,
[deploy `39424505774`](https://github.com/QahNaaRlN/grace/actions/runs/39424505774) PASS:
backup/migration29/helper/API health/public/auth. **Statement AI включён**,
state preserve и readiness подтверждены read-only `39425981524` 17:54:49 +05:00.
Новых paid calls/smoke/private inference не было.

[Review repair R1–R8](specs/0055-review-repair-evidence.md) — local/CI/agent browser PASS:
сохранение разметки, восстановление legacy drafts, наблюдаемые header totals/source
coverage и редактирование текущих строк. Реальный retained header FAIL объяснён
выбором банковских реквизитов вместо оборотов; через UI выполнена ручная коррекция
по документу. Row evidence/normalization сохранены, turnover matched, unmapped=0.
**Owner financial acceptance pending**: роли/страницы/семантика и duplicates требуют
явных решений; financial confirmation не выполнялся. Browser PASS не означает
улучшения автоматического извлечения или завершения full AC12/corpus/Jev.

Исторический S1.2 `7b2deea`: columns19/19 и cards20/20, один separately approved
smoke; [D1–D5](specs/0055-s12-evidence.md). Прежние model FAIL сохранены в
[истории](specs/0055-activation-evidence.md). SPEC-0056 model registry остаётся draft;
SPEC-0019 draft, номер0025 не использован. Унаследованные ShellCheck замечания
сохранены в evidence SPEC-0050.

Документ разделяет реализованные этапы и целевое развитие из [видения](vision.md).
Требования к целевому продукту уточнены по анализу действующей финансовой
таблицы и обсуждению с владельцем 2026-10-06; они не означают, что будущие
этапы реализованы или их спецификации приняты.
Архитектурные правила определяет [контракт](architecture.md).

## Текущая работа — 2026-10-20

Текущая принятая спецификация — [SPEC-0055](specs/0055-bank-statement-pdf-import.md).
Реализация интегрирована PR #64; protected activation/rollback — PR #65.
Ручной путь и Statement AI доступны в production `ec15861`. S1.2 synthetic
qualification сохранена; review repair PR #76 прошёл CI/release и R8 agent browser
проверку. Неизвестный исторический расход сверён 2026-10-19. Следующий owner gate —
финансовые решения/роли/страницы и duplicates; full corpus отдельно.
SPEC-0054 ранее выпущена как `a89a855` (run `39151600396`, 2026-10-18).

| Направление | Сделано | Осталось / зависимость |
| --- | --- | --- |
| Выпуск `main` | `ec15861`: review repair PR #76, CI/Delivery/deploy/runtime/public + R8 PASS 2026-10-20 | Owner financial confirmation pending; source review исправлен; docs follow-up не меняет код выпуска |
| ИИ (OpenRouter) | Statement AI enabled; columns19/19, cards20/20; pinned policy/global kill switch/consent/shared budget сохранены | Full AC12/Jev/cascade/corpus и SPEC-0056 registry отдельно |
| Напоминания | Durable outbox за 3 и 1 день, verified email, общий лимит | Подтвердить production SMTP и сверить reservations с облачным счётом |
| Вход | Почтовый код в production (SPEC-0022); пароль и email-bound приглашения реализованы (SPEC-0031) | Переключение production на password mode по [runbook](password-authentication-rollout.md) в evidence не записано |
| Кредиты с графиком | [SPEC-0053](specs/0053-loan-principal-interest.md) принята 2026-10-17: тело/проценты, вид кредита, досрочное погашение, снимки долга | Интегрирована: PR #57, merge `95c5fa8`, CI run 39032209158, в production с `95c5fa8` ([evidence](specs/0053-evidence.md)) |
| Рассрочки и BNPL | SPEC-0054 влита PR #60 (`a89a855`): покупка/долг/переводы и ручное подтверждение карточного грейса | Post-merge gates, main CI, Delivery и production release `a89a855` пройдены; остаётся owner UI smoke |
| CI и статусы | PR CI `39422108502`, main CI `39423345963`, Delivery `39424344690`, deploy `39424505774`, runtime `39425981524` PASS | Owner functional acceptance pending; не приравнивать synthetic PASS к качеству всех банков |
| Эксплуатация | Backup/restore на VPS, проверки rollback миграций; зашифрованные копии в Yandex Object Storage с проверенным восстановлением (SPEC-0050) | Повторять restore drill после крупных изменений схемы |

## Реализованные этапы

Код перечисленных спецификаций достижим из `main`. SPEC-0001–0049 из таблицы
и SPEC-0050–0055 имеют статус `integrated` для кода. SPEC-0054 прошла локальные
post-merge gates и выпущена в `a89a855`. Статусы не повышаются без соответствующего
post-merge evidence. Это не означает реализацию всего продуктового видения.
Колонка «Production» — первый документированный release, который содержит этап;
«—» означает, что выпуск не подтверждён или неприменим.

| Этап | Реализованный результат | Production | Спецификация | Отчёт проверки |
| --- | --- | --- | --- | --- |
| 0001 | Финансовые события, проводки, остатки, атомарные переводы и идемпотентность | ранние выпуски | [Scope](specs/0001-financial-events-and-postings.md) | [Evidence](specs/0001-evidence.md) |
| 0002 | Ручные подтверждённые чеки и позиции без повторного списания | ранние выпуски | [Scope](specs/0002-receipts-and-items.md) | [Evidence](specs/0002-evidence.md) |
| 0003 | Фиксированные обязательства, конечный график, частичные оплаты и просрочки | ранние выпуски | [Scope](specs/0003-obligations-and-payment-schedule.md) | [Evidence](specs/0003-evidence.md) |
| 0004 | Фактические сводки за период и ликвидная позиция после известных обязательств | ранние выпуски | [Scope](specs/0004-financial-periods-and-cash-position.md) | [Evidence](specs/0004-evidence.md) |
| 0005 | Детерминированные инструменты чтения финансового ядра | ранние выпуски | [Scope](specs/0005-assistant-read-tools.md) | [Evidence](specs/0005-evidence.md) |
| 0006 | Development-only оценка применимости Jev | — (dev-only) | [Scope](specs/0006-jev-applicability-evaluation.md) | [Evidence](specs/0006-evidence.md) |
| 0007 | Период после последнего подтверждённого дохода с точным фильтром income-категории | ранние выпуски | [Scope](specs/0007-confirmed-income-relative-periods.md) | [Evidence](specs/0007-evidence.md) |
| 0008 | Опциональный single-turn Jev router для шести инструментов чтения | выключен по умолчанию | [Scope](specs/0008-production-read-only-router.md) | [Evidence](specs/0008-evidence.md) |
| 0009 | Страница `/assistant` со структурированными результатами, уточнениями и ошибками | ранние выпуски | [Scope](specs/0009-assistant-ui.md) | [Evidence](specs/0009-evidence.md) |
| 0010 | Mobile-first визуальная система Web | ранние выпуски | [Scope](specs/0010-mobile-first-visual-system.md) | [Evidence](specs/0010-evidence.md) |
| 0011 | GitHub CI и публикация versioned API/Web images в GHCR | — (CI) | [Scope](specs/0011-github-repository-and-delivery.md) | [Evidence](specs/0011-evidence.md) |
| 0012 | Owner-only production, ручной release, backup/restore и механизм rollback | первый VPS-выпуск | [Scope](specs/0012-owner-vps-production-deployment.md) | [Evidence](specs/0012-evidence.md) |
| 0013 | Telegram OIDC, приглашения, серверные сессии и изоляция финансовых данных | да; Telegram временно отключён | [Scope](specs/0013-telegram-authentication-and-ownership.md) | [Evidence](specs/0013-evidence.md) |
| 0014 | Постоянная production-конфигурация Telegram и исходящий HTTPS API | да | [Scope](specs/0014-telegram-production-configuration.md) | [Evidence](specs/0014-evidence.md) |
| 0015 | Безопасная диагностика входа и совместимость со смешанным JWKS Telegram | да | [Scope](specs/0015-telegram-login-repair.md) | [Evidence](specs/0015-evidence.md) |
| 0016 | Офлайн-нормализация снимка таблицы, issues и построчная сверка без финансовых проводок | — (CLI оператора) | [Scope](specs/0016-spreadsheet-import-review.md) | [Evidence](specs/0016-evidence.md) |
| 0017 | Сохраняемые черновики импорта, audit/provenance, серверный preview и атомарное подтверждение | `f257f8e` | [Scope](specs/0017-persistent-import-drafts.md) | [Evidence](specs/0017-evidence.md) |
| 0018 | Публичный экран входа без старого Basic Auth | `f9875f5` | [Scope](specs/0018-public-telegram-entrypoint.md) | [Evidence](specs/0018-evidence.md) |
| 0020 | Обзор, мобильная навигация, модальные формы, карта отдельно от ledger account | `7b108df` | [Scope](specs/0020-home-redesign.md) | [Evidence](specs/0020-evidence.md) |
| 0021 | Общий доход не классифицируется как зарплата | `11f6b2e` | [Scope](specs/0021-explicit-salary-data.md) | [Evidence](specs/0021-evidence.md) |
| 0022 | Основной вход по почте, имена и приватный аватар, подтверждаемый перенос владельца | `5405363` | [Scope](specs/0022-email-authentication-profile.md) | [Evidence](specs/0022-evidence.md) |
| 0023 | Перенос месяца расходов: CSV/JSON, полная сверка месяца, защита от дублей | `f257f8e` | [Scope](specs/0023-monthly-expense-migration.md) | [Evidence](specs/0023-evidence.md) |
| 0024 | Favicon и плавность интерфейса с reduced motion | `f257f8e` | [Scope](specs/0024-interface-motion.md) | [Evidence](specs/0024-evidence.md) |
| 0026 | OpenRouter adapter, месячные объяснения, фото чека → черновик → подтверждение, удаление фото | `d475cca`; ИИ включается отдельно | [Scope](specs/0026-ai-assistant-receipts.md) | [Evidence](specs/0026-evidence.md) |
| 0027 | Исторические операции с неизвестным временем без выдуманных часов | `f257f8e` | [Scope](specs/0027-date-only-history.md) | [Evidence](specs/0027-evidence.md) |
| 0028 | Перенос истории в несверенные счета без подстановки нулевых остатков | `f257f8e` | [Scope](specs/0028-unreconciled-history-accounts.md) | [Evidence](specs/0028-evidence.md) |
| 0029 | Петрольная палитра, знак ₽ с пульсом, согласованные общие controls | `d475cca` | [Scope](specs/0029-design-foundations.md) | [Evidence](specs/0029-evidence.md) |
| 0030 | Графики статистики по неделям и месяцам на данных Go | `d475cca` | [Scope](specs/0030-statistics-charts.md) | [Evidence](specs/0030-evidence.md) |
| 0031 | Почта и пароль, email-bound приглашения и восстановление | `d475cca`; режим включается конфигурацией | [Scope](specs/0031-password-registration.md) | [Evidence](specs/0031-evidence.md) |
| 0032 | Единый полный интерфейс всех действующих экранов | `d475cca` | [Scope](specs/0032-ui-redesign.md) | [Evidence](specs/0032-evidence.md) |
| 0033 | Понятные счета/карты, личные категории, «Финансовое здоровье», согласие на ИИ | `d475cca` | [Scope](specs/0033-ux-refinement.md) | [Evidence](specs/0033-evidence.md) |
| 0034 | Подписки, сведения и PDF кредитов, черновик копии графика, почтовые напоминания | `75c8fbc` | [Scope](specs/0034-commitments-ai-reminders.md) | [Evidence](specs/0034-evidence.md) |
| 0035 | Ревизия дизайн-системы: выбор, фокус, шрифтовая шкала | `8ad9705` | [Scope](specs/0035-design-system-revision.md) | [Evidence](specs/0035-evidence.md) |
| 0036 | Ввод сумм с разделителями и обратная связь профиля | `8ad9705` | [Scope](specs/0036-amount-input-profile-feedback.md) | [Evidence](specs/0036-evidence.md) |
| 0037 | Ввод длинных графиков платежей | `8ad9705` | [Scope](specs/0037-long-schedule-entry.md) | [Evidence](specs/0037-evidence.md) |
| 0038 | Обратная связь при выборе и сохранении файла импорта | `8ad9705` | [Scope](specs/0038-import-file-feedback.md) | [Evidence](specs/0038-evidence.md) |
| 0039 | Компактный график обязательств на «Платежах» | `8ad9705` | [Scope](specs/0039-compact-payment-schedules.md) | [Evidence](specs/0039-evidence.md) |
| 0040 | Редактирование счетов и карт, подтверждаемая сверка остатка и её история | `8ad9705` | [Scope](specs/0040-account-reconciliation.md) | [Evidence](specs/0040-evidence.md) |
| 0041 | Исправление описательных полей и отмена операций с историей изменений | `8ad9705` | [Scope](specs/0041-operation-corrections.md) | [Evidence](specs/0041-evidence.md) |
| 0042 | Категории: изменение, удаление с переносом, иконки, скрытие общих | `8ad9705` | [Scope](specs/0042-category-management.md) | [Evidence](specs/0042-evidence.md) |
| 0043 | Список операций по дням, карточка операции, профиль в сайдбаре | `8ad9705` | [Scope](specs/0043-operations-list.md) | [Evidence](specs/0043-evidence.md) |
| 0044 | История импортов, пагинация и проверка сотен строк | `8ad9705` | [Scope](specs/0044-import-review-at-scale.md) | [Evidence](specs/0044-evidence.md) |
| 0045 | График обязательства по числу месяца и сроку договора | `8ad9705` | [Scope](specs/0045-schedule-by-period.md) | [Evidence](specs/0045-evidence.md) |
| 0046 | Статус «скоро платёж» за 3 и 1 день, рассчитанный в Go | `8ad9705` | [Scope](specs/0046-payment-due-soon.md) | [Evidence](specs/0046-evidence.md) |
| 0047 | Валюта списком, категория подписки, вёрстка графика | `e623775` | [Scope](specs/0047-subscription-form-and-schedule-layout.md) | [Evidence](specs/0047-evidence.md) |
| 0048 | Архитектура Web по Feature-Sliced Design без изменения поведения | `822b8f8` | [Scope](specs/0048-frontend-fsd-architecture.md) | [Evidence](specs/0048-evidence.md) |
| 0049 | Обновление счетов, операций и статистики после подтверждения фото чека | `822b8f8` | [Scope](specs/0049-receipt-confirmation-refresh.md) | [Evidence](specs/0049-evidence.md) |
| 0050 | Зашифрованный off-site backup и проверенное восстановление; статус `integrated` | `822b8f8`, production-проверки на `fc19d32` | [Scope](specs/0050-offsite-backup.md) | [Evidence](specs/0050-evidence.md) |
| 0051 | Версии условий карт, долг, грейс, минимум, снимки банка и погашение переводом; статус `integrated` | `95c5fa8` | [Scope](specs/0051-credit-cards.md) | [Evidence](specs/0051-evidence.md) |
| 0052 | Разделение Web по маршрутам, восстановление после ошибки чанка; статус `integrated` | `95c5fa8` | [Scope](specs/0052-route-code-splitting.md) | [Evidence](specs/0052-evidence.md) |
| 0053 | Тело/проценты кредита, досрочные погашения, снимки долга и разбивка статистики | `95c5fa8` | [Scope](specs/0053-loan-principal-interest.md) | [Evidence](specs/0053-evidence.md) |
| 0054 | Рассрочки/BNPL, связь покупки и долга, распределение переводов, общее банковское требование карты | `a89a855` | [Scope](specs/0054-installments-bnpl.md) | [Evidence](specs/0054-evidence.md) |
| 0055 | Текстовые банковские PDF, grounded spans, ручной/AI draft и явное подтверждение | `ec15861` (manual + AI + review repair/R8) | [Scope](specs/0055-bank-statement-pdf-import.md) | [Evidence](specs/0055-evidence.md), [activation/history](specs/0055-activation-evidence.md) |

Ранние записи evidence могут предшествовать итоговой интеграции; итоговый
статус этапа подтверждают строка статуса спецификации и дополнение в её
evidence. Production rollout Telegram проверен через CI, развёртывание точных
images, backup/restore drill и первый вход владельца; live benchmark Jev
не повторялся.

## Текущие возможности и ограничения

- **Финансовое ядро.** Go/PostgreSQL хранят факты и выполняют расчёты без LLM.
  Переводы между своими счетами отделены от расходов, чеки и settlement не
  создают повторных проводок. Деньги — integer minor units с явной валютой;
  валюты не суммируются через скрытую конвертацию.
- **Ввод и правка операций.** Web вводит покупки с категорией, доходы,
  переводы, платежи по обязательствам и подтверждённые чеки. Список операций
  сгруппирован по дням, фильтры дат и категорий выполняются на сервере.
  Описательные поля исправляются с историей «было → стало»; операция
  отменяется с причиной вместо удаления. Изменение денежных полей и массовая
  правка не реализованы; возвраты доступны через API без отдельной формы.
- **Импорт.** CSV/JSON снимки таблицы сохраняются как черновики с provenance;
  сервер сверяет каждую строку месяца, находит дубли и показывает preview перед
  атомарным подтверждением. История импортов, пагинация, фильтры по статусам
  и переход к следующей нерешённой строке рассчитаны на сотни строк. Даты без
  времени хранятся без выдуманных часов. История владельца перенесена
  2026-10-09 ([инструкция](imports.md)). Банковские PDF не импортируются.
- **Счета и сверка.** Счета и карты (банк и четыре цифры) редактируются,
  карта отключается без потери истории. Сверка остатка — подтверждаемое
  действие: Go считает разницу, для несверенного счёта выводит начальный
  остаток; история сверок сохраняется. Подключения к банкам нет.
- **Категории.** Личные категории расходов создаются, переименовываются,
  получают иконки и удаляются с переносом операций; общие категории можно
  скрыть. Правил автоматической категоризации нет.
- **Обязательства, подписки и кредиты.** Фиксированные графики строятся по
  числу месяца и сроку договора, вводятся длинными и показываются компактно
  (просрочка и ближайшие сверху); статус «скоро платёж» за 3 и 1 день
  считается в Go. Подписки (неделя/месяц/год, пауза, карта) связываются
  с реальными покупками без двойного учёта. Кредит хранит кредитора, договор,
  необязательные сумму и ставку, оригиналы PDF с версиями; копия графика
  редактируется как черновик и применяется отдельным подтверждением.
  По SPEC-0053 строки кредита имеют необязательные части тела/процентов/прочего;
  Grace помогает рассчитать части существующих строк по виду погашения,
  но не строит суммы/даты графика банка. Есть распределение фактической оплаты,
  досрочное погашение, снимки долга и сравнение с банком. Статистика отделяет
  погашенное тело от стоимости кредита и неизвестной части.
  В production `a89a855` SPEC-0054 связывает покупку с BNPL и детализирует
  рассрочку внутри карты. Старые графики автоматически не преобразуются.
  [Гайд](commitments-user-guide.md), [evidence](specs/0053-evidence.md).
- **Кредитные карты.** SPEC-0051 добавила версии условий, лимит и долг,
  модели грейса «от выписки»/«с первой покупки», оценку минимального платежа,
  снимки банка, PDF условий и напоминания. Погашение — перевод на кредитный
  счёт, без повторного расхода. Неизвестное не равно нулю; универсального
  расчёта банковских правил и рассрочек внутри карты нет. Поля грейса/минимума
  снимка сохраняются, но не заменяют расчёт Go автоматически.
- **Напоминания.** Опциональные письма за 3 и 1 день на verified email
  владельца через durable outbox с общим месячным лимитом и видимыми
  статусами; письма не содержат сумм и названий.
- **Фото чеков.** Фото внутри Grace распознаётся в типизированный черновик
  (RUB, один счёт, до 50 позиций), правится вручную и подтверждается как
  покупка с чеком либо привязывается к существующей; фото удаляется.
  После подтверждения обновляются счета, операции и статистика (SPEC-0049,
  код в production). Функция требует включённого ИИ
  ([гайд ИИ](ai-user-guide.md)).
- **Статистика и здоровье.** Графики расходов по неделям и месяцам, разбивка
  по категориям, явный период и ликвидная позиция после известных обязательств.
  «Финансовое здоровье» показывает доступность денег, известные платежи
  и полноту данных без выдуманного рейтинга. Прогнозов доходов и расходов,
  бюджетов и плана зарплаты нет.
- **Ассистент.** Шесть детерминированных tools чтения (`get_accounts`,
  `get_period_summary`, `get_cash_position`, `get_obligations`, `get_receipts`,
  `get_period_since_income`); Jev router выключен по умолчанию. LLM-объяснение
  месяца строится на отчёте Go и требует согласия и включённого OpenRouter.
  Изменяющих команд и истории диалога нет.
- **Вход и доступ.** Почтовый вход по приглашениям, admin/member, профиль
  с аватаром и изоляция данных (RLS). Password mode реализован, Telegram
  временно отключён. Без конфигурации API закрыт (`locked`).
- **Web-клиент.** Единая петрольная [дизайн-система](design-system.md):
  Radix controls, светлая и тёмная темы, mobile/desktop shell, reduced motion.
  Код `web/` организован по Feature-Sliced Design (app, pages, widgets,
  features, entities, shared) с автоматической проверкой границ слоёв
  (Steiger); серверное состояние — React Query. Бизнес-правила остаются в Go.
  Страницы загружаются по маршрутам (SPEC-0052); основной чанк после включения
  карт — 389,99 кБ по [проверке сборки](specs/0052-evidence.md).
- **Эксплуатация.** CI, публикация images в GHCR, ручной Deploy production
  с pre-release backup и отказом от небезопасного rollback миграций.
  Каждый backup шифруется ключом владельца и копируется в Yandex Object Storage
  (SPEC-0050); восстановление с машины владельца проверено на production-копии.

## Подтверждённые требования к целевому продукту

1. **Полнота расходов.** Сумма покупки и категория дополняются товарами:
   исходное название, конкретный продукт/вариант, бренд, упаковка, количество,
   единица, цена, скидка и итог. Два уровня анализа — товарная группа
   («молоко») и конкретный товар. Нормализованная цена за единицу позволяет
   сравнивать разную упаковку; неизвестные свойства не придумываются.
2. **Подтверждение владельцем.** Фото чека, PDF или предложение ассистента
   сначала создают черновик. Владелец проверяет и подтверждает данные, затем
   Go выполняет валидацию и атомарное сохранение. Извлечение ИИ не меняет
   финансовые факты самостоятельно. Неполная детализация остаётся видимой
   и не выдаётся за полную статистику корзины.
3. **Банковские выписки.** Поддержать PDF дебетовых и кредитных счетов:
   операции, период, начальный/конечный остаток, обороты и сверку. Для кредитных
   продуктов дополнительно извлекать указанные банком долг, проценты,
   комиссии, минимум, даты и требования грейса. Связывать выписки с чеками,
   договорами и графиками, сохраняя происхождение каждого значения.
4. **Действующие кредитные продукты.** Сравнивать только имеющиеся карты
   владельца. Разделять договор, счёт, карту/стикер, источник финансирования,
   основной долг, проценты, HOLD, доступный лимит и будущий график. Версии
   условий имеют дату действия и ссылку на документ; неизвестное не равно нулю.
5. **Одна кнопка анализа.** Запуск собирает согласованный снимок имеющихся
   подтверждённых данных. Ядро проверяет актуальность и полноту, рассчитывает
   платежи, грейс и сценарии; рассуждающая LLM объясняет результат и формирует
   мнение о стабилизации финансов и погашении долгов. При недостатке данных
   показываются пробелы и допущения. Прямой доступ к банкам не подразумевается.
6. **Jev и LLM.** Jev остаётся отдельным инструментом для ограниченных задач,
   где доказана его польза: классификации, маршрутизации и подобных решений.
   Рассуждающая модель подключается через отдельный заменяемый API-адаптер
   (например, OpenAI или Anthropic). Она помогает извлекать и заполнять данные,
   объяснять расчёты и давать подсказки. Финансовое ядро работает без обоих.

## Жизненный цикл источников

| Источник | Обработка и хранение | Что сохраняется независимо от файла |
| --- | --- | --- |
| Фото чека | Временная загрузка → распознавание → проверка/подтверждение → удаление фото. Архив фотографий не требуется; временные изображения не входят в постоянные резервные копии. Очистка отменённых/истёкших черновиков определяется спецификацией | Подтверждённый чек и позиции, исходные названия, суммы, связи с покупкой и история подтверждения; возможность последующей сверки без физического чека |
| PDF банковской выписки | Временная обработка и сверка; после подтверждения оригинал можно оставить или удалить по выбору владельца | Банк, счёт, период, дата импорта, подтверждённые операции, контрольные остатки/обороты, результат сверки, ссылки на исходные строки/страницы и отпечаток файла |
| PDF договора/условий кредитного продукта | Постоянное хранение оригинала и истории версий | Извлечённые условия, даты действия и ссылки на страницы, подтверждения и расхождения; объяснение оснований расчётов |

Повторная генерация выписки в банке может изменить байты PDF. Поэтому
отпечаток файла обнаруживает повтор того же файла, а отдельная сверка операций
предотвращает дубли между разными выписками. Чек и банковское списание могут
описывать одну покупку; перевод в двух выписках — одно событие с двумя
проводками. Погашение основного долга не является повторным потреблением;
проценты и комиссии учитываются отдельно. HOLD и окончательное списание
нельзя автоматически считать двумя покупками.

Новые документы и черновики сохраняют owner isolation, RLS, session, CSRF и
Origin. Спецификации извлечения определяют состав данных для внешнего API,
временное хранение и удаление; решения ИИ сохраняют происхождение и требуют
подтверждения перед финансовой записью. Публичные fixtures используют
синтетические данные.

## Порядок следующих этапов

2026-10-20: [S1.4 candidate screening](specs/0055-s14-candidate-screening.md)
accepted («работаем»), C1–C5 local/exact-source CI PASS, live C6 **FAIL provider-stage**
(run 39459614043), до первого score; качество UNVERIFIED. Разрешённый run выполнен: один Gemini 3.1
Flash Lite / Google EU run, columns/cards/RU, максимум 3 calls / $0.1617 provider.
Product policy и activation не расширяются; [evidence](specs/0055-s14-evidence.md).
Reconciliation расхода завершён отдельно в 17:10:51 UTC, uncertain block снят.
[Диагностика без inference](research/statement-quality/s14-provider-failure-diagnosis.md)
завершена 17:18:36 UTC: read-only audit/runtime/shared-ledger readiness PASS;
14 synthetic transport cases PASS. Точный исходный guard UNVERIFIED — не сохранён
chat envelope, несколько guards дают общий код. Следующий bounded пакет
[S1.5 diagnostics](specs/0055-s15-screening-diagnostics.md) — accepted/integrated (PR №78, `29a28a7`)
(прямое «Утверждаю», 2026-10-20), [D1–D5 PASS](specs/0055-s15-evidence.md):
безопасные коды отказа, HTTP category, private unverified correlation и report v2.
make test (334 web), real helper + fake-provider 19/20/23, SQL lifecycle/restore и
deployment mocks PASS. Не deployed, новых paid calls нет; current-head CI отдельно.
После него — отдельное решение о новом bounded smoke. S1.3 интегрирована (PR №78, `29a28a7`),
не deployed; новая модель не допущена в production.

2026-10-20 владелец поручил добавить регрессию выбора банковских реквизитов
вместо оборотов и подбор более современных и эффективных моделей.
[Следующий пакет качества](research/statement-quality/next-quality-package.md)
фиксирует последовательность: synthetic regression → исправление grounding →
сравнение актуальных model/endpoint пар → отдельно согласованный AI smoke →
полный corpus. [S1.3](specs/0055-s13-turnover-roles.md) **accepted** прямым поручением
«оформи спеку и вперед», реализация **integrated** (PR №78, `29a28a7`),
[T1–T6 PASS](specs/0055-s13-evidence.md), не deployed.
[Metadata shortlist](research/statement-quality/model-shortlist-2026-10-20.md):
Gemini 3.1 Flash Lite / Google EU и Mercury 3 / Inception; обе пары
**unqualified**, production policy не меняется. Новизна модели не заменяет качество; сравниваются
точность ролей/строк, полнота, стоимость успешного документа и задержка.
Production policy пока прежняя; новые платные вызовы этим поручением не запущены.

2026-10-19 согласовано направление автоматического выбора модели для выписки.
[SPEC-0056](specs/0056-statement-model-selection.md) — **accepted D1–D4**, реализация
**not_started**: quality/privacy gates, выбор стоимости полного плана до
consent, закреплённый маршрут без скрытого fallback. Сверка расхода
SPEC-0055 завершена; исторические FAIL сохраняются. Statement AI включён для owner testing по отдельному accepted S1.2. Конкретный реестр квалифицированных моделей SPEC-0056 ещё не утверждён.
2026-10-19 владелец принял [проработку устойчивости](research/statement-quality/README.md)
(«Принимаю», PR #68):
32 offline probes выявили разрыв record core при добавлении нефинансового
префикса. Предложены отдельные пакеты planner, явного coverage protocol и
24-case corpus (2 PDF существуют, 22 planned). Model calls = 0; реализация
S2/corpus not_started. S1 реализован, интегрирован и включён в текущий production с accepted S1.1/S1.2 amendments; его первоначальная offline проверка:
в рамках [принятого S1 контракта](specs/0055-record-atomic-planning.md):
[280 offline cases](research/statement-quality/atoms.md) отклонили геометрические
эвристики. Владелец принял целый документ с короткими aliases и новыми caps
90000 bytes / 100000 input / 16000 output, максимум 2000 spans. Два baseline
formats и все 56 structural cases этой стратегии помещаются без разрывов;
model quality не проверялась. [S1 implementation evidence](specs/0055-s1-evidence.md):
84 actual-planner layout cases, real HTTP/fake/SQL/confirmation, shared budget
and cancellation guards; локальные/CI/Delivery gates PASS, [PR #70](https://github.com/QahNaaRlN/grace/pull/70) integrated (`8ed8e80`); текущий release/enable зафиксирован в S1.2 evidence. Принятие
зафиксировано 2026-10-19 («Утвердить S1 с новыми лимитами»); paid campaigns
отдельно, общий бюджет не увеличивается.

Это рекомендуемая последовательность с зависимостями, без обещания сроков.
Каждый этап может потребовать нескольких технических спецификаций; до кода
нужно их явное принятие по [spec-first процессу](specs/README.md).
Отметки: **завершено**, **частично** (что сделано — со ссылками), **не начато**.

| Порядок | Этап и пользовательский результат | Состояние на 2026-10-19 | Зависимости и критерий готовности |
| --- | --- | --- | --- |
| 1 | **Нормализация эталонной таблицы и отчёт сверки.** Типизированный снимок, точные суммы/даты, проблемы схемы, сопоставление каждого дополнительного ряда | **Завершено:** [SPEC-0016](specs/0016-spreadsheet-import-review.md), [инструкция](import-review.md) | Все дополнительные строки проверены; кандидаты не удаляются/сливаются автоматически. Ledger не меняется |
| 2 | **Подтверждаемый импорт и происхождение данных.** Общие черновики, source records, история проверки, стабильные связи и preview перед записью | **Частично:** импорт таблицы (SPEC-0017/0023/0027/0028/0038/0044), черновики фото чека (0026) и копии графика кредита (0034), история правок операций (0041). Банковские PDF source records/spans/provenance и общий review/confirmation добавлены SPEC-0055; прочие типы документов ещё не унифицированы | Идемпотентность, tenant isolation, явное подтверждение, связь с существующими фактами. Неполные/конфликтующие данные остаются черновиками |
| 3 | **Счета, кредитные договоры и финансирование.** Канонические сущности/aliases, снимки долга, версии условий и графиков, требования minimum/grace/installment, выдачи и погашения с распределением тела/процентов/комиссий | **Частично:** счета/сверка (0040), сведения/PDF/копии графика (0034), удобные графики и сроки (0037/0039/0045/0046); карты — версии условий, снимки долга, minimum/grace и погашение (0051); кредиты — части строк, распределение оплаты, досрочное погашение и снимки долга (0053). Рассрочки/BNPL (0054) интегрированы и выпущены в `a89a855`. Остались запись выдачи кредита и общая модель источников финансирования | Принятый scope — SPEC-0054. Не менять legacy-графики неявно; покупка и погашение не удваивают расходы, карточная рассрочка не удваивает долг. Выдача не является доходом; правила конкретных банков требуют подтверждения |
| 4 | **Отдельное подключение рассуждающей LLM.** Заменяемый API-адаптер для извлечения, диалога и объяснений; Jev сохраняет отдельную роль | **Частично:** OpenRouter adapter, политика моделей и бюджет, месячные объяснения, чтение фото чека (0026/0033/0034). Production global AI/key readiness подтверждена 2026-10-19 без передачи бюджетных значений. PDF extraction adapter/consent/budget реализованы SPEC-0055; Statement AI enabled для owner testing (`7b2deea`); S1.2 оба synthetic cases PASS, full corpus отдельно, диалог остаётся отдельным scope | Извлечение возвращает валидируемые черновики; объяснения используют результаты tools. Ошибка/отключение ИИ не нарушает ledger и детерминированные расчёты |
| 5 | **Импорт банковских и кредитных PDF.** Дебетовые/кредитные выписки, документы условий и графики, контроль остатков, matching с существующими операциями | **Частично:** SPEC-0055 интегрирована и выпущена: один текстовый debit RUB PDF → spans → ручной/AI draft → Go validation → matching/confirmation. Ручной путь доступен, Statement AI enabled для owner testing (`7b2deea`); S1.2 оба synthetic cases PASS, full corpus отдельно. OCR, кредитные выписки/условия и подтверждение качества других банков не выполнены | Этапы 2–4. Страницы и источник каждого значения доступны; повторная выписка/парный перевод/покупка с чеком не дают повторного учёта. Правила хранения различаются по типу PDF |
| 6 | **Товарная детализация и временное распознавание чеков.** Группы и конкретные продукты, quantity/unit/price/discount, подтверждённый чек из фото | **Частично:** фото → черновик позиций → подтверждение с удалением фото и обновлением данных (0026/0049). Нет товарных групп, конкретных продуктов и цены за единицу | Этапы 2 и 4. Совпадение суммы чека с покупкой, сопоставление товаров с возможностью исправить, видимая неполнота; подтверждение не создаёт вторую проводку, фото удаляется |
| 7 | **Аналитика и план финансовой нагрузки.** Потребление own/borrowed/unknown, цены товаров/корзина, план и факт зарплаты, бюджеты, приоритеты и будущие обязательства | **Частично:** графики статистики (0030), личные категории (0033/0042), «Финансовое здоровье» с полнотой данных (0033), явная зарплата (0021), разбивка платежей кредита на тело/проценты/прочее/неизвестное (0053). Нет общей аналитики own/borrowed, корзины, бюджетов, плана зарплаты и прогноза | Этапы 3, 5 и 6 для соответствующих данных. Расчёты в Go, раздельные валюты, явные даты/точность, полнота и актуальность; прогноз отделён от подтверждённых фактов |
| 8 | **Анализ действующих карт и долгов одной кнопкой.** Снимок, детерминированные сценарии платежей/грейса и объяснение LLM с рекомендацией | **Не начато** | Этапы 3–5 и 7. Расчёт воспроизводим относительно даты анализа; используются применимые условия, собственные деньги и обязательства. Показаны источники, цели, ограничения и недостающие данные; нет самостоятельных платежей/изменений |

После общей основы этапы PDF и чеков можно разрабатывать независимо в рамках
их принятых scope. Анализ одной кнопкой не обещает универсальный расчёт грейса
до реализации подтверждённых правил конкретных действующих продуктов.

Ближайший owner-срез — явная проверка ролей/страниц, семантики строк и duplicates,
затем подтверждаемый preview. [Review repair](specs/0055-review-repair.md) выпущен
в `ec15861`; [R1–R8 evidence](specs/0055-review-repair-evidence.md) PASS в scope
ручного review, rows unchanged/turnover matched/unmapped=0. Отдельно требуется
исследование автоматического выбора банковских реквизитов вместо денежных итогов;
ручная коррекция не доказывает улучшение качества LLM. Statement AI enabled сохранён. Отдельно
остаются AC12 (репрезентативный корпус/русский язык, качество строк и колонок,
omissions/abstentions, latency, LLM-only/Jev/cascade), owner acceptance,
кредитные/мультивалютные PDF и OCR. Synthetic activation smoke не заменяет AC12.
Предоставленный личный reference не отправляется внешнему API без отдельного
согласия на конкретный preview в Grace.

После этого — продолжение источников/финансирования (этапы 2/3/5), товарная
детализация (этап 6), аналитика (7); анализ долгов одной кнопкой остаётся этапом 8.
Эксплуатационные evidence включают точный SHA, CI/Delivery/deploy, backup/restore,
readiness и auth smoke; пользовательская проверка фиксируется отдельно.

## Дополнительные направления

| Направление | Что сделано / что остаётся | Граница |
| --- | --- | --- |
| Дополнительные способы входа и recovery | Сделано: почта, пароль, восстановление по почте (0022/0031). Осталось: passkeys, Google/Apple ([SPEC-0019](specs/0019-google-apple-mobile-login.md) — draft) | Telegram временно отключён; identity не связываются по username |
| Категоризация и память правил | Сделано: личные категории и иконки (0033/0042). Осталось: CategorizationRule, MerchantRule, использование подтверждений | Правила работают детерминированно, предложения ИИ проверяются владельцем |
| Повторяющиеся расходы | Сделано: подписки с напоминаниями и связью с покупками (0034/0047). Осталось: правила ожидаемых платежей и счетов | Признак регулярности не создаёт фактические проводки |
| Правка операций | Сделано: описательные поля и отмена (0041). Осталось: денежные поля, массовая правка, форма возврата | Подтверждённые чеки неизменяемы; физического удаления нет |
| Изменяющие инструменты ассистента | Подтверждаемые команды поверх общего процесса черновиков | Самостоятельные изменения и выполнение платежей не входят в текущий план |
| Нативный iOS-клиент | SwiftUI, загрузка чеков, локальный cache | Использует Grace API; финансовая модель и арифметика остаются в Go |
| Эксплуатация | Сделано: off-site backup и проверенное восстановление базы (0050). Осталось: регулярный restore drill по расписанию | Учитывать разные правила хранения постоянных PDF и временных фото |

Целевые сценарии «куда ушёл аванс?», «какие продукты дорожают?» и «как
стабилизировать ситуацию с действующими картами?» пока не реализованы целиком.
Roadmap связывает их с проверяемыми этапами сбора данных, финансовых расчётов
и объяснений, сохраняя работоспособность продукта без ИИ.

## Как поддерживать документ

После интеграции новой спецификации обновить таблицу завершённых этапов,
границы возможностей и соответствующее оставшееся направление; после выпуска —
колонку «Production». Статус подтверждать спецификацией, кодом и итоговым
evidence, а не только наличием каталога или draft. Исторические решения,
спецификации и отчёты сохранять; их исходные наблюдения не считать описанием
текущей версии.

## Исторический owner-test release snapshot — 2026-10-19

S1 интегрирован PR #70 (`8ed8e80`), main CI `39243697470` и Delivery
`39244588305` PASS. Владелец отдельно принял
[owner-test activation amendment](specs/0055-owner-test-activation.md):
обновить smoke под S1, один запуск columns/cards (≤2 calls / theoretical
$0.14 total, прежний shared budget), включение только при полном PASS обоих.
[Implementation evidence](specs/0055-owner-test-activation-evidence.md) отделяет
offline проверки от предстоящих release/runtime gates. Production AI в этом
snapshot не включён; full qualification S2–S5 и AC12 остаются отдельными.

## Исторический S1 live outcome и S1.1 — 2026-10-19

Один разрешённый S1 smoke `39248364426` использован и остановлен после
columns quality FAIL; cards NOT_RUN, никакого повтора до PASS.
Read-only locators `39249075304` и network-none [offline evidence](research/statement-quality/owner-smoke-candidate-offline.json)
доказали permissive money eligibility и wrong description bindings. Владелец
ответил «Утвердить S1.1, один smoke и включение при PASS» на точный
[contract](specs/0055-s11-lexical-grounding.md); прежний shared budget/fees и
полные quality/confirmation gates сохраняются. На этом историческом этапе цель owner-test enable оставалась активна; текущий исход S1.2 — ниже.

## Исторический S1.1 live outcome и принятие S1.2 — 2026-10-19

S1.1 release `ef782ec` прошёл все offline/CI/Delivery/internal-health gates;
один smoke `39255669019` остановился на неполном описании columns, cards NOT_RUN.
Read-only `39255943568` и public/auth PASS; AI false. Бесплатный pinned-helper
[replay](research/statement-quality/s11-description-offline.json) подтвердил
пропущенный separator, без починки ответа или ослабления scorer. Владелец принял
[S1.2](specs/0055-s12-description-separators.md), реализацию и один новый bounded
smoke, включение только при обоих полных PASS. Следом — owner UI test; AC12,
Jev/cascade и expanded corpus остаются самостоятельными этапами.

## S1.2 owner-test activation — 2026-10-20

PR #74 integrated `7b2deea`, все local/PR/main CI и Delivery gates PASS.
Единственный разрешённый smoke `39396162864` прошёл columns19/19 и cards20/20.
Actual image/runtime/current policy PASS 13:51:07 +05:00; public/auth PASS.
Statement AI enabled. [D1–D5 и safe payloads](specs/0055-s12-evidence.md).
Owner private test, S2–S5/AC12, Jev/cascade, русский/expanded corpus, credit/OCR
и draft SPEC-0056 остаются открытыми; автоматического расширения scope нет.
