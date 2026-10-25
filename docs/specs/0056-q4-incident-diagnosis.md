# SPEC-0056 Q4: read-only разбор preflight INCONCLUSIVE

Дата: 2026-10-25, Asia/Yekaterinburg. Статус: **VERIFIED diagnosis / READY_FOR_OWNER_REVIEW**.
Основание: прямое поручение владельца «приступить к разбору инцидента» после
предложения отдельного read-only разбора сохранённого report без новых вызовов модели.
[Исходный Q4](0056-q4-evidence.md) остаётся **INCONCLUSIVE / CONSUMED**.
Диагноз не является разрешением repair, изменения тарифа или нового Q4.

## Причина и причинный путь

Endpoint `google-vertex/eu` исключён в `ModelGroups.refresh` ценовым guard:
**поле `pricing.request` отсутствует** (`requestFeeKeyPresent=false`;
это не null, пустая строка или числовой JSON type).
Код требует явные request/image rates; отсутствие сведений не трактуется как0.
Проверка `request == ""` в блоке flat fees срабатывает первой и исключает
единственный endpoint кандидата. Политика statement не выдаётся,
`CurrentDocumentPolicy` возвращает отказ, `runCore` заканчивает preflight.

Дополнительный независимый blocker: имеющийся `pricing.image` **не проходит
нулевой ImageFeeMicros ceiling**. Это проверенный результат предиката,
а не подтверждение конкретного размера/семантики image charge. Заполнение
одного отсутствующего request fee не устранило бы этот второй blocker.
Сохранённое image price — валидная положительная decimal string; это
подтверждено отдельно закрытыми type/positive predicates. Raw image price
не выводился; его численное значение и billing semantics
этим диагнозом не утверждаются. Потребуется отдельно подтвердить их прежде,
чем предлагать изменение cost-контракта.

| Проверка actual сохранённой metadata | Результат |
| --- | --- |
| Catalog, ZDR, endpoint HTTP | Все3 ответа200, transportFailure=false, JSON content type |
| Catalog: exact model/canonical + text/image/output + parameters | Ровно1 matching candidate |
| ZDR: exact model/tag/status | Ровно1 matching candidate |
| Endpoint exact tag | Ровно1 строка |
| model_id/provider name/status/context/input/output/parameters | Все используемые guard predicates PASS |
| Prompt≤$0.275/M, completion≤$1.65/M | Оба PASS |
| Request fee explicitly present | **FAIL** |
| Request fee≤0 | **FAIL**, вследствие отсутствия значения |
| Image fee explicitly present | PASS |
| Image fee≤0 | **FAIL** |

Source guard: `internal/assistant/model_groups.go`, `ModelGroups.refresh`,
блок `request, image := ep.Pricing.Request, ep.Pricing.Image` и следующий `if`;
затем `len(policies)==0 → no_eligible_model`. SHA256 файла на producer source:
`da8d3ec917daf55730507ef2c38182ee013664d3958854e7a774ab6e94b59361`.
`model_name` не используется этим identity guard; его несовпадение не объявляется
причиной. Все выводы относятся к сохранённому snapshot запуска, не к текущему
состоянию внешнего каталога OpenRouter.

## Как получено evidence

Выполнены **две** root SSH read-only sessions (основная и уточнение типов цен) на pinned `195.133.195.190`.
Стандартный Python reader поступил через stdin, исполнился с `env -i`, `-I -B`,
timeout35s; SCP, staging, Docker/SQL/provider clients и запись файлов на VPS
не использовались. Reader открывал только два fixed application paths:

- `/opt/grace/shared/document-reading-spec0056-q.started`;
- `/opt/grace/shared/document-reading-spec0056-q.private.json`.

Оба файла открыты `O_RDONLY|O_NOFOLLOW|O_CLOEXEC|O_NONBLOCK`, regular0600;
marker≤512B, report≤32MiB. Before/after сравнивались inode/device, mode,
uid/gid, size, mtime/ctime и SHA256. Source/manifest/approval в marker и report
совпали с опубликованным Q4 receipt. JSON duplicate/case aliases отвергаются.
Sealed report digest пересчитан и совпал с известным public summary digest.
Только после проверок из base64 metadata вычислены закрытые booleans/counts.
Provider bodies, произвольные строки, keys, UUID, DSN и env наружу не выводились.

| Identity / read-only evidence | Actual |
| --- | --- |
| Producer source | `9200b731a0dea8f171e31aaf988d038073ad577f` |
| Manifest SHA256 | `10200604c2a134577b3d22eed84fd106d7d951b1aececa0ec47f978dbf85653a` |
| Report object digest | `a3220ca4f7a82a69e39c62aff70afde6f2a9cf9a718ca88794bf0a67fe7a84be` |
| Private report file SHA256 | `3715f848ab53cf08d856ec1a45ec06f6386bff6626362a4f8f065dc84e34ad7f` |
| Marker file SHA256 | `d1363566f876907a9460b5e1bcd50e0e977b09e7d329c6ea730ae14dc4e71de9` |
| Fixed reader source SHA256 | `3ce0cfb36568e624509abe0481ed41686125879df18a791db55476cbd25debeb` (финальный reader с type predicates) |
| Snapshot unchanged | **true** |
| Дополнительные provider/metadata calls | **0** |
| SQL calls / persistent writes requested | **0 / 0** |
| Stored POST / metadata GET | **0 / 3**, это исторические counters Q4 |
| Stored budget events | **0** |

[Публичная закрытая проекция](../research/statement-quality/q4-40186488958.diagnosis.json)
сохранена в Git. Private report на VPS не менялся и не скачивался.
Report/marker file hashes совпали между обеими sessions; оба before/after
снимка неизменны. Для воспроизводимого аудита сохранены
[fixed reader](../research/statement-quality/q4-incident-reader.py) и
[локальный fake control](../research/statement-quality/q4-incident-reader-test.py).
Reader — research artifact только для этого fixed producer/report/marker;
он не добавлен в product CLI/workflow и не является допуском следующего SSH.
Test инъецирует synthetic manifest/report identities только в локальные globals
и fixed temp-root; исходный reader и его production constants не меняются.

## Проверки и границы заключения

`python3 -B docs/research/statement-quality/q4-incident-reader-test.py` PASS: positive fixed snapshot,
missing request/model predicates, отсутствие PRIVATE marker в projection,
отказы wrong mode/symlink/подменённый marker; network0.
Reader canonical JSON hash совпал с actual producer report digest до выдачи VERIFIED.
`make test` PASS, web339/339 и Go/lint/format/build/Jev/corpus;
лог `/tmp/grace-q4-incident-make-test.log`. `git diff --check` PASS.
В diff только research artifacts/документация; product code, guards, цены,
OpenAPI, prompts/schema/golden и production config не менялись.

Общий бюджет не являлся достигнутым этапом: stored budget events0 и отказ
произошёл при первом statement policy resolve до `budget.Remaining`.
Это **не доказывает**, что доступного allowance хватит после исправления.
Metadata200 не квалифицирует inference key/account для платного POST.
Распознавание, token usage и inference latency по-прежнему UNVERIFIED.

## Следующий возможный пакет

Предложить небольшой offline repair-контракт для явной семантики
request/image pricing: подтвердить значение и единицы официального API,
отделить реальную дополнительную комиссию от token-priced image input,
сохранить fail-closed поведение и пересчитать полные conservative reserves.
Не подставлять отсутствующую цену0 и не повышать потолок молча.

Изменение resolver/candidate/rates, production rollout и новый paid attempt
этим read-only разбором **не выполнялись и не разрешены**. Старый grant остаётся
CONSUMED, marker сохраняется. Перед новым запуском нужны принятый repair,
его executable evidence и отдельное разрешение Q4 на новый exact source/manifest.
