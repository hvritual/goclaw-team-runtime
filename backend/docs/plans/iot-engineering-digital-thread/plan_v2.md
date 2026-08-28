# IoT Engineering Digital Thread — Implementation Plan v2 Amendment

- Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`
- Version: `2`
- Status: **Approved / executing**
- Parent plan: `plan_v1.md`
- Scope: P1-S05 composition amendment only
- Canonical branch: `codex/multica-six-domain-baseline`

## Reason for amendment

P1-S05 in `plan_v1.md` correctly requires an explicit Workspace/Engineering contract and prohibits cross-context table access, but its allowed-path list omitted the composition root. A real consumer-owned seam cannot be injected without the canonical bootstrap graph. Avoiding the composition root would force either hidden globals or direct Workspace access to Engineering persistence, both of which violate the architecture.

This amendment does not change P1-S05 product semantics. It only permits the canonical composition root to bind the two bounded contexts through interfaces.

## P1-S05 — Work-plane linking, amended execution contract

### Allowed paths

- `backend/internal/modules/workspace/**`
- `backend/internal/modules/engineering/**`
- `backend/internal/bootstrap/**` **only** for interface composition, construction order, and integration tests
- `backend/docs/plans/iot-engineering-digital-thread/**`

`server/**` remains permanently read-only.

### Ownership

- Workspace owns the user-facing use case for associating work intent with engineering reality.
- Engineering owns canonical `ThreadEdge` storage, typed relation validation, provenance, and EngineeringEntity lifecycle checks.
- Bootstrap owns only dependency injection. It may not query Workspace or Engineering tables to implement the relation.

### Canonical work-link semantics

- `Project --changes--> EngineeringEntity`
- `Requirement --affects--> EngineeringEntity`
- `Task --affects--> EngineeringEntity`

The relation is derived from work kind. Clients cannot select an arbitrary relation.

The Six-Domain Task surface is backed by Workspace Todo identity; P1-S05 projects that same stable ID into Engineering as `NodeKindTask`. Runtime `Run` remains a different identity and is not linked by this use case.

### Provenance and authority

A link created through the Workspace use case is `authoritative` for the statement that the Workspace work item targets the EngineeringEntity because Workspace owns the work intent. Engineering generates canonical provenance with source `workspace`, a stable work-item locator, and observed time. User-controlled requests cannot forge authority or provenance.

### Lifecycle

- A new link requires both the Workspace work item and EngineeringEntity to exist in the same workspace.
- New links to an `archived` EngineeringEntity are rejected.
- Existing links are retained when a Project/Requirement/Task is deleted or archived. There are no FK/cascade actions between contexts.
- Existing links are retained if an EngineeringEntity is later archived so historical traceability is not destroyed.
- Only an explicit unlink operation removes the current work-to-engineering edge.
- Unlink does not delete either endpoint and does not modify historical Change/ContextPack records.

### Authorization

- Reads require a valid Workspace member.
- Link/unlink mutations require Workspace owner/admin in the first slice, matching Engineering P1-S04 write authority.
- Authorization is enforced in the Workspace application use case; the Engineering provider is an internal capability, not a second public bypass surface.

### Acceptance additions

In addition to `plan_v1.md` acceptance:

- boundary tests prove Workspace depends on an Engineering contract, not Engineering SQLite tables;
- Project/Requirement/Task existence is validated through Workspace-owned services/repositories;
- archived-target behavior and no-cascade retention are tested;
- Engineering edge persistence remains workspace-isolated and provenance-bearing;
- Task linking does not change Todo status, revision, DoneGate, or Runtime Run state;
- canonical `make check`, `make test-race`, frontend aggregation, and final `required` remain green.
