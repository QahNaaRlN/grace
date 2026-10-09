# SPEC-0032: implementation evidence

Accepted [SPEC-0032](0032-ui-redesign.md), branch codex/ui-redesign,
base51c5f83. Все действующие routes переработаны локально, production не менялся.

| AC | Результат | Исполняемое свидетельство |
| --- | --- | --- |
| AC1 | PASS | rg по pages/features не находит native select/checkbox/button/textarea. Все поля используют shared UI; ChoiceSelect и Checkbox — Radix library, PasswordInput использует Button/Input и Eye/EyeOff. controls.test.tsx: empty sentinel возвращает '', выбор USD, checkbox не submit и respects disabled, eye меняет type и сохраняет value/autocomplete, no submit |
| AC2 | PASS local | Browser actual components с mock data: overview, operations, accounts (и create dialog), payments, statistics, profile, assistant, finance health, imports и auth. Sidebar desktop, mobile menu/bottom nav, shared headers. Overview grid без пустого правого верхнего блока; compact chart адаптируется к ширине. Native file button styled через Input; cards имеют h2 titles, page header h1 |
| AC3 | PASS | Existing financial event/account/profile/password/import/statistics tests retained. Import tests выбирают реальные Radix options (account/category/create/link/exclude), сохраняют warm caches, date-only precision, saved decisions и explicit publish behavior. Статистика grouping/currency использует прежние значения. Никаких изменений Go/OpenAPI/migrations. Imports source amount отображается тем же formatMoney только при safe integer и известной3letter currency; иначе evidence fallback |
| AC4 | PASS local | make test: Go, Web58 tests/14 files, lint/build, Jev49/72. git diff --check. Browser measurements desktop1449/document1434; mobile390/document375,320/document305 (15px scrollbar), без overflow страницы. Проверены overview dark, all navigation routes320px, health table раскрыта390px, dropdown statistics320px, account select/checkbox/dialog, auth eye real clicks type=text→password. Визуальные screenshots сохранены, интерактивный preview воспроизводим через node docs/preview-ui.mjs |

Изменения: общий AppLayout/PageHeader/AuthShell; shared controls/tokens;
context actions; profile columns; assistant header; imports filters/file help;
overview grid; responsive SVG geometry. Финансовые/аутентификационные правила
и AI consent не менялись. Сервер/БД/платный inference не использовались для QA.
Unknown balances и точные source values не превращаются в нули.

Первый sandbox test run столкнулся с недоступным временным transform cache;
повтор вне sandbox прошёл. Для Radix browser APIs JSDOM setup имеет no-layout
ResizeObserver/pointer capture/scroll shims; browser smoke проверяет настоящий DOM.
Новые tests — meaningful control interaction, не snapshot CSS.

Итоговый log ignored web/node_modules/ui-release-check.log. Generated schema
получила только line-ending churn от build и возвращена к исходному содержимому;
контракт API не менялся. UI-only: дополнительные DB integrations не требуются.

Ограничения: QA использует искусственные fixed fixtures; новая library controls
проверена в Chromium IAB с responsive viewport, не на физических iOS/Android.
Нет полного assistive-technology certification. Main/remote CI/production пока
не менялись. Новые AI receipt flows отдельной feature branch не интегрированы;
все действующие ручные receipt формы используют новые shared controls.
