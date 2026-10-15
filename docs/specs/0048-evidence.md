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
