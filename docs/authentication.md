# Аутентификация и доступ Grace

SPEC-0013 вводит Telegram OIDC, серверные сессии и владение финансовыми данными.
Это новая версия приложения, требующая миграции 000006. Настоящий Telegram
вход и production rollout проверяются отдельно после настройки BotFather.

## Режимы запуска

- `locked` — режим по умолчанию. Финансовый API закрыт, login недоступен.
- `telegram` — HTTPS, Telegram OIDC и cookie-сессии. Без client credentials
  конфигурация отклоняется; автоматического обхода входа нет.
- `development` — только явный `APP_ENV=development` и loopback public URL.
  Локальные запросы получают test owner. Development Compose включает этот
  режим явно; production Compose его не включает.

В обычном development Compose `make dev` не требует Telegram. Для проверки
настоящего входа нужен HTTPS origin, который проксирует Web и `/api/*` к Go API.

## Настройка Telegram

1. Создать/выбрать бота в BotFather; открыть Login Widget.
2. Зарегистрировать origin сайта и точный callback
   `https://<domain>/api/auth/callback` в Allowed URLs.
3. Получить Client ID и Client Secret. Это OIDC credentials, не замена bot token.
4. На сервере задать `GRACE_AUTH_MODE=telegram`, `APP_ENV=production`,
   `GRACE_PUBLIC_URL=https://<domain>`, `CORS_ORIGINS=https://<domain>`,
   `TELEGRAM_CLIENT_ID` и `TELEGRAM_CLIENT_SECRET`.

Credentials передаются только серверу. Не добавлять их в Git, Vite-переменные,
CLI arguments или отчёт проверки. Запрашиваются только `openid profile`:
Grace не запрашивает телефон или право отправлять сообщения.
UserInfo не вызывается; ID token проверяется по Telegram JWKS.

Внешний контракт: [Telegram Login](https://core.telegram.org/bots/telegram-login).

## Первый вход владельца

Миграция создаёт владельца без Telegram identity и закрепляет за ним
существующие финансовые данные. Первый посетитель сайта не становится owner.

После миграции оператор с административным доступом к БД создаёт приглашение:

```sh
go run ./cmd/auth-admin owner-invitation
```

`DATABASE_URL` должен быть задан в защищённом окружении. В production image
также есть `grace-auth-admin`; через Compose можно выполнить:

```sh
docker compose --env-file release.env run --rm --no-deps \
  --entrypoint grace-auth-admin api owner-invitation
```

Команда выдаёт credential один раз в JSON: token и expiresAt. Не сохранять
его в общедоступных логах. Приглашение действует 24 часа и используется
однократно. Повторное создание owner приглашения до привязки отменяет старое;
после привязки команда отказывается создавать новое.

Владелец вводит код на `/login`, затем проходит Telegram login. Username
не определяет доступ: identity сохраняется по issuer/subject.

## Пользователи и сессии

В меню «Доступ» admin создаёт приглашение для нового member. Код передаётся
пользователю вручную; Grace не отправляет сообщений. У member свои пустые
финансовые данные. Admin тоже читает только собственные счета и операции.

`POST /api/auth/users/{userId}/disable` блокирует member и отзывает его сессии.
Нельзя блокировать себя или bootstrap owner. Этот endpoint не раскрывает
финансовые данные. HTTP admin-инструменты требуют session, CSRF и Origin.

Сессия действует 7 дней. «Выйти» отзывает текущую сессию, «Завершить все мои
сеансы» — все сессии пользователя. UI очищает финансовый cache при logout/401.
Cookie `__Host-grace_session` имеет Secure, HttpOnly, Path=/, SameSite=Lax и
не содержит identity или финансовые данные. В БД хранится hash token.

## Изоляция данных и миграция

Финансовые запросы выполняются под restricted PostgreSQL role
`grace_financial` с RLS и owner_id, установленным через SET LOCAL внутри
транзакции. Административный pool нужен auth repository и оператору;
финансовые модули не выполняют запросы от его имени напрямую.
Категории остаются общим read-only справочником. Составные FK запрещают
связать финансовые объекты разных владельцев.

Перед rollout выполнить backup/restore drill. После миграции проверить
остатки и первый вход owner. Down 000006 отказывается удалять ownership,
если финансовые данные есть у другого пользователя. В этом случае исправлять
вперёд или восстанавливать согласованный backup. Не обходить guard и не
откатываться на старый общий API после подключения других пользователей.

Production Caddy Basic Auth пока сохраняется как дополнительная внешняя
граница. Сам по себе Basic Auth больше не открывает финансовый API. До
отдельного rollout приложение остаётся в `locked`; включение Telegram и
изменение внешней Basic Auth границы не выполняются автоматически.

## Восстановление owner

Публичного recovery endpoint, passkey или второго провайдера пока нет.
При потере Telegram оператор после проверки личности владельца и backup
может в одной транзакции отозвать owner sessions, отменить его неиспользованные
invitations и удалить owner identity. Только затем создать owner invitation
и привязать новый Telegram. Финансовый owner UUID и финансовые записи не менять.
Эту процедуру не запускать автоматически или по запросу неаутентифицированного
посетителя. Независимый способ входа требует следующей спецификации.

## Проверки

`make test` проверяет RSA/JWKS OIDC fixture, replay, browser binding, CSRF,
Origin и Web auth boundary. `make test-integration` включает
`scripts/test-authentication.sh`: реальные PostgreSQL users/sessions,
финансовую изоляцию, assistant tools, owner-only down/up и запрет unsafe down.

В Windows запуск shell targets требует Git Bash (например,
`make SHELL=bash test`). POSIX 0600 для evaluator reports проверяется на Unix;
в Windows доступ к таким файлам определяется ACL каталога оператора.

## Постоянная production-конфигурация Telegram

Оператор хранит GRACE_AUTH_MODE=telegram, TELEGRAM_CLIENT_ID и
TELEGRAM_CLIENT_SECRET в /opt/grace/shared/secrets/telegram.env (mode 600,
владелец grace-deploy). deploy.sh загружает этот файл перед Compose, поэтому
последующие релизы сохраняют настройку. Credentials не входят в GitHub secrets
или release bundle. Файл является доверенным shell environment: редактировать
его может только оператор; не записывать туда данные посетителей.

Production API подключён к backend для PostgreSQL и edge для исходящих HTTPS
запросов Telegram. Host ports API/БД не публикуются. Внешний Basic Auth пока
остаётся включённым. После миграции 000006 operator создаёт owner invitation,
а владелец лично подтверждает Telegram login.