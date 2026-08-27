# Product capability roadmap v45 — Release 4 S08A bounded-input successor draft

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `45`
- Task-ID: `PCR-001-S08A-R050`
- Task-Revision: `r050`
- Work-Item: `PCR-S08A`
- Title: `Bound Issue-similarity normalized input before query construction`
- Exact base: `f412279976c74df575cb8038c64f5474cfdda25d`
- Integration branch: `codex/release4-s08a-integrate`
- Predecessor: immutable `plan_v44.md` / r049 is review-blocked by the independent code/security/quality finding that unbounded normalized terms can reach the SQLite similarity query
- Frozen policy bundle: `CLAUDE.md` SHA-256 `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646` / `backend/AGENTS.md` SHA-256 `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`
- Status: `draft — R50.1 independent plan review pending; product code unauthorized`
- Authority: The Human Customer confirmed selective S08A integration into `codex/multica-six-domain-baseline` on 2026-08-27. This mandatory successor corrects only the independently reported bounded-input defect and does not authorize a broader Release 4 merge, push, deployment, or legacy-tree change.

## Review blocker and goal

The r049 candidate normalizes title and description before constructing an FTS-like SQLite predicate, but it does not limit normalized string size or token count. A very large title or description can create an unbounded number of predicate fragments and query bindings.

R50 retains the warning-only S08A contract and adds an application-layer boundary before authorization or repository invocation:

- normalized title: at most 1,024 Unicode runes and 32 whitespace-delimited terms;
- normalized description: at most 4,096 Unicode runes and 32 whitespace-delimited terms; and
- an oversized field returns `contract.ErrInvalidIssueSimilarity`; the existing HTTP mapping returns 400.

An empty normalized description remains valid. No candidate ranking, workspace isolation, create-Issue availability, endpoint shape, migration, generated contract, or user-visible success behavior changes.

## Dependencies, invariants, risks, and rollback

R50 depends on exact r049 candidate `f412279976c74df575cb8038c64f5474cfdda25d`, its proven deterministic gates, the named policy hashes, the existing typed `ErrInvalidIssueSimilarity` mapping, and a clean isolated integration worktree. It has no external-service, deployment, database, migration, or legacy-tree dependency.

`server/**` remains permanently read-only. Generated protobufs, migrations, S08B/S09, source-roadmap history, push, deployment, and every pre-existing canonical dirty path remain excluded. The application layer remains the authoritative pre-repository validation boundary; invalid input must not call either authorizer or repository.

Risk: an incorrect count could reject valid content or leave the query unbounded. Mitigation: assertion-first title/description rune and term tests, including proof of zero authorizer/repository calls, plus an HTTP 400 proof. If a test, static gate, review, identity check, or scope check fails, stop and leave the canonical branch unchanged. Because no migration or runtime action is authorized, rollback is simply abandoning this isolated candidate; any material change needs another immutable successor.

## Scope and traceability

R50 may change only:

~~~text
backend/internal/modules/workspace/internal/application/issue_similarity.go
backend/internal/modules/workspace/internal/application/issue_similarity_test.go
backend/internal/modules/workspace/internal/interfaces/http/issue_similarity_test.go
backend/docs/plans/product-capability-roadmap/plan_v45.md
backend/docs/plans/product-capability-roadmap/plan.md
backend/docs/plans/product-capability-roadmap/story-map.md
backend/docs/plans/product-capability-roadmap/task-register.md
backend/docs/plans/product-capability-roadmap/journal.md
~~~

Every R50 commit must contain continuous Git trailers: `Task-ID: PCR-001-S08A-R050`, `Project-ID: PRODUCT-CAPABILITY-ROADMAP`, `Task-Revision: r050`, `Work-Item: PCR-S08A`, `Plan-ID: PRODUCT-CAPABILITY-ROADMAP-001`, `Plan-Version: 45`, a specific `Plan-Step`, `Issue: PCR-S08A`, and the frozen `Policy-Bundle` value above. No unrelated repair may be mixed into R50.

## Ordered execution and acceptance

### R50.1 — Freeze and independently review this successor plan

Record the exact plan, base, policy, predecessor, write boundary, risks, and required checks in the mutable registers. Obtain a fresh independent SPEC PASS and `git diff --check` pass before any product byte changes. Revalidate plan/policy/base identity at review, candidate freeze, and canonical fast-forward.

### R50.2 — Write assertion-first boundary tests (RED)

Add application tests for oversized title, oversized description, and excess terms; each must return `ErrInvalidIssueSimilarity` before the authorizer or repository is invoked. Add direct HTTP proof that the typed error is 400. Run the focused new application test first and retain its failing output before implementation.

### R50.3 — Minimal application-layer implementation (GREEN)

Add only bounded normalized-input validation in `CheckIssueSimilarity`, before authorization/repository work. Run the same focused tests until GREEN. Do not change the HTTP handler or repository implementation.

### R50.4 — Focused deterministic gates

Run relevant Workspace application and HTTP tests, then `cd backend && go test ./internal/modules/workspace/...`. Verify formatting, exact path scope, no `server/**`, no generated path, and no canonical-dirty overlap.

### R50.5 — Candidate regression gates

Run `pnpm typecheck`, `pnpm test`, `cd backend && go test ./...`, `cd backend && make check`, and `cd backend && make test-race`. Record any actual non-pass without substitution or retry-based concealment.

### R50.6 — Fresh independent dual review

After candidate identity/scope freeze, obtain separate independent SPEC and code/security/quality reviews. Both must PASS with no unresolved blocker.

### R50.7 — Fast-forward reviewed candidate

Only if `codex/multica-six-domain-baseline` remains at the exact ancestor and its unrelated dirt does not overlap candidate paths, fast-forward it to the reviewed candidate. Do not push or deploy. Record the real canonical HEAD and preserved dirty status.
