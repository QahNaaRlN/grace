# SPEC-0055 — технический контракт и qualification

- Дата: 2026-10-18
- Статус: candidate; архитектурное разделение принято владельцем
- Основание: «Принимаю такое разделение. Взять в работу» в текущем чате
- Functional spec: [SPEC-0055](0055-bank-statement-pdf-import.md)
- Baseline: main `a98e304`, ветка спецификации `codex/spec-0055-bank-pdf-import`
- Application implementation: not_started; private inference: not_run

## Принятое архитектурное решение

Локальный PDF/text/spans → LLM proposal → Go grounding/normalization/сверка →
сохраняемый review → явная financial confirmation владельца. Jev рассматривается
для измеряемых узких выборов/проверок. Он не обязателен для каждого документа.
Принятие этого разделения не является consent на передачу банковских данных.

## Локальный extractor — проверенный кандидат

`poppler-utils=25.04.0-r0` в `alpine:3.22`, отдельный executable
`pdftotext -bbox-layout`; не линковать PDF parser внутрь финансового Go-кода.
Пакет и лицензия подтверждены [официальным Alpine package index](https://pkgs.alpinelinux.org/package/v3.22/main/x86_64/poppler-utils).
Image digest qualification: `sha256:dd9106fd118efe5aea5cb375309f764ef2e606955c39cb43d13fd60325c2f1b4`.
Это временный локальный image, не production release artifact.

Условия испытания: uid/gid 10001, rootfs/input readonly, network=none,
cap-drop ALL, no-new-privileges, 512 MiB memory+swap cap, 1 CPU, pids-limit 16,
private tmpfs 16 MiB, внешний timeout 30 s. Исходник передавался только
read-only bind mount в локальный Docker daemon. Image build выполнял только
установку публичного пакета; файл не входил в build context.

Два независимых запуска на предоставленном reference:

| Проверка | Результат |
| --- | --- |
| Завершение под заданными лимитами | PASS, 0.471 s / 0.575 s на этой машине |
| Выход bbox-layout двух запусков | Побайтно совпадает |
| Страницы | 5 |
| Число word spans по страницам | 314 / 407 / 414 / 417 / 109 |
| Bounding boxes | Все finite, упорядочены и в пределах страницы |
| Начала строк по date/column bands | 14 / 26 / 26 / 26 / 4, всего 96 |
| Размер bbox XML | 347 124 bytes |
| Компактное состояние word IDs/box/text до redaction | 112 305 UTF-8 bytes |
| Сеть/inference/financial writes | Отсутствовали |

Это ограниченное qualification extractor, не банковского parser или AI модели.
Время на VPS, cancellation/crash, malicious PDF, максимальные 50 страниц/8 MiB,
стресс по памяти/конкуренции остаются UNVERIFIED. Нельзя интерпретировать cap
512 MiB как измеренный peak memory или эти два запуска как полное sandbox proof.
Word locator включает extractor version; замена Poppler требует нового review
совместимости, а не повторного присвоения прежних span IDs.

Для реализации нужен управляемый helper с такими же ограничениями и process-group
cancellation; обычный `exec.CommandContext(pdftotext)` внутри API без ресурсного
и сетевого ограничения не удовлетворяет контракту. Host бинарник 26.01.0 не
используется как доказательство совместимости production Alpine пакета.

## LLM — конкретная предлагаемая policy

Предлагается первый text-only путь через существующий OpenRouter adapter:

```json
{
  "model": "google/gemini-2.5-flash",
  "provider": "google-vertex/eu",
  "providerName": "Google",
  "inputMicrosPerMillion": 300000,
  "outputMicrosPerMillion": 2500000,
  "imageTokenUpperBound": 0,
  "vision": false
}
```

`verifiedAt` не зашивается в документ как вечное разрешение: его выставляет
проверка metadata непосредственно перед разрешённым inference. Policy относится
к новому purpose `statement`, не переиспользует consent `receipt` или `analysis`.
Production flag для statement extraction по умолчанию выключен.

По публичным model/endpoints metadata OpenRouter на 2026-10-18:

- `google/gemini-2.5-flash` поддерживает structured_outputs/response_format/max_tokens;
  публичная цена — $0.30 / $2.50 за 1M input/output tokens;
- endpoint tags `google-vertex/eu` и `google-vertex/global` имеют status 0;
- прежний `google-vertex` из примеров Grace имеет status -5 и не проходит
  текущий `PolicyVerifier.Current`, который требует status 0 и точное совпадение tag;
- имя провайдера у предлагаемого EU tag — `Google`, цена совпадает с потолками.

Источники: [model metadata](https://openrouter.ai/api/v1/models),
[endpoint metadata](https://openrouter.ai/api/v1/models/google/gemini-2.5-flash/endpoints),
[страница модели](https://openrouter.ai/google/gemini-2.5-flash).
Публичные GET не содержали пользовательских данных/ключей и не были inference.

**Смена `google-vertex` → `google-vertex/eu` не применяется автоматически.**
В существующем verifier обновление policy сохраняет уже одобренные model/provider/
price ceiling. Поэтому здесь требуется отдельное решение владельца о новом
statement-purpose и конкретном endpoint tag; существующие receipt/analysis
policies не меняются. `eu` не является заявлением о юридической резидентности
всей цепочки OpenRouter. Фактическая доступность ZDR endpoint с аккаунтом и
задачное качество модели ещё UNVERIFIED.

Каждый inference сохраняет действующие `zdr=true`, `data_collection=deny`,
`allow_fallbacks=false`, `only=[google-vertex/eu]`, price ceilings и проверку
requested/actual model/provider. При несовместимости — безопасный отказ,
не fallback. Для Jev понадобится собственная проверка privacy/consent; в
первый активируемый путь он не включается автоматически.

## Размеры вызовов и верхняя стоимость

Нельзя отправить весь PDF в один нынешний `LLMRequest`: существующие границы —
inputBound ≤10 000, outputBound ≤2 000. Измеренный word state reference уже
существенно больше одного такого запроса.

Предлагаемый bounded контракт:

- До 32 provider calls на один документ, включая повторные попытки; один
  документ не получает 32 новых бюджета после reload/нового extraction key.
- На call input reservation 10 000 tokens, max output 2 000. Structured messages
  + schema + redacted chunk в сумме ≤9 000 UTF-8 bytes, ещё 1 000 единиц bound
  оставлено для provider framing. Этот conservative bound требуется проверить
  на фактической usage выбранного endpoint; превышение — protocol/usage review.
- Чанки состоят из цельных локальных block/span groups, сохраняют page/span IDs,
  несут bounded повтор header context. Перекрытия отслеживаются по source IDs,
  не создают вторую строку. Документный plan покрывает все страницы/spans.
- Plan с >32 calls или oversized atomic block отклоняется до первого платного
  вызова с возможностью manual review. Источник не обрезается до доступного бюджета.
- Весь верхний bound резервируется до первого call через общий budget ledger:
  32 × (10 000 × $0.30/1M + 2 000 × $2.50/1M) = **$0.256**.
  Это потолок данного plan по текущим ценовым ceilings, не цена фактической выписки
  и не увеличение принятого общего бюджета. Меньше calls → меньший reservation.
- Неизвестный расход/потерянный ответ не освобождает reservation как бесплатный
  успех; дальнейшие calls приостанавливаются до reconciliation. Input/output
  usage и reasoning tokens входят в actual output bound/стоимость.
- Не более одного 30 s call за раз, полный attempt ≤120 s; time budget может
  остановить plan раньше 32 calls. Невыполненный остаток явно остаётся incomplete,
  не становится частичным финансовым импортом и не перезапускается бесконечно.

Размер 112 305 bytes — до redaction, prompts и schema; он не доказывает, что
reference поместится в 32 вызова, и не является измерением model token usage.
Число чанков/выход JSON и качество proposal проверяются на synthetic fixtures
после принятия технического контракта, до передачи банковского reference.

## Consent и хранение

Согласие `statement` связывает owner, document/text state hash, prompt/schema,
model/provider/purpose, redaction policy и верхний plan cost. Отдельные решения:
разрешить inference и подтвердить ledger. Upload PDF и принятие архитектуры
не выполняют ни одно из них.

Default retention из draft: 7 дней review, delete_after_confirmation после
завершения всех строк, keep явно. Metadata/provenance/minimal fragments остаются;
full page text/raw provider payload не становятся скрытым архивом. Старые backup
могут содержать оригинал до истечения retention — это явно показывается.

## Необходимые изменения существующих контрактов

До реализации проверены реальные ограничения baseline, которые нужно расширить:

1. `UsageRepository.Reserve` и CHECK purpose в миграции 000013 пока разрешают
   только analysis/receipt/jev_category. Нужен явный statement-purpose и
   document-level aggregate reservation/settlement, не 32 отдельных полных budgets.
2. Consent scope миграции 000014 пока analysis/receipt. Нельзя подменить receipt
   consent новой передачей банковской выписки или считать один глобальный hash
   согласованием нового redacted state после ручных правок.
3. `documents.content` обязателен, kind только credit_card_terms. Нужны новые
   constraints/cleanup guards, без удаления байтов условий карт.
4. Import source пока sheet/row, candidate matching использует kind. Нужны PDF
   source variant и posting-based candidates до того, как AI выбрал финансовый kind.
5. Нынешний `MeteredCompletion` резервирует один call. Одна только внешняя петля
   над ним не обеспечивает документную атомарность бюджета/crash recovery.
6. `PolicyVerifier` не обновляет endpoint tag по совпадению бренда. Отдельная
   statement policy не должна тихо перенастраивать текущие AI возможности.

## Порядок исполнения после технического решения

1. Зафиксировать accepted specification и интегрировать её в main; затем отдельный
   implementation worktree. Не смешивать draft PR и application changes.
2. Offline источник: bounded extractor, immutable span/evidence, manual draft,
   source variant/DDL/RLS/retention и финансовые guards без provider calls.
3. Statement consent/budget/extraction attempts и fake-provider integration;
   новая policy по умолчанию выключена. AC по атомарности, replay, matching,
   отсутствующим book balances и regression выполняются до live model test.
4. Synthetic model qualification с явным cap внутри прежнего бюджета, отдельно
   real-reference consent; Jev только если измерено преимущество.
5. После acceptance evidence — отдельные main/CI/Delivery и production decision.

## Решение, ожидающее владельца

Архитектурное разделение уже принято и не запрашивается повторно. На review
выносится конкретный технический вариант: Poppler 25.04.0-r0 на Alpine 3.22,
Gemini 2.5 Flash через **google-vertex/eu**, новый statement-purpose/consent,
bounded chunk plan и default retention выше. Принятие этого контракта открывает
offline implementation, но не включает AI в production и не отправляет личный
reference внешнему API без document-specific consent.
