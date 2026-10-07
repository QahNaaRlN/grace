# Google и Apple: проработка входа по приглашению

Дата: 2026-10-06. По указанию владельца Google и Apple Account (Apple ID)
выбраны двумя основными вариантами для следующего этапа.
[Draft SPEC-0019](specs/0019-google-apple-mobile-login.md)
обновлён; production implementation отсутствует.

## Пользовательский сценарий

На Grace доступны «Продолжить с Google» и «Продолжить с Apple», Telegram —
дополнительный способ. Обе кнопки можно показывать на Android и iPhone: выбор
аккаунта не равен выбору ОС. Не ограничивать Google владельцами Android,
а Apple — владельцами iPhone.

Первое подключение: код приглашения на Grace → выбранный provider → проверка
сервером → однократное потребление invitation → собственная сессия Grace.
Повторный вход не требует приглашения. При уже действующей семидневной сессии
страница /login сразу открывает приложение вместо нового внешнего входа.

Для существующего Telegram-пользователя нужен явный сценарий «Добавить Google»
или «Добавить Apple» внутри своего аккаунта с свежей проверкой существующим
способом. Все identities ссылаются на прежний user UUID; финансовые owner_id,
счета, роли и история не меняются. Email или отображаемое имя не объединяют
аккаунты. Apple может скрывать реальный email, а адрес Google может меняться.

## Сравнение двух методов

| Вопрос | Google | Apple |
| --- | --- | --- |
| На телефоне | Выбор знакомого Google account в обычном браузере | Apple Account; пользовательское подтверждение зависит от устройства/сессии |
| Настройка | Cloud project, Web OAuth client, consent screen, точный redirect | Developer Program, primary App ID, Services ID, domains/return URL и private key |
| Браузер Telegram | Embedded WebView официально запрещён для Google OAuth | Реальный flow требуется проверить; нельзя считать поведение равным Safari |
| Возврат | Server code callback в исходный браузер | При scopes name/email — form_post; нужен отдельный POST→GET finish |
| Идентификатор | Проверенный issuer + sub, не email | Проверенный issuer + sub, не real/relay email |
| Эксплуатация | Client secret и provider config | Key/JWT client-secret rotation; first-consent profile и privacy/revocation semantics |
| Готовность для Grace | Подходит существующей Go/OIDC архитектуре, но нужен adapter/discovery | Та же финансовая архитектура; есть дополнительная внешняя prerequisite-проверка |

## Google: предлагаемое решение

Серверный Authorization Code Flow, PKCE S256, state и nonce. JWT проверяется
на сервере по ожидаемым issuer/audience/JWKS, а не по данным, присланным браузером.
Для Grace достаточно login scopes openid/profile/email; доступ к Gmail, Drive
и Calendar не нужен. Не запрашивать offline access ради собственной сессии.
Использовать только зарегистрированный HTTPS callback и корректное оформление
кнопки Google. One Tap не является обязательным условием первого этапа.

Google запрещает OAuth в embedded user agents. Поэтому пользователь должен
открыть Grace в обычном Chrome/Safari до ввода invitation и начала flow.
Сайт может объяснить, как открыть внешний браузер через меню Telegram; он не
может гарантировать принудительный запуск Chrome на любой мобильной ОС.

## Apple: предлагаемое решение и зависимость

Для Web нужен Services ID, связанный с primary App ID, включённым для Sign in
with Apple, и private key. Документ Apple по настройке прямо указывает существующее
приложение в App Store, использующее этот вход; Account Help описывает связь
с primary App ID. Для текущего Web-only Grace нельзя обещать подключение только
добавлением кнопки: нужно подтвердить допустимую конфигурацию и наличие
prerequisites в аккаунте разработчика. Native app и его публикация не входят
в поручение проработать Web login. Сторонний auth broker не отменяет требований Apple.

Стандартная membership Apple Developer Program — 99 USD в год либо локальная
цена; региональные условия проверяются при enrollment. Наличие membership,
App ID и подходящего приложения у владельца пока не проверено. Оплата или
регистрация аккаунта этой проработкой не выполнялись.

Provider adapter обменивает code на сервере и проверяет ID token. Client secret
Apple — подписанный сервером JWT; предусмотреть создание/ротацию и мониторинг
expiry вместо бессрочной строковой настройки. Не переносить Telegram JWKS paths
или без проверки предполагать поддержку PKCE у Apple.

Если запрашиваются name/email, Apple возвращает данные cross-site form POST.
Текущий handler Grace принимает GET, а flow cookie имеет SameSite=Lax. Предлагается
bounded POST receiver: сохранить response краткоживуще на сервере, сделать 303
на same-origin GET finish и только там подтвердить исходную cookie до выдачи
сессии/привязки. Непрозрачный одноразовый locator не содержит code/token/invitation
и не даёт доступа без browser proof. Финансовую cookie не ослаблять.
Этот проект решения требует browser tests; не выдавать его за уже доказанный flow.

Apple profile name может быть доступен только при первом consent. Сохранить
нужные display metadata после проверки identity; отсутствие имени при следующих
входах допустимо. Hide My Email поддерживается, relay адрес не является ключом
объединения с Google или Telegram. Для revocation/account changes определить
минимальное token retention и защищённое хранение перед implementation.

## Что это решает в исходных жалобах

- Google/Apple вход не вызывает Telegram login notification. Однако обещать
  отсутствие любых уведомлений/2FA от самих Google или Apple нельзя.
- Приглашения остаются отдельным условием первого допуска; смена способа не
  создаёт второй аккаунт и не переносит финансовые данные.
- В обычном браузере flow должен завершаться на Grace в той же вкладке.
  Это проверяется на устройствах; альтернативный provider сам по себе не
  устраняет ограничения встроенного браузера или переключения приложений.
- /login сначала проверяет собственную сессию. В текущем коде эта проверка
  отсутствует, хотя session TTL уже семь дней.

Точная причина наблюдаемого Telegram-return сбоя не доказана. Callback Grace
уже отвечает 303 на / после успешного входа; возможны незавершённый provider
flow, браузер без исходной cookie либо просто Telegram на переднем плане.
Browser binding не отключать и не переносить session между Chrome/WebView.

## Предлагаемый порядок

1. Утвердить scope Google + Apple, same-account linking и mobile return.
2. Исправить повторное использование сессии и return path как общую часть.
3. Реализовать и проверить Google; параллельно подтвердить Apple prerequisites.
4. Реализовать Apple adapter и POST callback после подтверждения допустимости.
5. Проверить Android Chrome, iOS Safari/Chrome и Telegram browser: initial invite,
   повторный вход, linking, cancellation, back/reload и потеря cookie.
6. Выпустить выбранные провайдеры после acceptance/CI/отдельного rollout решения.

Если Apple prerequisites потребуют native-приложения, вынести это в отдельное
решение. Не заменять выбранный Apple другим provider без владельца.

## Проверенные источники

Источники проверены 2026-10-06; факты провайдеров необходимо перепроверять перед rollout.

- [Google OIDC](https://developers.google.com/identity/openid-connect/openid-connect): code flow, claims и sub.
- [Google embedded browser policy](https://developers.googleblog.com/en/upcoming-security-changes-to-googles-oauth-20-authorization-endpoint-in-embedded-webviews/).
- [Apple Web setup](https://developer.apple.com/help/account/capabilities/configure-sign-in-with-apple-for-the-web).
- [Apple environment prerequisites](https://developer.apple.com/documentation/signinwithapple/configuring-your-environment-for-sign-in-with-apple).
  Официальный Markdown доступен по тому же URL с суффиксом .md; он прочитан напрямую.
- [Apple authorization/callback](https://developer.apple.com/documentation/signinwithapplerestapi/request-an-authorization-to-the-sign-in-with-apple-server.).
- [Apple authentication/profile behavior](https://developer.apple.com/documentation/signinwithapple/authenticating-users-with-sign-in-with-apple).
- [Apple client secret](https://developer.apple.com/documentation/accountorganizationaldatasharing/creating-a-client-secret): ссылка из official Sign in with Apple setup.
- [Apple membership](https://developer.apple.com/help/account/membership/program-enrollment).
