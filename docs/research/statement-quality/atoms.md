# S1 offline comparison — 2026-10-19

Статус: исследование выполнено; product implementation `not_started`.
Основание: принятая [проработка](README.md) и поручение продолжить работу к
включению Statement AI. [Конкретный technical candidate](../../specs/0055-record-atomic-planning.md)
предлагает whole-document atom/aliases и новые request bounds для review.

## Что проверено

Два существующих публичных synthetic PDF, восемь prefix-вариантов из принятого
исследования и шесть новых in-memory mutations: y translation +3, scale ×1.5,
wide gap внутри первого record, continuation первого record на новую страницу,
перемещение второй записи в другую горизонтальную область и перестановка
source input внутри страниц. Это 28 layouts × 2 historical prompts × 5
кандидатов = **280 cases**. Не новые PDF/банки, не модельный benchmark.

Core oracle берёт уникальные printed dates/money/clocks. Full-record oracle
дополнительно включает **все** source IDs внутри напечатанного record: labels,
валюты, описания. Его границы заданы независимо по генератору public PDF:
для columns строка с operation date и следующая description (+20 PDF units);
для cards первая операция от operation line до +80, вторая до +60. Source ID
sets вычисляются на исходном extracted layout до mutations; алгоритм их не
получает. Общие column/header currencies остаются отдельным reusable evidence,
не входят в full-row set. Это не independent human golden review S3.

## Результат

| Кандидат | v7: accepted / refused | v8: accepted / refused | Core split среди accepted v7 / v8 | Full-record split среди accepted v7 / v8 |
| --- | --- | --- | --- | --- |
| gap: интервалы + 1.5 median line height | 24 / 4 | 24 / 4 | 2 / 2 | 2 / 3 |
| blocks: целые Poppler blocks, объединённые по y-overlap | 24 / 4 | 24 / 4 | 4 / 5 | 12 / 17 |
| весь документ, прежние IDs / 9000 bytes | 7 / 21 | 0 / 28 | 0 / 0 | 0 / 0 |
| весь документ, aliases / 9000 bytes | 21 / 7 | 20 / 8 | 0 / 0 | 0 / 0 |
| весь документ, aliases / **90000 bytes**, context пуст | **28 / 0** | **28 / 0** | **0 / 0** | **0 / 0** |

Геометрические кандидаты **REJECTED**: внутренний wide gap и page continuation
могут разделить record на atoms и на requests; source exact-once это не
обнаруживает. Расположение всего документа в одном запросе устраняет именно
структурный разрыв. Оно ничего не говорит о корректности классификации LLM.

Whole-document без увеличения cap не проходит прежний критерий сохранения
доступности. Новая стратегия проходит structural oracle, потому что один chunk
содержит все spans; её практическая ценность зависит от достаточности cap.
Это не доказательство способности понять multi-column/continued records.

Для каждого принятого плана checked: exact-once source ownership, actual
messages+schema ≤ declared cap и повторная сборка с тем же plan hash.
Дополнительно **34 boundary checks**: cap−1/cap/cap+1 для всех десяти сборок;
2000 accepted и 2001 refused spans для двух 90k сборок. Проверяется bijective
alias map; 90k candidate имеет один call, пустой context и theoretical provider
reservation ceiling 70000 micros при synthetic policy. Не production usage.

Второй старый prompt v8 — исторический stress overlay; его прежний live FAIL
не снят, он не предлагается для release. Полный source coverage/model mapping,
new response schema, token usage, latency и actual cost здесь **UNVERIFIED**.
Нулевые provider calls и database writes.

## Воспроизведение и provenance

```sh
./scripts/research/probe-statement-atoms.sh /tmp/grace-atoms-comparison
```

[Runner](../../../scripts/research/probe-statement-atoms.sh),
[geometry templates](../../../scripts/research/atoms/groups.go.txt),
[probe](../../../scripts/research/atoms/probe.go.txt),
[summary](atoms-evidence/summary.json), [provenance](atoms-evidence/provenance.json).
Десять detailed reports находятся рядом с summary. Исходные файлы продукта
не переписываются: Go build overlays заменяют planner и research entrypoint
только при сборке. Baseline exact ref проверяется до запуска. Go templates
не входят в обычную product compilation. Candidate source hashes, Go/extractor
versions и публичные fixture hashes записаны в provenance/reports.

Pinned Poppler helper и probe запускаются network=none/read-only UID 10001,
без API/DB/ключей, с отдельным socket volume. Compose создаёт только helper;
trap удаляет его собственные контейнеры/volume/tmp. Image build получает только
обычный репозиторий; банковские оригиналы в build context не добавляются.
Failed candidate results сохраняются, не заменяются успешными повторами.

Дополнительный local-only preflight ранее предоставленного owner PDF подтвердил,
что вариант 9000 bytes не подходит, а compact whole-document payload укладывается
в предлагаемые 90000 bytes / 2000 spans. Это проверка вместимости с прежним
Proposal schema/prompt, не inference и не quality. Исходный PDF, layout, его
текст/hash и private report в Git/CI не добавлены; временные контейнеры удалены.
Любые будущие prompt/schema изменения требуют повторного preflight.

## Следующий шаг и границы

Владелец принял [редакцию S1](../../specs/0055-record-atomic-planning.md):
один документ, обратимые короткие locators, 90000 bytes / 100000 input tokens /
16000 output tokens, максимум 2000 spans; прежняя exact route и общий бюджет.
Это повышает theoretical provider ceiling одного call с 0.008 до 0.07 USD.
Принятие зафиксировано 2026-10-19 («Утвердить S1 с новыми лимитами»);
платных calls эта работа не разрешает.
Дополнительно выявлены реальные transport blockers: общий reservation helper
ограничен 10000/2000, call timeout — 30 s. Общий API WriteTimeout — 15 s,
но extraction route уже имел best-effort deadline 145 s с игнорированием ошибки.
Прежняя формулировка о неизбежном обрыве этого route была неполной. В S1
предложен отдельный statement transport profile и route-specific write deadline,
с real HTTP/fake-provider test. Отдельный
[standard-library diagnostic](../../../scripts/research/httpdeadline/main.go)
[воспроизвёл](atoms-evidence/http-deadline.json) обрыв медленного ответа и
работу route deadline при сохранении общего малого timeout. Это feasibility
diagnostic. В реализации S1 [real HTTP/SQL evidence](../../specs/0055-s1-evidence.md)
проверяет сам handler, CORS, request cancellation и сохранение known billing.
Затем implementation, fake-provider/SQL checks, S2 coverage protocol и S3 corpus,
после них отдельно bounded model qualification. Включение до PASS не выполняется.
