CREATE TEMP TABLE project_requirement_migration_guard (
    safe INTEGER NOT NULL CHECK (safe = 1)
);

INSERT INTO project_requirement_migration_guard(safe)
SELECT CASE WHEN
    EXISTS (
        SELECT 1
        FROM workspace_requirements
        GROUP BY workspace_id, project_id
        HAVING COUNT(*) > 1
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirements r
        LEFT JOIN workspace_projects p
          ON p.workspace_id = r.workspace_id AND p.id = r.project_id
        WHERE p.id IS NULL
           OR r.current_version < 1
           OR CASE WHEN json_valid(r.issue_ids) THEN json_type(r.issue_ids) ELSE 'invalid' END <> 'array'
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirements r
        LEFT JOIN (
            SELECT requirement_id, COUNT(*) AS version_count,
                   MIN(version) AS minimum_version, MAX(version) AS maximum_version
            FROM workspace_requirement_versions
            GROUP BY requirement_id
        ) v ON v.requirement_id = r.id
        WHERE COALESCE(v.version_count, 0) <> r.current_version
           OR COALESCE(v.minimum_version, 0) <> 1
           OR COALESCE(v.maximum_version, 0) <> r.current_version
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirement_versions v
        LEFT JOIN workspace_requirements r ON r.id = v.requirement_id
        WHERE r.id IS NULL
    )
    OR EXISTS (
        SELECT 1
        FROM workspace_requirements r,
             json_each(CASE WHEN json_valid(r.issue_ids) THEN r.issue_ids ELSE '[]' END) j
        WHERE j.type <> 'text'
           OR (
               SELECT COUNT(*)
               FROM workspace_issues i
               WHERE i.workspace_id = r.workspace_id
                 AND i.project_id = r.project_id
                 AND (i.id = CAST(j.value AS TEXT) OR i.identifier = CAST(j.value AS TEXT))
           ) <> 1
    )
THEN 0 ELSE 1 END;

DROP TABLE project_requirement_migration_guard;

CREATE TABLE workspace_requirement_baselines (
    id TEXT PRIMARY KEY CHECK (length(trim(id)) > 0),
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    status TEXT NOT NULL CHECK (status IN ('draft','in_review','approved','frozen','changed','retired')),
    current_revision INTEGER NOT NULL CHECK (current_revision >= 1),
    approved_revision INTEGER CHECK (approved_revision IS NULL OR (approved_revision >= 1 AND approved_revision <= current_revision)),
    effective_revision INTEGER CHECK (effective_revision IS NULL OR (effective_revision >= 1 AND effective_revision <= current_revision)),
    review_origin TEXT CHECK (review_origin IS NULL OR review_origin IN ('draft','changed')),
    latest_content_author TEXT NOT NULL CHECK (length(trim(latest_content_author)) > 0),
    submitted_by TEXT,
    submitted_at TEXT,
    approved_by TEXT,
    approved_at TEXT,
    frozen_by TEXT,
    frozen_at TEXT,
    retired_by TEXT,
    retired_at TEXT,
    legacy_requirement_id TEXT,
    legacy_snapshot_json TEXT CHECK (legacy_snapshot_json IS NULL OR json_valid(legacy_snapshot_json)),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    UNIQUE (workspace_id, project_id),
    UNIQUE (legacy_requirement_id),
    CHECK ((submitted_by IS NULL) = (submitted_at IS NULL)),
    CHECK ((approved_by IS NULL) = (approved_at IS NULL)),
    CHECK ((frozen_by IS NULL) = (frozen_at IS NULL)),
    CHECK ((retired_by IS NULL) = (retired_at IS NULL))
);

CREATE TABLE workspace_requirement_revisions (
    baseline_id TEXT NOT NULL CHECK (length(trim(baseline_id)) > 0),
    revision INTEGER NOT NULL CHECK (revision >= 1),
    content_json TEXT NOT NULL CHECK (json_valid(content_json) AND length(CAST(content_json AS BLOB)) <= 131072),
    status TEXT NOT NULL CHECK (status IN ('draft','in_review','approved','frozen','changed','retired')),
    action TEXT NOT NULL CHECK (action IN (
        'create','save_draft','submit_review','withdraw_review','approve','freeze',
        'material_change','retire','link_issue','unlink_issue','link_outline',
        'unlink_outline','issue_deleted','legacy_import'
    )),
    change_summary TEXT NOT NULL CHECK (length(trim(change_summary)) BETWEEN 1 AND 500),
    actor_id TEXT NOT NULL CHECK (length(trim(actor_id)) > 0),
    submitted_by TEXT,
    submitted_at TEXT,
    approved_by TEXT,
    approved_at TEXT,
    frozen_by TEXT,
    frozen_at TEXT,
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    PRIMARY KEY (baseline_id, revision),
    CHECK ((submitted_by IS NULL) = (submitted_at IS NULL)),
    CHECK ((approved_by IS NULL) = (approved_at IS NULL)),
    CHECK ((frozen_by IS NULL) = (frozen_at IS NULL))
) WITHOUT ROWID;

CREATE TABLE workspace_requirement_issue_links (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    baseline_id TEXT NOT NULL CHECK (length(trim(baseline_id)) > 0),
    requirement_key TEXT NOT NULL CHECK (length(trim(requirement_key)) BETWEEN 1 AND 64),
    issue_id TEXT NOT NULL CHECK (length(trim(issue_id)) > 0),
    linked_revision INTEGER NOT NULL CHECK (linked_revision >= 1),
    unlinked_revision INTEGER CHECK (unlinked_revision IS NULL OR unlinked_revision > linked_revision),
    linked_by TEXT NOT NULL CHECK (length(trim(linked_by)) > 0),
    linked_at TEXT NOT NULL CHECK (length(trim(linked_at)) > 0),
    unlinked_by TEXT,
    unlinked_at TEXT,
    PRIMARY KEY (baseline_id, requirement_key, issue_id, linked_revision),
    CHECK ((unlinked_revision IS NULL) = (unlinked_by IS NULL)),
    CHECK ((unlinked_revision IS NULL) = (unlinked_at IS NULL))
) WITHOUT ROWID;

CREATE TABLE workspace_requirement_outline_links (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    baseline_id TEXT NOT NULL CHECK (length(trim(baseline_id)) > 0),
    requirement_key TEXT NOT NULL CHECK (length(trim(requirement_key)) BETWEEN 1 AND 64),
    node_id TEXT NOT NULL CHECK (length(trim(node_id)) > 0),
    linked_revision INTEGER NOT NULL CHECK (linked_revision >= 1),
    unlinked_revision INTEGER CHECK (unlinked_revision IS NULL OR unlinked_revision > linked_revision),
    linked_by TEXT NOT NULL CHECK (length(trim(linked_by)) > 0),
    linked_at TEXT NOT NULL CHECK (length(trim(linked_at)) > 0),
    unlinked_by TEXT,
    unlinked_at TEXT,
    PRIMARY KEY (baseline_id, requirement_key, node_id, linked_revision),
    CHECK ((unlinked_revision IS NULL) = (unlinked_by IS NULL)),
    CHECK ((unlinked_revision IS NULL) = (unlinked_at IS NULL))
) WITHOUT ROWID;

CREATE TABLE workspace_requirement_review_projections (
    baseline_id TEXT NOT NULL CHECK (length(trim(baseline_id)) > 0),
    requirement_key TEXT NOT NULL CHECK (length(trim(requirement_key)) BETWEEN 1 AND 64),
    issue_id TEXT NOT NULL CHECK (length(trim(issue_id)) > 0),
    source_revision INTEGER NOT NULL CHECK (source_revision >= 1),
    status TEXT NOT NULL CHECK (status = 'review_required'),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    PRIMARY KEY (baseline_id, requirement_key, issue_id, source_revision)
) WITHOUT ROWID;

CREATE TABLE workspace_project_requirement_access_sets (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    revision INTEGER NOT NULL CHECK (revision >= 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    PRIMARY KEY (workspace_id, project_id)
) WITHOUT ROWID;

CREATE TABLE workspace_project_requirement_grants (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    member_id TEXT NOT NULL CHECK (length(trim(member_id)) > 0),
    grant_kind TEXT NOT NULL CHECK (grant_kind IN ('project_editor','requirement_approver')),
    granted_by TEXT NOT NULL CHECK (length(trim(granted_by)) > 0),
    granted_at TEXT NOT NULL CHECK (length(trim(granted_at)) > 0),
    PRIMARY KEY (workspace_id, project_id, member_id, grant_kind)
) WITHOUT ROWID;

CREATE TABLE workspace_project_outline_sets (
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    revision INTEGER NOT NULL CHECK (revision >= 0),
    updated_at TEXT NOT NULL CHECK (length(trim(updated_at)) > 0),
    PRIMARY KEY (workspace_id, project_id)
) WITHOUT ROWID;

CREATE TABLE workspace_project_outline_nodes (
    id TEXT PRIMARY KEY CHECK (length(trim(id)) > 0),
    workspace_id TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
    project_id TEXT NOT NULL CHECK (length(trim(project_id)) > 0),
    title TEXT NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 500),
    created_by TEXT NOT NULL CHECK (length(trim(created_by)) > 0),
    created_at TEXT NOT NULL CHECK (length(trim(created_at)) > 0),
    UNIQUE (workspace_id, project_id, id)
);

INSERT INTO workspace_requirement_baselines(
    id,workspace_id,project_id,status,current_revision,approved_revision,
    effective_revision,review_origin,latest_content_author,submitted_by,
    submitted_at,approved_by,approved_at,frozen_by,frozen_at,retired_by,
    retired_at,legacy_requirement_id,legacy_snapshot_json,created_at,updated_at
)
SELECT id,workspace_id,project_id,'draft',current_version,NULL,NULL,NULL,
       'legacy-import',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,id,
       json_object(
           'id',id,
           'workspace_id',workspace_id,
           'project_id',project_id,
           'title',title,
           'current_version',current_version,
           'approval_status',approval_status,
           'coverage_status',coverage_status,
           'issue_ids',issue_ids,
           'created_at',created_at,
           'updated_at',updated_at
       ),
       created_at,updated_at
FROM workspace_requirements;

INSERT INTO workspace_requirement_revisions(
    baseline_id,revision,content_json,status,action,change_summary,actor_id,
    submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
)
SELECT r.id,v.version,
       json_object(
           'problem_statement',v.content,
           'goals',json_array(json_object('key','legacy-root','text',r.title)),
           'in_scope',json('[]'),
           'out_of_scope',json('[]'),
           'constraints',json('[]'),
           'acceptance_criteria',json('[]'),
           'dependencies',json('[]')
       ),
       'draft','legacy_import','Imported legacy Requirement version','legacy-import',
       NULL,NULL,NULL,NULL,NULL,NULL,v.created_at
FROM workspace_requirements r
JOIN workspace_requirement_versions v ON v.requirement_id = r.id;

INSERT INTO workspace_requirement_issue_links(
    workspace_id,project_id,baseline_id,requirement_key,issue_id,linked_revision,
    unlinked_revision,linked_by,linked_at,unlinked_by,unlinked_at
)
SELECT DISTINCT r.workspace_id,r.project_id,r.id,'legacy-root',i.id,1,
       NULL,'legacy-import',r.created_at,NULL,NULL
FROM workspace_requirements r,
     json_each(r.issue_ids) j
JOIN workspace_issues i
  ON i.workspace_id = r.workspace_id
 AND i.project_id = r.project_id
 AND (i.id = CAST(j.value AS TEXT) OR i.identifier = CAST(j.value AS TEXT));
