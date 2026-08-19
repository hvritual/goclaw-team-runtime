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
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
)

type ProjectRequirementRepository struct {
	db *sql.DB
}

func NewProjectRequirementRepository(config Config) (*ProjectRequirementRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &ProjectRequirementRepository{db: config.DB}, nil
}

func (r *ProjectRequirementRepository) SaveProjectRequirement(ctx context.Context, command application.ProjectRequirementSave) (result contract.ProjectRequirementBaselineResponse, err error) {
	connection, err := r.projectRequirementConnection(ctx, "save")
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectRequirementConnection(connection, &committed)()

	authority, err := loadProjectRequirementAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if !authority.canEdit() {
		return contract.ProjectRequirementBaselineResponse{}, contract.ErrWorkspacePermissionDenied
	}

	if command.ExpectedRevision == 0 {
		if strings.TrimSpace(command.IdempotencyKey) == "" || len(command.IdempotencyKey) > 200 || len(command.RequestHash) != 64 {
			return contract.ProjectRequirementBaselineResponse{}, application.ErrInvalidProjectRequirementRequest
		}
		var storedHash, responseBody string
		replayErr := connection.QueryRowContext(ctx, `SELECT request_hash,response_body
			FROM workspace_mutation_idempotency
			WHERE workspace_id=? AND action='workspace.requirement.create' AND idempotency_key=?`,
			command.WorkspaceID, command.IdempotencyKey).Scan(&storedHash, &responseBody)
		if replayErr == nil {
			if storedHash != command.RequestHash {
				return contract.ProjectRequirementBaselineResponse{}, contract.ErrIdempotencyConflict
			}
			if err = json.Unmarshal([]byte(responseBody), &result); err != nil {
				return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("decode Project Requirement replay: %w", err)
			}
			if _, err = connection.ExecContext(ctx, "ROLLBACK"); err != nil {
				return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("finish Project Requirement replay: %w", err)
			}
			committed = true
			return result, nil
		}
		if !errors.Is(replayErr, sql.ErrNoRows) {
			return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("read Project Requirement replay: %w", replayErr)
		}
	}

	current, currentRevision, readErr := readProjectRequirementBaselineOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID)
	switch {
	case errors.Is(readErr, sql.ErrNoRows):
		if command.ExpectedRevision != 0 {
			return contract.ProjectRequirementBaselineResponse{}, application.ErrProjectRequirementNotFound
		}
		current, currentRevision, err = requirementDomain.NewBaseline(
			command.BaselineID,
			strings.TrimSpace(command.WorkspaceID),
			strings.TrimSpace(command.ProjectID),
			command.Content,
			command.ChangeSummary,
			strings.TrimSpace(command.Actor.ID),
			command.OccurredAt,
		)
		if err != nil {
			return contract.ProjectRequirementBaselineResponse{}, mapProjectRequirementDomainError(err)
		}
		if err = insertProjectRequirementBaseline(ctx, connection, current); err != nil {
			if isProjectResourceConstraint(err) {
				return contract.ProjectRequirementBaselineResponse{}, application.ErrProjectRequirementConflict
			}
			return contract.ProjectRequirementBaselineResponse{}, err
		}
	case readErr != nil:
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("read current Project Requirement: %w", readErr)
	default:
		current, currentRevision, err = current.SaveDraft(
			command.ExpectedRevision,
			command.Content,
			command.ChangeSummary,
			strings.TrimSpace(command.Actor.ID),
			command.MaterialChange,
			command.OccurredAt,
		)
		if err != nil {
			return contract.ProjectRequirementBaselineResponse{}, mapProjectRequirementDomainError(err)
		}
		if currentRevision.Action == requirementDomain.ActionMaterialChange {
			if err = insertProjectRequirementReviewProjections(ctx, connection, current, command.OccurredAt); err != nil {
				return contract.ProjectRequirementBaselineResponse{}, err
			}
		}
		if err = updateProjectRequirementBaseline(ctx, connection, current, command.ExpectedRevision); err != nil {
			return contract.ProjectRequirementBaselineResponse{}, err
		}
	}

	if err = insertProjectRequirementRevision(ctx, connection, currentRevision); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	result, err = readProjectRequirementResponseOnConnection(ctx, connection, current, authority.projection())
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	requestID := projectRequirementRequestID(current.ID, string(currentRevision.Action), current.CurrentRevision)
	if command.ExpectedRevision == 0 {
		requestID = command.IdempotencyKey
	}
	if err = insertProjectRequirementAudit(ctx, connection, current, currentRevision.Action, command.Actor, requestID, command.OccurredAt); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if err = insertProjectRequirementOutbox(ctx, connection, current, currentRevision.Action, command.Actor, command.OccurredAt); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if command.ExpectedRevision == 0 {
		responseBody, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("encode Project Requirement replay: %w", marshalErr)
		}
		if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency(
			workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,
			resource_revision,response_status,response_body,created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`, command.WorkspaceID, "workspace.requirement.create", command.IdempotencyKey,
			command.RequestHash, "requirement_baseline", current.ID, current.CurrentRevision, 201,
			string(responseBody), command.OccurredAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("record Project Requirement replay: %w", err)
		}
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("commit Project Requirement save: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRequirementRepository) TransitionProjectRequirement(ctx context.Context, command application.ProjectRequirementTransition) (result contract.ProjectRequirementBaselineResponse, err error) {
	connection, err := r.projectRequirementConnection(ctx, "transition")
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectRequirementConnection(connection, &committed)()

	authority, err := loadProjectRequirementAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	action := strings.TrimSpace(command.Action)
	switch action {
	case "submit_review", "withdraw_review":
		if !authority.canEdit() {
			return contract.ProjectRequirementBaselineResponse{}, contract.ErrWorkspacePermissionDenied
		}
	case "approve", "freeze", "retire":
		if !authority.canApprove() {
			return contract.ProjectRequirementBaselineResponse{}, contract.ErrWorkspacePermissionDenied
		}
	default:
		return contract.ProjectRequirementBaselineResponse{}, application.ErrInvalidProjectRequirementRequest
	}
	current, _, err := readProjectRequirementBaselineOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRequirementBaselineResponse{}, application.ErrProjectRequirementNotFound
	}
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("read Project Requirement transition: %w", err)
	}
	var revision requirementDomain.Revision
	switch action {
	case "submit_review":
		current, revision, err = current.SubmitReview(command.ExpectedRevision, command.Actor.ID, command.OccurredAt)
	case "withdraw_review":
		current, revision, err = current.WithdrawReview(command.ExpectedRevision, command.Actor.ID, command.OccurredAt)
	case "approve":
		current, revision, err = current.Approve(command.ExpectedRevision, command.Actor.ID, command.OccurredAt)
	case "freeze":
		current, revision, err = current.Freeze(command.ExpectedRevision, command.Actor.ID, command.OccurredAt)
	case "retire":
		current, revision, err = current.Retire(command.ExpectedRevision, command.Actor.ID, command.OccurredAt)
	}
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, mapProjectRequirementDomainError(err)
	}
	if err = updateProjectRequirementBaseline(ctx, connection, current, command.ExpectedRevision); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if err = insertProjectRequirementRevision(ctx, connection, revision); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	requestID := projectRequirementRequestID(current.ID, action, current.CurrentRevision)
	if err = insertProjectRequirementAudit(ctx, connection, current, revision.Action, command.Actor, requestID, command.OccurredAt); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if err = insertProjectRequirementOutbox(ctx, connection, current, revision.Action, command.Actor, command.OccurredAt); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	result, err = readProjectRequirementResponseOnConnection(ctx, connection, current, authority.projection())
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("commit Project Requirement transition: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRequirementRepository) MutateProjectRequirementLink(ctx context.Context, command application.ProjectRequirementLinkMutation) (result contract.ProjectRequirementBaselineResponse, err error) {
	connection, err := r.projectRequirementConnection(ctx, "link")
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectRequirementConnection(connection, &committed)()

	authority, err := loadProjectRequirementAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	targetKind := strings.TrimSpace(command.TargetKind)
	switch targetKind {
	case "issue":
		if !authority.canEdit() {
			return contract.ProjectRequirementBaselineResponse{}, contract.ErrWorkspacePermissionDenied
		}
	case "outline":
		if !authority.canManageOutline() {
			return contract.ProjectRequirementBaselineResponse{}, contract.ErrWorkspacePermissionDenied
		}
	default:
		return contract.ProjectRequirementBaselineResponse{}, application.ErrInvalidProjectRequirementRequest
	}
	current, _, err := readProjectRequirementBaselineOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRequirementBaselineResponse{}, application.ErrProjectRequirementNotFound
	}
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("read Project Requirement link baseline: %w", err)
	}
	if command.ExpectedRevision != current.CurrentRevision {
		return contract.ProjectRequirementBaselineResponse{}, contract.RevisionConflictError{CurrentRevision: current.CurrentRevision}
	}
	key, targetID := strings.TrimSpace(command.RequirementKey), strings.TrimSpace(command.TargetID)
	if key == "" || targetID == "" {
		return contract.ProjectRequirementBaselineResponse{}, application.ErrInvalidProjectRequirementRequest
	}
	if _, ok := current.Content.TraceableSection(key); !ok {
		return contract.ProjectRequirementBaselineResponse{}, application.ErrInvalidProjectRequirementRequest
	}
	if err = validateProjectRequirementLinkTarget(ctx, connection, command.WorkspaceID, command.ProjectID, targetKind, targetID); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	action := requirementDomain.ActionLinkIssue
	if command.Unlink {
		action = requirementDomain.ActionUnlinkIssue
	}
	if targetKind == "outline" {
		action = requirementDomain.ActionLinkOutline
		if command.Unlink {
			action = requirementDomain.ActionUnlinkOutline
		}
	}
	current, revision, err := current.RecordTraceabilityMutation(command.ExpectedRevision, action, command.Actor.ID, command.OccurredAt)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, mapProjectRequirementDomainError(err)
	}
	if command.Unlink {
		err = closeProjectRequirementLink(ctx, connection, current, targetKind, key, targetID, command.Actor.ID, command.OccurredAt)
	} else {
		err = insertProjectRequirementLink(ctx, connection, current, targetKind, key, targetID, command.Actor.ID, command.OccurredAt)
	}
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if err = updateProjectRequirementBaseline(ctx, connection, current, command.ExpectedRevision); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if err = insertProjectRequirementRevision(ctx, connection, revision); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	requestID := projectRequirementRequestID(current.ID, string(action), current.CurrentRevision)
	if err = insertProjectRequirementAudit(ctx, connection, current, action, command.Actor, requestID, command.OccurredAt); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if err = insertProjectRequirementOutbox(ctx, connection, current, action, command.Actor, command.OccurredAt); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	result, err = readProjectRequirementResponseOnConnection(ctx, connection, current, authority.projection())
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("commit Project Requirement link: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRequirementRepository) ReplaceProjectRequirementAccess(ctx context.Context, command application.ProjectRequirementAccessReplace) (result contract.ProjectRequirementAccessSet, err error) {
	connection, err := r.projectRequirementConnection(ctx, "replace access")
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectRequirementConnection(connection, &committed)()
	authority, err := loadProjectRequirementAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if !authority.active() || authority.membership.Role != "owner" {
		return contract.ProjectRequirementAccessSet{}, contract.ErrWorkspacePermissionDenied
	}
	currentRevision, err := readProjectRequirementSetRevision(ctx, connection, "workspace_project_requirement_access_sets", command.WorkspaceID, command.ProjectID)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if currentRevision != command.ExpectedRevision {
		return contract.ProjectRequirementAccessSet{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	grants, err := validateProjectRequirementGrants(ctx, connection, command.WorkspaceID, command.Grants)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if _, err = connection.ExecContext(ctx, `DELETE FROM workspace_project_requirement_grants WHERE workspace_id=? AND project_id=?`, command.WorkspaceID, command.ProjectID); err != nil {
		return contract.ProjectRequirementAccessSet{}, fmt.Errorf("clear Project Requirement grants: %w", err)
	}
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	for _, grant := range grants {
		if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_requirement_grants(
			workspace_id,project_id,member_id,grant_kind,granted_by,granted_at
		) VALUES(?,?,?,?,?,?)`, command.WorkspaceID, command.ProjectID, grant.MemberID, grant.GrantKind, command.Actor.ID, timestamp); err != nil {
			return contract.ProjectRequirementAccessSet{}, fmt.Errorf("insert Project Requirement grant: %w", err)
		}
	}
	nextRevision := currentRevision + 1
	if err = writeProjectRequirementSetRevision(ctx, connection, "workspace_project_requirement_access_sets", command.WorkspaceID, command.ProjectID, currentRevision, nextRevision, timestamp); err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	result, err = readProjectRequirementAccessOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if err = insertProjectRequirementGovernance(ctx, connection, command.WorkspaceID, "project_requirement_access", command.ProjectID,
		nextRevision, "workspace.requirement.access.replace", "requirement:access_replaced", command.Actor, command.OccurredAt,
		map[string]any{"grant_count": len(grants)}); err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectRequirementAccessSet{}, fmt.Errorf("commit Project Requirement access: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRequirementRepository) ReadProjectRequirementAccess(ctx context.Context, workspaceID, projectID string, actor contract.WorkspaceActor) (contract.ProjectRequirementAccessSet, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	defer connection.Close()
	if _, err = loadProjectRequirementAuthority(ctx, connection, workspaceID, projectID, actor); err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	return readProjectRequirementAccessOnConnection(ctx, connection, workspaceID, projectID)
}

func (r *ProjectRequirementRepository) CreateProjectOutlineNode(ctx context.Context, command application.ProjectOutlineNodeCreate) (result contract.ProjectOutline, err error) {
	connection, err := r.projectRequirementConnection(ctx, "create outline root")
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackProjectRequirementConnection(connection, &committed)()
	authority, err := loadProjectRequirementAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	if !authority.canManageOutline() {
		return contract.ProjectOutline{}, contract.ErrWorkspacePermissionDenied
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" || len(command.IdempotencyKey) > 200 || len(command.RequestHash) != 64 {
		return contract.ProjectOutline{}, application.ErrInvalidProjectRequirementRequest
	}
	var storedHash, responseBody string
	replayErr := connection.QueryRowContext(ctx, `SELECT request_hash,response_body FROM workspace_mutation_idempotency
		WHERE workspace_id=? AND action='workspace.project.outline.create' AND idempotency_key=?`, command.WorkspaceID, command.IdempotencyKey).Scan(&storedHash, &responseBody)
	if replayErr == nil {
		if storedHash != command.RequestHash {
			return contract.ProjectOutline{}, contract.ErrIdempotencyConflict
		}
		if err = json.Unmarshal([]byte(responseBody), &result); err != nil {
			return contract.ProjectOutline{}, fmt.Errorf("decode Project outline replay: %w", err)
		}
		if _, err = connection.ExecContext(ctx, "ROLLBACK"); err != nil {
			return contract.ProjectOutline{}, err
		}
		committed = true
		return result, nil
	}
	if !errors.Is(replayErr, sql.ErrNoRows) {
		return contract.ProjectOutline{}, fmt.Errorf("read Project outline replay: %w", replayErr)
	}
	currentRevision, err := readProjectRequirementSetRevision(ctx, connection, "workspace_project_outline_sets", command.WorkspaceID, command.ProjectID)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	if currentRevision != command.ExpectedRevision {
		return contract.ProjectOutline{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	nodeID, title := strings.TrimSpace(command.NodeID), strings.TrimSpace(command.Title)
	if nodeID == "" || title == "" || len(title) > 500 {
		return contract.ProjectOutline{}, application.ErrInvalidProjectRequirementRequest
	}
	var count int
	if err = connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_project_outline_nodes WHERE workspace_id=? AND project_id=?`, command.WorkspaceID, command.ProjectID).Scan(&count); err != nil {
		return contract.ProjectOutline{}, err
	}
	if count >= 2000 {
		return contract.ProjectOutline{}, application.ErrProjectRequirementConflict
	}
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_outline_nodes(id,workspace_id,project_id,title,created_by,created_at)
		VALUES(?,?,?,?,?,?)`, nodeID, command.WorkspaceID, command.ProjectID, title, command.Actor.ID, timestamp); err != nil {
		if isProjectResourceConstraint(err) {
			return contract.ProjectOutline{}, application.ErrProjectRequirementConflict
		}
		return contract.ProjectOutline{}, fmt.Errorf("insert Project outline root: %w", err)
	}
	nextRevision := currentRevision + 1
	if err = writeProjectRequirementSetRevision(ctx, connection, "workspace_project_outline_sets", command.WorkspaceID, command.ProjectID, currentRevision, nextRevision, timestamp); err != nil {
		return contract.ProjectOutline{}, err
	}
	result, err = readProjectOutlineOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	if err = insertProjectRequirementGovernance(ctx, connection, command.WorkspaceID, "project_outline", command.ProjectID,
		nextRevision, "workspace.project.outline.create", "project_outline:root_created", command.Actor, command.OccurredAt,
		map[string]any{"node_id": nodeID}); err != nil {
		return contract.ProjectOutline{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency(
		workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,
		response_status,response_body,created_at
	) VALUES(?,?,?,?,?,?,?,?,?,?)`, command.WorkspaceID, "workspace.project.outline.create", command.IdempotencyKey,
		command.RequestHash, "project_outline", command.ProjectID, nextRevision, 201, string(encoded), timestamp); err != nil {
		return contract.ProjectOutline{}, fmt.Errorf("record Project outline replay: %w", err)
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectOutline{}, fmt.Errorf("commit Project outline root: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRequirementRepository) ReadProjectOutline(ctx context.Context, workspaceID, projectID string, actor contract.WorkspaceActor) (contract.ProjectOutline, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	defer connection.Close()
	if _, err = loadProjectRequirementAuthority(ctx, connection, workspaceID, projectID, actor); err != nil {
		return contract.ProjectOutline{}, err
	}
	return readProjectOutlineOnConnection(ctx, connection, workspaceID, projectID)
}

func (r *ProjectRequirementRepository) ReadProjectRequirement(ctx context.Context, workspaceID, projectID string, actor contract.WorkspaceActor) (contract.ProjectRequirementBaselineResponse, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, fmt.Errorf("acquire Project Requirement read connection: %w", err)
	}
	defer connection.Close()
	authority, err := loadProjectRequirementAuthority(ctx, connection, workspaceID, projectID, actor)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	baseline, _, err := readProjectRequirementBaselineOnConnection(ctx, connection, workspaceID, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRequirementBaselineResponse{History: []contract.ProjectRequirementRevision{}, IssueLinks: []contract.ProjectRequirementIssueLink{}, OutlineLinks: []contract.ProjectRequirementOutlineLink{}, Access: authority.projection()}, nil
	}
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	return readProjectRequirementResponseOnConnection(ctx, connection, baseline, authority.projection())
}

func (r *ProjectRequirementRepository) ReadProjectRequirementCoverage(ctx context.Context, workspaceID, projectID string, actor contract.WorkspaceActor) (result contract.ProjectRequirementCoverage, err error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRequirementCoverage{}, fmt.Errorf("acquire Project Requirement coverage connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, "BEGIN"); err != nil {
		return contract.ProjectRequirementCoverage{}, fmt.Errorf("begin Project Requirement coverage read: %w", err)
	}
	committed := false
	defer rollbackProjectRequirementConnection(connection, &committed)()
	if _, err = loadProjectRequirementAuthority(ctx, connection, workspaceID, projectID, actor); err != nil {
		return contract.ProjectRequirementCoverage{}, err
	}
	baseline, currentRevision, err := readProjectRequirementBaselineOnConnection(ctx, connection, workspaceID, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
			return contract.ProjectRequirementCoverage{}, fmt.Errorf("commit empty Project Requirement coverage read: %w", err)
		}
		committed = true
		return contract.ProjectRequirementCoverage{}, nil
	}
	if err != nil {
		return contract.ProjectRequirementCoverage{}, fmt.Errorf("read Project Requirement coverage baseline: %w", err)
	}
	status := string(baseline.Status)
	result.BaselineStatus = &status
	current, err := readProjectRequirementCoverageSnapshotOnConnection(ctx, connection, baseline.ID, currentRevision)
	if err != nil {
		return contract.ProjectRequirementCoverage{}, err
	}
	result.Current = &current
	if baseline.EffectiveRevision != nil {
		effectiveRevision := currentRevision
		if *baseline.EffectiveRevision != currentRevision.Revision {
			effectiveRevision, err = readProjectRequirementRevisionOnConnection(ctx, connection, baseline.ID, *baseline.EffectiveRevision)
			if errors.Is(err, sql.ErrNoRows) {
				return contract.ProjectRequirementCoverage{}, errors.New("Project Requirement coverage effective revision is missing")
			}
			if err != nil {
				return contract.ProjectRequirementCoverage{}, err
			}
		}
		effective, snapshotErr := readProjectRequirementCoverageSnapshotOnConnection(ctx, connection, baseline.ID, effectiveRevision)
		if snapshotErr != nil {
			return contract.ProjectRequirementCoverage{}, snapshotErr
		}
		result.Effective = &effective
	}
	if _, err = connection.ExecContext(ctx, "COMMIT"); err != nil {
		return contract.ProjectRequirementCoverage{}, fmt.Errorf("commit Project Requirement coverage read: %w", err)
	}
	committed = true
	return result, nil
}

type projectRequirementAuthority struct {
	membership projectResourceMembershipAuthority
	actorID    string
	leadType   string
	leadID     string
	editor     bool
	approver   bool
	status     string
}

func loadProjectRequirementAuthority(ctx context.Context, connection *sql.Conn, workspaceID, projectID string, actor contract.WorkspaceActor) (projectRequirementAuthority, error) {
	membership, err := projectResourceMembershipOnConnection(ctx, connection, strings.TrimSpace(workspaceID), actor)
	if err != nil {
		return projectRequirementAuthority{}, err
	}
	var authority projectRequirementAuthority
	authority.membership = membership
	authority.actorID = strings.TrimSpace(actor.ID)
	var leadType, leadID sql.NullString
	if err = connection.QueryRowContext(ctx, `SELECT status,lead_type,lead_id FROM workspace_projects
		WHERE workspace_id=? AND id=?`, strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)).Scan(&authority.status, &leadType, &leadID); errors.Is(err, sql.ErrNoRows) {
		return projectRequirementAuthority{}, application.ErrProjectSurfaceNotFound
	} else if err != nil {
		return projectRequirementAuthority{}, fmt.Errorf("read Project Requirement project: %w", err)
	}
	authority.leadType, authority.leadID = nullableText(leadType), nullableText(leadID)
	rows, err := connection.QueryContext(ctx, `SELECT grant_kind FROM workspace_project_requirement_grants
		WHERE workspace_id=? AND project_id=? AND member_id=?`, workspaceID, projectID, membership.MemberID)
	if err != nil {
		return projectRequirementAuthority{}, fmt.Errorf("read Project Requirement grants: %w", err)
	}
	for rows.Next() {
		var kind string
		if err = rows.Scan(&kind); err != nil {
			_ = rows.Close()
			return projectRequirementAuthority{}, err
		}
		switch kind {
		case "project_editor":
			authority.editor = true
		case "requirement_approver":
			authority.approver = true
		}
	}
	if err = rows.Close(); err != nil {
		return projectRequirementAuthority{}, err
	}
	return authority, nil
}

func (a projectRequirementAuthority) isLead() bool {
	return a.leadType == "member" && a.leadID != "" &&
		(a.leadID == a.actorID || a.leadID == a.membership.MemberID || a.leadID == a.membership.UserID)
}

func (a projectRequirementAuthority) active() bool {
	return a.status != "completed" && a.status != "cancelled"
}

func (a projectRequirementAuthority) canEdit() bool {
	return a.active() && (a.membership.Role == "owner" || a.membership.Role == "admin" || a.isLead() || a.editor)
}

func (a projectRequirementAuthority) canApprove() bool {
	return a.active() && (a.membership.Role == "owner" || a.approver)
}

func (a projectRequirementAuthority) canManageOutline() bool {
	return a.active() && (a.membership.Role == "owner" || a.membership.Role == "admin" || a.editor)
}

func (a projectRequirementAuthority) projection() contract.ProjectRequirementAccessProjection {
	return contract.ProjectRequirementAccessProjection{
		CanEdit: a.canEdit(), CanApprove: a.canApprove(),
		CanManageAccess: a.membership.Role == "owner", CanManageOutline: a.canManageOutline(),
	}
}

func (r *ProjectRequirementRepository) projectRequirementConnection(ctx context.Context, operation string) (*sql.Conn, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire Project Requirement %s connection: %w", operation, err)
	}
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		connection.Close()
		return nil, fmt.Errorf("configure Project Requirement %s connection: %w", operation, err)
	}
	if _, err = connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		connection.Close()
		return nil, fmt.Errorf("begin Project Requirement %s: %w", operation, err)
	}
	return connection, nil
}

func rollbackProjectRequirementConnection(connection *sql.Conn, committed *bool) func() {
	return func() {
		if !*committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}
}

func insertProjectRequirementBaseline(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline) error {
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_requirement_baselines(
		id,workspace_id,project_id,status,current_revision,approved_revision,effective_revision,
		review_origin,latest_content_author,submitted_by,submitted_at,approved_by,approved_at,
		frozen_by,frozen_at,retired_by,retired_at,legacy_requirement_id,legacy_snapshot_json,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, baseline.ID, baseline.WorkspaceID, baseline.ProjectID,
		baseline.Status, baseline.CurrentRevision, baseline.ApprovedRevision, baseline.EffectiveRevision,
		nullableStatusValue(baseline.ReviewOrigin), baseline.LatestContentAuthor, baseline.SubmittedBy,
		formatOptionalTime(baseline.SubmittedAt), baseline.ApprovedBy, formatOptionalTime(baseline.ApprovedAt),
		baseline.FrozenBy, formatOptionalTime(baseline.FrozenAt), baseline.RetiredBy, formatOptionalTime(baseline.RetiredAt),
		baseline.LegacyRequirementID, nil, baseline.CreatedAt.Format(time.RFC3339Nano), baseline.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert Project Requirement baseline: %w", err)
	}
	return nil
}

func updateProjectRequirementBaseline(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, expectedRevision int64) error {
	result, err := connection.ExecContext(ctx, `UPDATE workspace_requirement_baselines SET
		status=?,current_revision=?,approved_revision=?,effective_revision=?,review_origin=?,latest_content_author=?,
		submitted_by=?,submitted_at=?,approved_by=?,approved_at=?,frozen_by=?,frozen_at=?,retired_by=?,retired_at=?,updated_at=?
		WHERE workspace_id=? AND project_id=? AND id=? AND current_revision=?`, baseline.Status, baseline.CurrentRevision,
		baseline.ApprovedRevision, baseline.EffectiveRevision, nullableStatusValue(baseline.ReviewOrigin), baseline.LatestContentAuthor,
		baseline.SubmittedBy, formatOptionalTime(baseline.SubmittedAt), baseline.ApprovedBy, formatOptionalTime(baseline.ApprovedAt),
		baseline.FrozenBy, formatOptionalTime(baseline.FrozenAt), baseline.RetiredBy, formatOptionalTime(baseline.RetiredAt),
		baseline.UpdatedAt.Format(time.RFC3339Nano), baseline.WorkspaceID, baseline.ProjectID, baseline.ID, expectedRevision)
	if err != nil {
		return fmt.Errorf("update Project Requirement baseline: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return contract.RevisionConflictError{CurrentRevision: expectedRevision}
	}
	return nil
}

func insertProjectRequirementRevision(ctx context.Context, connection *sql.Conn, revision requirementDomain.Revision) error {
	content, err := json.Marshal(revision.Content)
	if err != nil {
		return fmt.Errorf("encode Project Requirement revision: %w", err)
	}
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_requirement_revisions(
		baseline_id,revision,content_json,status,action,change_summary,actor_id,submitted_by,
		submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, revision.BaselineID, revision.Revision, string(content), revision.Status,
		revision.Action, revision.ChangeSummary, revision.ActorID, revision.SubmittedBy, formatOptionalTime(revision.SubmittedAt),
		revision.ApprovedBy, formatOptionalTime(revision.ApprovedAt), revision.FrozenBy, formatOptionalTime(revision.FrozenAt),
		revision.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert Project Requirement revision: %w", err)
	}
	return nil
}

func validateProjectRequirementLinkTarget(ctx context.Context, connection *sql.Conn, workspaceID, projectID, targetKind, targetID string) error {
	var found string
	var err error
	switch targetKind {
	case "issue":
		err = connection.QueryRowContext(ctx, `SELECT id FROM workspace_issues WHERE workspace_id=? AND project_id=? AND id=?`, workspaceID, projectID, targetID).Scan(&found)
	case "outline":
		err = connection.QueryRowContext(ctx, `SELECT id FROM workspace_project_outline_nodes WHERE workspace_id=? AND project_id=? AND id=?`, workspaceID, projectID, targetID).Scan(&found)
	default:
		return application.ErrInvalidProjectRequirementRequest
	}
	if errors.Is(err, sql.ErrNoRows) {
		return application.ErrProjectRequirementNotFound
	}
	if err != nil {
		return fmt.Errorf("validate Project Requirement %s link: %w", targetKind, err)
	}
	return nil
}

func insertProjectRequirementLink(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, targetKind, key, targetID, actorID string, occurredAt time.Time) error {
	table, targetColumn := "workspace_requirement_issue_links", "issue_id"
	if targetKind == "outline" {
		table, targetColumn = "workspace_requirement_outline_links", "node_id"
	}
	var active int
	query := `SELECT COUNT(*) FROM ` + table + ` WHERE baseline_id=? AND requirement_key=? AND ` + targetColumn + `=? AND unlinked_revision IS NULL`
	if err := connection.QueryRowContext(ctx, query, baseline.ID, key, targetID).Scan(&active); err != nil {
		return fmt.Errorf("inspect active Project Requirement link: %w", err)
	}
	if active != 0 {
		return application.ErrProjectRequirementConflict
	}
	statement := `INSERT INTO ` + table + `(
		workspace_id,project_id,baseline_id,requirement_key,` + targetColumn + `,linked_revision,
		unlinked_revision,linked_by,linked_at,unlinked_by,unlinked_at
	) VALUES(?,?,?,?,?,?,NULL,?,?,NULL,NULL)`
	if _, err := connection.ExecContext(ctx, statement, baseline.WorkspaceID, baseline.ProjectID, baseline.ID, key, targetID,
		baseline.CurrentRevision, actorID, occurredAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert Project Requirement %s link: %w", targetKind, err)
	}
	return nil
}

func closeProjectRequirementLink(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, targetKind, key, targetID, actorID string, occurredAt time.Time) error {
	table, targetColumn := "workspace_requirement_issue_links", "issue_id"
	if targetKind == "outline" {
		table, targetColumn = "workspace_requirement_outline_links", "node_id"
	}
	statement := `UPDATE ` + table + ` SET unlinked_revision=?,unlinked_by=?,unlinked_at=?
		WHERE baseline_id=? AND requirement_key=? AND ` + targetColumn + `=? AND unlinked_revision IS NULL`
	result, err := connection.ExecContext(ctx, statement, baseline.CurrentRevision, actorID, occurredAt.UTC().Format(time.RFC3339Nano), baseline.ID, key, targetID)
	if err != nil {
		return fmt.Errorf("unlink Project Requirement %s: %w", targetKind, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return application.ErrProjectRequirementNotFound
	}
	return nil
}

func insertProjectRequirementReviewProjections(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, occurredAt time.Time) error {
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_requirement_review_projections(
		baseline_id,requirement_key,issue_id,source_revision,status,created_at
	) SELECT baseline_id,requirement_key,issue_id,?,'review_required',?
		FROM workspace_requirement_issue_links
		WHERE baseline_id=? AND unlinked_revision IS NULL`, baseline.CurrentRevision,
		occurredAt.UTC().Format(time.RFC3339Nano), baseline.ID)
	if err != nil {
		return fmt.Errorf("project Requirement material review impact: %w", err)
	}
	return nil
}

type validatedProjectRequirementGrant struct {
	MemberID  string
	GrantKind string
}

func validateProjectRequirementGrants(ctx context.Context, connection *sql.Conn, workspaceID string, requested []application.ProjectRequirementGrantChange) ([]validatedProjectRequirementGrant, error) {
	if len(requested) > 200 {
		return nil, application.ErrInvalidProjectRequirementRequest
	}
	result := make([]validatedProjectRequirementGrant, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, value := range requested {
		memberID, kind := strings.TrimSpace(value.MemberID), strings.TrimSpace(value.GrantKind)
		if memberID == "" || (kind != "project_editor" && kind != "requirement_approver") {
			return nil, application.ErrInvalidProjectRequirementRequest
		}
		key := memberID + "\x00" + kind
		if _, duplicate := seen[key]; duplicate {
			return nil, application.ErrInvalidProjectRequirementRequest
		}
		seen[key] = struct{}{}
		var role string
		if err := connection.QueryRowContext(ctx, `SELECT role FROM auth_members WHERE workspace_id=? AND id=?`, workspaceID, memberID).Scan(&role); errors.Is(err, sql.ErrNoRows) {
			return nil, application.ErrProjectRequirementNotFound
		} else if err != nil {
			return nil, fmt.Errorf("validate Project Requirement grant member: %w", err)
		}
		if (kind == "project_editor" && role != "member") || (kind == "requirement_approver" && role != "member" && role != "admin") {
			return nil, application.ErrInvalidProjectRequirementRequest
		}
		result = append(result, validatedProjectRequirementGrant{MemberID: memberID, GrantKind: kind})
	}
	return result, nil
}

func readProjectRequirementSetRevision(ctx context.Context, connection *sql.Conn, table, workspaceID, projectID string) (int64, error) {
	var revision int64
	err := connection.QueryRowContext(ctx, `SELECT revision FROM `+table+` WHERE workspace_id=? AND project_id=?`, workspaceID, projectID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read Project Requirement set revision: %w", err)
	}
	return revision, nil
}

func writeProjectRequirementSetRevision(ctx context.Context, connection *sql.Conn, table, workspaceID, projectID string, expected, next int64, timestamp string) error {
	if _, err := connection.ExecContext(ctx, `INSERT OR IGNORE INTO `+table+`(workspace_id,project_id,revision,updated_at) VALUES(?,?,0,?)`, workspaceID, projectID, timestamp); err != nil {
		return fmt.Errorf("initialize Project Requirement set revision: %w", err)
	}
	result, err := connection.ExecContext(ctx, `UPDATE `+table+` SET revision=?,updated_at=? WHERE workspace_id=? AND project_id=? AND revision=?`, next, timestamp, workspaceID, projectID, expected)
	if err != nil {
		return fmt.Errorf("advance Project Requirement set revision: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		current, readErr := readProjectRequirementSetRevision(ctx, connection, table, workspaceID, projectID)
		if readErr != nil {
			return readErr
		}
		return contract.RevisionConflictError{CurrentRevision: current}
	}
	return nil
}

func readProjectRequirementAccessOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID string) (contract.ProjectRequirementAccessSet, error) {
	revision, err := readProjectRequirementSetRevision(ctx, connection, "workspace_project_requirement_access_sets", workspaceID, projectID)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	rows, err := connection.QueryContext(ctx, `SELECT g.member_id,m.user_id,m.role,g.grant_kind,g.granted_by,g.granted_at
		FROM workspace_project_requirement_grants g
		JOIN auth_members m ON m.workspace_id=g.workspace_id AND m.id=g.member_id
		WHERE g.workspace_id=? AND g.project_id=?
		ORDER BY g.member_id,g.grant_kind`, workspaceID, projectID)
	if err != nil {
		return contract.ProjectRequirementAccessSet{}, fmt.Errorf("read Project Requirement grants: %w", err)
	}
	defer rows.Close()
	grants := make([]contract.ProjectRequirementGrant, 0)
	for rows.Next() {
		var grant contract.ProjectRequirementGrant
		if err = rows.Scan(&grant.MemberID, &grant.UserID, &grant.Role, &grant.GrantKind, &grant.GrantedBy, &grant.GrantedAt); err != nil {
			return contract.ProjectRequirementAccessSet{}, err
		}
		grants = append(grants, grant)
	}
	if err = rows.Err(); err != nil {
		return contract.ProjectRequirementAccessSet{}, err
	}
	return contract.ProjectRequirementAccessSet{Revision: revision, Grants: grants}, nil
}

func readProjectOutlineOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID string) (contract.ProjectOutline, error) {
	revision, err := readProjectRequirementSetRevision(ctx, connection, "workspace_project_outline_sets", workspaceID, projectID)
	if err != nil {
		return contract.ProjectOutline{}, err
	}
	rows, err := connection.QueryContext(ctx, `SELECT id,workspace_id,project_id,title,created_by,created_at
		FROM workspace_project_outline_nodes WHERE workspace_id=? AND project_id=? ORDER BY created_at,id`, workspaceID, projectID)
	if err != nil {
		return contract.ProjectOutline{}, fmt.Errorf("read Project outline roots: %w", err)
	}
	defer rows.Close()
	nodes := make([]contract.ProjectOutlineNode, 0)
	for rows.Next() {
		var node contract.ProjectOutlineNode
		if err = rows.Scan(&node.ID, &node.WorkspaceID, &node.ProjectID, &node.Title, &node.CreatedBy, &node.CreatedAt); err != nil {
			return contract.ProjectOutline{}, err
		}
		nodes = append(nodes, node)
	}
	if err = rows.Err(); err != nil {
		return contract.ProjectOutline{}, err
	}
	return contract.ProjectOutline{Revision: revision, Nodes: nodes}, nil
}

func insertProjectRequirementGovernance(ctx context.Context, connection *sql.Conn, workspaceID, resourceKind, resourceID string, revision int64, action, eventType string, actor contract.WorkspaceActor, occurredAt time.Time, metadata map[string]any) error {
	timestamp := occurredAt.UTC().Format(time.RFC3339Nano)
	metadataJSON, _ := json.Marshal(metadata)
	auditID := projectRequirementRequestID(resourceID, strings.ReplaceAll(action, ".", "-"), revision)
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_audit_entries(
		workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,
		resource_revision,request_id,metadata_json
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, workspaceID, timestamp, auditID, actor.Type, actor.ID, action,
		resourceKind, resourceID, revision, auditID, string(metadataJSON)); err != nil {
		return fmt.Errorf("record Project Requirement governance audit: %w", err)
	}
	payloadJSON, _ := json.Marshal(map[string]any{"resource_id": resourceID, "revision": revision, "metadata": metadata})
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_outbox_events(
		state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,
		payload_json,actor_type,actor_id,created_at
	) VALUES('ready',?,?,?,?,?,?,?,?,?,?,?)`, timestamp, workspaceID,
		projectRequirementRequestID(resourceID, "outbox-"+eventType, revision), eventType, resourceKind,
		resourceID, revision, string(payloadJSON), actor.Type, actor.ID, timestamp); err != nil {
		return fmt.Errorf("record Project Requirement governance outbox: %w", err)
	}
	return nil
}

func readProjectRequirementBaselineOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID string) (requirementDomain.Baseline, requirementDomain.Revision, error) {
	var baseline requirementDomain.Baseline
	var status, reviewOrigin string
	var approvedRevision, effectiveRevision sql.NullInt64
	var submittedBy, submittedAt, approvedBy, approvedAt, frozenBy, frozenAt, retiredBy, retiredAt, legacyID sql.NullString
	var createdAt, updatedAt string
	err := connection.QueryRowContext(ctx, `SELECT id,workspace_id,project_id,status,current_revision,
		approved_revision,effective_revision,COALESCE(review_origin,''),latest_content_author,
		submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,retired_by,retired_at,
		legacy_requirement_id,created_at,updated_at
		FROM workspace_requirement_baselines WHERE workspace_id=? AND project_id=?`, workspaceID, projectID).Scan(
		&baseline.ID, &baseline.WorkspaceID, &baseline.ProjectID, &status, &baseline.CurrentRevision,
		&approvedRevision, &effectiveRevision, &reviewOrigin, &baseline.LatestContentAuthor,
		&submittedBy, &submittedAt, &approvedBy, &approvedAt, &frozenBy, &frozenAt, &retiredBy, &retiredAt,
		&legacyID, &createdAt, &updatedAt)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, err
	}
	baseline.Status = requirementDomain.Status(status)
	baseline.ReviewOrigin = requirementDomain.Status(reviewOrigin)
	baseline.ApprovedRevision = nullableInt64Pointer(approvedRevision)
	baseline.EffectiveRevision = nullableInt64Pointer(effectiveRevision)
	baseline.SubmittedBy, baseline.ApprovedBy, baseline.FrozenBy, baseline.RetiredBy = nullableStringPointer(submittedBy), nullableStringPointer(approvedBy), nullableStringPointer(frozenBy), nullableStringPointer(retiredBy)
	baselineTimes, err := parseProjectRequirementOptionalTimes(submittedAt, approvedAt, frozenAt, retiredAt)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, fmt.Errorf("decode Project Requirement baseline lifecycle time: %w", err)
	}
	baseline.SubmittedAt, baseline.ApprovedAt, baseline.FrozenAt, baseline.RetiredAt = baselineTimes[0], baselineTimes[1], baselineTimes[2], baselineTimes[3]
	baseline.LegacyRequirementID = nullableStringPointer(legacyID)
	baseline.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, err
	}
	baseline.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, err
	}
	var revision requirementDomain.Revision
	var contentJSON, revisionStatus, action, revisionCreated string
	var revisionSubmittedBy, revisionSubmittedAt, revisionApprovedBy, revisionApprovedAt, revisionFrozenBy, revisionFrozenAt sql.NullString
	err = connection.QueryRowContext(ctx, `SELECT baseline_id,revision,content_json,status,action,change_summary,actor_id,
		submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
		FROM workspace_requirement_revisions WHERE baseline_id=? AND revision=?`, baseline.ID, baseline.CurrentRevision).Scan(
		&revision.BaselineID, &revision.Revision, &contentJSON, &revisionStatus, &action, &revision.ChangeSummary, &revision.ActorID,
		&revisionSubmittedBy, &revisionSubmittedAt, &revisionApprovedBy, &revisionApprovedAt, &revisionFrozenBy, &revisionFrozenAt, &revisionCreated)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, err
	}
	if err = json.Unmarshal([]byte(contentJSON), &revision.Content); err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, fmt.Errorf("decode Project Requirement content: %w", err)
	}
	revision.Status, revision.Action = requirementDomain.Status(revisionStatus), requirementDomain.Action(action)
	revision.SubmittedBy, revision.ApprovedBy, revision.FrozenBy = nullableStringPointer(revisionSubmittedBy), nullableStringPointer(revisionApprovedBy), nullableStringPointer(revisionFrozenBy)
	revisionTimes, err := parseProjectRequirementOptionalTimes(revisionSubmittedAt, revisionApprovedAt, revisionFrozenAt)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, fmt.Errorf("decode Project Requirement revision lifecycle time: %w", err)
	}
	revision.SubmittedAt, revision.ApprovedAt, revision.FrozenAt = revisionTimes[0], revisionTimes[1], revisionTimes[2]
	revision.CreatedAt, err = time.Parse(time.RFC3339Nano, revisionCreated)
	if err != nil {
		return requirementDomain.Baseline{}, requirementDomain.Revision{}, err
	}
	baseline.Content = revision.Content
	return baseline, revision, nil
}

func insertProjectRequirementAudit(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, action requirementDomain.Action, actor contract.WorkspaceActor, requestID string, occurredAt time.Time) error {
	metadata, _ := json.Marshal(map[string]any{"version": "project-requirement-v1", "action": action, "status": baseline.Status})
	timestamp := occurredAt.UTC().Format(time.RFC3339Nano)
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_audit_entries(
		workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,
		resource_revision,request_id,metadata_json
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, baseline.WorkspaceID, timestamp,
		projectRequirementRequestID(baseline.ID, string(action), baseline.CurrentRevision), actor.Type, actor.ID,
		"workspace.requirement."+string(action), "requirement_baseline", baseline.ID, baseline.CurrentRevision, requestID, string(metadata))
	if err != nil {
		return fmt.Errorf("record Project Requirement audit: %w", err)
	}
	return nil
}

func insertProjectRequirementOutbox(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, action requirementDomain.Action, actor contract.WorkspaceActor, occurredAt time.Time) error {
	payload, _ := json.Marshal(map[string]any{"baseline_id": baseline.ID, "project_id": baseline.ProjectID, "status": baseline.Status, "revision": baseline.CurrentRevision})
	timestamp := occurredAt.UTC().Format(time.RFC3339Nano)
	_, err := connection.ExecContext(ctx, `INSERT INTO workspace_outbox_events(
		state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,
		payload_json,actor_type,actor_id,created_at
	) VALUES('ready',?,?,?,?,?,?,?,?,?,?,?)`, timestamp, baseline.WorkspaceID,
		projectRequirementRequestID(baseline.ID, "outbox-"+string(action), baseline.CurrentRevision),
		"requirement:"+string(action), "requirement_baseline", baseline.ID, baseline.CurrentRevision,
		string(payload), actor.Type, actor.ID, timestamp)
	if err != nil {
		return fmt.Errorf("record Project Requirement outbox: %w", err)
	}
	return nil
}

func readProjectRequirementResponseOnConnection(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline, access contract.ProjectRequirementAccessProjection) (contract.ProjectRequirementBaselineResponse, error) {
	revisions, err := readProjectRequirementRevisionsOnConnection(ctx, connection, baseline.ID)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	result := projectRequirementResponse(baseline, revisions, access)
	if baseline.EffectiveRevision != nil {
		found := false
		for _, revision := range revisions {
			if revision.Revision == *baseline.EffectiveRevision {
				effective := projectRequirementContentToContract(revision.Content)
				result.EffectiveContent = &effective
				found = true
				break
			}
		}
		if !found {
			return contract.ProjectRequirementBaselineResponse{}, errors.New("Project Requirement effective revision is missing")
		}
	}
	result.IssueLinks, err = readProjectRequirementIssueLinksOnConnection(ctx, connection, baseline)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	result.OutlineLinks, err = readProjectRequirementOutlineLinksOnConnection(ctx, connection, baseline.ID)
	if err != nil {
		return contract.ProjectRequirementBaselineResponse{}, err
	}
	return result, nil
}

func readProjectRequirementRevisionOnConnection(ctx context.Context, connection *sql.Conn, baselineID string, revision int64) (requirementDomain.Revision, error) {
	row := connection.QueryRowContext(ctx, `SELECT baseline_id,revision,content_json,status,action,change_summary,actor_id,
		submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
		FROM workspace_requirement_revisions WHERE baseline_id=? AND revision=?`, baselineID, revision)
	return scanProjectRequirementRevision(row)
}

func readProjectRequirementCoverageSnapshotOnConnection(ctx context.Context, connection *sql.Conn, baselineID string, revision requirementDomain.Revision) (contract.ProjectRequirementCoverageSnapshot, error) {
	if revision.Revision < 1 || !validProjectRequirementCoverageState(string(revision.Status)) {
		return contract.ProjectRequirementCoverageSnapshot{}, errors.New("invalid Project Requirement coverage revision")
	}
	issuesByKey, err := readProjectRequirementCoverageIssuesOnConnection(ctx, connection, baselineID, revision.Revision)
	if err != nil {
		return contract.ProjectRequirementCoverageSnapshot{}, err
	}
	snapshot := contract.ProjectRequirementCoverageSnapshot{
		Revision: revision.Revision,
		State:    string(revision.Status),
		Items:    make([]contract.ProjectRequirementCoverageItem, 0),
	}
	sections := []struct {
		name  string
		items []requirementDomain.Item
	}{
		{name: "goals", items: revision.Content.Goals},
		{name: "in_scope", items: revision.Content.InScope},
		{name: "constraints", items: revision.Content.Constraints},
		{name: "acceptance_criteria", items: revision.Content.AcceptanceCriteria},
	}
	seenKeys := make(map[string]struct{})
	for _, section := range sections {
		for _, source := range section.items {
			key, itemText := strings.TrimSpace(source.Key), strings.TrimSpace(source.Text)
			if key == "" || itemText == "" {
				return contract.ProjectRequirementCoverageSnapshot{}, errors.New("invalid Project Requirement coverage item")
			}
			if _, duplicate := seenKeys[key]; duplicate {
				return contract.ProjectRequirementCoverageSnapshot{}, errors.New("duplicate Project Requirement coverage item")
			}
			seenKeys[key] = struct{}{}
			issues := issuesByKey[key]
			if issues == nil {
				issues = []contract.ProjectRequirementCoverageIssue{}
			}
			stage := projectRequirementCoverageStage(issues)
			item := contract.ProjectRequirementCoverageItem{
				RequirementKey: key,
				Section:        section.name,
				Text:           itemText,
				Stage:          stage,
				Issues:         issues,
			}
			snapshot.Items = append(snapshot.Items, item)
			snapshot.Total++
			switch stage {
			case "accepted":
				snapshot.Accepted++
				snapshot.Implemented++
				snapshot.Linked++
			case "implemented":
				snapshot.Implemented++
				snapshot.Linked++
			case "linked":
				snapshot.Linked++
			case "unlinked":
			default:
				return contract.ProjectRequirementCoverageSnapshot{}, errors.New("invalid Project Requirement coverage stage")
			}
		}
	}
	snapshot.Unlinked = snapshot.Total - snapshot.Linked
	if snapshot.Accepted > snapshot.Implemented || snapshot.Implemented > snapshot.Linked || snapshot.Linked > snapshot.Total || snapshot.Unlinked < 0 {
		return contract.ProjectRequirementCoverageSnapshot{}, errors.New("invalid Project Requirement coverage counters")
	}
	return snapshot, nil
}

func readProjectRequirementCoverageIssuesOnConnection(ctx context.Context, connection *sql.Conn, baselineID string, revision int64) (map[string][]contract.ProjectRequirementCoverageIssue, error) {
	rows, err := connection.QueryContext(ctx, `WITH active_links AS (
		SELECT workspace_id,project_id,requirement_key,issue_id,linked_revision
		FROM workspace_requirement_issue_links
		WHERE baseline_id=? AND linked_revision<=?
		  AND (unlinked_revision IS NULL OR unlinked_revision>?)
	), latest_acceptance AS (
		SELECT c.workspace_id,c.issue_id,c.result,
		       ROW_NUMBER() OVER (PARTITION BY c.workspace_id,c.issue_id ORDER BY c.created_at DESC,c.id DESC) AS rank
		FROM workspace_issue_acceptance_conclusions c
		JOIN (SELECT DISTINCT workspace_id,issue_id FROM active_links) l
		  ON l.workspace_id=c.workspace_id AND l.issue_id=c.issue_id
	)
		SELECT l.requirement_key,i.id,i.identifier,i.title,i.status,a.result
		FROM active_links l
		JOIN workspace_issues i
		  ON i.workspace_id=l.workspace_id AND i.project_id=l.project_id AND i.id=l.issue_id
		LEFT JOIN latest_acceptance a
		  ON a.workspace_id=l.workspace_id AND a.issue_id=i.id AND a.rank=1
		ORDER BY l.requirement_key,i.identifier COLLATE NOCASE,i.identifier,i.id,l.linked_revision`, baselineID, revision, revision)
	if err != nil {
		return nil, fmt.Errorf("read Project Requirement coverage Issues: %w", err)
	}
	defer rows.Close()
	result := make(map[string][]contract.ProjectRequirementCoverageIssue)
	seen := make(map[string]struct{})
	for rows.Next() {
		var key string
		var issue contract.ProjectRequirementCoverageIssue
		var acceptance sql.NullString
		if err = rows.Scan(&key, &issue.ID, &issue.Identifier, &issue.Title, &issue.Status, &acceptance); err != nil {
			return nil, fmt.Errorf("scan Project Requirement coverage Issue: %w", err)
		}
		key = strings.TrimSpace(key)
		issue.ID, issue.Identifier, issue.Title, issue.Status = strings.TrimSpace(issue.ID), strings.TrimSpace(issue.Identifier), strings.TrimSpace(issue.Title), strings.TrimSpace(issue.Status)
		if key == "" || issue.ID == "" || issue.Identifier == "" || issue.Title == "" || issue.Status == "" {
			return nil, errors.New("invalid Project Requirement coverage Issue")
		}
		identity := key + "\x00" + issue.ID
		if _, duplicate := seen[identity]; duplicate {
			return nil, errors.New("duplicate active Project Requirement coverage Issue link")
		}
		seen[identity] = struct{}{}
		if acceptance.Valid {
			value := strings.TrimSpace(acceptance.String)
			if value != "accepted" && value != "conditional" && value != "rejected" {
				return nil, errors.New("invalid Project Requirement coverage acceptance result")
			}
			issue.AcceptanceResult = &value
		}
		result[key] = append(result[key], issue)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Project Requirement coverage Issues: %w", err)
	}
	return result, nil
}

func projectRequirementCoverageStage(issues []contract.ProjectRequirementCoverageIssue) string {
	if len(issues) == 0 {
		return "unlinked"
	}
	allDone, allAccepted := true, true
	for _, issue := range issues {
		if issue.Status != "done" {
			allDone = false
		}
		if issue.AcceptanceResult == nil || *issue.AcceptanceResult != "accepted" {
			allAccepted = false
		}
	}
	if !allDone {
		return "linked"
	}
	if !allAccepted {
		return "implemented"
	}
	return "accepted"
}

func validProjectRequirementCoverageState(state string) bool {
	switch state {
	case "draft", "in_review", "approved", "frozen", "changed", "retired":
		return true
	default:
		return false
	}
}

type projectRequirementRevisionScanner interface {
	Scan(...any) error
}

func readProjectRequirementRevisionsOnConnection(ctx context.Context, connection *sql.Conn, baselineID string) ([]requirementDomain.Revision, error) {
	rows, err := connection.QueryContext(ctx, `SELECT baseline_id,revision,content_json,status,action,change_summary,actor_id,
		submitted_by,submitted_at,approved_by,approved_at,frozen_by,frozen_at,created_at
		FROM workspace_requirement_revisions WHERE baseline_id=? ORDER BY revision`, baselineID)
	if err != nil {
		return nil, fmt.Errorf("read Project Requirement history: %w", err)
	}
	defer rows.Close()
	revisions := make([]requirementDomain.Revision, 0)
	for rows.Next() {
		revision, scanErr := scanProjectRequirementRevision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		revisions = append(revisions, revision)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Project Requirement history: %w", err)
	}
	return revisions, nil
}

func scanProjectRequirementRevision(scanner projectRequirementRevisionScanner) (requirementDomain.Revision, error) {
	var revision requirementDomain.Revision
	var contentJSON, status, action, createdAt string
	var submittedBy, submittedAt, approvedBy, approvedAt, frozenBy, frozenAt sql.NullString
	if err := scanner.Scan(
		&revision.BaselineID, &revision.Revision, &contentJSON, &status, &action, &revision.ChangeSummary, &revision.ActorID,
		&submittedBy, &submittedAt, &approvedBy, &approvedAt, &frozenBy, &frozenAt, &createdAt,
	); err != nil {
		return requirementDomain.Revision{}, fmt.Errorf("scan Project Requirement history: %w", err)
	}
	if err := json.Unmarshal([]byte(contentJSON), &revision.Content); err != nil {
		return requirementDomain.Revision{}, fmt.Errorf("decode Project Requirement history content: %w", err)
	}
	revision.Status, revision.Action = requirementDomain.Status(status), requirementDomain.Action(action)
	revision.SubmittedBy, revision.ApprovedBy, revision.FrozenBy = nullableStringPointer(submittedBy), nullableStringPointer(approvedBy), nullableStringPointer(frozenBy)
	times, err := parseProjectRequirementOptionalTimes(submittedAt, approvedAt, frozenAt)
	if err != nil {
		return requirementDomain.Revision{}, fmt.Errorf("decode Project Requirement history lifecycle time: %w", err)
	}
	revision.SubmittedAt, revision.ApprovedAt, revision.FrozenAt = times[0], times[1], times[2]
	revision.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return requirementDomain.Revision{}, fmt.Errorf("decode Project Requirement history creation time: %w", err)
	}
	return revision, nil
}

func readProjectRequirementIssueLinksOnConnection(ctx context.Context, connection *sql.Conn, baseline requirementDomain.Baseline) ([]contract.ProjectRequirementIssueLink, error) {
	rows, err := connection.QueryContext(ctx, `SELECT l.requirement_key,l.issue_id,i.identifier,i.title,i.status,
		l.linked_revision,CASE WHEN EXISTS(
			SELECT 1 FROM workspace_requirement_review_projections p
			WHERE p.baseline_id=l.baseline_id AND p.requirement_key=l.requirement_key
			  AND p.issue_id=l.issue_id AND p.source_revision>COALESCE(?,0)
		) THEN 1 ELSE 0 END,l.linked_by,l.linked_at
		FROM workspace_requirement_issue_links l
		JOIN workspace_issues i ON i.workspace_id=l.workspace_id AND i.project_id=l.project_id AND i.id=l.issue_id
		WHERE l.baseline_id=? AND l.unlinked_revision IS NULL
		ORDER BY l.requirement_key,l.issue_id,l.linked_revision`, baseline.EffectiveRevision, baseline.ID)
	if err != nil {
		return nil, fmt.Errorf("read Project Requirement Issue links: %w", err)
	}
	defer rows.Close()
	links := make([]contract.ProjectRequirementIssueLink, 0)
	for rows.Next() {
		var link contract.ProjectRequirementIssueLink
		var reviewRequired int
		if err = rows.Scan(&link.RequirementKey, &link.IssueID, &link.Identifier, &link.Title, &link.Status,
			&link.LinkedRevision, &reviewRequired, &link.LinkedBy, &link.LinkedAt); err != nil {
			return nil, fmt.Errorf("scan Project Requirement Issue link: %w", err)
		}
		if _, err = time.Parse(time.RFC3339Nano, link.LinkedAt); err != nil {
			return nil, fmt.Errorf("decode Project Requirement Issue link time: %w", err)
		}
		link.ReviewRequired = reviewRequired == 1
		links = append(links, link)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Project Requirement Issue links: %w", err)
	}
	return links, nil
}

func readProjectRequirementOutlineLinksOnConnection(ctx context.Context, connection *sql.Conn, baselineID string) ([]contract.ProjectRequirementOutlineLink, error) {
	rows, err := connection.QueryContext(ctx, `SELECT l.requirement_key,l.node_id,n.title,l.linked_revision,l.linked_by,l.linked_at
		FROM workspace_requirement_outline_links l
		JOIN workspace_project_outline_nodes n
		  ON n.workspace_id=l.workspace_id AND n.project_id=l.project_id AND n.id=l.node_id
		WHERE l.baseline_id=? AND l.unlinked_revision IS NULL
		ORDER BY l.requirement_key,l.node_id,l.linked_revision`, baselineID)
	if err != nil {
		return nil, fmt.Errorf("read Project Requirement outline links: %w", err)
	}
	defer rows.Close()
	links := make([]contract.ProjectRequirementOutlineLink, 0)
	for rows.Next() {
		var link contract.ProjectRequirementOutlineLink
		if err = rows.Scan(&link.RequirementKey, &link.NodeID, &link.NodeTitle, &link.LinkedRevision, &link.LinkedBy, &link.LinkedAt); err != nil {
			return nil, fmt.Errorf("scan Project Requirement outline link: %w", err)
		}
		if _, err = time.Parse(time.RFC3339Nano, link.LinkedAt); err != nil {
			return nil, fmt.Errorf("decode Project Requirement outline link time: %w", err)
		}
		links = append(links, link)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Project Requirement outline links: %w", err)
	}
	return links, nil
}

func projectRequirementResponse(baseline requirementDomain.Baseline, revisions []requirementDomain.Revision, access contract.ProjectRequirementAccessProjection) contract.ProjectRequirementBaselineResponse {
	baselineContract := projectRequirementBaselineToContract(baseline)
	current := projectRequirementContentToContract(baseline.Content)
	history := make([]contract.ProjectRequirementRevision, 0, len(revisions))
	for _, revision := range revisions {
		history = append(history, projectRequirementRevisionToContract(revision))
	}
	result := contract.ProjectRequirementBaselineResponse{
		Baseline: &baselineContract, CurrentContent: &current, History: history,
		IssueLinks: []contract.ProjectRequirementIssueLink{}, OutlineLinks: []contract.ProjectRequirementOutlineLink{}, Access: access,
	}
	return result
}

func projectRequirementBaselineToContract(value requirementDomain.Baseline) contract.ProjectRequirementBaseline {
	return contract.ProjectRequirementBaseline{
		ID: value.ID, WorkspaceID: value.WorkspaceID, ProjectID: value.ProjectID, Status: string(value.Status),
		CurrentRevision: value.CurrentRevision, ApprovedRevision: cloneInt64Pointer(value.ApprovedRevision), EffectiveRevision: cloneInt64Pointer(value.EffectiveRevision),
		SubmittedBy: cloneStringPointer(value.SubmittedBy), SubmittedAt: formatTimePointer(value.SubmittedAt),
		ApprovedBy: cloneStringPointer(value.ApprovedBy), ApprovedAt: formatTimePointer(value.ApprovedAt),
		FrozenBy: cloneStringPointer(value.FrozenBy), FrozenAt: formatTimePointer(value.FrozenAt),
		RetiredBy: cloneStringPointer(value.RetiredBy), RetiredAt: formatTimePointer(value.RetiredAt),
		CreatedAt: value.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func projectRequirementRevisionToContract(value requirementDomain.Revision) contract.ProjectRequirementRevision {
	return contract.ProjectRequirementRevision{
		BaselineID: value.BaselineID, Revision: value.Revision, Content: projectRequirementContentToContract(value.Content),
		State: string(value.Status), Action: string(value.Action), ChangeSummary: value.ChangeSummary, ActorID: value.ActorID,
		SubmittedBy: cloneStringPointer(value.SubmittedBy), SubmittedAt: formatTimePointer(value.SubmittedAt),
		ApprovedBy: cloneStringPointer(value.ApprovedBy), ApprovedAt: formatTimePointer(value.ApprovedAt),
		FrozenBy: cloneStringPointer(value.FrozenBy), FrozenAt: formatTimePointer(value.FrozenAt),
		CreatedAt: value.CreatedAt.Format(time.RFC3339Nano),
	}
}

func projectRequirementContentToContract(value requirementDomain.Content) contract.ProjectRequirementContent {
	convert := func(items []requirementDomain.Item) []contract.ProjectRequirementItem {
		result := make([]contract.ProjectRequirementItem, len(items))
		for index, item := range items {
			result[index] = contract.ProjectRequirementItem{Key: item.Key, Text: item.Text}
		}
		return result
	}
	return contract.ProjectRequirementContent{
		ProblemStatement: value.ProblemStatement, Goals: convert(value.Goals), InScope: convert(value.InScope),
		OutOfScope: convert(value.OutOfScope), Constraints: convert(value.Constraints),
		AcceptanceCriteria: convert(value.AcceptanceCriteria), Dependencies: convert(value.Dependencies),
	}
}

func mapProjectRequirementDomainError(err error) error {
	var conflict requirementDomain.RevisionConflictError
	switch {
	case errors.As(err, &conflict):
		return contract.RevisionConflictError{CurrentRevision: conflict.Current}
	case errors.Is(err, requirementDomain.ErrInvalidTransition):
		return application.ErrProjectRequirementTransition
	case errors.Is(err, requirementDomain.ErrIndependentApprovalRequired):
		return application.ErrProjectRequirementSelfApproval
	case errors.Is(err, requirementDomain.ErrMaterialChangeRequired), errors.Is(err, requirementDomain.ErrInvalidBaseline):
		return application.ErrInvalidProjectRequirementRequest
	default:
		return err
	}
}

func projectRequirementRequestID(baselineID, action string, revision int64) string {
	return baselineID + "-" + action + "-" + strconv.FormatInt(revision, 10)
}

func nullableStatusValue(value requirementDomain.Status) any {
	if value == "" {
		return nil
	}
	return string(value)
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	copy := value.Int64
	return &copy
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func parseProjectRequirementOptionalTimes(values ...sql.NullString) ([]*time.Time, error) {
	result := make([]*time.Time, len(values))
	for index, value := range values {
		if !value.Valid {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, value.String)
		if err != nil {
			return nil, err
		}
		result[index] = &parsed
	}
	return result, nil
}

func formatOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatTimePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ application.ProjectRequirementRepository = (*ProjectRequirementRepository)(nil)
