# Milestone M1 — Canonical SQLite local runtime

## Outcome

Developers can start the existing Web frontend and a Canonical backend using a
local SQLite database, then complete the accepted Issue journey without running
the legacy server.

## User-visible journey

1. Start the local application with the repository-supported command.
2. Authenticate and restore the current user.
3. List and select an authorized Workspace.
4. View the Issue list and open an Issue detail page.
5. Read, put, and delete per-Issue metadata.
6. Observe the committed change after realtime refresh or reconnect.

## Delivery boundary

This is a thin compatible replacement milestone, not full product parity. It
includes only capabilities proven necessary by the journey frozen in `M1-S0`.
All backend product implementation belongs under `backend/**`; `server/**`
remains read-only evidence.

Request and response bodies are frozen. Endpoint URLs may change only through
the shared Core API client with explicit compatibility tests.

## Prerequisite gate

- Issue metadata v9 is scoped, committed, independently accepted, merged into
  `codex/multica-six-domain-baseline`, and synchronized locally.
- The merge is free of `server/**` changes and unrelated dirty files.
- The exact journey endpoint/event inventory and SQLite runtime ownership are
  approved in `M1-S0`.

## Release slices

| Order | Story | Demonstrable result | Promotion gate |
| --- | --- | --- | --- |
| 0 | `M1-S0` | Frozen compatibility/runtime contract | No critical `Unknown`; Human approval |
| 1 | `M1-S1` | Canonical process starts with real SQLite providers | Empty/restart/readiness tests |
| 2 | `M1-S2` | Login and trusted current-user identity work | Auth/session matrix passes |
| 3 | `M1-S3` | Authorized Workspace can be selected | Workspace isolation matrix passes |
| 4 | `M1-S4` | Issue list and detail load from Canonical SQLite | API, schema, and browser slice pass |
| 5 | `M1-S5` | Metadata read/write/delete works end to end | v9 parity and frontend tests pass |
| 6 | `M1-S6` | Committed changes refresh the correct client cache | Realtime ordering/reconnect tests pass |
| 7 | `M1-S7` | Canonical-only local startup is accepted | Cutover, rollback, review, customer acceptance |

Only one story may be active. Promotion changes `plan.md` and, when any frozen
contract changes, requires a new immutable `plan_vN.md`.

## Definition of Done

- The documented local command starts Canonical backend plus Web frontend on
  local SQLite.
- No legacy server process is needed for the accepted journey.
- Login, Workspace selection, Issue list/detail, and Issue metadata operations
  pass through Canonical code and real SQLite persistence.
- Body compatibility, authorization, tenant isolation, transactionality, and
  minimum realtime behavior are executable and green.
- Empty database, retained database, restart, and rollback evidence exist.
- The candidate diff contains no `server/**` path or unrelated dirty file.
- Independent review and Human Customer acceptance are recorded.

## Honest readiness labels

- `Proposed`: intent exists, contract is not frozen.
- `Story Ready`: scope, dependencies, acceptance, and tests are explicit.
- `Integrated`: deterministic checks for the story pass.
- `Milestone Accepted`: the entire browser journey and rollback pass, with
  independent review and Human Customer acceptance.

M1-S7 executable gates passed on 2026-08-14. Final milestone acceptance still
requires the independent post-live reviews and Human Customer acceptance
recorded by the active approved plan. Even after acceptance, the readiness
claim is limited to the frozen local Issue journey; it is not a general
replacement claim for every legacy surface.
