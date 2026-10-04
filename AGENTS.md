# Grace development guide

Read `docs/vision.md` and `docs/architecture.md` before changing product or
architecture behavior.

- Keep Grace usable without an LLM. Financial state and arithmetic belong to
  deterministic Go modules backed by PostgreSQL.
- Group backend code by business capability under `internal/`; do not introduce
  global controller/service/repository layers without a demonstrated need.
- Store and transport money in integer minor units with an explicit currency.
- Treat `api/openapi.yaml` as the client contract. Regenerate TypeScript types
  after every contract change; do not duplicate API DTOs manually in `web/`.
- Keep web business rules out of React. The web client owns presentation,
  interaction, forms, and safe client-side state.
- AI-originated writes require an explicit confirmation policy before they are
  implemented. The assistant module must depend on the financial core, never
  the reverse.

Before handing off a change, run:

    make test
    git diff --check
