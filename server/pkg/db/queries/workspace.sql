-- name: ListWorkspaces :many
SELECT w.id, w.name, w.slug, w.description, w.settings,
       w.created_at, w.updated_at, w.context, w.repos,
       w.issue_prefix, w.issue_counter, w.avatar_url
FROM member m
JOIN workspace w ON w.id = m.workspace_id
WHERE m.user_id = $1
ORDER BY w.created_at ASC;

-- name: GetWorkspace :one
SELECT * FROM workspace
WHERE id = $1;

-- name: GetWorkspaceBySlug :one
SELECT * FROM workspace
WHERE slug = $1;

-- name: CreateWorkspace :one
INSERT INTO workspace (name, slug, description, context, issue_prefix)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateWorkspace :one
UPDATE workspace SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    context = COALESCE(sqlc.narg('context'), context),
    settings = COALESCE(sqlc.narg('settings'), settings),
    repos = COALESCE(sqlc.narg('repos'), repos),
    issue_prefix = COALESCE(sqlc.narg('issue_prefix'), issue_prefix),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: IncrementIssueCounter :one
UPDATE workspace SET issue_counter = issue_counter + 1
WHERE id = $1
RETURNING issue_counter;

-- name: LockWorkspaceForDelete :one
SELECT id FROM workspace WHERE id = $1 FOR UPDATE;

-- name: LockWorkspaceForMemberMutation :one
SELECT id FROM workspace WHERE id = $1 FOR UPDATE;

-- name: DeleteWorkspace :exec
WITH ws_skills AS (
    SELECT id FROM skill WHERE workspace_id = $1
),
cleared_skill_labels AS (
    DELETE FROM skill_to_label WHERE skill_id IN (SELECT id FROM ws_skills)
),
cleared_project_resources AS (
    DELETE FROM project_resource WHERE workspace_id = $1
),
cleared_issue_properties AS (
    DELETE FROM issue_property WHERE workspace_id = $1
),
cleared_tasks AS (
    DELETE FROM task WHERE workspace_id = $1
)
DELETE FROM workspace WHERE workspace.id = $1;
