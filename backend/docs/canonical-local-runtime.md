# Canonical local runtime

The Canonical acceptance profile is explicit. It never seeds during normal
startup and never deletes either runtime database or mode-specific log.

Use the portable Node commands from the repository root:

```text
node scripts/runtime-selector.mjs select legacy
node scripts/runtime-selector.mjs select canonical --dry-run
node scripts/runtime-selector.mjs select canonical
node scripts/runtime-selector.mjs status
node scripts/runtime-selector.mjs start
```

Canonical startup owns Web `127.0.0.1:3000`, HTTP `127.0.0.1:8000`, gRPC
`127.0.0.1:9000`, and `data/multica-canonical.db`. It fails closed if legacy
port `8080` remains active. PID manifests and logs live under `.local-runtime/`.

After a first migration-only start, explicitly seed the acceptance fixture:

```text
node scripts/runtime-selector.mjs stop
cd backend
go run ./cmd/canonical-fixture -sqlite-path ../data/multica-canonical.db
cd ..
node scripts/runtime-selector.mjs start
```

The fixture reports `created` or `already_present`; partial/conflicting data is
rejected and existing Issue metadata is never reset. Login uses
`canonical-fixture@multica.local`, code `888888`, Workspace
`canonical-fixture`, and Issue `CAN-1`.

Restart/readback proof uses:

```text
node scripts/canonical-runtime-verifier.mjs mutate
node scripts/runtime-selector.mjs stop
node scripts/runtime-selector.mjs start
$env:CANONICAL_READBACK_VALUE="<printed-value>"
node scripts/canonical-runtime-verifier.mjs readback
pnpm exec playwright test e2e/canonical-runtime.spec.ts
```

The browser run writes a sanitized `canonical-network-trace.json` attachment
under Playwright's `test-results/` output. It records HTTP method/origin/path,
WebSocket origin/path, and received event types only; cookies, tokens, headers,
and response bodies are excluded.

Rollback stops manifest-owned processes, waits for exit, preserves databases
and logs, and restores the previous selector:

```text
node scripts/runtime-selector.mjs stop
node scripts/canonical-runtime-verifier.mjs snapshot
node scripts/runtime-selector.mjs rollback
node scripts/canonical-runtime-verifier.mjs preserved
node scripts/runtime-selector.mjs status
```

If PID identity no longer matches the manifest, stop fails closed instead of
killing an unrelated process. Inspect the manifest and logs before resolving
stale state manually.
