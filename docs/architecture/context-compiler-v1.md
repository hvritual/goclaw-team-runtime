# Context Compiler v1

Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`
Step: `P2-S06`

## Purpose

Context Compiler v1 freezes a reproducible manifest of authoritative engineering scope plus governed references for one immutable work-item revision. It does not copy source or knowledge bytes into Engineering storage and it does not bind the pack to Runtime yet.

Pipeline:

`WorkItem revision -> Scope Resolver -> pinned source selection -> governed references -> accepted Changes -> incident hook -> deterministic ranking/budget -> immutable ContextPack`.

## Inputs

A compile request contains:

- workspace ID;
- ContextPack ID;
- typed Project/Requirement/Task identity;
- immutable work-item revision;
- versioned compile policy.

The policy controls graph depth/entity limits, source/knowledge freshness windows, maximum reference count, estimated token budget and maximum recent accepted Changes.

## Scope ownership

Scope is supplied exclusively by P2-S05. Explicit authoritative Workspace work links are the seed. Context Compiler never invents scope from text similarity, PR titles, branch names or LLM inference.

## Source pinning

For every scoped EngineeringEntity, the compiler considers only `authoritative` or `observed` SourceBindings that contain a non-empty immutable revision.

Selection order is deterministic:

1. pinned source only;
2. fresh before stale;
3. authoritative before observed when freshness is equal;
4. newer observation time;
5. stable binding ID.

The selected source is frozen as an `engineering_entity` ContextReference whose revision is the source revision and whose checksum covers entity ID, binding ID, source type, locator, revision and authority.

Pinned source references are required selections. If the configured reference/token budget cannot retain them, compilation fails with a budget error rather than silently producing a less reproducible pack. Entities without a usable pin remain in `target_entity_ids` and produce `source_unpinned` warnings.

## Governed knowledge/reference hook

Engineering consumes a provider-neutral `PublishedContextReferenceReader`. The provider owns publication governance and content bytes. The compiler accepts only references with:

- supported kind (`architecture`, `adr`, `standard`, `runbook`, `knowledge`, `requirement`);
- stable ID;
- non-empty revision;
- non-empty checksum;
- explicit overlap with resolved EngineeringEntity scope, unless declared global.

The compiler never promotes Knowledge status and never invents a checksum for external governed content. Stale governed references remain eligible but receive a deterministic score penalty plus `reference_stale` warning.

## Recent Change hook

Recent Changes come from the Engineering repository itself. Only `accepted` Changes with an accepted timestamp are eligible. Proposed, rejected and superseded Changes are not included as current context. Change references use the accepted timestamp as revision and a canonical SHA-256 checksum over the accepted Change fields, sorted artifacts and provenance.

## Incident hook

Incident references are supplied through a separate provider-neutral reader. They must be pinned by ID/revision/checksum and overlap current scope (or be explicitly global). P2-S06 does not own Incident lifecycle.

## Ranking

Required pinned source references are selected first. Optional reference base priority is:

1. Standard
2. Architecture
3. ADR
4. Requirement
5. Runbook
6. Incident
7. accepted Change
8. generic governed Knowledge

Providers may add a bounded integer priority. Stale governed references receive a penalty. Stable `(kind,id,revision,checksum)` ordering breaks ties.

## Token-budget boundary

V1 stores metadata references rather than content bytes, but every optional reference carries an estimated materialization token cost. Selection stops at both `MaxReferences` and `MaxEstimatedTokens`; omitted optional references produce `reference_limit` or `token_budget` warnings.

This is a selection boundary, not a tokenizer implementation. Source content materialization remains deferred.

## Determinism and immutability

The resulting ContextPack checksum depends on:

- workspace and work-item identity/revision;
- sorted target EngineeringEntity IDs;
- sorted reference kind/ID/revision/checksum tuples;
- policy version.

Pack ID and creation timestamp do not affect content checksum. Recompiling the same semantic inputs under a different Pack ID yields the same checksum. Reusing an existing Pack ID with different content is rejected as a conflict; identical replay returns the frozen existing pack.

## Warning model

The compiler preserves P2-S05 warnings and adds:

- `source_unpinned`;
- `reference_invalid`;
- `reference_stale`;
- `reference_limit`;
- `token_budget`.

Warnings never silently upgrade lower-authority information into authoritative context.

## Boundaries retained

- no embeddings or vector-first scope discovery;
- no source/knowledge body duplication;
- no HTTP/MCP compile endpoint in P2-S06;
- no Runtime binding before P2-S07;
- no autonomous Change acceptance;
- no autonomous Knowledge publication;
- no `server/**` changes.
