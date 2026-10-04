# SPEC-0011 evidence: GitHub repository и CI/CD delivery

- Specification: `docs/specs/0011-github-repository-and-delivery.md`
- Accepted specification commit: `f47b5c3`
- Implementation commits: `2404133`, `42e34d3`
- Pull request: https://github.com/QahNaaRlN/grace/pull/1
- Integration commit: `c7a3f1da19a1032d67b953f08da65d469c8c76ee`
- Repository: https://github.com/QahNaaRlN/grace (private)

## Acceptance criteria and executable evidence

| Criterion | Evidence | Result |
| --- | --- | --- |
| Owner visibility and CD boundary | Owner accepted private `QahNaaRlN/grace` and GHCR artifact delivery; acceptance record is in SPEC-0011. | PASS |
| Repository and origin | `gh repo view QahNaaRlN/grace`; `git remote get-url origin` returns `https://github.com/QahNaaRlN/grace.git`. | PASS |
| Main published without history rewrite | Initial `git push -u origin main`, PR merge commit `c7a3f1d`, then `git pull --ff-only`; no force push used. | PASS |
| Real GitHub CI | PR run `37239030359` and main push run `37239305420` completed successfully. | PASS |
| Complete CI gates | Main run jobs `Quality gates`, `PostgreSQL integration gates`, `Build api image`, and `Build web image` all completed successfully. | PASS |
| Delivery guarded by CI | Delivery is triggered by completed `CI`; publish job requires successful push run for `main`. Manual dispatch independently checks a successful CI push run for the same SHA. | PASS |
| GHCR API image | Delivery run `37239514514` published `ghcr.io/qahnaarln/grace-api:sha-c7a3f1da19a1032d67b953f08da65d469c8c76ee` and `:main` as OCI digest `sha256:b6de90fa5a9d6c8f98e3e27b1ed853af5774f3a8407574e6af9d24c0236ece59`. | PASS |
| GHCR Web image | Delivery run `37239514514` published `ghcr.io/qahnaarln/grace-web:sha-c7a3f1da19a1032d67b953f08da65d469c8c76ee` and `:main` as OCI digest `sha256:08351b6922bea181a0088475b855e8c0582ac21eb46872b21a86818f390c6b54`. | PASS |
| Secrets boundary | CI permissions are `contents: read`; Delivery grants only `actions: read`, `contents: read`, and `packages: write`. Repository secret scan found no private keys, GitHub tokens, or AWS access keys. Runtime `.env` files were not committed. | PASS |
| Owner documentation | `docs/github-delivery.md` documents Actions, GHCR identity, permissions, and the boundary before production deployment. | PASS |
| Local verification | Commands below completed successfully on 2026-10-05. | PASS |

## Local verification transcript

```bash
make test
make test-integration
git diff --check
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 \
  .github/workflows/ci.yml .github/workflows/delivery.yml
docker build -f Dockerfile -t grace-api:spec0011 .
docker build -f web/Dockerfile -t grace-web:spec0011 .
```

The first Web image build exposed an existing Dockerfile defect: Vite wrote to
`/app/web/dist`, while the runtime stage copied `/app/dist`. Commit `2404133`
corrected the path, after which both the local build and GitHub container job
passed.

## GitHub-hosted evidence

```bash
gh pr checks 1
gh run view 37239030359
gh run view 37239305420
gh run view 37239514514
gh run view 37239514514 --job 111545218329 --log
gh run view 37239514514 --job 111545218369 --log
```

Direct package-version listing with the local GitHub CLI token returned HTTP
403 because that token lacks `read:packages`. This does not affect publication:
both Delivery publish jobs succeeded, and their Buildx outputs record the exact
image names, tags, and registry digests above.

## Branch protection limitation

The attempted `PUT /repos/QahNaaRlN/grace/branches/main/protection` returned
HTTP 403: GitHub requires Pro for branch protection on this private repository,
or public visibility. The repository remains private. CI checks are active on
pull requests, but GitHub cannot enforce their non-bypassability on the current
plan.
