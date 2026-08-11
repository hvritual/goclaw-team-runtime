# Governance and Architecture Drift Repair Journal

## 2026-08-12 — GOV-W01-S00 activated

- User explicitly prioritized the governance and architecture drift repair.
- Canonical integration base is
  `codex/multica-six-domain-baseline@c43f4300eb29cf6778e67594e54cc79f8fb5057e`.
- TC-W01 remains isolated on Draft PR #9 at the time of planning.
- Confirmed root `AGENTS.md` and `CLAUDE.md` still grant an approved-plan
  exception for `server/**`, conflicting with `backend/AGENTS.md`.
- Confirmed `CONTEXT-MAP.md` still leaves execution ownership unresolved
  despite the P2 Run/Lease/Heartbeat/Retry implementation.
- Confirmed root CI targets only `main`; the separate Backend workflow targets
  pull requests to the canonical branch and has no canonical push trigger.
- Local workspace has no repository checkout or GitHub CLI. Repository content,
  branch, commits, PR, and CI evidence are handled through the connected GitHub
  repository. Remote branch-protection mutation is a hard gate unless the
  available repository interface exposes it.
- Created isolated branch `agent/gov-w01-governance-drift-001`.
- Next action: publish this plan, then activate `GOV-W01-S01`.

## 2026-08-12 — plan v2 activated; GOV-W01-S01 active

- Context-source inspection found `docs/contexts/system/CONTEXT.md` also says
  execution ownership is unresolved.
- Plan v2 adds that exact documentation path to S02; no product or runtime scope
  changed.
- Root rules remain the active first repair step.

## 2026-08-12 — GOV-W01-S01 through S03 verified; S04 blocked

- Root `AGENTS.md` and `CLAUDE.md` now make `server/**` permanently
  read-only without plan-level exceptions.
- Execution/Runtime is established in the Context Map and dedicated context
  document; System and canonical backend vocabulary are aligned.
- Root CI now validates PRs and pushes for `main` and the canonical branch,
  owns the repository-wide server boundary, and runs canonical backend check
  plus race.
- Duplicate `.github/workflows/backend.yml` is removed.
- Unified CI exposed and repaired one pre-existing frozen-lockfile mismatch in
  the `packages/views` importer.
- Canonical CI no longer attempts absent legacy `server/**` or
  `apps/mobile` paths. Legacy runtime jobs remain main-only reporting.
- Candidate `7415c4e32346a1a89db966130ad2c0e4e577d6ad` passed GitHub CI Run
  `31546015838`: governance-policy, canonical-backend check/race, frontend
  build/typecheck/lint, frontend tests, frontend aggregate, and `required`.
- PR #10 is Draft and contains no `server/**` diff.
- Remote branch-protection/ruleset mutation is unavailable through the connected
  repository interface. The exact required configuration is recorded in
  `docs/governance/CANONICAL-BRANCH-PROTECTION.md`; documentation alone does
  not satisfy S04.
- Next action: apply and inspect the remote `canonical-integration` ruleset
  requiring `CI / required`, then obtain independent review before merge.
