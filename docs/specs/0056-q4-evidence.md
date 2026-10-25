# SPEC-0056 Q4: единственный запуск — INCONCLUSIVE / CONSUMED

Дата: 2026-10-25, Asia/Yekaterinburg. Исполнитель: Codex.
Статус отчёта: **READY_FOR_OWNER_REVIEW**, не owner acceptance.
Контракт: [accepted SPEC-0056 Q](0056-openrouter-document-qualification.md).
Отдельное основание запуска: [owner receipt](0056-q4-owner-receipt.md),
commit `8716e30c025fffb8718035ba548d2165127d83f2`, опубликован до dispatch.
Разрешение **CONSUMED**; повтор dispatch/rerun/смена модели не разрешены.

## Фактический результат

[Run40186488958](https://github.com/QahNaaRlN/grace/actions/runs/40186488958),
workflow_dispatch, attempt1, completed/success. Это успешное завершение
операционного workflow и проверки summary, а не успешное распознавание.
Итог campaign — **INCONCLUSIVE**: preflight остановил её до первого POST.
Качество настоящей модели **UNVERIFIED**.

| Показатель | Actual evidence |
| --- | --- |
| Exact source | `9200b731a0dea8f171e31aaf988d038073ad577f`, integrated main PR124 |
| Dispatch ref | `codex/document-q4-source-9200b73`, закреплён на exact source |
| Source CI | run40185076058, completed/success, все5 jobs PASS |
| Manifest SHA256 | `10200604c2a134577b3d22eed84fd106d7d951b1aececa0ec47f978dbf85653a` |
| Raw report digest из разрешённой проекции | `a3220ca4f7a82a69e39c62aff70afde6f2a9cf9a718ca88794bf0a67fe7a84be` |
| Candidate | Gemini3.1 Flash Lite / `google-vertex/eu`, без switch/fallback |
| Inference POST | **0** |
| Metadata GET | **3** из разрешённых6 |
| Known cost | **0 USD micros ($0)** |
| Unknown bills / over cap | **0 / false** |
| Policy proofs | Пустой map; qualified statement/receipt proof не опубликован |
| Campaign duration | **1302ms**, это preflight, не latency распознавания |
| Распознавание / token usage | Не измерены; ни один документ не отправлен модели |
| importReady / activationAuthorized | **false / false** |

Все восемь cases — cards, columns, cards-photo, full32-native, full32-scan,
receipt-discount, receipt-pdf, receipt-unreadable — имеют одинаковый actual
результат: `state=not_sent`, `reason=preflight`, `settlement=not_reserved`,
checks0, input/output tokens=null, actualMicros=null. Нулевой итоговый расход
следует из0POST и отсутствия неизвестных счетов; per-case null не подменяется
измеренным нулевым bill. Непрочитанные документы не получают quality FAIL/PASS.

Отдельный разрешённый read-only разбор после кампании установил
[ценовой guard отказа](0056-q4-incident-diagnosis.md): отсутствует явный request fee,
image fee не проходит нулевой ceiling. Исходная публичная summary ниже не
содержала этого диагноза; результат Q4 остаётся INCONCLUSIVE / CONSUMED.

## Доказательства и предел диагноза

Скачан только allowlisted artifact `document-reading-q-closed-result`;
[полная публичная summary](../research/statement-quality/q4-40186488958.summary.json)
прошла штатный `validate-public` тем же exact-source binary/bundle:

```sh
/tmp/grace-q4-integrated-eval-9200b73 -mode=validate-public \
  -bundle=/tmp/grace-q4-integrated-bundle-9200b73 \
  -authorized-source=9200b731a0dea8f171e31aaf988d038073ad577f \
  < /tmp/grace-q4-public-40186488958/document-reading-summary.json
```

Повторная offline сборка и export manifest выполняются командами из PR124.
Workflow guard подтвердил source/main ancestry/единственный dispatch/CI5/5,
сборка — approved manifest; remote step и summary/replay завершились success.
Это evidence работоспособности SSH/staging/container/replay в этой попытке.
Ключи/UUID/DSN и произвольный provider content не выводились.

По исходной публичной проекции точная причина была **UNVERIFIED**: она сохраняет
только `preflight`, counters и hashes. Она не содержит ответов metadata,
HTTP statuses и конкретного исключившего candidate guard. Нельзя объявить
причиной ключ, тарифы, fees, ZDR, модель или общий бюджет по одному этому коду.
Работоспособность inference-ключа и достаточный shared budget не доказаны.
`ModelGroups.refresh`/подготовка первого statement policy/request остаются
областью возможной локализации, не подтверждённым диагнозом.

Private raw report и replay остались на VPS в
`/opt/grace/shared/document-reading-spec0056-q.private.json`; marker
`/opt/grace/shared/document-reading-spec0056-q.started` сохраняется.
Их создание/чтение подтверждено успешным путём wrapper, отдельный SSH/stat
не выполнялся. Скачивания private payload, дополнительных metadata GET,
provider calls, SSH diagnostics, budget/secret writes и production activation
после остановки не было. Staging удаляет штатный cleanup; marker не удалялся.

## История dispatch без скрытого повтора

Первый административный запрос к GitHub с commit SHA в поле ref был отвергнут
HTTP422 `No ref found for: 9200b731a0dea8f171e31aaf988d038073ad577f`.
Это ошибка оператора: GitHub не создал workflow event/run; повторная read-only
проверка показала `total_count=0`. SSH/preflight/provider requests тогда0.

Затем создан branch ref `codex/document-q4-source-9200b73` на том же source.
Через него принят **один фактический dispatch** — run40186488958/attempt1;
после этого допуск CONSUMED. Rerun/attempt2/новый campaign не выполнялись.
До исполнения опубликован неизменный receipt; source/scorer/corpus/prompts/
schema/model/manifest не менялись. Отклонённый API запрос не скрывается как
успешный dispatch или как отдельный inference attempt.

## Проверки и следующий отдельный пакет

- Q1.1 / Q2.2: actual source/manifest/corpus/лимиты связаны source CI и run guard;
  preflight0POST,3GET; утверждения о real recognition отсутствуют.
- Q3.1: все cases not_reserved; known cost0, unknown bills0. Это не SQL ledger
  reconciliation или доказательство достаточного allowance.
- Q3.2 / Q3.3: workflow attempt1, публичный projection validator PASS;
  private raw/replay остаются remote, общий lock/marker/cleanup сохранены.
- Q3.4: `make test` PASS (Go, web339/339, lint/format/build, Jev/corpus),
  лог `/tmp/grace-q4-execution-make-test.log`; `git diff --check` PASS.
  Production code/OpenAPI/web source/scorer/golden не менялись.

Следующее возможное решение — отдельный ограниченный read-only разбор уже
сохранённого report на VPS, без новых provider/metadata calls, без marker reset,
без финансовых/секретных значений в выводе и без повторного распознавания.
Текущий допуск не открывает этот дополнительный SSH/diagnostic scope.
Изменение guard/candidate и новый live attempt требуют собственного решения.
