# SPEC-0058 P: production readiness и отсутствующая request price

- Статус: **accepted**
- Дата: 2026-10-26
- Acceptance record: владелец отменил предыдущий отказ и поручил выбрать другой вариант — «Я выбрал неправильно, выбери вариант другой». Принят вариант «Принять P и довести до production-прогона».
- Implementation status: integrated PR137–139 / deployed `db1d63d`; paid grant CONSUMED 1/1; [итог](0058-production-evidence.md)
- Родитель: [принятый SPEC-0058](0058-real-document-import.md)

## Найденная ошибка

Реализация принята и слита PR135, но проверка реального endpoint до inference выявила переносимость только исследовательской проверки. У Google Vertex EU для Gemini 3.1 Flash Lite поле `pricing.request` отсутствует. Продуктовый resolver требует явное поле и отвергает модель. Поэтому успешный research сам по себе не доказывал готовность product call. HTTP 200, unit tests и наличие ключа не заменяют эту проверку.

## Принятое ограниченное изменение

Только для purpose `statement` / profile `plain_document_reading` оператор может явно выбрать `requestFeeEvidence=plain_router_ceiling_only` при положительном request ceiling не более 1000 USD micros ($0.001). Отсутствующая request price остаётся неизвестной; она не подменяется нулём. Резерв включает весь ceiling, запрос передаёт его в `provider.max_price.request`. Если поле присутствует, оно должно быть корректной непустой неотрицательной decimal string в пределах ceiling; null, пустое и ошибочное значение отклоняются.

Остальные token/image цены должны быть опубликованы и укладываться в прежние потолки. Конкретная модель/canonical/provider, ZDR, EU, квалификация, срок, proof, limits, consent и отсутствие fallback/retry сохраняются. Strict statement, receipt и analysis этот режим не принимают. Исследовательский `router_ceiling_only` остаётся отдельным experiment-only контрактом.

Settlement использует только подтверждённый usage. Неизвестный счёт сохраняет резерв и блокирует новые POST; административный ноль не разрешён. Общий allowance не увеличивается.

## Разрешаемая проверка

Один новый платный product call через обычный production Web/API на ранее предоставленном владельцем PDF, ≤3 metadata GET для свежего proof, ≤$0.04 резерва и ≤$0.10 общего live бюджета SPEC-0058. Тот же model/provider. Без записи реальных финансовых операций. Это отдельный product run, не повтор закрытой серии SPEC-0056. Фото и независимое принятие эталона не считаются проверенными.

До вызова: merge/CI/Delivery/deploy exact SHA, backup/readiness, явная операторская конфигурация и браузерный consent с просмотром всех страниц. После: сохранённый draft/reload, сравнение с локальным эталоном, известный bill/settlement и sanitized evidence.

## Приёмка

- Table-driven unit checks: допускается только опциональное отсутствующее поле у явного plain режима с положительным bounded ceiling; присутствующие null/empty/malformed/negative/over-cap отклоняются; другие профили и experiment-only режим не расширяются.
- Wire check: request ceiling резервируется и отправляется, image/token ставки и route checks сохранены, один POST без повторов/schema/etalon.
- `make test`, `git diff --check`, post-merge CI и delivery.
- Production browser call и billing evidence отдельно от offline replay; документ, ответ и персональные строки не попадают в GitHub.

Поручение открывает реализацию P, CI/merge/deploy и ровно один указанный product call. Предыдущий выбор остановить прогон отменён владельцем до новых вызовов.
