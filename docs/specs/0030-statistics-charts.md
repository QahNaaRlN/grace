# SPEC-0030: графики статистики

- Status: integrated (объединённый выпуск `d475cca`: CI run 38038836282 и Deploy production run 38039374895 успешны)
- Date: 2026-10-09
- Acceptance owner: владелец Grace
- Acceptance record: «приступить к работе» после предложения первых трёх
  графиков и компактной динамики на обзоре.
- Branch: codex/statistics-charts; base aaddf64 (принятый дизайн SPEC-0029).

## Результат и scope

Статистика: категории покупок (горизонтальные столбцы, доля gross purchases),
динамика покупок, доходы рядом с покупками и платежами по обязательствам.
Выбор календарного периода <=1 года, недели/месяцы, валюта. Таблицы и суммы
без hover, keyboard/tap selection. На обзоре компактная динамика за6месяцев.
Текущие подробные period-summary/cash-position сохраняются.

## Контракт и инварианты

GET /api/v1/analytics/statistics: from/to dates YYYY-MM-DD, to exclusive,
granularity week|month. Календарь Asia/Yekaterinburg, partial edge buckets
с явными from/to. Обработка date-only history сохраняет точность даты.
Один read-only RepeatableRead tenant/RLS snapshot; endpoint за существующим auth.
PeriodSummary + ряды на Go/Postgres. Покупки gross, возвраты отдельные значения;
outflows = purchases + debtPayments. Internal transfers исключены из всех рядов.
Деньги int64 minor units, currencies отдельно, overflow fail closed.
Суммы вне точного диапазона JavaScript integer отклоняются endpoint,
не округляются при отображении; отсутствие данных не рисует фиктивную шкалу.
Категорийные доли в basis points вычисляются Go, без финансовой агрегации React.
Coverage recorded_only: ноль — отсутствие записанных событий, не доказательство
полной истории; неизвестные остатки остаются null. Без процентов сравнения
неполных месяцев и без прогноза/оценки здоровья.

## Текущее поведение / данные / rollback

Analytics — числовые сводки; categories уже есть, временных рядов нет.
Recent operations API ограничен100: ссылка ведёт на последние операции,
не обещает полный drilldown по категории/периоду. Полный drilldown с пагинацией
отдельный этап. Новых записей/миграций нет; rollback убирает endpoint и chart UI.
Прогноз, deployment, AI branch integration и новые runtime deps вне scope.

## Приёмка

- AC1: календарные boundaries, категории/валюты, refunds/debts/transfers,
  empty buckets, int64 overflow проверены Go tests; read-only/RLS DB integration
  при доступном PostgreSQL, без заявления PASS при недоступном engine.
- AC2: API authoritative, generated TypeScript; invalid ranges422, auth unchanged.
- AC3: три графика и overview, currency/period/granularity, empty/loading/error,
  таблицы, keyboard/tap,320/390px, обе темы с едиными chart tokens.
- AC4: make test, git diff --check, evidence по каждому AC и ограничения DB/CI.
