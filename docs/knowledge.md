# Knowledge capability operations

## Scope

Knowledge is a workspace capability layered beside the six core Multica
domains. Workspace, Member, Project, Issue, Task, and Skill remain usable when
knowledge is disabled or its durable store cannot open.

The module follows this lifecycle:

`Project delivery event -> source outbox -> evidence -> candidate -> review -> published knowledge`

Goal, decision, constraint, requirement, procedure, and lesson evidence requires
human review. A deterministic, validated, terminal reference may publish
automatically. Conflicting, low-confidence, or untraceable evidence is
quarantined.

## Storage and portability

The domain and application service depend on `Repository` and `SearchIndex`
ports. The first durable adapter uses `database/sql` with
`modernc.org/sqlite`; the in-memory adapter passes the same contract suite.
Future database adapters can implement these ports without changing the HTTP,
UI, evidence, or MCP layers.

Knowledge uses a database separate from the six-domain source database. It
stores canonical workspace and Project IDs as values and intentionally has no
foreign keys or cascade actions.

SQLite schema version: `2`.

SQLite startup settings:

- WAL journal mode
- normal synchronous mode
- 5-second busy timeout
- foreign keys disabled
- one controlled writer connection
- private database directory (`0700`) and file (`0600`)
- FTS5 capability detection and a rebuildable index, with a portable search
  fallback when FTS5 is unavailable

Migrations are ordered, transactional, and recorded in
`knowledge_schema_version`. A database with a newer schema is rejected without
modification.

## Configuration

The local SQLite server and default PostgreSQL service recognize:

| Variable | Purpose | Default |
| --- | --- | --- |
| `MULTICA_KNOWLEDGE_ENABLED` | Set to `false` to disable knowledge only | enabled |
| `MULTICA_KNOWLEDGE_SQLITE_PATH` | Knowledge SQLite file | local: sibling `*.knowledge.db`; default service: `data/multica-knowledge.db` |
| `MULTICA_PUBLIC_URL` | Canonical base URL used by MCP protected-resource metadata | inferred only for loopback requests |
| `MULTICA_MCP_AUTHORIZATION_SERVERS` | Comma-separated OAuth authorization-server issuer URLs | empty |

When knowledge is configured but unavailable, `/health` and all six domains
continue to operate. Knowledge endpoints return `503`, while an owner or admin
can inspect `/api/knowledge/health`, which reports `enabled: true` and
`available: false`.

## HTTP API and authorization

Published knowledge is available to workspace members:

- `GET /api/knowledge`
- `GET /api/knowledge/search`
- `GET /api/knowledge/{id}`
- `GET /api/knowledge/{id}/revisions`
- `GET /api/knowledge/{id}/sources`
- `POST /api/knowledge/proposals`

Owners, admins, and Project leads use the governed candidate endpoints. Owners
and admins see every workspace candidate; a Project lead sees and reviews only
candidates linked to Projects they lead:

- `GET /api/knowledge/candidates`
- `POST /api/knowledge/candidates/{id}/review`

Only owners and admins use `GET /api/knowledge/health`.

Every request resolves canonical workspace membership. Optional Project IDs
must belong to the active workspace. Ordinary members are not shown candidate
queues, permission descriptions, or governance configuration.

## Evidence outbox and recovery

Project, Issue, and terminal Task mutations append normalized evidence inside
the source transaction. The SQLite-local composition uses its primary SQLite
database; the default service uses the PostgreSQL
`knowledge_evidence_outbox`. Delivery to the separate knowledge SQLite adapter
is at least once and idempotent. A failed knowledge write never rolls back an
already committed source mutation.

The owner/admin health response includes:

- pending outbox count
- failed outbox count
- last successful delivery time
- SQLite schema, journal, and FTS5 capabilities

The dispatcher retries failed evidence and replays pending rows after restart.
Do not delete outbox rows or clear the knowledge database to recover. Correct
the storage problem and restart; idempotency prevents duplicate candidates or
entries.

## Backup, restore, and search rebuild

The SQLite adapter backup operation checkpoints WAL and uses SQLite
`VACUUM INTO` to create a consistent protected database file. The destination
must not exist and must differ from the active database. Restore by stopping
the local server, retaining the current database as a rollback copy, placing
the verified backup at the configured knowledge path, and starting the server.

The `SearchIndex.Rebuild` operation reconstructs search data from published
entries and their current authoritative revisions. Search data is never the
source of truth.

## MCP

Workspace endpoint:

`/mcp/{workspaceSlug}/knowledge`

Protected-resource metadata:

- `/.well-known/oauth-protected-resource`
- `/.well-known/oauth-protected-resource/mcp/{workspaceSlug}/knowledge`

Scopes:

- `knowledge:read`
- `knowledge:candidate:read`
- `knowledge:propose`

Tools:

- `knowledge_search`
- `knowledge_list`
- `knowledge_get`
- `knowledge_propose`

The search and list tools return candidates only when
`include_candidates: true` is requested and the token has
`knowledge:candidate:read`. Candidate results are further restricted to the
authenticated Project lead's Projects unless the caller is an owner or admin.
No MCP tool can approve, reject, manage roles,
change permissions, configure workspaces, or invoke runtime-agent behavior.

The transport is stateless Streamable HTTP with JSON responses, request-size
limits, cancellation propagation, bearer authentication, protected-resource
discovery, and Origin validation. Local session tokens and default-service
personal access tokens provide compatible development and self-hosted paths.
The transport accepts an injected OAuth 2.1 token verifier and authorization
server metadata at the composition boundary without coupling the knowledge
domain to a particular identity provider.

The implementation uses the official Go MCP SDK:
<https://github.com/modelcontextprotocol/go-sdk>.

## Current limits

- The first automatic evidence producers cover Project creation/completion,
  Issue creation/acceptance, and terminal Task completion. Comment decisions,
  deliverable revisions, acceptance conclusions, and retrospectives remain
  future evidence producers behind the same envelope.
- The local server has no standalone backup or index-rebuild CLI yet; both
  operations are available at the adapter boundary and covered by tests.
- SQLite is the first adapter, not the product-level storage contract.

The approved completion scope and vertical delivery sequence for these limits
is documented in
[`docs/plans/knowledge-next-phase-completion.md`](plans/knowledge-next-phase-completion.md).
