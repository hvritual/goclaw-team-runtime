# GoClaw Team Engineering Policy

These rules apply to every human and Codex change in this repository.

## Wave planning gate

- Every staged, multi-step, or multi-module update must have an entry in
  `docs/waves/wave-registry.json` and an approved revisioned plan under
  `docs/waves/` before product code is changed.
- Update the plan before implementation. A frozen task must reference the
  `Wave-ID`, `Issue-ID`, exact plan revision, and step ID that authorize its
  scope.
- Product code may change only when the registry marks that Wave `active`, the
  active plan permits product changes, and the file/contract is inside its
  allowed scope. `FE-W00` and any other discovery-only Wave are documentation
  and diagnostic work only.
- A reported or statically suspected problem is not a confirmed defect. It may
  enter a repair Wave only after reproduction evidence records the environment,
  role, project, action, expected result, actual result, and sanitized logs.
- An approved plan revision is immutable. Material scope, contract, gate, risk,
  or rollback changes require a new plan revision before implementation
  continues. Append status, decisions, and evidence to the Wave journal; do not
  rewrite history.
- Do not activate a later Wave while its dependencies are incomplete. Do not
  mark a Wave complete without indexed verification evidence and independent
  review.
- In Team mode, the runtime resolves the requested Wave step from the active
  registry at the registered repository's exact base commit, freezes the plan
  revision and hashes, and revalidates them at freeze, enqueue, and acceptance.
  Repository work outside that path must still follow this policy manually; do
  not describe an unbound local task as Wave-governed.

## Traceability

- Work only from a frozen task with a project, repository, assignee, base commit,
  policy bundle hash, acceptance criteria, and deterministic verification.
- Keep each change in its task worktree and revision-specific branch.
- Every commit must include `Task-ID`, `Project-ID`, `Task-Revision`, and
  `Work-Item` trailers. Include `Wave-ID`, `Wave-Revision`, `Wave-Step`,
  `Issue`, and `Policy-Bundle` when present.
- Do not mix unrelated fixes into one task or pull request.

## Context compaction and workspace recovery

- Before context compaction, handoff, or an expected interruption, write a
  recovery checkpoint to the task's durable journal or task artifacts. Include
  the task, project, Wave, plan revision, and step identifiers; repository,
  worktree, branch, and base commit; current scope and acceptance criteria;
  files changed; decisions made; verification already run and its results;
  unresolved risks or blockers; and the next concrete action.
- Keep recovery evidence in the repository or control plane. Do not rely on chat
  history, transient terminal output, or model memory as the only record. Never
  store credentials or unsanitized secrets in a checkpoint.
- After compaction, restart, or reassignment, restore the workspace from ground
  truth before changing product code: reread the applicable `AGENTS.md` and
  `CLAUDE.md`, the frozen task, active Wave registry entry, approved plan, and
  Wave journal; then inspect the current worktree, branch, base commit, Git
  status and diff, changed files, and recorded verification evidence.
- Treat a recovery checkpoint as a navigation aid, not as authority. The current
  repository, frozen task, registry, approved plan, and control-plane state take
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
