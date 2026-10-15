# SPEC-0048: архитектура Web по Feature-Sliced Design

- Статус: accepted
- Автор: Claude Code по поручению владельца
- Дата: 2026-10-15
- Acceptance owner: владелец Grace
- Acceptance record: владелец 2026-10-15 в чате: «Принимаю SPEC-0048 с рекомендованными
  решениями (Steiger, Prettier без ; и с одинарными кавычками). Начинай фазу 0.»
  Исходное поручение 2026-10-15: «Давай улучшим frontend составляющую. Интегрировать FSD,
  выработать стратегию по организации лучшей архитектуры».
- Implementation branch: фаза 0 — `codex/frontend-fsd` (вместе со спецификацией);
  следующие фазы — отдельные ветки `codex/frontend-fsd-<фаза>`
- Implementation commit: фаза 0 — `3edc372`, `7a7d946` (форматирование) и коммит evidence
- Evidence: [0048-evidence.md](0048-evidence.md)

## Проблема и пользовательский результат

В `web/src` есть каталоги слоёв FSD (`app`, `pages`, `widgets`, `features`,
`entities`, `shared`), но методология не соблюдается и ничем не проверяется.
Из-за этого крупные экраны трудно менять, а серверное состояние, правила дат и
форматирования повторяются в нескольких местах.

Результат для владельца: поведение и внешний вид Grace не меняются. Web получает
предсказуемую структуру, автоматическую проверку границ слоёв и меньшие по
размеру модули. Следующие этапы roadmap (кредиты, PDF, товарная детализация)
добавляются как новые срезы, а не как ветки внутри `HomePage`.

## Текущее поведение и доказательства

Снимок `main` `e623775`, 2026-10-15.

1. **Пустые слои.** `entities/` и `widgets/` содержат только `.gitkeep`. Доменные
   данные (счета, категории, операции, обязательства) загружаются прямо в
   страницах и фичах.
2. **Одна страница вместо восьми.** `app/router.tsx` направляет `/`, `/operations`,
   `/payments`, `/accounts`, `/categories`, `/analytics`, `/health` и `/profile`
   в один `HomePage`. Он выбирает раздел по `path === …`
   (`pages/home/ui/HomePage.tsx`, 650 строк, 18 вызовов `useQuery`/`useMutation`/API).
3. **Нарушения направления импортов:**
   - page → page: `pages/home` импортирует `pages/statistics/StatisticsPage`;
   - feature → feature: `create-obligation` и `loan-details` импортируют
     `features/schedule-builder/{ui,model,api,testing}`;
   - shared → бизнес: `shared/ui/layout/AuthActions.tsx` вызывает API выхода и
     читает `AuthContext`; `AppLayout` содержит навигацию продукта.
4. **Обход public API.** Слайсы `create-account`, `loan-details`,
   `payment-reminders`, `user-profile`, `obligation-schedule`, `schedule-builder`
   и страницы `imports`, `login`, `statistics`, `subscriptions` не имеют
   `index.ts`. Потребители импортируют внутренние файлы
   (`@/features/user-profile/ui/ProfilePanel`).
5. **Серверное состояние без владельца.** Ключи React Query — литералы в
   компонентах: `['accounts']` встречается в 6 местах, `['cash-position']` — в 4,
   есть разные формы одного ключа (`['financial-events', '', '', '']`).
   `invalidateQueries` вызывается из 14 файлов вручную.
6. **Дублирование транспорта.** `shared/api/client.ts` уже добавляет
   `X-CSRF-Token` ко всем небезопасным запросам. При этом 19 файлов `.tsx`
   импортируют `csrfToken` и передают заголовок вручную.
7. **Правила дат и денег разбросаны.** Часовой пояс `Asia/Yekaterinburg`/`+05:00`
   зашит в 6 файлах страниц. `ReceiptPhoto` и `AIAnalysis` имеют собственные
   форматтеры рублей на `BigInt` рядом с `shared/lib/finance-display`.
8. **Нет проверок архитектуры и форматирования.** ESLint проверяет только hooks
   и react-refresh. Prettier нет, стили смешаны: табы с `;` и пробелы без `;`,
   JSX сжат в одну строку на 1000+ символов (`ReminderHistory.tsx`, `LoanPanel.tsx`).

## Scope

### 1. Целевая структура

Слои и направление зависимостей — только сверху вниз:

    app → pages → widgets → features → entities → shared

| Слой | Ответственность в Grace | Срезы (целевые) |
| --- | --- | --- |
| `app` | Провайдеры, router, layout route с `AuthBoundary` и оболочкой, глобальные стили | — |
| `pages` | Один срез на маршрут. Компоновка виджетов, чтение параметров URL, без собственных запросов к API | `dashboard` (`/`), `operations`, `payments`, `accounts`, `categories`, `analytics`, `health`, `profile`, `subscriptions`, `assistant`, `imports`, `login`, `password-entry` |
| `widgets` | Самостоятельные блоки экрана, которые объединяют сущности и фичи | `app-shell`, `overview`, `operations-list`, `accounts-list`, `obligations-board`, `categories-list`, `financial-health`, `statistics-charts`, `reminder-history`, `import-review` |
| `features` | Пользовательское действие с побочным эффектом: форма, мутация, инвалидация | `create-account`, `reconcile-account`, `create-financial-event`, `edit-operation`, `create-receipt`, `create-obligation`, `edit-loan`, `settle-payment`, `manage-category`, `manage-subscription`, `ask-assistant`, `recognize-receipt-photo`, `logout`, `edit-profile`, `password-auth` |
| `entities` | Доменные объекты из OpenAPI: типы, `queryOptions`, ключи, отображение без действий | `session`, `user`, `account`, `category`, `financial-event`, `receipt`, `obligation`, `payment-schedule`, `subscription`, `import-batch`, `statistics` |
| `shared` | Без знания о домене Grace: API-транспорт, сгенерированные типы, UI kit, утилиты, конфиг | `api`, `ui`, `lib`, `config` |

Список срезов — ориентир. Срез появляется при переносе существующего кода, а не
заранее пустым.

### 2. Правила срезов

1. Каждый срез экспортирует наружу только через `index.ts`. Импорт
   `@/<layer>/<slice>/<segment>/…` из другого среза запрещён.
2. Сегменты внутри среза: `ui`, `model`, `api`, `lib`, `config`. Тесты лежат рядом
   с кодом. Тестовые фейки — в `testing/` и экспортируются отдельным входом
   `index.testing.ts`, не из `index.ts`.
3. Срезы одного слоя не импортируют друг друга. Общий код поднимается в
   вызывающий слой (композиция через props/children) или опускается ниже.
   `ScheduleBuilder` переносится в `entities/payment-schedule` как управляемый
   компонент: им пользуются `create-obligation` и `edit-loan`.
4. Связь сущностей (например, `financial-event` ссылается на `category`)
   оформляется через публичный cross-import `@x` по стандарту FSD 2.1:
   `entities/category/@x/financial-event.ts`. Других исключений нет.
5. Алиас импорта — только `@/`. Относительные импорты разрешены лишь внутри среза.

### 3. Серверное состояние

1. Каждая сущность публикует фабрику запросов в `api/`:

       export const accountQueries = {
         all: () => ['accounts'] as const,
         list: () => queryOptions({ queryKey: [...accountQueries.all(), 'list'], queryFn: … }),
         reconciliations: (id: string) => queryOptions({ … }),
       }

   Компоненты используют `useQuery(accountQueries.list())`. Литералы `queryKey`
   вне `entities/*/api` запрещены.
2. Мутации живут в `features/*/api` как хуки (`useCreateAccount`). Хук сам
   инвалидирует затронутые ключи через `entities/*` `all()`. Компонент формы не
   вызывает `invalidateQueries` напрямую.
3. Ошибки ответа API приводятся к одному типу в `shared/api` (`ApiError` с
   кодом и сообщением). Отображение ошибки и кнопка «Повторить» — общий
   `shared/ui/query-state`.
4. CSRF-заголовок ставит только middleware `shared/api/client.ts`. Хранилище
   токена переезжает в `shared/api/csrf.ts`. `entities/session` владеет запросом
   `/api/auth/me` и `AuthContext`.
5. Правило AGENTS.md сохраняется: DTO берутся из `shared/api/generated`, ручных
   дубликатов нет. `entities/*/model` только переэкспортирует и сужает типы.

### 4. Бизнес-правила и представление

1. Часовой пояс отображения — одна константа `APP_TIME_ZONE` в `shared/config`.
   Даты форматируются через `shared/lib/date`.
2. Форматирование денег (`number` и строковые minor units через `BigInt`) — один
   модуль `shared/lib/money`.
3. Если при переносе в React найдена финансовая арифметика (а не форматирование
   или проверка ввода), она не переписывается в этой спецификации. Она
   фиксируется в отчёте как кандидат на перенос в Go.
   Пример: `CreateReceiptForm` суммирует строки чека для подсказки при вводе. Это
   допустимая проверка формы, итог сверяет сервер.

### 5. Проверки архитектуры

1. Подключить [Steiger](https://github.com/feature-sliced/steiger) с
   `@feature-sliced/steiger-plugin` как `npm run lint:fsd` и включить его в
   `make test-web`.
2. ESLint `no-restricted-imports`/`no-restricted-syntax`: запрет литералов
   `queryKey` вне `entities/*/api`, импорта `csrfToken` вне `shared/api`,
   строки `Asia/Yekaterinburg` вне `shared/config`.
3. Prettier с единым стилем (см. «Открытые решения»): `npm run format:check`
   в `make test-web`. Массовое форматирование — отдельный коммит без других
   изменений, его SHA добавляется в `.git-blame-ignore-revs`.
4. Пока миграция не завершена, существующие нарушения перечисляются в
   baseline-файле Steiger. Каждая фаза только сокращает baseline. В последней
   фазе baseline удаляется.

### 6. Порядок работ

Каждая фаза — отдельная ветка и PR. Поведение и вёрстка не меняются, `make test`
зелёный, тесты меняют только пути импорта и способ рендера провайдеров.

| Фаза | Содержание | Готовность |
| --- | --- | --- |
| 0. Ограждения | Steiger с baseline, ESLint-правила, Prettier и отдельный коммит форматирования, ADR 0003, раздел Web в `architecture.md` | `make test` проверяет FSD и формат; baseline отражает текущее состояние |
| 1. Shared | `APP_TIME_ZONE`, `shared/lib/date`, `shared/lib/money`; удалить ручной CSRF-заголовок; `AppLayout`/`AuthActions` → `widgets/app-shell` + `features/logout`; `auth-*` → `entities/session` | В `shared` нет импортов API-путей продукта и `AuthContext` |
| 2. Entities | Фабрики `queryOptions` для account, category, financial-event, receipt, obligation, subscription, import-batch, statistics, user; замена литералов ключей | Литералов `queryKey` вне `entities/*/api` нет |
| 3. Pages и router | Layout route (`AuthBoundary` + `AppShell` + `<Outlet>`); `HomePage` делится на страницы маршрутов; `StatisticsPage` → `widgets/statistics-charts`; блоки Home → виджеты | Ни одна страница не ветвится по `pathname`; URL прежние |
| 4. Features | `index.ts` у всех срезов; `schedule-builder` → `entities/payment-schedule`; `OperationEditor` → `features/edit-operation`; мутации как хуки с инвалидацией; разбить `ImportsPage` | Нет межсрезовых импортов внутри слоя и глубоких импортов |
| 5. Закрепление | Удалить baseline, Steiger в режиме error, недостающие тесты срезов, итоговый evidence | Все критерии приёмки выполнены |

Фазы 1 и 2 независимы, их можно вести параллельно в разных worktree. Фазы 3 и 4
идут после 2.

## Non-goals

- Изменение UI, текстов, маршрутов, API или OpenAPI.
- Замена React Query, Zustand, react-hook-form, zod, Radix/shadcn или Tailwind.
- Перенос финансовой логики из React в Go. Найденные случаи только
  фиксируются (см. Scope 4.3).
- Code splitting и оптимизация бандла (см. «Открытые решения»).
- Monorepo-пакеты или вынос UI kit в отдельный пакет.

## Предметная модель и инварианты

- Доменная модель и арифметика остаются в Go. Web отображает данные и проверяет
  ввод.
- Деньги на клиенте — minor units с валютой из API. Новых преобразований во
  float для расчётов не появляется.
- Все запросы идут через один `apiClient`: cookie-сессия, CSRF и обработка 401
  сохраняются.

## API и OpenAPI

Без изменений. `npm run generate:api` и проверка CI на расхождение
`schema.d.ts` остаются.

## Данные, миграция и rollback

Миграций БД нет. Каждая фаза — отдельный PR без изменения поведения, откат —
`git revert` его коммита. Production-деплой выполняется обычным release после
проверки владельцем. Отдельного rollout нет.

## UI

Визуальных изменений нет. После фаз 3 и 5 проверяется smoke-чеклист
(`docs/specs/0020-smoke-checklist.md`) на mobile и desktop для всех маршрутов.

## Ошибки и негативные сценарии

- Потеря инвалидации при переносе мутаций: после сохранения данные не
  обновляются. Защита — тесты хуков мутаций проверяют инвалидируемые ключи.
- Изменение ключа запроса ломает существующие тесты, которые подставляют данные
  по ключу. Тесты переходят на те же фабрики.
- Повторный 401 или отсутствие CSRF после удаления ручного заголовка. Защита —
  тест middleware и интеграционный `scripts/test-password-authentication.sh`.

## Критерии приёмки

- [ ] AC1: `npm run lint:fsd` (Steiger) проходит без ошибок, baseline-файла нет.
- [ ] AC2: в `web/src` нет импортов `@/<layer>/<slice>/<segment>` из другого среза;
  каждый срез имеет `index.ts`.
- [ ] AC3: нет импортов page → page, feature → feature, widget → widget и
  entity → entity, кроме `@x`.
- [ ] AC4: `shared` не импортирует ни один верхний слой и не содержит путей
  `/api/…` продукта, кроме транспорта `shared/api`.
- [ ] AC5: литералы `queryKey` встречаются только в `entities/*/api`;
  `invalidateQueries` — только в хуках `features/*/api` и `entities`.
- [ ] AC6: `csrfToken` используется только в `shared/api`; ручного заголовка
  `X-CSRF-Token` в компонентах нет.
- [ ] AC7: `Asia/Yekaterinburg` и `+05:00` встречаются только в `shared/config`.
- [ ] AC8: каждому маршруту `app/router.tsx` соответствует свой срез `pages/*`;
  ни одна страница не ветвится по `pathname`; набор URL совпадает с `e623775`.
- [ ] AC9: ни один файл `web/src` вне `generated` и тестов не длиннее 300 строк.
- [ ] AC10: `npm run format:check` проходит, коммит форматирования указан в
  `.git-blame-ignore-revs`.
- [ ] AC11: `make test` и `make test-integration` проходят. Число тестов web не
  меньше, чем на `e623775`.
- [ ] AC12: smoke-чеклист пройден для всех маршрутов (mobile 375 px и desktop),
  расхождений со снимками `e623775` нет.
- [ ] AC13: `docs/architecture.md` (раздел Web UI) и ADR
  `docs/decisions/0003-web-feature-sliced-design.md` описывают правила слоёв.

## Исполняемые проверки

    cd web && npm run lint:fsd && npm run lint && npm run format:check
    make test
    make test-integration
    # AC5–AC7, AC9
    ! grep -rnE "queryKey: *\[" web/src --include=*.ts* | grep -v "/entities/[^/]*/api/"
    ! grep -rln "csrfToken" web/src --include=*.tsx
    ! grep -rnE "Asia/Yekaterinburg|\+05:00" web/src --include=*.ts* | grep -vE "shared/config|generated|\.test\."
    find web/src -name "*.ts*" ! -path "*generated*" ! -name "*.test.*" | xargs wc -l | awk '$2!="total" && $1>300'

## Открытые решения

Решения 1 и 2 приняты владельцем 2026-10-15: Steiger; Prettier — пробелы, 2 отступа,
без `;`, одинарные кавычки, ширина 100. Решения 3 и 4 остаются вне scope.

1. **Steiger или `eslint-plugin-boundaries`.** Рекомендация — Steiger: это
   официальный линтер FSD, он понимает public API и `@x`. Нужна новая
   devDependency, отдельный этап ESLint её не заменяет.
2. **Стиль Prettier.** Предложение: пробелы, 2 отступа, без `;`, одинарные
   кавычки, ширина 100 — так написано большинство файлов. Массовое
   форматирование меняет почти каждый файл, поэтому оно идёт одним коммитом в
   начале.
3. **Code splitting маршрутов** (`React.lazy` для `analytics` с recharts,
   `imports`, `assistant`). Структура страниц после фазы 3 это позволяет, но это
   изменение производительности, а не архитектуры. Предлагается отдельная
   спецификация.
4. **Часовой пояс пользователя.** Сейчас он зафиксирован как Екатеринбург и на
   сервере, и на клиенте. Перенос в профиль — продуктовое решение вне этой
   спецификации.
