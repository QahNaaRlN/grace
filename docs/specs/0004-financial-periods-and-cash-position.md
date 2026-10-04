# SPEC-0004: денежные периоды и доступность денег

- Статус: verified
- Автор: Codex
- Дата: 2026-10-04
- Acceptance owner: owner
- Acceptance record: явное согласие owner в Codex-чате, 2026-10-04
- Implementation branch: codex/financial-period-summary
- Implementation commit: `841edbd`

## Проблема и пользовательский результат

Grace хранит финансовые факты, чеки и графики обязательств, но пока показывает
их только отдельными списками. Пользователь не может получить проверяемый ответ
на два разных вопроса:

1. Что произошло с деньгами за выбранный период?
2. Сколько ликвидных денег останется после уже известных обязательств до
   выбранной даты?

Эти вопросы нельзя смешивать. Первый относится к фактическому денежному потоку
между двумя точными временными границами. Второй относится к текущим остаткам и
непогашенному графику до календарного горизонта.

Grace должен вычислять обе сводки в Go/PostgreSQL по подтверждённым данным.
Результат не является советом или прогнозом модели.

## Scope

- добавить read-only модуль `analytics` без собственных финансовых записей;
- ввести фактическую сводку за явный полуоткрытый интервал `[from, to)`;
- агрегировать данные отдельно по каждой валюте, без скрытой конвертации;
- различать income, purchase, refund, debt payment и internal transfer;
- показать категории покупок с возвратами и net expense;
- вычислять математический остаток на временных границах из opening balance и
  проводок;
- ввести текущую ликвидную позицию и непогашенные обязательства до включительной
  календарной даты `through`;
- вычислять `availableAfterObligations` только из debit/cash счетов;
- добавить Go API, OpenAPI и generated TypeScript;
- добавить в React выбор периода, фактическую сводку и горизонт обязательств;
- добавить PostgreSQL-интеграционные проверки без AI и внешних сервисов.

## Non-goals

- бюджеты, лимиты по категориям и budget envelopes;
- рекомендации, оценки «хорошо/плохо», AI/LLM и natural-language answers;
- прогноз будущих доходов или расходов;
- recurring bills, подписки и автоматическое создание обязательств;
- кредитные карты, минимальные платежи и расчётные периоды;
- валютные курсы, пересчёт или суммирование разных валют;
- бухгалтерский P&L, accrual accounting и налоговая отчётность;
- изменение FinancialEvent, Posting, Obligation или Settlement;
- сохранённые пользовательские периоды и настройки часового пояса;
- materialized views, background jobs и отдельное analytics-хранилище;
- графики трендов и сравнение нескольких периодов.

## Текущее поведение и доказательства

- Account хранит opening balance, а текущий balance вычисляется из всех
  проводок;
- FinancialEvent различает `purchase`, `income_received`,
  `internal_transfer`, `refund` и `debt_payment`;
- refund ссылается на purchase и наследует его категорию;
- internal transfer имеет debit и credit одной валюты и не является расходом;
- debt payment имеет отдельный kind и не должен повторно считаться purchase;
- Obligation и ScheduledPayment являются планом и не создают проводок;
- Settlement связывает существующий debt payment с графиком, не меняя баланс;
- API не предоставляет периодных агрегатов или сводки обязательств по валюте.

## Предметная модель и инварианты

`PeriodSummary` является вычисляемым представлением:

- `from` — включительная RFC 3339 граница;
- `to` — исключительная RFC 3339 граница;
- `currencies[]` — независимые `CurrencyPeriodSummary`.

`CurrencyPeriodSummary`:

- currency;
- opening_balance_minor;
- closing_balance_minor;
- income_minor;
- purchases_minor;
- refunds_minor;
- debt_payments_minor;
- net_external_cash_flow_minor;
- internal_transfer_volume_minor;
- expense_categories[].

`ExpenseCategorySummary`:

- category_id и category_name;
- purchases_minor;
- refunds_minor;
- net_expense_minor.

`CashPosition` является текущим вычисляемым представлением:

- as_of — календарная дата классификации;
- through — включительный горизонт обязательств;
- currencies[] — независимые `CurrencyCashPosition`.

`CurrencyCashPosition`:

- currency;
- liquid_balance_minor;
- overdue_obligations_minor;
- upcoming_obligations_minor;
- obligations_through_minor;
- available_after_obligations_minor;
- scheduled_payments[].

Инварианты фактической сводки:

1. `from < to`; обе границы содержат явный UTC offset.
2. Событие относится к периоду по `occurred_at`: `from <= occurred_at < to`.
3. Суммы возвращаются отдельно по валютам. Grace не складывает RUB, USD и
   другие валюты в одно число.
4. `income_minor` — credit-проводки `income_received`.
5. `purchases_minor` — debit-проводки `purchase`.
6. `refunds_minor` — credit-проводки `refund`.
7. `debt_payments_minor` — debit-проводки `debt_payment`; они не входят в
   purchases или category expense.
8. Internal transfer не входит в income, purchases, refunds, debt payments или
   net external cash flow. `internal_transfer_volume_minor` считает сумму
   debit-стороны, поэтому один перевод учитывается один раз.
9. `net_external_cash_flow_minor = income_minor + refunds_minor -
   purchases_minor - debt_payments_minor`.
10. Category purchases берутся только из purchase. Category refunds берутся из
    refund и категории исходной purchase. `net_expense_minor = purchases -
    refunds`; отрицательное значение разрешено.
11. `opening_balance_minor = Σ account.opening_balance_minor + Σ posting
    effects`, где event.occurred_at < from.
12. `closing_balance_minor` использует ту же формулу для event.occurred_at <
    to. Это вычисленный Grace balance на границе, а не импортированная банковская
    выписка.
13. Для любой валюты `closing = opening + net external cash flow`: internal
    transfers взаимно сокращаются. Нарушение означает ошибку данных или
    реализации.
14. Все компоненты одного ответа вычисляются из одного согласованного snapshot
    PostgreSQL. Конкурентно созданное событие не может попасть только в часть
    агрегатов и временно нарушить инварианты ответа.

Инварианты текущей ликвидной позиции:

1. `asOf <= through`; обе величины являются датами `YYYY-MM-DD`.
2. `liquid_balance_minor` — текущий вычисленный balance только счетов типов
   `debit` и `cash`. Credit и installment accounts возвращаются вне этого
   показателя и не считаются доступными деньгами.
3. В commitments входят только outstanding части ScheduledPayment с
   `due_on <= through`.
4. Outstanding часть вычисляется как scheduled amount минус все подтверждённые
   settlements на момент выполнения запроса. Историческое состояние settlement
   задним числом не реконструируется.
5. `overdue_obligations_minor` включает outstanding с `due_on < asOf`.
6. `upcoming_obligations_minor` включает outstanding с
   `asOf <= due_on <= through`.
7. `obligations_through_minor = overdue + upcoming`.
8. `available_after_obligations_minor = liquid balance - obligations through`;
   отрицательное значение разрешено и не является рекомендацией.
9. Полностью оплаченные scheduled payments не входят в commitments.
10. Все расчёты выполняются сервером целыми minor units.
11. `asOf` классифицирует due dates, но не является моментом исторического
    snapshot данных: liquid balance и settlements берутся в их текущем
    подтверждённом состоянии на момент выполнения запроса.
12. Все компоненты одного ответа вычисляются из одного согласованного snapshot
    PostgreSQL.

## API и OpenAPI

### Фактический период

`GET /api/v1/analytics/period-summary?from=<RFC3339>&to=<RFC3339>`

Пример:

```text
GET /api/v1/analytics/period-summary
    ?from=2026-10-01T00:00:00%2B05:00
    &to=2026-11-01T00:00:00%2B05:00
```

Ответ содержит исходные нормализованные границы и `currencies[]`. Пустой период
возвращает `200` и пустой массив, если нет ни счетов, ни проводок соответствующей
валюты.

### Текущая позиция

`GET /api/v1/analytics/cash-position?asOf=YYYY-MM-DD&through=YYYY-MM-DD`

`asOf` необязателен и по умолчанию равен текущей дате UTC. `through` обязателен.
Response включает только валюты, для которых существует liquid account или
outstanding scheduled payment до горизонта.

Оба endpoints read-only и не принимают `Idempotency-Key`.

## Данные, миграция и rollback

Новые таблицы и миграции не требуются. Модуль `analytics` читает канонические:

- accounts;
- financial_events;
- postings;
- categories;
- obligations;
- scheduled_payments;
- scheduled_payment_settlements.

Вычисления выполняются обычными PostgreSQL-запросами на текущем объёме данных.
Оптимизация индексами или materialized views допускается только после измерения
реального профиля; она не входит в первый контракт.

Rollback реализации состоит из удаления read-only routes/module/UI без
изменения пользовательских финансовых данных.

## UI

- раздел `Сводка периода` по умолчанию показывает текущий календарный месяц в
  локальной временной зоне браузера;
- React передаёт точные RFC 3339 границы с offset, сервер не угадывает timezone;
- пользователь может выбрать произвольные даты начала и конца;
- для каждой валюты показываются opening/closing balance, income, purchases,
  refunds, debt payments и net external cash flow;
- внутренние переводы показываются отдельно и явно помечаются как не влияющие
  на доходы/расходы;
- category breakdown показывает purchases, refunds и net expense;
- раздел `Доступно после обязательств` позволяет выбрать дату `through`;
- он показывает liquid balance, overdue, upcoming и математический остаток;
- разные валюты никогда визуально не объединяются в один total;
- интерфейс не формулирует советов и не окрашивает отрицательный результат как
  оценку поведения; он показывает факт дефицита нейтрально.

## Ошибки и негативные сценарии

- отсутствующие, невалидные или неупорядоченные `from/to`: `422`;
- RFC 3339 без явного offset: `422`;
- интервал длиннее 5 лет: `422` для ограничения ошибочных запросов;
- отсутствующий или невалидный `through`: `422`;
- `through < asOf`: `422`;
- горизонт дальше чем на 5 лет от `asOf`: `422`;
- отсутствие данных: `200` с пустыми currency collections;
- неизвестная категория отображается стабильной группой `Без категории`, а не
  теряется;
- отрицательный net expense или available after obligations остаётся
  допустимым числом;
- ошибка одного aggregate query не возвращает частичную сводку.

## Критерии приёмки

- [x] Period API принимает явный полуоткрытый RFC 3339 интервал.
- [x] События ровно на `from` включаются, ровно на `to` исключаются.
- [x] Результаты разделены по валютам и никогда не конвертируются скрыто.
- [x] Income, purchase, refund и debt payment агрегируются по отдельной
      семантике.
- [x] Internal transfer не меняет net external cash flow и считается один раз
      только как transfer volume.
- [x] Category breakdown корректно net-ит refunds против purchases.
- [x] Opening/closing balance согласованы с postings на границах периода.
- [x] Каждый ответ вычисляется из одного согласованного PostgreSQL snapshot и
      сохраняет свои арифметические инварианты при конкурентных записях.
- [x] Cash-position включает только liquid debit/cash balances.
- [x] Overdue и upcoming obligations разделены относительно `asOf` и `through`.
- [x] Settlements уменьшают commitments, полностью оплаченный платёж исчезает.
- [x] Available after obligations вычисляется сервером и допускает минус.
- [x] Read-only analytics не создаёт и не изменяет финансовые записи.
- [x] Невалидные границы и чрезмерные диапазоны возвращают `422`.
- [x] OpenAPI и generated TypeScript проходят сборку.
- [x] React показывает периодную сводку и cash position отдельно по валютам.
- [x] Existing account/event/receipt/obligation пути не регрессируют.
- [x] Production dependency graph не содержит AI/LLM/provider модулей.
- [x] `make test`, предметный integration test и `git diff --check` проходят в
      implementation worktree.
- [ ] `make test`, `make test-integration` и `git diff --check` проходят после
      merge.

## Исполняемые проверки

```bash
make test
make test-integration
./scripts/test-analytics.sh
git diff --check
```

`test-analytics.sh` должен создавать уникальный compose-project, использовать
автоматически назначенные host ports и очищать своё окружение через trap.

## Принятые решения owner

1. Фактический period summary и текущий cash position являются двумя разными
   read-only endpoints, а не одним смешанным показателем.
2. Период задаётся точными RFC 3339 границами `[from, to)`; timezone выбирает
   клиент явно.
3. Все результаты разделяются по валютам; FX conversion отсутствует.
4. Debt payment уменьшает денежный поток отдельно от purchases и никогда не
   попадает в category expense повторно.
5. Opening/closing balance — математическая реконструкция от существующего
   opening balance и occurred_at проводок, не банковский snapshot.
6. В liquid balance входят только debit и cash accounts; credit/installment
   accounts не считаются доступными деньгами.
7. Cash position использует текущие balance и подтверждённое
   settlement-состояние; `asOf` только классифицирует due dates и не
   реконструирует историческую версию финансовых данных.
8. `availableAfterObligations` — арифметический факт, может быть отрицательным и
   не является бюджетом, прогнозом или советом.
