# SPEC-0027: Исторические операции с неизвестным временем

- Статус: integrated (выпуск `f257f8e`: CI `main` и Deploy production №12 успешны, см. [0028-evidence](0028-evidence.md))
- Автор: Codex
- Дата: 2026-10-08
- Acceptance owner: owner
- Acceptance record: 2026-10-08 владелец подтвердил правило «сохранять календарный день с отметкой время неизвестно, без подстановки часов» для 160 исходных строк перед выпуском и переносом всей таблицы.
- Implementation branch: codex/history-release
- Implementation commit: 597cfcc

## Проблема и пользовательский результат

В «Финансовом здоровье» 156 операций имеют только дату, четыре — приблизительную
дробную serial date. SPEC-0017 требует вручную указать точное время для создания.
Это не позволяет честно перенести историю. Сохраняем исходный календарный день,
оставляя occurredAt=null, и учитываем его в календарной аналитике.

## Scope и UI

nullable occurred_at и отдельный occurred_date DATE в financial_events;
импорт может передать occurredDate YYYY-MM-DD вместо occurredAt RFC3339.
API возвращает occurredAt=null, occurredDate, datePrecision=date; старые операции
возвращают timestamp, occurredDate=null, datePrecision=instant.
UI журнала/деталей/импорта показывает только день и «время неизвестно»;
исходные metadata/formulas/precision сохраняются в provenance. Локальный
календарь истории — Asia/Yekaterinburg. Manual API по-прежнему принимает время.

## Non-goals

Придумывание времени, автоматическое слияние дублей, создание неизвестных
остатков/счетов, самостоятельное подтверждение импорта или AI интеграция.

## Предметная модель и инварианты

Ровно одно из occurredAt/occurredDate известно. Валидная реальная дата;
timestamp не подставляется в БД или публичный ответ. Внутренняя сортировка и
календарные границы SQL могут использовать начало дня как boundary key, но
оно не является временем события. Месячные/дневные интервалы в Yekaterinburg
учитывают полный день. При date-only записи на неполной граничной дате
точный timestamp-интервал отклоняется как time_precision_required, чтобы
не определять неизвестный внутридневной порядок. Доход с неизвестным временем
не может быть точным anchor для «после последней зарплаты»; нужен календарный
период. Финансовые суммы/валюты/проводки/tenant isolation неизменны.

## Текущее поведение

Основание: SPEC-0017/0023, financialevents/types.go и occurred_at NOT NULL;
приватная офлайн-сверка 373 records/191 transaction rows, 160 без точного времени.
Реальные строки не включаются в fixtures/публичные evidence.

## API и OpenAPI

FinancialEvent получает nullable occurredAt и optional occurredDate/datePrecision
для additive client migration. ImportFields accepts optional occurredAt or
occurredDate (ровно одно); provider date precision сохраняется отдельно.
422 time_precision_required для period summary, income_time_unknown для
income-anchor tool. Calendar analytics UI задаёт Yekaterinburg boundaries.
Generated TypeScript обновляется; неизвестная дата не отображается как epoch.

## Данные, миграция и rollback

Миграция 000010 adds nullable occurred_date, nullable occurred_at и XOR CHECK.
Старые строки не переписываются. Down допускается только при отсутствии date-only
событий. После публикации требуется совместимая версия приложения или восстановление
backup; downgrade в прежний API с occurredAt-only запрещён. До первого импорта
обычный deployment rollback безопасен. Полный backup перед release обязателен.

## Ошибки и негативные сценарии

Невалидная/несуществующая дата, одновременные дата и время, отсутствие обоих,
чужие счета, source mismatch, repeated confirmation, partial-day analytics,
unknown income order. Existing exact-time paths работают как раньше.

## Критерии приёмки

- [x] AC1: date-only import сохраняет occurred_at NULL и точный occurred_date,
  raw precision/provenance; повтор не создаёт postings.
- [x] AC2: даты точно валидируются; кандидаты и месячная сверка поддерживают оба
  вида даты без автоматического слияния.
- [x] AC3: дневная/месячная аналитика учитывает date-only, partial-day и unknown
  income anchor дают явный отказ; exact-time regressions отсутствуют.
- [x] AC4: UI и contract не показывают придуманного времени/epoch; mobile reflow.
- [x] AC5: migration up/down guard, imports integration, make test, go vet и
  git diff --check проходят; evidence привязывает критерии к executable checks.

## Исполняемые проверки

go test ./internal/financialevents ./internal/imports ./internal/analytics ./internal/assistant;
bash scripts/test-imports.sh; make test; go vet ./...; git diff --check.
Synthetic PostgreSQL integration для nullable timestamp, whole-day summary,
partial-day rejection, unknown income and guarded rollback.

## Открытые решения

Правило принято владельцем; фиктивное время не используется. Дубли, канонические
счета/категории и opening balance перед историей требуют отдельных решений
владельца на реальном preview. Этот scope расширяет ограничение date-only create
SPEC-0017/0023 на основании явного принятия от 2026-10-08.
