# Переход Grace на почту и пароль

Подготовлено по accepted [SPEC-0031](specs/0031-password-registration.md).
Реализация локальная; production переключение ещё не выполнено.

## Поведение

- Вход: почта + пароль. Приглашение не запрашивается.
- Регистрация: почта + пароль + код, затем подтверждение почты через письмо.
- Приглашения: admin → «Доступ» → имя и почта → создать → скопировать ссылку
  или код. Самостоятельно передать человеку. Ссылка открывает регистрацию и
  заполняет код; вводить нужно именно приглашённую почту. Срок24часа, одноразовое.
- Восстановление: почта → письмо → новый пароль. Invitation не нужен.
- Существующие пользователи устанавливают первый пароль через восстановление
  на свою уже подтверждённую почту. Новый UUID/профиль не создаётся.

Письма регистрации/сброса действуют30мин. Открытие ссылки ничего не меняет:
нужно подтвердить действие в форме. Пароль15–128символов, можно использовать
фразу, вставку и password manager. После reset все прежние сеансы завершены.
Profile email verification по коду остаётся для подтверждения нового адреса,
а не для passwordless login.

## Отдельный production rollout

1. Проверить CI, построить release с migration12; подтвердить доступ к
   рабочему SMTP без печати credentials. Из текущих приглашений нужны новые,
   привязанные к email; старые unbound коды в password mode не работают.
2. Сделать и проверить backup. Записать owner UUID/role/email identity и
   финансовые fingerprints всех существующих financial tables.
3. Применить additive migration12. В protected конфигурации установить
   GRACE_AUTH_MODE=password, сохранив существующие SMTP/Public URL settings.
   HTTPS и SMTP TLS обязательны; development bypass не используется.
4. Без сеанса /api/auth/me должен401, /api/auth/methods — password=true,
   email=false, telegram=false. Старые email login request/verify дают503.
5. Владелец сам запрашивает первое письмо и вводит новый пароль. Пароль/ссылка
   не передаются в чат. Проверить прежний UUID/admin и финансовые fingerprints.
6. Smoke: bound invite, verification, вход без invite, recovery, отзыв sessions,
   недоступность чужих financial data. Подтвердить отправку реальных писем
   только на согласованные тестовые адреса; локальный integration использует mock.

Если verified email отсутствует, требуется существующий безопасный операторский
процесс привязки к нужному UUID, а не новая регистрация или merge по имени.
Root/административный production доступ требует отдельной авторизации владельца.

## Rollback

Возвратить предыдущий release и GRACE_AUTH_MODE=email. Password hashes/tokens
сохраняются; не выполнять destructive down migration12. Down отказывается
удалять таблицу с credentials. Сброс уже отозванных сеансов не отменяется:
пользователи войдут заново по доступному старому способу.

Mail dispatch в текущем этапе — bounded in-process worker. При перезапуске
во время отправки письмо может не прийти; спустя cooldown пользователь повторяет
запрос. Сохраняемый outbox не входит в этот этап; обещания гарантированной доставки
не делаются. Лимиты и tokens сохраняются в PostgreSQL, restart их не сбрасывает.

## Проверки

make test; git diff --check; bash scripts/test-password-authentication.sh.
Windows: Git Bash, MSYS_NO_PATHCONV=1; race detector включается при CGO_ENABLED=1.
Детальные результаты: [0031-evidence.md](specs/0031-evidence.md).

Основания: [OWASP Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html),
[OWASP Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html).
