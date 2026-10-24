# OpenRouter и чтение документов — SPEC-0056

Реализация G1–G4 принята и интегрирована PR122/main10a0059. Production не переключён;
качество реальной модели на документах Grace пока **UNVERIFIED**.
Контракт: [SPEC-0056](specs/0056-openrouter-document-reading.md).

## Как работает чтение

Выписки принимаются как PDF или JPEG/PNG/WebP, чеки — как фото или PDF.
PDF разбирает существующий изолированный helper: локально строит изображения
всех страниц, сохраняет настоящий текст и геометрию, если они доступны.
В OpenRouter отправляются изображения data URL и доступный текст; исходный
PDF, публичная ссылка и внешние OCR/file-parser plugins не отправляются.

Helper имеет ограниченный tmpfs 32 MiB для исходника до 10 MiB и PNG до 8 MiB;
сетевой режим none и остальные sandbox guards сохранены.
Первый visual reader принимает PDF целиком до 4 страниц/10 MiB, до 200 операций
выписки либо 50 позиций одного чека. Изображения PDF — PNG до 1800 по длинной
стороне, вместе до 8 MiB. Фото — до 8 MiB/10 миллионов пикселей; нормализация
удаляет метаданные и применяет orientation. Большой документ не обрезается.
Прежний ручной native PDF сценарий до 50 страниц сохраняется.

Перед отправкой сервер показывает конкретную модель, provider, все страницы
и потолок стоимости. Изменение источника/плана требует нового согласия.
Вызов один; автоматического повторения, подмены модели и fallback нет.
Неизвестное списание сохраняет резерв и блокирует дальнейший расход до сверки.

В native PDF значения по-прежнему связываются с настоящими spans и проходят
прежние проверки. Фото и сканы имеют режим `image_review`: значения, цитаты и
области на странице предложены моделью, независимый OCR не заявляется.
Владелец сверяет изображения, исправляет значения, проверяет каждую строку,
страницы и роли итогов. Правка строки снимает её digest-подтверждение.
Проверка источника не выбирает финансовую семантику автоматически: затем
действуют прежние решения строк, поиск совпадений и preview/confirmation.
Распознавание не создаёт проводок. Чек также остаётся редактируемым черновиком
до отдельного подтверждения расхода либо привязки к существующей покупке.

Просмотр страниц приватный, с owner-проверкой и `no-store`. После удаления,
истечения retention либо подтверждения с `delete_after_confirmation`
источник недоступен; минимальное evidence результата остаётся.

## Конфигурация оператора

`GRACE_AI_MODEL_GROUPS` — JSON-массив групп с назначением `analysis`, `receipt`
или `statement`. Профили: `text_reasoning`, `document_reading`.
Это список квалифицированных оператором участников; каталог не добавляет
модели автоматически. Само наличие image input не доказывает качество чтения.
Ниже схема примера, **не рабочая квалификация и не инструкция активации**:

```json
[
  {
    "revision": "operator-reviewed-v1",
    "purpose": "statement",
    "profile": "document_reading",
    "members": [
      {
        "policy": {
          "model": "vendor/concrete-version",
          "provider": "provider/eu",
          "providerName": "Provider",
          "inputMicrosPerMillion": 100000,
          "outputMicrosPerMillion": 100000,
          "imageTokenUpperBound": 2000,
          "requestFeeMicros": 0,
          "imageFeeMicros": 0,
          "vision": true
        },
        "canonical": "vendor/canonical-version",
        "priority": 0,
        "maxImages": 4,
        "maxImageSide": 2048,
        "region": "eu",
        "qualification": "REPLACE with accepted task-quality evidence",
        "qualifiedUntil": "2000-01-01T00:00:00Z"
      }
    ]
  }
]
```

Срок примера истёк намеренно. `region`, если задан, должен соответствовать
суффиксу точного endpoint tag. Регион endpoint не доказывает размещение всей
инфраструктуры шлюза: такого нового обещания Grace не делает. Существующий
regional endpoint не заменяется global endpoint при миграции.

Resolver получает публичные catalog, ZDR list и endpoint metadata, проверяет
modalities, strict schema/max_tokens, context/output limits, provider/region,
request/image/token pricing и qualification. Proof действует не более 24 часов.
Один refresh: 2 GET плюс до 8 endpoint GET, без данных документа и retry.
Выбор: priority, затем model/provider ID; только до согласия и резерва.
`openrouter/auto`, wildcard, latest aliases запрещены. В запросе остаются
`provider.only`, ZDR, deny data collection, запрет fallback и ценовые ceilings.
Отсутствие любого доказательства означает недоступность профиля.

Цены задаются в integer USD micros: token rates — за миллион токенов,
request/image fees — за запрос/изображение. Резерв консервативно учитывает
30 000 input, 16 000 output и максимальное число изображений участника.
Для photo bounds оператор подтверждает imageTokenUpperBound конкретного
endpoint. Endpoint metadata должны явно показывать даже нулевые flat fees;
отсутствующая цена не считается бесплатной. Общий allowance не увеличен.

Если новой конфигурации нет, старые analysis/receipt policies преобразуются
в явные группы с тем же model/provider, максимум одним изображением для чека
и qualification не дольше 30 дней от исходного VerifiedAt. Canonical ID
консервативно берётся равным старому model ID: несовпадение с каталогом
отключит профиль до явного обновления оператором. Statement-группа из старой
span policy автоматически не создаётся. Новый consent hash исключает перенос
старого согласия на расширенный состав передаваемых данных.

## Окружение и старые пути

API читает только OpenRouter key для генеративных запросов. TypeSafe key
не загружается и не передаётся dev compose. Быстрый read-only router использует
тот же metered OpenRouter и согласие analysis; вероятности Jev не имитируются.
Исторические TypeSafe adapter fixtures остаются только в `_test.go`.
`jev-eval run` принимает лишь локальный HTTP fixture endpoint и специальный
`GRACE_JEV_OFFLINE_FIXTURE_KEY`; прямой `statement-page-reading run` закрыт
до чтения ключа. Старые отчёты и offline replay сохраняются.

В исходниках optional production AI overlay добавлена передача GRACE_AI_MODEL_GROUPS;
никакие реальные значения не заданы. Production flags, ключи и группы этим пакетом не настраиваются. Для реального
использования нужны отдельные проверка качества модели, настройка и rollout.
Локальная команда `make test-integration` использует одноразовые PostgreSQL и
networkless PDF helper; запросы к OpenRouter подменены fake transport.

## Следующая проверка качества

[Q1–Q3](specs/0056-openrouter-document-qualification.md) приняты; offline-
реализация восьми публичных документов и полного scorer подготовлена через
тот же adapter, [implementation evidence](specs/0056-qualification-evidence.md).
Приёмка/merge pending. `go run ./cmd/document-reading-eval -mode=check`
показывает corpus/лимиты без ключей и сети. Candidate пока null, Q4_READY=false.
Реальные вызовы Q4 требуют отдельного exact-source допуска после готового
manifest и выбора concrete model/provider/тарифа. Предложенные≤8POST/6GET/$2.00
не являются разрешением расходов; новых production group members пока нет.
