# SPEC-0057: один реальный документ и простой ответ модели

Статус: **accepted**. Основание: владелец задал порядок «реальный файл → совместная
расшифровка → короткое аналогичное задание OpenRouter → сравнение» и голосом подтвердил
«Да» на вопрос о запуске этой проверки сейчас. Это новый одиночный опыт, не продолжение
закрытой серии SPEC-0056 S/V. Данное решение фиксируется до реализации.

## Принятый scope

Один предоставленный владельцем PDF целиком, пять визуально проверенных страниц;
эталон остаётся локальным, не передаётся модели. Только изображения страниц и короткая
инструкция перечислить все операции, обе даты/время, обе суммы/валюты, описание и карту,
сохранить исходный порядок, явно отметить неразборчивое. Ответ — обычная Markdown-таблица,
без JSON Schema, координат, цитат, native binding, классификации доходов/расходов или импорта.
Перед отправкой проверяются hashes/порядок/размер/непустое изображение всех пяти страниц.
PDF/страницы/ответ/эталон не коммитятся и не публикуются в GitHub; private0600 только локально
и в краткоживущем VPS staging. Вывод сравнения пользователю относится к его документу.

## Одно ограниченное выполнение

Тот же Google Gemini3.1FlashLite / canonical20260507 / google-vertex/eu / Google.
Один POST, до3 metadata GET для свежих capability/ZDR/endpoint/цен, без retries/fallback/
plugins/нового model/provider. Input≤30000/output≤16000, HTTP180s; прежние ceiling rates
275000/1650000micros/M, request1000/image1, image4200tokens. Пять страниц разрешены
только новому изолированному research-конструктору; обычные product/SPEC-0056 paths
сохраняют4pages и strict schema. Максимальный резерв35655micros (<$0.04), в существующем
shared allowance через grace_ai_reserve/allowed/settle; нового allowance/off-ledger calls нет.
Неизвестный bill сохраняет reserve и останавливает опыт без административного policy0.

Новый research CLI с default check без credentials. Run требует explicit execute flag,
accepted receipt/source/bundle hashes и no-clobber marker; повтор запуска запрещён.
Существующий shared OpenRouter adapter получает private plain-response mode только от
изолированного research-конструктора. Основной API, financial core, existing grants,
старые corpus/scorer/criteria и production activation не меняются.

Pinned SSH/SCP только к прежнему VPS: private upload новых binary/bundle/wrapper,
один run в текущем API container без restart/deploy/config changes, private download ответа.
Operator principal берётся из единственного ранее проверенного private V audit
(проверка его фиксированного hash), без user lookup/financial table reads/env/key dumps.
Ключ и DB URL читает только CLI внутри текущего container как прежний runner.

## Проверка и результат

До live: causal offline tests actual plain wire/five-page completeness, old strict/four-page
negative controls, source/bundle/marker/call bounds/privacy/metering, make test и diff-check.
Это одиночный operator research run; прежняя S/V цепочка новых PR/mainCI перед каждой
синтетической попыткой к этому новому опыту не применяется. Реализация и обезличенный
отчёт публикуются reviewable; личные исходные данные и модельный ответ остаются private.

Сравнение всех96 согласованных строк локально: количество/порядок, обе даты/время,
обе суммы/валюты, описание, карта; пропуски/добавления/ошибки показываются отдельно.
Эталон имеет статус draft_for_owner_review: ответ владельца разрешает опыт, не превращает
все распознанные нами поля в независимо подтверждённую истину. Сверка с банковскими итогами
и визуальный просмотр всех страниц остаются evidence подготовки. Никаких автозаписей финансов.
