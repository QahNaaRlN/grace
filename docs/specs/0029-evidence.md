# SPEC-0029: evidence

Дата: 2026-10-09. Accepted [SPEC-0029](0029-design-foundations.md),
ветка codex/design-foundations от main f257f8e. Палитры представлены для выбора;
основная palette приложения не заменена, production не развёртывался.

| AC | Результат | Исполняемое свидетельство |
| --- | --- | --- |
| AC1 | PASS | web/public/favicon.svg без rect/background, fill=none; 16/32/64px browser preview на checkerboard; style media query задаёт светлый/тёмный stroke. HTML cache version grace-transparent-2 |
| AC2 | PASS, выбор цвета pending | node docs/preview-design.mjs; browser clicks переключают три варианта и обе темы. Overview/receipt compositions одинаковые, данные искусственные |
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

Осталось: выбрать palette, принять полный scope receipt/analysis screens,
применить tokens ко всем pages, проверить реальные сценарии и отдельный rollout.
