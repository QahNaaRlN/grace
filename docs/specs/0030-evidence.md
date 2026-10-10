# SPEC-0030: implementation evidence

2026-10-09. Accepted [SPEC-0030](0030-statistics-charts.md), branch
codex/statistics-charts от aaddf64 с принятым дизайном SPEC-0029.
Реализовано локально; main, remote CI и production не изменены.

| AC | Результат | Исполняемое свидетельство |
| --- | --- | --- |
| AC1 | PASS | go test ./internal/analytics; statistics_test.go проверяет partial month/week boundaries, понедельник, offset+05, invalid dates/year range, int64 overflow и JS precision limit. bash scripts/test-analytics.sh на настоящем PostgreSQL: RUB/ USD раздельно; October local purchases14000 (включая Sep30UTC), refunds3000, income50000, debt7000, outflows21000; weekly sum совпадает; transfers20000 исключены. Date-only purchase сохраняет дату. Пустой August имеет zero series и пустые категории. Unknown opening/closing remain null; historical unclassified purchase20 получает «Без категории». Other-owner GBP999999 не попадает в summary/series. До/после чтения GET financial-events полностью совпадает |
| AC2 | PASS | authoritative api/openapi.yaml + npm run generate:api; типы Statistics/StatisticsCurrency/StatisticsBucket из generated schema. Existing auth router/tenant pool retained; запросы invalid date, >year и day granularity дают422 в integration. Snapshot RepeatableRead ReadOnly через TenantPool (SET LOCAL grace_financial + owner_id) |
| AC3 | PASS в scope | StatisticsPage.test.tsx: gross/refunds labels, таблицы платежей/возвратов, keyboard Enter selection, grouping API params, currency switch, error/retry/empty, unsafe money rejection, compact navigation. Browser smoke actual components на synthetic data: tap выбирает точную сумму, table details открывается, обе темы;390px document375,320px document305 (15px scrollbar), без overflow страницы. Графики/таблицы прокручиваются внутри card. Desktop1449/document1434. Оси начинаются с нуля; empty periods не рисуют фиктивную шкалу; native date/select color-scheme соответствует theme |
| AC4 | PASS local | make test: Go, Web48 tests/11 files, lint/build, offline Jev49/72. git diff --check. Final bash scripts/test-analytics.sh: SPEC-0004 and SPEC-0030 analytics integration checks passed |

Windows integration requires Git Bash and MSYS_NO_PATHCONV=1, чтобы Docker
-path=/migrations не превращался в Windows Git path. Docker Desktop был запущен
для локальных checks; после временной остановки восстановлен и final run PASS.
Тестовые compose projects имеют уникальные имена, volumes удалены cleanup trap.
Ручной purchase API требует category; uncategorized fixture создаёт допустимый
исторический bank_import напрямую только в disposable integration DB.

Логи в ignored web/node_modules/statistics-final-check.log и
statistics-final-integration.log. Временный QA scaffold сохранён в ignored
web/node_modules/.grace-statistics-qa, не включён в приложение; Vite QA остановлен.
Визуальное превью использует искусственные данные. Реальный UI делает запросы
к новому endpoint; демонстрационный scaffold использовал fixed response, поэтому
группировка/период проверены API integration и automated client tests.

Ограничения: полнота истории не подтверждается существующей моделью —
coverage recorded_only явно виден. Доли — gross purchases, возвраты отдельно;
округлённые basis points могут дать сумму ниже100%. Link открывает последние
операции (API100), не полный category drilldown. Прогноз остатков, дополнительные
валютные конверсии, публикация и полный accessibility certification вне scope.
Read-only статистика не зависит от LLM и не вызывает платный inference.
