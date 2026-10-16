# SPEC-0050: реализация и проверка

Дата: 2026-10-16. Спецификация: [accepted SPEC-0050](0050-offsite-backup.md).
Ветка: `codex/offsite-backup` от `main` `cb4a432`, implementation commit — см. git log ветки.
Статус: `implementing`. Локальные проверки пройдены; production-проверки AC2, AC3,
AC6 и AC7 ждут создания bucket и ключей владельцем (открытое решение 2).

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
  lifecycle, политика `Deny`, установка, сбои, drill, полная потеря VPS).

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
| AC1 | `scripts/test-offsite-backup.sh`, «AC1/AC2»: daily загружается, объект скачивается, `sha256sum -c` проходит, расшифровка даёт исходный dump, в `.age` нет открытого текста | локально пройдено; production — после установки |
| AC2 | Тот же блок: в `/opt/grace` нет `AGE-SECRET-KEY`; в production — `grep -r AGE-SECRET-KEY /opt/grace` пуст | локально пройдено; production — ожидает |
| AC3 | Команды отказа из `docs/production-deployment.md` (list, cat, deletefile) и проверка версии после перезаписи | ожидает bucket; локальный S3-сервер политики не эмулирует |
| AC4 | Тест «AC4»: неверный ключ → выход ≠ 0, dump на месте, маркера нет, `stage=upload`, секретов в выводе нет; письмо с этапом, получателем, без паролей в аргументах и тексте | локально пройдено |
| AC5 | Тест «AC5»: без `offsite.env` одна строка вывода (путь dump) и ровно одно предупреждение | локально пройдено; `test-production-deployment.sh` — см. ниже |
| AC6 | Тест «AC6 (synthetic)»: `restore-offsite.sh` восстанавливает настоящий `pg_dump` (роль `grace_financial`, RLS-политика, bytea) и печатает ожидаемые числа строк | синтетически пройдено; на production-копии — ожидает |
| AC7 | Lifecycle в консоли Yandex Cloud | ожидает bucket |
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

## Что осталось владельцу (production)

1. Создать ключ `age` и bucket, аккаунты и политику по
   `docs/production-deployment.md`, заполнить `offsite.env` на VPS.
2. Выпустить релиз с этой веткой, установить units, выполнить ручной backup.
3. Проверить отказы (AC3), lifecycle (AC7), drill с машины владельца (AC6)
   и записать результаты сюда; после этого статус — `verified`.
