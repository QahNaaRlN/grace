# SPEC-0055 — owner review привязан к версии выписки

- Основание: принятая [SPEC-0055](0055-bank-statement-pdf-import.md), AC6 и AC10;
  технический контракт принят владельцем 2026-10-19. Это исправление существующего
  revision/review/409 поведения, не новый исследовательский pilot.
- База: main `84266bd72ac10d569099e922cb826a8a01330252`.
- Worktree/branch: `statement-import-readiness` / `codex/statement-import-readiness`.
- Дата проверки: 2026-10-23. Статус: verified locally, integration pending.

## Дефект и изменение

После materialization/regroup сервер сбрасывает rolesConfirmed и reviewedPages.
Форма сохраняла прежние useState flags, reason и отправляла их с новой revision
из props. Владелец мог повторно сохранить старые отметки для изменённой разметки.
Браузер до исправления: API revision3/roles=false/pages=[], форма checked=true,
кнопка «Сохранить проверку» активна. Financial writer при этом оставался закрыт.

Форма теперь хранит editingRevision и отправляет только её. При изменении source
она сохраняет ввод, показывает устаревание и блокирует submit. «Загрузить
актуальную проверку» загружает серверные retention/flags/pages и очищает прежнее
основание. Успешное собственное сохранение принимает возвращённую server revision.

Дополнительно обнаружен реальный conflict-cache дефект: app staleTime=30000,
а fetchQuery после409 мог вернуть прежние свежие cache data. Общий conflict path
owner/manual теперь использует staleTime=0 для принудительного GET. Ни money,
ни eligibility, ни scorer/client business rules не добавлены в React.
API/OpenAPI/DTO/schema/SQL/LLM prompt/model/runtime flags не меняются.

## AC и причинные проверки

| AC | Исполняемое evidence | Результат |
| --- | --- | --- |
| AC6, AC10: сброс проверки после regroup | `StatementOwnerReview.test.tsx`: source revision changes, save disabled/no PATCH; load current clears old flags/pages/reason; fresh review sends current editingRevision | RED до fix, GREEN после |
| AC10: сохранить ввод при409 | Тот же suite с настоящим mutation/query hook, app staleTime30000; forced GET, текст/галочки остаются видимыми, stale submit disabled | RED при старом fetchQuery, GREEN после |
| AC6, AC10: ручная разметка после409 | Existing `StatementManualDraft.test.tsx` переведён на app staleTime30000; selected spans/reason сохранены; retry после явной проверки использует revision2 | RED при старом fetchQuery, GREEN после |
| AC2: нет pre-confirm financial writes | Local PDF upload/manual/regroup/owner edit/real409, SQL counts events/postings/extraction calls/attempts =0/0/0/0 | PASS в указанном synthetic сценарии |
| AC10: настоящий браузер | Публичный columns.pdf, локальный pinned helper/API/PostgreSQL и UI, AI=false; source revision3→9, regroup/409/refresh/fresh save,375px и Tab | PASS указанного negative/recovery сценария |

Команда focused suite:

```sh
cd web
VITEST_MAX_WORKERS=1 npm test -- --run src/features/import/bank-statement/ui/StatementOwnerReview.test.tsx src/features/import/bank-statement/ui/StatementManualDraft.test.tsx src/features/import/bank-statement/ui/StatementAI.test.tsx
npx tsc -b
```

7/7 PASS, TypeScript PASS; `/tmp/grace-owner-review-warm-cache-green.log`,
`/tmp/grace-owner-review-typecheck.log`. Исходные RED logs:
`/tmp/grace-owner-review-red.log`, `/tmp/grace-owner-review-warm-cache-red.log`.

## Браузер: что фактически проверено

Только собственный localhost stack `grace-import-readiness-20261023`, bind127.0.0.1,
публичный synthetic columns.pdf; helper network=none, AI flags=false, keys пусты.
Upload выполнен формой. Намеренно сохранена неполная ручная разметка; пропущенные
суммы/строки остаются blockers. Это не успешный полноценный импорт всех строк.

До fix форма показывала старые checked roles/pages при сброшенном server review.
После fix тот же regroup показывает stale notice и disabled save. Явная загрузка
текущей проверки возвращает false/[] и пустое основание.
Реальный409 вызван конкурентной metadata правкой в этой же disposable БД.
В финальном image: конфликт через20.638s после загрузки (<30s app staleTime),
ввод сохранён, forced GET даёт новую revision, save disabled. После явной загрузки,
новых отметок и основания save успешен: revision9, roles=true, pages=[1,2],
publishedRows=0. SQL: financial_events/postings/extraction_calls/attempts=0/0/0/0.

Viewport375×812: innerWidth375, clientWidth=scrollWidth=360, overflow отсутствует;
Tab из recovery button переводит фокус в следующий combobox. В финальном image
новых console errors нет; прежний HydrateFallback warning остаётся.
Во время dev HMR возникала CSS resolve ошибка из-за host node_modules symlink;
повторено на пересобранном собственном image без bind mounts, дефект устранён.
Это исправление test environment, не изменение продукта.

## Общие gates и ограничения

Все итоговые gates PASS, exit0:

- `TMPDIR=<private /var/tmp> NODE_DISABLE_COMPILE_CACHE=1 VITEST_MAX_WORKERS=1 make test`:
  Go,89 Web files/336 tests, lint/FSD/Prettier/build/offline Jev;
  `/tmp/grace-owner-review-final-make-test.log`.
- `TMPDIR=/tmp make test-production`: `/tmp/grace-owner-review-production.log`.
- `TMPDIR=/tmp make test-integration`: `/tmp/grace-owner-review-integration.log`.
- `go vet ./cmd/... ./internal/...`: `/tmp/grace-owner-review-vet.log`.
- `git diff --check`; generated OpenAPI types unchanged.

Backend sources/scripts не менялись; полные SQL/helper/financial suites не
подменяются Web mocks. Это local verification, CI exact head проверяется отдельно.
Собственные browser tab, контейнеры, network и все3 named volumes удалены;
список ресурсов project label пуст. Private TMPDIR удалён. Read-only /tmp/.git,
production и чужие Docker проекты не менялись.

Первый make test с TMPDIR=/tmp остановлен privacy guard output_inside_repository:
в окружении существует посторонний read-only /tmp/.git. Он не изменялся;
повтор использует private /var/tmp. Ещё один прогон выявил ошибочные exact options
в новых Testing Library локаторах на стадии tsc; исправлено, typecheck PASS.

## Аудит общей цели (не заявление о готовности)

| Требование надёжного импорта | Текущая опора и оставшаяся работа |
| --- | --- |
| PDF/grounding/financial guards/atomic confirmation | Принятый SPEC0055 и реальные integration suites; PASS ограниченных fixture/assertions, не всех банков |
| Ручная корректировка и актуальная owner review | Этот дефект исправлен и проверен локально; review/интеграция/публикация ещё нужны |
| Качественный AI-черновик на согласованных layout | Не достигнуто: E6 unqualified, единственный S1.19 завершён REJECT; PR108 сохраняет этот факт |
| Объяснимый отказ и functional acceptance | Конкретную причину REJECT закрытый S1.19 summary не раскрывает; новый bounded diagnostic contract/разрешение и owner functional acceptance остаются открыты |

Новый paid run/reader/SSH/deploy этим исправлением не открыт. Frozen E6/Q6/S1.19
refs неизменны; private reports не читались. Общая цель остаётся **не достигнута**;
исправление закрывает конкретную ошибку актуальности review, не model qualification.
