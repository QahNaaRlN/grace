# SPEC-0012: Owner-only production deployment на VPS

- Статус: accepted
- Автор: Codex
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: owner 2026-10-05 явно принял SPEC-0012, owner-only Basic
  Auth deployment и временный residual risk backups только на том же VPS
- Specification branch: `codex/owner-vps-production`
- Implementation branch: отсутствует
- Implementation commit: отсутствует

## Проблема и пользовательский результат

SPEC-0011 публикует проверенные API и Web images в GHCR, но Grace ещё нельзя
безопасно использовать с телефона через публичный domain. Текущий Web image
обращается к `http://localhost:8080`, development Compose публикует PostgreSQL и
содержит тестовые credentials, production migrations включают development seed,
а HTTP API не имеет пользовательской аутентификации.

Пользовательский результат — owner-only Grace по адресу
`https://graceapp.ru`, доступный с телефона после Basic Auth, с TLS, постоянной
PostgreSQL, ручным выпуском проверенного commit SHA, локальным backup перед
изменением и проверяемым rollback. Публикация для других пользователей в этот
этап не входит.

Owner 2026-10-05 зафиксировал исходные параметры:

- domain: `graceapp.ru`;
- VPS: `195.133.195.190`, заявленная ОС Ubuntu Server 26.04 LTS;
- делегированный IPv6 prefix: `2a12:bec4:1ac0:1181::/64`; точный host
  address должен быть подтверждён на интерфейсе VPS до создания AAAA record;
- доступ: только owner, временная защита HTTP Basic Auth;
- backups: временно только на том же VPS.

## Scope

- добавить production topology в `deployments/production/`:
  - Caddy как единственную публичную точку входа на портах 80/443;
  - Web, API и PostgreSQL только во внутренней Docker network;
  - именованный volume PostgreSQL без публикации database port;
  - restart policies, health checks и resource-safe logging;
- обслуживать Web и API через один origin `https://graceapp.ru`:
  - `/api/*` направляется в API;
  - остальные запросы направляются в Web;
  - API URL в production Web не содержит `localhost` и не привязан к build-time
    environment;
- защитить весь пользовательский Web/API контур Caddy Basic Auth:
  - password хранится только как bcrypt hash в VPS secret environment;
  - plaintext password, hash и production `.env` не коммитятся и не попадают в
    GitHub Actions logs;
  - ACME/TLS lifecycle управляет Caddy;
- оставить assistant router default-off; `TYPESAFE_API_KEY` не нужен первому
  production release;
- отделить development seed от production migration path:
  - новая production database получает schema и справочники, но не фиктивный
    счёт `Основная карта` с балансом 125 000 RUB;
  - существующая локальная development database не повреждается;
- deploy только exact immutable tags `sha-<40-char commit>` из GHCR, не `main`;
- добавить manual `workflow_dispatch` production deployment:
  - принимает/вычисляет release SHA;
  - допускает только SHA с успешным CI main run;
  - подключается к VPS выделенным deploy SSH key;
  - временно авторизует pull приватных images из GHCR;
  - создаёт pre-deploy PostgreSQL backup;
  - применяет migrations;
  - поднимает exact SHA release;
  - проверяет container health, API readiness и HTTPS route;
  - при failed smoke check возвращает предыдущие API/Web tags;
- добавить owner runbook для bootstrap, DNS, release, backup, restore и rollback;
- выполнить первый release только после отдельного owner go-ahead на mutation
  VPS и DNS readiness.

## Non-goals

- multi-user registration, sessions, roles, password reset или tenant
  isolation;
- считать Basic Auth окончательной product authentication;
- публичный доступ без owner credentials;
- high availability, несколько VPS, Kubernetes или zero-downtime database
  migration;
- managed PostgreSQL;
- off-site backup в этом work item; отсутствие off-site copy остаётся явно
  принятым residual risk;
- автоматический deploy каждого push в `main`;
- включение assistant router или передача финансовых данных AI provider;
- хранение SSH private key, Basic Auth password, database password или GHCR PAT
  в repository;
- изменение финансовой модели, OpenAPI DTO или бизнес-арифметики.

## Текущее поведение и доказательства

- `graceapp.ru` на preflight 2026-10-05 не делегирован из зоны `.ru` и не
  возвращает NS, A или AAAA record; owner сообщил, что DNS настроен, поэтому
  release ожидает завершения регистрации/делегирования;
- owner сообщил IPv6 prefix `2a12:bec4:1ac0:1181::/64`, но это сеть, а не
  публикуемый host address; фактически назначенный VPS IPv6 пока не подтверждён;
- SSH key `id_ed25519_ubuntu_t7` авторизован, pinned host key принят ранее;
- VPS preflight подтвердил Ubuntu 26.04 LTS, x86_64, 1 vCPU, 1.6 GiB RAM,
  1.8 GiB swap и 30 GiB disk с 23 GiB свободного места;
- интерфейс `eth0` имеет глобальный адрес `2a12:bec4:1ac0:1181::/64`;
- Docker/Compose не установлены; порты 80/443 свободны, SSH слушает IPv4/IPv6;
- GHCR содержит API/Web images с exact SHA tags;
- `web/Dockerfile` по умолчанию компилирует
  `VITE_API_BASE_URL=http://localhost:8080`;
- development `compose.yaml` публикует PostgreSQL host port и использует
  credentials `grace/grace`; он запрещён для production reuse;
- API routes не имеют authentication middleware;
- migration `000002_development_seed` создаёт фиктивный debit account и четыре
  категории;
- production deployment workflow отсутствует;
- branch protection для private repository недоступен на текущем GitHub plan;
  deploy остаётся ручным owner action.

## Предметная модель и инварианты

- PostgreSQL остаётся единственным источником финансового состояния.
- Один production deployment обслуживает одного owner; Basic Auth является
  perimeter gate, а не новой сущностью financial domain.
- Только Caddy публикует host ports. API, Web и PostgreSQL не доступны напрямую
  из Internet.
- Release identity — полный Git commit SHA и соответствующие immutable GHCR
  digests. Moving tag `main` не используется для запуска или rollback.
- Deploy не меняет финансовые данные до успешного pre-deploy backup.
- Backup считается созданным только после успешного `pg_dump` и проверки, что
  dump читается `pg_restore --list`.
- Локальный backup на том же VPS защищает от неудачного release, но не от
  потери VPS. Этот residual risk видим и не называется disaster recovery.
- Rollback приложения не выполняет автоматический down migration. Несовместимая
  schema migration требует отдельного explicit recovery decision.
- Assistant остаётся default-off и не влияет на финансовое ядро.

## API и OpenAPI

OpenAPI paths и DTO не меняются. Browser использует тот же HTTPS origin:
`https://graceapp.ru/api/v1/...`. CORS разрешает только
`https://graceapp.ru`, хотя same-origin path не требует cross-origin доступа.

Health/readiness endpoints не раскрывают финансовые данные. Внешний smoke check
проходит через Basic Auth; container readiness проверяется внутри Docker
network.

## Данные, миграция и rollback

### Production database

- отдельный сильный PostgreSQL password находится в `/opt/grace/.env` с mode
  `0600`;
- database port не публикуется;
- persistent volume не удаляется командами deploy/rollback;
- production migration source не создаёт development account.

Implementation должна выбрать и документировать безопасный переход для уже
существующей migration `000002` без удаления пользовательских данных. Изменение
исторического migration допускается только потому, что production database ещё
не существует, и только вместе с executable проверкой fresh production schema
и сохранением локального development bootstrap.

### Backup

- перед первой migration и каждым release создаётся custom-format `pg_dump`;
- backup path: `/opt/grace/backups/<UTC timestamp>-<release SHA>.dump`;
- permissions не шире `0600`;
- минимум один restore drill выполняется в отдельную временную database;
- retention первого этапа: последние 7 daily backups и все manual pre-release
  backups до явной owner cleanup policy.

### Rollback

- `/opt/grace/releases/current.env` хранит текущие exact image tags;
- предыдущий успешный release сохраняется как `previous.env`;
- failed application smoke check возвращает `previous.env` и повторно запускает
  Compose;
- schema rollback автоматически не выполняется;
- restore из dump требует явного owner confirmation и отдельной команды runbook.

## UI

- mobile-first UI не меняет визуальную структуру;
- production Web обращается к API через текущий origin и работает по HTTPS;
- browser Basic Auth prompt является временным owner-only входом;
- при недоступном API существующие состояния ошибки остаются видимыми, но
  deployment smoke check не принимает такой release как успешный.

## Ошибки и негативные сценарии

- DNS не указывает на `195.133.195.190`: TLS/deploy блокируется до исправления;
- в AAAA записан prefix, network address или IPv6, которого нет на интерфейсе
  VPS: IPv6 publication блокируется; A-only release остаётся допустимым;
- SSH key не авторизован: никакой password не передаётся через chat или GitHub
  logs; bootstrap блокируется;
- host key изменился: deploy останавливается, `StrictHostKeyChecking` не
  отключается;
- порты 80/443 закрыты или заняты: Caddy/TLS gate fails до migration;
- Docker/Compose отсутствует или VPS не соответствует заявленной ОС: bootstrap
  выполняется отдельно и фиксируется в evidence;
- GHCR login/pull failed: текущий release продолжает работать;
- backup failed или dump unreadable: migration/deploy не начинается;
- migration failed: новый API/Web не запускаются, database и logs сохраняются;
- `/ready` или authenticated HTTPS smoke failed: application tags откатываются;
- Basic Auth password/hash отсутствует: Caddy не стартует fail-closed;
- production compose публикует 5432 или 8080: acceptance gate fails;
- development seed появляется в fresh production database: acceptance gate
  fails;
- disk pressure угрожает PostgreSQL/backups: deploy блокируется по preflight;
- rollback требует несовместимой schema: automation останавливается и требует
  owner decision, не вызывает down migration автоматически.

## Критерии приёмки

- [x] Owner явно принял SPEC-0012 и residual risk backup только на VPS.
- [ ] DNS A record `graceapp.ru` указывает на `195.133.195.190`; AAAA создаётся
      только для конкретного адреса из `2a12:bec4:1ac0:1181::/64`, который
      присутствует на VPS и проходит входящий IPv6 preflight.
- [ ] VPS identity, Ubuntu version, CPU/RAM/disk и Docker Compose подтверждены
      executable preflight.
- [ ] Deploy использует не-root dedicated user/key; host key pinning включён.
- [ ] В Internet доступны только SSH policy port и 80/443; PostgreSQL, API и Web
      host ports не опубликованы.
- [ ] `https://graceapp.ru` имеет валидный автоматически обновляемый TLS.
- [ ] Без Basic Auth Web и API возвращают 401; с owner credentials mobile Web и
      API работают.
- [ ] Web не содержит `localhost` API origin и использует same-origin HTTPS.
- [ ] Fresh production migration не создаёт фиктивный development account.
- [ ] Production database сохраняется после container recreation.
- [ ] Pre-deploy dump создан, проходит `pg_restore --list`, имеет mode 0600;
      restore drill в отдельную database успешен.
- [ ] Manual deploy принимает только exact SHA с successful CI и запускает
      соответствующие GHCR images.
- [ ] Failed smoke test проверяемо возвращает предыдущий application release без
      автоматического destructive database rollback.
- [ ] Runtime secrets отсутствуют в Git, workflow logs и image layers.
- [ ] Assistant router выключен в первом release.
- [ ] `make test`, production Compose validation, config secret scan и
      `git diff --check` проходят.
- [ ] Первый production release и post-deploy smoke evidence записаны отдельно.

## Исполняемые проверки

```bash
make test
make test-integration
git diff --check
docker compose -f deployments/production/compose.yaml config
docker build -f Dockerfile .
docker build -f web/Dockerfile .
rg -n 'localhost|grace:grace|published:|5432:|8080:' \
  deployments/production web/Dockerfile web/src
git ls-files | xargs rg -n \
  'BEGIN .*PRIVATE KEY|github_pat_|ghp_|POSTGRES_PASSWORD=|TYPESAFE_API_KEY='
```

VPS acceptance дополняется командами из runbook: DNS lookup, pinned SSH
preflight, `docker compose ps`, internal `/ready`, unauthenticated 401,
authenticated HTTPS smoke, certificate inspection, `pg_dump`,
`pg_restore --list`, restore drill и exact running image digests.

## Открытые решения

1. Owner 2026-10-05 принял specification и residual risk local-only backup.
2. Domain ещё не делегирован из `.ru`. После появления authoritative NS нужно
   подтвердить A `195.133.195.190` и AAAA `2a12:bec4:1ac0:1181::`, затем
   проверить оба маршрута извне до TLS issuance.
3. Для bootstrap owner выбрал отдельный существующий public key
   `id_ed25519_ubuntu_t7.pub`, fingerprint
   `SHA256:V2Cyp4tLlpWZJgNQt2B75hzDvAEOzvF3Wr8Q53KXv74`. Ключ
   `ssh_working` уже используется для другого доступа и не добавляется на VPS.
   Private keys не передаются и не копируются в repository.
4. Basic Auth username предлагается `grace`; password owner создаёт вне chat,
   в Git/VPS хранится только bcrypt hash.
5. После первого успешного release нужен отдельный work item для off-site backup
   до того, как Grace станет единственным местом хранения реальных финансовых
   данных.
