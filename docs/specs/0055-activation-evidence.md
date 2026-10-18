# SPEC-0055 — Statement AI activation

Актуализация: 2026-10-19 (Asia/Yekaterinburg).

| Срез | Дата / evidence | Статус |
| --- | --- | --- |
| Functional/technical contract | Owner «да, утверждаю», 2026-10-19; PR #62, main `4a6f1e7` | accepted |
| Код PDF import | PR #64, merge `b51b74f`, main CI `39167930999` | integrated; gates PASS |
| Production manual path | Deploy `39168570754`, healthy 2026-10-19 02:37:31 +05:00 / 2026-10-18 21:37:31 UTC | deployed |
| Statement activation approval | Owner «Включи Statement AI, затем обнови дорожную карту и документы с датами и статусами», 2026-10-19 | authorized |
| Production readiness | Read-only workflow [39169569800](https://github.com/QahNaaRlN/grace/actions/runs/39169569800), 2026-10-19; no configuration/budget/usage values exported | PASS |
| Activation implementation | `codex/statement-ai-activation`: protected operator flag/policy, bounded metered synthetic smoke, rollback and disable | in_progress |
| Live synthetic activation smoke | Не запущен на момент записи | NOT_RUN |
| Statement AI production | Последний проверенный runtime `b51b74f`: false | disabled; activation pending |
| AC12 representative corpus / Jev comparison / cascade | Small activation smoke не заменяет полный benchmark | NOT_RUN |
| Private bank inference / owner functional acceptance | Не выполнялись агентом / результата владельца ещё нет | NOT_RUN / pending |

Политика — только `google/gemini-2.5-flash`, `google-vertex/eu`, Google,
text-only, reasoning explicitly disabled for bounded span mapping
(prompt revision `statement-span-mapping.v2.no-thinking`), no fallback/ZDR/deny, ceilings input 300000/output 2500000 micros/1M.
Перед активацией endpoint metadata повторно проверяет серверный verifier.
[Первичный каталог endpoint](https://openrouter.ai/api/v1/models/google/gemini-2.5-flash/endpoints)
проверен 2026-10-19: identity/parameters/prices соответствуют принятому контракту.

Activation smoke использует два известных synthetic PDF (columns/cards),
≤4 calls, whole-plan reserve ≤32000 micros ($0.032) в существующем ledger;
лимит/fees не увеличиваются. Никакого raw PDF/image upload, личных документов,
import draft или financial events он не создаёт. Go проверяет row count,
operation/booking clocks, account/FX amounts/currencies, direction, printed
turnovers, coverage и различие available/book balances. Результат без grounding
или с пропусками не включает настройку. Unknown billing freezes future calls.

Usage report остаётся в `/opt/grace/shared/statement-qualification-*.json` с mode
0600; CI содержит только итог, дату и число synthetic cases. Бюджет, остаток,
tokens, generation IDs и actual cost не публикуются в Git/CI. Личные данные
передаются только после consent конкретного preview; финансовая запись требует
другого, явного owner confirmation. Jev по-прежнему выключен.

Локальные проверки активации: `make test`, `make test-production`,
`go test ./internal/statementqualification`, `git diff --check`.
Shell regression `scripts/test-statement-activation.sh`: failed qualification,
failed rollout/restored persistence, paid-call-free replay и disable.
`test-commitments-rollout.sh` проверяет сохранение прежней statement policy/flag
вместе с receipt/analysis при rollback. CI/live/deploy результаты будут дописаны
по факту, без преждевременного объявления enabled/accepted.
