# SPEC-0026: диагностика и исправление synthetic probes

Дата: 2026-10-09. Основание: accepted [SPEC-0026](0026-ai-assistant-receipts.md),
разрешение владельца на эксперимент до $10 с комиссиями и поручение устранить
ошибки. Только искусственные текст/чек, отдельная локальная PostgreSQL БД;
покупки и проводки не создавались. Production ИИ остаётся выключенным.

## Установленная причина и изменение

Receipt: повтор исходного запроса вернул HTTP 400 от Google, INVALID_ARGUMENT:
схема создаёт слишком много состояний constrained decoder. В provider schema
были вложенный массив до 50 объектов, ограничения строк до 300 и sourceText
до 12000. Убраны provider-side length bounds; типы, nullable, обязательные поля
и additionalProperties=false сохранены. Go проверяет все прежние пределы,
а также запрещает выбор счёта и groupSuggestion моделью. Ограничение ответа
2000 tokens и HTTP body 1 MiB сохраняется. Новый запрос прошёл без смены модели,
маршрута, privacy или price policy.

Analysis: первый probe был отклонён `unsupported_claim`; его candidate не
сохранялся, поэтому точный нарушенный инвариант установить задним числом нельзя.
Повтор исходного запроса прошёл. Возможные нарушения (числа/проценты, неизвестные
factIds, переносы строк) проверяются regression tests. Prompt явно запрещает
копировать дату и factIds в prose. При отклонённом ответе приложение сохраняет
Go facts и billing, возвращает пустые interpretations с явным предупреждением.
Непроверенный текст не публикуется, повторного inference автоматически нет.
Это исправляет обработку отказа; не обещает соблюдение prompt моделью всегда.

## Наблюдаемые запросы

Модель `google/gemini-2.5-flash-lite`, routing slug `google-vertex`, metadata
provider `Google`. Свежая public endpoint pricing: $0.10 input / $0.40 output
за миллион tokens. Только прямой маршрут, первая попытка, ZDR,
data_collection=deny, require_parameters, max_price; нет fallback/tools/plugins.

| Проверка | Результат | Latency | Input/output tokens | Учтено, microUSD |
| --- | --- | --- | --- | --- |
| Первоначальный analysis | unsupported_claim | сохранено в первоначальном отчёте | 519/264 | 158 |
| Первоначальный receipt | provider_unavailable, дополнительного списания нет | сохранено в первоначальном отчёте | неизвестно | 0 после billing reconciliation |
| Повтор исходного analysis | PASS validator | 1433 ms | 519/251 | 153 |
| Receipt до исправления | HTTP 400 schema complexity | 2197 ms | неизвестно | 0 после billing reconciliation |
| Receipt после исправления | PASS, 90.00 RUB, date-only | 2212 ms | см. generation billing | 309 |
| Analysis после уточнения prompt | PASS validator | 1389 ms | 576/247 | 157 |

Receipt: MILK, quantity 1, unitPrice 100.00, lineTotal 90.00, discount 10.00,
seller SYNTHETIC STORE, issuedDate 2026-09-01, issuedAt/brand/package/broadGroup
null. Сумма/валюта совпали с fixture; обязательное подтверждение осталось
заблокированным без выбранного человеком счёта/решения.

Successful generation IDs для сверки:
`gen-1791540097-8DNdvu4t6lUnHGbmCV2g`,
`gen-1791541606-TDvVxNktFlZfSp0Xgo8C`,
`gen-1791541821-jKM8E1em4ffmrgU4tFRP`,
`gen-1791541857-eofNK70yFAky1elUTghZ`.
Invoice costs: $0.0001575 + $0.0001523 + $0.000309 + $0.0001564 = $0.0007752.
Grace консервативно округляет каждый invoice вверх до microUSD: суммарно 777.
Комиссия пополнения — $0 по ответу владельца. Ключ имеет отдельный предел $5
без сброса. Локальный probe allowance $0.01 не сбрасывался; usage сохранён.
Отказы receipt сверены по равенству key total и известных generation invoices;
HTTP 400 сам по себе не является доказательством бесплатности.

## Границы результата

Это успешные точечные smoke checks, не сравнение 2–3 моделей на 20 чеках,
не завершение полного корпуса analysis и не калибровка Jev. AC2/3/4 остаются
частичными. Реальные пользовательские данные не передавались. Rollout,
backup/restore и отдельное согласие на личный анализ/чек ещё требуются.
Regression checks и критерии: [implementation evidence](0026-evidence.md).
