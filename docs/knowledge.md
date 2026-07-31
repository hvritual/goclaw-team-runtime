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

SQLite schema version: `1`.

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

The local SQLite server recognizes:

| Variable | Purpose | Default |
| --- | --- | --- |
| `MULTICA_KNOWLEDGE_ENABLED` | Set to `false` to disable knowledge only | enabled |
| `MULTICA_KNOWLEDGE_SQLITE_PATH` | Knowledge SQLite file | sibling `*.knowledge.db` |
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

Owners and admins additionally use:

- `GET /api/knowledge/candidates`
- `POST /api/knowledge/candidates/{id}/review`
- `GET /api/knowledge/health`

Every request resolves canonical workspace membership. Optional Project IDs
must belong to the active workspace. Ordinary members are not shown candidate
queues, permission descriptions, or governance configuration.

## Evidence outbox and recovery

Project, Issue, and terminal Task mutations append normalized evidence inside
the source SQLite transaction. Delivery to the separate knowledge database is
at least once and idempotent. A failed knowledge write never rolls back the
source mutation.

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
`knowledge:candidate:read`. No MCP tool can approve, reject, manage roles,
change permissions, configure workspaces, or invoke runtime-agent behavior.

The transport is stateless Streamable HTTP with JSON responses, request-size
limits, cancellation propagation, bearer authentication, protected-resource
discovery, and Origin validation. Local session tokens provide a PAT-compatible
development path. A remote deployment supplies an OAuth 2.1 token verifier and
authorization-server metadata through server options.

The implementation uses the official Go MCP SDK:
<https://github.com/modelcontextprotocol/go-sdk>.

## Current limits

- The first automatic evidence producers cover Project creation/completion,
  Issue creation/acceptance, and terminal Task completion. Comment decisions,
  deliverable revisions, acceptance conclusions, and retrospectives remain
  future evidence producers behind the same envelope.
- Project-lead review is not introduced as a separate role. The existing
  Project `lead_id` capability still needs candidate filtering and review
  authorization; owner/admin authorization is used in this increment.
- The default PostgreSQL server composition does not yet register the
  knowledge API, source outbox, or MCP transport. Deployment must use the
  SQLite-local composition until that wiring is added.
- Published-entry supersession and subsequent revision proposals remain to be
  implemented. The current model preserves immutable first revisions and
  optimistic candidate-review revisions.
- The local server has no standalone backup or index-rebuild CLI yet; both
  operations are available at the adapter boundary and covered by tests.
- SQLite is the first adapter, not the product-level storage contract.
