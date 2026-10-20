# Банковская выписка: загрузка, проверка и подтверждение

Актуализация: 2026-10-19. Production `ef782ec` (S1.1 PR #73), deploy `39255669019`
internal health PASS **17:55:14 +05:00 / 12:55:14 UTC**. Ручная загрузка/review
доступны, Statement AI false: единственный S1.1 smoke дал columns
`record_description` FAIL (пропущен дефис), cards NOT_RUN. Повторов нет.
Read-only `39255943568` подтвердил runtime disabled и readiness; public/auth PASS.
[S1.2](specs/0055-s12-description-separators.md) отдельно принят для уточнения
prompt и одного нового bounded smoke. [Актуальный audit](specs/0055-activation-evidence.md).
Поддерживается текстовый PDF одного дебетового RUB-счёта. OCR/сканы,
зашифрованные документы и кредитные условия в этот срез не входят.
Новый банк не требует обязательного parser-профиля, но успешное извлечение для
каждого формата должно быть проверено; универсальная точность не обещается.

## Работа владельца

1. В «Импортах» выбрать существующий дебетовый RUB-счёт и загрузить PDF.
   Загрузка сохраняет источник и локальные spans; финансовых событий и AI-вызова нет.
2. Без AI сопоставить исходные фрагменты полям строк и заголовка. Исходный текст,
   страница, нормализованные значения и проблемы видны рядом. Причина решения
   обязательна. Нераспознанное нельзя заменить догадкой.
3. Если оператор отдельно включил statement AI, сначала подготовить preview.
   Проверить передаваемые минимизированные фрагменты, модель, endpoint, верхнюю
   стоимость и дать отдельное согласие на inference. Это не согласие на ledger.
   При недоступном AI остаётся ручное сопоставление. Автоматического fallback нет.
4. Проверить страницы и роли полей. Даты операции и списания сохраняются раздельно;
   без исходной timezone проекция в ledger остаётся date-only. Выбрать именно
   сумму в валюте счёта, исходную валютную сумму оставить evidence.
5. В обычном review выбрать смысл каждой строки: создать допустимое событие,
   связать с найденным Posting или исключить с причиной. Credit не становится
   доходом автоматически. Перевод себе, заём и неподдерживаемые операции требуют
   осмысленного решения. Сопоставление Receipt не создаёт второго расхода.
6. Получить financial preview и явно подтвердить выбранные строки (до 100 за раз).
   До этого шага ledger не меняется. Повтор того же подтверждения не дублирует запись.

Обороты, book balances, source coverage и представление строк событиями — разные
проверки. Доступный остаток не является closing book balance. Отсутствующие
opening/closing показаны как отсутствующие, а не как ноль. Совпадение оборотов
не подтверждает правильность смысла каждой строки. Сверка исторического ledger
на границах выписки не выполняется; opening balance импорт не меняет.
Исключённые строки остаются видимым неполным финансовым покрытием.

Решения и источник доступны после reload через историю импорта. При конфликте
версии ввод сохраняется, источник перечитывается для повторной проверки.
После первой публикации нельзя перепривязать source к другому счёту или заново
сгруппировать весь документ; следующие правки идут через review строк.

## Хранение

По умолчанию оригинал удаляется после завершения всех строк или через 7 дней
review. Первая частичная группа подтверждения не удаляет оставшийся документ.
Можно явно выбрать keep или удалить оригинал раньше. После удаления скачивание
возвращает 410, полный layout и redacted extraction plan очищены; hash, metadata,
минимальные field fragments, решения и provenance сохраняются. Уже созданные
события не удаляются. Старые backup могут содержать оригинал до истечения их
собственного срока хранения. Keep не отменяет доступ только владельцу.

## Локальный запуск и ограничения

Существующий `compose.yaml` по умолчанию не включает PDF helper. Локальный opt-in:

```sh
docker compose -f compose.yaml -f compose.pdf-extractor.yaml up -d --build
```

Overlay добавляет только socket helper: Poppler 25.04.0-r0, network=none,
read-only root, UID 10001, 512 MiB memory/swap, 1 CPU, 16 PIDs и private tmpfs.
Ни API credentials, ни Docker socket ему не передаются. API получает только
Unix socket. PDF ≤10 MiB, ≤50 страниц; text ≤8 MiB; ≤5000 строк. Превышение
возвращается как проблема, документ не обрезается молча.

Statement AI по умолчанию выключен. Для отдельного квалифицированного окружения
нужны существующие OpenRouter credentials/global AI policy и отдельные
`GRACE_STATEMENT_AI_ENABLED`, `GRACE_STATEMENT_AI_POLICY`. Policy принимает только
утверждённые model `google/gemini-2.5-flash`, endpoint `google-vertex/eu`,
providerName `Google`, положительные ceilings input ≤300000/output ≤2500000
micros на миллион tokens. Это не команда на включение production; policy
проверяет транспорт. Analysis/receipt policies не перенастраиваются.

[Принятый S1](specs/0055-record-atomic-planning.md) передаёт один целый
redacted document: ≤2000 spans / ≤90000 messages+schema bytes,
≤100000 input / ≤16000 output tokens, один call на attempt. Statement call
≤120 s, product attempt ≤150 s, HTTP write deadline 180 s. До первой
передачи резервируется theoretical provider ceiling ≤$0.07; shared allowance
и fees не меняются. Превышение размера требует меньшего периода либо ручного
review, без усечения. Lifetime guard остаётся ≤32 calls/document, но новый
attempt требует нового актуального preview и consent; retries автоматически
не выполняются. Неизвестный расход блокирует новые calls до reconciliation.
Replay/recovery не повторяют оплаченный/неопределённый запрос. Preview показывает
bound, а не фактическую цену или обещание завершения за указанный timeout.

Для воспроизводимой offline проверки:

```sh
./scripts/test-pdf-extractor.sh
./scripts/test-bank-statements.sh
make test
```

Тестовые PDF генерируются `scripts/fixtures/generate-statement-pdfs.py`; это
синтетические данные двух разных layout. Сквозной тест использует настоящий
helper, fake provider и disposable PostgreSQL. Live model и Jev evaluation
не запускаются этими командами. [Evidence](specs/0055-evidence.md) фиксирует
границы выполненных проверок и следующий qualification gate.


## Управление Statement AI

Default остаётся выключенным; production opt-in хранится отдельно в защищённом
`/opt/grace/shared/secrets/statement.env` (400/600, operator/root). Release
загружает флаг и pinned policy, не меняя receipt/analysis. Rollback восстанавливает
снятые с прежнего API флаг/policy. Global AI kill switch сохраняется.

В Deploy production появился явный `statement_action=enable|disable|preserve`.
Enable сначала выполняет небольшой live smoke только на двух поставляемых
synthetic PDF, в общем budget ledger, максимум 2 calls / $0.14 aggregate provider ceiling, без retry и
без создания imports/events. Это отдельно [принятый owner-test smoke](specs/0055-owner-test-activation.md): 120 s/call, 300 s/run, ровно один разрешённый запуск. Неуспех оставляет настройку выключенной; новый платный запуск после FAIL требует нового решения. Неизвестный
расход фиксируется uncertain и блокирует новые calls. Report с usage хранится
приватно на VPS; CI получает только итог и дату, без budget/usage values.
Успешная повторная enable-команда не повторяет платный smoke. Disable не требует
модели/оплаты, исходные финансовые записи сохраняются.

Это проверка активации, не полная AC12 и не доказательство качества всех банков.
Реальная выписка передаётся только по отдельному document-specific consent.
Jev/cascade не включаются автоматически.
