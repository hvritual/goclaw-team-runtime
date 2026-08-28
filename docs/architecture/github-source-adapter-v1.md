# GitHub Source Adapter v1

Phase-2 source adapter for the Engineering Digital Thread.

## Purpose

Read GitHub-owned facts without copying GitHub ownership into the control plane. The adapter is intentionally read-only and has no repository, graph, Change-acceptance, or Knowledge-publication side effects.

## Supported reads

- Repository identity and default branch from `GET /repos/{owner}/{repo}`.
- Exact commit resolution from `GET /repos/{owner}/{repo}/commits/{ref}`.
- `engineering.yaml` bytes at an immutable commit SHA from `GET /repos/{owner}/{repo}/contents/engineering.yaml?ref={sha}`.
- One pull request identity/state/head/base/merge metadata from `GET /repos/{owner}/{repo}/pulls/{number}`.

The adapter does not list repositories or PRs in v1. Discovery/reconciliation remains a separate concern.

## Immutable source rule

A branch or tag may be resolved to a commit using `ResolveCommit`, but manifest content is read only by `ReadEngineeringManifestAtCommit` and that method accepts a 40- or 64-hex immutable Git object ID. This prevents a Context/Reconcile operation from mixing metadata from two moving branch states.

## Security

- Bearer tokens are injected into the client and are never returned in data values or error strings.
- Response bodies are not copied into status errors.
- Redirects are rejected by a copied HTTP client so credentials are not forwarded through redirect chains.
- GitHub API base URL is explicit and validated; tests may inject an HTTP `httptest` endpoint.
- Manifest contents are bounded by the same 256 KiB limit as the repository manifest parser.

## Error contract

The adapter maps source failures into stable categories: not found, unauthorized, forbidden, rate limited, unexpected status, malformed payload, invalid locator/revision, unsupported contents object, and oversized manifest.

GitHub request IDs may appear in errors for diagnostics, but credentials and response bodies do not.

## Ownership boundary

P2-S02 returns source snapshots only. P2-S03 Reconciler will decide how a pinned repository/commit/manifest snapshot becomes `SourceBinding`, `EngineeringEntity`, and provenance-bearing `ThreadEdge` projections. The adapter never writes those records directly.
