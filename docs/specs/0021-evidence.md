# SPEC-0021 verification

Accepted specification: [0021-explicit-salary-data.md](0021-explicit-salary-data.md).

- AC1/AC2: `FinancialHealth.test.tsx` passes: rent income remains in operations,
  while the salary section has no inferred payroll record.
- AC3: `make test` passes with Go tests, 25 Web tests, lint/build and offline
  Jev 49/72. `git diff --check` passes. CI/Delivery/deployment pending below.
- API, financial records and database schema are unchanged.


## Published and deployed

- PR 9: https://github.com/QahNaaRlN/grace/pull/9 — merged, release 11f6b2e270c3f976f275331aeb871412d15c8d85.
- PR CI: https://github.com/QahNaaRlN/grace/actions/runs/37482205090 — success.
- Main CI: https://github.com/QahNaaRlN/grace/actions/runs/37483016959 — success, including PostgreSQL and container gates.
- Delivery: https://github.com/QahNaaRlN/grace/actions/runs/37483507443 — success.
- Deploy: https://github.com/QahNaaRlN/grace/actions/runs/37484033796 — success.
- Post-merge make test: Go, 25 Web tests, lint/build, offline Jev 49/72 — passed.
- Public entrypoint: HTTP 200; live JS /assets/index-GdzvYdzi.js matches final verified build; financial API anonymous 401; no Basic Auth challenge.
- No production financial records were created or changed by this UI correction.
