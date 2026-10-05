# Grace

Grace — персональная система финансового здоровья. Она хранит достоверную
финансовую модель пользователя, выполняет расчёты детерминированно и
предоставляет опциональному ассистенту ограниченные инструменты чтения этой модели.

Текущее состояние, завершённые этапы и оставшиеся направления описаны в
[roadmap](docs/roadmap.md). [Видение](docs/vision.md) задаёт целевой продукт,
а [архитектурный контракт](docs/architecture.md) — правила реализации.

## Вход и доступ

Telegram OIDC, одноразовые приглашения и серверные сессии реализованы в
SPEC-0013. Каждый пользователь получает доступ только к своим финансовым
данным; административная роль не открывает чужие финансы. Настройка первого
входа и ограничения: [docs/authentication.md](docs/authentication.md).
По умолчанию API закрыт (`locked`); development Compose явно включает
локальный режим разработки. Настоящий Telegram вход требует HTTPS и
server-only credentials из BotFather.

## Финансовое ядро

Текущая версия реализует пути Account → FinancialEvent → Posting → Category и
Purchase → Receipt → ReceiptItem, а также
Obligation → ScheduledPayment → Settlement:

- PostgreSQL хранит счета, категории, финансовые события и проводки;
- Go API вычисляет остаток из начального значения и неизменяемых проводок;
- внутренний перевод атомарно создаёт debit и credit;
- изменяющие команды защищены обязательным Idempotency-Key;
- OpenAPI задаёт контракт для клиентов;
- TypeScript-типы генерируются из OpenAPI;
- React показывает счета и события и позволяет добавить покупку, доход или
  внутренний перевод или платёж по обязательству; возвраты доступны через API;
- подтверждённый чек детализирует существующую покупку упорядоченными позициями,
  но не создаёт проводки и не меняет остаток повторно;
- чек вводится вручную, а его итог и связь с покупкой проверяются Go и
  PostgreSQL;
- фиксированные обязательства имеют график с просрочками и частичными оплатами;
  связь фактического платежа с графиком не создаёт повторного движения денег;
- аналитика показывает денежный поток за выбранный период и доступные ликвидные
  средства после известных обязательств до выбранной даты;
- инструмент ассистента умеет строить период после последнего подтверждённого
  дохода, в том числе с точным фильтром по income-категории.

Финансовое ядро и обычный API не зависят от LLM. Опциональный single-turn
read-only router использует Jev только для выбора одного allowlisted tool;
арифметика и финансовые данные остаются в Go/PostgreSQL. Router выключен по
умолчанию. Для локального opt-in задайте на сервере:

    export GRACE_ASSISTANT_ROUTING_ENABLED=true
    export TYPESAFE_API_KEY=<server-only key>
    make dev

Текст текущего запроса, язык и reference time при этом передаются TypeSafe;
финансовые записи и результаты tools провайдеру не отправляются. OCR,
автономные записи и model-generated финансовые ответы не реализованы.

После запуска страница `/assistant` доступна из навигации Web. Она отправляет
один вопрос за раз, не хранит историю и показывает только структурированные
read-only результаты Grace. При выключенном router страница объясняет, что
ассистент недоступен, не затрагивая остальные функции приложения.

## Быстрый старт

    make dev

После запуска:

- Web: http://localhost:5173
- API: http://localhost:8080
- PostgreSQL: localhost:55432

Остановка:

    make down

## Проверки

    make test

На GitHub те же quality gates, PostgreSQL integration suites и production
Docker builds выполняет GitHub Actions. После успешного CI для `main` workflow
Delivery публикует versioned API/Web images в GHCR; это публикация artifacts, а
не автоматический production deployment. Подробности:
[docs/github-delivery.md](docs/github-delivery.md).

Owner-only deployment на `graceapp.ru`, ручной release, backup и rollback
описаны в [docs/production-deployment.md](docs/production-deployment.md).

После изменения api/openapi.yaml обновите клиентские типы:

    make generate-api

## Структура

    web/                  React-клиент
    cmd/api/              точка входа Go API
    internal/accounts/    предметный модуль счетов
    internal/categories/  предметный модуль категорий
    internal/financialevents предметный модуль событий и проводок
    internal/receipts/   предметный модуль подтверждённых чеков
    internal/obligations/ фиксированные обязательства, графики и оплаты
    internal/analytics/   периоды и ликвидная позиция
    internal/assistant/   инструменты чтения и опциональный Jev router
    internal/auth/        Telegram OIDC, приглашения и серверные сессии
    internal/jeveval/     development-only оценка применимости Jev
    internal/routereval/ проверка production routing policy
    internal/platform/    конфигурация, БД и HTTP-композиция
    api/openapi.yaml      стабильный контракт клиентов
    migrations/           PostgreSQL-миграции
    docs/                 архитектурные решения

Правила разработки зафиксированы в [docs/architecture.md](docs/architecture.md).

## Процесс разработки

Grace использует spec-first:

1. Создать draft-спецификацию в `docs/specs/`.
2. Получить явное принятие спецификации владельцем.
3. Создать отдельный worktree в соседнем каталоге `../grace-worktrees/`.
4. Реализовать только принятый scope и собрать доказательства критериев приёмки.
5. Интегрировать проверенный commit в `main`.

Полный контракт процесса: [docs/specs/README.md](docs/specs/README.md).
