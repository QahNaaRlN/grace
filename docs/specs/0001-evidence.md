# SPEC-0001 — verification evidence

- Статус проверки: integrated
- Implementation commit: `4dbb418`
- Integration merge: `c2a27b4`
- Ветка: `codex/financial-events-postings`
- Дата: 2026-10-04

## Автоматические проверки

| Проверка | Результат |
| --- | --- |
| `make test` | Go tests, ESLint, OpenAPI TypeScript generation, TypeScript и Vite production build прошли |
| `make test-integration` | Изолированная PostgreSQL-среда создана на автоматически назначенных портах; сценарий завершился `SPEC-0001 integration checks passed` |
| `git diff --check` | Прошёл без замечаний |

Интеграционный сценарий доказал:

- миграцию `v2 → v3`, сохранение legacy-операции и вычисленного остатка;
- безопасный rollback legacy-only состояния и повторный upgrade;
- purchase, income, debt payment, transfer и refund;
- повтор команды без дубликата и `409` при конфликтующем payload;
- равенство общего объёма средств до и после внутреннего перевода;
- отказ `422` для refund сверх исходной покупки;
- сохранение отрицательного вычисленного остатка;
- отсутствие удалённого `/api/v1/transactions` (`404`);
- CORS preflight для обязательного `Idempotency-Key`.

## Браузерный smoke test

Полный compose-стек был запущен из implementation worktree на изолированных
портах `55435/18081/15174`. В браузере подтверждены:

- загрузка двух счетов и списка финансовых событий;
- корректное направление перевода `Основная карта → Накопительный счёт`;
- создание покупки через React-форму;
- обновление производного остатка `137 650,10 ₽ → 137 524,60 ₽`;
- появление события `Кофейня`, суммы `−125,50 ₽` и уведомления
  `Покупка сохранена`.

Во время smoke test были найдены и исправлены два сквозных дефекта:

1. браузерный API URL и CORS origin теперь следуют `API_PORT` и `WEB_PORT`;
2. CORS разрешает обязательный заголовок `Idempotency-Key`.

## Post-merge verification

На `main` после merge `c2a27b4` повторно прошли:

- `make test`;
- `make test-integration` с итогом `SPEC-0001 integration checks passed`;
- `git diff --check`.

Рабочее дерево `main` после проверок осталось чистым. SPEC-0001 интегрирована.
