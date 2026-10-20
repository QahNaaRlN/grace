# S1.4: диагностика provider-stage failure без inference

Дата: 2026-10-20. Scope принят прямым «Хорошо, приступай»: диагностировать
исходный сбой без платного повтора. Production-код, policy, image и флаги не менялись.
База: [accepted S1.4](../../specs/0055-s14-candidate-screening.md), executable source
`e36dbcbb5c57ed8b2060cc2bcbc1ebc8482fd728`; исходный run 39459614043 остаётся FAIL.

## Подтверждённые факты

[Read-only run 39464907701](https://github.com/QahNaaRlN/grace/actions/runs/39464907701),
source `0f8a062`, читает только fixed private report и уже сохранённые generation/
reconciliation evidence. SHA256 report и metadata в обоих audit совпали;
settlementAuditCommitted=true. Новый inspector не делает GET, не читает ключи,
не обращается к SQL и ничего не записывает. Existing workflow отдельно читает
runtime flag и shared-ledger readiness через прежние SELECT; не изменяет их.

- Statement AI runtime enabled; shared-ledger readiness PASS. Это не end-to-end
  подтверждение импорта или свежая проверка доступности baseline endpoint.
- Gateway metadata: finish_reason=stop, cancelled=false; prompt/completion tokens
  в границах 100000/16000, generation_time/latency в пределах 120 секунд.
- provider_responses присутствует; в его непосредственных элементах нет непустого
  поля error. Это ограниченная структурная проверка, не доказательство HTTP 200.
- Metadata содержит streamed=true и data_region=global. Наш request builder
  отправляет stream=false и only=google-vertex/eu. Эти наблюдения нельзя незаметно
  приравнивать к исходному Content-Type или доказательству выбранного endpoint:
  сохранённого chat envelope нет. Их семантика для этого запроса UNVERIFIED.
- В metadata нет openrouter_metadata/choices, включая полный ограниченный обход
  вложенных объектов. Harness не сохранял исходный HTTP status/Content-Type/body
  и не сохранял точный код guard, отвергшего ответ.

[Официальный GET generation](https://openrouter.ai/docs/api/api-reference/generations/get-request-&-usage-metadata-for-a-generation)
возвращает request/usage metadata. Это другой контракт, чем chat completion
response, который проверяет Grace. Отдельный
[GET generation/content](https://openrouter.ai/docs/api/api-reference/generations/get-stored-prompt-completion-and-error-content-for-a-generation)
описывает сохранённый content и ответ 403 для отсутствующего management key.
Content endpoint не вызывался; management key/retention settings не запрашивались
и не изменялись. Не считаем его способом восстановить отсутствующий wire envelope.

## Что удалось воспроизвести локально

`go test ./internal/assistant -run TestScreeningFailureDiagnosis -count=1`:
14 synthetic cases PASS, только httptest localhost. Это counterexamples, не replay
утраченного ответа. Проверены именно guards исходного source e36dbcb:

| Вариант | Billing | Diagnostic detail |
| --- | --- | --- |
| Валидный exact envelope | known | без ошибки |
| Canonical slug только в верхнем model | unknown | envelope |
| Отсутствующая metadata, canonical requested/selected model, другой provider, retry attempt, pipeline, BYOK, отсутствие selected endpoint | unknown | общий provider_protocol без detail |
| Отсутствующий usage.cost | unknown | общий price_unknown без detail |
| finish length / invalid JSON content при корректном route/cost | known | finish_length / content |

Qualifier складывает ошибки без ProtocolReason в один qualification_provider_failed.
Поэтому простой перенос canonical mapping из billing recovery в верхний model
не является обоснованным исправлением: этот mismatch дал бы suffix envelope,
которого в исходном report нет. Разные metadata-route failures действительно дают
наблюдаемый общий код, но ни один из них не доказан как исходная причина.
HTTP/read/network/pre-request failures также не исключены одним общим кодом;
upstream timing не доказывает, что Grace получил полный HTTP response.

`python3 scripts/research/test-screening-diagnosis.py`: private sentinel/cost не
попадают в stdout; digest mismatch, unsafe permissions и symlink дают отказ.
Inspector не изменяет исходный report, reconciliation evidence, marker или ledger.
Обязательный `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test`
завершён exit0: Go + 334 web tests + lint/FSD/format/build/Jev PASS;
`git diff --check` PASS. Локальный журнал `/tmp/grace-s14-diagnosis-make.log`.

## Вывод и следующий шаг

Доказан дефект наблюдаемости: report теряет точное основание отказа до billing
validation. Первопричина конкретного provider failure — UNVERIFIED, восстановить
её однозначно из имеющихся evidence нельзя. Качество модели не измерено; scorer,
финансовые поля и accepted guards не менялись. Нового inference/deployment нет.

Перед новым smoke нужен узкий пакет безопасных diagnostic codes, HTTP category
и приватного correlation ID при unknown billing. [Конкретный кандидат S1.5](../../specs/0055-s15-screening-diagnostics.md)
не меняет alias acceptance, route/ZDR/schema/scorer/budget/consent и не разрешает
новый paid run. Перезапуск старого authorization marker запрещён.
