# SPEC-0013 evidence: Telegram и владение данными

- Specification: [SPEC-0013](0013-telegram-authentication-and-ownership.md).
- Accepted specification commit: `057f349`.
- Implementation commit: `6395e5b268a5c3753c38615d8e5d69a579d953ef`.
- Branch: `codex/telegram-auth`.
- Проверено: 2026-10-05, локально, без production deployment.

## Критерии и исполняемые доказательства

| Критерий | Проверка | Результат |
| --- | --- | --- |
| OIDC, PKCE, nonce, issuer/audience/expiry/signature | `go test ./internal/auth`; RSA/JWKS fixture, проверка token endpoint и Basic credentials, негативные токены | PASS |
| Browser/state binding, replay, fail-closed | `internal/auth/handler_test.go`, `internal/authtest/integration_test.go` | PASS |
| Одноразовые приглашения, запрет открытой регистрации, истечение flow/invite | PostgreSQL auth suite, `scripts/test-authentication.sh` | PASS |
| Logout одного сеанса, revoke всех, disable, expiry | Реальный auth repository и защищённые HTTP routes в PostgreSQL suite | PASS |
| CSRF, точный Origin и admin/member | Handler tests и PostgreSQL HTTP checks | PASS |
| Владение во всех финансовых модулях и шести tools | Два пользователя, списки/аналитика/tools, попеременное использование pool | PASS |
| Чужие account/event/receipt/payment IDs | Cross-owner transfer/refund/receipt/obligation/settlement отказывают, composite posting FK запрещает запись | PASS |
| Идемпотентность разделена по владельцу | Одинаковый key создаёт разные события у двух пользователей | PASS |
| Миграция и rollback | Сохранён исходный баланс 12500000, owner-only down/up, unsafe multi-owner down ожидаемо запрещён; прежние финансовые suites проходят | PASS |
| Web boundary и очистка данных | 20 Web tests, в том числе logout/401/смена identity в другой вкладке | PASS |
| OpenAPI/types/config | Generation, TypeScript build, config tests, locked default и отказ production development bypass | PASS |

## Общие проверки

- `make SHELL=bash test`: PASS (Go, Web tests, lint, build, offline Jev datasets).
- `make SHELL=bash test-integration`: PASS, все восемь suites.
- `go vet ./cmd/... ./internal/...`: PASS.
- `git diff --check`: PASS.
- `make test-production` в Linux-контейнере: PASS.
- API Docker image (в integration) и production Web Docker image: PASS.
- Повторная генерация OpenAPI types: PASS, без изменения generated-файла.

На Windows integration запускался через Git Bash с `MSYS2_ARG_CONV_EXCL=-path=`.
Тестовый адаптер curl передавал JSON через UTF-8 stdin: native Windows curl
при `-d` повреждал кириллицу из аргументов. Адаптер расположен вне репозитория
и не входит в приложение. Linux CI использует обычный curl. Production suite
выполнялся в Linux-контейнере с настоящими symlinks; Git Bash копирует их.
`.gitattributes` сохраняет shell scripts с LF. POSIX file-mode assertion
сохранён для Unix; Windows evaluator проверяет остальные файловые инварианты.

## Ограничения

Настоящий Telegram smoke, BotFather credentials/Allowed URLs, backup/restore
перед rollout и production deployment не выполнялись. Сначала настроить
[инструкцию входа](../authentication.md), создать одноразовое owner invitation
и проверить HTTPS callback. Default production mode — `locked`.
Basic Auth остаётся внешней защитой до отдельного rollout.
Независимый вход/passkeys и публичное восстановление в этот scope не входят.
Большой Web bundle вызывает прежнее предупреждение Vite (>500 kB), сборка проходит.

## Post-merge

Интеграция fast-forward в локальный `main`: `5aa84fe` (implementation достижим из main). После интеграции в `C:/PersonalProjects/grace` повторены `make SHELL=bash test` (20 Web tests), PostgreSQL auth suite, `go vet`, `git diff --check` и OpenAPI generation drift — PASS. Production suite проверен из основной копии в Linux-контейнере. GitHub push, CI и production deployment не выполнялись.
