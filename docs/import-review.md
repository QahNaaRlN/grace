# Офлайн-сверка финансовой таблицы

Основание: [SPEC-0016](specs/0016-spreadsheet-import-review.md).

`grace-import-review` читает локальный типизированный снимок и создаёт
`review.json` и `review.md`. Команда не импортирует финансовые факты, не
обращается к Google/банку/ИИ/БД и не использует DATABASE_URL или API keys.
Это операторский инструмент первого этапа, а не новая страница Web.

## Запуск

```sh
go build -o bin/grace-import-review ./cmd/import-review
bin/grace-import-review --input /private/snapshot.json --output-dir /private/new-review
```

На Windows бинарник имеет расширение .exe. `output-dir` должен быть новым
каталогом с существующим родителем вне Git checkout. Каталог создаётся с
0700, файлы с 0600 на Unix; на Windows оператор выбирает родительский каталог
с приватным ACL. Существующий результат никогда не перезаписывается.
Использование symlink родителя проверяется по его реальному пути; отчёт
внутри Git checkout отклоняется. Снимки и реальные отчёты не коммитить.

Для примера без реальных данных:

```sh
go run ./cmd/import-review --input testdata/import-review/snapshot.json --output-dir /private/example-review
```

stdout содержит только status, числа records/candidates/issues и
`published=false`. stderr — JSON с контролируемым error code, без содержимого
ячеек, имени приватного файла и ответов провайдеров. Exit 0 означает, что
анализ завершён, а не что данные подтверждены или записаны в ledger.
Exit 1 — ошибка входа/результата/лимита; exit 2 — неверные аргументы.

## Входной контракт

[JSON Schema v1](schemas/sheet-snapshot.v1.schema.json) и
[синтетический пример](../testdata/import-review/snapshot.json).

Snapshot содержит schemaVersion=grace.sheet-snapshot.v1, sourceId,
capturedAt (RFC3339), locale=ru_RU, declaredTimeZone, confirmedTimeZone,
sheets и опциональные mappings. Профиль рассчитан на эталонную рублёвую
таблицу; применение к другой валюте/структуре требует отдельного профиля.

Sheet: sheetId, title, rowCount, columnCount, ranges. Range: startRow,
startColumn **с единицы**, rows с values. В каждой CellData сохраняются
userEnteredValue, effectiveValue, formattedValue, note и dataValidation.
Typed Value содержит ровно одно numberValue/stringValue/boolValue/formulaValue.
numberValue — JSON number, не строка. Формула остаётся свидетельством:
используется только её предоставленный cached effectiveValue, сама формула
не вычисляется и её текст не исполняется. Пустые формульные строки не дают
операций; непустые вычисленные строки сохраняются с предупреждением.

Лимиты: 16 MiB, 20 sheets, 10 000 source rows, 100 000 cells, до 26 колонок,
50 000 кандидатов. Повторяющиеся JSON members, sheet identity и overlapping
ranges/cells запрещены. Профиль распознаёт десять русских названий листов
эталона; неизвестные листы не теряются, а сохраняются как unclassified/issue.
Заголовки известных профилей не являются финансовыми записями.

Aliases в mappings.accounts/categories задаются оператором явно и
сопоставляют исходную строку с канонической ссылкой. Это офлайн-ссылки:
существование/ownership/тип сущностей нужно проверять при будущем publication.
mappings.funding хранит own/borrowed/unknown и опциональный facilityId для
borrowed. Маршрут со стрелками не разбирается автоматически в счета.

## Нормализация

- Суммы разбираются как точные decimal и преобразуются в int64 копейки без
  float-арифметики. Поддерживаются числовые JSON tokens, десятичные строки
  с точкой/запятой, корректные пробелы тысяч и суффикс ₽. Неоднозначность,
  переполнение или дробная копейка дают issue; округление не выполняется.
- Пустое — null, ноль — 0. В transaction требуется положительная сумма;
  банковский остаток может быть отрицательным.
- ISO/российские календарные даты и серийные даты Sheets сохраняют precision.
  Date-only не получает выдуманного времени. Локальные часы используют явно
  подтверждённый timezone, RFC3339 сохраняет известный момент. Серийные
  дроби с точным целым числом секунд дают serial_second; для дробей,
  приближающих время, сохраняются календарная дата, precision=serial_fraction
  и raw decimal без выдуманного UTC-момента. Дата считается от 1899-12-30.
- Факт операции, план дохода, кредитный/долговой снимок, требование платежа,
  банковская сверка, документальный реестр и проекция имеют разные domains.
  reportedDebt не переименовывается в principal автоматически; будущий
  платёж и graceRequirement не смешиваются. Ставка без basis остаётся issue.
- Контроль остатков проверяется точно: opening + credits − debits = closing.
  Надпись «Сходится» не заменяет отсутствующее значение.
- У SourceRecord snapshot-local hash ID: sourceId/sheetId/row/canonical payload
  hash. Это не persistent financial UUID и не доказательство идентичности
  между версиями книги. Original ID, формулы и raw данные сохраняются.

## Сверка дополнительного блока

В профиле основными считаются строки «Операции» 5–304; каждая запись после
304 получает outside_formula_scope и рассматривается отдельно. Правила:
одинаковые календарная дата/тип/сумма либо описание/тип/сумма с другой датой.
Вывод содержит обе ссылки, совпадения, различающиеся колонки и обязательное
needs_confirmation. Все подходящие кандидаты сохраняются: одинаковые
платежи разным получателям и совпадение на другую дату не сливаются.
Если кандидата нет, это не доказательство новой операции. Файлы документов
из реестра не скачиваются и не считаются заново проверенными.

## Выход

[JSON Schema отчёта](schemas/import-review.v1.schema.json):
grace.import-review.v1, metadata источника, canonical payloadSha256,
records с raw/normalized/issues, candidates, общие issues и domainCounts.
Markdown содержит отдельные domains, контрольные значения, сверку и
объяснения ошибок. Вывод детерминирован для одного снимка; при изменении
содержимого изменяется payload hash. Нет статуса «готово к финансовому импорту».

Следующий отдельный этап — сохраняемые черновики, проверка реальных
канонических связей, preview и подтверждение публикации в financial core.
PDF, фото, LLM API и UI импорта не входят в SPEC-0016.
