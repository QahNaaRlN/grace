# SPEC-0013: Telegram OIDC, сессии и владение финансовыми данными

- Статус: integrated
- Автор: Codex
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: owner поручил «Определить следующую спеку и приступить к реализации» после рекомендации Telegram OIDC напрямую, серверных сессий и доступа по приглашению. Это разрешение на реализацию описанного варианта; production rollout требует отдельной настройки Telegram и проверки владельцем.
- Implementation branch: codex/telegram-auth
- Implementation commit: 6395e5b268a5c3753c38615d8e5d69a579d953ef

## Проблема и пользовательский результат

Grace защищён внешним Basic Auth, но внутри API не различает пользователей.
Вход через Telegram должен открывать только собственные финансовые данные.
Владелец может приглашать пользователей и отзывать доступ, не получая через
административные API доступ к их финансовым записям.

## Scope

- Telegram OIDC Authorization Code Flow с PKCE S256, state и nonce;
- проверка ID token готовой Go OIDC-библиотекой: подпись, issuer, audience,
  срок действия и nonce; scopes только openid/profile, без UserInfo;
- собственные пользователи и identity по паре issuer/subject;
- одноразовые приглашения, включая приглашение существующего владельца;
- непрозрачные серверные сессии в PostgreSQL, cookie Secure/HttpOnly/SameSite=Lax,
  срок 7 дней, logout и отзыв всех сессий;
- admin/member: admin управляет приглашениями и блокировками, финансовые
  данные доступны только владельцу независимо от административной роли;
- PostgreSQL RLS для всех финансовых таблиц и транзакционный контекст владельца;
- ограничения связей между финансовыми объектами разных владельцев;
- идемпотентность финансовых команд в пределах владельца;
- UI входа, приглашения, выхода и обработки истёкшей сессии;
- OpenAPI, generated TypeScript, документация настройки и локальные проверки.

## Non-goals

- production deployment и реальный Telegram smoke без настроенного BotFather;
- открытая регистрация, автоматическое связывание identity, email/phone matching;
- общие семейные счета, совместное владение или доступ admin к чужим финансам;
- passkeys, второй провайдер и self-service recovery: отдельная следующая спека;
- Mini Apps, отправка сообщений ботом, OAuth access tokens в браузере;
- изменяющие инструменты ИИ; текущий ассистент остаётся read-only;
- изменения финансовой арифметики и расширение предметного scope.

## Текущее поведение и доказательства

- `internal/platform/httpserver/server.go` не устанавливает principal;
- `accounts.Repository.List` читает все accounts, analytics агрегирует всю БД;
- migrations 1–5 не содержат users, identities, sessions и ownership;
- production Caddy закрывает приложение Basic Auth; это остаётся внешней
  защитой до отдельного rollout и не заменяет проверку principal в Go;
- Windows `make test` выявляет существующий POSIX-only тест mode 0600.

## Предметная модель и инварианты

`User`: UUID, display name, role admin/member, disabled flag. Все существующие
финансовые данные принадлежат фиксированному bootstrap owner UUID
`00000000-0000-4000-8000-000000000001`. Этот UUID не является credential.
Пользователь owner создаётся миграцией без identity и не может войти без
приглашения. Новые приглашения создают пустого member без финансовых данных.

`Identity`: issuer/subject уникальны, связаны с User. Username — только display.
Первое связывание возможно по одноразовому приглашению. Существующая identity
входит без приглашения. Занятая identity не может присвоить другое приглашение.

`Invitation`: SHA-256 от случайного 256-bit token, target user, срок 24 часа,
одноразовое потребление в транзакции связывания identity. Raw token выдаётся
один раз администратору/CLI; отправка приглашения выполняется владельцем.

`LoginFlow`: одноразовый 256-bit state, привязанный к отдельной cookie браузера,
PKCE verifier, nonce, optional invitation hash, срок 10 минут. Callback требует
совпадения state и browser cookie, валидного токена и неиспользованного flow.
Неверные и повторные callbacks не создают сессию. Redirect фиксирован конфигом.

`Session`: в cookie случайный 256-bit token; в БД только SHA-256 и CSRF hash,
user, created/expires/revoked. `/auth/me` возвращает CSRF token, вычисляемый из
session token, для заголовка `X-CSRF-Token`. Mutations требуют совпадения CSRF
и точного Origin; logout отзывает сессию и удаляет cookie. Disabled user
теряет доступ немедленно. Нет sliding expiration и refresh tokens.

Категории остаются общим read-only справочником. Accounts, financial_events,
postings, receipts/items, obligations/payments/settlements имеют owner_id.
RLS работает под ограниченной ролью БД внутри транзакции; principal приходит
только из middleware/явного test context, никогда из JSON, URL или заголовка.
Нет principal — нет финансового доступа. Tenant state не остаётся на pooled
connection. Аналитика и tools используют тот же контекст.

## API и OpenAPI

- `POST /api/auth/login`: optional invitation → authorization URL; Origin check;
- `GET /api/auth/callback`: state/code, фиксированный redirect на `/` или `/login`;
- `GET /api/auth/me`: user, CSRF token и режим; без сессии 401;
- `POST /api/auth/logout`, `POST /api/auth/revoke-sessions`: session + CSRF;
- `POST /api/auth/invitations`: admin, display name → invitation token/expires;
- `POST /api/auth/users/{userId}/disable`: admin → блокировка и отзыв сессий,
  нельзя блокировать самого себя/bootstrap owner;
- финансовые `/api/v1/*` требуют authenticated principal; mutations также CSRF;
- `/health`, `/ready`, `/api/meta` публичны и не раскрывают финансовые данные.

Ошибки: 401 unauthenticated, 403 csrf/access_denied, 503 auth_unavailable,
400 invalid_request. Для callback — фиксированная безопасная ошибка, без
provider response, token, username или secret в логах/URL. Чужой объект ведёт
себя как недоступный/несуществующий по существующему предметному контракту.

## Данные, миграция и rollback

Миграция 000006 добавляет users/auth tables, backfill owner_id без изменения
денег и дат, owner-scoped unique keys, составные FK, RLS и ограниченную роль.
Проверить исходные остатки/количества до и после. Migration role должна иметь
право создать restricted role; runtime финансовых запросов переключается в неё.
Identity/session operations используют отдельный auth repository.

Down migration запрещена при наличии финансовых данных второго владельца:
откат не должен объединять чужие данные. В этом случае восстановить backup
или исправить вперёд. При единственном owner down восстанавливает схему 000005.
Перед rollout необходим backup и restore drill. Старый Basic Auth API нельзя
возвращать после подключения второго пользователя без отдельной оценки.

## Конфигурация и UI

Режимы `locked` (default, финансовый API закрыт), `telegram` (требует client
ID/secret, HTTPS public URL) и `development` (явный APP_ENV=development,
loopback public URL, фиксированный test owner). Production не допускает
development. Credentials только server environment; cookie __Host-grace_session
в HTTPS режиме. Dev bypass не присутствует в production-конфигурации.

`cmd/auth-admin` создаёт первое приглашение owner через административный доступ
к БД. Recovery после потери Telegram возможен только документированной
операторской процедурой отзыва identity/session и нового owner приглашения,
без публичного recovery endpoint. Не выполнять её автоматически.

Страница `/login`: кнопка Telegram, optional invite input, объяснение ошибки.
Основные страницы закрыты auth boundary; logout очищает React Query cache.
401 финансового API переводит на login и очищает данные предыдущего пользователя.

## Ошибки и негативные сценарии

Тестировать неверную подпись/issuer/audience/expiry/nonce, недостающий code,
несовпадение browser cookie/state, повтор callback, истёкшие flow/invite/session,
uninvited identity, отключённого пользователя, CSRF/Origin, member admin API,
подстановку чужого account/event/receipt/payment ID и повтор idempotency key
между владельцами. Ошибка provider не открывает обход аутентификации.

## Критерии приёмки

- [x] Валидный OIDC flow создаёт сессию; негативные callbacks отвергаются.
- [x] Без приглашения новая identity не получает доступ; invite одноразовый.
- [x] Logout, expiry, revoke и disable закрывают доступ; mutations защищены CSRF.
- [x] Два владельца изолированы во всех модулях, аналитике и assistant tools.
- [x] Чужие связи запрещены БД; идемпотентность разделена по владельцу.
- [x] Миграция сохраняет остатки и факты; unsafe down запрещён.
- [x] Web показывает login/logout, очищает cache и переживает 401.
- [x] OpenAPI/types и конфигурация соответствуют поведению; секреты не в Git.
- [x] Общие gates и PostgreSQL auth integration suite имеют исполняемое evidence.

## Исполняемые проверки

`make test`, `make test-integration`, новый auth integration suite,
`go vet ./cmd/... ./internal/...`, `git diff --check`, OpenAPI generation drift.
OIDC smoke использует локальный RSA/JWKS test provider без Telegram credentials.
Для Windows POSIX file-mode check сохраняется на Unix; на Windows проверяются
создание и запрет перезаписи без ошибочного утверждения Unix permissions.

## Открытые решения

Выбран прямой Telegram OIDC. Credentials/Allowed URLs задаёт owner перед live
smoke. Passkeys и независимый способ входа остаются отдельной спецификацией;
это ограничение первого этапа явно указывается при передаче.

Источник внешнего контракта: https://core.telegram.org/bots/telegram-login
