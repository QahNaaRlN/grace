# SPEC-0055 / S1.7: точная идентичность модели в screening

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: local_verified.
- Acceptance record: 2026-10-21, прямое «Утвердить S1.7 и offline-реализацию».
  Разрешена реализация I1–I4, без inference/деплоя/переключения модели.
- Основание: [диагностика](../research/statement-quality/s17-selected-model-identity.md),
  [S1.6](0055-s16-evidence.md), [восстановление учёта](0055-s16-reconciliation-evidence.md).
- Подготовлен в существующем review-worktree `statement-diagnostic-smoke`;
  принятие зафиксировано до production-кода. По прямому поручению владельца
  реализация продолжает текущий review в этом worktree; интеграция в main
  остаётся отдельным действием после проверок.

## Проблема и пользовательский результат

S1.6 остановлен на selected_model до scorer; фактическая строка endpoint.Model
не сохранена. Каталог связывает запрошенный alias и отдельное canonical-имя.
Синтетический ответ с таким canonical сейчас отвергается. Это доказанный пробел
совместимости, но ещё не доказанная причина исторического ответа.

Нужен screening, способный проверить только эту точную пару имён, сохранить
безопасный класс наблюдаемого ответа и по-прежнему отвергать иной маршрут.
Качество извлечения и production adoption подтверждаются последующими этапами.

## Scope и non-goals

Scope: operator-only screening, проверка публичного каталога до POST, допуск
точной пары только в selected endpoint model, закрытая диагностика и offline
проверки с локальными HTTP/SQL fixtures. Внешних inference в этом этапе — 0.

Не входят: новый smoke/marker/разрешение расходов, автоповторы, fallback,
production model switch/activation/deployment, личные документы, новые модели,
изменение planner/prompt/schema/scorer/redaction/alias-плана и денежных лимитов,
автоматический settlement, повторная сверка S1.6. Исторические файлы неизменны.

## Текущее поведение и доказательства

`internal/assistant/llm_provider.go`: top-level model и metadata.requested
проверяются на exact alias; выбранный endpoint.Model также обязан равняться alias.
selected_model проверяется раньше provider/count. ModelPolicy не содержит
canonical identity; прежний PolicyVerifier подтверждает endpoint/model_id,
provider, возможности и цены, но не пару id→canonical_slug.

24 существующих контрпримера локального HTTP provider проходят на baseline
`0b98049`: canonical selected отвергается, product path остаётся строгим.
Сохранённый S1.6 report v2 immutable; он не является новым тестовым fixture
исходного ответа. Метаданные billing не подменяют wire evidence.

## Предметная модель и инварианты

1. Единственная пара:
   - alias `google/gemini-3.1-flash-lite`;
   - canonical `google/gemini-3.1-flash-lite-20260507`;
   - provider route `google-vertex/eu`, display provider `Google`.
   Никаких prefix/suffix/substring matching, lowercasing имён моделей,
   эвристик по endpoint.name или автоматического принятия нового canonical.
2. Перед любым screening POST обязателен GET `/api/v1/models`: ровно одна запись
   с exact id, её canonical_slug равен exact pin. Missing, duplicate, changed,
   malformed, non-200, redirect, timeout, oversized response — отказ до POST.
   Объём ≤8 MiB, timeout ≤5s; без ключа, личных данных и redirects.
   Результат действует ≤10 минут, future/zero timestamp отвергается; после
   истечения необходим новый GET. Cache key включает всю pinned identity;
   неуспешное обновление не разрешает использовать stale proof.
3. Proof: alias/canonical/provider, SHA-256 полученного каталога, verifiedAt.
   Только внутренний screening-путь создаёт и передаёт proof; внешний API,
   PDF, prompt, env и пользовательский DTO не могут его задать. Не менять
   общий ModelPolicy или публичный LLM/client контракт ради этой пары.
   Предпочтительно передавать proof как внутреннее поле отдельного screening
   запроса, без незащищённого mutable состояния общего provider.
4. Только при `screeningDocument=true`, exact ScreeningPolicy и свежем proof
   selected endpoint.Model может равняться alias **или** pinned canonical.
   Проверять proof до POST, включая случай alias-ответа. Top-level model и
   metadata.requested остаются exact alias. Вернувшийся другой/пустой selected
   model по-прежнему FAIL. LLMResponse.Model и billing сохраняют прежний alias;
   canonical не становится новой production policy.
5. Остальные guards сохраняются: direct, attempt1, non-BYOK, пустой pipeline,
   ровно один selected endpoint, прежние provider/display-provider проверки,
   известный конечный cost, token bounds, choices/tools/finish/JSON/schema.
   Запрос сохраняет only EU, ZDR, deny, no-fallback, reasoning disabled,
   прежние ceilings/reservation/shared budget/fees. Mapping каталога не
   доказывает региональность response; не вводить такое заявление.
6. Обычный provider, product statement reasoner/planner и production activation
   не принимают candidate/canonical даже при переданном screening proof.
   Нельзя условно включить послабление только по ModelPolicy.Model без
   независимого screening gate. Существующие accepted/rejected product cases
   и BillingKnown semantics совпадают с baseline.
7. Для screening диагностировать selected model закрытым enum:
   `not_observed`, `alias`, `pinned_canonical`, `other`, `empty`, `multiple`.
   `multiple` определяется по количеству selected до классификации имён;
   один selected с отсутствующим model — empty; при отсутствии выбранного
   endpoint — not_observed. Поле вычисляется из decoded wire до возврата guard.
   Не хранить и не печатать произвольные model/provider строки, raw envelope,
   headers, content, key или provider error. Новый enum не заменяет reason code,
   не подтверждает cost и сам по себе не разрешает settlement/activation.
8. Отдельная версия screening report `statement.screening.v3.gemini31-identity`;
   только screening получает optional proof/identity evidence, product report
   и его digest при отсутствии новых полей сохраняются. Report хранит proof
   (при наличии), закрытую классификацию каждого ответа и прежний private
   correlation/биллинг. Для каждого подтверждённого call сохранять локальное
   observedAt и использованный proof: verifiedAt≤observedAt≤verifiedAt+10min;
   каталог digest — ровно 64 lowercase hex. Не подменять эту проверку возрастом
   отчёта при его последующем чтении; будущие timestamps относительно окончания
   run отвергаются. Успешный report требует exact proof и
   допустимый класс для каждого подтверждённого call; несогласованный report
   отвергается. Public summary публикует только fixed codes/classes/booleans;
   report v3 не может включить production AI. Новую операцию запуска не добавлять.

## API, OpenAPI, UI

Публичные API/OpenAPI/TypeScript DTO и UI не меняются. Это отдельный операторский
протокол screening, не пользовательская возможность выбора модели.

## Данные, миграция и rollback

Миграций и финансовых записей нет. Существующие report v1/v2, markers и audits
не переписываются; исторические инспекторы/reconciliation остаются привязаны
к своим версиям и путям. Новый report v3 ещё не создаётся live. Rollback —
возврат implementation-коммита, без конфигурационных и database изменений.

## Ошибки и негативные сценарии

Невалидный proof/каталог прекращает работу до POST и не создаёт known billing.
Невалидный wire остаётся FAIL; валидный billing после всех route/cost guards
сохраняется при ошибке content, как прежде. Unknown cost остаётся uncertain,
не заменяется нулём. Нет автоматического GET generation/settlement/retry.
Если настоящая причина selected_model не canonical, классификация покажет
иной класс; расширять allowlist без нового решения запрещено.

## Критерии приёмки и исполняемые проверки

| ID | Подтверждение |
| --- | --- |
| I1 | Локальный HTTP catalog: exact mapping PASS; missing/duplicate/changed/invalid/status/redirect/timeout/oversize/future/zero/stale — FAIL, POST=0; успешное обновление stale proof обязательно |
| I2 | Локальный HTTP wire: screening принимает alias и pinned canonical только со свежим proof; другой canonical, пустое имя, canonical top/requested, другой provider, multiple/zero selected, retry/BYOK/pipeline/cost/token/content negatives сохраняют guards и точный BillingKnown; стандартный/product path canonical не принимает |
| I3 | Санитизация enum/summary, sentinel во всех произвольных строках; report v3/proof tampering и product activation reject; product report digest unchanged, old report/marker/recovery unchanged; известный billing при content error сохраняется |
| I4 | `make test`, `scripts/test-bank-statements.sh`, `make test-production`, `go vet ./cmd/... ./internal/...`, `git diff --check` PASS; проверка фактического request body и прежних 19/20/23 scorer; external inference=0 |

Offline PASS не подтверждает качество живой модели. Платный smoke возможен
только после отдельного конкретного решения о новом authorization/marker/cap;
старое разрешение S1.6 израсходовано. Интеграция принятого контракта и реализации в main остаётся отдельным
действием; основная копия сохраняется чистой.

## Решение владельца

Утверждены этот bounded контракт и offline реализация I1–I4, без платного smoke,
деплоя и переключения модели. Он меняет screening transport acceptance,
diagnostic/report profile относительно S1.5; прежнее принятие S1.6 recovery
такие изменения явно исключало. Решение получено 2026-10-21; acceptance commit `ad1b90b` предшествует реализации.
[Исполняемые подтверждения I1–I4](0055-s17-evidence.md): локальные проверки PASS.
Интеграция, paid smoke, deployment и adoption не выполнялись.
