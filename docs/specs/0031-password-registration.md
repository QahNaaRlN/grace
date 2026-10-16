# SPEC-0031: почта, пароль и приглашения

- Статус: integrated (объединённый выпуск `d475cca`: CI run 38038836282 и Deploy production run 38039374895 успешны)
- Дата: 2026-10-09
- Acceptance owner: владелец Grace
- Acceptance record: «Давай добавим ... регистрацию ... почте, паролю, коду
  приглашения ... восстановление пароля ... через почту»; ответы владельца:
  «Перейти только на почту и пароль», «Ссылка и код для конкретной почты».
- Branch: codex/password-registration, base1a142a8.

## Результат / scope

В режиме password вход требует только email/password. Signup требует
email/password/invitation, затем подтверждение адреса через одноразовую ссылку.
Admin создаёт приглашение на имя и конкретную почту; копирует ссылку или код,
сам отправляет человеку. Ссылка содержит fragment, не query token. TTL24часа.
Вход и восстановление приглашения не требуют. Регистрироваться без приглашения
нельзя. Ссылки не расходуются GET-запросом, действие подтверждается формой POST.

Forgot password: generic response, письмо с одноразовой ссылкой TTL30мин,
форма нового пароля, отзыв всех сеансов и старых reset tokens, затем обычный вход.
Existing verified email users устанавливают первый пароль тем же способом,
на существующий UUID/роль. Signup не объединяет существующие identities.
Existing sessions сохраняются до истечения/сброса; OTP-login недоступен в password
mode. Старые режимы остаются для controlled rollback, не предлагаются в new mode.
Подтверждение/смена email в профиле по коду остаётся (это не login).

## Инварианты / данные

Argon2id19MiB/t2/p1, случайная salt16bytes, hash32bytes, strict PHC format.
Пароль15–128 Unicode code points, <=512bytes; paste/autofill разрешены, без
trim/навязанных сочетаний. В БД только hash; password/token никогда не в logs/URL.
Invite bound canonical email + reserved user UUID; расходуется атомарно только
после email verification и создания credentials. Неуспешные/параллельные requests
не дают вторую identity. Signup pending hash истекает через30мин.
Reset token привязан к UUID/email/credential revision; изменение email/password
делает старую ссылку недействительной. Tokens в БД hashed, single use, locked tx.
Passwords hash/verification bounded concurrency2 + DB source/email attempt limits;
dummy verification для неизвестного адреса, generic errors. Reset/verification
mail async bounded queue; SMTP errors не раскрывают account existence, пользователь
может повторить после cooldown. Email quotas сохраняются в PostgreSQL.
Origin проверяется на публичных POST; приватные mutations сохраняют CSRF,
HttpOnly Secure SameSite sessions, auth/RLS. Финансовые таблицы/owner_id неизменны.

## API / UI

Новые password endpoints login/register/confirm/forgot/reset/invitations,
auth methods password availability. OpenAPI authoritative, regenerate TS.
Login: email/password, «Забыли пароль?», «Регистрация». Register: три поля,
подтверждение пароля, приглашение prefilled из link fragment. Recovery: email,
затем форма пароля по fragment token. Токены удаляются из адресной строки после
загрузки, no-referrer/no-store; нет сторонних ресурсов. Admin invite показывает
email, expiry, link/code и Copy; ни одна кнопка не отправляет приглашение сама.

## Текущее поведение / rollback

По SPEC-0022 есть email OTP, приглашение при первом login, проверенные identities
и SMTP. Паролей нет. Migration0031 создаёт credentials + pending tokens, без
финансовой migration. Только additive SQL; rollback возвращает email mode/code,
hashed данные сохраняются; destructive down отказывается при существующих credentials.
Production rollout отдельно: backup, переход подтверждённого владельца,
before/after UUID/roles/финансовые fingerprints; production не меняем в этом scope.

## Ошибки / приёмка

- AC1: регистрация только bound invite + email verification; expiry/replay/race,
  existing email и disable fail closed; вход без invite; пароли/токены не раскрыты.
- AC2: reset/first-password без invite, token revision/email binding; одноразовость,
  все sessions revoked, owner UUID/финансы сохранены; generic unknown responses.
- AC3: лимиты, Argon2/dummy/bounded workload, Origin/CSRF/RLS, email mode rollback.
- AC4: typed contract, обе темы/mobile/autofill/feedback и copy приглашения;
  make test, git diff --check, PostgreSQL integration, отчёт по всем AC.

Sources: OWASP Password Storage / Forgot Password Cheat Sheets.
