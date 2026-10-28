# Q4.2 REJECT: причина установлена по сохранённому ответу

2026-10-25, Asia/Yekaterinburg. Основание — прямой запрос владельца
«разбор REJECT» после [run40191109474](https://github.com/QahNaaRlN/grace/actions/runs/40191109474).
Это read-only diagnosis; production repair и новый paid run не выполнялись.
[Исходный execution evidence](../../specs/0056-q42-execution-evidence.md),
[принятый R](../../specs/0056-q4-pricing-repair.md),
[public summary](q42-40191109474.summary.json).

Причина REJECT — нарушение контракта evidence в первом `cards`:
все23 bounding boxes содержат координаты вне0..1, а два непустых поля
второй строки (`transactionAmount`, `transactionCurrency`) имеют `evidence: []`.
Это две независимые причины отказа `VisualReading.Validate`.

## Наблюдение и причинная цепочка

[Закрытая diagnosis projection](q42-40191109474.diagnosis.json) получена
на VPS из фиксированного saved report. HTTP200, finish_reason=stop;
7150bytes content, корректный JSON без дубликатов и markdown fence,
структура header/rows и typed keys соответствуют протоколу:8header fields,2rows.
Признаков превышения output limit/обрыва по length нет.

-23 boxes /92 координаты — целые числа, каждая >1 и≤1000.
  Они нарушают явное требование prompt «normalized page coordinates0..1»
  и `internal/statementdata/visual.go:74`. Это совместимо со шкалой0..1000;
  намерение модели и единицы координат из такого наблюдения не доказаны.
- В `rows[2].fields.transactionAmount` и `transactionCurrency` присутствуют
  непустые value, но evidence пуст. `visual.go:65,93` требует1..8 цитат
  для любого non-null field. Null разрешён, заполненное поле без цитаты — нет.
- `internal/documentqualification/measure.go:35` сводит parse/Validate failures
  в один `protocol` FAIL и возвращается до распознавания полей/строк.
  `campaign.go` фиксирует quality_reject/REJECT и останавливает следующие cases.

Синтаксически корректный JSON поэтому не достаточен. Финансовая точность
полей, полнота full32 и качество чеков остаются UNVERIFIED.
1POST/6GET/$0.005183 settled и остальные7not_sent сохраняются без изменений.

## Воспроизведение без новых модельных вызовов

На VPS над копией parsed object **только в памяти** выполнены counterfactual checks;
исходный report не переписывался. Разделение всех координат на1000 оставляет
два evidence нарушения. После дополнительной замены двух uncited fields на null
структурная проверка проходит. Это диагностическое преобразование с потерей
двух предложенных значений, не исправленный ответ и не quality PASS.

Локальный [probe](q42-reject-probe/main.go) использует настоящие
`statementextract.DirectJSON` и `statementdata.VisualReading.Validate` на явно
synthetic structural controls. Его [вывод](q42-offline-structural-probe.json):

| Control | Actual DirectJSON | Actual Validate |
| --- | --- | --- |
| Целые boxes100..400 + два uncited fields | PASS | FAIL |
| Только boxes разделены на1000 | PASS | FAIL |
| Все поля cited, boxes остаются100..400 | PASS | FAIL |
| Boxes нормализованы, два uncited fields=null | PASS | PASS |
| Boxes нормализованы, все поля cited | PASS | PASS |

Probe не получает private response и не выполняет raw Replay/recognition scoring.
Совпадение двух классов нарушений проверено actual Go кодом, а не сменой scorer.
Деление на1000 показывает достаточность нормализованной геометрии для этого
валидатора; не доказывает правильность регионов на странице.

## Почему structured output не предотвратил отказ

Actual `VisualSchema()` (`internal/statementdata/visual.go:114`) описывает box
как4 numbers, но не объявляет minimum0/maximum1 для координат.
Evidence/anchor описаны массивами цитат без minItems1/maxItems8.
Domain validator требует эти ограничения. Значит JSON Schema допускает
наблюдаемые классы нарушений, которые затем закономерно отвергает Go.
Probe извлёк actual schema и подтвердил отсутствие этих keywords.
Это выявленный пробел между wire schema и validator; он не оправдывает
принятие ненормализованных/неподтверждённых данных.

Направление следующего repair: согласовать schema с действующими ограничениями
Validate, явно закрепить нормализованные boxes и правило «нет цитаты → null»
в prompt, проверить отрицательные/положительные controls и offline replay.
Автоматическое угадывание шкалы либо придумывание цитат не обосновано диагнозом.
Модель/тарифы/scorer/corpus менять по этим данным оснований нет.
Production code не менялся; это рекомендация, а не принятый новый пакет.

## Read-only доказательства и ограничения

Два bounded pinned SSH sessions запустили Python из stdin с isolated interpreter,
без staging/upload, Docker/SQL/key/env inspection и model/metadata requests.
Reader открывает **только** fixed q2 marker/report с O_RDONLY/O_NOFOLLOW/O_NOATIME,
проверяет mode0600, bounds, stat/hash до/после; наружу выводит whitelist flags,
counts, schema paths и hashes. Сырые content/quotes/value/unknown keys не выводятся.

- Source `b1d310ed8746d4c1159d0b345c4acc3a7d8d3fdf`.
- Manifest `28f1e68838743a597203a938e0a36b80bf60c459d24e26614e86c08f307aded5`.
- Private report object digest `444feb9ddf89d4b8a13c532a6638a65fdfab271e6932747dd4fa8693ab2b7329`.
- Report file SHA256 `34110e191f6f34c2137c520ad55751151a42fe272c9c736ac9af77f10b0c28b6`.
- Marker file SHA256 `74bd1c301c9bccc5506e8406d9d5dd8db2eb69ccce46650db42303c525f1dc81`.
- [Snapshot audit](q42-readonly-snapshots.json): оба файла неизменны внутри/между sessions.
- New provider/metadata calls0, SQL0, requested persistent remote writes0.
  SSH/auth audit logging инфраструктуры этими счётчиками не объявляется выключенным.

[Reader](q42-reject-reader.py), SHA256
`760c358fd8b35e4fabf7ea706771a3c28f8ff47036f54369eb7d64fc95f2af74`;
[offline controls](q42-reject-reader-test.py):6PASS, включая private canary,
wrong marker/seal, changing snapshot, unknown keys, duplicate/markdown.

Команды:

```sh
python3 -B docs/research/statement-quality/q42-reject-reader-test.py
GOPROXY=off go run ./docs/research/statement-quality/q42-reject-probe
make test
git diff --check
```

Reader controls и Go probe PASS. Первый `make test` FAIL в неизменённом
`TestCLISyntheticSnapshotRunsOffline`: output_inside_repository, потому что
существующий `/tmp/.git` делает default temporary output запрещённым.
Guard/код не менялись; повтор использует отдельный `TMPDIR` в `/var/tmp`
с удалением созданного каталога. Повторный `make test` **PASS**, включая339/339Web.
Первый cleanup `rmdir` отказал из-за task-created Node compile cache; удалён
только этот cache и точный temp каталог, отсутствие каталога проверено.
Повтор probe в ограниченном shell встретил snap-confine refusal; тот же probe
через рабочий Go runtime PASS, output byte-for-byte совпал с сохранённым.
Никаких изменений snapd/AppArmor/системного окружения не выполнялось.
`git diff --check` PASS.
Правила accepted Q/R и неизменность production code соблюдены; разрешение
live Q4.2 остаётся CONSUMED. Ни read-only diagnosis, ни structural counterfactual
не являются новым допуском, qualification/activation или успешным распознаванием.
