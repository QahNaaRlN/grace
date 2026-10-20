# S1.7: разбор selected_model без inference

Дата: 2026-10-21. Baseline `0b98049`; production code не изменён.
[Кандидат контракта](../../specs/0055-s17-selected-model-identity.md).

## Что доказано

S1.6 дошёл до exact selected model guard, но значение строки утрачено.
Текущее строгое сравнение отвергает и synthetic canonical выбранной модели.
Команда `go test ./internal/assistant -run TestScreeningFailureDiagnosisSeparatesEnvelopeRouteAndContent -count=1 -v`
exit0: 24 сценария, каждый screening/product; canonical_selected_model даёт
selected_model/unknown billing в screening. Это контрпример совместимости,
не replay исходного provider response. Лог: `/tmp/grace-s17-baseline-guards.log`.

## Публичные metadata

Чтение GET без ключей, redirects, PDF и inference, с timeout15s/лимитом8MiB.
Время проверки: **2026-10-21 02:17:55 +05:00** (21:17:55 UTC предыдущего дня).

- [models](https://openrouter.ai/api/v1/models), SHA-256 `3d2576ce2600c559fe930ff3382db14bad773cac863487e5e5bb3d31e50e53c0`.
- [endpoints](https://openrouter.ai/api/v1/models/google/gemini-3.1-flash-lite/endpoints), SHA-256 `e31a1c5146b548465f5a4efb4e956f335e2dd442092a0acf583e5211df2d6929`.

Каталог связывает exact id `google/gemini-3.1-flash-lite` с canonical_slug
`google/gemini-3.1-flash-lite-20260507`. EU endpoint: model_id равен alias,
tag `google-vertex/eu`, provider_name `Google`, status0. Его name содержит
canonical, однако **name не является wire endpoint.Model** и не может
заменить catalog proof или исторический response. Prompt/completion prices
`0.000000275` / `0.00000165` за token совпадают с ранее принятым ceiling.
Это snapshot, а не гарантия будущей доступности, ZDR или живого маршрута.

Сырые публичные снимки локальны: `/tmp/grace-s17-public-models.json`,
`/tmp/grace-s17-public-endpoints.json`; безопасная выжимка:
`/tmp/grace-s17-public-summary.json`. Выжимка не содержит ключей или private ID.

[Официальная документация router metadata](https://openrouter.ai/docs/guides/features/router-metadata)
различает requested alias и выбранный endpoint, описывает selected flags,
номер attempt и pipeline. [Схема каталога](https://openrouter.ai/docs/api/api-reference/models/list-all-models-and-their-properties)
содержит id и canonical_slug. Документы не доказывают, какое имя было в
утраченном ответе S1.6. Допуск точной пары — предлагаемое решение Grace,
а не установленное требование принимать любой иной model slug.

## Выбор ограниченного решения

Предлагается допускать exact canonical только в selected endpoint screening,
после свежего GET точной пары; top-level/requested и product path сохраняют
строгость. Закрытый класс selected model позволит отличить эту гипотезу от
other/empty/multiple при будущем отдельно разрешённом run без raw text.
Старые S1.6 artifacts/recovery не меняются. Новый paid smoke этим разбором
не выполняется и не разрешается. Production implementation — not_started
до принятия S1.7. Результат диагностики не равен quality PASS/adoption.
