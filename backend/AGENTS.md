# GoClaw Team Engineering Policy

These rules apply to every human and Codex change in this repository.

## Agent orchestration

- Follow `Backend Agent Orchestration` in the root `CLAUDE.md`. It keeps Sol on
  primary planning and uses the project `side_luna` agent for bounded read-only
  investigations without weakening the versioned-plan gate below.

## Backend path boundary

This section records the user's explicit repository-governance decision of
2026-08-11 and overrides any older plan-level exception elsewhere in the
repository.

- `backend/` is the only writable backend implementation root.
- `server/` is read-only migration evidence. Never modify `server/**` for
  features, bug fixes, refactors, tests, configuration, schemas, migrations,
  generators, or implementation-coupled documentation.
- When required behavior exists only in `server/`, inspect it and port the
  behavior into `backend/` under an approved versioned plan with tests. Do not
  patch, synchronize, mirror, or extend the legacy tree.
- Frozen tasks, plan steps, worktrees, and review scopes for backend work must
  exclude `server/**` from allowed paths. A candidate diff containing any
  `server/**` change is invalid and must be stopped before deterministic checks,
  model review, or DoneGate.
- This boundary has no plan-level exception. Changing it requires a new explicit
  repository-governance decision from the user; an implementation plan alone is
  insufficient authority.

## Versioned implementation plans

- Every staged, multi-step, or multi-module update must have a versioned plan
  under `backend/docs/plans/<work-item>/` before product code is changed.
- Use `plan.md` as the current plan entry point and `plan_vN.md` (for example,
  `plan_v1.md` and `plan_v2.md`) as immutable execution snapshots. `plan.md`
  must identify and link to the exact approved version currently in force.
- Each versioned plan must declare a stable `Plan-ID`, version, status, base
  commit, scope and non-goals, invariants, dependencies, ordered step IDs,
  acceptance criteria, deterministic verification, risks, and rollback.
- A task may change product code only when it references an approved plan
  version and one active step. The changed file or contract must be inside that
  step's declared scope; discovery-only steps permit documentation and
  diagnostics only.
- Before starting or resuming a step, reread `plan.md`, the referenced
  `plan_vN.md`, and the task scope. Stop when they disagree or when the next
  action is not explicitly covered. Do not infer broader authority from the
  plan's overall goal.
- An approved `plan_vN.md` is immutable after execution begins. Material changes
  to scope, contracts, gates, dependencies, risks, or rollback require a new
  `plan_v(N+1).md`; update `plan.md` only after the new version is approved.
- Record progress, decisions, deviations, and verification evidence in an
  append-only journal next to the plan. Never rewrite an older plan version to
  match work that has already happened.
- A reported or statically suspected problem is not a confirmed defect. A repair
  step requires reproduction evidence covering the environment, role, project,
  action, expected result, actual result, and sanitized logs.
- Execute steps in dependency order. Do not start a dependent step while its
  prerequisites are incomplete, and do not mark a plan complete without indexed
  verification evidence and independent review.
- In Team mode, freeze the exact plan version, step ID, base commit, and content
  hashes at task creation, then revalidate them at enqueue, execution, and
  acceptance. Work outside Team mode must apply the same checks manually.

## Traceability

- Work only from a frozen task with a project, repository, assignee, base commit,
  policy bundle hash, `Plan-ID`, `Plan-Version`, `Plan-Step`, acceptance
  criteria, and deterministic verification.
- Keep each change in its task worktree and plan-version-specific branch.
- Every commit must include `Task-ID`, `Project-ID`, `Task-Revision`, and
  `Work-Item` trailers. Include `Plan-ID`, `Plan-Version`, `Plan-Step`, `Issue`,
  and `Policy-Bundle` when present.
- Do not mix unrelated fixes into one task or pull request.

## Context compaction and workspace recovery

- Before context compaction, handoff, or an expected interruption, write a
  recovery checkpoint to the task's durable journal or task artifacts. Include
  the task, project, plan ID, plan version, and step identifiers; repository,
  worktree, branch, and base commit; current scope and acceptance criteria;
  files changed; decisions made; verification already run and its results;
  unresolved risks or blockers; and the next concrete action.
- Keep recovery evidence in the repository or control plane. Do not rely on chat
  history, transient terminal output, or model memory as the only record. Never
  store credentials or unsanitized secrets in a checkpoint.
- After compaction, restart, or reassignment, restore the workspace from ground
  truth before changing product code: reread the applicable `AGENTS.md` and
  `CLAUDE.md`, the frozen task, `plan.md`, the exact approved `plan_vN.md`, and
  the plan journal; then inspect the current worktree, branch, base commit, Git
  status and diff, changed files, and recorded verification evidence.
- Treat a recovery checkpoint as a navigation aid, not as authority. The current
  repository, frozen task, approved plan version, and control-plane state take
  precedence. If they disagree with the checkpoint, stop implementation, record
  the drift, and refresh or re-freeze the task before continuing.
- Preserve user and agent work already present in the worktree. Do not repeat
  completed steps, overwrite unrelated or uncommitted changes, or claim checks
  that cannot be verified. Resume from the last verified step and rerun the
  narrowest relevant validation when recovered state may have changed.
- If scope, authorization, acceptance criteria, or the next safe action cannot
  be reconstructed, pause product changes and request clarification or task
  recovery instead of guessing.

## Reuse first

- Search existing packages, components, APIs, schemas, templates, and the
  Component Registry before introducing a new abstraction.
- Extend a compatible shared component instead of copying it.
- New shared components require a clear owner, compatibility contract, tests,
  usage example, and deprecation policy.

## Go

- Run `gofmt` on changed Go files.
- Prefer small packages with explicit dependencies and constructor validation.
- Wrap errors with operation context; never log or persist credentials.
- Use table-driven tests for state transitions and authorization matrices.
- File-backed services must use atomic writes and be safe for concurrent use.

## TypeScript and Obsidian

- Keep gateway contracts typed and project-scoped.
- Render explicit loading, empty, denied, and error states.
- Do not put access tokens, reviewer tokens, or device secrets into the Vault.
- Views are projections; task, issue, runner, and policy state belongs to the
  central control plane.

## Governance

- Deterministic checks precede model review.
- A creator or assignee cannot perform final acceptance of their own work.
- Project and repository authorization is resolved server-side.
- Documentation, regression evidence, and issue links are part of DoneGate when
  required by the frozen task.
