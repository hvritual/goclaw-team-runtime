# Canonical backend

This directory is the only writable backend implementation root. The legacy
`server/` tree is read-only migration evidence and is never a change target.

## Architecture vocabulary

The runtime currently assembles four platform bounded contexts:

1. Workspace
2. Auth
3. Space
4. System

The "six-domain" delivery baseline refers to six cross-context capabilities
built on that platform, not six top-level bounded contexts:

1. Foundation: workspace identity, membership, authorization, audit, storage
2. Delivery kernel: event history, replay, evidence, Work Graph, DoneGate
3. Requirement: request, intent, solution, reviews, freeze, task
4. Quality: defect, risk, verification, close gates
5. Review and knowledge: findings, candidates, publication, evaluation
6. Execution: queue, lease, runner, evidence return, human termination

The control-plane process exposes these capabilities without allowing a Runner
or model review to advance authoritative acceptance state.

## Commands

```bash
make check
make test-race
make policy-check BASE_REF=codex/multica-six-domain-baseline
make run
docker build -t goclaw-controlplane:p0p2 .
```

`make check` is the backend-local deterministic gate. Repository CI must call
it from a separately authorized root workflow change; until then, CI wiring is
an explicit path blocker rather than a completed gate.
