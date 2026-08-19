CREATE TEMP TABLE project_requirement_rollback_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO project_requirement_rollback_guard(safe)
SELECT CASE WHEN
    EXISTS (
        SELECT 1
        FROM workspace_requirement_baselines b
        LEFT JOIN workspace_requirements r ON r.id = b.legacy_requirement_id
        WHERE r.id IS NULL
           OR b.id <> r.id
           OR b.workspace_id <> r.workspace_id
           OR b.project_id <> r.project_id
           OR b.status <> 'draft'
           OR b.current_revision <> r.current_version
           OR b.approved_revision IS NOT NULL
           OR b.effective_revision IS NOT NULL
           OR b.review_origin IS NOT NULL
           OR b.latest_content_author <> 'legacy-import'
           OR b.submitted_by IS NOT NULL OR b.submitted_at IS NOT NULL
           OR b.approved_by IS NOT NULL OR b.approved_at IS NOT NULL
           OR b.frozen_by IS NOT NULL OR b.frozen_at IS NOT NULL
           OR b.retired_by IS NOT NULL OR b.retired_at IS NOT NULL
           OR b.legacy_snapshot_json IS NULL
           OR b.legacy_snapshot_json <> json_object(
               'id',r.id,
               'workspace_id',r.workspace_id,
               'project_id',r.project_id,
               'title',r.title,
               'current_version',r.current_version,
               'approval_status',r.approval_status,
               'coverage_status',r.coverage_status,
               'issue_ids',r.issue_ids,
               'created_at',r.created_at,
               'updated_at',r.updated_at
           )
           OR b.created_at <> r.created_at
           OR b.updated_at <> r.updated_at
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirements r
        LEFT JOIN workspace_requirement_baselines b ON b.legacy_requirement_id = r.id
        WHERE b.id IS NULL
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirement_revisions c
        LEFT JOIN workspace_requirement_baselines b ON b.id = c.baseline_id
        LEFT JOIN workspace_requirements r ON r.id = b.legacy_requirement_id
        LEFT JOIN workspace_requirement_versions v
          ON v.requirement_id = r.id AND v.version = c.revision
        WHERE v.id IS NULL
           OR c.content_json <> json_object(
               'problem_statement',v.content,
               'goals',json_array(json_object('key','legacy-root','text',r.title)),
               'in_scope',json('[]'),
               'out_of_scope',json('[]'),
               'constraints',json('[]'),
               'acceptance_criteria',json('[]'),
               'dependencies',json('[]')
           )
           OR c.status <> 'draft'
           OR c.action <> 'legacy_import'
           OR c.change_summary <> 'Imported legacy Requirement version'
           OR c.actor_id <> 'legacy-import'
           OR c.submitted_by IS NOT NULL OR c.submitted_at IS NOT NULL
           OR c.approved_by IS NOT NULL OR c.approved_at IS NOT NULL
           OR c.frozen_by IS NOT NULL OR c.frozen_at IS NOT NULL
           OR c.created_at <> v.created_at
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirement_versions v
        JOIN workspace_requirements r ON r.id = v.requirement_id
        LEFT JOIN workspace_requirement_revisions c
          ON c.baseline_id = r.id AND c.revision = v.version
        WHERE c.baseline_id IS NULL
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirement_issue_links l
        LEFT JOIN workspace_requirements r ON r.id = l.baseline_id
        WHERE r.id IS NULL
           OR l.requirement_key <> 'legacy-root'
           OR l.linked_revision <> 1
           OR l.unlinked_revision IS NOT NULL
           OR l.linked_by <> 'legacy-import'
           OR l.linked_at <> r.created_at
           OR l.unlinked_by IS NOT NULL
           OR l.unlinked_at IS NOT NULL
           OR NOT EXISTS (
               SELECT 1
               FROM json_each(CASE WHEN json_valid(r.issue_ids) THEN r.issue_ids ELSE '[]' END) j
               JOIN workspace_issues i
                 ON i.workspace_id = r.workspace_id
                AND i.project_id = r.project_id
                AND (i.id = CAST(j.value AS TEXT) OR i.identifier = CAST(j.value AS TEXT))
               WHERE i.id = l.issue_id
           )
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirements r,
             json_each(CASE WHEN json_valid(r.issue_ids) THEN r.issue_ids ELSE '[]' END) j
        JOIN workspace_issues i
          ON i.workspace_id = r.workspace_id
         AND i.project_id = r.project_id
         AND (i.id = CAST(j.value AS TEXT) OR i.identifier = CAST(j.value AS TEXT))
        LEFT JOIN workspace_requirement_issue_links l
          ON l.baseline_id = r.id
         AND l.requirement_key = 'legacy-root'
         AND l.issue_id = i.id
         AND l.linked_revision = 1
         AND l.unlinked_revision IS NULL
        WHERE l.baseline_id IS NULL
    )
    OR EXISTS (SELECT 1 FROM workspace_requirement_outline_links)
    OR EXISTS (SELECT 1 FROM workspace_requirement_review_projections)
    OR EXISTS (SELECT 1 FROM workspace_project_requirement_access_sets)
    OR EXISTS (SELECT 1 FROM workspace_project_requirement_grants)
    OR EXISTS (SELECT 1 FROM workspace_project_outline_sets)
    OR EXISTS (SELECT 1 FROM workspace_project_outline_nodes)
    OR EXISTS (
        SELECT 1 FROM workspace_resource_revisions
        WHERE resource_kind IN ('requirement_baseline','project_requirement_access','project_outline')
    )
    OR EXISTS (
        SELECT 1 FROM workspace_audit_entries
        WHERE resource_kind IN ('requirement_baseline','project_requirement_access','project_outline')
           OR action LIKE 'workspace.requirement.%'
           OR action LIKE 'workspace.project.outline.%'
    )
    OR EXISTS (
        SELECT 1 FROM workspace_mutation_idempotency
        WHERE resource_kind IN ('requirement_baseline','project_requirement_access','project_outline')
           OR action LIKE 'workspace.requirement.%'
           OR action LIKE 'workspace.project.outline.%'
    )
    OR EXISTS (
        SELECT 1 FROM workspace_outbox_events
        WHERE aggregate_kind IN ('requirement_baseline','project_requirement_access','project_outline')
           OR event_type LIKE 'requirement:%'
           OR event_type LIKE 'project_outline:%'
    )
THEN 0 ELSE 1 END;

DROP TABLE project_requirement_rollback_guard;
DROP TABLE workspace_requirement_review_projections;
DROP TABLE workspace_requirement_outline_links;
DROP TABLE workspace_requirement_issue_links;
DROP TABLE workspace_requirement_revisions;
DROP TABLE workspace_requirement_baselines;
DROP TABLE workspace_project_requirement_grants;
DROP TABLE workspace_project_requirement_access_sets;
DROP TABLE workspace_project_outline_nodes;
DROP TABLE workspace_project_outline_sets;

DELETE FROM workspace_schema_migrations
WHERE version = '000019_project_requirements.up.sql';
