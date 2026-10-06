# SPEC-0021 verification

Accepted specification: [0021-explicit-salary-data.md](0021-explicit-salary-data.md).

- AC1/AC2: `FinancialHealth.test.tsx` passes: rent income remains in operations,
  while the salary section has no inferred payroll record.
- AC3: `make test` passes with Go tests, 25 Web tests, lint/build and offline
  Jev 49/72. `git diff --check` passes. CI/Delivery/deployment pending below.
- API, financial records and database schema are unchanged.
