# GitHub delivery Grace

## Что считается CI и CD

`CI` проверяет каждый pull request и каждый push в `main`:

- Go tests, Web tests, ESLint, TypeScript/Vite build и Jev offline datasets;
- изолированные PostgreSQL integration suites;
- production Docker builds API и Web;
- отсутствие drift сгенерированного TypeScript API contract.

`Delivery` запускается только после успешного CI на push в `main` или вручную.
Он публикует два container image в GitHub Container Registry:

- `ghcr.io/qahnaarln/grace-api`;
- `ghcr.io/qahnaarln/grace-web`.

Фактический owner в image path приводится workflow к lowercase. Для каждого
image создаются tags:

- `sha-<полный commit SHA>` — versioned tag конкретной revision;
- `main` — moving pointer на последнюю принятую сборку.

Оба tag в одном run указывают на один immutable OCI digest; digest записывается
в workflow summary.

GHCR delivery — ещё не production deployment. Workflow не подключается к
постоянной PostgreSQL, не запускает production migrations, не создаёт domain/TLS
и не включает assistant router.

## Где смотреть результат

1. GitHub repository → **Actions**: отдельные runs `CI` и `Delivery`.
2. Commit или pull request → **Checks**: `Quality gates`, integration и две
   production image builds.
3. GitHub profile/repository → **Packages**: `grace-api` и `grace-web`.

Красный job нужно открывать до первого failing step. Итоговый зелёный workflow
не заменяет проверку всех обязательных jobs.

## Permissions и secrets

CI имеет только `contents: read`. Delivery имеет `contents: read` и
`packages: write`; публикация использует короткоживущий `GITHUB_TOKEN`.

В GitHub нельзя добавлять:

- `.env`;
- `TYPESAFE_API_KEY` в Git, image build args или logs;
- PostgreSQL dumps и локальные volumes;
- personal access tokens в workflow YAML.

Runtime secrets понадобятся только отдельному deployment environment после
выбора hosting.

## Особенность Web image

React получает `VITE_API_BASE_URL` во время build. Текущий production Dockerfile
по умолчанию собирает Web для `http://localhost:8080`, что подходит локальному
запуску опубликованных artifacts. Публичный deployment должен до сборки
зафиксировать настоящий API origin либо принять отдельный runtime-config
контракт; менять один и тот же immutable SHA image под разные origins нельзя.

## Следующая deployment-фаза

После выбора hosting и domain нужна отдельная спецификация, которая определит:

- production compose/PaaS/Kubernetes topology;
- managed PostgreSQL, backups и restore drill;
- migration preflight, apply и rollback gate;
- HTTPS, CORS и `VITE_API_BASE_URL`;
- health checks и автоматический rollback;
- environment protection и ручное production approval.
