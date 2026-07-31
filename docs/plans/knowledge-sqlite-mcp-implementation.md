# Knowledge SQLite MCP implementation plan

## Outcome

Add workspace-scoped knowledge to Multica without changing the six core domains.
Knowledge is accumulated from project delivery evidence, governed through
candidate review, stored through replaceable adapters, and exposed through the
Multica UI and a remote MCP endpoint.

SQLite is the first durable knowledge adapter. The knowledge domain, application
service, HTTP API, UI, and MCP transport must not depend on SQLite or PostgreSQL
query types.

## Implementation status

All delivery stages are implemented on the feature branch. Verification and
commit evidence are recorded in the final handoff report. Deferred evidence
producers and the absence of a standalone operations CLI are documented in
[`docs/knowledge.md`](../knowledge.md).

## Confirmed boundaries

- Keep Workspace, Member, Project, Issue, Task, and Skill as the six core domains.
- Treat Knowledge as a workspace-level capability with an optional Project link.
- Use `Evidence -> Candidate -> Published Knowledge`.
- Reuse the existing `owner`, `admin`, and `member` authorization model.
- Do not restore runtime agent code or expose permission explanations to members.
- Do not add database foreign keys or cascade actions.
- Use a transactional source outbox and idempotent delivery across the primary
  store and knowledge SQLite store.
- MCP may read, search, and propose. It may not approve or reject.

## Confirmed test seams

Tests observe behavior through these public boundaries:

1. Knowledge application service.
2. Repository and search adapter contracts shared by memory and SQLite.
3. Workspace HTTP API.
4. MCP tools, resources, transport, and authorization.
5. Shared web/desktop knowledge views.

Tests must not assert private methods or inspect adapter tables when the same
behavior can be observed through a public port.

## Delivery stages

### 0. Baseline and documentation

- Confirm branch, worktree, six-domain implementation, authorization vocabulary,
  SQLite driver, local server wiring, and absence of overlapping changes.
- Preserve the accepted handoff in the repository.

Acceptance:

- Worktree is clean before implementation.
- Existing user changes are identified and preserved.
- The plan and handoff record all confirmed boundaries.

### 1. Domain, application service, and memory adapter

- Add evidence, candidate, entry, revision, source, and review concepts.
- Add promotion policy, repository, search index, artifact store, source reader,
  and unit-of-work ports where used.
- Implement proposal, ingestion, review, quarantine, supersede, provenance, and
  optimistic revision behavior.
- Add an in-memory adapter and public contract tests.

Acceptance:

- Domain tests cover governed kinds, idempotent evidence, human review, stale
  revisions, immutable history, and workspace isolation.
- The package imports no SQL driver.

### 2. SQLite adapter

- Add explicit schema versions and ordered adapter-local migrations.
- Use `database/sql` and the existing `modernc.org/sqlite` driver.
- Configure WAL, busy timeout, controlled connections, and protected files.
- Add replaceable search, FTS5 capability detection, rebuild, backup, and restore.

Acceptance:

- Memory and SQLite pass the same repository/search contracts.
- Restart, migration, lock, permissions, backup, and rebuild tests pass.
- The schema has no foreign keys or cascade actions.

### 3. Project evidence outbox

- Define a normalized evidence envelope.
- Append evidence to a source transaction outbox for relevant project events.
- Dispatch at least once and ingest idempotently into the knowledge store.
- Expose backlog, retry, and last-success health.

Acceptance:

- Source rollback creates no evidence.
- Knowledge-store failure does not roll back the source operation.
- Restart and replay do not duplicate evidence or candidates.

### 4. Application API and authorization

- Add workspace-scoped list, search, get, propose, candidate review, quarantine,
  source, and revision endpoints.
- Validate workspace membership and Project ownership in the canonical service.
- Reuse existing role capabilities and record review audit data.

Acceptance:

- Cross-workspace access is rejected.
- Members can read published knowledge and propose when allowed.
- Members cannot inspect candidates, permission matrices, or permission guidance.
- Stale review revisions return a conflict.

### 5. Shared web and desktop UI

- Add typed API schemas and React Query hooks in `packages/core`.
- Add browse, details, proposal, and review views in `packages/views`.
- Wire `/{slug}/knowledge` in web and desktop.

Acceptance:

- Web and desktop share business views.
- Loading, empty, error, malformed response, and authorization states are tested.
- Management actions are absent for ordinary members.

### 6. Remote MCP

- Add a Streamable HTTP endpoint for workspace knowledge.
- Add OAuth protected-resource metadata integration and PAT compatibility.
- Add `knowledge_search`, `knowledge_list`, `knowledge_get`, and
  `knowledge_propose`.
- Add knowledge resources and resource templates.

Acceptance:

- Origin, workspace, application permission, and token scopes are all enforced.
- MCP cannot approve, reject, manage roles, or access runtime agent behavior.
- Protocol initialization, cancellation, pagination, and errors are tested.

### 7. Operations, review, and commits

- Document configuration, schema upgrades, backup/restore, index rebuild, outbox
  recovery, MCP setup, scopes, and known limitations.
- Run narrow checks throughout and the full relevant suite at the end.
- Review changes against repository standards and this plan.
- Commit atomic, conventional changes on the current branch.

Acceptance:

- Verification results are reported accurately.
- No unrelated cleanup or compatibility layer is included.
- Final commits contain no runtime agent reintroduction.

## Stop conditions

Stop and report if implementation would require production writes, credentials,
destructive migration, a new role system, a CGO SQLite driver, restoration of
runtime agents, or overwriting overlapping user changes.
