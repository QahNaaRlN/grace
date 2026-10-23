# S1.18 / Q6 — предложение одного read-only запуска над saved E6

- Статус: **proposed / NOT_AUTHORIZED / NOT_DISPATCHED**.
- Дата: 2026-10-23, Asia/Yekaterinburg; acceptance owner: владелец Grace.
- Основание подготовки: «слил PR» после реализации Q1–Q5. Слияние №104
  разрешает подготовку следующего решения, но не заменяет отдельное approval Q6.
- Контракт: [принятая S1.18](0055-s118-saved-quality-diagnosis.md), Q1–Q5 integrated;
  [offline evidence](0055-s118-evidence.md), [post-merge №104](0055-pr104-postmerge.md).
- Предлагаемый verifier SHA: `5fdd0f93380b72f6f3bdad03715f9a83744cace5`.
- Frozen ref: `codex/statement-s118-q6-source`, local/remote exact SHA проверен.
- [Exact-source CI39945454718](https://github.com/QahNaaRlN/grace/actions/runs/39945454718):
  SUCCESS5/5; canonical full job/SHA validation PASS. Source интегрирован №104;
  tree source5fdd0f9 и merge438256b byte-identical. Это отдельный CI источника,
  не перенесённый PASS от спецификации или ещё выполняющегося merge workflow.
- Наличие/содержимое private artifact и его права на VPS: **UNVERIFIED**,
  до отдельного решения владельца не читаются.

## Конкретное предлагаемое действие

Разрешить **один** `workflow_dispatch` файла `deploy-production.yml` на exact
SHA/ref выше, с единственным выбранным job `diagnose-statement-s118` и
[точными логическими inputs](0055-s118-q6-inputs.json). Все paid/deploy/inspect/
reconciliation/legacy diagnostic modes=false. Offline actual workflow conditions
для этих inputs выбирают только Q6; полная matrix131072 PASS.

До dispatch опубликовать receipt с этим approval, source/ref и CI; повторно
проверить ref/SHA/ancestry/CI. Один dispatch потребляет разрешение независимо
от результата, включая preflight failure. Автоматического retry/repair нет.
Изменение SHA/ref/path/mode требует нового решения; запрещено менять источник
или выбирать иной mode после отказа.

Ожидаемый результат — `statement.saved-quality-diagnosis.s118.v1` с6 tuples:
exact replay/scorer equivalence, unmapped positions, разбор sanitization unknown,
clocks и printed/reconciled turnover. Каждый факт VERIFIED либо UNVERIFIED;
context/file integrity failure возвращает closed diagnostic_unavailable без
частичных witnesses. Historical E6 quality FAIL и permission CONSUMED сохраняются.

## Read-only граница

- Только host `grace-deploy@195.133.195.190`, pinned known_hosts.
- Только fixed `/opt/grace/shared/statement-comparison-s117-comparison-20261023.json`
  и `.started`. Marker artifact source остаётся
  `24f73223b9b3370ca78c41d34e89edca9a276488`; verifierSource5fdd0f9 — другое поле.
  Не переписывать marker/report/profile/digests, не читать старые P6/D5 payloads.
- Public contexts готовятся локально pinned Poppler25.04.0 из3 fixed PDFs,
  network=none. На VPS только Python `-B`, sealed memfd binary, env={},
  fixed O_NOFOLLOW regular0600 reads; before/after inode/hash/size/mode/mtime.
- Provider inference/catalog/endpoint/ZDR/generation GET=0, SQL/API/Docker/
  metering calls0. Нет reserve/settle/budget/fees/ledger/financial writes,
  model/flags/image/config/allowance изменений или новых health probes.
- Verifier/bootstrap persistentWritesRequested0; обычные OS/SSH audit logs
  этим не отрицаются. Никаких установок, chmod, новых remote reports/markers.
- Binary<=16MiB, context<=524288bytes, envelope<=32MiB, stdout<=65536bytes;
  remote45s/SSH75s. Failed stdout/stderr не публикуются, ошибок с raw text нет.
- Закрытые witnesses допустимы только в stdout отдельно разрешённого CI job.
  Не переносить payload в Git/PR/Actions artifacts. Artifact только SHA256SUMS и
  SOURCE_SHA; публичное evidence — safe status/counts и run/job references.
- Недоступность/timeout/UNVERIFIED не разрешает второй dispatch, восстановление
  snapshot по golden или новый платный E6. Scope ремонта остаётся отдельным.

## Локальный readiness manifest — без SSH

[SHA256SUMS всех5 файлов](0055-s118-q6-local-SHA256SUMS), digest самого manifest:
`beb301f3b79a0607b34e98228529b83393010a898372002e46b0ea099af3519e`.
Bundle сохранён вне Git в
`/tmp/grace-s118-q6-5fdd0f93380b72f6f3bdad03715f9a83744cace5/`.
В Git только fingerprint, binary/public context payload не добавлены.

- Toolchain: **go1.24.0 linux/amd64**, как в source CI Set up Go;
  локальный defaultgo1.27.3 не использован для readiness binary.
- Build: `CGO_ENABLED=0 go build -trimpath -ldflags '-X main.verifierSource=5fdd0f93380b72f6f3bdad03715f9a83744cace5'`.
- Binary10375878bytes, context21829bytes, подготовленный envelope13861411bytes;
  все caps PASS. Wrapper source/binary/context/SOURCE_SHA hash-check5/5 PASS.
- Настоящий local helper, verifier context и real-helper causal diagnosis PASS;
  source worktree/ref до/после clean5fdd0f9; SSH/provider calls0.

Это readiness manifest локальной сборки, не отчёт ещё не выполненного workflow.
Q6 job заново собирает exact source и до SSH создаёт/проверяет собственный полный
manifest binary/context/source/bootstrap/wrapper. Его фактические fingerprints
фиксируются отдельно в manifest artifact; byte-identity CI binary и локальной
сборки до этого не заявляется. Ни manifest, ни успешный offline preflight сами
по себе не разрешают SSH. Решение владельца относится к exact reviewed source,
fixed read и этой проверяемой процедуре сборки/manifest, а не к другому binary.

## Неполученное решение

Требуется отдельное прямое разрешение одного Q6 с SSH-чтением fixed saved E6
и закрытым результатом через CI на указанном source. «Слил PR»/принятие Q1–Q5
не считаются этим разрешением. До ответа Q6 NOT_AUTHORIZED / NOT_DISPATCHED.
