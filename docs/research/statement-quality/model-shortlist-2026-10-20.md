# Shortlist Statement AI — 2026-10-20

Статус: metadata researched; quality/latency **UNVERIFIED**. Реестр production
не изменён. [Воспроизводимый snapshot](model-shortlist-2026-10-20.json),
`checkedAtUtc=2026-10-20T13:31:16.615008+00:00`; обновление:
`python3 scripts/research/statement-model-shortlist.py` (только публичные GET,
без ключей, личных данных и model inference).

| Модель / exact endpoint | USD / 1M input / output | Ceiling на 100k input + 16k output, без fees | Metadata / решение |
| --- | --- | --- | --- |
| Baseline `google/gemini-2.5-flash` / `google-vertex/eu` | В этом snapshot точная пара отсутствует | Approved baseline $0.0700; это прежний ceiling, не свежая цена отсутствующего endpoint | В публичном models/endpoints и ZDR snapshot EU-пара не найдена; текущую server readiness этим не проверяли. Нельзя незаметно заменить route на AI Studio |
| `google/gemini-3.1-flash-lite` / `google-vertex/eu` | 0.275 / 1.65 | **$0.0539** | Metadata: status 0, structured outputs, context/output limits подходят, reasoning none поддерживается моделью; exact EU pair есть в ZDR list. Первый кандидат на отдельно согласованный screening, не qualified и не allowlisted |
| `inception/mercury-3` / `inception` | 0.10 / 0.35 | **$0.0156** | Metadata: status 0, structured outputs, caps подходят, reasoning none; exact pair в ZDR list. Новый provider/region profile требует review, качество/совместимость фактического wire не проверены |

Цены — snapshot точного endpoint, не обещание цены будущего вызова. Ceiling
считается Decimal → integer USD micros, без предположений о cache/скидках;
для сравнения берётся весь bounded plan. Fees и shared budget не увеличиваются.
Низкая цена токена не доказывает низкую стоимость успешно разобранной выписки.
Metadata reasoning capability не заменяет fake/real wire proof и paid smoke.

Gemini 3.8 Flash дополнительно просмотрен как более свежий вариант:
[официальный endpoint metadata](https://openrouter.ai/api/v1/models/google/gemini-3.8-flash/endpoints)
не показывает прежнюю EU-пару, standard цены выше прежних ceilings; catalog
указывает mandatory reasoning. В shortlist для текущего профиля не включён;
более новая дата сама по себе не даёт допуска. Чтение metadata не проверяет
реальную задержку и не является inference.

Источники: [каталог моделей](https://openrouter.ai/api/v1/models),
[Gemini 3.1 Flash Lite endpoints](https://openrouter.ai/api/v1/models/google/gemini-3.1-flash-lite-20260507/endpoints),
[Mercury 3 endpoints](https://openrouter.ai/api/v1/models/inception/mercury-3-20261019/endpoints),
[baseline endpoints](https://openrouter.ai/api/v1/models/google/gemini-2.5-flash/endpoints),
[реестр ZDR](https://openrouter.ai/api/v1/endpoints/zdr),
[смысл ZDR у OpenRouter](https://openrouter.ai/docs/guides/features/zdr).
Наличие в ZDR list означает опубликованную policy маршрутизации OpenRouter,
не независимую юридическую гарантию резидентности всей цепочки.

## Следующий bounded run — предложение, не разрешённые расходы

Предпочтительный первый кандидат: Gemini 3.1 Flash Lite / Google EU, чтобы
сохранить endpoint family и прежние ceilings. После принятия технического
контракта candidate-only policy и offline wire guards: columns → cards →
turnover-roles-ru, не более 3 calls, provider cap **161700 micros ($0.1617)**
при snapshot ценах, прежний общий бюджет и fees, whole-run reserve, без retry,
без private PDF, без enable только по этому smoke. Смена metadata/цен требует
нового preflight; отказ до inference при превышении cap. Имена/golden/hashes
fixture фиксируются до запуска. Это новый run, прежний двухвызовный S1.2
allowance не переносится. Полный corpus и автоматический registry — отдельные
пакеты; один PASS не доказывает устойчивость всех банков.
