# SPEC-0018: Публичный экран Telegram-входа без старого Basic Auth

- Статус: accepted
- Автор: Codex
- Дата: 2026-10-06
- Acceptance owner: owner
- Acceptance record: владелец поручил «устранить баг» после диагноза и конкретного предложения убрать внешний Basic Auth, сохранив Telegram-вход, приглашения и защиту API, и обновить deployment-проверки.
- Implementation branch: codex/public-telegram-entrypoint
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Новый браузер получает форму логина/пароля Caddy до загрузки Grace.
Приглашение приложения не открывает эту внешнюю границу. Пользователь должен
сразу видеть экран Telegram-входа, а финансовые данные — только после
авторизации. Сообщённая минутная задержка отдельно не воспроизведена;
исправление устраняет подтверждённую внешнюю блокировку, не обещая объяснить
все возможные задержки провайдера/браузера.

## Scope

Убрать Basic Auth из production Caddy и зависимость Compose от его secret.
Заменить release smoke test проверками публичной страницы и session-protected
API. Добавить регрессионную проверку реального Caddy proxy без credentials.
Обновить текущую эксплуатационную документацию. Выпустить проверенный релиз
через существующий GitHub pipeline и повторить анонимные проверки на сайте.

## Non-goals

Изменение Telegram OIDC, приглашений, сессий, CSRF/Origin, RLS, денежных данных,
Google/LLM, импорт и обход авторизации. Старые секреты оператора не удалять:
они могут потребоваться для rollback. Не использовать/потреблять приглашение
другого человека для диагностики.

## Текущее поведение и доказательства

main `1f0115f`: Caddy требует basic_auth на всех routes, Compose читает
basic_auth_hash, Deploy production требует HTTP 401 от главной страницы.
2026-10-06 анонимные запросы к `https://graceapp.ru/`, `/login`, `/api/auth/me`
получили 401 и `WWW-Authenticate: Basic realm="restricted"`.
SPEC-0013–0015 уже обеспечивают Telegram-вход, session auth, приглашения,
tenant isolation и успешный production-вход владельца.

## Предметная модель и инварианты

Web shell и экран входа публичны. API проверяет собственные сессии:
анонимные `/api/auth/me` и `/api/v1/accounts` возвращают 401 JSON без Basic
challenge. Новые identities получают доступ только через приглашение.
Admin не получает доступ к чужим финансам. При locked mode API остаётся
закрытым. Только Caddy публикует host ports; API/БД остаются внутренними.

## API и OpenAPI

Контракт приложения не меняется; DTO и generated types не меняются.
Меняется внешний proxy: HTTP 200 для Web shell, `/login` и `/api/meta`,
HTTP 401 JSON для защищённых anonymous API routes. Ни один ответ не должен
содержать `WWW-Authenticate: Basic`; публичная страница содержит только shell,
не финансовые данные. HTTPS/redirect и security middleware сохраняются.

## Данные, миграция и rollback

Миграций нет. Существующие users, invitations, sessions и financial tables
не меняются. Compose перестаёт монтировать basic_auth_hash; файл VPS и
GitHub secret остаются нетронутыми. Предыдущий immutable release позволяет
откатить приложение/proxy; rollback не меняет ownership и не возвращает
старую версию API без session auth.

## UI

Пользователь открывает сайт и видит существующую форму «Войти через Telegram»
с полем приглашения. Браузерный запрос логина/пароля исчезает. Реальная
авторизация остаётся интерактивным действием пользователя.

## Ошибки и негативные сценарии

Без сессии финансовые API возвращают 401 JSON. Invalid/missing приглашение,
подпись/issuer/audience/nonce, expired flow/session, CSRF и чужие данные
по-прежнему отклоняются существующими проверками. Proxy не превращает API
401 в Web shell. Неисправный релиз не проходит deploy smoke test.

## Критерии приёмки

- [ ] AC1: Реальный Caddy пропускает Web/login/meta без Basic credentials;
  ответы не содержат Basic challenge.
- [ ] AC2: Анонимные me/accounts через Caddy возвращают 401 JSON приложения;
  существующие auth/isolation tests проходят.
- [ ] AC3: Production Compose валиден без basic_auth_hash; host ports только
  у Caddy; credentials Telegram и locked mode сохраняют прежнее поведение.
- [ ] AC4: Deploy smoke проверяет AC1/2 без Basic password; CI/Delivery проходят
  до публикации; на production повторены анонимные проверки.
- [ ] AC5: make test, make test-production, git diff --check проходят;
  evidence связывает AC с исполняемыми результатами и release SHA.

## Исполняемые проверки

`make test`, `go vet ./...`, `make test-production`, `make test-integration`
(GitHub CI), `git diff --check`. Production test поднимает изолированный
Caddy с mock Web/API и проверяет status/body/headers, forwarding API 401
и отсутствие Basic secret. Deployment использует тот же anonymous smoke
на HTTPS сайте; реальные Telegram credentials, cookies и коды не выводятся.

## Открытые решения

Нет: scope исправления принят прямым поручением после предложения.
Отдельный минутный инцидент требует browser/server trace, если сохраняется
после удаления подтверждённой блокировки; не менять retry/таймауты наугад.
