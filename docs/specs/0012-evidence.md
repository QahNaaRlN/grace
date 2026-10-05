# SPEC-0012 evidence: owner-only production deployment

- Specification: `docs/specs/0012-owner-vps-production-deployment.md`
- Accepted specification commit: `35bd7f5`
- Implementation commits: `bd408c1..23f2a72`
- Implementation branch: `codex/owner-vps-production-implementation`
- Pull request: https://github.com/QahNaaRlN/grace/pull/2
- Integration commit: `a9bc3ac47cf5f363d39737d6eb0b1e7a740180c1`
- Production target: `graceapp.ru`, `195.133.195.190`

## Acceptance criteria and executable evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| Accepted scope | Owner explicitly accepted SPEC-0012, local-only backup risk, Basic Auth, and preservation of the existing WireGuard service. | PASS |
| VPS preflight | SSH commands confirmed Ubuntu 26.04 LTS, x86_64, 1 vCPU, 1.6 GiB RAM, 1.8 GiB swap, 23 GiB free disk, IPv4 `195.133.195.190`, and assigned IPv6 `2a12:bec4:1ac0:1181::`. | PASS |
| WireGuard preservation | Preflight recorded active `wg0`, UDP 51820 and existing nftables FORWARD/NAT state. After Docker install and firewall activation, `wg0` still listened on IPv4/IPv6 and retained its transfer counters and FORWARD rules. | PASS |
| Same-origin Web | Production Web build uses an empty API base URL; image scan found no `localhost:8080`. | PASS |
| Private service ports | Rendered production Compose publishes ports only for Caddy; API, Web, and PostgreSQL remain Docker-internal. | PASS |
| Owner authentication | Public `https://graceapp.ru` returned `401` without credentials and `{"service":"grace","version":"0.5.0-dev"}` with Basic Auth. | PASS |
| Production migration | Production database applied migrations 1-5, contained four reference categories and zero accounts; development seed was absent. | PASS |
| Persistence | PostgreSQL restart retained schema migration version 5. | PASS |
| Backup and restore | VPS has verified pre-release, manual and daily dumps with mode `0600`; the post-migration dump restored into a separate database. Timer is active, persistent and retains seven daily files by executable test. | PASS |
| Immutable manual release | Run `37245993850` accepted integration SHA `a9bc3ac`, required successful main CI, and deployed exact API/Web `sha-a9bc3ac...` images. | PASS |
| Rollback | Deploy keeps the previous release symlink and restarts previous application containers on failed smoke without a down migration. | PASS (static/local path) |
| Secret boundary | Repository scan found no SSH private key, GitHub token, production Basic Auth password, PostgreSQL password, or TypeSafe key. Test-only placeholders are confined to the test script. | PASS |
| Assistant disabled | Production Compose sets `GRACE_ASSISTANT_ROUTING_ENABLED=false`. | PASS |
| DNS delegation | `.ru`, Timeweb authoritative DNS, Cloudflare and Google resolve A `195.133.195.190` and AAAA `2a12:bec4:1ac0:1181::`; that IPv6 is assigned to `eth0`. | PASS |
| Dedicated deploy identity | `grace-deploy` owns `/opt/grace`, belongs to the Docker group, authenticates with a dedicated Actions key, and the root bootstrap key is absent from GitHub. | PASS |
| Public TLS and first release | Caddy serves a Let's Encrypt certificate for `graceapp.ru`, valid through 2027-01-02. Public Web and all formerly failing API routes returned `200` with credentials. | PASS |

## Local verification transcript

```bash
make test
make test-integration
make test-production
git diff --check
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 \
  .github/workflows/ci.yml .github/workflows/delivery.yml \
  .github/workflows/deploy-production.yml
docker build -f Dockerfile -t grace-api:spec0012 .
docker build -f web/Dockerfile -t grace-web:spec0012 .
ssh root@195.133.195.190 'nft -c -f -' \
  < deployments/production/grace-firewall.nft
```

All commands above passed on 2026-10-05. `make test` reports the existing Vite
chunk-size warning for the 646.61 kB minified main bundle; it is non-fatal and
does not change the production deployment contract.

## Isolated production exercise

An isolated Compose project on alternate host ports exercised the exact
production bundle. It proved pre-migration backup, migrations, readiness,
Basic Auth, same-origin `/api/meta`, zero seeded accounts, manual backup,
restore drill, and persistence after PostgreSQL restart. Both temporary test
projects and their Docker volumes were removed afterward.

## GitHub and production evidence

- PR CI run `37245019653`: all quality, integration and image jobs passed.
- Main CI run `37245216799`: all jobs passed for integration SHA `a9bc3ac`.
- Delivery run `37245386283`: exact-SHA API and Web images published.
- Production run `37245993850`: SHA validation, upload, GHCR login, deploy,
  internal readiness, external Basic Auth smoke and logout passed.
- `docker compose ps` reported API, Web and PostgreSQL healthy; only Caddy
  published host ports 80/443 TCP and 443 UDP.
- Public routes `/`, `/api/meta`, accounts, events, receipts, obligations,
  cash position and period summary all returned `200` with Basic Auth.
- Host listeners after firewall activation were SSH 22, Grace 80/443 and the
  existing WireGuard UDP 51820; PostgreSQL 5432 and API 8080 were not published.
- A fresh SSH connection, authenticated HTTPS and self-addressed IPv6 HTTPS
  returned successfully after the additive nftables policy was applied.
- Daily backup service completed with status 0 and the timer scheduled its next
  run. Manual restore drill completed before the timer was enabled.
