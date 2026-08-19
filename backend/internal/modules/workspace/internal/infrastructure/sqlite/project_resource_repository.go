package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

const projectResourceColumns = `id,workspace_id,project_id,resource_type,canonical_url,resource_ref,label,position,status,revision,
	connection_state,connection_diagnostic_code,connection_checked_at,created_at,created_by,updated_at,updated_by,archived_at,archived_by`

type ProjectResourceRepository struct{ db *sql.DB }

func NewProjectResourceRepository(config Config) (*ProjectResourceRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &ProjectResourceRepository{db: config.DB}, nil
}

func (r *ProjectResourceRepository) ProjectResourceAccess(ctx context.Context, workspaceID, projectID string) (application.ProjectResourceProjectAccess, error) {
	var status string
	var leadType, leadID sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT status,lead_type,lead_id FROM workspace_projects WHERE workspace_id=? AND id=?`, workspaceID, projectID).Scan(&status, &leadType, &leadID)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ProjectResourceProjectAccess{}, application.ErrProjectSurfaceNotFound
	}
	if err != nil {
		return application.ProjectResourceProjectAccess{}, fmt.Errorf("read Project Resource access: %w", err)
	}
	return application.ProjectResourceProjectAccess{Status: status, LeadType: nullableText(leadType), LeadID: nullableText(leadID)}, nil
}

func (r *ProjectResourceRepository) ListProjectResources(ctx context.Context, workspaceID, projectID string, includeArchived bool) (contract.ProjectResourceList, error) {
	var revision int64
	err := r.db.QueryRowContext(ctx, `SELECT revision FROM workspace_project_resource_sets WHERE workspace_id=? AND project_id=?`, workspaceID, projectID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		revision = 0
	} else if err != nil {
		return contract.ProjectResourceList{}, fmt.Errorf("read Project Resource revision: %w", err)
	}
	query := `SELECT ` + projectResourceColumns + ` FROM workspace_project_resources WHERE workspace_id=? AND project_id=?`
	if !includeArchived {
		query += ` AND status='active'`
	}
	query += ` ORDER BY CASE status WHEN 'active' THEN 0 ELSE 1 END,position,id`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, projectID)
	if err != nil {
		return contract.ProjectResourceList{}, fmt.Errorf("list Project Resources: %w", err)
	}
	defer rows.Close()
	resources := make([]contract.ProjectResource, 0)
	for rows.Next() {
		resource, scanErr := scanProjectResource(rows)
		if scanErr != nil {
			return contract.ProjectResourceList{}, scanErr
		}
		resources = append(resources, resource)
	}
	if err = rows.Err(); err != nil {
		return contract.ProjectResourceList{}, fmt.Errorf("iterate Project Resources: %w", err)
	}
	return contract.ProjectResourceList{Resources: resources, Total: len(resources), Revision: revision}, nil
}

func (r *ProjectResourceRepository) GetProjectResource(ctx context.Context, workspaceID, projectID, resourceID string) (contract.ProjectResource, error) {
	resource, err := scanProjectResource(r.db.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND id=?`, workspaceID, projectID, resourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectResource{}, application.ErrProjectResourceNotFound
	}
	if err != nil {
		return contract.ProjectResource{}, fmt.Errorf("get Project Resource: %w", err)
	}
	return resource, nil
}

func (r *ProjectResourceRepository) CreateProjectResource(ctx context.Context, command application.ProjectResourceCreate) (result contract.ProjectResource, err error) {
	connection, err := r.projectResourceConnection(ctx, "create")
	if err != nil {
		return contract.ProjectResource{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectResourceConnection(connection, &committed)()
	if err = ensureProjectResourceManagerOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor); err != nil {
		return contract.ProjectResource{}, err
	}

	var storedHash, responseBody string
	replayErr := connection.QueryRowContext(ctx, `SELECT request_hash,response_body FROM workspace_mutation_idempotency
		WHERE workspace_id=? AND action='workspace.project.resource.create' AND idempotency_key=?`, command.WorkspaceID, command.IdempotencyKey).Scan(&storedHash, &responseBody)
	if replayErr == nil {
		if storedHash != command.RequestHash {
			return contract.ProjectResource{}, application.ErrProjectResourceConflict
		}
		if err = json.Unmarshal([]byte(responseBody), &result); err != nil {
			return contract.ProjectResource{}, fmt.Errorf("decode Project Resource replay: %w", err)
		}
		if _, err = connection.ExecContext(ctx, "ROLLBACK"); err != nil {
			return contract.ProjectResource{}, fmt.Errorf("finish Project Resource replay: %w", err)
		}
		committed = true
		return result, nil
	}
	if !errors.Is(replayErr, sql.ErrNoRows) {
		return contract.ProjectResource{}, fmt.Errorf("read Project Resource replay: %w", replayErr)
	}
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_project_resource_sets(workspace_id,project_id,revision,updated_at) VALUES(?,?,0,?)`, command.WorkspaceID, command.ProjectID, timestamp); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("initialize Project Resource revision: %w", err)
	}
	var currentRevision int64
	if err = connection.QueryRowContext(ctx, `SELECT revision FROM workspace_project_resource_sets WHERE workspace_id=? AND project_id=?`, command.WorkspaceID, command.ProjectID).Scan(&currentRevision); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("read Project Resource set: %w", err)
	}
	duplicate, err := projectResourceFingerprintExists(ctx, connection, command.WorkspaceID, command.ProjectID, command.Fingerprint, "")
	if err != nil {
		return contract.ProjectResource{}, fmt.Errorf("check duplicate Project Resource: %w", err)
	}
	if duplicate {
		return contract.ProjectResource{}, application.ErrProjectResourceConflict
	}
	var position int
	if err = connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND status='active'`, command.WorkspaceID, command.ProjectID).Scan(&position); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("count Project Resources: %w", err)
	}
	revision := currentRevision + 1
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_resources(
		id,workspace_id,project_id,resource_type,canonical_url,resource_ref,fingerprint,label,position,status,revision,
		connection_state,connection_diagnostic_code,created_at,created_by,updated_at,updated_by
	) VALUES(?,?,?,?,?,?,?,?,?,'active',?,'unchecked','',?,?,?,?)`,
		command.ID, command.WorkspaceID, command.ProjectID, command.ResourceType, command.ResourceRef.URL,
		command.ResourceRef.Ref, command.Fingerprint, command.Label, position, revision, timestamp, command.Actor.ID,
		timestamp, command.Actor.ID)
	if err != nil {
		if isProjectResourceConstraint(err) {
			return contract.ProjectResource{}, application.ErrProjectResourceConflict
		}
		return contract.ProjectResource{}, fmt.Errorf("insert Project Resource: %w", err)
	}
	if err = updateProjectResourceSetRevision(ctx, connection, command.WorkspaceID, command.ProjectID, currentRevision, revision, timestamp); err != nil {
		return contract.ProjectResource{}, err
	}
	result, err = scanProjectResource(connection.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE id=?`, command.ID))
	if err != nil {
		return contract.ProjectResource{}, fmt.Errorf("read created Project Resource: %w", err)
	}
	if err = insertProjectResourceAudit(ctx, connection, result, command.Actor, "create", command.IdempotencyKey, command.OccurredAt); err != nil {
		return contract.ProjectResource{}, err
	}
	response, err := json.Marshal(result)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency(
		workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at
	) VALUES(?,'workspace.project.resource.create',?,?,'project_resource',?,?,201,?,?)`,
		command.WorkspaceID, command.IdempotencyKey, command.RequestHash, result.ID, revision, string(response), timestamp); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("record Project Resource replay: %w", err)
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("commit Project Resource create: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectResourceRepository) MutateProjectResource(ctx context.Context, command application.ProjectResourceMutation) (result contract.ProjectResource, err error) {
	connection, err := r.projectResourceConnection(ctx, command.Action)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectResourceConnection(connection, &committed)()
	if err = ensureProjectResourceManagerOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor); err != nil {
		return contract.ProjectResource{}, err
	}
	currentRevision, err := lockProjectResourceSet(ctx, connection, command.WorkspaceID, command.ProjectID)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if currentRevision != command.ExpectedRevision {
		return contract.ProjectResource{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	current, err := scanProjectResource(connection.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND id=?`, command.WorkspaceID, command.ProjectID, command.ResourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectResource{}, application.ErrProjectResourceNotFound
	}
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if (command.Action == "update" || command.Action == "reorder") && current.Status != "active" {
		return contract.ProjectResource{}, application.ErrInvalidProjectResourceRequest
	}
	if command.Action == "restore" && current.Status != "archived" {
		return contract.ProjectResource{}, application.ErrInvalidProjectResourceRequest
	}
	nextRevision := currentRevision + 1
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	switch command.Action {
	case "update":
		urlValue, refValue := current.ResourceRef.URL, current.ResourceRef.Ref
		if command.ResourceRef != nil {
			urlValue, refValue = command.ResourceRef.URL, command.ResourceRef.Ref
		}
		label := current.Label
		if command.Label != nil {
			label = *command.Label
		}
		fingerprint := command.Fingerprint
		if fingerprint == "" {
			var storedFingerprint string
			if err = connection.QueryRowContext(ctx, `SELECT fingerprint FROM workspace_project_resources WHERE id=?`, command.ResourceID).Scan(&storedFingerprint); err != nil {
				return contract.ProjectResource{}, err
			}
			fingerprint = storedFingerprint
		}
		duplicate, duplicateErr := projectResourceFingerprintExists(ctx, connection, command.WorkspaceID, command.ProjectID, fingerprint, command.ResourceID)
		if duplicateErr != nil {
			return contract.ProjectResource{}, fmt.Errorf("check duplicate Project Resource update: %w", duplicateErr)
		}
		if duplicate {
			return contract.ProjectResource{}, application.ErrProjectResourceConflict
		}
		_, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET canonical_url=?,resource_ref=?,fingerprint=?,label=?,revision=?,updated_at=?,updated_by=?
			WHERE workspace_id=? AND project_id=? AND id=? AND status='active'`,
			urlValue, refValue, fingerprint, label, nextRevision, timestamp, command.Actor.ID,
			command.WorkspaceID, command.ProjectID, command.ResourceID)
		if isProjectResourceConstraint(err) {
			return contract.ProjectResource{}, application.ErrProjectResourceConflict
		}
	case "reorder":
		err = reorderProjectResources(ctx, connection, command.WorkspaceID, command.ProjectID, command.ResourceID, command.BeforeResourceID)
		if err == nil {
			_, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET revision=?,updated_at=?,updated_by=? WHERE id=?`, nextRevision, timestamp, command.Actor.ID, command.ResourceID)
		}
	case "restore":
		var position int
		if err = connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND status='active'`, command.WorkspaceID, command.ProjectID).Scan(&position); err == nil {
			_, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET status='active',position=?,revision=?,updated_at=?,updated_by=?,archived_at=NULL,archived_by=NULL
				WHERE workspace_id=? AND project_id=? AND id=? AND status='archived'`,
				position, nextRevision, timestamp, command.Actor.ID, command.WorkspaceID, command.ProjectID, command.ResourceID)
		}
	default:
		return contract.ProjectResource{}, application.ErrInvalidProjectResourceRequest
	}
	if err != nil {
		return contract.ProjectResource{}, fmt.Errorf("%s Project Resource: %w", command.Action, err)
	}
	if err = updateProjectResourceSetRevision(ctx, connection, command.WorkspaceID, command.ProjectID, currentRevision, nextRevision, timestamp); err != nil {
		return contract.ProjectResource{}, err
	}
	result, err = scanProjectResource(connection.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE id=?`, command.ResourceID))
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if err = insertProjectResourceAudit(ctx, connection, result, command.Actor, command.Action, projectResourceRequestID(command.ResourceID, command.Action, nextRevision), command.OccurredAt); err != nil {
		return contract.ProjectResource{}, err
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("commit Project Resource %s: %w", command.Action, err)
	}
	committed = true
	return result, nil
}

func (r *ProjectResourceRepository) RefreshProjectResource(ctx context.Context, command application.ProjectResourceMutation, resolver application.ProjectResourceConnectionResolver) (result contract.ProjectResource, err error) {
	if resolver == nil {
		return contract.ProjectResource{}, application.ErrInvalidProjectResourceRequest
	}
	connection, err := r.projectResourceConnection(ctx, "refresh")
	if err != nil {
		return contract.ProjectResource{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectResourceConnection(connection, &committed)()
	if err = ensureProjectResourceManagerOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor); err != nil {
		return contract.ProjectResource{}, err
	}
	currentRevision, err := lockProjectResourceSet(ctx, connection, command.WorkspaceID, command.ProjectID)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if currentRevision != command.ExpectedRevision {
		return contract.ProjectResource{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	current, err := scanProjectResource(connection.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND id=?`, command.WorkspaceID, command.ProjectID, command.ResourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectResource{}, application.ErrProjectResourceNotFound
	}
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if current.Status != "active" {
		return contract.ProjectResource{}, application.ErrInvalidProjectResourceRequest
	}
	projection := resolver(ctx, contract.ProjectResourceConnectionRequest{
		WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID, ResourceID: command.ResourceID,
		ResourceType: current.ResourceType, ResourceRef: current.ResourceRef,
	})
	nextRevision := currentRevision + 1
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET connection_state=?,connection_diagnostic_code=?,connection_checked_at=?,revision=?,updated_at=?,updated_by=?
		WHERE workspace_id=? AND project_id=? AND id=? AND status='active'`,
		projection.State, projection.DiagnosticCode, nullableStringValue(projection.CheckedAt), nextRevision, timestamp,
		command.Actor.ID, command.WorkspaceID, command.ProjectID, command.ResourceID); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("refresh Project Resource: %w", err)
	}
	if err = updateProjectResourceSetRevision(ctx, connection, command.WorkspaceID, command.ProjectID, currentRevision, nextRevision, timestamp); err != nil {
		return contract.ProjectResource{}, err
	}
	result, err = scanProjectResource(connection.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE id=?`, command.ResourceID))
	if err != nil {
		return contract.ProjectResource{}, err
	}
	if err = insertProjectResourceAudit(ctx, connection, result, command.Actor, "refresh", projectResourceRequestID(command.ResourceID, "refresh", nextRevision), command.OccurredAt); err != nil {
		return contract.ProjectResource{}, err
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectResource{}, fmt.Errorf("commit Project Resource refresh: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectResourceRepository) ArchiveProjectResource(ctx context.Context, workspaceID, projectID, resourceID string, expectedRevision int64, actor contract.WorkspaceActor, occurredAt time.Time) (err error) {
	connection, err := r.projectResourceConnection(ctx, "archive")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectResourceConnection(connection, &committed)()
	if err = ensureProjectResourceManagerOnConnection(ctx, connection, workspaceID, projectID, actor); err != nil {
		return err
	}
	currentRevision, err := lockProjectResourceSet(ctx, connection, workspaceID, projectID)
	if err != nil {
		return err
	}
	if currentRevision != expectedRevision {
		return contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	current, err := scanProjectResource(connection.QueryRowContext(ctx, `SELECT `+projectResourceColumns+` FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND id=?`, workspaceID, projectID, resourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return application.ErrProjectResourceNotFound
	}
	if err != nil {
		return err
	}
	if current.Status != "active" {
		return application.ErrInvalidProjectResourceRequest
	}
	nextRevision := currentRevision + 1
	timestamp := occurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET status='archived',revision=?,updated_at=?,updated_by=?,archived_at=?,archived_by=?
		WHERE workspace_id=? AND project_id=? AND id=? AND status='active'`,
		nextRevision, timestamp, actor.ID, timestamp, actor.ID, workspaceID, projectID, resourceID); err != nil {
		return fmt.Errorf("archive Project Resource: %w", err)
	}
	if err = compactActiveProjectResourcePositions(ctx, connection, workspaceID, projectID); err != nil {
		return err
	}
	if err = updateProjectResourceSetRevision(ctx, connection, workspaceID, projectID, currentRevision, nextRevision, timestamp); err != nil {
		return err
	}
	current.Status, current.Revision, current.ArchivedAt, current.ArchivedBy = "archived", nextRevision, timestamp, actor.ID
	current.UpdatedAt, current.UpdatedBy = timestamp, actor.ID
	if err = insertProjectResourceAudit(ctx, connection, current, actor, "archive", projectResourceRequestID(resourceID, "archive", nextRevision), occurredAt); err != nil {
		return err
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit Project Resource archive: %w", err)
	}
	committed = true
	return nil
}

func (r *ProjectResourceRepository) projectResourceConnection(ctx context.Context, operation string) (*sql.Conn, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire Project Resource %s connection: %w", operation, err)
	}
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		connection.Close()
		return nil, fmt.Errorf("configure Project Resource %s connection: %w", operation, err)
	}
	if _, err = connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		connection.Close()
		return nil, fmt.Errorf("begin Project Resource %s: %w", operation, err)
	}
	return connection, nil
}

func rollbackProjectResourceConnection(connection *sql.Conn, committed *bool) func() {
	return func() {
		if !*committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}
}

type projectResourceMembershipAuthority struct {
	MemberID string
	UserID   string
	Role     string
}

func ensureProjectResourceManagerOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID string, actor contract.WorkspaceActor) error {
	membership, err := projectResourceMembershipOnConnection(ctx, connection, workspaceID, actor)
	if err != nil {
		return err
	}
	var status string
	var leadType, leadID sql.NullString
	err = connection.QueryRowContext(ctx, `SELECT status,lead_type,lead_id FROM workspace_projects WHERE workspace_id=? AND id=?`, workspaceID, projectID).Scan(&status, &leadType, &leadID)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ErrProjectSurfaceNotFound
	}
	if err != nil {
		return fmt.Errorf("find Project for Resource: %w", err)
	}
	if status == "completed" || status == "cancelled" {
		return contract.ErrWorkspacePermissionDenied
	}
	if projectResourceMembershipCanManage(membership, actor.ID, nullableText(leadType), nullableText(leadID)) {
		return nil
	}
	return contract.ErrWorkspacePermissionDenied
}

func ensureInitialProjectResourceManagerOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, status string, leadType, leadID *string, actor contract.WorkspaceActor) error {
	membership, err := projectResourceMembershipOnConnection(ctx, connection, workspaceID, actor)
	if err != nil {
		return err
	}
	if status == "completed" || status == "cancelled" {
		return contract.ErrWorkspacePermissionDenied
	}
	typeValue, idValue := "", ""
	if leadType != nil {
		typeValue = strings.TrimSpace(*leadType)
	}
	if leadID != nil {
		idValue = strings.TrimSpace(*leadID)
	}
	if projectResourceMembershipCanManage(membership, actor.ID, typeValue, idValue) {
		return nil
	}
	return contract.ErrWorkspacePermissionDenied
}

func projectResourceMembershipOnConnection(ctx context.Context, connection *sql.Conn, workspaceID string, actor contract.WorkspaceActor) (projectResourceMembershipAuthority, error) {
	actorID := strings.TrimSpace(actor.ID)
	if actor.Type != "member" || actorID == "" {
		return projectResourceMembershipAuthority{}, contract.ErrWorkspacePermissionDenied
	}
	var membership projectResourceMembershipAuthority
	err := connection.QueryRowContext(ctx, `SELECT id,user_id,role FROM auth_members
		WHERE workspace_id=? AND (user_id=? OR id=?)
		ORDER BY CASE WHEN user_id=? THEN 0 ELSE 1 END LIMIT 1`,
		workspaceID, actorID, actorID, actorID).Scan(&membership.MemberID, &membership.UserID, &membership.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return projectResourceMembershipAuthority{}, contract.ErrActorOutsideWorkspace
	}
	if err != nil {
		return projectResourceMembershipAuthority{}, fmt.Errorf("read current Project Resource membership: %w", err)
	}
	return membership, nil
}

func projectResourceMembershipCanManage(membership projectResourceMembershipAuthority, actorID, leadType, leadID string) bool {
	if membership.Role == "owner" || membership.Role == "admin" {
		return true
	}
	return leadType == "member" && leadID != "" &&
		(leadID == strings.TrimSpace(actorID) || leadID == membership.MemberID || leadID == membership.UserID)
}

func projectResourceFingerprintExists(ctx context.Context, connection *sql.Conn, workspaceID, projectID, fingerprint, excludeID string) (bool, error) {
	var found int
	err := connection.QueryRowContext(ctx, `SELECT 1 FROM workspace_project_resources
		WHERE workspace_id=? AND project_id=? AND fingerprint=? AND (?='' OR id<>?) LIMIT 1`,
		workspaceID, projectID, fingerprint, excludeID, excludeID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func lockProjectResourceSet(ctx context.Context, connection *sql.Conn, workspaceID, projectID string) (int64, error) {
	var revision int64
	err := connection.QueryRowContext(ctx, `SELECT revision FROM workspace_project_resource_sets WHERE workspace_id=? AND project_id=?`, workspaceID, projectID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, application.ErrProjectResourceNotFound
	}
	return revision, err
}

func updateProjectResourceSetRevision(ctx context.Context, connection *sql.Conn, workspaceID, projectID string, current, next int64, timestamp string) error {
	result, err := connection.ExecContext(ctx, `UPDATE workspace_project_resource_sets SET revision=?,updated_at=? WHERE workspace_id=? AND project_id=? AND revision=?`, next, timestamp, workspaceID, projectID, current)
	if err != nil {
		return fmt.Errorf("update Project Resource revision: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		var actual int64
		_ = connection.QueryRowContext(ctx, `SELECT revision FROM workspace_project_resource_sets WHERE workspace_id=? AND project_id=?`, workspaceID, projectID).Scan(&actual)
		return contract.RevisionConflictError{CurrentRevision: actual}
	}
	return nil
}

func reorderProjectResources(ctx context.Context, connection *sql.Conn, workspaceID, projectID, resourceID string, beforeID *string) error {
	rows, err := connection.QueryContext(ctx, `SELECT id FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND status='active' ORDER BY position,id`, workspaceID, projectID)
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	found := false
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if id == resourceID {
			found = true
			continue
		}
		ids = append(ids, id)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if !found {
		return application.ErrProjectResourceNotFound
	}
	insertAt := len(ids)
	if beforeID != nil {
		if *beforeID == resourceID {
			return application.ErrInvalidProjectResourceRequest
		}
		insertAt = -1
		for index, id := range ids {
			if id == *beforeID {
				insertAt = index
				break
			}
		}
		if insertAt < 0 {
			return application.ErrInvalidProjectResourceRequest
		}
	}
	ids = append(ids, "")
	copy(ids[insertAt+1:], ids[insertAt:])
	ids[insertAt] = resourceID
	for position, id := range ids {
		if _, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET position=? WHERE workspace_id=? AND project_id=? AND id=? AND status='active'`, position, workspaceID, projectID, id); err != nil {
			return err
		}
	}
	return nil
}

func compactActiveProjectResourcePositions(ctx context.Context, connection *sql.Conn, workspaceID, projectID string) error {
	rows, err := connection.QueryContext(ctx, `SELECT id FROM workspace_project_resources WHERE workspace_id=? AND project_id=? AND status='active' ORDER BY position,id`, workspaceID, projectID)
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for position, id := range ids {
		if _, err = connection.ExecContext(ctx, `UPDATE workspace_project_resources SET position=? WHERE id=?`, position, id); err != nil {
			return err
		}
	}
	return nil
}

func insertProjectResourceAudit(ctx context.Context, connection *sql.Conn, resource contract.ProjectResource, actor contract.WorkspaceActor, action, requestID string, occurredAt time.Time) error {
	metadata, _ := json.Marshal(map[string]any{"version": "project-resource-v1", "action": action, "resource_type": resource.ResourceType, "status": resource.Status})
	timestamp := occurredAt.UTC().Format(time.RFC3339Nano)
	auditID := projectResourceRequestID(resource.ID, action, resource.Revision)
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_audit_entries(
		workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		resource.WorkspaceID, timestamp, auditID, actor.Type, actor.ID, "workspace.project.resource."+action,
		"project_resource", resource.ID, resource.Revision, requestID, string(metadata))
	if err != nil {
		return fmt.Errorf("record Project Resource audit: %w", err)
	}
	return nil
}

func projectResourceRequestID(resourceID, action string, revision int64) string {
	return resourceID + "-" + action + "-" + strconv.FormatInt(revision, 10)
}

type projectResourceScanner interface{ Scan(...any) error }

func scanProjectResource(scanner projectResourceScanner) (contract.ProjectResource, error) {
	var value contract.ProjectResource
	var checkedAt, archivedAt, archivedBy sql.NullString
	err := scanner.Scan(
		&value.ID, &value.WorkspaceID, &value.ProjectID, &value.ResourceType,
		&value.ResourceRef.URL, &value.ResourceRef.Ref, &value.Label, &value.Position,
		&value.Status, &value.Revision, &value.Connection.State, &value.Connection.DiagnosticCode,
		&checkedAt, &value.CreatedAt, &value.CreatedBy, &value.UpdatedAt, &value.UpdatedBy,
		&archivedAt, &archivedBy,
	)
	if err != nil {
		return contract.ProjectResource{}, err
	}
	value.Connection.CheckedAt = nullableText(checkedAt)
	value.ArchivedAt, value.ArchivedBy = nullableText(archivedAt), nullableText(archivedBy)
	return value, nil
}

func nullableText(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func nullableStringValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func isProjectResourceConstraint(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "constraint") || strings.Contains(message, "unique")
}

var _ application.ProjectResourceRepository = (*ProjectResourceRepository)(nil)
