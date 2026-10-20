# SPEC-0055 / S1.7: подтверждения точной идентичности модели

Дата: 2026-10-21, Asia/Yekaterinburg.
[Принятый контракт](0055-s17-selected-model-identity.md), acceptance `ad1b90b`
предшествует коду `abf321eea34489c1dba050f55ab0777f8c40886b`.
Владелец утвердил S1.7 и offline-реализацию, без inference/деплоя/смены модели.
[PR №79](https://github.com/QahNaaRlN/grace/pull/79) продолжает текущий review.

| ID | Исполняемое подтверждение |
| --- | --- |
| I1 | `TestScreeningCatalogProofAndRefresh`: exact mapping, missing/duplicate/changed/invalid/status/redirect/timeout/oversize; stale/future/zero proof обновляется, неудачное обновление удаляет stale proof. Настоящий HTTP302 не следует на другой URL; GET не содержит Authorization, ограничен 5s/8MiB. `TestScreeningCatalogFailureStopsReasonerBeforeInference` проверяет именно reasoner: ошибка каталога, billing=nil, POST=0. `TestScreeningProofRejectsBeforePOST`: nil/stale/future/zero/wrong pair/provider/digest/alias/policy — POST=0. `TestScreeningProofConcurrentCacheCopies`: 32 goroutines, один GET, возвращаются независимые копии. Race PASS |
| I2 | `TestScreeningCanonicalKeepsAllRouteAndContentGuards`: alias/canonical PASS; other/empty/multiple/none/provider/display/top/requested/retry/BYOK/pipeline/cost/token/content сохраняют точные error/class/BillingKnown. Проверен фактический request body: only EU, ZDR, deny, no fallback, reasoning disabled, output16000. Прежние 24 сценария `TestScreeningFailureDiagnosisSeparatesEnvelopeRouteAndContent` сохранены; единственная намеренная разница — canonical selected в screening со свежим proof; product остаётся строгим. `TestScreeningProofDoesNotOpenStandardProvider` дополнительно передаёт правильный candidate proof обычному provider: canonical той же модели отвергается; product statement останавливается ещё до POST. `TestScreeningTransportIsExplicitAndPinned` проверяет передачу proof reasoner→provider→billing и сохранение известного расхода при невалидном JSON |
| I3 | `TestScreeningIdentityReportTampering`: nil/wrong pair/provider/digest/stale/future/before-run/after-end/other/multiple/missing-end/legacy-version отвергаются даже после пересчёта digest. Чтение спустя 11min сохраняет корректную историческую свежесть proof на observedAt. `TestProductReportWireKeepsPreviousShape`: product digest равен прежней wire-форме без S1.7 полей; screening evidence не разрешает activation. Санитизация route/diagnostic и CLI summary проверяют все закрытые классы и PRIVATE-sentinels; новые optional поля отсутствуют в обычном product report |
| I4 | `scripts/test-bank-statements.sh` exit0: настоящий pinned helper + offline fake-provider columns19/19, cards20/20, RU23/23 и negative RU, SQL lifecycle/restore/отказ destructive down. `make test-production` exit0: старые screening/activation/restore/deploy/offsite mocks, workflow512, privacy/reconciliation PASS. `go test -race ./internal/assistant -run TestScreening -count=1`, focused report/extract/CLI tests, `go vet ./cmd/... ./internal/...`, `git diff --check` PASS. `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0: Go, 88 web files / 334 tests, lint/FSD/форматирование/сборка/Jev PASS |

## Реализация

Отдельный verifier получает публичный каталог без ключа, проверяет ровно одну
точную id→canonical_slug пару, сохраняет SHA-256 и UTC timestamp. Cache защищён
mutex, выдаёт копии; proof свежий ≤10min, не может поступить из API/PDF/DTO/env.
Внутреннее поле screening request передаёт proof в provider. До POST повторно
проверяются proof и pinned policy. Послабление действует только для selected
endpoint model в screening; top-level/requested alias и прочие guards сохранены.

Закрытая классификация вычисляется после decode перед возвратом wire guard.
Произвольные model/provider строки не сохраняются. Report v3 содержит route
proof, класс и local observedAt каждого billed call и screeningFinishedAt;
историческая свежесть проверяется на время вызова, не на время чтения отчёта.
Product report не содержит новых optional полей; его прежний digest сохранён.

## Неизменённые границы

`git diff --exit-code 0b98049 -- api web migrations internal/statementdata internal/statementextract/plan.go internal/statementextract/proposal.go deployments .github scripts/ops-reconcile-statement-screening.py scripts/ops-reconcile-statement-screening-s16.py scripts/research/inspect-statement-screening-s16.py` exit0.
Нет нового runner/marker/authorization, смены модели/flags/config, миграций,
пользовательского API и финансовых записей. Основной checkout clean.

S1.6 report/marker/recovery остаются immutable/version2 и не используются как
replay утраченного envelope. S1.6 FAIL selected_model и quality UNVERIFIED
сохранены. Canonical mismatch — проверенная гипотеза совместимости, а не
установленная строка исторического ответа. Никаких inference/нового settlement
или production операций в этом этапе не выполнялось.

[Диагностика и публичные источники](../research/statement-quality/s17-selected-model-identity.md):
GET catalogue/endpoints без ключа, inference=0. Реальные PDF в тестах — только
репозиторные синтетические fixtures; provider подменён, helper network=none.

Локальные журналы: `/tmp/grace-s17-unit.log`, `/tmp/grace-s17-race.log`,
`/tmp/grace-s17-report-tests.log`, `/tmp/grace-s17-vet.log`,
`/tmp/grace-s17-bank.log`, `/tmp/grace-s17-production.log`,
`/tmp/grace-s17-make.log`. Финальные дополнительные регрессии reasoner/summary/product isolation выполнены отдельным focused/race run; повтор большого неизменённого web suite не потребовался.

После локальных gates — review/интеграция. Новый ограниченный paid smoke требует
отдельного нового решения о marker, SHA/CI и cap; здесь не разрешён и не запущен.
Offline PASS не подтверждает живое качество модели или production adoption.
