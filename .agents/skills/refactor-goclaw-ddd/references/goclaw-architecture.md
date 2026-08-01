# goclaw DDD Architecture Profile

Keep this reference project-specific. Apply it only after loading the installed standard `refactor-go-ddd` skill.

## Contents

- Sources of truth and context candidates
- Target layout and technology invariants
- Data, migration, and boundary rules
- Migration order, lint generation, and verification

## Sources of truth

Read before changing backend code:

1. Root `AGENTS.md` and `CLAUDE.md`.
2. Relevant accepted plans or architecture documents.
3. `server/go.mod`, `server/sqlc.yaml`, `server/cmd/server`, affected handlers, services, queries, migrations, and tests.
4. `docs/six-domain-baseline.md` and its machine-readable baseline when the change touches a core domain.
5. Built-in skill sources when CLI flags, API fields, or documented product behavior changes.

Do not use this profile to override newer repository instructions.

## Business context candidates

The accepted first-stage product baseline names six domains:

- Workspace: tenant, authorization, slug, issue prefix, and lifecycle.
- Member: roles, invitations, membership, and last-owner invariants.
- Project: project lifecycle, lead, resources, and issue association.
- Issue: collaborative work object, hierarchy, assignment, properties, labels, and comments.
- Task: agent execution lifecycle, lease, messages, usage, cancellation, and retry.
- Skill: skill content, files, agent attachment, discovery, bundle resolution, and caching.

Treat these as bounded-context hypotheses, not permission for a six-way big-bang move. Preserve the distinction between Issue and execution Task. Keep Workspace as the tenancy and authorization boundary.

Knowledge is a workspace capability with an optional Project link and already demonstrates replaceable repository/search adapters. Project requirements also demonstrates application-facing repository contracts with SQLite adapters. Study these packages as local patterns without copying their layout mechanically.

## Target layout

Place newly migrated backend slices under the Go module root:

```text
server/
  cmd/server/
  modules/<context>/
    domain/
    application/
    dependency/
    interfaces/
  modules/shared/kernel/
  modules/platform/
```

Migrate one use case at a time from `server/internal/...`. Do not move packages solely to match the target tree. Remove each old path after its callers and tests move; do not leave permanent forwarding packages.

Because `modules/` removes Go's `internal/` visibility protection, install and enforce depguard rules in the same change that introduces the first module.

## Technology invariants

- Keep Chi for HTTP routing.
- Keep sqlc and pgx/PostgreSQL for the default server.
- Do not replace them with GORM or MySQL.
- Keep sqlc-generated types in dependency adapters or mapping boundaries; do not make them domain entities.
- Keep Redis, storage providers, email clients, analytics, realtime, and other integrations behind application/domain ports when a migrated use case consumes them.
- Keep concrete construction in `server/cmd/server`.
- Preserve the separate SQLite-local composition where affected; do not assume the PostgreSQL handler graph is shared.

## Data and migration invariants

- Filter workspace-scoped queries by `workspace_id`.
- Preserve membership authorization and `X-Workspace-ID` selection semantics.
- Never add database foreign keys, cascading deletes, or cascading updates.
- Validate relationships and perform dependent cleanup explicitly in application code.
- Use an application transaction when related writes must commit atomically.
- Create every index with `CREATE INDEX CONCURRENTLY` or `CREATE UNIQUE INDEX CONCURRENTLY`.
- Put each concurrent index build in its own single-statement migration file.
- Regenerate sqlc code after query changes and do not hand-edit generated files.

## Boundary rules

- Resolve human-readable or UUID resource path parameters through the established handler loaders before writes.
- Parse raw UUID inputs at request boundaries and return immediately on failure.
- Keep trusted sqlc UUID round-trips distinct from untrusted inputs.
- Preserve API error status, JSON shape, websocket events, cache behavior, and self-event guards.
- Keep domain errors stable and map pgx/sqlc errors inside dependency adapters or interface error mappers.
- Do not move authentication or workspace middleware decisions into domain entities.
- Preserve application-owned cleanup because the database does not enforce relationships.

## Suggested migration order

For a chosen use case:

1. Characterize the current Chi route, middleware, handler, sqlc calls, events, and tests.
2. Define the domain behavior and stable errors.
3. Define an application command/query plus repository and event ports.
4. Adapt existing sqlc queries in `modules/<context>/dependency/postgres`.
5. Add a thin Chi adapter in `modules/<context>/interfaces/http`.
6. Wire it from `server/cmd/server` without changing the public route.
7. Preserve or add the equivalent SQLite-local seam when the behavior exists there.
8. Run focused Go tests, sqlc generation when applicable, lint, and then broader Go checks.
9. Delete the superseded handler/service path only after all callers migrate.

## Lint generation

Resolve the installed `refactor-go-ddd` skill directory from the skills catalog and run its generator with the Go module root:

```bash
go run <refactor-go-ddd-skill-dir>/scripts/generate_depguard.go \
  -root server \
  -modules-dir modules
```

If the repository still has no governing golangci-lint configuration, introduce the pinned tool version, configuration, local command, and CI gate together with the first new module. Apply full lint to `server/modules/...` and use an incremental baseline for untouched legacy packages until separately cleaned.

## Verification

Use repository commands as the source of truth. Typical checks include:

```bash
make sqlc
make test
make check
```

Run focused `go test` commands from `server/` while iterating. Do not run real-agent smoke tests unless explicitly authorized because they can access authenticated accounts and consume quota.

Report skipped broad verification and preserve unrelated worktree changes.
