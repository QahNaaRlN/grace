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

1. pull exact API/Web images;
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
drill. Пока копии находятся только на том же VPS, потеря VPS остаётся
непокрытым риском.

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
