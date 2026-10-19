# SPEC-0055 / S1: атомарное планирование и multi-chunk harness

- Дата: 2026-10-19 (Asia/Yekaterinburg).
- Статус: technical candidate; implementation `not_started`.
- Основание: [проработка качества](../research/statement-quality/README.md) принята владельцем ответом «Принимаю» на PR #68 (`c63e7cd3f7ca355ee7470036919b948d0078b73f`).
- Принято: выводы исследования и порядок S1–S5. Этот новый технический документ ещё не принят.
- Baseline product tree: `7be572ceaac5819a2031059ef60388dcd0711614` (PR #66).
- Область: только planner, проверка актуальности plan/consent и offline/fake-provider harness SPEC-0055.

## Пользовательский результат и границы

Добавление служебного текста не должно переносить поля одной операции в разные
model requests. Если безопасное разбиение невозможно, preflight предлагает
ручной review до consent и расхода бюджета. Отказ не означает успешный разбор.

В S1 не входят новый response protocol S2, классификация оставшихся spans как
service, новые модели, автоматический выбор, платные проверки, расширение
лимитов или production enable. Числа, даты, роли полей и финансовый импорт
продолжают проверять существующие Go-модули и confirmation flow.

## Что требуется выбрать до реализации

Текущая полоса `floor(YMin / 6)` объединяет горизонтальные поля, но ничего не
говорит о целой вертикальной карточке. Заменить её константой большего размера
недостаточно: сдвиг координат снова может разрезать запись.

Следующий исполняемый шаг — offline comparison геометрических кандидатов.
Минимум два кандидата: объединение соседних полос по нормализованному межстрочному
интервалу и консервативное объединение layout blocks с пересечением по строкам.
Оба используют исходные bbox/block/line, а не финансовый текст и не ответ LLM.
Для каждого нужен самостоятельный исследовательский prototype; product
`BuildPlan` в этом шаге не меняется. Зафиксировать точные правила сортировки,
округления, обработки равных координат, пороги, зависимости от page/font scale
и правила отказа. Выбор порогов на двух PDF не доказывает универсальность.

Пробел между карточками нельзя считать доказанной семантической границей:
внутри одной операции тоже бывают большие промежутки. Если кандидат не умеет
различить такие layouts, объединять сомнительные блоки или отказывать, а не
выдавать гарантию record atomicity. Golden oracle независим от алгоритма.
Если ни один кандидат не проходит критерии ниже, сохранить FAIL и пересмотреть
подход, не объявлять S1 завершённым за счёт отказа на всех документах.

До product implementation эта редакция должна содержать выбранный алгоритм,
полный набор констант и результаты comparison и получить статус `accepted`.
Сейчас выбор алгоритма и его выполнимость — `UNVERIFIED`.

## Контракт atom → packing

1. Геометрия анализируется до redaction/сериализации prompt. Atom содержит
   ordered source IDs, исходные страницы и bbox; текст, размер JSON, schema и
   модель не создают границ. Один source ID принадлежит ровно одному atom.
2. Атомы могут быть крупнее операции. Нельзя утверждать соответствие
   atom = банковская операция без независимой проверки. Сложные колонки,
   wrapped descriptions и continuation через страницу входят в comparison.
3. Чтение идёт в детерминированном порядке, определённом выбранным алгоритмом.
   Одинаковые document/visible IDs/policy/versions дают побайтно одинаковый plan.
4. Packing добавляет целый atom и измеряет реальный `InputBytes(candidate)`:
   messages + schema + context. Сохраняются 9000 bytes и `MaxCalls=32`.
   Переполнение завершает предыдущий непустой chunk, затем проверяется atom
   отдельно. Непомещающийся atom → `statement_atomic_block_limit`;
   превышение числа chunks → `statement_call_limit`. Частичный plan не выдаётся
   как пригодный для consent, нет reservation или model call.
5. Context разрешён только как повторное read-only evidence со старыми IDs;
   каждый span в основных `Spans` принадлежит одному chunk. Context не заменяет
   отсутствующее поле основного record и не разрешает duplicate transactions.
6. Новый prompt/schema может менять packing целых atoms, но не их состав.
   Успешный plan не теряет, не дублирует и не превращает source spans в service.
   Критические поля, description и отдельные служебные подписи проверяются
   раздельно; старый core oracle покрывает лишь часть record.

## Version, consent, хранение и API

Предлагается добавить в persisted `Plan` обязательные `plannerVersion` и
`atoms` (ordered lists of source IDs). Полный plan hash охватывает эти поля,
chunks, document/redaction/prompt/schema и pinned policy. Нельзя хранить только
версию без фактической atom map. Новая версия определяется при реализации
принятого алгоритма и меняется при изменении правил/порогов.

`currentPlanContract` проверяет planner version наряду с существующими версиями.
Отсутствующая/иная версия или изменённая карта делает старый preview непригодным:
`preview_changed` (409) до нового consent/reservation/call. Повторное открытие
старого preview не должно молча строить новый план. Владелец получает новый
preview и подтверждает именно его hash. Чужой owner, stale revision и
идемпотентность сохраняют существующие ограничения.

Plan уже хранится как JSON в `statement_extraction_previews`; отдельная SQL
колонка не предполагается. Это предложение нужно проверить на фактическом
storage path и обработке завершённых attempts. Их historical audit должен
оставаться читаемым без повторного inference. Rollback не переносит consent
между версиями: старый binary отказывает на новом plan; явная проверка этого
сценария обязательна, нельзя полагаться на игнорирование лишних JSON полей.
Один из совместимых способов — также изменить уже проверяемую prompt version,
сохранив текст prompt; окончательный механизм включается в accepted редакцию.

Preview содержит plan в публичном API: additions проходят через
`api/openapi.yaml`, затем regenerate TypeScript types. В Web не появляется
геометрических правил или финансовой арифметики. API contract/compatibility
и error presentation должны быть проверены до реализации, а не постфактум.

## Acceptance matrix и исполняемое evidence

Следующие проверки — требования к будущей реализации, сейчас `NOT_RUN`.
Исторические 32 probes уже выполнены, но не проверяют новый planner.

| ID | Свойство | Требуемое executable evidence |
| --- | --- | --- |
| S1-AC1 | Базовые columns/cards продолжают приниматься; не все случаи превращены в отказ | Реальный pinned helper + независимые source selectors; оба baseline layouts PASS с полными records |
| S1-AC2 | 32 исторических prompt/prefix случая: ни одного успешно принятого плана с разрывом core | Машиночитаемые reports v7/v8 на одном candidate geometry; accepted/refused/split counts отдельно; все прежние 24 accepted cases остаются accepted и atomic |
| S1-AC3 | Не теряются description/currency/labels, geometry не зависит от prompt bytes | Независимые full-record ID sets; prompt padding, translations, schema-size и bbox translation/scaling tests; сравнение atom maps |
| S1-AC4 | Безопасные пределы и однозначное владение | Размер actual messages+schema на 8999/9000/9001; single-atom overflow, 32/33 chunks; exact-once source membership; отказ до вызова Reasoner |
| S1-AC5 | Layout ambiguities не приводят к молчаливому разрыву | Reordered columns, side-by-side regions, wide within-record gaps, continuation через страницу, repeated page headings; independently reviewed expected accept/refuse |
| S1-AC6 | Consent связан с конкретной картой и версией | Integration tests: changed atoms/version, missing version, source revision, stale preview, чужой owner, повторный request; ноль calls/reservations на отказе |
| S1-AC7 | Multi-chunk fake не видит чужие IDs и не фабрикует глобальный ответ | Real helper → planner → fake на фактические chunks → SQL draft/reconciliation; asserts row fields/clocks/FX, coverage, duplicates, totals и owner isolation |
| S1-AC8 | Финансовая запись остаётся подтверждаемой | В AC7 ноль financial writes до confirmation; после неё ожидаемые postings ровно один раз; manual path работает без Reasoner |
| S1-AC9 | Миграция/rollback безопасны, контракт API синхронен | Historical attempt читается; старый/новый consent несовместимы в обе стороны; generated types diff; API/web build |

Для fake-provider запрещены IDs, отсутствующие в `Spans ∪ Context` конкретного
request. Он не возвращает глобальные rows на каждый chunk и не получает
финансовые поля через непубличный обход. Context ID не даёт права повторить
row. Проверять минимум один успешный настоящий multi-chunk сценарий, отсутствие
coverage и недопустимое повторение row отдельными negative cases.

Финальная проверка принятой реализации: `make test`, `make test-integration`,
`go vet ./cmd/... ./internal/...`, `git diff --check`; отдельно существующий
bank/PDF integration script на disposable PostgreSQL и fake provider.
Исследовательские commands, exact refs, tool versions, input hashes и полные
FAIL/PASS сохраняются в evidence. Платных вызовов в S1 нет.

## Состояние на 2026-10-19

Исследование принято. Этот документ фиксирует границы, инварианты и проверяемые
критерии следующего пакета. Prototype comparison, выбранный алгоритм, API diff,
новые тесты и реализация пока отсутствуют; production статус не перепроверялся
этой документационной правкой. Последний зафиксированный статус — Statement AI
false; этот документ его не меняет.
