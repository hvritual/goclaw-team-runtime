# Runtime Context Binding v1

P2-S07 binds one immutable Engineering `ContextPack` to one Runtime `Run` without moving ownership of Run attempts, leases, retries, completion, or human DoneGate.

## Boundary

Runtime owns the consumer contract:

```text
RuntimeContextPackReader
    ResolveFrozenContextPack(workspaceID, contextPackID)
```

Runtime does not read Engineering tables and the control-plane package does not import the Engineering bounded context. A composition adapter may satisfy this interface from Engineering or another trusted ContextPack service.

The ContextPack reader is authoritative only for the already-frozen pack identity. It cannot compile a new pack, accept a Change, publish Knowledge, or mutate Workspace work.

## Frozen execution identity

A contextual Run pins:

- ContextPack ID;
- ContextPack checksum;
- work-item kind, ID, and revision inherited from the frozen ContextPack;
- context-policy version inherited from the frozen ContextPack;
- Agent Release ID;
- zero or more sorted `(Skill ID, Skill Version ID)` audit pins.

The ContextPack ID/checksum is verified through `RuntimeContextPackReader` on first creation. Agent Release and Skill Version IDs are immutable audit references in v1; Runtime does not query System tables because the existing System contract does not yet expose a suitable read projection.

## Event model

Contextual queueing appends exactly three events under one kernel command/transaction:

```text
Run node (kind=run, state=queued)
    |
    | trace
    v
Run Context node (kind=run_context, revision=1, state=frozen)
```

The events are:

1. `work.node.upserted.v1` for the Run;
2. `work.node.upserted.v1` for the frozen Run Context;
3. `work.edge.added.v1` for the trace edge.

`KernelStore.AppendCommand` persists all proposed events atomically, so a Run cannot be created with only part of its contextual binding.

## Why context is not embedded in RunData

Existing `ClaimRun`, `HeartbeatRun`, `CompleteRun`, `CancelRun`, and `RetryRun` deserialize and rewrite `RunData`. Embedding the ContextPack pin in that mutable payload would couple immutable execution identity to mutable attempt state and risk accidental loss during lifecycle updates.

The separate `run_context` node avoids that coupling. Normal Run lifecycle commands update only the `run` node. The context node remains `revision=1/state=frozen` for replay and audit.

## Idempotency and immutability

The original kernel command ID is the only idempotent replay identity. If its receipt already exists, Runtime can replay the stored result without contacting the ContextPack provider again.

A different command ID cannot reuse an existing Run ID, Run Context ID, or context trace. Rebinding a Run to another ContextPack, Agent Release, Skill Version set, or work revision therefore conflicts rather than appending a second execution identity.

## Legacy compatibility

Existing `QueueRun` remains unchanged and creates a legacy Run with no `run_context` trace. `ResolveRunExecutionContext` returns `found=false` for such Runs. P2-S07 adds contextual execution without changing existing clients or Run lifecycle behavior.

## Governance invariants

P2-S07 does not change:

- Workspace Project/Requirement/Task ownership;
- Todo/Task acceptance semantics;
- Runtime attempt, lease, heartbeat, retry, cancellation, or completion rules;
- human DoneGate authority;
- Change acceptance;
- Knowledge publication;
- System Agent Release / Skill publication ownership.

## Acceptance status

This implementation is staging-only on `agent/iot-edt-p2-staging`. Canonical P1-EXIT and P2 acceptance remain blocked while GitHub-hosted Actions jobs fail before runner allocation (`runner_id=0`, no executed steps). This document does not claim canonical CI acceptance.
