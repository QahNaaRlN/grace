# SPEC-0022: основной вход по почте и персональный профиль

- Статус: integrated
- Автор: Codex
- Дата: 2026-10-07
- Acceptance owner: владелец Grace
- Acceptance record: «добавим авторизацию через почту ... аватарку, имя, фамилия ... почту как основной способ, телеграм временно отключить»; владелец подтвердил отсутствие почтового сервиса и текущего сеанса, требуется перенос через администратора.
- Implementation branch: codex/email-profile

## Проблема и результат

Вход в той же вкладке по одноразовому почтовому коду, с сохранением приглашений.
Профиль содержит отображаемое имя, имя, фамилию, подтверждённую почту, аватар
и управление собственными сеансами. Telegram отсутствует среди способов входа
в email mode; прежние identities сохраняются для обратимого отключения.

## Scope / non-goals

Go/PostgreSQL OTP flow, SMTP c TLS, login/request+verify, явная привязка/смена
почты, имена, загрузка/удаление своего аватара PNG/JPEG, актуальный OpenAPI,
React формы и операторская миграция существующего пользователя.
Google/Apple/SMS, пароли, публичные профили, дополнительные PII, финансовые
изменения и автоматическое объединение по имени/почте исключены.

## Поведение до SPEC-0022

Telegram OIDC — единственный production login. User имеет id/name/role.
Почтовой доставки, verified email, имён/фамилии и аватара нет. Приглашение
создаёт user заранее; auth identities и финансовые owner_id используют его UUID.

## Модель и инварианты

Identity email имеет canonical lowercase ASCII address; plus/dots не удаляются.
Сессии остаются непрозрачными на сервере, HttpOnly Secure SameSite=Lax cookie.
OTP 6 ASCII digits, TTL 10 минут, до 5 попыток, однократный и привязан к случайной
browser cookie и purpose/user. В БД только digest OTP и digest browser token.
Вход нового пользователя возможен только с действительным invitation; token
расходуется атомарно при успешном входе. Никаких автоматических новых владельцев
для существующих профилей: связывание внутри сеанса либо операторское приглашение,
привязанное к целевой почте и существующему UUID, затем проверка почтового кода.
Связанные роли, owner_id, счета/операции/долги не изменяются.

Один подтверждённый email на пользователя, адрес уникален. Смена через новый код;
коллизия адреса — отказ без объединения. Привязка/смена подтверждаются текущим
сеансом, созданным не более 15 минут назад, и CSRF. При смене email другие сеансы отзываются. OTP, invitation, SMTP
credentials не попадают в логи/URL/аватар/профиль. Request response не раскрывает
наличие адреса в базе. Лимиты отправки адрес/источник/глобальный бюджет — в БД,
перезапуск API их не сбрасывает; просроченные challenge очищаются.

Аватар <=2 MiB на входе, PNG/JPEG, <=4M pixels; decode config до полного decode,
перекодировать в квадратный PNG 256px без EXIF/оригинального файла; хранение
малого изображения в БД пользователя, доступ только самому пользователю.
SVG/GIF/URL-upload запрещены; filename и Content-Type не являются доказательством
формата. Ни один профильный endpoint не принимает owner_id/role.

## API / OpenAPI

GET /api/auth/methods — email/telegram/locked availability без секретов.
POST /api/auth/email/request и /verify — почта+invitation, затем code.
GET/POST /api/auth/profile; POST /email/request, /email/verify; GET/POST/DELETE
/avatar под Require/CSRF; UserProfile DTO и typed errors. POST mutations и DELETE
avatar требуют существующей проверки Origin/CSRF. Avatar uses multipart binary.
Обновить /me mode enum и generated TypeScript.

## Данные / миграция / rollback

Аддитивная 000008: first_name/last_name/avatar в users; auth email challenges,
rate windows, optional bound_email в invitations. Финансовые таблицы не менять.
Rollback приложения сохраняет новые данные; down migration только после backup
и отключения email flow. Операторская миграция создаёт приглашение на 24 часа,
привязанное к указанной почте и существующему пользователю. CLI не печатает секрет:
сервер подбирает подготовленное приглашение при запросе кода для этой почты.
Без доказательства владения mailbox профиль не связывается. При неуспешном
развёртывании откат восстанавливает режим авторизации предыдущего работающего API.

## UI / ошибки

Почта → код на той же странице → собственная сессия; invitation для первого
входа. Pending, resend cooldown, invalid/expired code, delivery failure, collision,
revoke/disabled user; generic messages. Изменение имени и аватара отражается
в текущем профиле, существующие финансовые данные не переносятся и не теряются.
Почта не имитируется placeholder SMTP на production. Telegram выключается
при действующей доставке и подготовленном пути восстановления владельца.

## Acceptance / checks

- AC1: login code/invitation/expiry/one use/browser binding/attempt limit — unit+PG tests.
- AC2: migration keeps existing UUID, finance and role; different email collision denied — PG.
- AC3: name/profile/verified email/avatar bounds and ownership — unit+PG+Web tests.
- AC4: email mode disables Telegram Login and callback, active own sessions still usable — tests.
- AC5: SMTP TLS required, credentials/OTP absent logs, persistent rate/budget — tests.
- AC6: make test, git diff --check, API regeneration, isolated integration and CI.
- AC7: production delivery test, owner migration and successful owner email sign-in before handoff.

## External prerequisites / open decisions

SMTP provider/domain setup and owner verification are required before switching
production login. These prerequisites were completed on 2026-10-07 with Yandex
Cloud Postbox and Timeweb DNS; the owner personally verified email sign-in.
Previous Telegram identities are retained. Google's/Apple's SPEC-0019 remains draft.
Release and executable acceptance evidence: [0022-evidence.md](0022-evidence.md).
