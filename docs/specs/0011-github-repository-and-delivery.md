# SPEC-0011: GitHub repository и CI/CD delivery

- Статус: accepted
- Автор: Codex
- Дата: 2026-10-05
- Acceptance owner: owner
- Acceptance record: owner 2026-10-05 разрешил GitHub CLI authorization, private repository `QahNaaRlN/grace` и GHCR delivery как первую CD-фазу
- Specification branch: `codex/github-delivery`
- Implementation branch: отсутствует
- Implementation commit: отсутствует

## Проблема и пользовательский результат

Grace существует только в локальном Git repository без remote. Проверки
выполняются вручную, deployable artifacts не публикуются, а состояние `main`
нельзя независимо проверить на GitHub.

Пользовательский результат — owner-controlled GitHub repository, где pull
requests получают обязательный CI verdict, а принятый `main` автоматически
порождает идентифицируемые container images. Реальный deploy отделён от
публикации artifacts и включается только после выбора целевого runtime.

## Scope

- создать GitHub repository `QahNaaRlN/grace` с выбранной owner visibility;
- добавить GitHub как `origin` и впервые отправить локальный `main` без
  переписывания истории;
- добавить `.github/workflows/ci.yml` для pull requests и push в `main`;
- CI выполняет Go tests, Web tests, ESLint, generated OpenAPI types,
  TypeScript/Vite build, Jev offline validation и Docker builds;
- добавить `.github/workflows/delivery.yml` для push в `main` и manual dispatch;
- delivery публикует immutable API и Web images в GHCR с tag по commit SHA и
  moving tag `main`;
- workflow использует минимальные permissions: CI `contents: read`, delivery
  `contents: read`, `packages: write`, provenance только если он будет реально
  генерироваться;
- secrets не записываются в repository, images или workflow logs;
- зафиксировать owner runbook: GitHub Actions, packages, branch protection и
  отличие artifact delivery от runtime deployment;
- после первого push проверить реальные GitHub Actions runs и доступность
  опубликованных packages через GitHub API/CLI.

## Non-goals

- выдавать публикацию images в GHCR за работающий production deployment;
- выбирать VPS, PaaS, Kubernetes, domain, TLS, backups или monitoring без
  отдельного owner-решения;
- переносить локальные `.env`, `TYPESAFE_API_KEY`, PostgreSQL volume или
  финансовые данные в GitHub;
- включать assistant router в опубликованном окружении;
- автоматически применять migrations к неизвестной production database;
- менять financial domain, API или UI;
- force-push, переписывание локальной Git history или удаление локального repo.

## Текущее поведение и доказательства

- `git remote -v` пуст;
- локальный `main` чист и находится на commit `39c1d3d`;
- GitHub CLI установлен, default account — `QahNaaRlN`, но token invalid;
- `.github/workflows/` отсутствует;
- `make test` и `make test-integration` являются существующими executable gates;
- API и Web имеют отдельные production Dockerfiles;
- development `compose.yaml` не является production deployment manifest.

## Предметная модель и инварианты

- GitHub не становится источником финансового состояния; PostgreSQL boundary не
  меняется.
- CI имеет read-only доступ к исходникам и не получает runtime secrets.
- Delivery публикует artifacts только из commit, прошедшего обязательные gates.
- Один Git commit SHA однозначно соответствует published image tags.
- Runtime deployment, migration execution и rollback остаются отдельной фазой с
  явным environment и human gate.

## API и OpenAPI

Публичный API не меняется. CI проверяет, что generated TypeScript contract
собирается из `api/openapi.yaml` без ручных DTO.

## Данные, миграция и rollback

Schema и данные не меняются. Workflows не подключаются к постоянной базе.
Integration tests используют временные PostgreSQL volumes и удаляют их после
run.

Rollback repository/workflows — обычный revert commit. Published SHA images
immutable и не удаляются автоматически. Runtime rollback будет определён после
выбора deployment target.

## UI

Без изменений.

## Ошибки и негативные сценарии

- invalid GitHub auth блокирует создание repository и push; owner проходит
  повторную authentication, token не передаётся Codex текстом;
- занятое имя `QahNaaRlN/grace` требует owner-confirmed existing repository или
  другое имя;
- CI failure блокирует delivery job и должен сохранять точный failing step;
- GHCR permission failure не маскируется успешной Docker build;
- delivery не запускает migrations и не касается реальных данных;
- fork pull requests не получают write permissions или repository secrets;
- container build должен использовать production `Dockerfile` и
  `web/Dockerfile`, а не development compose service.

## Критерии приёмки

- [ ] Owner принял repository visibility и границу первой CD-фазы.
- [ ] GitHub repository существует под owner account и `origin` указывает на него.
- [ ] Локальный `main` опубликован без force push и соответствует remote `main`.
- [ ] CI workflow имеет минимальные permissions и проходит на реальном GitHub run.
- [ ] CI выполняет `make test`, integration gates и production Docker builds.
- [ ] Delivery не запускается при failed CI и публикует API/Web images только из
      прошедшего commit.
- [ ] GHCR содержит immutable SHA tags; `main` указывает на тот же digest/commit.
- [ ] Ни один runtime secret или `.env` не попал в Git, artifacts или logs.
- [ ] Документация объясняет GitHub Actions, checks, packages и следующий шаг к
      реальному deployment.
- [ ] `make test`, `make test-integration`, `git diff --check` проходят локально.

## Исполняемые проверки

```bash
make test
make test-integration
git diff --check
docker build -f Dockerfile .
docker build -f web/Dockerfile .
gh repo view QahNaaRlN/grace
gh run list --repo QahNaaRlN/grace
git ls-remote origin refs/heads/main
```

После первого delivery run дополнительно проверяются package versions/digests
через GitHub API или `gh api`.

## Открытые решения

1. Branch protection/ruleset для `main`: GitHub Free поддерживает разные
   возможности для public/private repositories; точная настройка проверяется
   после создания repository.
2. Реальный deployment target, domain, TLS, backups и migration gate будут
   определены отдельной спецификацией после выбора hosting.

## Нормативные ссылки

- GitHub Docs: Publishing Docker images —
  https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images
- GitHub Docs: Building and testing Go —
  https://docs.github.com/en/actions/tutorials/build-and-test-code/building-and-testing-go
- GitHub Docs: Building and testing Node.js —
  https://docs.github.com/en/actions/tutorials/build-and-test-code/building-and-testing-nodejs
