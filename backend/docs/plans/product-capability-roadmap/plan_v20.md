# Product capability roadmap implementation plan v20

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `20`
- Status: `approved-for-execution`
- Active step: `PCR-S05A Skill catalog and version lifecycle`
- Task-ID: `PCR-001-S05A-R25`
- Task-Revision: `r025`
- Work-Item: `PCR-S05A`
- Base commit: `0aed36871271184325b6147841348847b004a7a4`
- Supersedes: `plan_v19.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer approved continued execution through Release 2. Exact base
`0aed3687` closes Release 1 with independent review and leaves no active task.
Three bounded read-only discovery rounds reached dependency closure for
PCR-S05A with no unresolved human-owned choice. The explicit S05A/S05B story
split resolves file ownership: S05A installs the metadata catalog and version
lifecycle without Skill file bodies or archive import; S05B remains the sole
owner of Space-backed file objects, manifests, preview, import, and download.

This version activates only PCR-S05A. It does not authorize PCR-S05B, either
Knowledge story, push, merge, deployment, or Release 2 completion before all
four Release 2 stories close. `server/**` remains permanently read-only, and
all earlier plan versions remain immutable evidence.

## Frozen S05A behavior

1. System owns stable Skill definitions, immutable numbered version snapshots,
   lifecycle state, provenance, and audit references. Workspace owns visibility
   and Agent bindings and may reference System only through a public Skill
   reference contract. Neither module reads the other's tables.
2. Install `GET/POST /api/skills`, `GET/PUT/DELETE /api/skills/{id}`, and
   `POST /api/skills/{id}/restore`. Install lifecycle commands below the same
   Skill route family for publishing and deprecating a named version. `PUT`
   creates a new draft version and never edits an existing snapshot.
3. Manual create accepts a trimmed non-empty `name`, optional `description`,
   and JSON `config`; it creates definition plus draft version `1` and a
   Workspace visibility binding. S05A rejects `files`, file bodies, URLs, and
   import inputs as explicitly unavailable instead of silently discarding them.
4. A successful version edit copies omitted fields from the selected latest
   version, applies supplied metadata, and creates the next monotonic positive
   version. A published version's name, description, configuration, creator,
   timestamp, and provenance never change. Concurrent creates serialize and
   cannot reuse a version number.
5. Version state is `draft -> published -> deprecated -> archived`. Publish and
   deprecate require the exact version ID and expected definition revision;
   stale requests return the canonical 409 revision conflict without mutation.
   Publishing a new version does not rewrite a Workspace binding pinned to an
   older version. Referenced versions remain readable after deprecation or
   definition archive.
6. Definition archive hides the Skill from normal list/create-edit surfaces but
   retains every version, binding, provenance, audit, and referenced-version
   read. Restore re-enables the definition without altering its versions or
   bindings. Permanent deletion is not part of S05A.
7. Trusted Workspace identity, authentication, CSRF, and capability-specific
   authorization run before repository access. Owner/admin may create, version,
   publish, deprecate, archive, and restore. Members may read published Skills;
   agents are denied unless an explicit binding authorizes the exact version.
   Missing capability installation or action mapping denies access.
8. Every successful mutation writes immutable governance audit evidence in the
   same database transaction as its owning domain state. No file content,
   configuration body, credential, URL, or sensitive provenance value appears
   in audit or operational output.
9. Core parses every installed Skill response with strict Zod schemas and tests
   malformed responses. Query keys remain Workspace-scoped. Shared Web/Desktop
   list/detail/create/version/lifecycle affordances require loaded explicit
   `skill_administration=true`; read-only published content remains available
   to authorized members. Existing creator-based edit permission is removed.
10. `/api/config` retains all Release 1 flags and sets only
    `skill_administration=true` after the complete installed slice passes.
    `skill_import`, `knowledge_query`, and `knowledge_review` remain false.

## Writable scope

- `backend/internal/modules/system/**` for hand-owned Skill domain, application,
  SQLite persistence/migration, public contracts, HTTP installation, and tests;
- `backend/api/system/v1/skill.proto` and generated
  `backend/rpc/pb/system/v1/skill*.go` only when required for the public local/
  gRPC contract; generated files may be changed only by the repository generator;
- `backend/internal/modules/workspace/contract/**`, Skill-binding application/
  persistence paths, and direct tests only for the public visibility/reference
  boundary and binding retention; no other Workspace capability may change;
- `backend/internal/bootstrap/**` for System SQLite composition, trusted HTTP
  identity/authorization wiring, feature installation, audit wiring, and
  installed runtime acceptance;
- `packages/core/api/**`, `packages/core/types/skill.ts`,
  `packages/core/workspace/**`, `packages/core/permissions/**`, and direct tests
  for strict version/provenance contracts and Workspace-scoped caching;
- `packages/views/skills/**`, the direct Skills navigation-gating surface, and
  direct tests for installed lifecycle behavior and explicit permissions;
- `e2e/**` for installed Web acceptance;
- current roadmap `plan.md`, `plan_v20.md`, `story-map.md`, `task-register.md`,
  and append-only `journal.md`.

S05B file/import code, Space Asset implementation, Knowledge code, lockfiles
unless a direct dependency classification changes, unrelated packages, every
pre-existing dirty path, and all `server/**` paths are out of scope. In
particular, protected `packages/ui/components/ui/input.tsx` must remain blob
`a830fd2f0f82770563908d512558fe6ba48f50dd`; existing generated protobuf status
noise has index-equal blobs and must not be mistaken for user content changes.

## Ordered execution

### PCR-S05A-R25.1 — Activate exact authority

- Freeze this plan at exact base `0aed3687` and establish r025 as the sole
  active task.
- Record policy hashes, dirty exclusions, protected Input blob, generated-file
  blob equality, and empty `server/**` range/worktree diffs.

### PCR-S05A-R25.2 — RED contracts

- Add failing migration/repository/application tests for definition/version
  creation, immutable snapshots, concurrent numbering, lifecycle revisions,
  archive/restore, provenance, audit atomicity, binding retention, cancellation,
  failure rollback, and reopen.
- Add failing HTTP/runtime tests for route shape, trusted identity, CSRF,
  owner/admin/member/agent authorization, exact conflicts, explicit S05B
  rejection, installed feature flags, and referenced-version reads.
- Add failing Core/Views tests for strict parsing, version-aware queries,
  malformed responses, admin-only mutations, loaded explicit feature gating,
  and lifecycle interaction.

### PCR-S05A-R25.3 — GREEN installed vertical

- Add the System-owned migration, repository, use case, public reference
  adapter, Workspace visibility composition, HTTP lifecycle, governance audit,
  and installed capability declaration required by the RED contracts.
- Tighten Core Skill schemas and update the shared Skills surface without
  implementing file bodies, imports, Knowledge, or unrelated UI behavior.

### PCR-S05A-R25.4 — Verify and close

- Run focused backend/Core/Views tests, migration/reopen/rollback/concurrency
  checks, root typecheck/test, backend check, and the official real race suite.
- Run a fresh-database installed-Chrome journey proving admin create/version/
  publish/archive/restore, member published read and mutation denial, pinned
  historical-version reads, restart persistence, and explicit false later flags.
- Verify exact scope, policy hashes, excluded dirty blobs, generated provenance,
  and empty `server/**`; then obtain fresh independent read-only review. Only a
  complete PASS may close r025 and PCR-S05A.

## Acceptance criteria

1. Installed owner/admin lifecycle works through HTTP and shared Web/Desktop;
   member/agent denials, trusted identity, CSRF, and missing-provider fail-closed
   behavior are deterministic and cause no mutation.
2. Version snapshots, provenance, revision conflicts, concurrent numbering,
   archive/restore, binding retention, referenced reads, transaction rollback,
   cancellation, and database reopen satisfy the frozen behavior.
3. Core rejects malformed Skill/version/provenance responses; queries are
   Workspace-scoped; UI mutation affordances require loaded explicit installation
   and server-authoritative admin permission.
4. `skill_administration=true` is reported only for the installed S05A slice;
   S05B and both Knowledge flags remain false.
5. Full deterministic, production-build, installed-browser, scope/process,
   traceability, and fresh independent-review gates pass without waiver.

## Risks and rollback

The primary risks are accidental mutable published content, a global Skill
leaking across Workspaces, partial definition/binding creation, lost concurrent
version numbers, binding drift on publish/archive, and frontend assumptions from
the legacy single-object shape. System-owned transactions, public cross-module
contracts, exact revisions, unique version numbers, pinned binding tests, strict
schemas, and explicit flags address those risks. Rollback disables the S05A
routes and flag while retaining catalog, versions, audit, and bindings. A down
migration may run only when it proves no retained Skill/version/audit/binding
data would be discarded. It must never touch S05B, Knowledge, Release 1 state,
user dirty files, or `server/**`.
