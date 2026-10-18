# SPEC-0052: реализация и проверка

Дата: 2026-10-16. Спецификация: [accepted SPEC-0052](0052-route-code-splitting.md).
Ветка: `codex/route-code-splitting` от `main` `4785143`.
Статус: `integrated` (локальные post-merge gates 2026-10-18; исходная
реализация проверялась на Windows).

## Размеры чанков

`npm run build` (Vite 8, rolldown), одинаковые зависимости.

| Чанк | `main` `4785143` | Эта ветка |
|---|---|---|
| основной `index-*.js` | 900,79 кБ (gzip 263,67) | 389,84 кБ (gzip 122,68) |
| `button-*.js` (общие компоненты `shared/ui`) | — | 121,08 кБ (gzip 36,64) |
| `skeleton-*.js` | — | 64,16 кБ (gzip 22,24) |
| `imports` | — | 48,45 кБ |
| `assistant` | — | 40,32 кБ |
| `payments` | — | 36,60 кБ |
| `zod` | — | 34,11 кБ |
| `operations` | — | 27,03 кБ |
| остальные 36 чанков | — | от 0,19 до 16,05 кБ |
| всего JS | 900,79 кБ, 1 файл | 917,76 кБ, 45 файлов |
| CSS | 57,28 кБ | 57,28 кБ |
| «Some chunks are larger than 500 kB» | есть | нет |

Суммарный JS вырос на 17 кБ (обвязка динамических импортов и общих чанков),
но при открытии «Обзора» загружается 17 файлов вместо всего приложения;
страницы `imports`, `assistant`, `payments`, `analytics` и остальные
загружаются только при переходе на них.

## Критерии приёмки

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `app/router.tsx`: все страницы через `lazy` | Таблица выше; `npm run build` без предупреждения о 500 кБ (в составе `make test`) |
| AC2 | Статических `import … from '@/pages/*'` в `router.tsx` нет; у каждого листового маршрута `lazy`, без `element`/`Component` | `router.test.tsx` «loads every page through route lazy, not a static import» |
| AC3 | URL и layout route прежние | `router.test.tsx`: «keeps the URL set of e623775», «puts every protected page under one layout route…» — без изменений ожиданий |
| AC4 | `/register`, `/forgot-password`, `/reset-password` — один чанк `password-entry` с прежним `view` | `router.test.tsx`: «renders a protected page inside the shell…» (`/categories`) и «renders a sign-in page with its view» (`/register`) |
| AC5 | `app/lazyPage.ts`: при ошибке `import()` одна отметка в `sessionStorage` и `location.assign` на целевой адрес навигации; повторная ошибка или недоступный `sessionStorage` — ошибка уходит в React Router; успешная загрузка снимает отметку | `lazyPage.test.ts` — 4 теста; ручная проверка ниже |
| AC6 | — | Ручная проверка ниже |
| AC7 | — | `make test`: exit 0 — Go `ok`, web 76 файлов / 300 тестов, ESLint, `steiger --fail-on-warnings`, Prettier, сборка без предупреждения, Jev offline checks; `git diff --check` — чисто |

## Ручная проверка (AC5, AC6)

Сборка `vite build` с `VITE_API_BASE_URL=http://localhost:18080`, `vite preview`
на 4173, API и PostgreSQL из `compose.yaml` в отдельном проекте
`grace-0052` (режим `GRACE_AUTH_MODE=development`), встроенный браузер.

- Все 11 пунктов навигации пройдены дважды: заголовок каждой страницы верный
  («Операции», «Платежи», …, «Обзор»); чанк каждой страницы запрошен ровно
  один раз, повторных загрузок нет; ошибок в консоли и `role=alert` нет.
- `/register`, `/forgot-password`, `/reset-password` показывают «Регистрация в
  Grace», «Восстановление пароля», «Новый пароль». `/login` при активной
  development-сессии уводит на «Обзор» — прежнее поведение `widgets/sign-in`.
- Устаревший чанк: файл `password-entry-*.js` убран из `dist`, переход на
  `/register` из открытой вкладки. Страница один раз перезагрузилась на
  `/register` (отметка `grace:page-chunk-reload` = `1`), чанк по-прежнему 404,
  показан стандартный экран React Router, цикла нет. После возврата файла
  `/register` открылся, отметка снята.

Не проверено: выход и вход паролем — в development-режиме API сессия
владельца выдаётся без входа. Покрыто тем, что страницы входа — те же
компоненты, загружаемые через тот же `lazyPage`.

## После слияния `main` `28259e2` (SPEC-0051, кредитные карты)

Конфликт только в `docs/specs/README.md` (обе спецификации добавили запись в
начало), разрешён сохранением обеих. `make test`: exit 0 — web 81 файл /
313 тестов, сборка без предупреждения о 500 кБ; основной `index-*.js` —
389,99 кБ (gzip 122,74), код карт загружается вместе со страницами
«Счета и карты» и «Обзор».

## Production

Release `95c5fa8` выпущен 2026-10-17: Deploy production run 39034114035 —
success, все шаги, включая проверенный pre-deploy backup, миграции
(000024 и 000025 — в составе SPEC-0051 и SPEC-0053), внутреннюю проверку
готовности и внешний authenticated smoke. После выпуска
`https://graceapp.ru/health` — 200, API без сессии — 401. Проверка интерфейса
владельцем после выпуска ещё не выполнена.


## Post-merge verification — 2026-10-18

Проверяемый `main`: `a89a855a8074e4bd0fc957f72b2cb6ab807624fc`.
Интеграция: PR #56, merge `1879889`.
Implementation достижим из `main`; код продукта при проверке не менялся.

- `make test` — exit 0: Go, 327 Web-тестов, ESLint, Steiger, Prettier,
  OpenAPI generation, TypeScript/Vite build и Jev datasets.
- `make test-integration` — exit 0: все 14 наборов.
- `make test-production` — exit 0.
- `go vet ./cmd/... ./internal/...` и `git diff --check` — exit 0.

Локальные post-merge gates закрыты, статус `integrated`. Это дополнение
не заменяет исторические production- и браузерные проверки выше и не
объявляет новую live проверку VPS или интерфейса владельцем.

Основной JS chunk текущей сборки — 390.04 kB (gzip 122.75 kB), предупреждения
о чанке больше 500 kB нет. Router/lazyPage tests входят в 327 Web-тестов.
Ручной обход маршрутов и сценарий устаревшего чанка остаются evidence
предыдущего раздела; повторный browser smoke здесь не заявляется.
