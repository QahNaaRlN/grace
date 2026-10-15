# SPEC-0048: реализация и проверка

Спецификация: [accepted SPEC-0048](0048-frontend-fsd-architecture.md). Спецификация
реализуется по фазам. Статус остаётся `accepted`, пока не выполнены все критерии
приёмки (фаза 5).

## Фаза 0 — ограждения

Дата: 2026-10-15. Ветка: `codex/frontend-fsd`. Проверено локально на Windows, Node 23.

| Коммит | Содержание |
| --- | --- |
| `3edc372` | Принятие спецификации; Steiger и обёртка `web/scripts/lint-fsd.mjs` с `steiger.baseline.json`; ESLint-правила `web/eslint-rules/grace.js` с `eslint.baseline.js`; конфиг Prettier; ADR 0003; раздел «Структура Web-кода» в `architecture.md` |
| `7a7d946` | Только `prettier --write .` в `web/` (114 файлов) |
| следующий | `.git-blame-ignore-revs`, `format:check` в `make test-web`, этот отчёт |

Ветку нужно интегрировать merge-коммитом, без squash. Иначе SHA `7a7d946`
из `.git-blame-ignore-revs` не попадёт в `main`.

### Что зафиксировано в baseline

- Steiger: 87 диагностик. 68 — `no-public-api-sidestep` (в основном
  `shared/api`, `shared/model`, `shared/config` без `index.ts` и глубокие импорты
  фич), 8 — `public-api` (4 фичи и 4 сегмента `shared` без `index.ts`),
  6 — `no-segmentless-slices`, 4 — `forbidden-imports`
  (`create-obligation` → `schedule-builder`), 1 — `segments-by-purpose`
  (`app/providers`).
- ESLint (по файлам): `grace/no-inline-query-key` — 23,
  `grace/no-direct-invalidate` — 14, `grace/no-time-zone-literal` — 10,
  `no-restricted-imports` (`csrfToken`) — 20.

### Решения по ходу фазы

- **Baseline Steiger — по диагностикам, не через `files`/`off`.** Проверка
  показала, что Steiger исключает «выключенные» файлы из модели проекта для
  правила. Новый файл с глубоким импортом
  `@/features/create-financial-event/ui/CreateFinancialEventForm` не
  обнаруживался, потому что цель импорта была в списке `off`. Обёртка
  `lint-fsd.mjs` запускает Steiger на всём дереве и сверяет каждую диагностику
  (правило, файл, сообщение) со списком.
- `fsd/insignificant-slice` выключено (см. ADR 0003).
- Prettier `endOfLine: auto`: в репозитории `core.autocrlf=true` на Windows, а в CI LF.

### Ограничение, найденное проверкой

Steiger не распознаёт срезы без сегментов (`pages/login`, `pages/imports`,
`pages/statistics`, `pages/subscriptions`, `features/loan-details`,
`features/payment-reminders`). Поэтому импорт page → page и
`loan-details` → `schedule-builder` сейчас не диагностируется. Это закрывается
в фазах 3–4, когда срезы получают сегменты; после этого такие импорты станут
новыми диагностиками и их нужно будет исправить, а не добавлять в baseline.

### Исполняемые свидетельства

| Проверка | Результат |
| --- | --- |
| `make test` | exit 0: Go `ok` во всех пакетах; web — 33 файла и 215 тестов, `lint`, `lint:fsd` (OK, 87 в baseline), `format:check`, `build`; Jev eval `ok` |
| Новое нарушение FSD | Временный `src/pages/assistant/ui/Probe.tsx` с глубоким импортом фичи → `lint:fsd` exit 1, «new Feature-Sliced Design violations». Файл удалён |
| Исправленное нарушение остаётся в baseline | Лишняя запись в `steiger.baseline.json` → `lint:fsd` exit 1, «fixed violations are still listed». Откат |
| ESLint-правила | Временный `EslintProbe.tsx` → 6 ошибок: `no-restricted-imports` (`csrfToken`), `grace/no-inline-query-key` ×2, `grace/no-direct-invalidate`, `grace/no-time-zone-literal` ×2. Файл удалён |
| Форматирование без изменения поведения | После `7a7d946` те же 215 тестов, lint и build проходят |
| `git diff --check` | чисто |

Не выполнено в фазе 0: AC1–AC13 закрываются фазами 1–5. CI GitHub (Node 22,
Steiger требует ≥ 22.18) проверяется после push.

## Фаза 1 — Shared

Дата: 2026-10-15. Ветка: `codex/frontend-fsd-1` от `codex/frontend-fsd` (`57de0c2`).
Проверено локально на Windows, Node 23.

| Коммит | Содержание |
| --- | --- |
| `685b017` | `APP_TIME_ZONE`/`APP_UTC_OFFSET` и `index.ts` в `shared/config`; `shared/lib/date` (функции дат из `finance-display` и замены локальных `Intl`-форматтеров страниц); `shared/lib/money` (`formatMoney`, разбор сумм, BigInt-форматтеры из `AIAnalysis`/`ReceiptPhoto`, дубликат `formatMoney` из `AssistantPage` удалён) |
| `a60a7f1` | Хранилище токена — `shared/api/csrf.ts`; ручной `X-CSRF-Token` удалён из 19 файлов; `apiClient` типизирован через `ClientPaths` без CSRF-заголовка; тест middleware; `shared/api/index.ts`; правило `no-restricted-imports` охраняет `@/shared/api/csrf` |
| `2cfdf03` | `AuthContext` → `entities/session`; `AppLayout` → `widgets/app-shell`; `AuthActions` → `features/logout`; `ProfilePanel` получает его через слот `sessionActions` от `HomePage` |
| следующий | Этот отчёт |

### Решения по ходу фазы

- **CSRF в типе клиента.** OpenAPI объявляет `X-CSRF-Token` обязательным
  параметром мутаций, поэтому после удаления ручного заголовка TypeScript его
  требовал. Контракт не меняется (non-goal): `shared/api/client.ts` строит
  `ClientPaths`, где этот заголовок убран, а middleware ставит его сам.
  `Categories.tsx` раньше читал токен на уровне модуля, то есть всегда пустым;
  заголовок туда и так ставил middleware.
- **`features/logout` и `features/user-profile`.** Профиль показывал
  `AuthActions` сам, что после переноса стало бы импортом feature → feature.
  Теперь страница передаёт `<AuthActions />` в слот `sessionActions`. Вёрстка
  та же. Компонент сохраняет имя `AuthActions` и диалог «Доступ»
  (приглашения, завершение сеансов, выход), как в спецификации.
- **Подсказка в `ImportsPage`.** Подпись «например +05:00» и placeholder
  строятся из `APP_UTC_OFFSET`. Текст на экране тот же; тест
  `ImportsPage.test.tsx` находит поле по прежней подписи.
- **Что осталось на фазу 2.** Запрос `/api/auth/me` пока живёт в
  `AuthBoundary` и `LoginPage` с литералом `['auth-session']`. Перенос в
  `entities/session/api` — вместе с фабриками `queryOptions`.
- **Тестовые фикстуры с `+05:00`.** `AssistantPage.test.tsx`,
  `Accounts.test.tsx`, `ImportsPage.test.tsx` содержат моменты из API и
  остаются в baseline `grace/no-time-zone-literal`. Проверка AC7 исключает
  `.test.`; правило для тестов решается в фазе 5, когда удаляется baseline.

### Исполняемые свидетельства

| Проверка | Результат |
| --- | --- |
| `make test` | exit 0: Go `ok` во всех пакетах; web — 35 файлов и 219 тестов (было 33 и 215 после фазы 0), `lint`, `lint:fsd` (OK, 25 в baseline), `format:check`, `build`; Jev eval `ok` |
| Baseline только уменьшились | `git diff --stat codex/frontend-fsd -- web/eslint.baseline.js web/steiger.baseline.json` → 0 вставок, 94 удаления. ESLint: −27 записей файлов (`no-time-zone-literal` −7, `no-restricted-imports` −20, весь блок). Steiger: 87 → 25 диагностик |
| AC6 (в смысле критерия) | `grep -rnE "import .*\bcsrfToken\b\|X-CSRF-Token" web/src \| grep -v "^web/src/shared/api/"` → пусто |
| AC7 | `grep -rnE "Asia/Yekaterinburg\|\+05:00" web/src --include=*.ts* \| grep -vE "shared/config\|generated\|\.test\."` → пусто |
| Готовность фазы 1 | `grep -rnE "AuthContext\|'/api/" web/src/shared \| grep -v "^web/src/shared/api/"` → пусто: в `shared` нет `AuthContext` и путей API продукта |
| Middleware CSRF | `web/src/shared/api/client.test.ts`: POST получает токен, GET — нет, 401 очищает токен и шлёт `grace:unauthenticated` |
| Деньги | `web/src/shared/lib/money/money.test.ts`: BigInt-формат больших и отрицательных сумм, доллары в micros |
| ESLint-правило CSRF | Временный `EslintProbe.ts` с `import { csrfToken } from '@/shared/api/csrf'` и литералом пояса → `no-restricted-imports` и `grace/no-time-zone-literal`. Файл удалён |
| Steiger-проба | Временный `src/pages/assistant/ui/SteigerProbe.ts` с `import { apiClient } from '@/shared/api/client'` → `lint:fsd` exit 1, `fsd/no-public-api-sidestep`. Файл удалён, `lint:fsd` снова OK |
| `git diff --check` | чисто |

Проверка AC6 из спецификации (`grep -rln "csrfToken" web/src --include=*.tsx`)
в буквальном виде пустой не станет: она находит поле `csrfToken` ответа
`/api/auth/me` (`setCSRFToken(data.csrfToken)`) и фикстуры сессии в тестах.
Предлагаю владельцу уточнить проверку в спецификации до формы выше.

## Фаза 2 — Entities

Дата: 2026-10-15. Ветка: `codex/frontend-fsd-2` от `main` (`2620440`, фазы 0 и 1).
Проверено локально на Windows, Node 23.

| Коммит | Содержание |
| --- | --- |
| `d0e25b2` | `entities/session/api` (`sessionQueries.me`, `loadSession`; запрос `/api/auth/me` убран из `AuthBoundary` и `LoginPage`), `entities/user/api` (`userQueries.profile`, `authMethodQueries.list`), `entities/account/api`; `ProfilePanel`, `PasswordEntry` и тесты на фабриках |
| `bd16de0` | Фабрики `category`, `financial-event`, `receipt`, `obligation` (+ `loanQueries`, `paymentReminderQueries`), `subscription` (+ `subscriptionExpenseQueries`), `import-batch`, `analytics-report` (`statisticsQueries`, `periodSummaryQueries`, `cashPositionQueries`) |
| `8964133` | Фичи инвалидируют через `entities/*` `all()`; `LoanPanel` и `ReminderHistory` читают через фабрики |
| `bc93f78` | `HomePage`, `Accounts`, `Categories`, `Operations`, `OperationEditor` на фабриках |
| `cd07d89` | `ImportsPage`, `ImportHistory`, `RecordReview`, `MonthPanel`, `SubscriptionsPage`, `StatisticsPage`; `ImportsPage.test` на ключах фабрик |
| `bc15273` | Блок `grace/no-inline-query-key` удалён из `eslint.baseline.js` |
| следующий | Этот отчёт |

### Решения по ходу фазы

- **Формы ключей сохранены.** Каждая фабрика строит ключи от своего `all()`,
  и они совпадают с прежними литералами (`['accounts']`, `['obligations', asOf]`,
  `['imports', 'records', batchId, view]` и т. д.). Поэтому прежняя инвалидация по
  префиксу работает как раньше. Изменён один ключ: список событий на главной
  `['financial-events', '', '', '']` стал `['financial-events', 'list']`. По этому
  ключу ничего не подставляется, а инвалидация идёт по префиксу `['financial-events']`.
- **Несколько корней в одном срезе.** Если у ключа был свой корень, он остался:
  `['cash-position']`, `['period-summary']`, `['statistics']` лежат в
  `entities/analytics-report`, `['loan-details']` и `['payment-reminders']` в
  `entities/obligation`, `['subscription-expenses']` в `entities/subscription`,
  `['auth-methods']` в `entities/user`. Каждая фабрика имеет свой `all()`. Если
  перенести их под общий корень, инвалидация `['statistics']`, `['obligations']`
  или `['subscriptions']` стала бы запрашивать лишние данные.
- **`entities/analytics-report` вместо `entities/statistics`.** Steiger
  `fsd/inconsistent-naming` отклоняет срез во множественном числе среди
  единственных. В baseline добавлять нельзя, поэтому срез переименован.
  Спецификация называет список срезов ориентиром.
- **Один `queryFn` на ключ.** Раньше один ключ мог загружаться разными функциями
  с разными запасными текстами ошибок. Теперь у каждого ключа одна функция.
  Затронуты случаи:
  - `['cash-position', through]` на главной и в «Аналитике» использует один
    кэш. Запасной текст теперь всегда «Не удалось загрузить доступность денег»
    (раньше на главной было «Не удалось получить расчёт»). Сообщение сервера
    показывается, как и раньше.
  - `['accounts']` и `['categories']` в `SubscriptionsPage` используют загрузчики
    главной. Подписки эти ошибки не показывают.
  - `['auth-methods']` в `PasswordEntry` и `['auth-session']` в `LoginPage` тоже
    не показывают текст ошибки. `retry: false` и `staleTime`/`refetchOnWindowFocus`
    у `AuthBoundary` переданы поверх фабрики.
- **Инвалидация не тронута.** Прямые `invalidateQueries` остаются в тех же 14
  файлах до хуков мутаций фазы 4. Теперь они берут ключи у фабрик, без литералов.
- **Срезы `entities` друг друга не импортируют**, поэтому `@x` не понадобился.
  `ImportRecordsView` (фильтр строк импорта) объявлен в `entities/import-batch`,
  а `RecordReview` использует его как `View`.

### Исполняемые свидетельства

| Проверка | Результат |
| --- | --- |
| `make test` | exit 0: Go `ok` во всех пакетах; web — 35 файлов и 219 тестов (столько же, сколько после фазы 1), `lint`, `lint:fsd` (OK, 25 в baseline), `format:check`, `build`; Jev eval `ok` |
| AC5, первая часть (готовность фазы 2) | `grep -rnE "queryKey: *\[" web/src --include=*.ts* \| grep -v "/entities/[^/]*/api/"` → пусто |
| Baseline ESLint | `git diff --stat main -- web/eslint.baseline.js web/steiger.baseline.json` → 0 вставок, 28 удалений. `grace/no-inline-query-key`: 23 → 0 файлов (блок удалён); `no-direct-invalidate` 14 → 14; `no-time-zone-literal` 3 → 3 |
| Baseline Steiger | `npm run lint:fsd -- --prune` → OK, 25 → 25. Записей не добавлено и не удалено: нарушения в baseline не связаны с ключами запросов |
| Правило без baseline | ESLint по `src` с одним `grace/no-inline-query-key` (без `eslint.baseline.js`) → ни одного файла |
| Связи entities | `grep -rn "@/entities\|@/features\|@/pages\|@/widgets" web/src/entities` → пусто |
| ESLint-проба | Временный `src/pages/home/ui/EslintProbe.ts` с `useQuery({ queryKey: ['accounts'], … })` → `npx eslint` exit 1, `grace/no-inline-query-key`. Файл удалён |
| `git diff --check` | чисто |

## Фаза 3 — Pages и router

Дата: 2026-10-15. Ветка: `codex/frontend-fsd-3` от `main` (`6f98a0c`, фазы 0–2).
Проверено локально на Windows, Node 23.

| Коммит | Содержание |
| --- | --- |
| `77c3b2e` | Layout route в `app/router.tsx` и `router.test.tsx`. `pages/home` разделён на 8 срезов, блоки главной и `StatisticsPage` перенесены в `widgets`, добавлен `widgets/ledger-screen`. Сегменты `ui/` и `index.ts` у `imports`, `login`, `subscriptions`. Новые `features/password-auth` и `pages/password-entry`. Baseline Steiger сокращён |
| следующий | Этот отчёт |

### Структура после фазы

- **`app/router.tsx`.** Защищённые маршруты — дочерние элементы одного layout
  route без `path`: `AuthBoundary` → `AppLayout` → `<Outlet>`. `/login`,
  `/register`, `/forgot-password` и `/reset-password` лежат вне layout. Страницы
  `assistant`, `imports` и `subscriptions` больше не оборачивают себя в `AppLayout`.
- **`pages/*`: один срез на маршрут (AC8).** `dashboard` (`/`), `operations`,
  `payments`, `accounts`, `categories`, `analytics`, `health`, `profile`,
  `subscriptions`, `assistant`, `imports`, `login`, `password-entry`. Срез
  `password-entry` обслуживает `/register`, `/forgot-password` и `/reset-password`
  через параметр `view`, как раньше. Новые страницы не обращаются к API и не
  читают `pathname`. Они задают заголовок и компонуют виджеты.
- **`widgets/ledger-screen`.** Общий каркас денежных страниц. В нём:
  - шесть запросов главной: счета, категории, события, чеки, обязательства и
    позиция денег;
  - статус в заголовке;
  - кнопки «Добавить счёт» и «Добавить операцию» с диалогами;
  - баннер «Обзор загружен частично»;
  - скелетон и приглашение «Начните с первого счёта».

  Страница задаёт `accountAction` (`always`, `when-empty` или `never`, по
  SPEC-0036), `operationAction` и `gated`. Данные приходят в render-prop
  `children(ledger)`. Поэтому пустой ledger и диалоги создания написаны один раз.
- **Виджеты главной:**
  - `overview`;
  - `operations-list`, вместе с `OperationEditor` до фазы 4;
  - `accounts-list`;
  - `obligations-board` — бывший `Obligations` из `HomePage`;
  - `categories-list`;
  - `financial-health`;
  - `period-analytics` — бывший `Analytics` («Подробные расчёты остатков и
    платежей»);
  - `statistics-charts` — бывший `StatisticsPage`, теперь `StatisticsPanel`:
    `compact` на `/`, полный вид на `/analytics`.

  Импорт page → page `home → statistics` исчез.
- **`features/password-auth`.** `PasswordForm` нужен и на `/login`, и на
  `/register` и соседних страницах. Если оставить его в `pages/login`, появится
  импорт page → page. Поэтому форма опустилась в фичу из списка спецификации.
- **`index.ts` у `features/create-account`, `user-profile`, `obligation-schedule`.**
  Новые страницы и виджеты импортируют эти фичи через public API. Без него
  Steiger выдал бы новые `no-public-api-sidestep` на новых путях.
- **`shared/lib/date`: `browserMonthPeriod` и `browserMonthEnd`.** Это бывшие
  `initialPeriod` и `initialThrough` из `HomePage` без изменений, по локальной
  дате браузера. Ими пользуются `ledger-screen` и `period-analytics`.

### Решения по ходу фазы

- **Поведение layout route.** `AuthBoundary` и `AppLayout` больше не
  монтируются заново при каждом переходе. Следствия:
  - Запрос `/api/auth/me` выполняется при входе в защищённую часть и при
    возврате фокуса окна (`refetchOnWindowFocus`), а не при каждом переходе по
    меню. Защита от 401 сохраняется: клиент API по-прежнему отправляет событие
    `grace:unauthenticated`, кэш очищается, пользователь попадает на `/login`.
  - Мобильное меню после выбора раздела закрывается с той же анимацией, что и
    по крестику. Раньше оно исчезало вместе с размонтированной оболочкой.
- **Скрытая кнопка «Добавить операцию».** На `main` диалог операции рендерился на
  всех страницах `HomePage`. Вне `/` и `/operations` его кнопка имела класс
  `hidden`. Теперь диалог есть только на этих двух страницах. Видимых изменений
  нет.
- **Тесты.** `HomePage.test.tsx` и `DueSoon.test.tsx` разнесены по срезам,
  проверки не изменены:
  - `pages/dashboard` — пустой ledger;
  - `pages/profile` — SPEC-0036;
  - `pages/payments` — SPEC-0046, «скоро»;
  - `widgets/overview` — 3 теста;
  - `features/create-account` — 2 теста;
  - `features/create-financial-event` — 1 тест.

  `AssistantPage.test` теперь рендерит страницу внутри `AppLayout`, потому что
  проверяет меню оболочки. Добавлен `app/router.test.tsx` с 3 тестами.
- **ESLint `no-time-zone-literal`.** Перенесённые `Accounts.test.tsx` и
  `ImportsPage.test.tsx` берут смещение из `APP_UTC_OFFSET`. Записи удалены и
  на новых путях не появились.
- **ESLint `no-direct-invalidate`: решение владельца.** Вместе с кодом
  перенесены 7 файлов с прямыми `invalidateQueries`:
  - `Accounts`;
  - `Categories`;
  - `OperationEditor`;
  - `LedgerScreen` — кнопка «Повторить» бывшего `HomePage`;
  - `ImportsPage`;
  - `MonthPanel`;
  - `SubscriptionsPage`.

  Чтобы их исправить, нужны хуки мутаций в `features/*/api`. Это работа фазы 4,
  где `ImportsPage` ещё и делится. Делать это в фазе 3 неразумно. Поэтому
  записи временно стоят на новых путях отдельным блоком
  `eslint.baseline.js`. Число файлов в правиле не выросло (14 → 14). Владелец
  2026-10-15 в чате: «Keep the moved no-direct-invalidate entries in the separate
  baseline block until phase 4.» Блок удаляется в фазе 4 вместе с хуками мутаций.
- **Финансовая арифметика в React** при переносе не найдена: вычисления не
  менялись. `Obligations` по-прежнему суммирует `remainingMinor` по валютам для
  подписи «Осталось». Это отображение серверных остатков, а не новая арифметика.

### Исполняемые свидетельства

| Проверка | Результат |
| --- | --- |
| `make test` | Exit 0. Go: `ok` во всех пакетах. Web: 40 файлов и 222 теста (было 219), `lint`, `lint:fsd` (OK, 13 записей в baseline), `format:check`, `build`. Jev eval: `ok` |
| Ни одна страница не ветвится по `pathname` | `grep -rn "pathname\|useLocation" web/src/pages` → пусто |
| Нет импортов page → page и widget → widget | `grep -rn "from '@/pages/" web/src/pages` → пусто; `grep -rn "from '@/widgets/" web/src/widgets \| grep -v test` → пусто |
| AC8: прежние URL, у каждого маршрута свой срез | `npx vitest run src/app/router.test.tsx` → 3 passed. Набор из 15 путей совпадает с `e623775`. Есть один layout route без `path` с 11 защищёнными путями, 4 публичных пути лежат вне его. `/categories` рендерится в оболочке после проверки сессии |
| Baseline Steiger | `npm run lint:fsd -- --prune` удалил 12 записей, 25 → 13. Удалены 5 `no-public-api-sidestep` (`HomePage`, `HomePage.test`), 4 `no-segmentless-slices` (`imports`, `login`, `statistics`, `subscriptions`) и 3 `public-api` (`create-account`, `obligation-schedule`, `user-profile`). Новых диагностик нет |
| Baseline ESLint | `git diff --stat main -- web/eslint.baseline.js web/steiger.baseline.json` → 13 вставок, 22 удаления. Путей файлов 17 → 15. `no-time-zone-literal`: 3 → 1. `no-direct-invalidate`: 14 → 14, из них 7 путей перенесены (см. выше) |
| `git diff --check` | Чисто |

### Smoke-чеклист (`0020-smoke-checklist.md`)

Стенд: `docker compose -p grace-fsd3-smoke up postgres migrate api` с
`GRACE_AUTH_MODE=development` и отдельным томом, плюс `npm run dev` из ветки.
Проверка во встроенном браузере: desktop 1280 px и mobile 375×812. После
проверки стенд удалён (`down -v`).

| Пункт | Результат |
| --- | --- |
| 1. Разделы и меню | Все 11 защищённых маршрутов открыты на desktop через боковую навигацию и на 375 px через «Меню». Заголовок страницы и активный пункт совпадают с URL. После перехода фокус на `#main-content`, на телефоне видна нижняя панель. `back` возвращает в предыдущий раздел. Диалог «Добавить операцию» открывается и закрывается по Esc. Ошибок в консоли нет |
| 9. Узкий экран | На 375 px `scrollWidth` = 375 на всех маршрутах, горизонтальной прокрутки нет. На desktop `scrollWidth` не больше ширины окна |
| Публичные маршруты | `/register`, `/forgot-password` и `/reset-password` открываются без оболочки. `/login` с активной сессией ведёт на `/`, как на `main` |
| 2. Пустое пространство | В браузере не проверено: на стенде уже есть счёт. Покрыто тестом `pages/dashboard/ui/DashboardPage.test.tsx` |
| 3–8 и 10 (записи) | Не выполнялись. Чеклист разрешает тестовые записи только после явного подтверждения пользователя, а его не было. Формы не менялись, их тесты перенесены без изменений |
| Сравнение со снимками `e623775` (AC12) | Не выполнялось: это критерий фазы 5 |

## Фаза 4 — Features

Дата: 2026-10-15. Ветка: `codex/frontend-fsd-4` от `codex/frontend-fsd-3` (`ac34c21`).
PR фазы 3 ещё не влит, поэтому PR фазы 4 содержит и её коммиты. Проверено локально
на Windows, Node 23.

| Коммит | Содержание |
| --- | --- |
| `6d44586` | `features/schedule-builder` → `entities/payment-schedule` (с `index.ts` и входом `index.testing.ts` для фейка предпросмотра). `loan-details` → `features/edit-loan`, `payment-reminders` → `widgets/reminder-history`. Хуки мутаций в `create-account`, `create-financial-event`, `create-obligation`, `create-receipt`, `edit-loan`, `obligation-schedule` (оплата по графику) и `user-profile` |
| `696d4fc` | `features/reconcile-account` и `features/edit-account` вынесены из `widgets/accounts-list`, `features/manage-category` — из `widgets/categories-list`. Подписи типов счёта и исходов сверки лежат в `entities/account`, `operationsText` — в `entities/category` |
| `9effb9d` | `OperationEditor` → `features/edit-operation` |
| `8f9113c` | `features/manage-subscription`: пауза, привязка расхода, `SubscriptionForm` |
| `1034bc6` | `ImportsPage` разделён: `widgets/import-review`, `features/upload-import`, `features/review-import-record`, `features/confirm-import`. Тесты всех хуков мутаций. Блок `no-direct-invalidate` удалён из ESLint baseline |
| следующий | Этот отчёт |

### Решения по ходу фазы

- **Единый вид хука мутации.** Хук лежит в `features/<slice>/api` и построен на
  `useMutation`. `mutationFn` возвращает ответ API или бросает `Error` с текстом
  для пользователя, как раньше это делал компонент. `onSuccess` инвалидирует
  ключи через `entities/*` `all()`, причём только при успешном ответе.
  - Если раньше компонент ждал обновления (`await invalidateQueries`), то
    `onSuccess` тоже возвращает promise. Тогда `mutateAsync` завершается после
    обновления, и порядок «сохранено → обновлено → закрыть диалог» не меняется.
  - Если раньше обновление шло в фоне (`void`), хук тоже не ждёт его: так
    устроены `edit-loan` и привязка расхода подписки.
  - Если часть состояния раньше менялась до обновления, хук принимает колбэк и
    вызывает его перед инвалидацией. Так работают `onConfirmed`/`onStale` в
    `confirm-import` и `onVerified` в подтверждении почты.
- **Ввод проверяется внутри мутации.** Формы кредита, черновика графика и
  подписки раньше бросали ошибку проверки в `mutationFn`, и она уходила в
  `onError` (toast). Теперь они передают в хук функцию `prepare`, которая
  проверяет ввод и строит тело запроса. Сообщения и их путь не изменились.
- **Межсрезовых импортов нет.** `ScheduleBuilder` теперь — управляемый
  компонент `entities/payment-schedule`, им пользуются `create-obligation` и
  `edit-loan`. Подписи и форматтеры, которые раньше были общими у виджета и
  диалога, опустились в `entities/*/model`.
- **Чтение импорта.** Кнопки «Повторить» после ошибки загрузки черновика и
  сверки месяца — это не мутации. Они используют `useImportRefresh` из
  `entities/import-batch/api`. Поиск следующей нерешённой строки
  (`nextUnresolvedRecord`) тоже перенесён туда.
- **«Повторить» в `LedgerScreen`.** Раньше кнопка инвалидировала весь кэш
  (`invalidateQueries()` без ключа). Теперь она заново запрашивает шесть
  запросов ledger, о которых сообщает баннер. Остальные активные запросы
  страницы (статистика, список операций) кнопка больше не трогает: у каждого
  из них своя кнопка «Повторить» на месте ошибки. Это единственное изменение
  поведения в фазе. Альтернатива — служебная фича, которая очищает весь кэш,
  но тогда нарушилось бы правило «инвалидация — только в хуке мутации».
- **Мелкие отличия по времени.** Поскольку `mutateAsync` ждёт обновления, в
  нескольких диалогах (`ReconcileDialog`, `EditAccountDialog`, `CardDialog`,
  правка и отмена операции, связь платежа с графиком) кнопка показывает
  «Сохраняем…» до конца обновления, а не только до ответа API. Тексты, итог и
  порядок сообщений не изменились.
- **Код из страниц и виджетов.** Страница `/imports` теперь — только заголовок
  и `widgets/import-review`. Виджет возвращает фрагмент, поэтому DOM страницы
  не изменился. `Accounts.tsx`: 676 → 286 строк, `ImportsPage`: 790 → 14 (виджет
  264, `ReviewForm` 418). `SubscriptionsPage` (323 строки) всё ещё читает
  данные сам; перенос запросов страниц в виджеты и AC9 (≤ 300 строк) остаются
  на фазу 5.
- **Тесты.** У каждого хука мутации есть тест: какие ключи инвалидированы после
  успеха и что при ошибке кэш не трогается (`src/test/query-harness.tsx`).
  `NewDraftCard.test` рендерит карточку внутри `QueryClientProvider`, потому что
  она сохраняет через хук. Проверки тестов не менялись.

### Исполняемые свидетельства

| Проверка | Результат |
| --- | --- |
| `make test` | Exit 0. Go: `ok` во всех пакетах. Web: 55 файлов и 257 тестов (после фазы 3 было 222), `lint`, `lint:fsd` (OK, 2 записи в baseline), `format:check`, `build`. Jev eval: `ok` |
| AC5, вторая часть | `grep -rln invalidateQueries web/src --include=*.ts* \| grep -v "\.test\." \| grep -vE "web/src/(features\|entities)/[^/]+/api/"` → пусто |
| Глубокие импорты (AC2) | `grep -rnoE "from '@/(pages\|widgets\|features\|entities)/[^/']+/[^']+'" web/src \| grep -v "index.testing'"` → пусто |
| `index.ts` у каждого среза | цикл по `web/src/{pages,widgets,features,entities}/*` без `index.ts` → пусто |
| Импорты внутри слоя (AC3) | для каждого среза `grep "from '@/<слой>/"` без собственного имени → пусто |
| Baseline ESLint | `git diff --stat codex/frontend-fsd-3 -- web/eslint.baseline.js` → 26 удалений. `no-direct-invalidate`: 14 → 0 файлов, блоки удалены. Остался один файл в `no-time-zone-literal` (`AssistantPage.test.tsx`, решение в фазе 5) |
| Baseline Steiger | `npm run lint:fsd -- --prune` → 13 → 2. Удалены 4 `forbidden-imports` и 4 `no-public-api-sidestep` (`schedule-builder`), 2 `no-segmentless-slices` и `public-api` у `schedule-builder`. Остались `shared/model` (нет public API) и `app/providers` (имя сегмента) — вне scope фазы |
| Хуки мутаций | `npx vitest run src/features/*/api` → 15 файлов, 35 тестов |
| `git diff --check` | чисто |

Smoke-чеклист для фазы 4 спецификация не требует: он проходится после фаз 3 и 5.
Пункты, где создаются записи, в любом случае требуют подтверждения владельца.
