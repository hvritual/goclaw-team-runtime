# Repository Guidelines

This file provides guidance to AI agents when working with code in this
repository.

> **Single source of truth:** This file is a concise pointer document.
> Authoritative architecture, coding rules, and conventions live in
> [CLAUDE.md](CLAUDE.md). Backend work must additionally obey
> [backend/AGENTS.md](backend/AGENTS.md). A nested policy may narrow these rules
> but cannot grant an exception to the `server/**` boundary below.

## Canonical Development Baseline

- `codex/multica-six-domain-baseline` is the integration base for continuing
  work. New pull requests target this branch unless an explicit repository
  migration decision establishes a successor.
- `backend/**` is the only writable backend implementation root.
- `server/**` is permanently read-only migration evidence. No implementation
  plan, task, review, compatibility requirement, test request, or generated-code
  workflow may authorize a `server/**` modification.
- When required behavior exists only in `server/**`, inspect it and port the
  behavior into `backend/**` under a versioned plan with tests. Never patch,
  synchronize, mirror, refactor, extend, generate into, or delete from the
  legacy tree.
- Every candidate diff containing any `server/**` path is invalid and must be
  blocked before deterministic checks, review, merge, or DoneGate.
- Legacy root-level backend trees such as `teamcontrol/`, `gateway/`, and
  `workstation/` are migration inputs only. Port required behavior into the
  canonical backend; do not merge those trees into the baseline unchanged.
- `main` and historical `agent/*` branches are migration inputs and audit
  history, not implementation bases.

## Quick Reference

### Architecture

Go backend plus a pnpm/Turborepo frontend monorepo with shared packages.

- `backend/` — canonical GoClaw/Team Runtime backend and versioned-plan root.
- `server/` — permanently read-only legacy Multica backend migration evidence.
- `apps/web/` — Next.js frontend using the App Router.
- `apps/desktop/` — Electron desktop app.
- `packages/core/` — headless business logic, React Query hooks, API clients,
  and client-owned Zustand state.
- `packages/ui/` — atomic shadcn/Base UI components with no business logic.
- `packages/views/` — shared business pages and components.
- `packages/tsconfig/` — shared TypeScript configuration.

### State Management

- React Query owns server state such as issues, members, agents, inbox, and
  workspaces.
- Zustand owns client/view state such as filters, drafts, modals, and desktop
  tabs; current workspace identity remains route-driven.
- Shared Zustand stores live in `packages/core/`, never in
  `packages/views/` or app directories.
- Realtime events update React Query for server data. They may clear
  client-owned pointers only through one responder with a self-event guard.

### Package Boundaries

- `packages/core/`: no `react-dom`, direct `localStorage`, or
  `process.env`.
- `packages/ui/`: no `@multica/core` imports or business logic.
- `packages/views/`: no `next/*`, `react-router-dom`, or stores; use the
  navigation adapter.
- `apps/web/platform/`: the only place for Next.js platform APIs.

### Database Migrations

- Do not add database foreign keys or cascading actions. Enforce relationships
  and dependent cleanup in application transactions.
- Every created index, including unique indexes, must use
  `CREATE [UNIQUE] INDEX CONCURRENTLY` in its own single-statement migration.

### Commands

```bash
make dev
pnpm typecheck
pnpm test
make test
make check
cd backend && make check && make test-race
```

Use repository scripts as the command source of truth. See
[CLAUDE.md](CLAUDE.md) for the authoritative rules.
