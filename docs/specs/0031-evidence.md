# SPEC-0031: implementation evidence

2026-10-09. Accepted [SPEC-0031](0031-password-registration.md),
codex/password-registration от1a142a8. Реализация локальная, production не менялся.

| AC | Результат | Исполняемое свидетельство |
| --- | --- | --- |
| AC1 | PASS local | bash scripts/test-password-authentication.sh на PostgreSQL12migration: valid bound invitation→pending verification; wrong email403; unverified login401; invitation остаётся неиспользованным до confirmation. Два concurrent confirm дают ровно один204 и один422. После подтверждения login без приглашения200; expired token422, disabled user401. Token hash/PHC password hash сохраняются, raw values не сериализуются в profiles/sessions |
| AC2 | PASS local | Тот же integration: bootstrap owner verified identity без credentials получает first-password link без invite; UUID/admin прежние. Replay422; password replacement отзывает прежнюю session и все stale tokens. Financial fingerprint accounts/events/postings до/после совпадает. PasswordSession проверяет тот же credential hash под user lock, чтобы не выдать сеанс по устаревшему паролю |
| AC3 | PASS в scope | Password unit tests: salt uniqueness, valid/wrong password, Unicode/policy, malicious Argon params, wrong Origin403 и legacy mode503. Persistent DB limiter остаётся активным между Repository instances. Integration проверяет protected invite: без CSRF403, valid session+Origin+CSRF201, anonymous401; login с invitation body400. Existing auth/provider/email/config tests PASS. Bounded work2/mail2 в handler; canonical emails сохраняют plus/dots. Reset token checks email+revision под lock |
| AC4 | PASS local | make test: Go, Web55 tests/13 files, lint/build, Jev49/72. Typed endpoints в authoritative OpenAPI, generated TS. PasswordEntry tests: login только два поля, invitation fragment cleared/pre-filled, explicit confirmation, mismatch validation, generic forgot result, cooldown. AuthActions tests: bound recipient/link/code, no automatic copy/send, recipient не меняется после редактирования нового draft email; member не получает invite controls. Browser smoke320/390: document305/375 (15px scrollbar), обе темы, Tab достигает submit с visible focus, login без invite, forgot только email. git diff --check PASS |

Integration завершился «SPEC-0031 password authentication integration checks passed».
Проверка запрета unsafe down специально ожидает отказ migration12 при наличии
credentials; это не ошибка итогового gate. Windows CGO_ENABLED=0: Go race detector
не запускался; concurrent transaction behavior проверен на настоящем PostgreSQL.
В средах CGO_ENABLED=1 скрипт добавляет -race.

Source tests: internal/auth/password_test.go, password_integration_test.go,
web/src/pages/login/PasswordEntry.test.tsx, shared/ui/layout/AuthActions.test.tsx.
Логи ignored web/node_modules/password-complete-check.log,
password-final-integration.log. Временный визуальный scaffold сохранён в ignored
web/node_modules/.grace-password-qa; не включён в приложение, сервер QA остановлен.
Все mailbox/password/token fixtures искусственные, реальных писем/credentials нет.

Mode password добавлен отдельно; production email mode не переключён. Новый
methods response скрывает OTP/Telegram только в password mode. Старые modes
сохраняются для rollback, protected финансовые endpoints/RLS не менялись.
Новый x/sys transitive dependency нужен существующему x/crypto Argon2id;
versions существующих библиотек не обновлялись.

Ограничения: реальная SMTP-доставка и production owner migration ещё не проверены;
их порядок в [runbook](../password-authentication-rollout.md). Mail worker in-process:
restart в момент dispatch может потерять отправку (повтор после cooldown), tokens
и quotas durable. Full accessibility certification/production rollout вне scope.
