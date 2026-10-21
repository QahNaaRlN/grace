# Statement AI: metadata для сравнения после S1.10

Дата проверки: 2026-10-21 12:37:42 +05:00 / 07:37:42 UTC.
[Snapshot публичных GET](model-shortlist-2026-10-21.json), collector
`scripts/research/statement-model-shortlist.py` на `2fe0d79`, providerCalls=0.
Старый snapshot 2026-10-20 сохранён. Новых inference и чтения production нет.

| Alias / canonical | Exact endpoint / display | USD за 1M input / output | Provider ceiling на документ | Решение |
| --- | --- | --- | --- | --- |
| `google/gemini-2.5-flash` / тот же canonical | `google-vertex/eu` / не подтверждён | Точная пара отсутствует | UNVERIFIED; прежние $0.07 не являются текущей ценой | Исключён из нового live comparison; без подмены endpoint |
| `google/gemini-3.1-flash-lite` / `google/gemini-3.1-flash-lite-20260507` | `google-vertex/eu` / Google | 0.275 / 1.65 | 53900 micros ($0.0539) | Кандидат A, metadata compatible, quality/latency UNVERIFIED |
| `inception/mercury-3` / `inception/mercury-3-20261019` | `inception` / Inception | 0.10 / 0.35 | 15600 micros ($0.0156) | Кандидат B, metadata compatible; отдельное принятие provider profile, quality/latency UNVERIFIED |

Для A/B collector подтвердил status=0, exact pair в ZDR list, metadata
поддержки structured_outputs/response_format/max_tokens/reasoning, возможность
reasoning=none, context >=116000 и max completion >=16000. Это metadata, а не
доказательство реального wire или качества извлечения. Collector v1 сохраняет
историческое имя reason `current_s13_quality_unverified`; оно не означает
проверку качества нового S1.10. Весь новый профиль остаётся UNVERIFIED до run.

Расчёт в integer USD micros, Decimal/ceil: inputBound=100000,
outputBound=16000. A: 27500+26400=53900; B: 10000+5600=15600.
Три документа на каждого: 161700+46800=208500 micros ($0.2085), без fees.
Это предлагаемый потолок будущей кампании, не разрешённые расходы; общий
ledger/fees/$10 limit не меняются. Свежий preflight обязателен перед POST.

Проверены [официальные правила routing](https://openrouter.ai/docs/guides/routing/provider-selection),
[ZDR](https://openrouter.ai/docs/guides/features/zdr) и
[structured outputs](https://openrouter.ai/docs/guides/features/structured-outputs).
ZDR — policy конкретного endpoint; наличие в списке не доказывает EU residency
всей цепочки. Inception не объявляется EU route. В предлагаемом сравнении ему
могут передаваться только публичные synthetic fixtures после принятия контракта.
Для личных данных и production допуска отдельное решение остаётся обязательным.

Воспроизведение только публичных metadata:

```sh
python3 scripts/research/statement-model-shortlist.py > /tmp/grace-model-shortlist.json
```

Hashes исходных models/ZDR/endpoints, timestamp, canonical identity и цены
сохранены в JSON. Изменение alias/canonical/endpoint или превышение ceilings
останавливает будущий preflight, не выбирает замену. Исторические S1.8 FAIL17/19
и S1.10 current-trace16/19 не считаются новыми измерениями кандидата A.
