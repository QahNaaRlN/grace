# SPEC-0034: реализация и проверка

Дата: 2026-10-10. Спецификация: [accepted SPEC-0034](0034-commitments-ai-reminders.md).
Ветка: `codex/commitments-ai`, база `d475cca78275d5fb18c0901508734dfdb6dcabf6`.
Статус: локальная реализация; production включение и серверные счётчики ещё не
проверены. Пользовательский сценарий: [гайд](../commitments-user-guide.md).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | Подписки/пауза/конкретная карта; календарь Go; idempotent create; связь и история реальных покупок | `go test ./internal/commitments`; `scripts/test-commitments.sh`: повтор create/link, чужой account, старый связанный purchase |
| AC2 | Сведения кредита, private original PDF, версии/SHA-256/лимиты | `scripts/test-commitments.sh`: original bytes, чужой download, non-PDF rejection; SQL owner FK/RLS и SELECT/INSERT-only документы |
| AC3 | Copy/edit/apply draft, snapshot/revision, отдельное подтверждение, сохранённые оплаты | `scripts/test-commitments.sh`: stale settlement, retained paid IDs, inactive history; `LoanPanel.test.tsx`: paid fields disabled, dirty apply blocked, separate confirm |
| AC4 | Durable 3/1 outbox, verified owner, quota, no uncertain retry | `scripts/test-commitments.sh`: restart, uncertain and 8 concurrent workers with fake SMTP; реальных писем в тестах нет |
| AC5 | Synthetic comparison, exact policy/privacy route, bounded usage, timestamp refresh without price/model changes | Live результаты ниже; `go test ./internal/assistant`; `scripts/test-commitments-rollout.sh`: config permissions, prior AI policy rollback, UID gate, incompatible old release refusal |
| AC6 | OpenAPI regeneration, единая Radix UI, desktop/mobile | `make test`: Go + 19 web files/67 tests + lint/build + offline datasets; полный `make test-integration` прошёл, затем дополненный commitments suite повторён; `git diff --check` |

На Windows полный `make test-production` упирается в POSIX symlink (Git Bash
делает копию вместо ссылки), до production действий. Новый rollout test и
существующий email rollback test отдельно проверены в Linux alpine:3.22 с mock
Docker; live server они не трогают. Полный production gate должен пройти Linux CI.

Миграция 18 добавляет owner-scoped сущности и `plan_active`; финансовые суммы
считаются только по активному плану. DOWN отказывается удалять новые записи или
историю. После появления неактивных строк старое приложение до миграции 18
небезопасно для чтения: deploy rollback отказывается его запускать. Требуется
совместимый выпуск; резервная копия не восстанавливается автоматически поверх
пользовательских данных.

## OpenRouter: 80 синтетических попыток, ни одного реального документа

Одинаковый checked-in корпус: 20 чеков (18 читаемых, 2 с обязательными unknown)
и 20 вопросов на модель. Маршрут: exact model + `google-vertex` / Google,
`zdr=true`, `data_collection=deny`, no fallback, structured output, no tools/plugins.

| Модель | Точные итог/валюта читаемых чеков | Unknown блокированы | Валидные объяснения | Учтённый расход, USD |
|---|---:|---:|---:|---:|
| Gemini 2.5 Flash Lite | 14/18 (77,78%) | 2/2 | 19/20 | 0,008147 |
| Gemini 2.5 Flash | 18/18 (100%) | 2/2 | 19/20 | 0,025881 |

Выбраны Flash для чеков (порог >=95% выполнен на этом небольшом корпусе),
Flash Lite для объяснений (такой же результат проверки, меньшая стоимость).
Метрика чека проверяет итог/валюту и обязательные unknown; она не доказывает
100% точность всех товарных полей на реальных фото. Они всегда проверяются
пользователем в черновике. У обеих моделей один ответ на пустую историю отклонён
валидатором unsupported_claim: Grace оставляет достоверные числа Go и предупреждение.

Одна попытка Lite receipt-03 завершилась provider_unavailable. Платные вызовы
остановлены до сверки; разница key usage совпала с двумя известными invoice
(0,000612 USD), неуспешная попытка не тарифицирована. Она осталась ошибкой в
метрике и не перезапускалась автоматически. Остальные 79 попыток имеют стоимость;
локальный журнал содержит 80 settled records, включая сверенную нулевую попытку.

До запуска usage ключа 0,00077520 USD, после 0,03477389 USD: фактическая разница
**0,03399869 USD**. Сумма округлённого вверх журнала: 0,034028 USD. Лимит ключа
5 USD, остаток **4,96522611 USD**, без reset. Пополнение 10 USD, комиссия 0 USD
по сообщению владельца. Raw responses/ключи/usage snapshots не входят в Git.

Публичные endpoint prices проверены: Lite 0,10/0,40 USD за миллион input/output
tokens; Flash 0,30/2,50. Vision ограничен геометрией нормализованного JPEG и
верхней границей image tokens; длинное изображение, не помещающееся в лимит,
отклоняется до отправки. Цены/идентичность перепроверяются после 23 часов; при
росте цены, изменении маршрута или недоступности проверки AI блокируется.
Цены не повышаются автоматически. In-app consent сохраняется обязательным;
включение сервера само по себе не разрешает отправку реальных данных.

## Стоимость Yandex и хранения

Источник, проверен 2026-10-10:
[Postbox pricing, обновлено 15 сентября 2026](https://yandex.cloud/ru/docs/postbox/pricing).
Первые 2 000 исходящих писем/месяц бесплатны; следующие до 10 000 стоят
80,31666666 ₽/1 000 с НДС. Считаются принятые письма на каждого получателя,
включая впоследствии недоставленные.

При 1 платеже в месяц напоминания дают 2 письма; 10 подписок + 3 кредита дают
26 писем, 50 подписок + 10 кредитов — 120, плюс регистрация/восстановление.
Недельная подписка даёт 8–10 писем/месяц. Штатный общий cap Grace 1 500 означает
**0 ₽ дополнительной платы Postbox**, если нет других отправителей, расходующих
те же бесплатные 2 000. Это сценарий по лимиту, **не измеренный счёт Yandex**.
Серверные reservations и облачный биллинг пока не получены. При уже выбранном
free tier дополнительные 100 писем стоят около 8,03 ₽ до следующего порога.

База, обработчик напоминаний и документы используют существующий VPS;
новых Yandex Compute/Storage ресурсов нет. PDF cap 100 МиБ/владелец: при двух
владельцах максимум 200 МиБ originals, до 1,4 ГиБ на семь несжатых daily copies,
плюс основная база, manual backups и WAL. Реальная компрессия/свободное место
проверяются на сервере. Стоимость существующего VPS не изменяется этим кодом;
дополнительный диск автоматически не заказывается.

Средний расход synthetic: 100 чеков Flash около 0,1085 USD; 100 объяснений
Lite около 0,0131 USD. Это сценарий по наблюдаемому среднему, не фиксированная
цена. Жёсткие token bounds дают верхнюю reservation 0,008 USD/чек и
0,0018 USD/объяснение; фактическое потребление сверяется, uncertain блокирует
следующие обращения. Перед production enable проверить свежий key balance и
установить остаточный allowance не больше 4,90 USD, сохранив прежний usage.
