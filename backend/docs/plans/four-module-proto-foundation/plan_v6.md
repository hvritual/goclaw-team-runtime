# Bootstrap and runnable transport entrypoint

- Plan-ID: `four-module-proto-foundation`
- Version: `6`
- Status: `approved`
- Approval source: user instruction dated 2026-08-02 to add bootstrap and
  HTTP/gRPC entrypoints and start the backend service
- Base commit: `0c4f848a8458e256dd4fe2ed51498af969aa3c59`
- Repository: `/Users/fworld/Hvritual/goclaw`
- Branch: `codex/multica-six-domain-baseline`
- Task type: `change`

## Goal

Add a minimal standalone composition root and executable that register all four
generated modules with Kratos v3 HTTP and gRPC transports, then start and probe
the backend service.

## Scope

- Add `backend/internal/bootstrap` for four-module assembly and transport
  registration.
- Add a validated runtime configuration for service name/version and HTTP/gRPC
  TCP listen addresses.
- Add HTTP `/healthz` and `/readyz` endpoints.
- Use Kratos v3's built-in gRPC health and reflection services.
- Add `backend/cmd/server` with flags:
  - `-http-addr` default `127.0.0.1:8000`
  - `-grpc-addr` default `127.0.0.1:9000`
  - `-name` and `-version`
- Add bootstrap and CLI configuration tests.
- Start the service on the default addresses and verify live HTTP responses and
  the gRPC listener.

## Non-goals

- No persistence selection, database, business stub implementation, auth
  enforcement, legacy HTTP compatibility adapter, OpenAPI/access manifest, or
  installed `server/` cutover.
- No changes outside `backend/`.

## Invariants

- The composition root depends on module roots only, never module internals.
- All four modules register on both transports.
- Extension services without HTTP annotations remain gRPC-only.
- Workspace ownership/isolation contracts remain unchanged.
- Startup and shutdown are owned by the Kratos application lifecycle.

## Dependencies

- Completed P5 generated module and `rpc/pb` trees.
- Kratos v3 HTTP/gRPC servers already declared in `backend/go.mod`.
- Default ports 8000 and 9000 were confirmed free before implementation.

## Ordered steps

### P6-S1 — Implement, verify, and start the standalone backend

Implement bootstrap assembly, runtime construction, health endpoints, CLI
configuration, and tests. Run formatting, Buf, Go test/vet, architecture and
scope gates. Start the process, probe health/readiness and module Ping routes,
confirm the gRPC listener, and leave the verified process running.

## Acceptance criteria

1. `internal/bootstrap` registers Workspace, Auth, Space, and System modules.
2. HTTP and gRPC servers start on independently configurable addresses.
3. Health/readiness and four primary module HTTP Ping routes return success.
4. gRPC health/reflection and all generated module services are registered.
5. `go test ./...`, `go vet ./...`, and existing generation gates pass.
6. The launched process remains listening on the reported addresses.
7. No path outside `backend/` changes.

## Deterministic verification

Run `gofmt`, Buf format/lint, `go mod verify`, `go test ./...`, and
`go vet ./...`. Tests inspect HTTP responses and the registered gRPC service
catalog. Start `go run ./cmd/server`, probe HTTP endpoints with curl, inspect the
gRPC TCP listener, and rerun the repository scope/artifact checks.

## Risks

- A port may become occupied between preflight and startup; startup must fail
  clearly and may be retried on explicit alternate flags.
- Generated application methods are still not implemented; only primary Ping
  and health endpoints are expected to succeed.
- A process left running can become stale after later code changes; the final
  response must report the PID/session addresses and current probe evidence.

## Rollback

Stop the standalone backend process and remove only `backend/internal/bootstrap`
and `backend/cmd/server`. Generated contracts/modules remain intact. Do not
touch the installed `server/` runtime.

