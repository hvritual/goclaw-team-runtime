# Product capability roadmap implementation plan v21

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `21`
- Status: `approved-for-execution`
- Active step: `PCR-S05B Skill import and logical files`
- Task-ID: `PCR-001-S05B-R26`
- Task-Revision: `r026`
- Work-Item: `PCR-S05B`
- Base commit: `27def068f07cb56ef7e58471fbfacb397b11b639`
- Supersedes: `plan_v20.md` for active execution only
- Approved: `2026-08-18`

## Outcome and authority

The Human Customer approved continuous execution through Release 2. Exact base
`27def068` closes PCR-S05A with fresh independent SPEC PASS and CODE QUALITY
PASS and leaves no active task. Three bounded read-only discovery rounds reached
dependency closure for PCR-S05B with no unresolved human-owned choice:

1. the frozen contracts require archive preview/import, version-scoped logical
   files, exact limits, checksums, download, and safe cleanup;
2. the shared client already declares URL import and file editing while the
   Canonical backend rejects both, and read-only legacy evidence supports URL
   and archive inputs but is not safe implementation authority;
3. Space attachment persistence supplies reusable object-store mechanics, but
   the generated AssetService is still an unimplemented scaffold. S05B therefore
   needs a narrow hand-owned Space Skill-object contract instead of System table
   or object-store access.

This version activates only PCR-S05B. It does not authorize Knowledge, push,
merge, deployment, or Release 2 completion. `server/**` remains permanently
read-only and all earlier plan versions remain immutable evidence.

## Frozen S05B behavior

1. System owns immutable version-scoped logical manifests. Each manifest entry
   contains canonical path, Space asset/version references, media type, byte
   size, SHA-256 checksum, and timestamps; it never contains a file body. Space
   exclusively owns bytes, object keys, quarantine, promotion, opening, and
   reconciliation. Cross-module work uses narrow public contracts only.
2. Install `POST /api/skills/import/preview`, `POST /api/skills/import`,
   `GET/POST /api/skills/{id}/files`, and
   `GET/PUT/DELETE /api/skills/{id}/files/{path}`. File path is the URL-encoded
   canonical path remainder. Reads accept an exact `version_id`; mutations
   require `expected_revision` and create a new monotonically numbered draft
   version. No prior version or manifest is rewritten.
3. `GET .../files` returns the selected version's ordered manifest. Exact file
   GET returns strict metadata plus UTF-8 `content` for editable text. The same
   route with `download=true` streams any allowed body with safe attachment
   headers, `nosniff`, a body checksum ETag, and no inline active content.
   `SKILL.md` is mandatory and cannot be deleted.
4. Preview and commit accept either multipart `.skill`/`.zip` field `file` or
   the existing JSON `{url}` source. URL imports are HTTPS-only and limited to
   explicitly recognized GitHub, ClawHub, and Skills.sh source adapters; redirects
   and resolved requests remain within adapter-owned public hosts. Arbitrary URL,
   embedded credentials, non-default ports, loopback/private/link-local targets,
   and cross-provider redirects are rejected. Fetching is injected in tests,
   bounded, cancelable, and never logs source URLs or response bodies.
5. Preview performs the complete validator without durable metadata or object
   writes and returns an expiring, Workspace/actor/source-checksum-bound opaque
   token plus normalized name, description, manifest, total bytes, and warnings.
   Commit reruns the same validator version over the supplied source and fails
   if token, actor, Workspace, expiry, validator version, or complete content
   checksum differs. Preview state contains no body and is safe to expire.
6. Import requires `Idempotency-Key` and a declared `conflict_mode`. Supported
   modes are `new_version` and `replace`; omitted mode is `new_version` for
   client compatibility. A same-name conflict targets the existing Workspace
   Skill and creates a new draft version. `replace` additionally archives the
   previous latest draft in the same transaction, requires exact
   `expected_revision`, and still never mutates or deletes a historical snapshot.
   A key replays only the same canonical request hash; reuse with different
   source, token, mode, target, or revision returns conflict.
7. Canonical paths use UTF-8 NFC, `/` separators, no empty or dot segment, no
   leading slash, drive/UNC prefix, parent traversal, NUL, control character, or
   trailing slash. Duplicate canonical paths are rejected before staging.
   Symbolic links, hard-link aliases, device entries, nested archives, and
   executable/container binary formats are rejected. UTF-8 text/source and safe
   static image/media bodies may be stored; only UTF-8 text is editable inline.
8. Exact frozen limits apply before allocation and while streaming: compressed
   request `10 MiB`, decompressed total `50 MiB`, file count `500`, individual
   file `5 MiB`, path depth `16`, and path length `512` UTF-8 bytes. Declared
   archive sizes are not trusted. Limit, checksum, path, type, and duplicate
   failures leave no manifest, version, audit, binding, idempotency result, temp
   file, quarantine object, or promoted object leak.
9. Space stages accepted bytes under opaque quarantine keys. System commits the
   new version, complete manifest, audit, existing/new Workspace binding, and
   idempotency result in one SQLite transaction only after all stages succeed.
   Post-commit promotion is idempotent; failed/canceled commits roll back staged
   objects. Startup reconciliation promotes referenced quarantine objects and
   removes only proven-unreferenced expired objects; it never guesses from paths.
10. Owner/admin may preview, import, add, replace, and delete files. Members may
    list/read/download published exact versions. Agents may read/download only an
    explicitly bound exact version. Trusted Workspace identity, authentication,
    CSRF, capability installation, and authorization run before source fetching,
    body staging, or repository access. Successful mutations append content-free
    audit actions in the owning transaction.
11. Core strictly parses preview, import, manifest, content, and error responses;
    query keys include Workspace, Skill, and version. Shared Web/Desktop exposes
    archive and recognized-URL preview, explicit conflict handling, versioned
    file tree/editor/download, checksums, progress/cancel, and stale-revision
    recovery only after loaded `skill_import=true`. It never persists preview
    tokens or file bodies outside the active draft/query cache.
12. `/api/config` retains all earlier installed flags and sets
    `skill_import=true` only after the complete installed slice passes.
    `knowledge_query` and `knowledge_review` remain false.

## Writable scope

- `backend/internal/modules/system/**` for manifest/import domain, application,
  persistence, audit, and HTTP adapters;
- `backend/internal/modules/space/**` for the narrow Skill-object contract,
  quarantine/promotion/open/reconciliation, and guarded migrations if needed;
- `backend/internal/modules/workspace/**` only for public Skill visibility and
  capability composition required by import atomicity;
- `backend/internal/bootstrap/**` and `backend/cmd/server/**` for installed
  composition, configuration, limits, and runtime tests;
- `packages/core/**`, `packages/views/skills/**`, Skills locale files,
  `apps/web/**`, and `apps/desktop/**` for strict shared client/UI integration;
- `e2e/**` for installed Web acceptance;
- current roadmap `plan.md`, this immutable `plan_v21.md`, `story-map.md`,
  `task-register.md`, and append-only `journal.md`.

Knowledge code, unrelated generic attachment/UI behavior, generated proto unless
a proven contract gap requires a new approved plan, dependency manifests and
lockfiles, every recorded dirty path, and all `server/**` paths are out of scope.
Protected `packages/ui/components/ui/input.tsx` must remain blob
`a830fd2f0f82770563908d512558fe6ba48f50dd`.

## Ordered execution

### PCR-S05B-R26.1 — Activate exact authority

- Freeze this plan at exact base `27def068` and establish r026 as the sole
  active task.
- Record policy hashes, dirty exclusions, protected Input blob, generated-file
  blob equality, and empty `server/**` range/worktree diffs.

### PCR-S05B-R26.2 — RED safety and transaction contracts

- Add failing validator/archive/URL adapter tests for every path, entry type,
  limit, checksum, redirect, cancellation, and preview-token boundary.
- Add failing Space/System repository and runtime tests for quarantine,
  promotion, manifest immutability, default/replace conflicts, idempotency,
  exact revisions, rollback, restart reconciliation, and retained old versions.
- Add failing HTTP/Core/Views tests for strict routes, identities, permissions,
  CSRF, malformed bodies/responses, version-scoped caches, file UX, loaded flag,
  cancellation, and stale recovery.

### PCR-S05B-R26.3 — GREEN installed vertical

- Implement the narrow Space object lifecycle, System manifests/import service,
  safe validator/source adapters, atomic composition, audit, HTTP handlers, and
  installed capability declaration required by RED contracts.
- Replace Core's explicit S05B rejection with strict APIs and activate the shared
  archive/URL preview and versioned file-management UI without changing unrelated
  surfaces.

### PCR-S05B-R26.4 — Verify and close

- Run focused adversarial/transaction/reopen/reconciliation tests, root
  typecheck/test, backend check, official race, and production Web build.
- Run a fresh-database installed-Chrome journey proving preview, default
  new-version conflict, explicit replacement confirmation, file add/replace/
  delete/download/checksum, old-version retention, rejection with no leaks,
  restart persistence, member/agent read bounds, and later false flags.
- Verify exact scope, policy hashes, dirty exclusions, process cleanup, trailers,
  and empty `server/**`; obtain fresh independent SPEC and code-quality review.
  Only a complete PASS may close r026 and PCR-S05B.

## Acceptance criteria

1. Installed owner/admin archive and recognized-URL preview/import plus logical
   file management work through HTTP and shared Web/Desktop with exact revision,
   idempotency, immutable version, and authorization behavior.
2. Every frozen adversarial archive/path/type/size/checksum/source case fails
   before durable publication and leaves zero partial metadata, body, binding,
   audit, idempotency, temp, or quarantine leak.
3. Old published/bound versions retain exact manifests and downloads after
   later imports or file mutations; member and Agent reads remain exact-scoped.
4. `skill_import=true` is reported only for the installed S05B slice;
   both Knowledge flags remain false.
5. Full deterministic, race, production-build, installed-browser, scope/process,
   traceability, and fresh independent-review gates pass without waiver.

## Risks and rollback

Primary risks are archive bombs and path escape, SSRF, mutable historical files,
partial cross-module publication, orphaned objects, idempotency drift, content or
URL leakage, and frontend editing the wrong version. Streaming caps, recognized
source adapters, one validator, opaque Space keys, quarantine plus reconciliation,
immutable complete manifests, canonical request hashes, exact revisions, strict
schemas, and version-aware query keys address them. Rollback disables S05B routes
and `skill_import` while retaining manifests and referenced Space objects. Down
migrations may run only after proving no retained preview/idempotency/manifest or
referenced object data would be discarded. Rollback never touches S05A history,
Knowledge, Release 1 state, user dirty files, or `server/**`.
