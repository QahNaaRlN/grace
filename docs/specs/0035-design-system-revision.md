# SPEC-0035: ревизия дизайн-системы — состояния, статусы, суммы, поля

- Статус: integrated
- Автор: Claude Code по поручению владельца
- Дата: 2026-10-13
- Acceptance owner: владелец Grace
- Acceptance record: владелец 2026-10-13 в чате: «Принимаю все три спеки, начинай реализацию»
- Implementation branch: codex/design-system-revision
- Implementation commit: 4dc5c2f..c6691d7 ([evidence](0035-evidence.md))
- Основа: ревью дизайн-системы 2026-10-13; живое сравнение — тема
  «Light/Dark (proposed)» в артефакте Grace Design System
  (https://claude.ai/artifact/3ZBvYBywqV5npnG36BwcXu, раздел «Предлагаемая ревизия»).

## Проблема и пользовательский результат

Система SPEC-0029/0032 спокойная и доступная, но несколько правил размыты:
наведение и текущий пункт выглядят одинаково, фокус слабый, нет статуса
«внимание», серии графиков различаются только оттенком, суммы, поля и
уведомления каждый экран собирает вручную. Результат: те же петрольные
light/dark, но состояния, статусы и суммы читаются однозначно и единообразно.

## Scope

1. Токены (`web/src/app/styles/index.css`, обе темы):
   - `accent` = только hover (значение `muted`); новый `selected` +
     `selected-foreground` для текущего пункта, вкладки, выбранной строки;
   - пары статусов `success|warning|info|destructive|unknown` + `-subtle`;
   - фон `#eef3f1` / `#0e1918`, `border` на шаг сильнее и без 70% на cards;
   - `chart-1..5` с чередованием светлоты (соседние ≥ 1.5:1, каждая ≥ 3:1 к `card`).
2. Фокус: сплошной outline `ring` 2px с offset 2px для всех focusable;
   halo-кольца `ring/30–50` удаляются.
3. Нажатие: только `scale(.98)`; `active:translate-y-px` удаляется.
4. Button: размеры `default | lg | icon | icon-lg`; `xs | sm | icon-xs | icon-sm`
   удаляются с заменой мест использования. Новый вариант `destructive-solid`
   для финального подтверждения необратимого действия.
5. Шрифтовая шкала: `amount 40/48 (32/40 <640px)`, `title-1 28/36`,
   `title-2 18/26 600`, `body 16/24`, `secondary 14/20`, `caption 12/16`;
   `CardTitle` = `title-2`. Произвольные `text-[…]`/`clamp()` удаляются.
6. Новые shared компоненты: `Money`, `Field`, `Notice`, `StatusBadge`
   (контракты — в артефакте, `components/index.d.ts`).
7. Гигиена `index.css`: убрать `.grace-page { animation: none }`, отменяющий
   fade-in страниц, и повторное объявление `.grace-row`.
8. `docs/design-system.md` ссылается на артефакт как на источник значений.

## Non-goals

- Не меняются Go, API, DTO, деньги, политика подтверждений, auth, AI consent.
- Не меняется бренд: `primary`, тёмная палитра как отдельная, радиусы,
  spacing, 44px-цели, motion durations, тон текста.
- Свой шрифт (например, Golos UI) — отдельное решение, не здесь.
- `success`/`destructive` по светлоте не разводятся: различие — иконка и слово.

## Текущее поведение и доказательства

- `index.css`: `--secondary` и `--accent` = `#e4f1ed` / `#263f37`;
  `AppLayout` navClass: active `bg-accent text-primary`, hover `hover:bg-accent`.
- `button.tsx`: `xs`, `sm`, `default`, `icon`, `icon-xs`, `icon-sm` — все `h-11`/`size-11`.
- Фокус: `focus-visible:ring-3 ring-ring/30|40|50`; ореол на белом ≈ 1.8:1.
- `index.css` (`.grace-page { animation: none; }`) идёт после `@import './motion.css'`
  и отменяет `grace-fade-in`.
- Серии графиков в light: контраст к белому 6.1–7.4:1, между соседями 1.02–1.19:1.
- Статусы: только `success`/`destructive`; «внимание» выражается красным или текстом.

## Предметная модель и инварианты

- Presentation only. `Money` форматирует integer minor units через
  `Intl.NumberFormat('ru-RU')`; арифметики не выполняет; `null` → «Не сверён».
- Статус всегда передаётся словом и иконкой; цвет только усиливает.
- Каждый текстовый токен ≥ 4.5:1 на своих фонах в обеих темах; `input`, `ring`,
  серии графиков ≥ 3:1 к `card`.

## API и OpenAPI

Без изменений.

## Данные, миграция и rollback

Без миграций и зависимостей. Rollback — revert UI commit.

## UI

Экраны не перестраиваются. Меняются: навигация (selected), все focus/press
состояния, заголовки карточек, суммы через `Money`, формы через `Field`,
coverage/consent/ошибки модели через `Notice`, статусы платежей через `StatusBadge`,
финальные подтверждения удаления через `destructive-solid`, палитра Statistics.

## Ошибки и негативные сценарии

320/390px, длинные русские подписи и суммы, dark/light, клавиатура (Tab,
Shift+Tab, Enter/Space), reduced motion, disabled, invalid fields, пустые
значения сумм, 5 серий графика при нарушении цветового зрения (симуляция).

## Критерии приёмки

- [ ] AC1: hover и текущий пункт навигации различимы (fill ≥ 1.15:1 между
  собой + вес 600) в обеих темах.
- [ ] AC2: каждый focusable показывает сплошной outline 2px `ring` с offset 2px;
  нет halo-колец.
- [ ] AC3: Button принимает только `default|lg|icon|icon-lg`;
  `destructive-solid` используется в финальных подтверждениях удаления.
- [ ] AC4: токены статусов и `selected` есть в обеих темах; таблица контрастов
  в evidence: все пары ≥ порогов.
- [ ] AC5: `Money`, `Field`, `Notice`, `StatusBadge` в `shared/ui` с unit-тестами;
  суммы в обзоре, операциях, платежах и счетах рендерятся через `Money`.
- [ ] AC6: соседние серии графиков ≥ 1.5:1 по светлоте; легенда/подписи сохраняются.
- [ ] AC7: страницы снова появляются с fade-in; reduced motion его отключает.
- [ ] AC8: `text-[…]` и `clamp()` для типографики отсутствуют в `web/src`
  (кроме `Money size="display"`).

## Исполняемые проверки

    make test
    git diff --check
    cd web && npm run lint && npm test
    node docs/preview-ui.mjs   # ручная проверка 320/390/1280, light/dark

Плюс скрипт контрастов над `index.css`, результат — в evidence.

## Открытые решения

Решения по открытым вопросам (по предложенным в спецификации вариантам, владелец не выбирал иное): 1 — целиком, отдельными коммитами по этапам; 2 — background light `#eef3f1`; 3 — `warning` используется для статуса «скоро платёж» в UI; почтовые напоминания SPEC-0034 не меняются.

1. Принимать целиком или поэтапно (гигиена → статусы/selected → Money/Field →
   шкала/поверхности → графики).
2. `background` light: `#eef3f1` (предложено) или оставить `#f4f7f6` и
   усиливать только `border`.
3. Использовать ли `warning` в напоминаниях о платежах SPEC-0034 сразу.
