# Change Handoff

## Baseline

- Upstream repository: `smallnest/goclaw`
- Upstream baseline commit: `e05c79d`
- Delivery: `goclaw-team-runtime-*-0.8.0-pilot.1`; Obsidian adapter optional

## Delivered behavior

### Wave update governance

- Added `docs/waves/` as the repository authority for staged update plans,
  revisioned scope, append-only progress journals, issue facts, decisions, and
  verification evidence.
- Added the `PILOT-READINESS-2026-07` track with `PILOT-W00` active and
  revisioned product-code scope for the controlled three-person pilot.
- Added the Wave-first repository policy to `AGENTS.md` and linked it from the
  team development, Web Console, implementation-status, and README documents.
- Team `dev create` now sends only a Step intent. The Gateway resolves the
  registered repository at the exact base commit, validates the one active
  Wave, approved/active plan, dependencies, Step, scope and plan/Registry
  SHA-256, then repeats the binding checks at freeze, enqueue and acceptance.

### Team control plane

- Added central, single-writer TeamControl behind the authenticated Gateway.
- Added one-time first-admin bootstrap and per-person access tokens stored as SHA-256 digests.
- Added Team → Project → Repository boundaries and server-side project RBAC.
- Added team owner/admin/member and project owner/maintainer/developer/reviewer/viewer roles.
- Added project member business domains and capacity points.
- Added Issue, WorkItem, and Assignment contracts with guarded state transitions and dependency checks.
- Added Artifact and CorrelationLink records for task → run → diff/evidence → commit → PR → CI → release → regression traceability.
- Added versioned Document, Component, and hierarchical team → project → repository → component Policy registries.

### Workstation runtime

- Added per-computer Runner registration with owner, project, capability, and a dedicated device key.
- Added a durable central queue with idempotent claim/heartbeat/complete/fail operations, one active lease per Runner, retries after lease expiry, and offline detection.
- Added secret-free, content-hashed ExecutionPacks carrying project, repository, task revision, Issue/WorkItem IDs, base commit, policy hash, path scope, and frozen verification argv.
- Added local revision/attempt worktrees and use of each member's own Codex OAuth login.
- Added signed EvidenceBundles with diff, changed paths, deterministic verification, Trace IDs, Harness version, and policy hash.
- Added server-trusted `dev.task.enqueue`, which rebuilds the ExecutionPack from a frozen task and rejects client-provided packs.
- Added idle-only Runner device-key rotation, signed evidence/patch retrieval, and HTTPS Gateway CLI overrides.
- Added child-process environment filtering: GoClaw control-plane secrets are always removed and other sensitive variables require explicit allowlisting.
- Added an explicit no-automatic-commit check; the Runner never pushes, opens or merges a PR, waits for CI, or releases.

### ChatGPT subscription runtime

- Added the `codex-app-server` provider.
- Reuses the local Codex CLI ChatGPT login.
- Does not accept, copy, or persist an OpenAI API key or Codex OAuth files.
- Keeps provider decisions read-only and validates the structured result in Go.

### Project memory and Obsidian

- Added project/topic session routing shared by Obsidian and Feishu.
- Added governed knowledge search/read tools and approval-gated full-content proposals.
- Uses SHA-256 optimistic concurrency when applying knowledge proposals.
- Added a provenance-aware catalog with Work/Expression/Manifestation/Item identity, descriptive metadata, authority control, explicit relations, lifecycle review, expiry, citations, and circulation events.
- Added candidate-only Agent/Vault ingestion; only authenticated `memory_approve` reviewers can make records active.
- Added project scope enforcement, cross-project reference checks, shared read-only records, and bounded untrusted-evidence prompt injection.
- Added an offline builtin hash embedder and portable SQLite cosine fallback, removing the external embedding-key requirement.
- Added the Obsidian Project Console for chat, team status, memory, approvals, development, traces, and Harness state.
- Added a read-only Team view for member load, project work, Bugs, Runner/lease health, policy layers, documents, and reusable components.

### Better Harness

- Added immutable Harness versions, isolated candidates, experiments, promotion CAS, and rollback.
- Added Optimization, Golden, and Holdout eval splits.
- Added baseline-versus-candidate case comparison and regression rejection.
- Added candidate target scope and protected-path enforcement.
- Added token and latency delta gates for explicit trace metrics.
- Trace records now include tool inputs, results, errors, and status.

### Orchestrator Lite

- Added structured development contracts and four mandatory reviews:
  Scenario, Capacity, Risk, and Cost.
- Added append-only, hash-chained SessionEvents as the sole task state source.
- Added Git base freezing, execution-bundle hashing, isolated worktrees, and task locks.
- Added a Codex `exec --json` development Hand with resume support.
- Added Go policy checks, deterministic argv verification, independent read-only review, EvidencePackage, and Go-only DoneGate.
- Added bounded repair, human acceptance with evidence/worktree anti-tamper checks, and local commit.
- Added Gateway read/review/freeze/accept methods and opt-in long-running execution.
- Added `goclaw dev` CLI lifecycle commands.

### Go-native Ouroboros

- Added the native interview → Seed → execute → evaluate → evolve state machine, adapted from `Q00/ouroboros` concepts under its MIT license.
- Added deterministic weighted ambiguity, per-dimension floors, two-pass readiness, repeated-question escalation, and strict model JSON with one bounded repair.
- Added immutable content-addressed Seeds and a hash-chained event ledger.
- Added explicit human Seed/evolution approvals and one-way compilation into the existing four-review Orchestrator Lite contract.
- Added mechanical EvidencePackage checks, semantic evaluation, independent role reviews, and Go majority consensus.
- Added ontology similarity, generation cap, oscillation detection, and candidate-only evolution.
- Added `goclaw ouroboros`, JSON-RPC methods, low-privilege Feishu tools, an embedded `ouroboros-spec` skill, and the Obsidian “规格” control surface.

## Migration

1. Build the updated GoClaw binary with its embedded Team Web Console.
2. Back up the existing `~/.goclaw` directory, Catalog and Markdown knowledge root.
3. Merge the `team_control`, `workstation`, `memory.catalog`, `ouroboros`, and `development` blocks from `deploy/config.codex-obsidian.example.json`.
4. Keep TeamControl, Workstation queue/leases, Catalog/builtin SQLite, `ouroboros.root`, `harness.root`, `development.root`, worktrees, sessions, locks, and active traces outside the synchronized Vault.
5. Open `/dashboard/`, verify secure browser login, and install the optional Obsidian adapter only when required.
6. Build and verify these release artifacts:

   - `goclaw-team-runtime-linux-amd64-0.8.0-pilot.1.tar.gz`
   - `goclaw-team-runtime-linux-arm64-0.8.0-pilot.1.tar.gz`
   - `goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz`
   - `obsidian-goclaw-plugin-0.8.0-pilot.1.tar.gz`（可选）
   - `SHA256SUMS-0.8.0-pilot.1.txt`

7. Bootstrap the first admin once, create the team/project/repositories, then create each user in the order user → team membership → personal token → project membership.
8. Restart central GoClaw using the same OS user that ran the central `codex login`.
9. On each of the three pilot computers, set `GOCLAW_GATEWAY_HTTP_URL`,
   `GOCLAW_GATEWAY_TOKEN`, and that member's `GOCLAW_USER_TOKEN`; use the
   member's own device key, repository mappings, local `codex login`, and the
   same reviewed bwrap wrapper to register and start one Linux-substrate Runner.
10. Run:

   ```bash
   goclaw memory catalog status --project project-alpha
   goclaw harness status
   goclaw ouroboros init
   goclaw ouroboros list
   goclaw dev init
   goclaw dev list --json
   ```

11. Create a disposable test repository and complete one four-review task plus one Runner lease/evidence cycle before enabling the runtime on a production repository.

Existing Harness versions and traces remain readable. The richer validation report adds baseline fields but does not mutate old reports. Existing Obsidian settings remain compatible; Team modules degrade independently when a service is disabled or denied.

## Operational invariants

- Do not synchronize active JSONL, locks, runtime registries, or worktrees through Obsidian Sync.
- Do not synchronize TeamControl state, Workstation tasks/leases, personal tokens, Runner device keys, or Codex OAuth through the Vault.
- Run exactly one central GoClaw writer for each TeamControl and Workstation root.
- Give every member a separate personal token, Runner key, and local Codex login.
- Run each workstation under a dedicated least-privilege OS user, VM, or container and mount only authorized repositories, the work root, device key, and that user's Codex OAuth.
- Resolve project/repository authorization on the server; Feishu project routing and Obsidian settings are not authorization sources.
- Do not treat Obsidian boards as a queue or authoritative state.
- Do not let Runner completion bypass human review, commit/PR policy, CI, final acceptance, or release controls.
- Do not synchronize `catalog.db`, its WAL/SHM files, or the builtin memory index through Obsidian Sync.
- Do not promote imported or model-proposed memory without human evidence review; expiry and contradiction warnings are not proof of correctness.
- Do not edit Ouroboros events or immutable Seeds; restore a consistent runtime backup after any hash failure.
- Do not treat Seed approval as correctness or allow an evolution candidate to mutate active state without human approval.
- Do not treat `task.json`, Markdown boards, or UI state as authoritative.
- Do not edit a frozen task; create a ChangeIntent and repeat all four reviews.
- Do not use model text as proof of completion; inspect DoneGate and EvidencePackage.
- Do not use `resume --force` until the old process is confirmed stopped.
- Do not expose a plaintext `ws://` Gateway remotely.
- Keep privileged development execution disabled unless the personal-token and project-RBAC path is explicitly enabled and audited.

## Known production gaps

- Team Web Console deterministic scope/auth/history/mutation tests and build
  pass, but cloud-browser localhost policy blocked the real Desktop/Mobile and
  three-context interaction run; target-browser acceptance remains open.
- Wave governance is server-enforced for Team create/freeze/enqueue/accept.
  External Git hosting still has no branch protection, signed-commit, PR or CI
  integration in this product.
- TeamControl and Workstation are file-backed, safe only for in-process concurrency and one central writer; there is no HA leader, external transaction database, or multi-process consensus.
- A non-empty ExecutionPack assignee is enforced against the Runner owner; business domain and member capacity remain planning/dashboard fields rather than scheduling optimizers.
- Frozen verification argv run as the Runner OS identity after environment filtering; process/VM/container isolation remains a deployment responsibility.
- No GitHub/GitLab/Jira two-way synchronization.
- No public-key or hardware-backed Runner identity or cryptographic reviewer signature; device-key rotation is implemented with the current HMAC shared-secret model and is allowed only while the Runner is idle.
- No external signature/WORM store for SessionEvents and evidence.
- No automatic commit, Git push, PR creation, CI wait, merge-policy enforcement, or release.
- No container/microVM network isolation for the development Hand.
- No automated trace clustering or statistically powered online experiments.
- No RDF/SPARQL exchange, automatic subject authority disambiguation, field-level catalog encryption, or multi-leader catalog consensus.
- The Obsidian Team view passed TypeScript build and unit tests, but cloud-browser visual QA was blocked by the local-address security policy; production handoff still requires a real Obsidian Desktop visual and interaction check.

These are explicit future boundaries, not hidden completion claims.
