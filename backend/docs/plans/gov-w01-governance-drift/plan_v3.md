# Governance and Architecture Drift Repair Plan v3

- Plan-ID: `GOV-W01-GOVERNANCE-DRIFT-001`
- Version: `v3`
- Status: `approved-for-execution`
- Base commit: `c43f4300eb29cf6778e67594e54cc79f8fb5057e`
- Branch: `agent/gov-w01-governance-drift-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `GOV-W01-GOVERNANCE-DRIFT-001`
- Task-Revision: `r003`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

## Goal

Remove repository-governance drift before continuing TC-W01: make `server/**`
permanently read-only at every policy layer, establish Execution/Runtime as an
owned bounded context with explicit relationships, and consolidate root and
canonical-backend validation behind one stable required CI gate.

## Hard invariants

1. `backend/**` is the only writable backend implementation root.
2. `server/**` is read-only migration evidence without a plan-level exception.
3. Required legacy behavior is inspected and ported into `backend/**`; it is
   never patched, synchronized, mirrored, or extended in `server/**`.
4. Every candidate diff containing `server/**` fails before deterministic
   checks or DoneGate.
5. Runtime execution cannot own Todo intent, Agent identity, Agent release, or
   final acceptance.
6. CI success is evidence only; merge and release remain independent human gates.
7. TC-W01 PR #9 remains Draft and is not modified by this governance Wave.

## Allowed paths

- `AGENTS.md`
- `CLAUDE.md`
- `CONTEXT-MAP.md`
- `docs/contexts/execution/**`
- `docs/contexts/system/CONTEXT.md`
- `docs/governance/**`
- `.github/workflows/ci.yml`
- `.github/workflows/backend.yml` (removal only)
- `backend/README.md`
- `backend/ci/check-policy.sh`
- `backend/docs/plans/gov-w01-governance-drift/**`

## Forbidden paths

- `server/**` without exception.
- Backend product code, database schemas, APIs, frontend product code, deployment,
  release, and TC-W01 implementation paths.

## Ordered steps

### GOV-W01-S00 — Freeze plan and evidence

Publish this immutable plan, current pointer, and append-only journal from the
exact canonical baseline commit.

Acceptance: scope, non-goals, invariants, checks, risks, rollback, and the
external branch-protection dependency are explicit.

### GOV-W01-S01 — Unify repository rules

Make root `AGENTS.md` and `CLAUDE.md` agree with `backend/AGENTS.md`:
`server/**` is permanently read-only; only behavior migration into
`backend/**` is allowed; every `server/**` diff is invalid.

Acceptance: no root rule grants an implementation plan authority to modify
`server/**`, and legacy-specific guidance is labeled read-only reference.

### GOV-W01-S02 — Establish Execution/Runtime context

Replace the unresolved ownership decision with an explicit Execution/Runtime
bounded context and context document.

Acceptance:

- Workspace, Auth, System, Space, and Runtime relationships are explicit;
- Todo and Run have separate identity, lifecycle, and multiplicity;
- Agent Identity, Agent Release, and Agent Execution each have one owner;
- Runtime cannot advance governed acceptance.

### GOV-W01-S03 — Consolidate CI

Move canonical backend validation into the root CI workflow, run it for pull
requests and pushes to both `main` and
`codex/multica-six-domain-baseline`, remove the duplicate backend workflow,
and expose one stable aggregate required check.

Acceptance:

- a repository policy job rejects every `server/**` diff;
- canonical `backend/**` runs `make check` and `make test-race`;
- a push to the canonical branch revalidates the backend;
- the legacy `server/**` regression job is named as legacy and cannot be
  confused with the canonical backend;
- the stable aggregate status is `CI / required`.

### GOV-W01-S04 — Protect the integration branch

Apply or hand off a repository ruleset for
`codex/multica-six-domain-baseline` requiring pull requests, conversation
resolution, and `CI / required`, while blocking force pushes and deletion.

Acceptance: the remote ruleset is verified through repository administration
state. A checked-in specification alone does not satisfy this step.

### GOV-W01-S05 — Verify and hand off

Verify the exact diff, workflow syntax, policy behavior, CI results, and absence
of `server/**` changes; then open a Draft PR for independent review.

Acceptance: deterministic evidence is indexed, remaining external blockers are
explicit, and no merge, release, or TC-W01 modification occurs.

## Deterministic verification

```bash
BASE_REF=<canonical-base-sha> backend/ci/check-policy.sh
ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml")'
git diff --name-only <canonical-base-sha>...HEAD
cd backend && make check && make test-race
```

GitHub Actions must additionally prove `CI / required` on the exact candidate
SHA. Remote branch-protection state must be inspected separately.

## Dependencies

- Canonical baseline `c43f4300eb29cf6778e67594e54cc79f8fb5057e`.
- Repository admin permission for branch-protection/ruleset mutation.
- Existing root CI remains available as migration input.
- TC-W01 Draft PR #9 remains open and may need a base refresh after this Wave
  merges.

## Risks and mitigations

- **Workflow-name drift:** one aggregate job owns the stable required-check name.
- **Skipped required checks:** aggregate dependencies always reach a terminal
  result; conditional work happens inside jobs rather than removing the gate.
- **Push diff ambiguity:** full checkout plus an explicit event base SHA is used.
- **Legacy CI cost:** legacy verification remains visible and separately named;
  retirement is a later authorized migration.
- **Ruleset tooling gap:** report it as an incomplete hard gate; do not claim a
  repository setting from documentation alone.
- **TC-W01 conflict:** use an isolated branch and leave PR #9 untouched.

## Rollback

Revert the GOV-W01 commits as one governance unit. Restoring the deleted
backend workflow is allowed only as rollback of this Wave; rollback must not
restore any permission to modify `server/**`. Repository ruleset rollback is
a separate admin action and must retain a required CI gate.

## Amendment from v1

Repository inspection found a second stale ownership statement in
`docs/contexts/system/CONTEXT.md`: it says the Agent execution lifecycle is
unconfirmed. Version v2 adds only that file to S02 so the System context and
root Context Map cannot contradict each other. All other scope, invariants,
steps, verification, risks, and rollback remain unchanged.

## Amendment from v2

The canonical backend README still enumerates four platform contexts and treats
Execution only as a cross-context capability. Version v3 adds
`backend/README.md` to S02 so the operator-facing architecture vocabulary is
consistent with the new Execution/Runtime bounded context. Historical immutable
plans remain untouched. All other v2 scope and gates remain unchanged.
