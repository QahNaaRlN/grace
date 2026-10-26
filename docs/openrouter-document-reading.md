# OpenRouter и чтение документов — SPEC-0056

Реализация G1–G4 принята и интегрирована PR122/main10a0059. В production включён ограниченный plain statement профиль SPEC-0058 P;
обобщённое качество на photo/scan документах остаётся **UNVERIFIED**.
Один реальный PDF успешно прочитан в research SPEC-0057; статус продукта ниже.
Контракт: [SPEC-0056](specs/0056-openrouter-document-reading.md).

## SPEC-0058: простая таблица выписки

Принята [SPEC-0058](specs/0058-real-document-import.md), реализация в ветке
`codex/real-document-import`; production `db1d63d` подтверждён по принятому P.
[Live evidence](specs/0058-production-evidence.md): один новый product call,96 строк, raw767/768, после одной UI-правки768/768; $0.013741 settled, financial writes0.
Разрешение на этот прогон CONSUMED1/1; qualification до02.11.2026 14:59 Asia/Yekaterinburg.
При заданных groups readiness проверяет действующую statement group, без inference. Новый plain planner0058.v2 стабилен после JSONB roundtrip; устаревший preview требует новой подготовки/согласия.
Для новых загрузок Web выбирает `plain_review`: PDF до пяти страниц или одно фото,
200 операций, один RUB debit-счёт. Старые источники сохраняют прежний режим;
кнопка перехода отменяет их прежние согласие, сверку и preview до нового чтения.

В новом режиме передаются только изображения всех страниц и короткая инструкция.
Native text, эталон, JSON Schema, цитаты и координаты в запрос не добавляются.
Исходный ответ сохраняется отдельно от редактируемой таблицы. Ошибка формата
не уничтожает ответ: доступны ручные правки. Даты сохраняют напечатанное время
без выдуманного часового пояса, суммы переводятся в minor units кодом Go.

Порядок: загрузить → посмотреть страницы и стоимость → разрешить передачу →
проверить/исправить таблицу → посмотреть все страницы и сохранить одну сверку
документа → выбрать решения строк в preview → подтвердить импорт. 96 отдельных
отметок по строкам не требуется. Правка таблицы отменяет сверку и старый preview;
повтор подтверждения не создаёт вторую проводку. Ручной путь доступен без модели.
Доступный остаток не является начальным балансом счёта, а банковские списания
не являются автоматически покупками. Неясные переводы и кредитные движения
остаются для решения владельцем в существующем импорте.

Новый operator profile — `plain_document_reading`, только purpose `statement`.
Он допускает до пяти изображений и требует image/text, `max_tokens`, конкретные
model/provider, fresh pricing/ZDR/qualification proof. Strict-schema capabilities
для него не требуются. Чеки и старый `document_reading` сохраняют прежний контракт.
Нужна отдельная операторская конфигурация группы: сам факт наличия кода не
включает AI. Лимиты, общий ledger, неизвестная стоимость, no retry/fallback
и privacy guards сохранены. После удаления оригинала raw answer и plain state
удаляются вместе с ним; подтверждённый финансовый результат сохраняется.

Networkless PDF helper теперь содержит закреплённый `font-dejavu=2.37-r6`:
PDF без встроенного шрифта должен давать читаемые страницы. Проверка пустого
набора перед сетью дополняет визуальную сверку, но не является проверкой OCR.

Подтверждение качества относится к конкретному формату и реальному примеру.
Сохранённый исследовательский ответ совпал с локальным draft эталона на одном
PDF; это не автоматическая qualification фотографий и сканов.
[Текущее состояние и evidence](statement-ai-status.md).

## Сохранённый контракт SPEC-0056

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
или `statement`. Профили: `text_reasoning`, `document_reading`, `plain_document_reading` (только statement).
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
отсутствующая цена не считается бесплатной. Для принятого [SPEC-0058 P](specs/0058-production-price-ceiling-candidate.md) только plain statement допускает явный `requestFeeEvidence=plain_router_ceiling_only`: отсутствующий ключ request price при положительном ceiling ≤1000 micros. Полный ceiling входит в резерв и `provider.max_price.request`; присутствующее некорректное значение отклоняется. Остальные профили остаются строгими, experiment-only режим не используется. Общий allowance не увеличен.

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
Q1–Q3 интегрированы PR123/main `1bf7993`, main CI40141643032 success.
`go run ./cmd/document-reading-eval -mode=check` показывает corpus/лимиты без
ключей и сети. [Кандидат Q4 и тарифные границы](research/statement-quality/document-q4-candidate-2026-10-25.md)
подготовлены локально; runtime доступ и свежий endpoint proof UNVERIFIED, Q4_READY=false.
Реальные вызовы Q4 требуют отдельного exact-source допуска после готового
manifest и выбора concrete model/provider/тарифа. Предложенные≤8POST/6GET/$2.00
не являются разрешением расходов; новых production group members пока нет.
