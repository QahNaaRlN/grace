# Production deployment Grace

После SPEC-0013 новая версия API закрыта по умолчанию (`GRACE_AUTH_MODE=locked`).
По SPEC-0018 публичный экран входа не требует Basic Auth. По SPEC-0022 основной
production-вход — почтовый код; Telegram временно отключён. Настройка SMTP,
аддитивной миграции 000008 и подтверждаемый перенос owner описаны в
[email runbook](email-authentication-rollout.md). Историческая проверка SPEC-0012
относится к версии без пользовательских сессий.

Этот runbook реализует SPEC-0012 с обновлениями SPEC-0013–0015, SPEC-0018 и SPEC-0022.
Caddy открывает Web shell и экран входа; API закрывает финансовые данные
серверными сессиями и tenant isolation. Доступ новых пользователей — по
приглашению. PostgreSQL остаётся во внутренней Docker network, assistant router выключен.

## Production topology

```text
Internet
  ├─ TCP 22                 SSH
  ├─ TCP 80/443, UDP 443    Caddy / HTTPS
  └─ UDP 51820              существующий WireGuard, не принадлежит Grace

Caddy → Web
      → Grace API → PostgreSQL volume
```

API, Web и PostgreSQL не публикуют host ports. `deployments/production` нельзя
заменять development-файлом `compose.yaml`.

## DNS

Authoritative DNS должен содержать:

```text
A     graceapp.ru  195.133.195.190
AAAA  graceapp.ru  2a12:bec4:1ac0:1181::
```

До release обе записи проверяются через независимый resolver и прямой IPv4/IPv6
маршрут. Если IPv6 не проходит, AAAA удаляется до исправления; наличие A
достаточно для первого release.

## One-time VPS bootstrap

Bootstrap выполняется по pinned host key. Существующие `wg0`, UDP 51820,
WireGuard FORWARD/NAT rules не изменяются.

Установить distribution packages:

```bash
apt-get update
apt-get install -y docker.io docker-compose-v2 apache2-utils curl
systemctl enable --now docker
```

После первого успешного release additive nftables unit разрешает SSH, Grace,
ICMP/ICMPv6 и существующий WireGuard, не очищая другие таблицы и не меняя
WireGuard FORWARD/NAT rules:

```bash
install -m 0644 /opt/grace/current/grace-firewall.service \
  /etc/systemd/system/grace-firewall.service
systemctl daemon-reload
/opt/grace/current/install-firewall.sh
systemctl enable grace-firewall.service
```

Перед закрытием bootstrap SSH session нужно открыть второе SSH-соединение и
повторно проверить `wg show`; при любой ошибке удалить только таблицу Grace:

```bash
nft delete table inet grace_input
```

Создать отдельного deploy-пользователя и каталоги:

```bash
useradd --create-home --shell /bin/bash grace-deploy
usermod --append --groups docker grace-deploy
install -d -m 0700 -o grace-deploy -g grace-deploy /home/grace-deploy/.ssh
install -d -m 0750 -o grace-deploy -g grace-deploy /opt/grace
install -d -m 0750 -o grace-deploy -g grace-deploy /opt/grace/releases
install -d -m 0750 -o grace-deploy -g grace-deploy /opt/grace/shared
install -d -m 0750 -o grace-deploy -g grace-deploy /opt/grace/shared/secrets
install -d -m 0700 -o grace-deploy -g grace-deploy /opt/grace/backups
```

Dedicated deployment public key записывается в
`/home/grace-deploy/.ssh/authorized_keys` с mode `0600`. Bootstrap root key не
копируется в GitHub и после проверки deploy-user может быть удалён из root.

## Runtime secrets

Базовые файлы без завершающих комментариев:

```text
/opt/grace/shared/secrets/postgres_password
```

PostgreSQL password генерируется как URL-safe random value. Telegram credentials
хранятся отдельно в защищённом `telegram.env`, см. authentication runbook.

```bash
umask 077
openssl rand -hex 32 > /opt/grace/shared/secrets/postgres_password
chown grace-deploy:grace-deploy /opt/grace/shared/secrets/*
chmod 0600 /opt/grace/shared/secrets/*
```

Dedicated SSH private key записывается как `PRODUCTION_SSH_KEY`.
Ни один secret не добавляется в repository. Старые `basic_auth_hash` и
`PRODUCTION_BASIC_AUTH_PASSWORD` больше не используются новым релизом;
сохранить их для rollback, не удалять автоматически.

После выпуска проверить без cookies и общих credentials:

```bash
./scripts/check-public-entrypoint.sh https://graceapp.ru
```

Web/login/meta должны вернуть 200 без Basic challenge; me/accounts — 401 JSON.

## Release bundle

Bundle создаётся из exact repository SHA:

```bash
./scripts/build-production-bundle.sh /tmp/grace-release
```

Он содержит production Compose, Caddy configuration, deploy/backup scripts и
production migration set. Development migration `000002_development_seed` в
bundle отсутствует; вместо неё создаются только reference categories.

## Manual GitHub release

GitHub → Actions → **Deploy production** → **Run workflow**.

Можно указать полный SHA из `main`; пустое поле означает текущий SHA workflow.
Workflow проверяет successful CI push run, передаёт exact bundle, временно
авторизует VPS в GHCR и запускает immutable tags `sha-<commit>`.

Deploy выполняет:

1. pull exact API/Web/PDF-helper images;
2. verified pre-deploy `pg_dump`, включая пустую database первого release;
3. production migrations;
4. API/Web/Caddy rollout;
5. internal readiness и unauthenticated `401`;
6. внешний authenticated HTTPS smoke;
7. GHCR logout.

При failed internal smoke application containers возвращаются к предыдущему
release. Database down migration не выполняется автоматически.

## Manual backup and restore drill

```bash
/opt/grace/current/backup.sh
/opt/grace/current/restore-drill.sh /opt/grace/backups/<backup>.dump
```

Backup считается пригодным только после `pg_restore --list` и отдельного restore
drill. Зашифрованная копия вне VPS описана в разделе «Off-site backup».

Ежедневный timer устанавливается при bootstrap:

```bash
install -m 0644 /opt/grace/current/grace-backup.service \
  /etc/systemd/system/grace-backup.service
install -m 0644 /opt/grace/current/grace-backup.timer \
  /etc/systemd/system/grace-backup.timer
systemctl daemon-reload
systemctl enable --now grace-backup.timer
```

Команда без аргумента создаёт `manual-*` backup. Deploy создаёт
`pre-release-*` backup, в том числе пустой проверенный dump перед первой
migration. Эти копии автоматически не удаляются. Timer создаёт `daily-*` и
хранит последние семь таких файлов.

## Off-site backup (SPEC-0050)

Каждый `daily`, `manual` и `pre-release` dump после проверки шифруется
открытым ключом владельца (`age`) и загружается в Yandex Object Storage.
Закрытого ключа на VPS нет. Ключ облака на VPS может только загружать объекты.
Без `/opt/grace/shared/secrets/offsite.env` всё работает как раньше, с одним
предупреждением «copies stay on this VPS only».

### Ключ владельца

На своей машине (Linux/macOS: пакет `age`; Windows: через Docker):

```bash
docker run --rm -it alpine:3.20 sh -c 'apk add --no-cache --quiet age && age-keygen'
```

Строка `AGE-SECRET-KEY-…` — закрытый ключ. Сохранить его в менеджер паролей и
отдельно офлайн (например, распечатка или флешка в надёжном месте). Без него
копии не расшифровать. Строка `public key: age1…` идёт в `offsite.env`.

### Bucket и сервисные аккаунты (консоль Yandex Cloud)

1. Bucket с ограниченным доступом (не публичный), например `grace-backups-<суффикс>`,
   класс хранения «Стандартное». Включить **версионирование**.
2. Жизненный цикл bucket, три правила; в каждом удалять неактуальные версии
   (NoncurrentVersionExpiration) через 35 дней:
   - префикс `grace/daily-` — удалять объекты через 35 дней;
   - префикс `grace/pre-release-` — через 180 дней;
   - префикс `grace/manual-` — через 180 дней.
3. Сервисные аккаунты без ролей на каталог. Роли назначаются только на bucket
   (Безопасность → Права доступа):
   - `grace-backup-uploader` — `storage.uploader`. Его статический ключ
     попадает только в `offsite.env` на VPS;
   - `grace-backup-reader` — `storage.viewer`. Его статический ключ хранится
     только у владельца, для восстановления, и на VPS не кладётся.
4. Политика доступа bucket (Безопасность → Политика доступа), четыре правила:

   | Правило | Результат | Кто | Действия | Ресурсы |
   |---|---|---|---|---|
   | `uploader-put-only` | Разрешить | `grace-backup-uploader` | `s3:PutObject` | `<bucket>/*` |
   | `reader-restore` | Разрешить | `grace-backup-reader` | `s3:GetObject`, `s3:ListBucket` | `<bucket>/*`, `<bucket>` |
   | правило для доступа из консоли | Разрешить | все, условие `PrincipalIsAWSService = true` | все | `<bucket>/*`, `<bucket>` |
   | `uploader-no-read` | Запретить | `grace-backup-uploader` | `s3:GetObject`, `s3:GetObjectVersion`, `s3:ListBucket`, `s3:ListBucketVersions` | `<bucket>/*`, `<bucket>` |

   Почему так: если политика задана, запрос, не подошедший ни под одно её
   правило, проверяется по ACL объекта ([порядок проверки
   доступа](https://yandex.cloud/ru/docs/storage/security/overview)). Поэтому
   каждому, кому нужен доступ, — загрузчику, читателю и консоли — нужно явное
   `Allow`, иначе он может потерять доступ. Правило `Deny` тоже не даёт
   окончательного отказа, а передаёт решение ACL объекта, поэтому отказ на чтение
   проверяется командами ниже. Удаление роль `storage.uploader` не разрешает;
   перезапись возможна, но версионирование сохраняет прежнюю версию.

### Установка на VPS

```bash
apt-get install -y age rclone curl
install -m 0600 -o grace-deploy -g grace-deploy /dev/null /opt/grace/shared/secrets/offsite.env
# заполнить по deployments/production/offsite.env.example
install -m 0644 /opt/grace/current/grace-backup.service /etc/systemd/system/grace-backup.service
install -m 0644 /opt/grace/current/grace-backup-failure.service \
  /etc/systemd/system/grace-backup-failure.service
systemctl daemon-reload
sudo -u grace-deploy /opt/grace/current/backup.sh
```

Последняя команда должна вывести путь dump и строку
`off-site copy uploaded: <name>.dump.age sha256=…`.

Проверка прав (каждая команда должна получить отказ доступа):

```bash
sudo -u grace-deploy sh -c 'set -a; . /opt/grace/shared/secrets/offsite.env; set +a
export RCLONE_CONFIG_O_TYPE=s3 RCLONE_CONFIG_O_PROVIDER=Other \
  RCLONE_CONFIG_O_ENDPOINT="$GRACE_OFFSITE_ENDPOINT" RCLONE_CONFIG_O_REGION="$GRACE_OFFSITE_REGION" \
  RCLONE_CONFIG_O_ACCESS_KEY_ID="$GRACE_OFFSITE_ACCESS_KEY_ID" \
  RCLONE_CONFIG_O_SECRET_ACCESS_KEY="$GRACE_OFFSITE_SECRET_ACCESS_KEY"
o="o:$GRACE_OFFSITE_BUCKET/grace"
rclone lsf "$o"; rclone cat "$o/<name>.dump.age" | head -c1; rclone deletefile "$o/<name>.dump.age"'
```

### Сбои

При ошибке backup или загрузки `grace-backup.service` завершается с ошибкой,
а `grace-backup-failure.service` отправляет письмо на `GRACE_OFFSITE_ALERT_TO`
через Postbox (те же SMTP-настройки, что в `email.env`). Письмо содержит этап,
имя файла и время, без секретов и финансовых данных. Локальный dump при сбое
загрузки сохраняется. Догрузить всё, что не ушло в облако:

```bash
sudo -u grace-deploy /opt/grace/current/backup.sh upload-pending
```

Сбой загрузки во время deploy не останавливает выпуск: выводится
предупреждение, копию догружает `upload-pending` или следующий запуск.

### Restore drill с машины владельца

Нужны Docker и POSIX shell (на Windows — Git Bash). Файл с ключом читателя
(`GRACE_OFFSITE_ENDPOINT`, `_REGION`, `_BUCKET`, `_PREFIX`, `_ACCESS_KEY_ID`,
`_SECRET_ACCESS_KEY` аккаунта `grace-backup-reader`) и закрытый ключ `age`
хранятся у владельца.

```bash
scripts/restore-offsite.sh ~/grace-reader.env ~/grace-age-identity.txt
```

Скрипт скачивает последний `daily-*.dump.age`, проверяет SHA-256, расшифровывает,
восстанавливает его в одноразовый PostgreSQL и печатает число строк в
`financial_events`, `postings`, `accounts`, `loan_documents`. Для сравнения
с production сразу после ручного backup выполнить на VPS тот же `SELECT count(*)`.

### Полная потеря VPS

1. Поднять новый VPS по разделам выше (bootstrap, секреты из хранилища
   владельца, первый deploy пустой базы не выполнять).
2. На машине владельца скачать и расшифровать нужную копию:
   `rclone copyto <remote>/<name>.dump.age .` и
   `age -d -i <identity> -o grace.dump <name>.dump.age`.
3. Перенести `grace.dump` на VPS, запустить `postgres` текущего release, создать
   роль `grace_financial` (`CREATE ROLE grace_financial NOLOGIN`) и выполнить
   `pg_restore -U grace -d grace --exit-on-error < grace.dump`, затем deploy.

## Operations

```bash
cd /opt/grace/current
docker compose --project-name grace-production --env-file release.env \
  -f compose.yaml ps
docker compose --project-name grace-production --env-file release.env \
  -f compose.yaml logs --tail=200 api web caddy postgres
```

Не использовать `docker compose down -v`: production volumes не удаляются во
время release или rollback.


Для Telegram deploy.sh дополнительно загружает защищённый /opt/grace/shared/secrets/telegram.env; см. docs/authentication.md. API имеет исходящий HTTPS-доступ через edge, PostgreSQL остаётся на внутренней backend network.


## PDF statements — SPEC-0055

Production Compose запускает versioned `grace-pdf-extractor:sha-<commit>` вместе
с API. Helper не имеет сети, secrets, host ports или Docker socket; read-only
root, UID 10001, 512 MiB memory/swap, 1 CPU, 16 PIDs и private tmpfs 16 MiB.
API получает read-only mount Unix socket и ждёт helper healthcheck.
Delivery публикует helper image с тем же exact SHA, что API/Web. Deploy проверяет
наличие socket и non-root helper. Statement AI по умолчанию `false`;
защищённый `statement.env` хранит явный opt-in владельца и pinned policy.

После релиза владелец проверяет PDF upload и ручной review в «Импортах».
Сетевой LLM extraction и Jev qualification не включаются командой deployment.
Существующие analysis/receipt settings сохраняются. Откат на образ до SPEC-0055
отказывается, если уже есть bank-statement source/original/evidence: требуется
совместимый forward fix. Database down и удаление sources не выполняются.


### Активация Statement AI — 2026-10-19

По прямому поручению владельца подготовлен `statement_action=enable` в workflow
Deploy production. `preserve` (default) сохраняет существующий runtime choice;
`disable` выключает statement inference без вызовов модели. Перед enable
`statement-ai.sh` запускает `grace-statement-admin -mode=qualify` на двух
поставляемых synthetic PDF. Весь верхний plan ≤$0.032 резервируется в прежнем
usage ledger; неизвестная стоимость не освобождается как бесплатная.

После успешного smoke и проверки endpoint metadata атомарно записывается
`/opt/grace/shared/secrets/statement.env` (0600) и выполняется обычный deploy
того же SHA с backup/readiness. Флаг/policy переживают следующие релизы.
Существующие analysis/receipt не меняются, global kill switch сохраняется.
Failed smoke не меняет настройку; failed rollout восстанавливает прежний файл
и snapshotted runtime. Повтор enable при уже включённой действующей policy
не делает новых платных calls. Host lock исключает конкурентную активацию.

Полный synthetic report хранится приватно на VPS; CI получает только итог/дату,
без ключа, budget/usage values и provider payload. Read-only
`inspect_statement_only=true` также не экспортирует runtime values.
Актуальные даты/фактический enabled status: [activation evidence](specs/0055-activation-evidence.md).

Исторический срез до reconciliation 2026-10-19: `1ada384` deployed/internal healthy
(run `39172255492`), последующий activation step FAIL, statement flag false.
Последний synthetic candidate probe `39174108946`: columns PASS, cards provider
FAIL; readiness `39174447002` требует billing review. Не повторять платный
qualifier и не обнулять uncertain reservation. Для сверки нужен generation ID,
authoritative metadata и соответствие конкретному запуску; данные остаются
приватно на VPS. После сверки cards smoke всё ещё обязан пройти. Временный
workflow для paid candidate probes не включается в основной release workflow;
`inspect_statement_only` остаётся read-only.

Сверка 2026-10-19 12:38:04 +05:00: run `39217226634` PASS, точный расход
соответствующего synthetic attempt подтверждён через `grace_ai_settle`,
ledger готов к дальнейшей qualification. Приватный audit сохранён на VPS;
Statement AI пока false до успешного quality gate и отдельной активации.

Model gate после сверки: v7 probe `39217497119` FAIL (coverage); v8
`39218319463` FAIL (неполная лишняя строка/coverage), v8 откачен. Statement AI
не включён, дальнейшие paid probes этой редакции остановлены. Все ошибочные
результаты сохранены; при зелёном CI нельзя обходить этот quality gate.
