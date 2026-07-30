-- name: ListCommentsForIssue :many
SELECT * FROM comment
WHERE issue_id = $1 AND workspace_id = $2
ORDER BY created_at ASC, id ASC
LIMIT $3;

-- name: GetCommentInWorkspace :one
SELECT * FROM comment
WHERE id = $1 AND workspace_id = $2;

-- name: CreateComment :one
INSERT INTO comment (
    issue_id, workspace_id, author_type, author_id, content, type, parent_id
) VALUES (
    $1, $2, 'member', $3, $4, $5, sqlc.narg('parent_id')
)
RETURNING *;

-- name: UpdateComment :one
UPDATE comment
SET content = $3, updated_at = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeleteComment :exec
DELETE FROM comment
WHERE id = $1 AND workspace_id = $2;

-- name: ResolveComment :one
UPDATE comment
SET resolved_at = now(),
    resolved_by_type = 'member',
    resolved_by_id = $3,
    updated_at = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: UnresolveComment :one
UPDATE comment
SET resolved_at = NULL,
    resolved_by_type = NULL,
    resolved_by_id = NULL,
    updated_at = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;
