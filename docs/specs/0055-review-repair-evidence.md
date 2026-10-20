# SPEC-0055 review repair — implementation evidence

Дата: 2026-10-20 (Asia/Yekaterinburg). Основание:
[accepted контракт](0055-review-repair.md), прямое утверждение владельца «да, утверждаю».
Worktree `statement-review-repair`, branch `codex/statement-review-repair`,
база main `d0c109d`. Production release **`ec15861`**, [PR #76](https://github.com/QahNaaRlN/grace/pull/76),
[deploy 39424505774](https://github.com/QahNaaRlN/grace/actions/runs/39424505774) PASS. Это executor evidence, не owner functional acceptance.

## Что реализовано

- Migration 000029: nullable private snapshot в statement_sources, точечный
  column grant, удаление через общий original/layout lifecycle и published guard.
  Down отказывается уничтожать сохранённую mapping, пустая migration обратима.
- Canonical Proposal + layout hash + origin + draft revision + точные Go coverage
  IDs сохраняются атомарно с Review/records. Чтение проверяет schema/hash/coverage.
  `complete` означает сохранённость разметки и допускает видимые financial blockers.
- Legacy projection восстанавливает поля из exact evidence, сохраняет строки,
  не угадывает service. Известное unmapped и неизвестное unclassified раздельны;
  broken evidence/hash/layout не используется как доказательство.
- API draftMapping optional, TS types сгенерированы из OpenAPI. UI показывает
  выбранные и рассчитанные итоги, selected source context, поиск/пагинацию
  непривязанных/неклассифицированных fragments и предупреждения legacy.
- Редактор открывает текущую разметку, сохраняет все service groups при точечной
  правке, поддерживает отмену и предупреждает о сбросе решений/проверок. После
  revision conflict правки остаются, save требует явного review актуального
  черновика; автоматической перезаписи свежей revision нет.
- Исправлен выявленный дефект coverage bookkeeping: отвергнутая из-за overlap
  service-группа не должна частично помечать свободные fragments использованными.
  Денежная арифметика, model/prompt/redaction/consent/лимиты не менялись.

## Критерии и исполняемое evidence

| ID | Состояние | Проверка |
| --- | --- | --- |
| R1 | PASS локально | `go test ./internal/statementdata -run TestMappingSplitTotalsAndRoundTrip -v`; `Reconciliation.test.tsx`: отображение server totals, missing != zero, source page 2 |
| R2 | PASS локально | `TestStatementMappingSnapshotLifecycle`: exact manual round-trip/replay, rollback; `internal/statementextract/integration_test.go`: actual orchestration fake-provider mapping origin=model; blocked review сохраняется |
| R3 | PASS локально | `TestMappingLegacyDoesNotGuessService`, `TestMappingUnavailableAndTampering`; SQL legacy GET не меняет revision, unmapped=null и unclassified отдельно |
| R4 | PASS локально | `StatementManualDraft.test.tsx`: текущие rows/service сохранены, изменение одного header, отмена без POST, legacy/unavailable; `MappingCoverage` UI: пагинация без потери последнего fragment |
| R5 | PASS локально | `TestStatementMappingSnapshotLifecycle`, existing HTTP/isolation/confirmation suite, strengthened `TestStatementPartialConfirmationAndDirectSQLCleanup`: stale/replay/reset, no postings до confirm, published API/SQL guard; web conflict требует review |
| R6 | PASS локально | SQL direct document tombstone и expiry очищают snapshot/layout; deleted API не выдаёт Proposal; audit не содержит synthetic service reason; существующие cross-owner и restore cases проходят |
| R7 | PASS локально | API generation, make test (88 files /334 web tests), full integration, local production suite, final affected bank suite, go vet и diff check PASS |
| R8 | PASS agent browser | После deploy: actual header/source context, legacy restoration, cancel, explicit manual correction, full mapping persistence/reload, unchanged row evidence/normalization, zero unmapped и blocked negative preview; owner financial acceptance pending |

Synthetic split-total regression проверяет поведение Go и UI, но не утверждает,
что причина исторического частного FAIL была пропущенным соседним префиксом.
Read-only diagnostic 39414784421 эту узкую гипотезу не подтвердил.

## Команды и журналы

- `make generate-api` — PASS, `/tmp/grace-review-generate.log`.
- `go test ./internal/statementdata ./internal/imports ./internal/statementextract`
  — PASS, `/tmp/grace-review-go-final.log`.
- Узкие UI regressions — 5/5 PASS; дополнительная source-page regression 2/2 PASS.
- `./scripts/test-bank-statements.sh` — PASS после published guard,
  `/tmp/grace-review-bank-complete.log`; real helper/fake-provider 84 layout cases,
  activation synthetic cases и PostgreSQL/restore guards. Provider calls=0.
- `make test-production` — PASS, `/tmp/grace-review-production-tests.log`:
  локальные activation/restore/rollout/deployment/offsite synthetic проверки,
  не production deployment и не платный smoke.
- `make test` — PASS, `/tmp/grace-review-tests-complete.log` (88 files /334 web tests, Go, lint/FSD/format/build, offline Jev).
- `make test-integration` — PASS, `/tmp/grace-review-integration.log`. После финальной SQL/integrity защиты повторён весь затронутый bank suite: `/tmp/grace-review-bank-complete.log` PASS.
- `go vet ./cmd/... ./internal/...` — PASS (также повторяется после make test),
  `/tmp/grace-review-vet.log`.
- `git diff --check` — PASS.

Первый общий make test остановился на React `set-state-in-effect` в SourceText.
Правило не отключалось: focus page теперь выводится из selection и явного page
choice, без setState в effect; targeted lint/FSD/format и UI regression пройдены.
Второй общий прогон выявил TypeScript ошибку только в тестовом getByRole options (`exact` не входит в Testing Library API); вызов исправлен. Окончательный полный make test PASS; правила и проверки не ослаблялись. Исторические model FAIL/activation evidence
сохранены; качество новых банков, AC12/Jev/cascade и owner acceptance не заявляются.


## Финальная совместимость и границы

Snapshot дополнительно сверяется с текущими Review.Rows/Header. SQL UPDATE review
без нового snapshot (в том числе byte-identical review от прежнего приложения
при rollback) инвалидирует snapshot и включает честную legacy projection.
Patch отметок владельца/retention его сохраняет. Эти случаи проверены targeted
Go + полным bank integration после финальных изменений; остальные integration
и production scripts успешно пройдены общими прогонами.

Локальная стадия завершилась R1–R7 PASS; на ней R8 ещё не выполнялся.
Публикация, release и agent browser R8 подтверждены ниже. Provider calls=0;
private corpus не добавлялся. Owner financial acceptance не подменяется
локальными/CI/browser PASS.


## PR и release gate — 2026-10-20

- Implementation head `52abb11e78d36e772b37cabb15502fde8cbbc445`.
- [PR #76](https://github.com/QahNaaRlN/grace/pull/76) merged,
  `ec1586100819d1bd77902e5a5d94d508cdc984af`, 2026-10-20 12:34:41 UTC.
- [PR CI 39422108502](https://github.com/QahNaaRlN/grace/actions/runs/39422108502)
  — все 5 jobs PASS. Merge tree совпадает с проверенным feature tree.
- [Main CI 39423345963](https://github.com/QahNaaRlN/grace/actions/runs/39423345963)
  — все 5 jobs PASS.
- [Delivery 39424344690](https://github.com/QahNaaRlN/grace/actions/runs/39424344690)
  — source guard и три published images PASS для exact `ec15861`.
- [Deploy 39424505774](https://github.com/QahNaaRlN/grace/actions/runs/39424505774)
  — PASS, statement_action=preserve; verified pre-release backup, migration29,
  helper/API health и public/auth checks. Healthy release зафиксирован
  2026-10-20 **12:44:58 UTC /17:44:58 +05:00**.
- [Read-only runtime 39425981524](https://github.com/QahNaaRlN/grace/actions/runs/39425981524)
  — PASS, Statement AI enabled и current-policy readiness подтверждены
  **12:54:49 UTC /17:54:49 +05:00**. Paid smoke/inference не запускались.
- Владелец разрешил PR, выпуск и повторный browser test поручением «приступить».
  Statement action при deploy — preserve; paid qualification/inference не запускаются.


## R8 — реальный браузерный review после выпуска

Agent-run проверка выполнена в существующей авторизованной вкладке, без нового
upload, consent или model call. UI показал причину retained header FAIL:
банковские реквизиты (БИК/КПП), включая их подписи, были выбраны как обороты.
Это ошибка выбора финансовой роли; прежняя узкая гипотеза пропущенного соседнего
числового префикса остаётся неподтверждённой. Автоматическое качество модели этим
пакетом не исправлено и full corpus qualification не объявляется завершённой.

Редактор восстановил все прежние строки из evidence, честно показал отсутствующую
legacy service classification. Реальная несохранённая правка отменена; повторное
открытие вернуло исходную selection без потери строк. Через UI исправлены только
оба header turnover и их label selections по печатным итогам. Остальные поля,
доступный остаток и отсутствующие book balances сохранены.

Каждый оставшийся source fragment просмотрен в контексте: document/bank metadata,
период документа, column headings, instrument identifiers, legal footer/signature,
page markers и repeated currencies. Затем они явно размечены service с причинами;
нет автоматического complement-to-service переноса. Сохранён один manual draft,
с причиной, явно называющей agent review и отсутствие financial confirmation.

После save и reload: turnover matched, exact unmapped=0, полный snapshot и service
reasons сохраняются; legacy warning отсутствует. Все прежние row evidence и
normalized records целиком совпали с независимой проверкой до выпуска.
Отрицательный preview для нерешённой строки: 0 новых операций /0 связей /
0 исключений, confirmation disabled, duplicate и decision guards сохранены.
Owner attestations ролей/страниц остаются unchecked, финансовое подтверждение
не выполнялось. Это **PASS review-repair**, не завершённый финансовый импорт
и не owner functional acceptance.

Приватные UI/PDF evidence и screenshot остаются локально, не в Git/CI. В публичный
отчёт не включены денежные значения, source IDs, реквизиты пользователя или текст
документа. Statement AI enabled сохранён; future model/corpus/Jev работа отдельно.
