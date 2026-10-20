# SPEC-0055 review repair — implementation evidence

Дата: 2026-10-20 (Asia/Yekaterinburg). Основание:
[accepted контракт](0055-review-repair.md), прямое утверждение владельца «да, утверждаю».
Worktree `statement-review-repair`, branch `codex/statement-review-repair`,
база main `d0c109d`. Production release остаётся `7b2deea`; публикация и deploy
этого пакета не выполнялись. Это executor evidence, не owner functional acceptance.

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
| R8 | NOT_RUN | Требует отдельного release/deploy и повторного browser owner-test; private source не отправлялся модели и не включён в fixtures |

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

R1–R7 проверены локально; R8 NOT_RUN. Нет новых provider calls, private corpus,
конфигурационных/финансовых production changes, merge или deploy. Primary checkout
оставлен чистым. Diff находится в feature worktree; CI/release/owner acceptance
не подменяются локальными PASS.
