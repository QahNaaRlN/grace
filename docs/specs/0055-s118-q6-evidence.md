# S1.18 / Q6 — SUCCESS read-only diagnosis,6/6 VERIFIED

- Контракт: [accepted S1.18](0055-s118-saved-quality-diagnosis.md),
  [отдельное разрешение](0055-s118-q6-run-decision.md),
  [pre-dispatch receipt](0055-s118-q6-execution-receipt.md).
- Владелец прямо разрешил «Разрешаю один Q6» на exact source/ref и один fixed-host
  SSH read с закрытым stdout через CI. Receipt опубликован до dispatch коммитом
  `882d64e86cf67aee63508aa54e8b71b974b4de0f`.
- Source `5fdd0f93380b72f6f3bdad03715f9a83744cace5`, frozen ref
  `codex/statement-s118-q6-source`; exact-source CI39945454718 SUCCESS5/5
  повторно проверен до dispatch. Source интегрирован №104, tree=merge438256b.
- **Один** [run39949603261](https://github.com/QahNaaRlN/grace/actions/runs/39949603261),
  eventworkflow_dispatch, создан2026-10-23 11:56:07 UTC / 16:56:07 +05:00.
- [Job120400553356](https://github.com/QahNaaRlN/grace/actions/runs/39949603261/job/120400553356)
  11:56:11–11:57:31 UTC / 16:56:11–16:57:31 +05:00, SUCCESS.
  Run completedSUCCESS, updated11:57:35 UTC. Только Q6 выполнен; остальные12
  jobs skipped, включая deploy/paid/inspect/reconciliation/legacy diagnostics.
- **EXECUTED / read-only diagnosis SUCCESS; permission CONSUMED.**
  Ровно один dispatch; retry/repair не выполнялись и не разрешены.

## Наблюдаемый результат — безопасные counts/status

Go/Python strict schema повторно подтвердила только closed stdout разрешённого
job. ContextVerified=true, artifactsUnchanged=true, providerCalls0,
persistentWritesRequested0. Все6 tuples VERIFIED;22 failed scorer checks и23
reason entries воспроизведены,15 public unknown получили verified reason mapping.
UNVERIFIED checks/reasons0. Это SUCCESS диагностики; **качество E6 остаётся FAIL**,
ни одна модель этим результатом не квалифицирована.

| Tuple | Candidate / fixture | Status | Verified failed checks | Verified reason entries | Ранее public unknown | Unmapped count |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | A / columns | VERIFIED | 2 | 2 | 1 | 6 |
| 2 | B / columns | VERIFIED | 2 | 2 | 1 | 7 |
| 3 | A / cards | VERIFIED | 6 | 2 | 1 | 1 |
| 4 | B / cards | VERIFIED | 2 | 7 | 6 | 10 |
| 5 | A / turnover-roles-ru | VERIFIED | 6 | 5 | 3 | не запрашивалось Q2 |
| 6 | B / turnover-roles-ru | VERIFIED | 4 | 5 | 3 | не запрашивалось Q2 |

[Public summary](0055-s118-q6-public-summary.json) содержит только safe status/counts
и source/run/job references, не позиции/IDs/aliases/proposals, даты, суммы,
source/model/service text, owner/reservation/generation/provider identifiers.
Закрытые witnesses не копируются в Git/PR/Actions artifacts.

Подтверждённые aggregate counts:

- Coverage: actual unmapped membership6/7/1/10 соответственно tuples1–4;
  source_coverage_incomplete4 и unmapped_spans4 reason entries. Это не разрешение
  считать пропущенные spans service или автоматически игнорировать их.
- A/cards:4 clock checks —4 date-part matches и4 UTC matches, но4 time-part
  differs и4 precision differs. Это наблюдаемые comparisons с pinned public
  golden, не объяснение рассуждения модели и не новая шкала scorer.
- Overlapping_evidence13 entries в tuples4–6; invalid_source_rows2 entries
  в RU tuples5–6. Counts включают отдельные row/blocker записи одной причины;
  **13 не объявляется13 независимыми дефектами**.
- A/RU:2 printed amount comparisons differ при2 matching currency comparisons.
  Actual values/selected roles/positions не публикуются; неверный financial role
  не выводится из одного numerical mismatch. B/RU не имел этих printed checks
  среди failures; row issues/turnover/blockers остаются FAIL.

## Неизменность и границы

Verifier проверил fixed E6 report и marker0600/O_NOFOLLOW, before/after
inode/hash/size/mode/mtime совпали. Artifact source остаётся24f7322;
verifier source5fdd0f9 — отдельный идентификатор. Сам private report с VPS не
скачивался, raw proposal/service/source data не публиковались.

Provider inference и provider metadata GET0; SQL/API/Docker/metering на VPS0,
reserve/settle/budget/fees/ledger/financial writes0. Нет chmod/install/config/
image/flag/allowance changes или новых health probes; OS/SSH audit logs не
считаются запретными writes verifier. Runtime health отдельно не измерялся.
Исторический расход E6 не перечитывался и не пересчитывался; old quality FAIL
и consumed E6 permission сохранены. Старые P6/D5 payloads не читались.

Frozen source/ref до dispatch и после анализа clean5fdd0f9; документы не двигают
его SHA. Временный local Python import cache удалён после анализа; tracked source
не менялся. Primary main438256b остаётся clean/mode0700. Никакой deployment.

## Фактический runtime manifest

Run содержит ровно1 artifact `statement-s118-manifest` (id12498874344), в нём
только SHA256SUMS и SOURCE_SHA. Диагностического payload/witness artifacts нет.
Проверен [фактический SHA256SUMS](0055-s118-q6-runtime-SHA256SUMS), digest manifest:
`89ed0b36bfb33bcf33152eaf412551f8d63db1a2ef412b3e3e22813fe86bdc98`.

Context/SOURCE_SHA/bootstrap/wrapper hashes совпали с local readiness4/4.
Фактический verifier hash `803906e4e495683849b4b07f9e7ceda8b3f4c552e9d92ab3ce0f21b6313d8c67`
отличается от локальной сборки. Byte-identity до запуска не заявлялась и не была
условием approval: разрешена сборка exact source в проверяемом CI job с полным
manifest до SSH. Runtime manifest зафиксирован отдельно, source не подменялся.
Причина различия binary fingerprints здесь не утверждается.

## Исполняемая проверка evidence

- До dispatch: receipt published, local/remote ref/SHA, ancestry/tree equality,
  canonical exact-source CI5/5; existing runs0; actual logical inputs выбирают
  только Q6. После одного запроса — один run с тем же SHA/ref/event.
- Final metadata: jobSUCCESS и12 skipped, все Q6 stages SUCCESS, exact SHA/ref.
- Из разрешённого job stdout извлечена ровно1 closed JSON projection; повторная
  Go/Python strict validation на source5fdd0f9 PASS, safe counts пересчитаны без
  копирования witnesses. Неуспешный process stdout/stderr не переносился в evidence.
- Artifact API и скачанный manifest подтвердили только2 manifest/source files,
 5 expected hash entries и exact source; wrapper/context fingerprints checked.
- `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` — PASS, exit0:
  Go,88 Web файлов/334 tests, lint/FSD/Prettier/build/offline Jev.
- `git diff --check` и документные ссылки — PASS. Runtime diff0.

Закрытый job log и projection остаются вне Git в `/tmp`,
оба файла имеют permissions0600. Публично сохранены только safe summary и manifest fingerprint.

## Следующий возможный пакет

Отдельный bounded offline контракт для причин coverage/clock selection/overlap,
с неизменными financial guards и causal replay/counterexamples. Точный scope и
вариант ремонта ещё не приняты. Q6 не разрешает prompt/guard repair, новый paid
run, смену модели или SPEC-0056 registry; эти действия здесь не выполнялись.
