# SPEC-0026: подключение и проверка ИИ

Статус на 2026-10-09: локальная реализация в `codex/ai-assistant-receipts`.
В production ИИ не включён. Аккаунт OpenRouter и пополнение владелец подтвердил;
ключ ещё не установлен. Сравнение моделей и передача реальных данных не выполнены.

## Что подготовлено

Серверный OpenRouter adapter, точный бюджет в USD micro-units, owner-scoped
согласие, объяснение календарного месяца с числами Go, сохраняемые разговоры.
Фото чека нормализуется без EXIF и хранится временно. Распознавание возвращает
черновик; редактирование не меняет книгу. Preview связан с revision и снимком
книги. Подтверждение атомарно создаёт покупку + Receipt либо добавляет Receipt
к существующей покупке; повтор возвращает прежний результат. Cleanup после
confirm/cancel и ежечасный TTL продолжается после перезапуска.

Jev: подготовлены типизированный вопрос выбора товарной группы, no-match,
проверка распределения вероятностей и порт предложений. Live adapter и
эмпирически выбранный threshold ещё не подключены; ручная группа доступна.
Нельзя считать это завершением AC4 или передавать товарные строки TypeSafe
на основании согласия, которое называет только OpenRouter.

## Ключ: действие владельца на сервере

Ключ никогда не отправляется в чат, Git или браузер. На сервере оператор
запускает интерактивный Bash, проверяет каталог существующих секретов и вводит
ключ в скрытый prompt. Команды отказываются перезаписывать существующий файл:

```bash
test -d /opt/grace/shared/secrets && test ! -e /opt/grace/shared/secrets/openrouter_api_key && (
  umask 077
  read -r -s -p 'OpenRouter API key: ' grace_openrouter_key
  printf '\n'
  test -n "$grace_openrouter_key" && ( set -C; printf '%s' "$grace_openrouter_key" > /opt/grace/shared/secrets/openrouter_api_key )
  unset grace_openrouter_key
)
```

Не включать `set -x` и не выводить файл. В приложении используется
`OPENROUTER_API_KEY_FILE`; при выключенном ИИ файл не требуется.
После ввода оператор с правами root задаёт файлу владельца `10001:10001`
и mode `0400`: это закреплённый UID/GID пользователя API в Dockerfile.
Каталог серверных секретов остаётся `root:root`, mode `0700`.
Локальный Docker Compose bind-secret сохраняет ownership исходного файла;
нельзя полагаться на `uid/gid` в short secret syntax. При установке необходимо
проверить чтение файла из API-контейнера без вывода содержимого.
Опциональный `deployments/production/compose.ai.yaml` монтирует ключ через
Docker secret и отдельный временный volume. Init-контейнер без сети задаёт
volume владельца `grace` и mode `0700`; API работает без root. Основной production compose
сохраняет выключенный ИИ. Дополнительный compose ещё не входит в автоматический
release pipeline: его включение и подключение deploy/rollback выполняются
перед отдельным rollout, после проверки моделей и backup/restore.

## Бюджет и маршрут перед включением

Владелец сообщает фактически приобретённые credits и сумму комиссий.
Оператор устанавливает `ai_experiment_budget.allowance_micros` и `fees_micros`
под административной ролью. БД не допускает сумму выше 10 000 000 ($10).
Не увеличивать лимит на основании номинального платежа и не обнулять ai_usage.
Неопределённая стоимость блокирует платные вызовы до сверки.

Для `GRACE_AI_ANALYSIS_POLICY` и `GRACE_AI_RECEIPT_POLICY` проверяются
точные model/provider IDs, structured outputs, vision, pricing и ZDR endpoint.
Policy JSON содержит `model`, `provider`, `inputMicrosPerMillion`,
`outputMicrosPerMillion`, `verifiedAt` (RFC3339), `vision` и
`imageTokenUpperBound`. Цены выражены в USD micro-units за миллион tokens;
например, цена $0.50 — 500000. Проверка действительна 24 часа. Для vision
upper bound должен покрывать нормализованное изображение до 2048 px;
если верхнюю оценку нельзя подтвердить, vision не включается.

Запрос требует `provider.only`, `allow_fallbacks=false`, `require_parameters`,
`data_collection=deny`, `zdr=true`, `max_price`; external tools/plugins отсутствуют.
Ответ проверяется на модель/provider, полный JSON, usage и стоимость.
Первые реальные запросы — только synthetic fixtures. Отдельное согласие в
Grace требуется перед передачей личного анализа или чека, отдельно от
подтверждения расхода. Общий лимит включает будущие платные Jev-вызовы.

## Одинаковый набор оценки

```bash
go run ./cmd/ai-eval
go run ./cmd/ai-eval -render testdata/ai-evaluation/images
go run ./cmd/ai-eval -score /protected/path/model-results.private.json
```

CLI офлайн: не делает сетевых запросов и не пишет финансовые данные.
Корпус `synthetic.v1.json` содержит 20 чеков (18 читаемых, 2 неизвестных),
20 вопросов (пустая/неполная история, несколько валют, ложные предпосылки,
injection) и 30 товарных строк с no-match. PNG содержат только искусственный текст.
Для 2–3 моделей сохраняется одинаковый набор результатов, все ошибки,
latency, input/output tokens, cost и item-level discrepancies. Файл результатов
содержит `model`, `provider`, `receipts`; каждая запись содержит `id`, `total`,
`currency`, `mandatoryUnknown`, `error`, `latencyMs`, `inputTokens`,
`outputTokens`, `costMicros`, `itemErrors`. Scorer учитывает все cases,
включая ошибки, а не только успешные ответы. Порог для читаемой группы ≥95%.
Для неизвестных обязательных полей запись блокируется независимо от модели.

Jev оценивается отдельно; threshold выбирается по precision/coverage,
не устанавливается по произвольному значению. После выбора требуется
подключить metered adapter и disclosure TypeSafe. До этого ручной ввод остаётся.

## Отключение и сохранение данных

`GRACE_AI_ENABLED=false` останавливает inference. Уже подтверждённые покупки
и Receipt остаются финансовой историей. Каталог фото остаётся подключённым,
пока cleanup не удалит pending/expired объекты. Backup содержит pg_dump,
но не временный volume фото. Down-migration отказывается удалять подтверждённую
AI provenance, usage или неудалённые фото. Перед production rollout обязательны
backup/restore, CI, выбранные endpoint policies и live evidence.

Официальные контракты проверены 2026-10-09:
[OpenRouter routing](https://openrouter.ai/docs/guides/routing/provider-selection),
[ZDR](https://openrouter.ai/docs/guides/features/zdr),
[TypeSafe API](https://docs.typesafe.ai/api),
[Choice](https://docs.typesafe.ai/primitives/choice).
