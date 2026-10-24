# Архитектурный контракт Grace

## Граница продукта

Grace — один продукт и один monorepo. Web и будущие нативные приложения
являются клиентами Go API. PostgreSQL — источник истины.

## Направление зависимостей

    Web / iOS → Grace API → financial modules → PostgreSQL
                         ↘ assistant tools

Финансовые модули не зависят от ассистента. Отключение языковой модели не
должно нарушать хранение, расчёты или API.

## Организация backend

Код группируется по предметным областям: accounts, financialevents, categories,
receipts, obligations, analytics, assistant, auth; creditcards, loans, installments
и commitments расширяют кредитные продукты, графики и напоминания.
Development-only оценка модели и маршрутизации находится в jeveval и routereval;
интеграционные проверки инструментов — в assistanttest. Техническая композиция
находится в internal/platform.

Внутри модуля новые слои выделяются только при появлении реальной сложности.
Глобальные каталоги controllers/services/repositories/models не вводятся.

## Данные и деньги

- Денежные суммы передаются и хранятся целым числом в минимальных единицах
  валюты: amountMinor и balanceMinor.
- Валюта всегда указана явно.
- Сумма проводки положительна; направление задаётся отдельно.
- FinancialEvent выражает финансовый смысл, а Posting — движение по счёту.
- Перевод между собственными счетами является одним событием с ровно двумя
  атомарными проводками и не считается расходом.
- Текущий остаток вычисляется как opening balance + credits - debits.
- Повтор изменяющей команды защищён Idempotency-Key.
- Receipt детализирует существующую purchase, но не создаёт Posting.
- Итог Receipt равен сумме line totals и debit-проводке связанной purchase.
- Quantity хранится целым `quantityMilli`; деньги позиций остаются minor units.
- Подтверждённые Receipt и ReceiptItem неизменяемы в текущем контракте.
- Obligation и ScheduledPayment описывают план и не создают Posting.
- Фактическая оплата обязательства существует только как `debt_payment`;
  Settlement связывает это событие с одним ScheduledPayment и не меняет баланс
  повторно.
- SPEC-0054: для управляемой рассрочки credit-проводка `internal_transfer`
  распределяется по нескольким строкам графика без новых проводок. Покупка
  на долговом счёте учитывается один раз, погашение тела не становится
  вторым расходом. Старые Settlement и графики не преобразуются автоматически.
- Состояния графика, paid/remaining и просрочка вычисляются детерминированно в
  финансовом модуле относительно явной календарной даты.

## API

Финансовый API требует principal, полученный из серверной сессии. Все
финансовые таблицы изолированы по owner_id с RLS под ограниченной PostgreSQL
ролью; SET LOCAL внутри транзакции не оставляет identity в pooled connection.
Общие категории доступны только для чтения. Составные FK не допускают
связей финансовых объектов разных владельцев. Admin управляет доступом,
но не получает финансовых привилегий другого пользователя.

Основной вход использует одноразовый почтовый код по SPEC-0022. Telegram OIDC
Code Flow/PKCE сохранён в коде и identities, но временно отключён. Сессии непрозрачны и хранятся
на сервере. Mutations требуют CSRF token и точный Origin. Без настроенного
входа действует locked mode; development bypass разрешён только явно в
локальной development-конфигурации. См. [почтовый вход](email-authentication-rollout.md)
и [Telegram runbook](authentication.md).

По принятой SPEC-0031 подготовлен password mode: регистрация по привязанному
к почте приглашению, подтверждение адреса, вход email/password и mail-link reset.
Существующие verified identities/UUID сохраняются. Production переключение ещё
не выполнено; [порядок перехода и rollback](password-authentication-rollout.md).

api/openapi.yaml — публичный контракт. TypeScript-клиент не объявляет DTO
вручную. Изменение API считается завершённым только после обновления контракта
и сгенерированных типов.

## Web UI

Web следует mobile-first подходу: базовые стили и информационная иерархия
рассчитаны на телефон, а breakpoint-правила только расширяют layout. Базовая
компонентная система — shadcn/ui `radix-nova` на Radix UI и Tailwind CSS;
параллельная библиотека компонентов без отдельного принятого решения не
вводится. Интерактивные элементы должны иметь видимое focus-состояние,
корректный cursor и touch target не меньше 44 CSS px для основных действий.

### Структура Web-кода

`web/src` следует Feature-Sliced Design ([решение 0003](decisions/0003-web-feature-sliced-design.md),
[SPEC-0048](specs/0048-frontend-fsd-architecture.md)). Слои зависят только сверху вниз:
`app → pages → widgets → features → entities → shared`.

- `app` — провайдеры (`app/setup`) и router: публичные страницы входа и один
  layout route (`AuthBoundary` → оболочка `widgets/app-shell` → `<Outlet>`) для
  остальных маршрутов.
- `pages` — один срез на маршрут, только компоновка: страница не обращается к API
  и не ветвится по `pathname`. `widgets` — самостоятельные блоки экрана, которые
  читают данные и собирают фичи (например, `ledger-screen` — общий каркас
  денежных страниц). `features` — пользовательские действия с записью; фичи
  входа сгруппированы в `features/auth`, импорта — в `features/import`. `entities` — доменные объекты из
  OpenAPI, их `queryOptions`, подписи и управляемые компоненты (график платежей).
  `shared` — транспорт API, UI kit, утилиты и конфиг без знания о домене.
- Срез открыт наружу только через `index.ts` (тестовые фейки — через
  `index.testing.ts`); срезы одного слоя не импортируют друг друга, общее
  передаётся сверху через props или опускается ниже, связь сущностей — через `@x`.
- Ключи React Query объявляются в `entities/*/api` от корня `all()`; запись идёт
  через хуки `features/*/api`, которые сами инвалидируют затронутые ключи через
  `all()` и только при успехе. CSRF-заголовок ставит только `apiClient`.
  Часовой пояс отображения — константа в `shared/config`.
- Модуль `web/src` вне сгенерированных типов и тестов — не длиннее 300 строк.
- `npm run lint:fsd` (Steiger, `--fail-on-warnings`), ESLint-правила
  `eslint-rules/grace.js` и `npm run format:check` (Prettier) входят в
  `make test`. Списков исключений нет: нарушение исправляется в коде.

## ИИ

ИИ не хранит финансовое состояние и не выполняет арифметику вместо Grace.
Текущий ассистент предоставляет шесть read-only tools: get_accounts,
get_period_summary, get_cash_position, get_obligations, get_receipts и
get_period_since_income. Опциональный Jev router выбирает один инструмент
для одного запроса; Go валидирует аргументы и выполняет финансовые расчёты.
Router выключен по умолчанию и в production-конфигурации.

TypeSafe получает текст запроса, язык и reference time; финансовые записи и
результаты tools провайдеру не отправляются.
SPEC-0056 переводит генеративную классификацию этого режима на общий metered
OpenRouter, с отдельным согласием analysis и без имитации вероятностей Jev.
TypeSafe live adapter исключён из API composition.

По accepted SPEC-0026 локально реализованы OpenRouter: объяснения месячных итогов,
история разговоров и временные черновики фото чеков. В ветке SPEC-0033 эта работа
объединена с password mode и общим UI. Production global AI/key readiness
проверена 2026-10-19; Statement AI disabled / activation BLOCKED (расход сверён; cards smoke pending). Доступность
определяет сервер по policy/провайдеру/бюджету. Расчёты Go и факты отделены от
интерпретаций модели. Передача итогов и фото требует отдельных согласий;
запись расхода или привязка чека требует явного preview/confirmation, а не ответа
LLM. [Гайд пользователя](ai-user-guide.md), [спецификация](specs/0026-ai-assistant-receipts.md).

Категории бывают общими (read-only) и личными расходными. RLS ограничивает чтение
и переименование owner_id; DB trigger не допускает чужую категорию в событии.
Переименование сохраняет id/ссылки, не меняет суммы/проводки/историю времени.

## Банковские PDF — SPEC-0055, integrated 2026-10-19

PR #64 (`b51b74f`, production 2026-10-19) добавил `internal/statementdata` для
чистых grounding/decimal/date/coverage проверок и `internal/pdfextract` для
локального Unix-socket транспорта. Отдельный `cmd/pdf-extractor` запускает
проверенную версию Poppler в контейнере без сети и credentials. Финансовый API
не разбирает PDF внутри своего процесса. Локальный opt-in overlay:
`compose.pdf-extractor.yaml`. Production использует отдельный sandbox в
`deployments/production/compose.yaml` и exact-SHA GHCR helper image.

`internal/statementextract` зависит от `imports` и получает Reasoner через
interface. Composition root подключает существующий OpenRouter transport из
`assistant`; `imports` и финансовые модули не зависят от модели. Statement имеет
собственные document-scoped consent, plan и attempts, но использует общий budget
ledger. AI выключен по умолчанию. Ответ модели содержит span IDs, а значения
копирует и проверяет Go. Ручной draft использует те же проверки без inference.

PDF source — отдельный вариант, без вымышленных sheet/row. `imports` сохраняет
решения и matching по Posting; общий preview/confirmation остаётся единственным
writer. PDF reconciliation отдельно показывает обороты, book balances, available
balance, source coverage и financial representation. Legacy `/month` для PDF
возвращает явный 422; JSON/CSV month-review сохраняется. Оригинал/full layout и
consent plan очищаются по retention, минимальные fragments/provenance остаются.
[Гайд](bank-statements.md), [состояние проверок](specs/0055-evidence.md).

## Проверяемое финансовое ядро

Account → FinancialEvent → Posting → Category должен оставаться работоспособным через
миграцию, Go API, OpenAPI, TypeScript-клиент и React.

Purchase → Receipt → ReceiptItem расширяет этот путь подтверждённой детализацией
без второго движения денег. Этот контракт также работает без AI, OCR и внешних
провайдеров.

Obligation → ScheduledPayment → Settlement связывает будущий план с уже
состоявшимся `debt_payment`. График не является бухгалтерской проводкой и не
дублирует расход.

## Границы текущей реализации

Обязательства имеют фиксированную сумму и конечный график. SPEC-0051 добавила
условия карт, долг, грейс и минимальный платёж; SPEC-0053 — части строк кредита,
досрочные погашения и снимки долга. SPEC-0054 связывает беспроцентную рассрочку
с покупкой и долговым счётом: отдельный BNPL использует счёт `installment`,
карточная рассрочка детализирует существующий credit-счёт. Требование банка
для карты с активной рассрочкой подтверждается вручную; неполные или устаревшие
данные не дают точной доступной суммы. Это не универсальная модель правил банков.
Аналитика вычисляет фактический поток и текущую ликвидную позицию, но не
прогнозирует будущие доходы и расходы.

React Web — реализованный клиент; iOS остаётся направлением развития.
По SPEC-0018 Caddy открывает Web shell и экран входа без Basic Auth;
по SPEC-0022 production использует почтовый код.
Финансовый API требует собственную сессию; приглашения, CSRF/Origin и RLS
сохраняются. Публикация исправления проходит через отдельный release pipeline.
API, Web и PostgreSQL доступны только во внутренней Docker network.

Статусы этапов и ограничения: [roadmap](roadmap.md). Эксплуатация:
[production runbook](production-deployment.md).

## Чтение изображений документов — SPEC-0056

Принятый G1–G4 реализуется отдельным пакетом: `assistant.ModelGroups` выбирает
квалифицированный конкретный model/provider по свежему capabilities proof.
`documentdata` хранит ограниченные нормализованные страницы, а networkless
`pdfextract` рендерит PDF локально. `statementextract.DocumentReasoner` получает
снимок выбранной policy, исходника и страниц до согласия и бюджетного резерва.
Чеки используют тот же transport и групповой resolver. Прямые OpenAI/TypeSafe
live research entrypoints закрыты; исторический replay сохранён.

Native PDF сохраняет реальные spans и grounding. Для фото/скана отдельный
`image_review` хранит сырой ответ, правки владельца, source SHA и digests строк.
Независимый OCR не заявляется. Проверка владельцем открывает только прежний
import preview/confirmation; LLM не является финансовым writer. Migration30
добавляет source/render/review metadata и очистку по существующему retention.
API-контракт и UI поддерживают приватные страницы и ручную сверку.
Production не активирован, реальное качество модели UNVERIFIED.
[Конфигурация и пользовательский путь](openrouter-document-reading.md).
