# SPEC-0019: Passkeys, приглашения и мобильный вход

- Статус: draft
- Автор: Codex
- Дата: 2026-10-06
- Acceptance owner: owner
- Acceptance record: отсутствует; владелец поручил проработать варианты, scope реализации не принят
- Implementation branch: отсутствует
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Владелец получает уведомление Telegram при каждом новом входе через провайдера,
хочет независимый вход с сохранением приглашений и наблюдает отсутствие возврата
на телефоне во встроенном браузере Telegram и Chrome. Нужны вход на странице
Grace, повторное использование собственной сессии и понятное завершение/повтор
Telegram flow без потери границы между браузерами.

Проработка и сравнение: [authentication-options](../authentication-options.md).
Рекомендуемый вариант этого draft — passkey + существующий Telegram как дополнительный вход.

## Scope

- Passkey/WebAuthn: вход, создание первого ключа по приглашению, добавление
  нескольких ключей к действующему аккаунту, список и удаление своих ключей.
- Существующие пользователи и финансовый owner UUID сохраняются; Telegram
  identities остаются пригодными для входа.
- `/login` проверяет текущую сессию; вход возвращает в разрешённый раздел.
- Мобильный UX: один браузер для flow, восстановление состояния при возвращении,
  инструкция открыть внешний браузер при несовместимости, ясные ошибки.
- Серверные проверки WebAuthn, одноразовые challenges, browser binding,
  свежая проверка личности при управлении ключами, audit и ограничения попыток.
- OpenAPI/generated types, миграция, mobile/negative/integration tests и runbook.

## Non-goals

Email OTP, пароль, новые OIDC providers, Mini App, native SDK, автоматическое
слияние аккаунтов, публичное восстановление по одному имени/email/приглашению,
перенос cookie между браузерами, подавление служебных сообщений Telegram,
изменение финансовой модели и production rollout без отдельного решения.
Срок сессии остаётся семь дней; поддержка всех WebView не обещается.

## Текущее поведение и доказательства

`LoginPage.tsx` не запрашивает текущую сессию перед показом формы.
`handler.go` ставит session cookie на семь дней и после callback отвечает 303
на `/`; `repository.go` задаёт такой же абсолютный срок сессии.
Flow живёт 10 минут и требует совпадения state и browser cookie. Provider
запрашивает `openid profile`, PKCE и nonce. Дополнительного метода входа нет.
Мобильная жалоба подтверждена владельцем; конкретный этап сбоя не воспроизведён.
Перед исправлением получить на устройстве ОС/версии и trace без секретов.

## Предметная модель и инварианты

1. User и его финансовые owner_id не зависят от метода входа. Приглашение —
   право первого подключения, а не многоразовый пароль или recovery token.
2. Первый passkey регистрируется только с неиспользованным, действующим
   приглашением к допустимому target user. Для уже привязанного пользователя
   добавление credential требует его активной сессии и повторной проверки
   существующим методом не старше пяти минут. Mixed-purpose запросы запрещены.
3. Только после успешной проверки registration в одной транзакции создать
   credential, потребить invitation и выдать сессию. Отмена, expiry, replay и
   конфликт не расходуют invitation и не создают частичной привязки.
4. WebAuthn использует ожидаемые RP ID/origin из доверенной конфигурации,
   HTTPS, криптографический challenge, проверку подписи и user verification.
   Attestation по умолчанию none, discoverable credentials; hardware keys
   не исключать принудительным platform-only ограничением.
5. Challenges живут не более пяти минут, одноразовые и связаны с браузером,
   purpose и при необходимости owner/invitation. Нельзя использовать login
   assertion как registration или binding. Проверять точный Origin для POST.
6. Credential ID уникален, userHandle непрозрачен, owner определяется сервером.
   Private key/биометрия на сервер не передаются. Для synced credentials
   поддерживать корректную семантику backup/signCount, включая допустимый ноль;
   не вводить самодельную криптографию. Выбор Go-библиотеки и совместимость
   с текущим toolchain проверяются до принятия implementation dependencies.
7. Метод сессии не меняет principal/CSRF/RLS. Locked/development и production
   config остаются явными. Включение passkey отдельно конфигурируется; проверка
   сессии не должна зависеть только от строки `mode=telegram`.
8. Добавление/удаление ключа требует явного действия, fresh authentication,
   CSRF/Origin и audit. Нельзя удалить последний основной рабочий метод.
   Проверка последнего метода и изменения credentials сериализуются по user;
   одновременное удаление разных ключей не обходит guard. Отзыв ключа должен
   отзывать связанные с ним сессии; сохраняется audit без private material.
   Admin не может присвоить другой аккаунт через публичное управление ключами.
9. Имя, email и Telegram username не являются основанием автопривязки.
   Recovery при потере всех методов — другой уже привязанный метод или ручная
   операторская процедура с проверкой личности и сохранением user UUID.
10. Return path только локальный из allowlist; запрещены внешние URL,
    protocol-relative/backslash/encoded обходы и redirect loops. Default `/`.
    Browser-context mismatch не снимается ради успешного redirect.

## API и OpenAPI

Предварительный контракт, уточняется при принятии:

| Маршрут | Назначение |
| --- | --- |
| GET `/api/auth/methods` | Доступные способы и безопасные UI-capabilities, без users/credentials |
| POST `/api/auth/passkeys/registration/options` | Приглашение для первого ключа либо fresh authenticated enrollment |
| POST `/api/auth/passkeys/registration/finish` | Проверить registration и атомарно завершить purpose |
| POST `/api/auth/passkeys/login/options` | Discoverable assertion challenge, без раскрытия аккаунтов |
| POST `/api/auth/passkeys/login/finish` | Проверить assertion, выдать непрозрачную session cookie |
| GET `/api/auth/passkeys` | Собственные ключи: имя/даты, без private material |
| DELETE `/api/auth/passkeys/{id}` | Удалить свой ключ после fresh auth; guard последнего метода |

Telegram login получает безопасный returnTo, сохранённый сервером в flow.
Все DTO/challenge/error limits задаются в OpenAPI, TypeScript генерируется.
Ошибки: malformed/oversize 400/413, unauthenticated 401, Origin/CSRF/fresh-auth
403, чужой key 404, replay/purpose/last-method conflict 409, failed authentication
без раскрытия identity, unsupported/unavailable method 503. Rate limit 429.
Неверное/использованное/истёкшее invitation не раскрывает чужую учётную запись.

## Данные, миграция и rollback

Новые auth credential/challenge/audit таблицы и metadata метода/времени auth
в sessions; финансовые таблицы не изменяются. Версия миграции назначается по
актуальному main при реализации. Credential хранит public key, credential ID,
userHandle, owner, sign counter/backup flags и безопасные management metadata.
Auth repository имеет отдельную цель доступа; financial role не получает grants
на credential/challenge tables. Management queries проверяют owner на сервере.
Challenge/identity/credential refs защищены FK и constraints.

TTL cleanup удаляет истёкшие ephemeral challenges, не permanent credentials/audit.
Секреты не попадают в URL, логи, CI artifacts и аналитику. Инструкции backup
учитывают auth credentials и sessions. Отключение метода не удаляет пользователей.
После появления passkey-only аккаунтов downgrade должен отказать, если уничтожит
единственный способ входа; не обходить guard удалением ключей или user rows.

## UI

На сайте: «Войти ключом доступа», дополнительный «Войти через Telegram»,
отдельный сценарий первого подключения по приглашению. Понятное объяснение
локального PIN/биометрии и возможности синхронизации/второго ключа.
Действующая сессия открывает разрешённый раздел без нового OIDC. Смена пользователя
на общем устройстве — явное действие, с очисткой приватного client cache.

Passkey finish восстанавливает session через `/auth/me` и открывает раздел в той
же странице. При unsupported browser, cancel, expiry или другом browser context
предлагается понятный fallback. Встроенный Telegram browser тестируется отдельно;
отсутствие platform authenticator не должно ошибочно скрывать поддерживаемые
hardware/cross-device credentials; capability detection не является доказательством
поддержки конкретного WebView.
при необходимости инструкция открыть Grace в Chrome/Safari до начала flow.
Нельзя обещать принудительный автоматический foreground switch мобильной ОС.
Popup SDK не заменяет существующий code flow без отдельной оценки.

## Ошибки и негативные сценарии

Wrong RP/origin/signature/UV/userHandle, disabled owner, foreign key management,
used/revoked invitation, concurrent enrollment, replay challenge/assertion,
expired/revoked session, stale auth при добавлении ключа, последний метод,
смена браузера, back/refresh/cancel и незавершённый provider flow.
Ошибка не выдаёт сессию, не меняет owner_id, не потребляет invitation частично.
Browser binding/PKCE/state/nonce не ослабляются. Повторный показ страницы не
должен сам создавать новый credential или запускать Telegram.

## Критерии приёмки

- [ ] AC1: Пользователь с действующей сессией открывает `/login` без нового OIDC;
  reload/возвращение в ту же вкладку восстанавливает сессию и private cache безопасно.
- [ ] AC2: Invite-only passkey enrollment атомарно и одноразово; cancel/failure
  сохраняют invitation; concurrent completion не привязывает двух владельцев.
- [ ] AC3: Существующий Telegram user добавляет ключ к тому же UUID после fresh
  auth; следующий passkey login не обращается к Telegram и открывает те же данные.
- [ ] AC4: Проверяются origin/RP/challenge/signature/UV/owner и replay/TTL;
  ни другой admin, ни совпадающее имя/email не присваивают аккаунт.
- [ ] AC5: Управление собственными ключами требует fresh auth/CSRF/audit;
  последний метод защищён, отключённый пользователь не входит.
- [ ] AC6: Успешный callback в исходном браузере возвращает к разрешённому
  разделу; browser mismatch даёт понятный fallback и не ослабляет проверку.
- [ ] AC7: Реальная матрица телефона владельца: Chrome и Telegram browser;
  проверить Android/iOS целевые версии, cancel/back/reload и метод открытия Chrome.
  Отдельно записать, требуется ли ручное возвращение из приложения Telegram.
- [ ] AC8: Совместимые migrations/rollback/locked mode, OpenAPI/generated types,
  make test, go vet, integration, git diff --check и evidence для каждого AC.

## Исполняемые проверки

План, production implementation отсутствует:
unit auth/Web тесты для active-session login и redirect allowlist;
WebAuthn cryptographic fixtures вместо mocked success-only assertions;
PostgreSQL enrollment/concurrency/ownership tests, multi-method session/CSRF
regressions; отдельные browser contexts для отсутствующей cookie; device QA
и диагностический trace без tokens. Действующие финансовые suites должны пройти.

## Открытые решения

1. Подтвердить passkey как первый новый метод либо выбрать email/пароль.
2. Уточнить ОС/версии телефона и этап, на котором остаётся пользователь после Telegram.
3. Утвердить правила recovery для passkey-only пользователей и operator verification.
4. Выбрать поддерживаемую Go-библиотеку и limits после совместимости/security review.
5. Новая длительность сессии, другие providers и production rollout — отдельные решения.
