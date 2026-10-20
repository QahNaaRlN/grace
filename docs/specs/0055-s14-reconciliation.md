# SPEC-0055 / S1.4: сверка единственной uncertain reservation

- Дата: 2026-10-20, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: not_started.
- Основание: accepted [S1.4](0055-s14-candidate-screening.md), просьба о generation
  ID для authoritative reconciliation; владелец предоставил ID в прямом ответе.
- Scope: GET generation metadata → exact match → owner-scoped grace_ai_settle
  для одной reservation неуспешного run 39459614043. Никакого нового inference,
  повторного smoke, budget/fees/config/image/financial-record changes.

Использовать прежний проверенный ops-подход 41869c3, ограничив target фиксированным
private report `statement-screening-s14-gemini31-20261020.json`, marker исходного
SHA e36dbcbb5c57ed8b2060cc2bcbc1ebc8482fd728, версией screening.v1 и моделью
Gemini 3.1 Flash Lite / Google EU. Report обязан иметь 0 recorded calls / 0 scores,
unknown cost и ожидаемый provider-stage error. В ai_usage должна найтись ровно
одна соответствующая reservation за окно 2026-10-20 16:41:00–16:42:30 UTC,
reserved161700 и marker statement-qualification:<reservation ID>.

ID передаётся transient GitHub secret, в Git/CI не публикуется. Ключ остаётся
на VPS. Exact authoritative generation ID/model/provider/non-BYOK/time должны
совпасть с report/usage row; cost Decimal ceil → integer micros, >=0 и <=53900
для единственного первого call. Missing/mismatch/overrun не разрешает settlement.
Сначала read-only inspect и приватный audit0600, затем apply по тем же проверкам;
использовать существующую owner-scoped функцию с блокировкой ledger. Повтор apply
идемпотентен, конфликт не перезаписывается. До/после сохраняются приватно, public
outputs — только fixed codes/booleans. При успехе не возобновлять paid calls:
авторизация S1.4 уже израсходована, quality UNVERIFIED и FAIL сохранены.

Проверки: fake metadata/ledger match и mismatch/unknown/ceil/replay; отсутствие
inference/financial/budget writes; workflow modes mutually exclusive; diff check,
обязательный make test. Read-only verification и exact settlement runs фиксируются
отдельно. После завершения transient secret удаляется, отсутствие проверяется.
