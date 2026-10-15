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
