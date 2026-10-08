# SPEC-0028: проверка реализации

Scope принят владельцем 08.10.2026. Основание: SPEC-0028, принятие
«Да, реализовать и продолжить импорт». Приватные источники не входят в Git.

| Критерий | Исполняемое свидетельство | Состояние |
|---|---|---|
| AC1 неизвестный/известный opening на несверенном счёте | TestOpeningBalanceKnowledge; TestReadToolsAgainstPostgreSQL / SPEC-0028 unknown balances preserve flows | unit и PostgreSQL прошли |
| AC2 прежняя арифметика | прежние auth/assistant/analytics integration suites, адаптация nullable DTO без смены числовых ожиданий | unit и integration прошли |
| AC3 смешанная аналитика | тот же PostgreSQL subtest: RUB unknown и EUR 500−23=477; unknown credit не портит liquid RUB | PostgreSQL прошёл |
| AC4 отображение и подтверждение | HomePage.test.tsx / SPEC-0028 unreconciled accounts; build TypeScript | 43 web tests, lint и build прошли |
| AC5 контракт/изоляция/миграция | generate:api; auth integration; rollback guard test; migration 000011 | generated types сохранены; integration прошёл |
| AC6 реальная история | server previews и confirmation receipts; сверка событий с приватным планом импорта | ожидает выпуска и переноса |

`bash scripts/test-windows.sh unit`: make test и go vet прошли.
`git diff --check`: прошёл. `bash scripts/test-windows.sh integration`:
все PostgreSQL suites прошли (exit 0).

Миграция разрешает NULL opening, добавляет balance_reconciled, сохраняет
прежний default=true и не меняет событий/проводок/provenance. Откат
несверенных счетов запрещён, без подстановки нулей. Тест импорта теперь
откатывает до версии 8 явно, чтобы новые миграции не ослабляли прежнюю
проверку сохранности импортированной истории.

Remote CI №47 (37770830193), commit b890fe9: success. Выпуск и итог импорта пока ожидаются.
