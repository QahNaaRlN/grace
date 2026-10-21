# SPEC-0055 / S1.14: контролируемое сравнение с сохранённым alias-proposal

- Статус: accepted; implementation_status: implemented (offline; local gates PASS; exact-head CI pending); live_status: not_authorized.
- Автор: Codex; дата: 2026-10-21, Asia/Yekaterinburg.
- Acceptance owner: владелец Grace; acceptance record: 2026-10-21, прямое «принимаю» на offline P1–P5; P6 не разрешён.
- Основание: «работаем дальше» после предложения подготовить отдельную спецификацию
  контролируемого повторного эксперимента. P1–P5 приняты отдельным последующим решением до кода.
- База: main `d63c16fe69e8edad22e869dfe5b8f58ee40ec01f`, PR №89.
- Spec branch: `codex/statement-observed-comparison-spec`.
- Implementation branch: `codex/statement-observed-comparison`; code commit
  `9db80afc57c42fd283e1b6daf46d9e1efe027d01`; [P1–P5 evidence](0055-s114-evidence.md).
- Разрешённый live source SHA: отсутствует.

## Проблема и пользовательский результат

Единственный [S1.11 C6](0055-s111-c6-evidence.md) остановился после A/columns:
`qualification_protocol_failed`, остальные5 tuples not_sent. [S1.12 R5](0055-s112-evidence.md)
установил stage canonicalization, но exact guard UNVERIFIED: raw alias-proposal
не сохранился. Известный расход1995 USD micros учтён; прежнее разрешение consumed.

[S1.13](0055-s113-protocol-observability.md) интегрирован PR №89 и добавил private
bounded snapshot до canonicalization, closed diagnostic и offline replay.
[Post-merge evidence](0055-pr89-postmerge.md): local gates, CI5/5, Delivery PASS.
Это готовность наблюдаемости, а не доказательство качества реальных моделей.

Результат S1.14 — одна отдельно разрешённая новая synthetic campaign на прежних
A/B и columns/cards/RU. При повторении canonicalization FAIL владелец получает
сохранённый bounded proposal и проверяемый guard либо явное UNVERIFIED. Если
протокол проходит, результат даёт ограниченное сравнение quality/cost/latency.
Старая ошибка этим запуском не восстанавливается и не становится PASS.

## Scope

**Offline P1–P5**, только после принятия: новый фиксированный campaign context,
отдельный one-shot runner/workflow mode, strict report/inspector/replay для нового
разрешения, causal lifecycle/privacy/SQL/HTTP tests и evidence.
Допустимые runtime области: comparison runner/validation в statementqualification,
operator CLI/meter, production operator scripts, comparison inspector/replay и
workflow isolation. Общие validators/reasoners используются повторно без копии
canonicalizer и без изменения финансовых правил.

**Live P6**, только после отдельного решения: preflight и ровно одна campaign
на принятом exact implementation SHA, до6POST/$0.2085 provider, с сохранением
закрытой сводки, приватного artifact, billing и image/health evidence.
Принятие P1–P5 не разрешает P6; merge спецификации/реализации не является dispatch.

## Non-goals

Исправление prompt/schema/planner/aliases/redaction/scorer/guards, model/route/price
замена, новый corpus, retry/fallback/repair, личные PDF/owner drafts, product
API/Web/OpenAPI, production policy/flags/model switch/deploy, financial writes,
бюджет/fees increase/reset, удаление marker или изменение старых reports/ledger.
Новых generation metadata GET/reconciliation автоматически нет. SPEC-0056 и
production qualification не входят. В документационном и offline scope нет
provider metadata/inference, SSH/VPS, dispatch или чтения live private artifacts.

## Текущее поведение и доказательства

На d63c16f `ComparisonAuthorization`, runner и inspector жёстко закреплены за
`s111-comparison-20261021`. Старый marker consumed; повтор существующего режима
должен по-прежнему отказать. S1.13 writer уже вызывает `RunObservedComparison`,
но новый source SHA сам по себе не создаёт разрешение.

`ValidateComparisonReport` проверяет прежнюю authorization; нельзя просто
подменить константу, переписать JSON до digest validation или ослабить validation.
Legacy bare v1, S1.12 fixed reader, existing summary и offline tests должны
сохранить прежний контракт. Exact implementation/source и live runtime state
S1.14 пока UNVERIFIED; свежая доступность/цены провайдеров не проверялись.

## Предметная модель и инварианты

### P1. Фиксированный новый context и совместимость

Кандидат новой единственной authorization: `s114-comparison-20261021`.
Она не является разрешением, credential или новым бюджетом. Никаких произвольных
campaign IDs, user-supplied report paths или выбора модели через dispatch.

Legacy entry points сохраняют старую authorization и semantics. Новые S1.14
entry points передают один фиксированный context в общую prepare/run/validation
логику. JSON validation проверяет исходный digest и exact context; report нельзя
нормализовать в старую authorization перед проверкой. Некорректная комбинация
старого/нового identity, artifact version, marker или reservation отвергается.

Новый private envelope `statement.comparison.observed.s114.v1` содержит прежние
report fields/semantics и observations S1.13; вложенный report сохраняет schema
version `statement.comparison.v1.grounding`, но authorization равна только S1.14.
Оба digest включают фактические значения. Новый decoder явно выбирает фиксированный
S1.14 context по новой outer version. Bare v1 с новой authorization не принимается;
legacy validators/decoder остаются привязаны к S1.11. Snapshot bounds, whole-object
omission, diagnostics priority, stages и privacy полностью сохраняются из S1.13.
Replay поддерживает новый envelope через тот же canonicalizer и context validation.

### P2. Однократность и изоляция operator mode

Отдельные fixed marker/report в `/opt/grace/shared`:
`statement-comparison-s114-comparison-20261021.started` и
`statement-comparison-s114-comparison-20261021.json`, private0600.
Preflight/exit files также принадлежат только этому identity. Старые artifacts
не удаляются, не переименовываются и не используются для освобождения допуска.
Прежний общий exclusive lock сохраняется между всеми screening/comparison modes.
Existence/symlink/orphan/duplicate checks выполняются до нового POST; marker
создаётся no-clobber до первой возможной отправки и остаётся после любого FAIL.
Неиспользованные tuples не переносятся в следующую campaign.

Предлагаемый default-false workflow input `statement_comparison_s114_only` и
отдельный `comparison_s114_authorized_sha`. Ровно один mode допустим; новые
предикаты исключают deploy, inspect, reconcile, enable/disable и все старые
screening/comparison/diagnosis jobs. Для ambiguous inputs — отказ без SSH/POST.
`statement_action=preserve`. Exact40hex source/ref/SHA проверяется до staging,
полный bundle manifest и все5 CI jobs должны соответствовать тому же SHA.
Не переиспользовать разрешение или inputs старого C6.

### P3. Прежние пары, порядок и ceilings

| ID | Request alias → exact canonical | Endpoint / provider | Max/document USD micros |
| --- | --- | --- | --- |
| A | google/gemini-3.1-flash-lite → google/gemini-3.1-flash-lite-20260507 | google-vertex/eu / Google | 53900 |
| B | inception/mercury-3 → inception/mercury-3-20261019 | inception / Inception | 15600 |

Это унаследованные pins, не утверждение о свежем каталоге. P6 preflight проверяет
fresh catalog/endpoint/ZDR proof, цены не выше прежних ceilings и exact identity
обеих пар. Любое изменение требует возврата владельцу, не подстановки маршрута.

Порядок: A-columns → B-columns → A-cards → B-cards → A-turnover-roles-ru →
B-turnover-roles-ru. Только3 public synthetic PDFs с прежними exact hashes из
S1.11, те же planner/prompt/schema/redaction/aliases/S1.10 guards/scorer19/20/23.
На tuple максимум1POST, input100000/output16000,120s/call; inference context900s,
settlement5s. До6POST, общий provider cap208500micros=$0.2085; A161700+B46800.
Прежний общий $10 ledger, фактический remaining, historical costs и fees сохраняются;
provider ceiling не объявляется fee-inclusive стоимостью. Inception допускается
только для public fixtures; EU residency всей цепочки не заявляется.

Known quality/grounding FAIL позволяет закончить оставшиеся заранее разрешённые
tuples. Protocol/transport/identity/privacy/cost/timeout, unknown billing,
settlement failure или shared block останавливают campaign. Порядок не меняется
ради получения B после protocol FAIL A. Все6plans/total reserve проверяются до
отправки; call_allowed перед каждым Complete. Никаких дополнительных пробных POST.

### P4. Artifact, billing и read-only объяснение

Private report пишется existing private writer, stdout/stderr/Actions получают
только закрытые enums/bools/counts и принятую числовую cost/latency сводку.
Alias strings/proposal/free text/UUID/key/generation IDs не попадают в публичные
логи, Actions artifacts или evidence Markdown. Partial/overflow/missing snapshot
не превращается в PASS и не разрешает новый вызов. Artifact validation и accounting
имеют разные результаты; известный расход не становится0 из-за ошибки storage.

Одна новая reservation: model=`statement-comparison.s114`, provider=`fixed-A-B`,
purpose=`statement`, прежние reserve/settle функции. Inspector требует точную
единственную reservation UUID из verified report и сравнивает amount/status/time;
никакого выбора последней строки. Legacy inspector ожидает S1.11 и не принимает
новый identity. Старые rows не меняются; uncertain блокирует дальнейшие paid actions.

В P6 включена только read-only диагностика **нового** fixed artifact после run:
проверить оба digest, summary, billing correlation и replay captured snapshots по
тем же bundled public PDFs/pinned helper. Разрешается staged exact-source replay
binary внутри operator execution, без model/key/config/SQL constructors у replay.
Никакого копирования raw snapshot в Actions. При недоступном replay — UNVERIFIED,
отдельный будущий recovery не запускается автоматически. Artifact/marker bytes,
mode и mtime до/после read-only inspection должны совпасть.

### P5. Offline evidence до live решения

Причинные tests проверяют observed Complete→capture→guard→writer→decode→inspector→
replay, legacy v1 compatibility и замкнутый S1.14 context; delivered-alias fakes,
не golden-created proposals. Проверить no-clobber повтор с другим source SHA,
symlinks/orphans, preflight failure, concurrent mode, failure after marker,
known/unknown billing и settlement, snapshot overflow, digest corruption и
privacy sentinel на каждом публичном exit. SQL tests — disposable integration DB.
Workflow matrix доказывает отсутствие соседних jobs и deploy при новом mode.
Real pinned helper подтверждает прежние fixture/plan/request hashes, fake scores
19/20/23 и offline replay6/6. Это не live quality evidence.

### P6. Отдельное решение об исполнении

После P1–P5 implementation review, integration и exact-source CI5/5 предъявить
владельцу конкретный immutable source SHA/ref и итоговый one-shot контракт.
Source d63c16f — baseline, **не** разрешённый executable S1.14. До этого live source
отсутствует, dispatch/metadata/SSH/VPS=0. Решение о paid run принимается отдельно.

После такого решения preflight проверяет global AI/key, actual image/health,
helper, единственного active admin, shared remaining/uncertain, exact bundle и
fresh proofs. Отказ не даёт права менять среду или повторять платный smoke.
После run записать tuple table, stage/guard/capture/replay, known cost/Complete
latency (включает proof/network/decode, не pure generation), reservation state и
image/health до/после. Без n достаточного для статистики не заявлять p95, победителя
при incomplete или qualification всех банков. Production adoption отдельно.

## API и OpenAPI

Product HTTP/API/DTO не меняются, regeneration TypeScript не требуется.
Новая версия только private operator artifact и отдельный workflow input.

## Данные, миграция и rollback

Миграций/financial writes нет. Offline — только disposable test data.
После отдельного P6 — новая reservation и новые private artifacts в прежнем
storage/retention boundary; raw snapshots остаются private, без публичного upload.
Rollback отключает новый mode без удаления marker/report или сведений о расходе;
старые S1.11/S1.12 artifacts/readers и результаты остаются неизменными.

## UI

Пользовательский интерфейс и consent не меняются.

## Ошибки и негативные сценарии

Consumed marker при другом SHA, orphan/symlink, concurrent mode, wrong source,
missing/partial CI, changed metadata/prices/endpoint, no admin/key/helper/budget,
unknown bill, settlement failure, guard FAIL, capture overflow, corrupt digest,
missing report/replay и public sentinel leak дают закрытый отказ/UNVERIFIED.
Нельзя лечить их повтором вызова, удалением artifact, сменой модели или budget reset.

## Критерии приёмки

| ID | Исполняемое evidence после принятия |
| --- | --- |
| P1 | Новый fixed context/version; legacy fixtures остаются валидны; перекрёстные auth/version/digest combinations отвергаются настоящими Go decoders, без JSON repair |
| P2 | Shell/workflow matrix: no-clobber, same/different SHA, orphan/symlink, lock/concurrent modes; mixed inputs не запускают SSH/deploy/POST |
| P3 | HTTP/fake tests сохраняют exact policies, delivered input/schema, order<=6, прежние caps, continuation/stop, one reserve/settle и historical guards |
| P4 | Реальный private writer→Go summary→inspector→replay; exact SQL correlation, known/unknown/overflow/error paths, privacy sentinel и file immutability PASS |
| P5 | make test, make test-production, make test-integration, scripts/test-bank-statements.sh, affected race tests, go vet и diff check PASS; exact-head CI5/5 и критерий-за-критерием report |
| P6 | Только после отдельного approval: один run либо preflight refusal; exact source/run/job IDs, до6POST/$0.2085, честные scores/diagnostics/replay/cost/latency и billing/image/health evidence. FAIL/incomplete не скрываются |

## Исполняемые проверки

```sh
TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test
TMPDIR=/tmp make test-production
TMPDIR=/tmp make test-integration
TMPDIR=/tmp scripts/test-bank-statements.sh
go test -race ./internal/statementextract ./internal/statementqualification
go vet ./cmd/... ./internal/... ./scripts/research/s113-replay
git diff --check
```

Новые lifecycle/context/inspector/replay tests должны быть подключены к обязательным
gates, точные команды и результаты фиксируются implementation evidence. Эта
спецификация не содержит команды dispatch: executable SHA ещё не существует.

## Открытые решения

1. Offline P1–P5 приняты 2026-10-21 прямым «принимаю»; acceptance commit предшествует коду.
2. Live P6: NOT_AUTHORIZED. Потребует отдельного решения на exact реализованном SHA,
   фиксированный новый identity и предел до6POST/$0.2085 provider.
3. Свежая runtime/provider readiness: UNVERIFIED до отдельно разрешённого preflight.
   Несовместимость pins возвращает контракт на review, не расширяет scope.
