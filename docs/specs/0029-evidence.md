# SPEC-0029: evidence

Дата: 2026-10-09. Accepted [SPEC-0029](0029-design-foundations.md),
ветка codex/design-foundations от main f257f8e. Первичная проработка сохранена
в 2d2f43e. Затем владелец выбрал петрольный вариант и поручил заменить знак.
Общие tokens приложения обновлены в обеих темах; production не развёртывался.

| AC | Результат | Исполняемое свидетельство |
| --- | --- | --- |
| AC1 | PASS | web/public/favicon.svg: прозрачная монета, circle + два штриха, без rect/background, fill=none; 16/32/64px browser preview на checkerboard. Light stroke #0f6b63, dark #88d4c4. HTML cache version grace-petrol-coin-3 |
| AC2 | PASS, петрольный выбран | node docs/preview-design.mjs; три варианта и обе темы сохранены. CSS приложения использует выбранные tokens. GraceMark и favicon имеют одинаковые circle/path. rg по web/src не находит HeartPulse, старые indigo hex или fixed emerald classes |
| AC3 | PASS в scope | Real React smoke: Button default/sm44px, icon-sm44×44, icon-lg48×48, disabled native button/input cursor not-allowed, enabled button pointer, input text; Tab показывает focus ring. Прототип320/390px без horizontal overflow, сумма abc блокирует demo confirm |
| AC4 | PASS local | make test (Go, 43 Web tests/10 files, lint/build, offline Jev gates); git diff --check. API generation не меняет содержимое schema, backend/ledger не менялись |

Contrast считается формулой WCAG relative luminance для каждой из шести themes,
пары text/surface, muted/surface, muted/bg, muted/accent, on-primary/primary,
on-accent/accent. Минимумы (без округления при сравнении с4.5):

| Направление | Light minimum | Dark minimum |
| --- | --- | --- |
| Петрольный | 5.28:1 | 5.63:1 |
| Графит/медь | 4.93:1 | 5.81:1 |
| Синий | 5.06:1 | 5.56:1 |

Все пары выше4.5, но это не полный accessibility certification: keyboard/forms,
assistive technology, границы controls и все production pages требуют проверки
при принятии окончательной палитры. [WCAG contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html).

JS прототипа syntax checked via new Function; браузерная проверка:
palette/theme switch, receipt screen, account selection, amount validation,
confirmation-demo message без записи данных. Browser measurements:
390px viewport/375px document;320px viewport/305px document (scrollbar занимает15px).
Существующие UI states/анимации не перестраивались; prefers-reduced-motion
правило в prototype отключает transitions/animations. Реальные shared controls
сохраняют native disabled behavior; QA scaffold хранится только в ignored
web/node_modules/.grace-design-qa и не включается в приложение.

Прототип доступен локально http://127.0.0.1:6219 после запуска
`node docs/preview-design.mjs` (только loopback; явные routes / и /favicon.svg;
connect-src none). Не требует login/API/ключей/платного ИИ. Фото-чек показан
как synthetic review composition; это не завершённый реальный camera/upload flow.

## Проверка принятого выбора и денежного знака

`make test` повторно PASS: Go, 43 Web tests/10 files, lint/build, offline Jev49/72.
Первый sandbox-run не смог обратиться к локальным тестовым HTTP-серверам;
повтор с доступом к localhost прошёл полностью. `git diff --check` PASS.
Generated schema имеет только line-ending churn, возвращена к исходному содержимому.

Real React smoke через временный ignored scaffold: фон light rgb(244,247,246),
primary rgb(15,107,99), foreground white; dark фон rgb(16,28,27), primary
rgb(136,212,196), foreground rgb(16,43,37). Enabled button pointer,
disabled not-allowed. GraceMark использует currentColor и ту же геометрию,
что favicon; shared размеры/отступы не менялись при применении palette.
В assistant фиксированные emerald decorations заменены semantic tokens,
чтобы foreground кнопки и баннера сохраняли контраст в dark mode.

Контраст посчитан из реального `web/src/app/styles/index.css` через Node
по WCAG luminance для foreground/background, card-foreground/card,
muted-foreground/{card,muted,accent}, primary-foreground/primary,
secondary-foreground/secondary, accent-foreground/accent,
destructive-foreground/destructive, success/card, primary/card.
Минимум light5.28, dark5.63; input border/card light3.52, dark4.41 (>3).
SVG не содержит внешних ресурсов, script, фоновых shapes или шрифтов.

Осталось: принять полный scope receipt/analysis screens, проверить реальные
сценарии после интеграции и выполнить отдельный rollout.
