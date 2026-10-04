# SPEC-0012 evidence: owner-only production deployment

- Specification: `docs/specs/0012-owner-vps-production-deployment.md`
- Accepted specification commit: `35bd7f5`
- Implementation commits: `bd408c1`, `3a0c436`
- Implementation branch: `codex/owner-vps-production-implementation`
- Production target: `graceapp.ru`, `195.133.195.190`

## Acceptance criteria and executable evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| Accepted scope | Owner explicitly accepted SPEC-0012, local-only backup risk, Basic Auth, and preservation of the existing WireGuard service. | PASS |
| VPS preflight | SSH commands confirmed Ubuntu 26.04 LTS, x86_64, 1 vCPU, 1.6 GiB RAM, 1.8 GiB swap, 23 GiB free disk, IPv4 `195.133.195.190`, and assigned IPv6 `2a12:bec4:1ac0:1181::`. | PASS |
| WireGuard preservation | Preflight recorded active `wg0`, UDP 51820 and existing nftables FORWARD/NAT state. `nft -c -f -` on the VPS accepted the additive Grace table without applying it. | PASS |
| Same-origin Web | Production Web build uses an empty API base URL; image scan found no `localhost:8080`. | PASS |
| Private service ports | Rendered production Compose publishes ports only for Caddy; API, Web, and PostgreSQL remain Docker-internal. | PASS |
| Owner authentication | Isolated production stack returned `401` without credentials and `{"service":"grace","version":"0.5.0-dev"}` with Basic Auth. | PASS |
| Production migration | Fresh isolated production database applied migrations 1-5 and contained zero accounts; development seed was absent from the release bundle. | PASS |
| Persistence | PostgreSQL restart retained schema migration version 5. | PASS |
| Backup and restore | First pre-migration and post-migration dumps had mode `0600`; both restored into a separate database. Daily retention test retained seven daily files and preserved the manual file. | PASS |
| Immutable manual release | Workflow accepts a full SHA reachable from `main`, requires successful push CI for that SHA, and deploys `sha-<40>` image tags. | PASS (local lint) |
| Rollback | Deploy keeps the previous release symlink and restarts previous application containers on failed smoke without a down migration. | PASS (static/local path) |
| Secret boundary | Repository scan found no SSH private key, GitHub token, production Basic Auth password, PostgreSQL password, or TypeSafe key. Test-only placeholders are confined to the test script. | PASS |
| Assistant disabled | Production Compose sets `GRACE_ASSISTANT_ROUTING_ENABLED=false`. | PASS |
| DNS delegation | `.ru` trace did not yet expose NS/A/AAAA for `graceapp.ru` during preflight. | PENDING DNS |
| Dedicated deploy identity | Bootstrap of `grace-deploy` and dedicated GitHub Actions key has not yet mutated the VPS. | PENDING VPS |
| Public TLS and first release | Requires DNS readiness, merged implementation, published exact-SHA images, and the manual production workflow. | PENDING RELEASE |

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

GitHub-hosted CI, VPS bootstrap, DNS/TLS, public smoke, firewall application,
backup timer activation, and first release evidence must be appended after
those external gates actually complete.
