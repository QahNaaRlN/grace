# S1.17 E6 — проверка готовности после PR №100

Дата: 2026-10-23, Asia/Yekaterinburg.

- Владелец: «влил PR, переходим к E6». Это подтверждает приёмку и интеграцию
  offline реализации и ранее данное отдельное разрешение E6.
- Accepted contract: [S1.17](0055-s117-alias-comparison.md), E1–E5/E6 scopes сохранены.
- Offline реализация интегрирована [PR №100](0055-pr100-postmerge.md),
  main `7651668013db38c9d472d118e2542aae20a215c8`.
- **E6 NOT_DISPATCHED / readiness NOT_READY / permission UNUSED.**
- Immutable live source/ref ещё не закреплён, execution receipt не создан.
  Непригодный7651668 не превращён в разрешённый source, замены закреплённого
  source здесь нет. После интеграции исправления source должен быть выбран
  и записан до единственного dispatch вместе с его CI5/5.

## Причина остановки до dispatch

[Автоматический push run39923248401](https://github.com/QahNaaRlN/grace/actions/runs/39923248401)
завершился FAILURE на exact7651668 без jobs. Это не E6 workflow_dispatch.
В `.github/workflows/deploy-production.yml:419` имя job было записано как
`name: S1.17: ...` без кавычек. YAML parser видит второе двоеточие как начало
вложенного mapping и отвергает файл. Ошибка внесена в PR №100.

Source-head CI5/5 и regex-тест65 536 job expressions не разбирали YAML целиком,
поэтому не выявили дефект. Эти прежние PASS не доказывают исполнимость workflow
и не снимают blocker. Запуск по другому workflow/ручному SSH в обход gate
не выполнялся. Runtime/metadata/ledger/artifacts не читались.

## Ограниченное исправление E2/E5

Ветка `codex/statement-s117-workflow-yaml`, base7651668,
implementation `a053ad16308970a8e508274cae2bb5aee1a8a46d`:

- Единственное runtime изменение — кавычки вокруг имени job.
  Conditions, inputs, source pins, runner и private paths не меняются.
- `scripts/research/check-workflow-syntax.mjs` разбирает все реальные workflow
  YAML через parser уже закреплённого Web-инструмента Prettier. Новых зависимостей
  и прав доступа нет. Formatting файлов не меняется.
- Parser включён в начало обязательного `make test-production`, который
  выполняется в Quality gates до integration/build jobs.

Новый check сначала отверг исходный файл с ошибкой419 (RED), затем принял все3
workflow после исправления (GREEN). Логи: `/tmp/grace-s117-yaml-red.log`.
`python3 scripts/research/test-screening-workflow-modes.py` — PASS65 536;
`git diff --check` — PASS. Финальные `TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` и `make test-production`
завершены exit0:88 Web файлов/334 теста, Go, lint/FSD/format/build, offline Jev,
все прежние production guards и новый parser. Логи:
`/tmp/grace-s117-yaml-make.log`, `/tmp/grace-s117-yaml-production.log`.
Exact-head CI исправления проверяется после публикации; его SHA/run/5jobs
фиксируются в описании PR. До этого CI5/5 заранее не заявляется.

Это исправление принятого E2/E5, не новая campaign и не расход E6 разрешения.
Перед E6 обязательны интеграция исправления, закрепление immutable source/ref,
его CI5/5 и единичный preflight с прежними caps/markers/stop rules. Дополнительное
общее разрешение E6 не требуется. Повтор dispatch после любого исхода по-прежнему
не разрешён.
