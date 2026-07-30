-- name: ListTasks :many
SELECT *
FROM task
WHERE workspace_id = sqlc.arg('workspace_id')
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('issue_id')::uuid IS NULL OR issue_id = sqlc.narg('issue_id'))
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY position ASC, created_at DESC;

-- name: GetTask :one
SELECT *
FROM task
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id');

-- name: CreateTask :one
INSERT INTO task (
    workspace_id,
    project_id,
    issue_id,
    title,
    description,
    status,
    priority,
    assignee_id,
    creator_id,
    position,
    start_date,
    due_date
) VALUES (
    sqlc.arg('workspace_id'),
    sqlc.narg('project_id'),
    sqlc.narg('issue_id'),
    sqlc.arg('title'),
    sqlc.arg('description'),
    sqlc.arg('status'),
    sqlc.arg('priority'),
    sqlc.narg('assignee_id'),
    sqlc.arg('creator_id'),
    sqlc.arg('position'),
    sqlc.narg('start_date'),
    sqlc.narg('due_date')
)
RETURNING *;

-- name: UpdateTask :one
UPDATE task
SET
    project_id = sqlc.narg('project_id'),
    issue_id = sqlc.narg('issue_id'),
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    assignee_id = sqlc.narg('assignee_id'),
    position = COALESCE(sqlc.narg('position'), position),
    start_date = sqlc.narg('start_date'),
    due_date = sqlc.narg('due_date'),
    completed_at = CASE
        WHEN sqlc.narg('status')::text = 'done' THEN COALESCE(completed_at, now())
        WHEN sqlc.narg('status')::text IS NOT NULL THEN NULL
        ELSE completed_at
    END,
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id')
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM task
WHERE id = sqlc.arg('id')
  AND workspace_id = sqlc.arg('workspace_id');
