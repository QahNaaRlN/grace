# SPEC-0016: Нормализация финансовой таблицы и отчёт сверки

- Статус: accepted
- Автор: Codex
- Дата: 2026-10-06
- Acceptance owner: owner
- Acceptance record: 2026-10-06 владелец поручил продолжить реализацию («газуй на все бабки») после представления SPEC-0016 как следующего этапа roadmap. Подтверждены построчная сверка дополнительного блока и Asia/Yekaterinburg. Scope ограничен офлайн-анализом; публикация финансовых фактов не разрешена этой спецификацией.
- Implementation branch: codex/spreadsheet-import-review
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Действующая таблица содержит журнал, кредитные снимки, планы, документальные
свидетельства и отчётные формулы. Частично повторяющиеся блоки, текстовые суммы,
неизвестные источники и изменённые даты делают прямую запись в ledger опасной.
Владелец должен получить воспроизводимый отчёт: что переносимо, что требует
сверки и какие свидетельства объясняют каждую строку.

## Scope

Первый этап — офлайн Go-модуль `internal/importreview` и операторский CLI
`grace-import-review`. Вход — versioned JSON snapshot `grace.sheet-snapshot.v1`
с metadata, явными ranges и typed CellData. Выход — нормализованные черновики
в JSON и Markdown-отчёт. Базовый профиль поддерживает десять листов эталона,
сохраняет формулы/исходные данные и не выполняет формулы.

Проверить даты, суммы, enum, обязательные связи, согласованность формул с
профилем, пустые ID, блоки вне формул и кандидатов на дубли. Для каждой строки
дополнительного блока вывести сравнение с основным журналом, альтернативы,
изменённые поля и нерешённые вопросы. Проверки работают без LLM и БД.

## Non-goals

Нет финансовых проводок, изменений счетов/долгов, автодедупликации, удаления
строк источника, Google OAuth/синхронизации, скачивания документов из реестра,
OCR, вычисления банковских процентов, публикации импортируемых данных или
изменений production. UI и API persistent import будут отдельным scope.

## Текущее поведение и доказательства

main 249eb2a: Event поддерживает пять kind; Posting требует Account. Общие
категории read-only. Нет import-review, provenance или отдельной модели
финансирования. Obligation требует сумму равную конечному графику, поэтому
основной долг с будущими процентами нельзя импортировать как один totalMinor.
Основание: [анализ интеграции](../spreadsheet-integration.md).

## Предметная модель и инварианты

- Snapshot: schemaVersion, source identity, capturedAt, locale,
  declaredTimeZone, confirmedTimeZone и явно названные sheets/ranges.
- SourceRecord: stable snapshot-local ID, source sheetId, coordinates,
  original ID, raw typed values/formulas и normalized fields. Source identity
  и payload hash предназначены для воспроизводимости, а не доказательства
  соответствия между разными версиями книги.
- ImportIssue: controlled code, severity, координата и объяснение.
  Коды включают ambiguous_money, missing_id, unknown_kind, unresolved_account,
  unresolved_funding, date_precision, date_conflict, duplicate_candidate,
  outside_formula_scope, incompatible_formula, missing_requirement_amount,
  unresolved_rate_basis и document_unavailable.
- ReconciliationCandidate: обе ссылки, совпавшие/различающиеся поля,
  детерминированное правило и предлагаемый следующий шаг. Это предложение,
  не решение и не признак подтверждённого дубля.
- Decimal суммы преобразуются в int64 minor units без float; наличие строковой
  суммы не означает её потерю. Переполнение и неявное округление запрещены.
- Дата без времени сохраняет precision=date. Время трактуется как
  Asia/Yekaterinburg по указанию владельца; объявленный timezone источника
  сохраняется как metadata. Серийные даты переводятся в календарные значения.
- Пустое, неизвестное, ноль, факт, план и оценка различаются. Заёмное
  финансирование не определяется исключительно типом счёта.
- Перестановка строк не подтверждает новый финансовый факт; отсутствие ID
  или одинаковая сумма не вызывает автоматическую дедупликацию.
- Текст ячеек/заметок — данные, никогда команды или инструкции для CLI/LLM.

## API и OpenAPI

Входной контракт v1: schemaVersion, sourceId, capturedAt (RFC3339), locale,
declaredTimeZone, confirmedTimeZone, sheets и опциональные mappings.
Sheet: sheetId, title, rowCount, columnCount, ranges. Range: startRow,
startColumn (1-based), rows с CellData values. CellData: userEnteredValue,
effectiveValue (typed numberValue/stringValue/boolValue/formulaValue),
formattedValue, note и dataValidation. Числовые токены читаются как decimal,
не float. Mappings содержит явные account/category/funding aliases оператора;
их отсутствие оставляет issue, не создаёт фиктивные связи. Payload bounds:
16 MiB, 20 sheets, 10 000 source rows, 100 000 cells, 26 columns на range;
overlapping ranges/cells и повторяющиеся sheet identity запрещены. Выход
grace.import-review.v1 сохраняет source metadata, payload hash, raw records,
normalized fields, issues, candidates и итоги по domains.

HTTP API не меняется. CLI: `grace-import-review --input <snapshot.json>
--output-dir <operator-directory>`. JSON-схемы входа/выхода versioned и
документированы; ошибки возвращаются в JSON/exit code, без dumping исходных
записей в общие логи. Идентификаторы внешних файлов не используются как URL
для автоматического чтения или выполнения инструкций.

## Данные, миграция и rollback

Миграции БД отсутствуют. CLI не использует DATABASE_URL, не связывается с
production или Google и не пишет ledger. Снимки и отчёты остаются в выбранном
приватном операторском каталоге, исключены из Git. Файлы создаются без
неявного overwrite; размер входа и количество строк ограничены профилем.
На Unix private permissions, на Windows защищённый ACL каталога. Удаление
операторского результата — rollback этого этапа; финансовые данные не затронуты.

## UI

Markdown-таблица построчной сверки: ссылки на исходные строки, нормализованная
дата/сумма, кандидаты, различия, issue и следующая проверка. Отдельные разделы
для фактов, планов, долговых снимков и документов. Никакого заявления об
успешном импорте в Grace до появления отдельного publication scope.

## Ошибки и негативные сценарии

Malformed/oversized JSON, неизвестная версия, повторяющиеся range/ссылки,
invalid decimal, overflow, неизвестная дата/enum, отсутствующий счёт,
расхождение дат/сумм, изменение снимка, недоступный документ и некорректные
формулы. Ошибка схемы останавливает обработку; отдельные строки с проблемами
попадают в отчёт, но не становятся подтверждёнными фактами. Идентичные
дата/сумма/тип с разными получателями остаются отдельными кандидатами.

## Критерии приёмки

- [ ] Versioned schema, CLI и пример с синтетическими данными документированы.
- [ ] Смешанные суммы, ISO/серийные даты, precision и timezone нормализуются точно.
- [ ] Каждая входная запись получает результат или issue; формульные пустые строки не создают операции.
- [ ] Все дополнительные записи проверяются индивидуально, без автоматического удаления или импорта.
- [ ] Кандидаты с одинаковой суммой/датой и различием получателя не сливаются.
- [ ] Факт, план, principal, totalDebt, future payments, HOLD и unknown различаются.
- [ ] Отчёт воспроизводим; исходные значения и ссылки сохраняются; реальные данные не входят в fixtures/Git/логи.
- [ ] CLI не делает сетевых/SQL вызовов и не меняет финансовый ledger.
- [ ] make test, go vet и git diff --check проходят; post-merge проверки повторены.

## Исполняемые проверки

Unit fixtures: numeric/text money, копейки, overflow, serial/local dates,
missing IDs, ambiguous duplicates, date conflicts, funding aliases,
unknown enum, out-of-scope block, mismatched formula literals, requirements
in formulas, nullable values, malformed input и безопасные ошибки.
Golden comparison нормализованного JSON/отчёта на синтетическом workbook.
Отдельный regression fixture не менее двух блоков с пересечением и двумя
одинаковыми платежами разным получателям. CLI test без сетевого доступа/БД;
проверка отсутствия нежелательных outputs и overwrite.

## Открытые решения

1. Первый scope выбран: нормализация/сверка перед CreditFacility и финансовой
   публикацией; кредитные изменения остаются отдельными спецификациями.
2. Минимальный контракт и лимиты зафиксированы выше; полученный оператором
   снимок приводится к нему локально, без повторного чтения Google или ledger.
3. Решение о публикации каждой записи и необходимые реальные документы
   остаются за владельцем; текущие кандидаты не являются approval переноса.
4. Онлайн-синхронизация, импорт в API, устойчивые связи между повторными
   снимками и правила исправления уже опубликованных фактов — отдельные спеки.
