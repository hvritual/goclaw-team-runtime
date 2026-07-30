-- Preserve human ownership where legacy machine actors authored six-domain data.
UPDATE issue i
SET creator_type = 'member',
    creator_id = COALESCE(
        (SELECT a.owner_id FROM agent a WHERE a.id = i.creator_id),
        (SELECT m.user_id
         FROM member m
         WHERE m.workspace_id = i.workspace_id
         ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.created_at
         LIMIT 1),
        i.creator_id
    )
WHERE i.creator_type <> 'member';

UPDATE issue
SET assignee_type = NULL, assignee_id = NULL
WHERE assignee_type IS NOT NULL AND assignee_type <> 'member';

UPDATE project
SET lead_type = NULL, lead_id = NULL
WHERE lead_type IS NOT NULL AND lead_type <> 'member';

UPDATE comment c
SET author_type = 'member',
    author_id = COALESCE(
        (SELECT a.owner_id FROM agent a WHERE a.id = c.author_id),
        (SELECT m.user_id
         FROM member m
         WHERE m.workspace_id = c.workspace_id
         ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.created_at
         LIMIT 1),
        c.author_id
    )
WHERE c.author_type = 'agent';

UPDATE comment c
SET resolved_by_type = 'member',
    resolved_by_id = COALESCE(
        (SELECT a.owner_id FROM agent a WHERE a.id = c.resolved_by_id),
        (SELECT m.user_id
         FROM member m
         WHERE m.workspace_id = c.workspace_id
         ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.created_at
         LIMIT 1),
        c.resolved_by_id
    )
WHERE c.resolved_by_type = 'agent';

UPDATE attachment a
SET uploader_type = 'member',
    uploader_id = COALESCE(
        (SELECT agent.owner_id FROM agent WHERE agent.id = a.uploader_id),
        (SELECT m.user_id
         FROM member m
         WHERE m.workspace_id = a.workspace_id
         ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.created_at
         LIMIT 1),
        a.uploader_id
    )
WHERE a.uploader_type = 'agent';

UPDATE activity_log l
SET actor_type = 'member',
    actor_id = COALESCE(
        (SELECT a.owner_id FROM agent a WHERE a.id = l.actor_id),
        (SELECT m.user_id
         FROM member m
         WHERE m.workspace_id = l.workspace_id
         ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, m.created_at
         LIMIT 1),
        l.actor_id
    )
WHERE l.actor_type = 'agent';

DELETE FROM issue_subscriber WHERE user_type <> 'member';
DELETE FROM comment_reaction WHERE actor_type <> 'member';
DELETE FROM issue_reaction WHERE actor_type <> 'member';

ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_assignee_type_check;
ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_creator_type_check;
ALTER TABLE issue ADD CONSTRAINT issue_assignee_type_check
    CHECK (assignee_type IS NULL OR assignee_type = 'member');
ALTER TABLE issue ADD CONSTRAINT issue_creator_type_check
    CHECK (creator_type = 'member');
ALTER TABLE issue
    DROP COLUMN IF EXISTS origin_type,
    DROP COLUMN IF EXISTS origin_id,
    DROP COLUMN IF EXISTS first_executed_at;

ALTER TABLE project DROP CONSTRAINT IF EXISTS project_lead_type_check;
ALTER TABLE project ADD CONSTRAINT project_lead_type_check
    CHECK (lead_type IS NULL OR lead_type = 'member');

ALTER TABLE comment DROP CONSTRAINT IF EXISTS comment_author_type_check;
ALTER TABLE comment DROP CONSTRAINT IF EXISTS comment_resolved_by_type_check;
ALTER TABLE comment ADD CONSTRAINT comment_author_type_check
    CHECK (author_type IN ('member', 'system'));
ALTER TABLE comment ADD CONSTRAINT comment_resolved_by_type_check
    CHECK (resolved_by_type IS NULL OR resolved_by_type IN ('member', 'system'));
ALTER TABLE comment DROP COLUMN IF EXISTS source_task_id;

ALTER TABLE attachment DROP CONSTRAINT IF EXISTS attachment_uploader_type_check;
ALTER TABLE attachment ADD CONSTRAINT attachment_uploader_type_check
    CHECK (uploader_type = 'member');
ALTER TABLE attachment
    DROP COLUMN IF EXISTS chat_session_id,
    DROP COLUMN IF EXISTS chat_message_id,
    DROP COLUMN IF EXISTS task_id;

ALTER TABLE activity_log DROP CONSTRAINT IF EXISTS activity_log_actor_type_check;
ALTER TABLE activity_log ADD CONSTRAINT activity_log_actor_type_check
    CHECK (actor_type IS NULL OR actor_type IN ('member', 'system'));

ALTER TABLE issue_subscriber DROP CONSTRAINT IF EXISTS issue_subscriber_user_type_check;
ALTER TABLE issue_subscriber ADD CONSTRAINT issue_subscriber_user_type_check
    CHECK (user_type = 'member');
ALTER TABLE comment_reaction DROP CONSTRAINT IF EXISTS comment_reaction_actor_type_check;
ALTER TABLE comment_reaction ADD CONSTRAINT comment_reaction_actor_type_check
    CHECK (actor_type = 'member');
ALTER TABLE issue_reaction DROP CONSTRAINT IF EXISTS issue_reaction_actor_type_check;
ALTER TABLE issue_reaction ADD CONSTRAINT issue_reaction_actor_type_check
    CHECK (actor_type = 'member');

-- Break legacy cross-domain cycles before explicitly removing their tables.
ALTER TABLE agent_task_queue
    DROP COLUMN IF EXISTS autopilot_run_id,
    DROP COLUMN IF EXISTS chat_session_id,
    DROP COLUMN IF EXISTS parent_task_id,
    DROP COLUMN IF EXISTS runtime_id;

DROP TABLE IF EXISTS webhook_delivery;
DROP TABLE IF EXISTS autopilot_subscriber;
DROP TABLE IF EXISTS autopilot_collaborator;
DROP TABLE IF EXISTS autopilot_rule_version;
DROP TABLE IF EXISTS autopilot_run;
DROP TABLE IF EXISTS autopilot_trigger;
DROP TABLE IF EXISTS autopilot;

DROP TABLE IF EXISTS channel_outbound_card_message;
DROP TABLE IF EXISTS channel_inbound_message_dedup;
DROP TABLE IF EXISTS channel_inbound_audit;
DROP TABLE IF EXISTS channel_media_pending_object;
DROP TABLE IF EXISTS channel_chat_session_binding;
DROP TABLE IF EXISTS channel_user_binding;
DROP TABLE IF EXISTS channel_binding_token;
DROP TABLE IF EXISTS channel_installation;
DROP TABLE IF EXISTS lark_outbound_card_message;
DROP TABLE IF EXISTS lark_inbound_message_dedup;
DROP TABLE IF EXISTS lark_inbound_audit;
DROP TABLE IF EXISTS lark_chat_session_binding;
DROP TABLE IF EXISTS lark_user_binding;
DROP TABLE IF EXISTS lark_binding_token;
DROP TABLE IF EXISTS lark_installation;

DROP TABLE IF EXISTS chat_draft_restore;
DROP TABLE IF EXISTS chat_pinned_agent;
DROP TABLE IF EXISTS chat_message;
DROP TABLE IF EXISTS chat_session;

DROP TABLE IF EXISTS task_token;
DROP TABLE IF EXISTS task_message;
DROP TABLE IF EXISTS task_usage_daily_dirty;
DROP TABLE IF EXISTS task_usage_daily;
DROP TABLE IF EXISTS task_usage_hourly_dirty;
DROP TABLE IF EXISTS task_usage_hourly;
DROP TABLE IF EXISTS task_usage_dashboard_dirty;
DROP TABLE IF EXISTS task_usage_dashboard_daily;
DROP TABLE IF EXISTS task_usage_dashboard_rollup_state;
DROP TABLE IF EXISTS task_usage_hourly_rollup_state;
DROP TABLE IF EXISTS task_usage_rollup_state;
DROP TABLE IF EXISTS task_usage;
DROP TABLE IF EXISTS inbox_item;
DROP TABLE IF EXISTS agent_task_queue;

DROP TABLE IF EXISTS squad_member;
DROP TABLE IF EXISTS squad;
DROP TABLE IF EXISTS agent_invocation_target;
DROP TABLE IF EXISTS agent_to_label;
DROP TABLE IF EXISTS agent_skill;
DROP TABLE IF EXISTS runtime_usage;
DROP TABLE IF EXISTS agent_runtime;
DROP TABLE IF EXISTS runtime_profile;
DROP TABLE IF EXISTS daemon_connection;
DROP TABLE IF EXISTS daemon_pairing_session;
DROP TABLE IF EXISTS daemon_token;
DROP TABLE IF EXISTS agent;

DROP TABLE IF EXISTS github_pending_check_suite;
DROP TABLE IF EXISTS github_pull_request_check_run;
DROP TABLE IF EXISTS github_pull_request_check_suite;
DROP TABLE IF EXISTS issue_pull_request;
DROP TABLE IF EXISTS github_pull_request;
DROP TABLE IF EXISTS github_pending_installation;
DROP TABLE IF EXISTS github_installation;
DROP TABLE IF EXISTS issue_vcs_pull_request;
DROP TABLE IF EXISTS vcs_commit_status;
DROP TABLE IF EXISTS vcs_pull_request;
DROP TABLE IF EXISTS vcs_connection;

DROP TABLE IF EXISTS user_composio_connection;
DROP TABLE IF EXISTS notification_preference;
DROP TABLE IF EXISTS client_usage_daily;
DROP TABLE IF EXISTS contact_sales_inquiry;
DROP TABLE IF EXISTS feedback;
DROP TABLE IF EXISTS sys_cron_executions;

ALTER TABLE workspace DROP COLUMN IF EXISTS attribution_fail_closed;
