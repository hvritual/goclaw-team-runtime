---
name: refactor-goclaw-ddd
description: Refactor, review, or plan the goclaw/Multica Go backend with its project-specific Domain-Driven Design boundaries and repository constraints. Use when Codex changes backend architecture, extracts a goclaw bounded context, moves code from server/internal into server/modules, designs ports and adapters, or applies DDD lint enforcement while preserving workspace isolation, Chi/sqlc/pgx behavior, SQLite-local parity, events, authorization, and migration rules.
---

# Refactor goclaw DDD

Apply goclaw-specific business and engineering constraints on top of the standard Go DDD workflow. Keep project behavior and repository policy authoritative over generic architecture preferences.

## Load the standard foundation

Resolve the installed `refactor-go-ddd` skill from the available skills catalog. Read its complete `SKILL.md` and every standard reference it routes to for the requested task.

If `refactor-go-ddd` is unavailable, stop before proposing or implementing a DDD refactor and report the missing prerequisite. Do not reconstruct or duplicate the generic workflow inside this project skill.

## Load project authority

Before planning or editing:

1. Read root `AGENTS.md` and `CLAUDE.md` completely.
2. Read `references/goclaw-architecture.md` completely.
3. Read only the accepted plans, architecture documents, affected code, migrations, generated-code configuration, and tests relevant to the selected slice.
4. Reconcile this skill with newer repository instructions. Repository instructions win on conflict.

## Verify the repository

Apply this skill only when the backend module path is `github.com/multica-ai/multica/server` and repository instructions identify the goclaw/Multica backend. If either signal is missing, use only the standard `refactor-go-ddd` skill.

## Execute one verified slice

Follow the standard skill's inventory, context discovery, tracer-slice, inward dependency, lint, and verification workflow. Add these project requirements:

- Preserve the public API, authorization, workspace boundary, database semantics, realtime behavior, and SQLite-local behavior unless the user explicitly changes them.
- Treat the six-domain baseline as a starting context map, not permission for a repository-wide move.
- Keep generated and technology-specific types outside domain and application packages.
- Introduce `server/modules/<context>` incrementally and remove each superseded path after callers migrate.
- Add depguard enforcement with the first new module because `modules/` has no Go `internal/` visibility protection.
- Keep concrete construction in the existing composition roots.
- Preserve unrelated worktree changes and avoid broad cleanup.

## Verify and report

Run focused Go tests first, then generation, lint, integration, and broader repository checks in proportion to risk. Never execute real-agent smoke tests without explicit authorization.

Report:

- bounded context and use case;
- goclaw invariants preserved;
- legacy and target paths;
- ports, adapters, composition roots, and generated-code boundaries;
- tests, generation, and lint commands actually run;
- remaining legacy paths, overlapping changes, and decisions still needed.

## Stop conditions

Stop before changing code when ownership, last-owner safety, workspace isolation, cross-context consistency, API compatibility, PostgreSQL/SQLite parity, event ordering, or transactional cleanup cannot be established from repository evidence. Stop before destructive schema work, production operations, real-agent execution, or overwriting overlapping user changes.

## Resources

- `references/goclaw-architecture.md`: goclaw business contexts, technology constraints, migration rules, boundary rules, and verification requirements.
- Installed `refactor-go-ddd`: generic DDD workflow, architecture model, lint policy, and depguard generator.
