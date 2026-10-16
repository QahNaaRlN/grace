# SPEC-0035: реализация и проверка

Дата: 2026-10-13. Спецификация: [accepted SPEC-0035](0035-design-system-revision.md).
Ветка: `codex/design-system-revision`, база `474222f` (SPEC-0036 и SPEC-0037).
Implementation commits по этапам: `4dc5c2f` (фокус, нажатие, размеры кнопок,
гигиена), `38fcdd4` (токены), `d1e4f6c` (Money/Field/Notice/StatusBadge),
`a7969e1` (шрифтовая шкала, правила), `c6691d7` (устойчивость статусов).
Статус: `integrated` — PR #22, merge `ed641ac`; post-merge `make test` в основной копии и CI `main` на `517e621` пройдены. Production не развёрнут (деплой вручную).

Обновление 2026-10-16 (сверка roadmap): release `8ad9705`, содержащий merge `ed641ac`, развёрнут 2026-10-14 (Deploy production run 38560119989).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `accent` = `muted` (только hover), новый `selected`; текущий раздел навигации `bg-selected text-selected-foreground font-semibold` | `tokens.test.ts`: `selected`/`accent` ≥ 1.15:1 в обеих темах. Ручная проверка mock-превью (`docs/preview-ui.mjs`) light/dark |
| AC2 | Один глобальный `:focus-visible` — сплошной 2px `ring`, offset 2px (вне `@layer`, сильнее `outline-none`); halo-кольца удалены из всех shared компонентов, навигации и статистики | `design-rules.test.ts`: нет `focus-visible:ring`/`outline-*`/`ring-[3px]` в `web/src`; `tokens.test.ts`: `ring` ≥ 3:1 к `card` и `background`; ручная проверка Tab в превью SPEC-0035 артефакта |
| AC3 | Button: `default/lg/icon/icon-lg`; варианты + `destructive-solid`; `translate-y-px` удалён | `design-revision.test.tsx` (класс `destructive-solid`, нет `translate`/halo); `design-rules.test.ts`: нет `<Button size="xs|sm|icon-xs|icon-sm">` (проверка самого правила включена) |
| AC4 | `index.css`: `selected`, `success/warning/info/destructive/unknown` + `-subtle` в обеих темах | `tokens.test.ts`: 66 проверок контраста по самому `index.css` — 23 текстовые пары ≥ 4.5:1 и 8 графических ≥ 3:1 в light и dark |
| AC5 | `shared/ui/money.tsx`, `field.tsx`, `status.tsx`; суммы Обзора, операций, платежей, счетов, финансового здоровья, подписок, ассистента — через `Money`; статусы платежей и счётов — `StatusBadge`; покрытие, сбой модели, похожие покупки — `Notice`; форма обязательства — `Field` | `design-revision.test.tsx` (6 тестов: формат, знак, «Не сверён», aria-связи Field, alert/status, fallback тона); существующие тесты страниц (`HomePage`, `FinancialHealth`, `AssistantPage`) проходят с новыми компонентами |
| AC6 | `chart-1..5` с чередованием светлоты; графики Statistics уже берут `var(--chart-n)`, легенда и таблица значений сохранены | `tokens.test.ts`: соседние серии ≥ 1.5:1, каждая ≥ 3:1 к `card`, обе темы |
| AC7 | Удалён `.grace-page { animation: none }`; fade-in из `motion.css` снова действует, reduced motion его отключает (правило не менялось) | Просмотр стилей; `design-rules`-проверка не нужна — правило удалено целиком |
| AC8 | Шкала в `@theme`: `text-amount`, `text-amount-sm`, `text-title-1`, `text-title-2`; h1/h2, PageHeader, AuthShell, CardTitle на шкале | `design-rules.test.ts`: нет `text-[…]` и `clamp()` в `web/src` |

Общие gates в worktree: `make test` — Go (все пакеты `ok`), web 25 файлов /
162 теста, lint, build, офлайн Jev; `git diff --check` — чисто.

Отклонения и решения при реализации:

- `selected` в светлой теме — `#cadedd` (в артефакте было `#cfe1e0`): у
  предложенного значения разница с hover 1.148:1, ниже порога AC1.
- Решение 3 («`warning` для статуса „скоро платёж“»): API даёт только
  `overdue`/`upcoming`; вычислять «скоро» в React — бизнес-правило в UI.
  Сейчас `warning` обозначает «частично оплачен»; «скоро платёж» требует
  отдельного значения `timing` от Go — предлагается отдельной спецификацией.
- Панели-акценты (герой Обзора, блок «Хватит ли денег», заметка ассистента,
  аватар, кнопка файла) переведены с `accent` на `secondary` — прежний мятный цвет.
- Wordmark «Grace» оставлен `text-xl` (бренд, не шкала текста).
- Mock-превью UI нашло падение на статусе `pending`, которого нет в API:
  фикстура исправлена, `StatusBadge`/`Notice` теперь показывают неизвестный тон.

Не проверено: живые устройства (iOS/Android), высокий контраст ОС.
Обнаружено попутно (вне scope): страница «Платежи» рисует все строки графика
обязательства — для графика на 120+ платежей нужен такой же свёрнутый вид,
как в `ScheduleBuilder`.
