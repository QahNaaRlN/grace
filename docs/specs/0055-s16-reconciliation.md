# SPEC-0055 / S1.6: восстановление учёта единственного запроса

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Статус: draft; implementation_status: not_started.
- Основание: [S1.6 evidence](0055-s16-evidence.md), run `39496554051`,
  source `9e89ae08b7ec0b2b60d642b7f2c3d29a7f52598f`.
- Цель: подтвердить фактический расход через GET metadata и учесть его ровно
  в одной uncertain reservation. Никакого нового inference или повтора smoke.

## Зафиксированная проблема

Единственный прогон вернул qualification_provider_failed_selected_model до
первой оценки. HTTP200 и correlation ID сохранены приватно; report.calls/results
пусты и actualMicros неизвестен. Reservation уникальна и uncertain, общий ledger
блокирует новые платные вызовы. Это не доказательство нулевого расхода.

## Предлагаемый контракт

1. Только `/opt/grace/shared/statement-screening-s16-gemini31-diag-20261021.json`
   и его `.started` marker. Проверить source SHA, приватность 0600/отсутствие
   symlink, report v2, policy/profile/cap161700, FAIL selected_model/HTTP200,
   calls=[]/results=[]/actualMicros=null. Исходные файлы не изменять.
2. ID брать только из private failure.unverifiedCorrelationId; regex
   ^gen-[0-9A-Za-z-]+$, length<=128. Пользователю не требуется искать ID в Activity.
   Он остаётся неподтверждённой ссылкой до следующей проверки и в logs не выводится.
3. На сервере существующим ключом выполнить только GET `/api/v1/generation?id=...`
   и публичный GET `/api/v1/models`, без redirects и с ограничениями размера/времени.
   Не читать PDF, personal drafts, request/response content; management key и
   изменение retention не нужны. Проверить exact ID, provider_name=Google,
   non-BYOK, время metadata в пределах report.time−2s…+120s.
4. Billing model должен совпасть с alias `google/gemini-3.1-flash-lite` либо с
   единственным заранее известным canonical slug
   `google/gemini-3.1-flash-lite-20260507`, только если свежий официальный каталог
   подтверждает именно эту точную пару id→canonical_slug. Иная пара — отказ.
   Это сравнение полей generation metadata для billing; оно не меняет transport
   и не доказывает, какую строку содержало утраченное selected endpoint model.
5. По report.time найти ровно одну reservation: purpose=statement, exact
   model/provider, reserved161700, marker statement-qualification:<row UUID>,
   created_at в ±5s от report.time. Статус uncertain либо уже settled с тем же
   authoritative расходом. Owner и UUID проверить, не выбирать другую строку.
6. total_cost должен быть известен, конечен, >=0; Decimal ceil до integer USD
   micros, не более **53900** для единственного первого запроса. Никакого нуля
   по догадке. Missing/mismatch/overrun не разрешают settlement.
7. Сначала read-only inspect с private audit0600, затем отдельный apply при тех
   же проверках: owner-scoped существующая grace_ai_settle с прежним marker,
   общей блокировкой ledger и проверкой row после COMMIT. Конфликтующий settled
   replay не перезаписывается; тот же replay идемпотентен. Budget/fees/allowance,
   финансовые записи, модель, config/image/flags не меняются.
8. До/после, digest report/catalog и authoritative metadata сохраняются приватно
   в отдельном reconciliation audit S1.6. Публично — только fixed codes/booleans.
   Старый S1.4 audit и его report/marker не трогать. Временный script удаляется;
   нового секрета с ID не создавать. Успех снимает только metering uncertainty;
   FAIL/consumed marker/неподтверждённое качество остаются. Общий uncertain flag
   перечитать; другие reservations не изменять и не обещать снятие их блокировок.

## Критерии

| ID | Исполняемое подтверждение |
| --- | --- |
| R1 | Offline match/mismatch/unknown/cost ceiling/ceil/replay и точная alias/canonical пара; malformed ID/status/report и иная reservation отвергаются |
| R2 | Read-only run сохраняет private audit и подтверждает metadata/time/model/provider/cost без settlement/inference; неподтверждённое не становится нулём |
| R3 | Exact apply меняет только одну ai_usage reservation существующей функцией; before/after, отсутствие uncertain и конфликтный replay проверены |
| R4 | make test, затронутые ops/приватность проверки и git diff --check; внешний inference=0, transport/adoption/deploy/smoke не разрешены |

После восстановления учёта отдельно разобрать совместимость selected_model.
Настоящий контракт не разрешает изменение alias acceptance, новый prompt,
переключение production policy, повтор S1.6 или следующий платный прогон.
