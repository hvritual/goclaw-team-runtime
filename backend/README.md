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

## Team Control identity

Production requests must set `CONTROLPLANE_IDENTITY_UPSTREAM_URL` to the
absolute origin of the existing Multica API. The control plane forwards only
the incoming `Authorization` bearer value or `multica_auth` cookie to
`/api/me`, `/api/workspaces/{id}`, and `/api/workspaces/{id}/members`.
Redirects, incomplete member snapshots, and upstream failures fail closed.

Cookie-authenticated mutations also require the existing `multica_csrf` value
in `X-CSRF-Token`. `CONTROLPLANE_ALLOW_HEADER_IDENTITY=true` is an explicit
local-development escape hatch and must not be enabled in production. With
neither setting, business endpoints deny all requests.

The v1 contract is in `openapi/team-control.v1.yaml`. Project updates stream
from `/v1/workspaces/{workspace}/projects/{project}/events` using SSE and may
resume with `after` or `Last-Event-ID`.

## Commands

```bash
make check
make test-race
make policy-check BASE_REF=codex/multica-six-domain-baseline
make run
docker build -t goclaw-controlplane:tc-w01 .
```

`make check` is the backend-local deterministic gate. Repository CI wiring is
explicitly deferred from TC-W01 by user direction and remains a follow-up item;
this slice does not modify root workflows or treat deferred CI as a merge gate.
