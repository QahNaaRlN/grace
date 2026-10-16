# SPEC-0050: реализация и проверка

Дата: 2026-10-16. Спецификация: [accepted SPEC-0050](0050-offsite-backup.md).
Ветка: `codex/offsite-backup` от `main` `cb4a432`, implementation commit — см. git log ветки.
Статус: `verified`. Все критерии AC1–AC8 проверены локально и в production
(см. «Production-проверки»).

## Что сделано

- `deployments/production/offsite-upload.sh`: шифрует dump открытым ключом `age`,
  считает SHA-256, загружает `*.dump.age` и `*.dump.age.sha256` через `rclone`
  одиночным PUT (`--s3-no-head`, `--no-check-dest`, `--s3-no-check-bucket`), ставит
  маркер `*.dump.uploaded`, удаляет локальный `.age`. Без `offsite.env` —
  одно предупреждение и выход 0. При сбое пишет `backups/.offsite-failure`
  (этап, файл, время) и выходит с ошибкой; режим `--pending` догружает все
  dump без маркера.
- `backup.sh`: после проверки dump вызывает загрузку; новый режим
  `upload-pending`; при ротации daily удаляет и маркер.
- `deploy.sh`: загрузка pre-release копии; сбой — предупреждение, выпуск
  продолжается.
- `grace-backup.service` → `OnFailure=grace-backup-failure.service` →
  `backup-failure-mail.sh`: письмо через Postbox (`curl` SMTP, учётные данные
  только в приватном config-файле curl, не в аргументах).
- `scripts/restore-offsite.sh`: drill на машине владельца в контейнерах
  (скачивание, SHA-256, расшифровка, восстановление в одноразовый PostgreSQL,
  число строк).
- `offsite.env.example`, бандл и `docs/production-deployment.md` (bucket,
  lifecycle, политика из четырёх правил, установка, сбои, drill, полная потеря VPS).

## Уточнение механизма проверки загрузки

Scope говорит «после загрузки размер и SHA-256 загруженного объекта
сравниваются с локальным файлом». Ключ VPS по SPEC-0050 не может читать
объекты, поэтому прочитать объект обратно нельзя. Реализация проверяет
целостность при самой загрузке: одиночный PUT возвращает ETag = MD5, и rclone
сравнивает его с локальным MD5. SHA-256 загружается рядом (`.sha256`) и
проверяется при восстановлении ключом владельца (`restore-offsite.sh`).
Цель AC1 (копия в облаке совпадает с локальной) сохраняется.

## Критерии приёмки

| Критерий | Исполняемое свидетельство | Статус |
|---|---|---|
| AC1 | `scripts/test-offsite-backup.sh`, «AC1/AC2»: daily загружается, объект скачивается, `sha256sum -c` проходит, расшифровка даёт исходный dump, в `.age` нет открытого текста | пройдено локально и в production |
| AC2 | Тот же блок: в `/opt/grace` нет `AGE-SECRET-KEY`; в production — `grep -rlE 'AGE-SECRET-KEY-1[0-9A-Z]{20}' /opt/grace` пуст | пройдено локально и в production |
| AC3 | Команды отказа из `docs/production-deployment.md` (list, cat, deletefile) и проверка версии после перезаписи | пройдено в production |
| AC4 | Тест «AC4»: неверный ключ → выход ≠ 0, dump на месте, маркера нет, `stage=upload`, секретов в выводе нет; письмо с этапом, получателем, без паролей в аргументах и тексте | локально пройдено |
| AC5 | Тест «AC5»: без `offsite.env` одна строка вывода (путь dump) и ровно одно предупреждение | локально пройдено; `test-production-deployment.sh` — см. ниже |
| AC6 | Тест «AC6 (synthetic)»: `restore-offsite.sh` восстанавливает настоящий `pg_dump` (роль `grace_financial`, RLS-политика, bytea) и печатает ожидаемые числа строк | пройдено синтетически и на production-копии |
| AC7 | Lifecycle в консоли Yandex Cloud | пройдено |
| AC8 | `make test`, `git diff --check`, `shellcheck` (v0.10.0) новых и изменённых скриптов | см. ниже |

## Результаты (Windows, Docker 29.8.2)

- `sh scripts/test-offsite-backup.sh`: exit 0, «off-site backup checks passed»
  (локальный S3 — `rclone serve s3` из Alpine 3.20; образы MinIO больше не
  скачиваются публично). Тест добавлен в `make test-production`, который CI
  запускает на Linux.
- `make test`: exit 0 — Go `ok`, web 75 файлов / 294 теста, ESLint, Steiger,
  Prettier, сборка, Jev offline checks. В одном прогоне параллельно с Docker-тестом
  два web-теста упали по 5-секундному таймауту; отдельно и в повторном полном
  прогоне проходят. Web-код не менялся.
- `git diff --check`: чисто.
- `shellcheck` v0.10.0 (`-s sh`): `backup.sh`, `offsite-upload.sh`,
  `backup-failure-mail.sh`, `restore-offsite.sh`, `test-offsite-backup.sh` — без
  замечаний. У `deploy.sh` те же 6 замечаний, что на `main`; новые строки их не добавляют.
- `scripts/test-production-deployment.sh` на Windows падает с «no current Grace
  release» и на чистом снимке `main` тоже (символические ссылки Git Bash);
  проверяется CI на Linux.

## Настройка Yandex Cloud — 2026-10-16

По поручению владельца через консоль в его каталоге настроены: bucket с
ограниченным доступом и версионированием; три правила lifecycle (daily 35 дней,
pre-release и manual 180 дней, неактуальные версии 35 дней); сервисные аккаунты
`grace-backup-uploader` (`storage.uploader` на bucket) и `grace-backup-reader`
(`storage.viewer` на bucket) без ролей на каталог; политика доступа из четырёх
правил по `docs/production-deployment.md`. После сохранения политики bucket
открывается в консоли. Статические ключи не создавались: их создаёт владелец.

Сначала планировалась политика только с `Deny` для загрузчика. По описанию
порядка проверки доступа Yandex запрос, не подошедший ни под одно правило
политики, уходит на проверку ACL объекта, поэтому такая политика могла отнять
доступ у читателя и консоли. Политика дополнена явными `Allow`.

## Первый production deploy — 2026-10-16

Deploy `822b8f8` (run 38876154732) прошёл, релиз здоров. Pre-release копия
в облако не ушла: в `GRACE_OFFSITE_AGE_RECIPIENT` оказался закрытый ключ `age`,
а не открытый. `age` отказал и вывел значение в лог deploy, что нарушало AC4.

Меры:
- логи run 38876154732 удалены в GitHub (API отвечает 404) и в локальном кэше `gh`;
- ключевая пара считается скомпрометированной: владелец создаёт новую и удаляет
  старый закрытый ключ с VPS. Старым ключом ничего не зашифровано, потому что
  шифрование не прошло;
- `offsite-upload.sh` принимает только значение `age1…`. Закрытый ключ
  (`AGE-SECRET-KEY-…`) и любое другое значение отклоняются на этапе `config`,
  при этом значение не выводится. Вывод ошибок `age` больше не попадает
  в журнал;
- `test-offsite-backup.sh`: закрытый ключ в настройке → отказ `stage=config`,
  ключа нет ни в выводе, ни в письме. Неверный `age1…` → отказ без вывода `age`.
  Проверка AC2 ищет само значение ключа, а не строку-префикс.

## Production-проверки — 2026-10-16

Релиз `fc19d32` (deploy run 38878996293) здоров. Pre-release backup ушёл
в облако во время deploy: `off-site copy uploaded: pre-release-20261016T091417Z-….dump.age`.

- **AC1.** Ручной backup `manual-20261016T092101Z-sha-fc19d32….dump.age` и его
  `.sha256` есть в bucket. `upload-pending` догрузил всю прежнюю историю: 7 daily,
  pre-release с 2026-10-05, manual 2026-10-05.
- **AC2.** `offsite.env` проходит проверку из PR #53: получатель `age1…`.
  `sudo grep -rlE 'AGE-SECRET-KEY-1[0-9A-Z]{20}' /opt/grace` на VPS ничего не вывел:
  закрытого ключа на сервере нет.
- **AC3.** С ключом VPS: список — `AccessDenied 403`, чтение — `AccessDenied 403`,
  прямой `DELETE` через `curl --aws-sigv4` — `403`. Объект после этого на месте.
  Версионирование bucket включено.
- **AC6.** `scripts/restore-offsite.sh` на машине владельца (Windows, Git Bash,
  ключ читателя) скачал `manual-20261016T092101Z…`, `sha256sum -c` — OK, расшифровал,
  восстановил в чистый PostgreSQL: `financial_events 153`, `postings 162`,
  `accounts 11`, `loan_documents 0`. Совпадает с production сразу после backup
  (`153 | 162 | 11 | 0`).
- **AC7.** Три правила lifecycle настроены (см. «Настройка Yandex Cloud»).

Найдено при drill и исправлено в `restore-offsite.sh`: файлы, сохранённые
в Блокноте Windows, заканчивают строки на CRLF, и `
` попадал в значения; файл
ключа начинался со строки `Public key: …` с экрана `age-keygen`. Скрипт теперь
удаляет `
` и берёт из файла ключа только строки `AGE-SECRET-KEY-`. Тест
`test-offsite-backup.sh` проверяет восстановление из таких файлов.

## Что осталось владельцу

Ничего обязательного. Закрытый ключ `age` и ключ читателя хранятся у владельца
(менеджер паролей и офлайн-копия). Restore drill стоит повторять после крупных
изменений схемы или раз в несколько месяцев.
