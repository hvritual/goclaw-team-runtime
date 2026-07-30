-- name: ListIssues :many
SELECT i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
       i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
       i.parent_issue_id, i.position, i.start_date, i.due_date, i.created_at,
       i.updated_at, i.number, i.project_id, i.metadata, i.stage, i.properties
FROM issue i
WHERE i.workspace_id = $1
ORDER BY i.position ASC, i.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetIssue :one
SELECT * FROM issue
WHERE id = $1;

-- name: GetIssueInWorkspace :one
SELECT * FROM issue
WHERE id = $1 AND workspace_id = $2;

-- name: CreateIssue :one
INSERT INTO issue (
    workspace_id, title, description, status, priority,
    assignee_type, assignee_id, creator_type, creator_id,
    parent_issue_id, position, start_date, due_date, number, project_id,
    stage
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
    sqlc.narg('stage')
) RETURNING *;

-- name: GetIssueByNumber :one
SELECT * FROM issue
WHERE workspace_id = $1 AND number = $2;

-- name: UpdateIssue :one
UPDATE issue SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    assignee_type = sqlc.narg('assignee_type'),
    assignee_id = sqlc.narg('assignee_id'),
    position = COALESCE(sqlc.narg('position'), position),
    start_date = sqlc.narg('start_date'),
    due_date = sqlc.narg('due_date'),
    parent_issue_id = sqlc.narg('parent_issue_id'),
    project_id = sqlc.narg('project_id'),
    stage = sqlc.narg('stage'),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: LockIssueDuplicateKey :exec
SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0));

-- name: FindActiveDuplicateIssue :one
SELECT * FROM issue
WHERE workspace_id = $1
  AND status NOT IN ('done', 'cancelled')
  AND project_id IS NOT DISTINCT FROM sqlc.arg('project_id')::uuid
  AND parent_issue_id IS NOT DISTINCT FROM sqlc.arg('parent_issue_id')::uuid
  AND lower(btrim(regexp_replace(title, '[[:space:]]+', ' ', 'g'))) = sqlc.arg('normalized_title')
ORDER BY created_at ASC
LIMIT 1;

-- name: DeleteIssue :exec
DELETE FROM issue
WHERE id = $1 AND workspace_id = $2;

-- name: ListOpenIssues :many
SELECT i.id, i.workspace_id, i.title, i.description, i.status, i.priority,
       i.assignee_type, i.assignee_id, i.creator_type, i.creator_id,
       i.parent_issue_id, i.position, i.start_date, i.due_date, i.created_at, i.updated_at, i.number, i.project_id, i.metadata, i.stage, i.properties
FROM issue i
WHERE i.workspace_id = $1
  AND i.status NOT IN ('done', 'cancelled')
  AND (sqlc.narg('priority')::text IS NULL OR i.priority = sqlc.narg('priority'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR i.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('assignee_ids')::uuid[] IS NULL OR i.assignee_id = ANY(sqlc.narg('assignee_ids')::uuid[]))
  AND (sqlc.narg('creator_id')::uuid IS NULL OR i.creator_id = sqlc.narg('creator_id'))
  AND (sqlc.narg('project_id')::uuid IS NULL OR i.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('metadata_filter')::jsonb IS NULL OR i.metadata @> sqlc.narg('metadata_filter')::jsonb)
  -- properties_filter is a jsonb array of groups, each group an array of
  -- containment patterns (built by parsePropertiesFilterParam): the issue
  -- must match at least one pattern from EVERY group (AND of ORs). The
  -- correlated form skips the GIN index, which is fine here: open_only is
  -- an unpaginated workspace scan already narrowed by status.
  AND (
    sqlc.narg('properties_filter')::jsonb IS NULL
    OR NOT EXISTS (
      SELECT 1
      FROM jsonb_array_elements(sqlc.narg('properties_filter')::jsonb) AS pf(alternatives)
      WHERE NOT EXISTS (
        SELECT 1
        FROM jsonb_array_elements(pf.alternatives) AS alt(pattern)
        WHERE i.properties @> alt.pattern
      )
    )
  )
ORDER BY i.position ASC, i.created_at DESC;

-- name: ListChildIssues :many
-- Order by number ASC so sub-issues display in stable creation order
-- (oldest first), matching how a parent's plan reads top-to-bottom. The
-- position column is computed per-(workspace, status) by NextTopPosition,
-- not relative to siblings, so ordering by it interleaves children
-- unpredictably across batches and statuses; number is a per-workspace
-- monotonic counter and is sibling-stable.
SELECT * FROM issue
WHERE parent_issue_id = $1
ORDER BY number ASC;

-- name: ListChildrenByParents :many
-- Batched variant of ListChildIssues: returns all children for the given
-- parent set in one round trip. Used by Swimlane to avoid an N+1 fan-out
-- (one request per visible parent lane). Result is grouped client-side by
-- parent_issue_id; the workspace filter is also enforced so callers can't
-- enumerate children of parents in workspaces they don't belong to.
-- Within each parent, order by number ASC for the same sibling-stable
-- creation order as ListChildIssues.
SELECT * FROM issue
WHERE workspace_id = sqlc.arg('workspace_id')
  AND parent_issue_id = ANY(sqlc.arg('parent_ids')::uuid[])
ORDER BY parent_issue_id, number ASC;

-- name: ChildIssueProgress :many
SELECT parent_issue_id,
       COUNT(*)::bigint AS total,
       COUNT(*) FILTER (WHERE status IN ('done', 'cancelled'))::bigint AS done
FROM issue
WHERE workspace_id = $1
  AND parent_issue_id IS NOT NULL
GROUP BY parent_issue_id;

-- SearchIssues: moved to handler (dynamic SQL for multi-word search support).

-- name: SetIssueMetadataKey :one
-- Atomically sets a single key in the issue's metadata JSONB. The
-- workspace_id filter is the authorization gate — handler resolves the
-- issue first so this is also the tenant check.
UPDATE issue SET
    metadata = jsonb_set(metadata, ARRAY[sqlc.arg('key')::text], sqlc.arg('value')::jsonb),
    updated_at = now()
WHERE id = sqlc.arg('id') AND workspace_id = sqlc.arg('workspace_id')
RETURNING *;

-- name: DeleteIssueMetadataKey :one
-- Atomically removes a single key from the issue's metadata JSONB.
-- Deleting a missing key is a no-op (still returns the row).
UPDATE issue SET
    metadata = metadata - sqlc.arg('key')::text,
    updated_at = now()
WHERE id = sqlc.arg('id') AND workspace_id = sqlc.arg('workspace_id')
RETURNING *;
