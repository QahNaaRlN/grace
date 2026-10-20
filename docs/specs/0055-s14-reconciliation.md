# SPEC-0055 / S1.4: сверка единственной uncertain reservation

- Дата: 2026-10-20, Asia/Yekaterinburg.
- Статус: accepted; implementation_status: verified; exact settlement завершён 2026-10-20 17:10:51 UTC.
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

## Уточнение сопоставления identity — 2026-10-20

Read-only API verification выявила: generation metadata использует canonical slug
`google/gemini-3.1-flash-lite-20260507`, report policy — alias
`google/gemini-3.1-flash-lite`. Принимается только эта фиксированная пара при
свежем подтверждении официальным GET /models (`id` → `canonical_slug`), совпадающая
с опубликованным до smoke snapshot shortlist. Digest каталога сохраняется в private
audit. Любой другой canonical slug или отсутствие mapping — отказ. Product
policy, transport identity checks и scorer не меняются; это сопоставление полей
metadata для metering recovery, а не разрешение нового endpoint или paid replay.

## Исполнение и критерии приёмки — 2026-10-20

| Критерий | Исполняемое evidence |
| --- | --- |
| Exact metadata и расход до изменения ledger | [Inspect 39463492420](https://github.com/QahNaaRlN/grace/actions/runs/39463492420), source `865ccbb`: metadataMatched=true, unknownBillingVerified=true; официальный alias/canonical mapping подтверждён |
| Одна owner-scoped settlement, отсутствие uncertain | [Apply 39463769303](https://github.com/QahNaaRlN/grace/actions/runs/39463769303), source `865ccbb`: billingReconciled=true, sharedUncertainBlockCleared=true, newInferenceAuthorized=false; private audit содержит before/after |
| Fail closed для несовпадений и ceiling/replay | `python3 scripts/research/test-screening-reconciliation.py` PASS: ID/model/provider/time/BYOK/reservation/cost mismatch, exact canonical mapping, Decimal ceil, settled replay |
| Без inference и deployment | В обоих workflow runs screening/deploy jobs skipped; script использует только GET metadata/catalog и существующую grace_ai_settle для одной reservation |
| Регрессии и формат | `TMPDIR=/var/tmp/grace-quality-next-tests VITEST_MAX_WORKERS=1 make test` exit0, 334 web tests + Go/lint/FSD/format/build/Jev; `git diff --check` PASS |
| Очистка transient secret | После apply `gh secret delete GRACE_STATEMENT_SCREENING_GENERATION`; exact-name lookup вернул 0 |

Неизменяемый исходный screening report сохраняет billingKnown=false и FAIL.
Authoritative reconciliation хранится отдельно; это исправление metering,
не повтор smoke и не подтверждение качества кандидата. Первоначальная причина
provider-stage failure ещё не установлена.
