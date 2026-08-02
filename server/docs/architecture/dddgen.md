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
make verify-ddd-tools
make generate
make generated-clean
make vet-ddd
make lint-ddd
make test-race-ddd
```

`make bootstrap` installs every public generator into `server/bin/ddd-tools` at
an explicit version. `dddgen` and `protoc-gen-access` remain private,
externally distributed binaries from `github.com/fworld/go-ddd-scaffold`;
bootstrap locates them through `DDDGEN` and `PROTOC_GEN_ACCESS`, stages them,
and then validates their shared module version pinned in the Makefile instead
of vendoring generator source. `make verify-ddd-tools` reads the Go build
metadata from both staged binaries and fails if either package or module
version differs from that pin. This check also runs before generation, so a
stale binary cannot silently rewrite owned artifacts. The Kratos HTTP generator
is pinned to v3; a v2 plugin produces incompatible transport imports and must
not be used.

If the distributed binaries are not on `PATH`, pass their explicit locations:

```sh
make bootstrap DDDGEN=/path/to/dddgen \
  PROTOC_GEN_ACCESS=/path/to/protoc-gen-access
```

When upgrading the scaffold, change `DDD_SCAFFOLD_VERSION`, run `make
bootstrap` with externally distributed binaries built from that version,
inspect the complete reconciled/generated diff, run `make generated-clean`,
and commit the version bump with all resulting artifacts. An ambient binary
with missing or mismatched Go build metadata is rejected.

`make generate` ends with the repository-owned `cmd/postprocess-generated`
step. It reads Proto descriptors rather than duplicating route policy and
normalizes two gaps in the pinned upstream generators:

- RPCs annotated with `annotations.v1.http_success_status` emit the declared
  2xx status in the Kratos server, OpenAPI response, and generated HTTP client.
  A 204 client consumes the empty body without asking ProtoJSON to decode it.
- Generated client path placeholders use Proto JSON field names so Kratos
  `BuildPath` can bind snake_case Proto fields such as `workspace_id`.
- RPCs using `google.api.http.response_body` keep their public unwrapped JSON
  shape in the Kratos server/client and OpenAPI schema; response-body clients
  negotiate standard JSON because a repeated field is not itself a Proto
  message. Explicitly optional scalar fields in repeated message elements keep
  their JSON key when absent, so nullable fields remain `null` rather than
  disappearing from the established public response.

Treat this postprocessor as part of the native generation pipeline. Change its
tests and contract coverage together with any new transport normalization; do
not patch the generated files by hand.

## Runtime cutover

The generated modules are compile- and contract-tested foundations. They are
not registered in the current Chi server. Existing production handlers move to
them only through separately approved tracer slices that preserve API,
authorization, workspace, event, transaction, PostgreSQL, and SQLite behavior.

The generated Space PostgreSQL provider is currently a composition seam, not a
finished persistence implementation. The existing upload slice remains under
`server/modules/space` until its dedicated native-module cutover is implemented
and verified.
