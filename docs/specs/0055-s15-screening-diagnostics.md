# SPEC-0055 / S1.5: диагностика отказов screening

- [Интеграция PR №78 и отдельные post-merge проверки](0055-pr78-postmerge.md).
- Дата: 2026-10-20, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: integrated (PR №78, `29a28a7`, 2026-10-21); проверки (D1–D5 PASS); not_deployed.
- [Implementation evidence](0055-s15-evidence.md).
- Acceptance: прямое «Утверждаю» в ответ на S1.5 без нового smoke и смены модели.
- Основание: [read-only diagnosis S1.4](../research/statement-quality/s14-provider-failure-diagnosis.md).
- Утверждение охватывает только это bounded исправление; новый live smoke,
  production adoption и deployment в пакет не входят.

## Проблема и результат

S1.4 upstream generation завершился, но локальный report сохранил только
qualification_provider_failed / unknown billing. Разные guards дают этот код,
поэтому ID пришлось запрашивать вручную, а точная причина утрачена. Новый
operator-only screening report должен сохранять безопасное основание отказа,
не принимая ранее отвергавшиеся ответы и не включая personal/provider text.

## Принятый контракт

1. Только screeningDocument path получает закрытую классификацию: request,
   transport, timeout, http_4xx/http_5xx/http_other, read/oversize, envelope,
   metadata_missing/requested/strategy/attempt/byok/pipeline,
   selected_model/provider/count/display_provider, cost/usage/token_bound,
   choices/tools/finish_length/finish_other/content и unknown. Product transport
   и его error/BillingKnown semantics сохраняются. Любой неизвестный reason
   сводится к unknown, произвольный текст в report/log не попадает.
2. Для screening сохранить только HTTP integer status и category Content-Type
   (json/event_stream/other/missing) при наличии response. Без headers, raw body,
   request/response content, upstream error message или key. Read/size ошибки
   сохраняют unknown billing. streamed/data_region из generation metadata не
   подменяют wire evidence и не служат доказательством региональности.
3. Из успешно разобранного envelope приватно сохранить correlation ID только
   при regex ^gen-[0-9A-Za-z-]+$ и length<=128. Это unverified correlation,
   не подтверждение model/provider/cost. Его нельзя использовать в качестве
   known Billing, для auto-settlement/activation или смены маршрута. На ошибках
   до decode ID отсутствует. CLI summary публикует fixed codes/booleans.
4. Отдельный version screening report; production activation продолжает reject.
   Ошибочный attempt остаётся FAIL с пустыми scored results, consumed marker и
   uncertain ledger при unknown billing. Failed attempt diagnostic отделён от
   verified calls и не создаёт фиктивный нулевой cost. Старые reports immutable.
5. Guards model/alias/provider/attempt/BYOK/pipeline, schema/scorer, caps,
   privacy/ZDR/deny/only EU, shared budget/fees, no retry/fallback не ослабляются.
   Ни новой модели, ни её canonical slug автоматически не разрешать.

## Критерии приёмки

| ID | Исполняемое evidence |
| --- | --- |
| D1 | Локальные HTTP fixtures различают каждый guard и I/O/status/size failure; accepted/rejected ответы и BillingKnown совпадают с baseline |
| D2 | Report сохраняет fixed code/status/category, valid unverified ID только приватно; unknown cost остаётся uncertain, нет auto-reconcile/retry |
| D3 | Sentinel в request/content/error/headers и malformed ID не появляется в summary/error text; неизвестный reason → unknown |
| D4 | Candidate report rejected activation; product path behavior сохранён; marker/old report не меняются |
| D5 | make test, affected fake screening/integration and deployment mocks, git diff --check; число external inference=0 |

После локальной проверки отдельное решение по новому bounded smoke с новым
marker, cap и authorization. S1.4 fail и его reconciliation не переписываются.
