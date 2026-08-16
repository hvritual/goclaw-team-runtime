# S01B governance foundation design

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Design-ID: `PCR-S01B-DESIGN-001`
- Status: `proposed for Customer approval with plan_v2`
- Design base: `2589ce450aa8527a19e21ff23812e1659bfd0ddd`
- Scope owner: Canonical Workspace
- Updated: `2026-08-16`

## 1. Outcome

S01B adds one reusable mutation-governance foundation for roadmap capabilities:

- optimistic revision checks;
- command idempotency and deterministic replay;
- immutable, redacted audit records;
- durable transactional outbox delivery;
- bounded operational diagnostics.

It does not retrofit existing Issue, project, pin, attachment, or realtime
mutations. Each later capability opts into this foundation in its own vertical
slice. Existing feature flags remain false until those slices are accepted.

## 2. Ownership and boundaries

Canonical Workspace owns all S01B contracts and SQLite rows. The independent
Control Plane kernel is read-only migration evidence; S01B does not import its
packages, read its tables, invoke its process, or dual-write its records.

```text
Workspace application command
  -> capability repository owns BEGIN IMMEDIATE
     -> validate trusted workspace and actor
     -> check idempotency key and request hash
     -> compare expected resource revision
     -> perform domain mutation
     -> advance revision
     -> append redacted audit
     -> enqueue outbox event(s)
     -> store replay response
     -> COMMIT

Outbox dispatcher
  -> claim ready rows with a lease
  -> publish through injected sink after commit
  -> mark delivered, retry_wait, or dead_letter
```

Domain and application packages never receive `*sql.DB`, `*sql.Tx`, or SQLite
types. SQLite infrastructure owns the native connection and transaction. A
capability repository composes the governance helper on the same connection as
its domain writes; nested or cross-repository transactions are forbidden.

## 3. Contract values

The Workspace public contract defines transport-neutral values:

- `MutationIdentity`: workspace ID, trusted actor type/ID, request ID;
- `MutationCommand`: action, resource kind/ID, expected revision, optional
  idempotency key, canonical request hash;
- `MutationResult`: resource revision, replayed flag, bounded response envelope;
- `AuditRecord`: allowlisted metadata and resource revision;
- `OutboxEvent`: stable event ID/type, aggregate identity/revision, bounded
  payload, actor projection;
- `OutboxDiagnostics`: ready count, oldest ready age, inflight count,
  dead-letter count, last successful delivery time.

Stable errors are:

| Error code | Meaning | HTTP mapping when a later route uses it |
| --- | --- | --- |
| `revision_conflict` | expected revision is stale | `409`, include current revision |
| `idempotency_conflict` | same key, different canonical request hash | `409` |
| `idempotency_response_too_large` | replay envelope exceeds 64 KiB | `400` |
| `governance_unavailable` | required provider is not installed | `503` |
| `outbox_claim_conflict` | claim token or lease is stale | internal retry decision |

Unknown action/resource mappings fail closed. Error bodies never echo request
content, imported file bodies, credentials, or raw payloads.

## 4. SQLite schema

Migration `000009_workspace_governance` adds four Workspace-owned tables. It
adds no foreign keys, cascades, triggers, explicit `CREATE INDEX`, or
`CREATE UNIQUE INDEX` statements.

### `workspace_resource_revisions`

```text
workspace_id TEXT
resource_kind TEXT
resource_id TEXT
revision INTEGER >= 0
updated_at TEXT
PRIMARY KEY (workspace_id, resource_kind, resource_id)
```

Revision `0` means the resource does not yet have a committed governed
mutation. A successful create advances to `1`; every later successful mutation
advances by exactly one. Failed validation, stale writes, replay, rollback, and
outbox delivery do not advance it.

### `workspace_mutation_idempotency`

```text
workspace_id TEXT
action TEXT
idempotency_key TEXT
request_hash TEXT
resource_kind TEXT
resource_id TEXT
resource_revision INTEGER
response_status INTEGER
response_body TEXT <= 64 KiB
created_at TEXT
expires_at TEXT nullable
PRIMARY KEY (workspace_id, action, idempotency_key)
```

Only a committed mutation leaves an idempotency row. Same key and hash returns
the stored status/body without running the domain mutation or publishing a
second event. Same key and different hash returns `idempotency_conflict`.

### `workspace_audit_entries`

```text
workspace_id TEXT
occurred_at TEXT
id TEXT
actor_type TEXT
actor_id TEXT
action TEXT
resource_kind TEXT
resource_id TEXT
resource_revision INTEGER
request_id TEXT
metadata_json TEXT, valid JSON, allowlisted and bounded to 16 KiB
PRIMARY KEY (workspace_id, occurred_at, id)
```

Audit rows are append-only. S01B provides no update or delete API. The frozen
730-day retention is an operational minimum; physical retention processing is
deferred until an approved retention task.

### `workspace_outbox_events`

```text
state TEXT: ready | inflight | retry_wait | delivered | dead_letter
available_at TEXT
workspace_id TEXT
id TEXT
event_type TEXT
aggregate_kind TEXT
aggregate_id TEXT
aggregate_revision INTEGER
payload_json TEXT, valid JSON, bounded to 64 KiB
actor_type TEXT
actor_id TEXT
attempt_count INTEGER
claim_token TEXT nullable
lease_expires_at TEXT nullable
last_error_code TEXT nullable
created_at TEXT
delivered_at TEXT nullable
PRIMARY KEY (state, available_at, workspace_id, id)
```

The state and availability columns lead the primary key so the dispatcher can
perform bounded ready/retry scans without a secondary index. State transitions
rewrite the primary-key tuple atomically. Callers retain all tuple values and a
claim token; an event ID by itself is never accepted as write authority.

## 5. Migration-policy resolution

Root policy requires explicit indexes to use one standalone
`CREATE [UNIQUE] INDEX CONCURRENTLY` statement outside a transaction. SQLite
does not implement that syntax, and the current Workspace runner applies its
provider migrations inside one transaction.

S01B resolves the conflict without changing either rule:

1. the SQLite provider migration contains table definitions and primary-key
   identity constraints only;
2. it contains no explicit index DDL or secondary uniqueness constraint;
3. access paths are supplied by the declared composite primary keys;
4. a future PostgreSQL provider must place every secondary/unique index in its
   own concurrent single-statement migration and run it outside a transaction;
5. if measured SQLite budgets require a secondary index, implementation stops
   and proposes a new plan version rather than adding non-concurrent index DDL.

Migration tests must prove first install, second-run idempotence, retained-data
upgrade from version 8, restart persistence, and rollback of a failed migration.

## 6. Transaction algorithm

For a governed mutation the capability repository:

1. acquires one dedicated SQLite connection and executes `BEGIN IMMEDIATE`;
2. resolves the trusted Workspace actor before any write;
3. checks the idempotency primary key when one is required;
4. returns the stored response for a matching hash, or conflict for a mismatch;
5. reads the current resource revision and compares `expected_revision`;
6. performs the domain mutation on the same connection;
7. advances the resource revision by one using the expected current value;
8. inserts one redacted audit row and zero or more outbox rows;
9. inserts the bounded idempotency replay row when applicable;
10. commits once, then returns the result.

Any error rolls back every write. Audit/outbox/idempotency records never
describe a mutation that did not commit. A test-only abort point must prove the
rollback invariant after each write phase.

## 7. Request hashing and replay

- The transport validates and normalizes the request first.
- Application code creates a versioned canonical JSON projection containing
  only fields that affect the command result.
- SHA-256 of `contract-version + action + canonical-json` is persisted.
- Workspace ID and action are part of the idempotency primary key; keys cannot
  replay across workspaces or actions.
- Stored response envelopes use a version field and are limited to 64 KiB.
- Expiry never permits replay of a destructive command as a new mutation inside
  its capability retention window.

## 8. Audit safety

Audit metadata is an action-specific allowlist. It may contain identifiers,
state transitions, revision numbers, counts, reason codes, and content hashes.
It must not contain credentials, cookies, bearer tokens, raw Skill archives,
attachment bodies, notification text, model prompts, or unrestricted request
bodies. Tests scan serialized audit payloads for forbidden fixture values.

## 9. Outbox delivery

- Default claim batch: `100`; hard cap: `500`.
- Lease: `60s`; reclaim only after lease expiry.
- Attempts: initial delivery plus three bounded retries.
- Retry scheduling uses injected deterministic jitter in tests.
- Publish succeeds at least once. A crash after publish and before ack may
  duplicate delivery; consumers use event ID and aggregate revision to dedupe.
- A stale claim token cannot ack, retry, or dead-letter an event.
- Exhausted events enter `dead_letter`; replay requires the existing
  `PermissionReminderReplayRepair`-class operator authority or a later
  capability-specific equivalent and preserves the original event ID.

The default Canonical runtime injects an outbox sink only after the durable
repository and dispatcher are both installed. Missing repository or sink keeps
the foundation unavailable and does not enable any roadmap feature flag.

## 10. Diagnostics

S01B exposes a Workspace-scoped, owner/admin-only governance projection through
`GET /api/operations/governance`. It returns counts and timestamps only:

- ready count and oldest ready age;
- inflight count and oldest lease age;
- retry-wait and dead-letter counts;
- last successful delivery timestamp;
- schema version and dispatcher running state.

It returns no event payload, audit metadata, resource title, URL, file path, or
notification text. Database unavailability fails readiness. Backlog older than
15 minutes is reported degraded but does not by itself restart the process.

## 11. Verification matrix

Tests must cover:

- same expected revision under concurrency: one commit and one conflict;
- same idempotency key/body: one mutation and deterministic replay;
- same key/different body: conflict and no second mutation;
- workspace/action isolation for keys, revisions, audit, and outbox;
- rollback after domain, revision, audit, outbox, and replay-row phases;
- audit redaction and payload bounds;
- outbox claim exclusivity, lease expiry, stale token, retry cap,
  dead-letter, replay, restart, and duplicate-delivery tolerance;
- diagnostics authorization and content redaction;
- retained version-8 database upgrade and restart;
- existing Issue/project/pin/auth/attachment/realtime behavior unchanged;
- all roadmap feature flags remain false.

## 12. Rollback

Code rollback disables governance composition and the dispatcher while retaining
the additive tables. The down migration is test-only and removes only empty
S01B tables in reverse dependency order. Normal operational rollback never
deletes audit, idempotency, or outbox evidence. Data deletion requires separate
Customer authority.
