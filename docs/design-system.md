# Grace: единая система интерфейса

Локально реализовано по accepted [SPEC-0032](specs/0032-ui-redesign.md).
Выбранные петрольные цвета SPEC-0029 сохранены. Production ещё не обновлён.

## Основа

Существующая Radix UI library, общие компоненты web/src/shared/ui и Lucide icons.
Кнопки, inputs, textarea, selects, checkboxes, dialogs и cards не создаются
заново в страницах. ChoiceSelect использует тот же Radix Select, включая
клавиатуру, typeahead, disabled и empty selection. Empty sentinel не уходит в API.
Checkbox — Radix Root/Indicator, boolean callback. PasswordInput — Input +
Button с EyeOff/Eye, accessible label/state, type=button и сохранённым value.

Native date/file picker остаются через Input; SVG data markers графиков
имеют самостоятельное keyboard/tap взаимодействие. Это не отдельные формы
или конкурирующая библиотека controls. Native disclosures используют общие
tokens/states. Не добавлено новых dependencies.

## Визуальные правила

- Размер обычного control44px; длинный select label может увеличить высоту.
  Icon actions44px, large48px; checkbox20px внутри label target44px.
- Radius controls12px, nested8px, cards/dialogs20px.
- Spacing4/8/12/16/20/24/32. Card padding24, mobile20; section gap24.
- Body/inputs16, secondary/labels14, section titles16–18, page heading26–32.
- Surface card, мягкая border, минимальные тени. Primary action петрольный.
  Нет дополнительных градиентов, декоративных шаров или цветов каждой функции.
- Visible focus; enabled pointer, input text, disabled not-allowed. Sidebar
  scrolling, skip link и focus/scroll reset при смене страницы.
- Labels постоянны. Unknown/coverage/confirmation/consent остаются видимыми.
  Приборная карточка показывает реальные значения, не рейтинг здоровья.

## Экранные композиции

Desktop: sidebar со всеми разделами + спокойная верхняя панель. Mobile:
brand/menu сверху, три основных раздела снизу, все разделы в menu.
Overview: balance/payments + operations/accounts, затем широкий график.
Accounts/operations/payments: общие cards/rows и действия по контексту.
Statistics: единые filters/charts/tables; Finance health сохраняет раскрываемые
источники/таблицы. Imports: общее оформление, toolbar filters, подробные условия
подготовки файла по раскрытию. Assistant: общий header вместо большого banner,
consent retained. Profile: две спокойные панели. Auth: единый AuthShell и eye fields.

## Просмотр

После npm ci в web: `node docs/preview-ui.mjs`.
http://127.0.0.1:6224/node_modules/.grace-ui-qa/ui-preview.html

Source fixtures — docs/ui-review; запускаются только локально, скопированы в
ignored node_modules. GET mocked, POST/PATCH/fetch blocked. Данные искусственные,
суммы фиксированы; фильтры в preview показывают взаимодействие, не перерасчёт.
Auth формы: добавить `?screen=/register`, `/login` или `/forgot-password`.
В production эти mocks и fixture entry не включены.

Проверки и ограничения: [0032-evidence.md](specs/0032-evidence.md).
