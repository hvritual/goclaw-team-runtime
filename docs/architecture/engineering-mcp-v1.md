# Engineering MCP v1

P2-S08 exposes the accepted Engineering Digital Thread and deterministic Context Compiler to LLM/Agent clients without transferring ownership of Engineering truth, Workspace work intent, Runtime lifecycle, governed Knowledge, authorization, or human DoneGate.

## Transport and process boundary

V1 is an independent stdio sidecar implemented by `backend/cmd/engineering-mcp` using the official Tier-1 Go MCP SDK `github.com/modelcontextprotocol/go-sdk` v1.7.0.

The process is started with:

- `--sqlite-path` — canonical SQLite database path;
- `--user-id` — fixed Auth user identity for the lifetime of the process;
- `--version` — optional server implementation version.

V1 intentionally does not expose an unauthenticated HTTP MCP transport. Tool arguments cannot supply or override `user_id`.

```text
LLM / Agent client
      |
      | MCP stdio
      v
engineering-mcp process
      |
      | fixed user identity
      v
Engineering application contracts
      |
      +--> Auth WorkspaceMembershipReader
      +--> Engineering Service
      +--> Authorized Context Compiler
```

Engineering MCP does not query Auth tables directly. The sidecar constructs the existing Auth-owned workspace-membership reader and passes it through the Engineering application boundary.

## Tool catalog

V1 exposes exactly eight tools:

### EngineeringEntity reads

- `engineering_entity_get`
- `engineering_entity_list`
- `engineering_entity_search`

Entity list/search are deterministic and bounded. V1 search is graph/data-model first rather than vector-first: it matches canonical ID, name, type, status, and owner reference, then returns stable ID ordering.

### Engineering Thread traversal

- `engineering_thread_traverse`

Traversal is deterministic breadth-first traversal with bounded input:

- default depth: 2;
- hard maximum depth: 4;
- default node budget: 64;
- hard maximum node budget: 256.

The result contains stable ordered nodes, stable ordered edges, and an explicit `truncated` flag when the node budget is reached.

### Change reads

- `engineering_change_get`
- `engineering_change_list`

These tools are read-only. They cannot accept, reject, supersede, or otherwise mutate a Change.

### ContextPack capabilities

- `context_pack_get`
- `context_pack_compile`

`context_pack_get` reads an already frozen immutable ContextPack.

`context_pack_compile` is the only V1 MCP tool that persists state. It calls the accepted deterministic P2-S06 Context Compiler and freezes an immutable ContextPack. It does not mutate an existing pack. Same-ID semantic drift remains a conflict through the existing ContextPack persistence contract.

## Authorization

Read tools use the existing Engineering `contract.Service`, so the established application-boundary workspace authorization remains authoritative:

- owner/admin: read;
- member: read;
- non-member: forbidden.

Compilation uses `AuthorizedContextCompiler`, a thin Engineering application wrapper over the P2-S06 compiler:

- owner/admin: compile and freeze ContextPack;
- member: forbidden;
- non-member: forbidden;
- membership resolution failure: unavailable.

The MCP transport itself does not make authorization decisions and cannot bypass application policy.

## Error boundary

MCP returns stable semantic error classes and does not forward dependency/internal-storage details to clients. Internal errors such as SQLite paths, credentials, tokens, or wrapped dependency messages are reduced to stable categories such as `invalid_argument`, `forbidden`, `not_found`, `conflict`, and `unavailable`.

## Explicit non-capabilities

Engineering MCP v1 cannot:

- create/update/archive EngineeringEntity records;
- create or mutate SourceBinding or ThreadEdge records;
- accept/reject/supersede a Change;
- publish or quarantine governed Knowledge;
- change Workspace membership, role, or permission;
- claim/heartbeat/retry/complete a Runtime Run;
- satisfy Todo/Task acceptance or human DoneGate;
- change Agent Release or Skill publication state.

These remain owned by their existing bounded contexts and human/governed workflows.

## Dependency reproducibility

The MCP dependency update was generated on a GitHub-hosted Go 1.26.1 runner rather than manually reconstructed.

The temporary probe branch `agent/iot-edt-p2-s08-tidy-probe` ran `go mod tidy` and produced the exact module diff. Probe run `33482625215` showed the required changes, and run `33482732419` persisted the machine-generated result as commit `594c1348fccaa3dd894845c0b191c278d249c2f7`.

Only the resulting `backend/go.mod` and `backend/go.sum` Git blobs were reused in product commit `896d19947f8dd8522491a238406f132a95c1a2a4`; the temporary workflow was not copied. The product tidy commit changes only those two files. `go.sum` is additions-only (`+16/-0`), so no pre-existing checksum was modified or removed.

## Validation evidence

P2-S08 product validation head:

`896d19947f8dd8522491a238406f132a95c1a2a4`

Validation-only draft PR:

`#14 — WIP: validate P2-S08 Engineering MCP`

Canonical workflow run:

`33482861699`

The run passed:

- `governance-policy`;
- repository-declared Go 1.26.1 deterministic `make check`;
- canonical `make test-race`;
- frontend test/build aggregate;
- final `CI / required`.

This proves the official MCP SDK v1.7.0 module graph, typed tool registration/handlers, stdio sidecar, authorization wrapper, existing backend tests, vet, and race suite under the repository canonical workflow.

## Integration status

P2-S08 was deliberately developed on `agent/iot-edt-p2-s08-mcp`, forked from the latest P2-S07 integration head, so no P2-S08 push can stale PR #13's independent-review gate.

PR #14 is validation-only and must not be merged directly. Canonical integration remains ordered behind protected PR #13 (`P1-EXIT + P2-S01..S07`), which requires an independent GitHub approval from someone other than the last pusher. After that integration lands, P2-S08 must be rebased/integrated through the same protected-branch rules; validation success does not claim that canonical branch already contains P2-S08.
