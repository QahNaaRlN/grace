# SPEC-0029: проработка палитры и единое оформление

- Статус: accepted
- Дата: 2026-10-09
- Acceptance owner: владелец Grace
- Acceptance record: прямое поручение начать проработку, предложить другую
  палитру, обновить favicon с прозрачным фоном и обеспечить единообразие
  отступов/курсоров/дизайна. Принят scope ниже; окончательная палитра ещё выбирается.
- Ветка: codex/design-foundations

## Проблема и результат

Индиго не устраивает владельца. Нужны сравнимые визуальные варианты на одних
сценариях, прозрачный favicon и согласованные общие состояния элементов.
Основание: docs/vision.md, docs/architecture.md и текущий UX research.

## Scope

Три палитры в отдельном локальном интерактивном прототипе: петрольный,
тёплый графит/медь, спокойный синий. Светлая/тёмная темы, одинаковые synthetic
overview и receipt-review, общие spacing/type/radius/interaction tokens.
Прототип показывает состояния, а не подключается к данным/платному ИИ.

В приложении: убрать прямоугольный фон favicon, сохранить сердце/пульс,
адаптировать цвет контура под светлую/тёмную тему браузера, обновить cache URL.
Исправить общие размеры кнопок: sm и icon-sm >=44px, icon-lg 48px вместо36px;
нативные disabled button/input сохраняют not-allowed cursor. Сохранить текущую palette до выбора владельца.

## Non-goals

Production deploy, массовая перестройка страниц, выбор окончательной палитры
за владельца, API/БД/денежная логика/изменение consent и confirmation.
Будущий redesign receipt/analysis сначала уточняется в отдельном scope.

## Текущее поведение

Main f257f8e: SVG favicon имеет закрашенный rect; icon-lg 36px, icon40px
в sm при базовом icon44px; disabled:pointer-events-none скрывает cursor
нативных controls. Входные данные неизвестной точности не становятся нулями.

## Инварианты, API и rollback

Представление не меняет финансы или формы. Новых runtime dependencies,
API changes, DTO или миграций нет. SVG без script/внешних ресурсов/шрифтов.
Palette preview — synthetic; никакие кнопки не публикуют расходы.
Rollback возвращает SVG/HTML/shared button/input styles.

## Негативные сценарии

16/32px favicon на обеих темах, длинные деньги и русские labels,
320/390px layouts, keyboard focus, disabled native button/input, reduced motion.

## Критерии приёмки и checks

- AC1: прозрачный SVG без background rect; smoke на 16/32px в обеих темах.
- AC2: три одинаковых варианта overview/receipt-review и обе темы в прототипе;
  пары body/muted/action text contrast >=4.5:1 проверяются расчётом.
- AC3: общая шкала4/8/12/16/24/32; touch44px, icon-lg48px;
  cursor/focus/disabled доступны и безопасны в browser smoke.
- AC4: make test, git diff --check; отчёт по каждому AC, прототип не меняет ledger.

Окончательная palette, rollout и финальный redesign — открытые решения.
