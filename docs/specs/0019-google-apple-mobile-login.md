# SPEC-0019: Google, Apple и мобильный вход по приглашению

- Статус: draft
- Автор: Codex
- Дата: 2026-10-06
- Acceptance owner: owner
- Acceptance record: отсутствует; владелец выбрал Google и Apple для проработки, реализация и rollout не разрешены этим поручением
- Implementation branch: отсутствует
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Нужны популярные альтернативы Telegram: Google и Apple Account (Apple ID),
с сохранением приглашений, финансового аккаунта и собственного сеанса Grace.
Вход должен завершаться в исходном браузере. Обе кнопки доступны на обеих
платформах при настроенном provider; Android/iOS не ограничивают выбор аккаунта.
Telegram остаётся дополнительным методом для уже привязанных пользователей.

Исследование, зависимости и источники: [authentication-options](../authentication-options.md).

## Scope

- Google OIDC и Sign in with Apple для Web, два явных provider adapters.
- Первое подключение только с действующим одноразовым приглашением.
- Явная привязка Google/Apple к существующему Telegram/другому Grace user;
  управление собственными привязками и guard последнего рабочего метода.
- Восстановление активной сессии на `/login`, безопасный return path,
  мобильные ошибки/отмена/возвращение и переход из встроенного браузера.
- Google Cloud OAuth credentials; Apple Services ID/App ID/private key,
  ротация client secret и документированный операторский runbook.
- OpenAPI/types, migrations, ownership/session/CSRF regressions, device QA.

## Non-goals

Passkeys, пароль, email OTP, открытая регистрация, автоматическое объединение
по email/имени, Mini App, native приложения/публикация в App Store, перенос
сессии между браузерами, отключение защитных уведомлений провайдеров,
изменение финансовой модели и production rollout без отдельного разрешения.
Срок собственной сессии остаётся семь дней.

## Текущее поведение и доказательства

Grace имеет users отдельно от identities `(issuer,subject)`; invitation
потребляется атомарно при первом связывании. Telegram использует Code Flow,
PKCE, state/nonce и browser cookie; cookie и server session живут семь дней.
`LoginPage.tsx` не проверяет существующую сессию. Callback принимает GET,
ставит cookie и возвращает 303 на `/`. Provider paths и auth mode привязаны
к Telegram; этот adapter нельзя просто переименовать для Google/Apple.

Владелец наблюдает отсутствие возврата на телефоне в Telegram browser и Chrome.
Конкретный этап не воспроизведён. Google официально запрещает OAuth в embedded
WebView; замена Telegram на Google без внешнего браузера не гарантирует улучшения.
Apple form_post для запрошенных scopes несовместим с текущим GET-only callback
и требует отдельной обработки browser binding при SameSite=Lax.

## Предметная модель и инварианты

1. Один User UUID может иметь Google, Apple и Telegram identities. Финансовые
   owner_id/роли не меняются. Email, включая Apple relay, — metadata, не ключ
   связывания; verified email также не разрешает автоматическое объединение.
2. Flow хранит provider, purpose (login/link/reauth), browser binding, state,
   nonce, verifier если provider поддерживает PKCE, return path, target user
   при linking и invitation hash при первом подключении. TTL 10 минут,
   одноразовое завершение, запрет provider/purpose mix-up и replay.
3. Token exchange и JWT validation выполняются на Go-сервере: подпись/JWKS,
   ожидаемые issuer/audience, сроки и nonce; provider-specific алгоритмы и
   endpoint configuration. Issuer/JWKS не выбираются из непроверенного token.
   Для Google использовать PKCE S256; для Apple строго следовать текущему
   документированному confidential-client flow, не считать PKCE поддержанным
   только потому, что он работает у Telegram/Google.
4. Для неизвестной identity требуется invitation. Отмена/provider failure,
   неправильный browser/context или used token не расходуют invitation.
   Identity binding, потребление invitation и session issuance атомарны.
5. Linking начинается из собственной сессии, требует свежей проверки
   существующим методом (не старше пяти минут), CSRF/Origin и явного согласия.
   Сервер привязывает к своему target user; чужую identity не перепривязывает.
   Неизвестному вошедшему без приглашения предлагается войти существующим
   методом и добавить новый, а не создаётся второй финансовый аккаунт.
6. Связи `(issuer,subject)` уникальны. Concurrent linking/unlinking сериализованы
   по пользователю/identity. Последний рабочий основной метод защищён;
   удаление binding отзывает связанные Grace sessions. Audit не содержит tokens.
7. Apple `user` profile может прийти только при первом consent; отсутствие
   имени при следующих входах допустимо. Profile сохраняется после проверки
   identity; unsigned name/email не дают полномочий. Hide My Email поддерживается.
8. Apple callback POST получает provider response, сохраняет её краткоживуще
   на сервере и возвращает 303 к same-origin GET finish с непрозрачным одноразовым
   locator. Этот GET обязан подтвердить исходную browser cookie до exchange,
   binding и session issuance. Locator сам не является правом входа; он не
   содержит invitation, code, JWT или identity. Cookie финансовой сессии остаётся
   Secure/HttpOnly/SameSite=Lax. Нет обхода cookie проверки через state alone.
9. Обычные provider authorization codes передаются только штатным callback;
   session/refresh/id tokens и invitation не копируются в URL/clipboard.
   Callback/finish — no-store, referrer policy и безопасные access logs;
   после завершения transient URL очищается redirect. Bound payload/attempt limits.
10. Собственная session проверяется независимо от provider. Locked/dev остаются
    явными; provider availability конфигурируется отдельно. Ошибка настройки
    Apple не отключает рабочий Google/Telegram. Действующая сессия на `/login`
    открывает разрешённый раздел; смена пользователя явная с очисткой cache.
11. Return path — allowlist локальных routes, без external/protocol-relative/
    backslash/encoded обходов и loops. Default `/`. Нельзя отключать browser
    binding, чтобы исправить переключение Chrome/Telegram.

## API и OpenAPI

Предварительный контракт, требуется acceptance:

| Маршрут | Назначение |
| --- | --- |
| GET `/api/auth/methods` | Настроенные доступные providers и безопасные UI capabilities |
| POST `/api/auth/login/{provider}` | Start code flow; optional invitation и safe returnTo |
| GET `/api/auth/callback/google` | Google code/state callback |
| POST `/api/auth/callback/apple` | Ограниченный form-urlencoded receiver, без выдачи session |
| GET `/api/auth/callback/apple/finish` | Same-origin browser-bound completion |
| GET `/api/auth/identities` | Собственные provider bindings, без credential material |
| POST `/api/auth/identities/{provider}/link` | Start explicit fresh-auth binding flow |
| DELETE `/api/auth/identities/{id}` | Unlink свой provider с last-method guard |

Существующие Telegram endpoints совместимы. Flow purpose не задаётся
непроверенным callback body. DTO/limits/error codes в OpenAPI, types генерируются.
Public errors не раскрывают users/email. CSRF/Origin для собственных mutations;
provider callback защищён protocol state/nonce/browser proof, не обычным UI-CSRF.

## Данные, миграция и rollback

Расширить flow/session metadata для provider/purpose/target/auth time и связанной
identity, добавить bounded ephemeral Apple response и binding audit. Unique
identities и users сохраняются, финансовые таблицы не изменяются. Auth data
недоступны financial role. Management queries используют server principal.
TTL cleanup удаляет только ephemeral flows. Sensitive provider data не логируются.

Rollback не должен уничтожать единственные способы входа Google/Apple-only
пользователей; после binding небезопасный downgrade отказывает. Отключение кнопки
не удаляет identities или users. Recovery не выполняет новый financial owner.
Если Apple revocation/account-change требует refresh material, определить до
реализации минимальное зашифрованное хранение, key lifecycle и retention; не
запрашивать Google offline access ради обычной Grace session без необходимости.

## UI

«Продолжить с Google», «Продолжить с Apple», дополнительный Telegram.
Обе основные кнопки доступны всем; UA не выбирает провайдера за пользователя.
При первом подключении invitation вводится на Grace и хранится в server flow.
При следующем входе код не нужен. Настройки показывают собственные привязки,
добавление другого способа и понятное сообщение last-method guard.

Из Telegram WebView открыть Grace в обычном Chrome/Safari ДО начала flow.
Инструкция/ссылка не обещает автоматический запуск конкретного приложения на
любой ОС. По возврате в исходную вкладку проверять session, а не запускать
новый OIDC. Успешное завершение возвращает на разрешённый раздел. Нельзя
обещать отсутствие всех Google/Apple security alerts или принудительный OS switch.

## Настройка провайдеров и зависимости

Google: Cloud project, Web OAuth client, consent screen, production domain,
точный HTTPS redirect, secret management и тестовые/production settings.
Минимальные login scopes; нет доступа к Gmail/Drive/Calendar.

Apple: подтвердить наличие Apple Developer Program и допустимого primary App ID,
создать Services ID и key, зарегистрировать домен/return URL, настроить
серверное создание и ротацию JWT client secret. Официальный setup документ
указывает существующее приложение в App Store, использующее Sign in with Apple;
Account Help описывает связь Services ID с primary App ID. Для текущего Web-only
Grace применимость должна быть подтверждена до обещания реализации Apple.
Создание/публикация native app и покупка membership не входят в это поручение.

## Ошибки и негативные сценарии

Provider mix-up, wrong audience/issuer/nonce/signature, replay/expiry, foreign
linking, disabled users, relay email, отсутствие повторного Apple profile,
cross-site POST без Lax cookie, forged locator/response, потеря браузерного
контекста, provider consent cancellation, неверные scopes/config/expired secret,
concurrent unlink и last-method removal, active-session login и account switching.
Ошибка не создаёт session/второй аккаунт и не потребляет invitation частично.

## Критерии приёмки

- [ ] AC1: Google и Apple code flows валидируются server-side; ни token mix-up,
  ни replay, ни disabled owner не дают доступ.
- [ ] AC2: Invite-only onboarding атомарен для обоих providers, cancel/failure
  сохраняют invitation; race не создаёт две привязки или два owner UUID.
- [ ] AC3: Existing Telegram user явно добавляет Google/Apple к тому же аккаунту;
  email/relay совпадения не объединяют users; чужие identities защищены.
- [ ] AC4: Apple form_post завершает flow через исходный браузер без ослабления
  cookie/CSRF policy; поддельный POST/locator не даёт session и не тратит invite.
- [ ] AC5: `/login` использует действующую сессию, return allowlist защищена,
  смена пользователя очищает financial client cache; TTL остаётся семь дней.
- [ ] AC6: Unlink/audit/fresh-auth/session revocation и concurrent last-method
  guard корректны; admin не получает чужие финансовые или binding привилегии.
- [ ] AC7: Android Chrome, iOS Safari/Chrome и Telegram browser tested на целевых
  устройствах: открыть внешний браузер, consent/2FA, back/cancel/reload, return.
  Не выдавать server fixture за доказательство мобильного возврата.
- [ ] AC8: Provider config/Apple eligibility и secret rotation проверены, отсутствие
  одного provider не отключает другой; секреты не попадают в Git/логи/URL.
- [ ] AC9: OpenAPI/types, migration/rollback, make test, go vet, integration,
  git diff --check и evidence каждого AC проходят.

## Исполняемые проверки

План: отдельные cryptographic Google/Apple fixtures, code exchange mocks без
real credentials; PostgreSQL identity/invitation/linking races и cross-owner
cases; browser contexts для Apple POST→GET, lost cookie, stale session и redirect;
Web tests active-session/restoration/account switching; actual-device matrix.
Финансовые suites сохраняются. Runtime implementation отсутствует.

## Открытые решения

1. Подтвердить полный scope реализации Google + Apple после этой проработки.
2. Проверить Apple account/App Store prerequisites для текущего Web-only Grace.
3. Утвердить login scopes/profile retention, revocation/recovery/token retention.
4. Уточнить ОС/версии телефона владельца для обязательного mobile QA.
5. При внешней блокировке Apple решить порядок: Google сначала, Apple после
   выполнения prerequisites; не заменять второй метод другим без решения владельца.
