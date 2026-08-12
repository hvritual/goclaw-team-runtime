# Team Control Connection Plan v3

- Plan-ID: `TC-W01-TEAM-CONTROL-001`
- Version: `v3`
- Status: `approved-for-execution`
- Supersedes: `plan_v2.md`
- Base commit: `8ca49704e720a545bd8a39436e74a1fc4608d9f6`
- Branch: `agent/tc-w01-team-control-001`
- Project-ID: `goclaw-team-runtime`
- Task-ID: `TC-W01-TEAM-CONTROL-001`
- Task-Revision: `r003`
- Policy bundle: `backend/AGENTS.md@ffe45b83ef3884d9b8bf66e6f7994b3a43d4f86e`

This immutable amendment inherits the goal, invariants, allowed and forbidden
paths, ordered steps, acceptance criteria, dependencies, risks, and rollback
from `plan_v1.md` and the single-authority membership decision from
`plan_v2.md`, except for the explicit changes below.

## Amendment D — resume and completion authority

The user explicitly resumed TC-W01 on 2026-08-12 and authorized completion of
S03 through S06, including merging Draft PR #9 after deterministic local and
rendered verification and independent product, code, security, and
documentation reviews have no blocking findings. This supersedes the v1
non-goal that prohibited automatic merge for this Wave.

The exact PR Head must be revalidated before every remote write and before
merge. The implementation author still cannot provide the independent review
or final acceptance evidence.

## Amendment E — CI is outside this execution

The user explicitly deferred CI work. TC-W01 S05 therefore must not add,
remove, or modify `.github/workflows/**`, required checks, branch protection,
or repository Rulesets.

S05 remains responsible for deterministic local verification:

- focused Core, Views, Web, and Desktop tests and type checks;
- Web and Desktop build coverage available in the local environment;
- local backend `make check` and `make test-race` when Go is available;
- local backend Docker build when Docker is available;
- Playwright coverage for project entry, projection rendering, command
  submission, CAS-conflict refresh, SSE invalidation, denied state, and a
  mobile-sized viewport;
- page identity, non-blank content, no framework overlay, console health,
  interaction, responsive layout, keyboard focus, accessible names, headings,
  and dialog titles;
- final diff inspection for `server/**`, credentials, secrets, and files
  outside the approved Wave scope.

Missing local runtimes must be reported as an explicit residual risk; they do
not authorize substituting or changing CI in this execution.

## Amendment F — delivery slices

S03 and S04 may be committed as separate atomic slices before S05/S06. Every
commit must use the traceability trailers from `backend/AGENTS.md` and identify
the active step. S06 freezes the final exact candidate only after all review
findings are either fixed and revalidated or recorded as non-blocking with
evidence.

## Active step

`TC-W01-S03` is active. S04, S05, and S06 remain ordered dependencies and may
not be marked complete early.
