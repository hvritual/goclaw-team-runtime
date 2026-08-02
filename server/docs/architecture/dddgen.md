# Native dddgen scaffold

The backend now carries the native modular-monolith foundation for the accepted
`workspace`, `auth`, `space`, and `system` contexts.

## Ownership

- `api/` owns Proto services, Kratos HTTP annotations, and access metadata.
- `gen/` is replaceable generated output for Go, gRPC, HTTP, OpenAPI, and the
  access manifest.
- `internal/modules/<module>/contract` is the only package another module may
  import.
- `internal/modules/<module>/internal` contains domain, application,
  infrastructure, and interface implementations.
- `internal/platform/module` supports generated extension-service registration.
- `internal/bootstrap` is the generated module composition root.
- `internal/architecturetest` enforces inward imports and contract-only edges.

## Commands

```sh
make bootstrap
make generate
make generated-clean
make vet-ddd
make lint-ddd
make test-race-ddd
```

`make bootstrap` installs public generators into `server/bin/ddd-tools` and
stages the separately distributed `dddgen` and `protoc-gen-access` binaries.
The Kratos HTTP generator is pinned to v3; a v2 plugin produces incompatible
transport imports and must not be used.

## Runtime cutover

The generated modules are compile- and contract-tested foundations. They are
not registered in the current Chi server. Existing production handlers move to
them only through separately approved tracer slices that preserve API,
authorization, workspace, event, transaction, PostgreSQL, and SQLite behavior.

The generated Space PostgreSQL provider is currently a composition seam, not a
finished persistence implementation. The existing upload slice remains under
`server/modules/space` until its dedicated native-module cutover is implemented
and verified.
