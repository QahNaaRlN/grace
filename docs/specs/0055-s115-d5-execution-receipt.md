# S1.15 / D5 — receipt единственного разрешённого workflow

Дата: 2026-10-23, Asia/Yekaterinburg. Этот документ содержит только operational
metadata, уже доступные в GitHub run. Private-derived projection/witnesses и
содержимое artifacts в репозиторий не переносятся.

- [Execution decision](0055-s115-d5-run-decision.md); owner прямо ответил «разрешаю»
  на отдельный вопрос об одном D5 на exact SHA, SSH-чтении private P6 и closed CI
  report без model/SQL/data writes. Approval commit `e5961a4`, до dispatch.
- Source `703b6c6ac960c21c0b7ab7bd84b3411459204c92`, ref
  `codex/statement-saved-proposal-diagnosis`, frozen/clean и интегрирован.
- Source [CI39639515724](https://github.com/QahNaaRlN/grace/actions/runs/39639515724)
  exact5/5 перепроверен canonical validator до dispatch; runtime diff с main отсутствует.
- Единственный [run39860669842](https://github.com/QahNaaRlN/grace/actions/runs/39860669842)
  **SUCCESS**, head703b6c6; создан2026-10-23 02:52:14 +05:00 / 2026-10-22 21:52:14 UTC.
- [Job120110244121](https://github.com/QahNaaRlN/grace/actions/runs/39860669842/job/120110244121)
  `S1.15 read-only saved P6 diagnosis`: SUCCESS,02:52:17–02:53:42 +05:00.
  Все остальные10 jobs SKIPPED, включая deploy/paid comparisons/reconciliation.
- Из GitHub step metadata: exact CI gate SUCCESS; local public-context/manifest
  preparation SUCCESS; fixed read step02:53:35–02:53:39 +05:00 SUCCESS; retain-artifact
  step SUCCESS. Это время operator workflow, не latency модели.
- Ровно один реальный dispatch, permission **CONSUMED**. Дополнительных dispatch,
  SSH/health/provider probes, repair или source/environment изменений после него нет.

## Разделение доступа к CI и публикации в Git

Closed report `statement-s115-sanitized` сохранён в CI. Его проекция проверена
локально verifier `validate`, собранным из frozen source, без повторного VPS read.
Тело projection, structural facts, witnesses, их численные значения и content hash
в Git/PR body не копируются. Этот receipt не заменяет сам closed CI artifact и не
объявляет model qualification/financial acceptance.

Automatic approval review отдельно отклонил команду переноса private-derived
payload из CI в repo/evidence docs: owner approval покрывал closed CI report,
но не публикацию его содержимого в Git. Отказ был до CreateProcess: копирование
не выполнилось. Другого способа публикации или обхода отказа не предпринималось.
Безопасная альтернатива — только этот operational receipt; CI остаётся единственным каналом публикации payload; локальный временный
cache использован только для проверки и не включён в Git. Дополнительное право на публикацию payload не предполагается.

## История локальной подготовки

Предыдущий отказ на расплывчатое «продолжить работу» был до CreateProcess и
закрыт последующим явным human approval. Затем первый вызов gh CLI локально
отклонил JSON из-за boolean values: его parser требует map[string]string.
Это было до HTTP; GitHub run list оставался пустым. В transport values использованы
строки true/false с прежними логическими inputs, после чего создан только run выше.
Ни один из локальных отказов не являлся D5 execution или повтором VPS run.

## Проверки handoff и следующий gate

`TMPDIR=/tmp VITEST_MAX_WORKERS=1 make test` PASS,
`/tmp/grace-s115-d5-approved-make.log`; `git diff --check` PASS.
Дальнейший diff только docs; runtime/workflow/API/Web и source703b6c6 не менялись.
Новый локальный full integration не заявляется; source CI содержит полный suite.

Диагностический workflow завершён. Следующий этап — отдельное owner решение по
закрытому CI evidence. Fix/new paid comparison/model switch/deploy этим разрешением
не открыты. Сохранение payload в Git требует отдельного согласования, если оно понадобится.
