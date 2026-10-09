# SPEC-0032: единый полный интерфейс Grace

- Status: accepted
- Date: 2026-10-09
- Owner: владелец Grace
- Acceptance record: «Займись полной переработкой ... лаконичный, приятный
  дизайн ... иконку глаза ... единая система компонентов ... библиотека обязательна».
- Branch: codex/ui-redesign; base51c5f83 (SPEC-0029/0030/0031).

## Scope / результат

Все действующие экраны: обзор, операции, счета, платежи, статистика, финансовое
здоровье, импорт/сверка, ассистент, профиль/доступ и auth формы.
Петрольная light/dark palette сохраняется. Единые spacing/type/radius/surface
tokens, лёгкие cards, спокойная иерархия. Desktop navigation sidebar,
mobile bottom navigation + menu; все sections доступны. Общие page headers,
actions по контексту, стабильные empty/loading/error состояния.
Все selects/checkboxes/buttons/inputs используют shared UI на существующей
Radix UI library; не добавлять ещё одну библиотеку. Native file/date controls
сохраняются через Input. Password eye toggle внутри поля, Eye/EyeOff icons,
type=button, aria-label/state и сохранённые autofill/validation.

## Инварианты / non-goals

Presentation only. Не менять Go, API, DTO, money, confirmation, auth policies,
invitation binding, AI consent, batch review/reconciliation и financial isolation.
Платёжные/публикующие действия остаются явно подтверждаемыми. Нет deploy.
Не скрывать unknown/coverage/source evidence, не выдумывать balances/operations.
Перестройка не означает новые финансовые сценарии или полный category drilldown.

## Миграция / rollback / негативные сценарии

Без data migrations/deps/API. Rollback возвращает UI commit.
320/390px, long Russian labels/money, dark/light, keyboard, disabled,
empty selects, required fields, checkbox toggles, errors, dialogs with keyboard.
Radix select использует empty sentinel только внутри UI; API получает прежнее ''.

## Приёмка

- AC1: shared Radix selects/checkboxes и password input; нет локальных native
  selects/checkboxes/custom buttons за пределами shared UI. Eye changes type,
  не submit и не теряет value, accessible label reflects state.
- AC2: все screens используют единый shell/styles/headers/controls; Desktop
  navigation/mobile menu/bottom bar, читаемые tables/forms, actions доступны.
- AC3: сохранены прежние financial/auth/import interactions; tests covers
  confirmations and control changes, keyboard/disabled/empty selection.
- AC4: make test, git diff --check, browser visual QA по screens, both themes,
 320/390px и report с evidence/ограничениями. Новые DB checks не требуются,
 поскольку backend/contract не меняются.
