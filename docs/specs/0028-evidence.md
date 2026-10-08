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
| AC6 реальная история | server previews и сохранённые confirmation receipts; повторное чтение всех событий и проводок; полная построчная сверка с приватным планом | production импорт завершён; расхождений нет |

`bash scripts/test-windows.sh unit`: make test и go vet прошли.
`git diff --check`: прошёл. `bash scripts/test-windows.sh integration`:
все PostgreSQL suites прошли (exit 0).

Миграция разрешает NULL opening, добавляет balance_reconciled, сохраняет
прежний default=true и не меняет событий/проводок/provenance. Откат
несверенных счетов запрещён, без подстановки нулей. Тест импорта теперь
откатывает до версии 8 явно, чтобы новые миграции не ослабляли прежнюю
проверку сохранности импортированной истории.

Remote CI №47 (37770830193), commit b890fe9: success. Финальный PR CI №48
(37771729295), commit 11ba05d: success. Main CI (37831992772) и Delivery
(37832513086) для f257f8e: success.

Deploy production №12 (37832621051) успешно выпустил точный f257f8e.
В журнале выпуска наблюдались verified pre-deploy backup и migration 11;
internal health и публичные проверки authentication/application прошли.
После миграции решения черновика сохранились, до публикации журнал был пуст.

2026-10-09 история перенесена через UI Grace. Созданы несверенные счета
с известным либо неизвестным начальным остатком; дубли не создают новых
событий, общие итоги чеков исключены. Сервер пометил весь импорт completed;
оба месячных отчёта показывают отсутствие неподтверждённых строк.
Сохранённые квитанции повторно прочитаны: уникальность event IDs, вид,
описание, получатель, точность даты/времени, счета и каждая сумма проводки
совпали с проверенными preview. Число проводок и суммы по видам совпали
с приватным планом; расхождений нет. Банковская полнота и текущие остатки
не объявлены сверенными. Приватные источники и результаты сверки не входят
в Git/PR; они сохранены локально вне checkout.
