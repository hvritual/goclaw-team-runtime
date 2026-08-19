package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
)

const (
	projectRetrospectiveCreateAction       = "workspace.project.retrospective.create"
	projectRetrospectiveTargetAction       = "workspace.project.retrospective.action_item.target"
	projectRetrospectiveTargetLinkedAction = "workspace.project.retrospective.action_item_linked"
	projectRetrospectiveResourceKind       = "project_retrospective"
)

type ProjectRetrospectiveRepository struct {
	db *sql.DB
}

type projectRetrospectiveCreateReplay struct {
	Version  int                                 `json:"version"`
	ID       string                              `json:"id"`
	Revision int64                               `json:"revision"`
	Access   contract.ProjectRetrospectiveAccess `json:"access"`
}

type projectRetrospectiveTargetReplay struct {
	Version int                                     `json:"version"`
	Link    contract.ProjectRetrospectiveActionLink `json:"link"`
}

func NewProjectRetrospectiveRepository(config Config) (*ProjectRetrospectiveRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &ProjectRetrospectiveRepository{db: config.DB}, nil
}

type projectRetrospectiveAuthority struct {
	membership projectResourceMembershipAuthority
	actorID    string
	leadType   string
	leadID     string
}

func (a projectRetrospectiveAuthority) manager() bool {
	return projectResourceMembershipCanManage(a.membership, a.actorID, a.leadType, a.leadID)
}

func (r *ProjectRetrospectiveRepository) CreateProjectRetrospective(ctx context.Context, command application.CreateProjectRetrospectiveCommand) (result contract.ProjectRetrospective, err error) {
	if strings.TrimSpace(command.WorkspaceID) == "" || strings.TrimSpace(command.ProjectID) == "" || strings.TrimSpace(command.RetrospectiveID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" || len(command.IdempotencyKey) > 200 || len(command.RequestHash) != 64 {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("acquire Project Retrospective connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("configure Project Retrospective lock wait: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("begin Project Retrospective create: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	var storedHash, responseBody string
	replayErr := connection.QueryRowContext(ctx, `SELECT request_hash,response_body FROM workspace_mutation_idempotency
		WHERE workspace_id=? AND action=? AND idempotency_key=?`, command.WorkspaceID, projectRetrospectiveCreateAction, command.IdempotencyKey).Scan(&storedHash, &responseBody)
	if replayErr == nil {
		if storedHash != command.RequestHash {
			return contract.ProjectRetrospective{}, contract.ErrIdempotencyConflict
		}
		var replay projectRetrospectiveCreateReplay
		if err = json.Unmarshal([]byte(responseBody), &replay); err != nil || replay.Version != 1 || replay.ID == "" || replay.Revision != 1 {
			return contract.ProjectRetrospective{}, fmt.Errorf("decode Project Retrospective replay: %w", contract.ErrInvalidGovernanceMutation)
		}
		authority, authorityErr := loadProjectRetrospectiveAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
		if authorityErr != nil {
			return contract.ProjectRetrospective{}, authorityErr
		}
		result, err = readProjectRetrospectiveRevisionOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, replay.ID, replay.Revision, authority)
		if err != nil {
			return contract.ProjectRetrospective{}, err
		}
		result.PublishedRevision = nil
		result.Access = replay.Access
		if _, err = connection.ExecContext(ctx, `ROLLBACK`); err != nil {
			return contract.ProjectRetrospective{}, fmt.Errorf("finish Project Retrospective replay: %w", err)
		}
		committed = true
		return result, nil
	}
	if !errors.Is(replayErr, sql.ErrNoRows) {
		return contract.ProjectRetrospective{}, fmt.Errorf("read Project Retrospective replay: %w", replayErr)
	}

	authority, err := loadProjectRetrospectiveAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	content, err := retrospectiveDomain.NormalizeContent(command.Content)
	if err != nil {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	participants, err := retrospectiveDomain.NormalizeParticipants(command.Participants, authority.membership.MemberID)
	if err != nil {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	if err = validateProjectRetrospectiveParticipants(ctx, connection, command.WorkspaceID, participants, authority.manager()); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if err = validateProjectRetrospectiveAssignees(ctx, connection, command.WorkspaceID, content); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	contentJSON, err := json.Marshal(content)
	if err != nil || len(contentJSON) > 131072 {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_retrospectives(
		workspace_id,project_id,id,status,current_revision,published_revision,created_by,created_at,updated_at
	) VALUES(?,?,?,'draft',1,NULL,?,?,?)`, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.Actor.ID, timestamp, timestamp); err != nil {
		if isProjectResourceConstraint(err) {
			return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveStateConflict
		}
		return contract.ProjectRetrospective{}, fmt.Errorf("insert Project Retrospective: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_retrospective_revisions(
		workspace_id,project_id,retrospective_id,revision,lifecycle_status,action,content_json,actor_id,created_at
	) VALUES(?,?,?,1,'draft','create',?,?,?)`, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, string(contentJSON), command.Actor.ID, timestamp); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("insert Project Retrospective revision: %w", err)
	}
	for _, participant := range participants {
		if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_retrospective_participants(
			workspace_id,project_id,retrospective_id,revision,member_id,role
		) VALUES(?,?,?,1,?,?)`, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, participant.MemberID, participant.Role); err != nil {
			return contract.ProjectRetrospective{}, fmt.Errorf("insert Project Retrospective participant: %w", err)
		}
	}
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_resource_revisions(workspace_id,resource_kind,resource_id,revision,updated_at)
		VALUES(?,?,?,1,?)`, command.WorkspaceID, projectRetrospectiveResourceKind, command.RetrospectiveID, timestamp); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("record Project Retrospective resource revision: %w", err)
	}
	result = projectRetrospectiveResult(command, content, participants, authority, timestamp)
	if err = insertProjectRetrospectiveAuditAndOutbox(ctx, connection, result, command.Actor, command.IdempotencyKey, "create", "retrospective:drafted", command.OccurredAt); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	replay, _ := json.Marshal(projectRetrospectiveCreateReplay{Version: 1, ID: result.ID, Revision: result.CurrentRevision, Access: result.Access})
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency(
		workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at
	) VALUES(?,?,?,?,?,?,1,201,?,?)`, command.WorkspaceID, projectRetrospectiveCreateAction, command.IdempotencyKey, command.RequestHash,
		projectRetrospectiveResourceKind, command.RetrospectiveID, string(replay), timestamp); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("record Project Retrospective replay: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("commit Project Retrospective create: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRetrospectiveRepository) ReadProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string, actor contract.WorkspaceActor) (result contract.ProjectRetrospective, err error) {
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	retrospectiveID = strings.TrimSpace(retrospectiveID)
	if workspaceID == "" || projectID == "" || retrospectiveID == "" {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("acquire Project Retrospective read connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `BEGIN`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("begin Project Retrospective read: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	authority, err := loadProjectRetrospectiveAuthority(ctx, connection, workspaceID, projectID, actor)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	result, err = readProjectRetrospectiveOnConnection(ctx, connection, workspaceID, projectID, retrospectiveID, authority)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("commit Project Retrospective read: %w", err)
	}
	committed = true
	return result, nil
}

func (r *ProjectRetrospectiveRepository) ListProjectRetrospectives(ctx context.Context, query application.ProjectRetrospectiveListQuery) (page application.ProjectRetrospectivePage, err error) {
	query.WorkspaceID = strings.TrimSpace(query.WorkspaceID)
	query.ProjectID = strings.TrimSpace(query.ProjectID)
	if query.WorkspaceID == "" || query.ProjectID == "" || query.Limit < 1 || query.Limit > application.MaxProjectRetrospectiveListLimit {
		return application.ProjectRetrospectivePage{}, contract.ErrInvalidProjectRetrospective
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return application.ProjectRetrospectivePage{}, fmt.Errorf("acquire Project Retrospective list connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `BEGIN`); err != nil {
		return application.ProjectRetrospectivePage{}, fmt.Errorf("begin Project Retrospective list: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	authority, err := loadProjectRetrospectiveAuthority(ctx, connection, query.WorkspaceID, query.ProjectID, query.Actor)
	if err != nil {
		return application.ProjectRetrospectivePage{}, err
	}
	statement := `SELECT id,workspace_id,project_id,status,current_revision,published_revision,created_by,created_at,updated_at
		FROM workspace_project_retrospectives WHERE workspace_id=? AND project_id=?`
	arguments := []any{query.WorkspaceID, query.ProjectID}
	if !query.IncludeArchived {
		statement += ` AND status<>'archived'`
	}
	if query.Cursor != nil {
		if strings.TrimSpace(query.Cursor.UpdatedAt) == "" || strings.TrimSpace(query.Cursor.ID) == "" {
			return application.ProjectRetrospectivePage{}, contract.ErrInvalidProjectRetrospective
		}
		statement += ` AND (updated_at<? OR (updated_at=? AND id<?))`
		arguments = append(arguments, query.Cursor.UpdatedAt, query.Cursor.UpdatedAt, query.Cursor.ID)
	}
	statement += ` ORDER BY updated_at DESC,id DESC LIMIT ?`
	arguments = append(arguments, query.Limit+1)
	rows, err := connection.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return application.ProjectRetrospectivePage{}, fmt.Errorf("list Project Retrospective identities: %w", err)
	}
	values := make([]contract.ProjectRetrospective, 0, query.Limit+1)
	for rows.Next() {
		value, scanErr := scanProjectRetrospectiveHead(rows)
		if scanErr != nil {
			rows.Close()
			return application.ProjectRetrospectivePage{}, scanErr
		}
		values = append(values, value)
	}
	if err = rows.Close(); err != nil {
		return application.ProjectRetrospectivePage{}, err
	}
	if len(values) > query.Limit {
		page.HasMore = true
		values = values[:query.Limit]
	}
	if err = hydrateProjectRetrospectivesOnConnection(ctx, connection, query.WorkspaceID, query.ProjectID, values, authority); err != nil {
		return application.ProjectRetrospectivePage{}, err
	}
	page.Retrospectives = values
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return application.ProjectRetrospectivePage{}, fmt.Errorf("commit Project Retrospective list: %w", err)
	}
	committed = true
	return page, nil
}

func (r *ProjectRetrospectiveRepository) PrepareProjectRetrospectiveTarget(ctx context.Context, command application.PrepareProjectRetrospectiveTargetCommand) (claim application.ProjectRetrospectiveTargetClaim, err error) {
	command.WorkspaceID = strings.TrimSpace(command.WorkspaceID)
	command.ProjectID = strings.TrimSpace(command.ProjectID)
	command.RetrospectiveID = strings.TrimSpace(command.RetrospectiveID)
	command.ActionItemID = strings.TrimSpace(command.ActionItemID)
	command.TargetKind = strings.TrimSpace(command.TargetKind)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	if command.WorkspaceID == "" || command.ProjectID == "" || command.RetrospectiveID == "" || command.ActionItemID == "" ||
		(command.TargetKind != "task" && command.TargetKind != "issue") || command.IdempotencyKey == "" || len(command.IdempotencyKey) > 200 || len(command.RequestHash) != 64 {
		return application.ProjectRetrospectiveTargetClaim{}, contract.ErrInvalidProjectRetrospective
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("acquire Project Retrospective target claim connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("configure Project Retrospective target claim lock wait: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("begin Project Retrospective target claim: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	authority, err := loadProjectRetrospectiveAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, err
	}
	value, err := readProjectRetrospectiveOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, authority)
	if err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, err
	}
	if !projectRetrospectiveTargetAuthorized(value, authority) {
		return application.ProjectRetrospectiveTargetClaim{}, contract.ErrWorkspacePermissionDenied
	}
	actionItem, found := projectRetrospectiveActionItem(value, command.ActionItemID)
	if !found {
		return application.ProjectRetrospectiveTargetClaim{}, contract.ErrProjectRetrospectiveTargetConflict
	}
	var storedHash, responseBody string
	replayErr := connection.QueryRowContext(ctx, `SELECT request_hash,response_body FROM workspace_mutation_idempotency
		WHERE workspace_id=? AND action=? AND idempotency_key=?`, command.WorkspaceID, projectRetrospectiveTargetAction, command.IdempotencyKey).Scan(&storedHash, &responseBody)
	if replayErr == nil {
		if storedHash != command.RequestHash {
			return application.ProjectRetrospectiveTargetClaim{}, contract.ErrIdempotencyConflict
		}
		var replay projectRetrospectiveTargetReplay
		if err = json.Unmarshal([]byte(responseBody), &replay); err != nil || replay.Version != 1 {
			return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("decode Project Retrospective target replay: %w", contract.ErrInvalidGovernanceMutation)
		}
		stored, exists, readErr := readProjectRetrospectiveTargetLinkOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.ActionItemID)
		if readErr != nil {
			return application.ProjectRetrospectiveTargetClaim{}, readErr
		}
		if !exists || stored.RequestHash != command.RequestHash || stored.Link != replay.Link || replay.Link.State != "linked" || replay.Link.TargetKind != command.TargetKind || replay.Link.ActionItemID != command.ActionItemID || replay.Link.TargetID == "" {
			return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("validate Project Retrospective target replay: %w", contract.ErrInvalidGovernanceMutation)
		}
		if _, err = connection.ExecContext(ctx, `ROLLBACK`); err != nil {
			return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("finish Project Retrospective target replay: %w", err)
		}
		committed = true
		return application.ProjectRetrospectiveTargetClaim{
			ActionItem: actionItem, SourceRevision: replay.Link.SourceRevision, TargetKind: replay.Link.TargetKind,
			TargetID: replay.Link.TargetID, ChildIdempotencyKey: projectRetrospectiveTargetChildKey(command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.ActionItemID, replay.Link.SourceRevision, replay.Link.TargetKind),
		}, nil
	}
	if !errors.Is(replayErr, sql.ErrNoRows) {
		return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("read Project Retrospective target replay: %w", replayErr)
	}
	if value.Status != retrospectiveDomain.StatusPublished {
		return application.ProjectRetrospectiveTargetClaim{}, contract.ErrProjectRetrospectiveStateConflict
	}
	stored, exists, err := readProjectRetrospectiveTargetLinkOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.ActionItemID)
	if err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, err
	}
	if exists {
		if stored.RequestHash != command.RequestHash || stored.Link.TargetKind != command.TargetKind {
			return application.ProjectRetrospectiveTargetClaim{}, contract.ErrProjectRetrospectiveTargetConflict
		}
	} else {
		timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
		if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_retrospective_action_links(
			workspace_id,project_id,retrospective_id,action_item_id,source_revision,state,target_kind,target_id,request_hash,claimed_by,claimed_at,linked_by,linked_at
		) VALUES(?,?,?,?,?,'pending',?,NULL,?,?,?,NULL,NULL)`, command.WorkspaceID, command.ProjectID, command.RetrospectiveID,
			command.ActionItemID, value.CurrentRevision, command.TargetKind, command.RequestHash, command.Actor.ID, timestamp); err != nil {
			return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("insert Project Retrospective target claim: %w", err)
		}
		stored = storedProjectRetrospectiveTargetLink{
			Link: contract.ProjectRetrospectiveActionLink{
				RetrospectiveID: command.RetrospectiveID, ActionItemID: command.ActionItemID, SourceRevision: value.CurrentRevision, State: "pending", TargetKind: command.TargetKind,
				CreatedBy: command.Actor.ID, CreatedAt: timestamp,
			},
			RequestHash: command.RequestHash,
		}
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return application.ProjectRetrospectiveTargetClaim{}, fmt.Errorf("commit Project Retrospective target claim: %w", err)
	}
	committed = true
	return application.ProjectRetrospectiveTargetClaim{
		ActionItem: actionItem, SourceRevision: stored.Link.SourceRevision, TargetKind: stored.Link.TargetKind,
		TargetID: stored.Link.TargetID, ChildIdempotencyKey: projectRetrospectiveTargetChildKey(command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.ActionItemID, stored.Link.SourceRevision, stored.Link.TargetKind),
	}, nil
}

func (r *ProjectRetrospectiveRepository) CompleteProjectRetrospectiveTarget(ctx context.Context, command application.CompleteProjectRetrospectiveTargetCommand) (link contract.ProjectRetrospectiveActionLink, err error) {
	command.WorkspaceID = strings.TrimSpace(command.WorkspaceID)
	command.ProjectID = strings.TrimSpace(command.ProjectID)
	command.RetrospectiveID = strings.TrimSpace(command.RetrospectiveID)
	command.ActionItemID = strings.TrimSpace(command.ActionItemID)
	command.TargetKind = strings.TrimSpace(command.TargetKind)
	command.TargetID = strings.TrimSpace(command.TargetID)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	if command.WorkspaceID == "" || command.ProjectID == "" || command.RetrospectiveID == "" || command.ActionItemID == "" || command.SourceRevision < 1 ||
		(command.TargetKind != "task" && command.TargetKind != "issue") || command.TargetID == "" || command.IdempotencyKey == "" || len(command.IdempotencyKey) > 200 || len(command.RequestHash) != 64 {
		return contract.ProjectRetrospectiveActionLink{}, contract.ErrInvalidProjectRetrospective
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("acquire Project Retrospective target completion connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("configure Project Retrospective target completion lock wait: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("begin Project Retrospective target completion: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	authority, err := loadProjectRetrospectiveAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, err
	}
	value, err := readProjectRetrospectiveOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, authority)
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, err
	}
	if !projectRetrospectiveTargetAuthorized(value, authority) {
		return contract.ProjectRetrospectiveActionLink{}, contract.ErrWorkspacePermissionDenied
	}
	if _, found := projectRetrospectiveActionItem(value, command.ActionItemID); !found {
		return contract.ProjectRetrospectiveActionLink{}, contract.ErrProjectRetrospectiveTargetConflict
	}
	var storedHash, responseBody string
	replayErr := connection.QueryRowContext(ctx, `SELECT request_hash,response_body FROM workspace_mutation_idempotency
		WHERE workspace_id=? AND action=? AND idempotency_key=?`, command.WorkspaceID, projectRetrospectiveTargetAction, command.IdempotencyKey).Scan(&storedHash, &responseBody)
	if replayErr == nil {
		if storedHash != command.RequestHash {
			return contract.ProjectRetrospectiveActionLink{}, contract.ErrIdempotencyConflict
		}
		var replay projectRetrospectiveTargetReplay
		if err = json.Unmarshal([]byte(responseBody), &replay); err != nil || replay.Version != 1 {
			return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("decode Project Retrospective target completion replay: %w", contract.ErrInvalidGovernanceMutation)
		}
		stored, exists, readErr := readProjectRetrospectiveTargetLinkOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.ActionItemID)
		if readErr != nil {
			return contract.ProjectRetrospectiveActionLink{}, readErr
		}
		if !exists || stored.RequestHash != command.RequestHash || stored.Link != replay.Link || replay.Link.State != "linked" || replay.Link.TargetID == "" || replay.Link.TargetKind != command.TargetKind || replay.Link.SourceRevision != command.SourceRevision {
			return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("validate Project Retrospective target completion replay: %w", contract.ErrInvalidGovernanceMutation)
		}
		if _, err = connection.ExecContext(ctx, `ROLLBACK`); err != nil {
			return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("finish Project Retrospective target completion replay: %w", err)
		}
		committed = true
		return replay.Link, nil
	}
	if !errors.Is(replayErr, sql.ErrNoRows) {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("read Project Retrospective target completion replay: %w", replayErr)
	}
	stored, exists, err := readProjectRetrospectiveTargetLinkOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, command.ActionItemID)
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, err
	}
	if !exists || stored.RequestHash != command.RequestHash || stored.Link.SourceRevision != command.SourceRevision || stored.Link.TargetKind != command.TargetKind {
		return contract.ProjectRetrospectiveActionLink{}, contract.ErrProjectRetrospectiveTargetConflict
	}
	linkedNow := false
	switch stored.Link.State {
	case "pending":
		if value.Status != retrospectiveDomain.StatusPublished {
			return contract.ProjectRetrospectiveActionLink{}, contract.ErrProjectRetrospectiveStateConflict
		}
		timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
		result, updateErr := connection.ExecContext(ctx, `UPDATE workspace_project_retrospective_action_links
			SET state='linked',target_id=?,linked_by=?,linked_at=?
			WHERE workspace_id=? AND project_id=? AND retrospective_id=? AND action_item_id=? AND state='pending' AND source_revision=? AND target_kind=? AND request_hash=?`,
			command.TargetID, command.Actor.ID, timestamp, command.WorkspaceID, command.ProjectID, command.RetrospectiveID,
			command.ActionItemID, command.SourceRevision, command.TargetKind, command.RequestHash)
		if updateErr != nil {
			return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("link Project Retrospective target: %w", updateErr)
		}
		if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
			return contract.ProjectRetrospectiveActionLink{}, contract.ErrProjectRetrospectiveTargetConflict
		}
		stored.Link.State = "linked"
		stored.Link.TargetID = command.TargetID
		linkedNow = true
	case "linked":
		if stored.Link.TargetID != command.TargetID {
			return contract.ProjectRetrospectiveActionLink{}, contract.ErrProjectRetrospectiveTargetConflict
		}
	default:
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("validate Project Retrospective target state: %w", contract.ErrInvalidGovernanceMutation)
	}
	if linkedNow {
		if err = insertProjectRetrospectiveTargetAuditAndOutbox(ctx, connection, command, stored.Link); err != nil {
			return contract.ProjectRetrospectiveActionLink{}, err
		}
	}
	replay, err := json.Marshal(projectRetrospectiveTargetReplay{Version: 1, Link: stored.Link})
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("encode Project Retrospective target replay: %w", err)
	}
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_mutation_idempotency(
		workspace_id,action,idempotency_key,request_hash,resource_kind,resource_id,resource_revision,response_status,response_body,created_at
	) VALUES(?,?,?,?,?,?,?,?,?,?)`, command.WorkspaceID, projectRetrospectiveTargetAction, command.IdempotencyKey, command.RequestHash,
		projectRetrospectiveResourceKind, command.RetrospectiveID, command.SourceRevision, 201, string(replay), timestamp); err != nil {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("record Project Retrospective target replay: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("commit Project Retrospective target completion: %w", err)
	}
	committed = true
	return stored.Link, nil
}

func (r *ProjectRetrospectiveRepository) MutateProjectRetrospective(ctx context.Context, command application.MutateProjectRetrospectiveCommand) (result contract.ProjectRetrospective, err error) {
	command.WorkspaceID = strings.TrimSpace(command.WorkspaceID)
	command.ProjectID = strings.TrimSpace(command.ProjectID)
	command.RetrospectiveID = strings.TrimSpace(command.RetrospectiveID)
	command.Action = strings.TrimSpace(command.Action)
	command.RequestID = strings.TrimSpace(command.RequestID)
	if command.WorkspaceID == "" || command.ProjectID == "" || command.RetrospectiveID == "" || command.ExpectedRevision < 1 || command.RequestID == "" || len(command.RequestID) > 200 {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("acquire Project Retrospective mutation connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("configure Project Retrospective mutation lock wait: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("begin Project Retrospective mutation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()

	authority, err := loadProjectRetrospectiveAuthority(ctx, connection, command.WorkspaceID, command.ProjectID, command.Actor)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	var currentStatus, createdBy, createdAt string
	var currentRevision int64
	var publishedRevision sql.NullInt64
	err = connection.QueryRowContext(ctx, `SELECT status,current_revision,published_revision,created_by,created_at
		FROM workspace_project_retrospectives WHERE workspace_id=? AND project_id=? AND id=?`,
		command.WorkspaceID, command.ProjectID, command.RetrospectiveID).Scan(&currentStatus, &currentRevision, &publishedRevision, &createdBy, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveNotFound
	}
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("read Project Retrospective mutation head: %w", err)
	}
	if currentRevision != command.ExpectedRevision {
		return contract.ProjectRetrospective{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	currentContent, currentParticipants, err := readProjectRetrospectiveSnapshotOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, currentRevision)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if err = validateProjectRetrospectiveParticipants(ctx, connection, command.WorkspaceID, currentParticipants, true); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if err = validateProjectRetrospectiveAssignees(ctx, connection, command.WorkspaceID, currentContent); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	creator := createdBy == authority.actorID || createdBy == authority.membership.MemberID || createdBy == authority.membership.UserID
	facilitator := projectRetrospectiveHasFacilitator(currentParticipants, authority.membership.MemberID)
	manager := authority.manager()
	canEdit := creator || facilitator || manager
	privileged := facilitator || manager
	nextStatus, transitionErr := retrospectiveDomain.NextStatus(currentStatus, command.Action)
	if transitionErr != nil {
		return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveStateConflict
	}

	nextContent, nextParticipants := currentContent, currentParticipants
	switch command.Action {
	case retrospectiveDomain.ActionSaveDraft:
		if !canEdit {
			return contract.ProjectRetrospective{}, contract.ErrWorkspacePermissionDenied
		}
		if command.Content == nil || command.Participants == nil {
			return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
		}
	case retrospectiveDomain.ActionPublishRevision:
		if !privileged {
			return contract.ProjectRetrospective{}, contract.ErrWorkspacePermissionDenied
		}
		if command.Content == nil || command.Participants == nil {
			return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
		}
	case retrospectiveDomain.ActionPublish, retrospectiveDomain.ActionArchive:
		if !privileged {
			return contract.ProjectRetrospective{}, contract.ErrWorkspacePermissionDenied
		}
		if command.Content != nil || command.Participants != nil {
			return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
		}
	default:
		return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveStateConflict
	}
	if command.Content != nil {
		nextContent, err = retrospectiveDomain.NormalizeContent(*command.Content)
		if err != nil {
			return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
		}
		creatorMemberID, creatorErr := projectRetrospectiveCreatorMemberID(ctx, connection, command.WorkspaceID, createdBy)
		if creatorErr != nil {
			return contract.ProjectRetrospective{}, creatorErr
		}
		nextParticipants, err = retrospectiveDomain.NormalizeParticipants(*command.Participants, creatorMemberID)
		if err != nil {
			return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
		}
		if err = validateProjectRetrospectiveParticipants(ctx, connection, command.WorkspaceID, nextParticipants, true); err != nil {
			return contract.ProjectRetrospective{}, err
		}
		if !manager && !projectRetrospectiveFacilitatorsEqual(currentParticipants, nextParticipants) {
			return contract.ProjectRetrospective{}, contract.ErrWorkspacePermissionDenied
		}
		if err = validateProjectRetrospectiveAssignees(ctx, connection, command.WorkspaceID, nextContent); err != nil {
			return contract.ProjectRetrospective{}, err
		}
		linked, linkedErr := readProjectRetrospectiveLinkedActionItems(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID)
		if linkedErr != nil {
			return contract.ProjectRetrospective{}, linkedErr
		}
		if err = retrospectiveDomain.ValidateLinkedActionItemsUnchanged(currentContent, nextContent, linked); err != nil {
			return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveStateConflict
		}
	}
	contentJSON, err := json.Marshal(nextContent)
	if err != nil || len(contentJSON) > 131072 {
		return contract.ProjectRetrospective{}, contract.ErrInvalidProjectRetrospective
	}
	nextRevision := currentRevision + 1
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_retrospective_revisions(
		workspace_id,project_id,retrospective_id,revision,lifecycle_status,action,content_json,actor_id,created_at
	) VALUES(?,?,?,?,?,?,?,?,?)`, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, nextRevision, nextStatus, command.Action, string(contentJSON), command.Actor.ID, timestamp); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("insert Project Retrospective mutation revision: %w", err)
	}
	for _, participant := range nextParticipants {
		if _, err = connection.ExecContext(ctx, `INSERT INTO workspace_project_retrospective_participants(
			workspace_id,project_id,retrospective_id,revision,member_id,role
		) VALUES(?,?,?,?,?,?)`, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, nextRevision, participant.MemberID, participant.Role); err != nil {
			return contract.ProjectRetrospective{}, fmt.Errorf("insert Project Retrospective mutation participant: %w", err)
		}
	}
	if command.Action == retrospectiveDomain.ActionPublish || command.Action == retrospectiveDomain.ActionPublishRevision {
		publishedRevision = sql.NullInt64{Int64: nextRevision, Valid: true}
	}
	headResult, err := connection.ExecContext(ctx, `UPDATE workspace_project_retrospectives
		SET status=?,current_revision=?,published_revision=?,updated_at=?
		WHERE workspace_id=? AND project_id=? AND id=? AND current_revision=?`, nextStatus, nextRevision, nullableInt64Value(publishedRevision), timestamp,
		command.WorkspaceID, command.ProjectID, command.RetrospectiveID, currentRevision)
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("update Project Retrospective mutation head: %w", err)
	}
	if affected, rowsErr := headResult.RowsAffected(); rowsErr != nil || affected != 1 {
		return contract.ProjectRetrospective{}, contract.RevisionConflictError{CurrentRevision: currentRevision}
	}
	revisionResult, err := connection.ExecContext(ctx, `UPDATE workspace_resource_revisions SET revision=?,updated_at=?
		WHERE workspace_id=? AND resource_kind=? AND resource_id=? AND revision=?`, nextRevision, timestamp,
		command.WorkspaceID, projectRetrospectiveResourceKind, command.RetrospectiveID, currentRevision)
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("update Project Retrospective resource revision: %w", err)
	}
	if affected, rowsErr := revisionResult.RowsAffected(); rowsErr != nil || affected != 1 {
		return contract.ProjectRetrospective{}, contract.ErrInvalidGovernanceMutation
	}
	result, err = readProjectRetrospectiveOnConnection(ctx, connection, command.WorkspaceID, command.ProjectID, command.RetrospectiveID, authority)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	eventType := map[string]string{
		retrospectiveDomain.ActionSaveDraft:       "retrospective:draft_saved",
		retrospectiveDomain.ActionPublish:         "retrospective:published",
		retrospectiveDomain.ActionPublishRevision: "retrospective:published",
		retrospectiveDomain.ActionArchive:         "retrospective:archived",
	}[command.Action]
	if err = insertProjectRetrospectiveAuditAndOutbox(ctx, connection, result, command.Actor, command.RequestID, command.Action, eventType, command.OccurredAt); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("commit Project Retrospective mutation: %w", err)
	}
	committed = true
	return result, nil
}

func loadProjectRetrospectiveAuthority(ctx context.Context, connection *sql.Conn, workspaceID, projectID string, actor contract.WorkspaceActor) (projectRetrospectiveAuthority, error) {
	membership, err := projectResourceMembershipOnConnection(ctx, connection, strings.TrimSpace(workspaceID), actor)
	if err != nil {
		return projectRetrospectiveAuthority{}, err
	}
	var leadType, leadID sql.NullString
	if err = connection.QueryRowContext(ctx, `SELECT lead_type,lead_id FROM workspace_projects WHERE workspace_id=? AND id=?`, workspaceID, projectID).Scan(&leadType, &leadID); errors.Is(err, sql.ErrNoRows) {
		return projectRetrospectiveAuthority{}, application.ErrProjectSurfaceNotFound
	} else if err != nil {
		return projectRetrospectiveAuthority{}, fmt.Errorf("read Project Retrospective project: %w", err)
	}
	return projectRetrospectiveAuthority{membership: membership, actorID: strings.TrimSpace(actor.ID), leadType: nullableText(leadType), leadID: nullableText(leadID)}, nil
}

func validateProjectRetrospectiveParticipants(ctx context.Context, connection *sql.Conn, workspaceID string, participants []retrospectiveDomain.Participant, canAssignFacilitator bool) error {
	for _, participant := range participants {
		var found int
		if err := connection.QueryRowContext(ctx, `SELECT 1 FROM auth_members WHERE workspace_id=? AND id=?`, workspaceID, participant.MemberID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
			return contract.ErrActorOutsideWorkspace
		} else if err != nil {
			return err
		}
		if participant.Role == retrospectiveDomain.RoleFacilitator && !canAssignFacilitator {
			return contract.ErrWorkspacePermissionDenied
		}
	}
	return nil
}

func validateProjectRetrospectiveAssignees(ctx context.Context, connection *sql.Conn, workspaceID string, content retrospectiveDomain.Content) error {
	for _, item := range content.ActionItems {
		if item.AssigneeID == "" {
			continue
		}
		var found int
		if err := connection.QueryRowContext(ctx, `SELECT 1 FROM auth_members WHERE workspace_id=? AND id=?`, workspaceID, item.AssigneeID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
			return contract.ErrActorOutsideWorkspace
		} else if err != nil {
			return err
		}
	}
	return nil
}

func projectRetrospectiveResult(command application.CreateProjectRetrospectiveCommand, content retrospectiveDomain.Content, participants []retrospectiveDomain.Participant, authority projectRetrospectiveAuthority, timestamp string) contract.ProjectRetrospective {
	revision := contract.ProjectRetrospectiveRevision{
		Revision: 1, Status: retrospectiveDomain.StatusDraft, Action: "create",
		Content: projectRetrospectiveContentToContract(content), Participants: projectRetrospectiveParticipantsToContract(participants),
		ActorID: command.Actor.ID, CreatedAt: timestamp,
	}
	return contract.ProjectRetrospective{
		ID: command.RetrospectiveID, WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID,
		Status: retrospectiveDomain.StatusDraft, CurrentRevision: 1, CreatedBy: command.Actor.ID,
		CreatedAt: timestamp, UpdatedAt: timestamp, Current: &revision,
		History: []contract.ProjectRetrospectiveRevision{revision}, ActionLinks: []contract.ProjectRetrospectiveActionLink{},
		Access: contract.ProjectRetrospectiveAccess{CanEdit: true, CanPublish: authority.manager(), CanArchive: authority.manager()},
	}
}

func projectRetrospectiveContentToContract(content retrospectiveDomain.Content) contract.ProjectRetrospectiveContent {
	actions := make([]contract.ProjectRetrospectiveActionItem, len(content.ActionItems))
	for index, item := range content.ActionItems {
		actions[index] = contract.ProjectRetrospectiveActionItem{ID: item.ID, Title: item.Title, Description: item.Description, AssigneeID: item.AssigneeID, DueDate: item.DueDate}
	}
	return contract.ProjectRetrospectiveContent{Summary: content.Summary, Successes: content.Successes, Problems: content.Problems, Lessons: content.Lessons, ActionItems: actions}
}

func projectRetrospectiveParticipantsToContract(participants []retrospectiveDomain.Participant) []contract.ProjectRetrospectiveParticipant {
	result := make([]contract.ProjectRetrospectiveParticipant, len(participants))
	for index, participant := range participants {
		result[index] = contract.ProjectRetrospectiveParticipant{MemberID: participant.MemberID, Role: participant.Role}
	}
	return result
}

func projectRetrospectiveCreatorMemberID(ctx context.Context, connection *sql.Conn, workspaceID, createdBy string) (string, error) {
	var memberID string
	err := connection.QueryRowContext(ctx, `SELECT id FROM auth_members
		WHERE workspace_id=? AND (user_id=? OR id=?)
		ORDER BY CASE WHEN user_id=? THEN 0 ELSE 1 END LIMIT 1`, workspaceID, createdBy, createdBy, createdBy).Scan(&memberID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", contract.ErrActorOutsideWorkspace
	}
	if err != nil {
		return "", fmt.Errorf("read Project Retrospective creator membership: %w", err)
	}
	return memberID, nil
}

func projectRetrospectiveHasFacilitator(participants []retrospectiveDomain.Participant, memberID string) bool {
	for _, participant := range participants {
		if participant.MemberID == memberID && participant.Role == retrospectiveDomain.RoleFacilitator {
			return true
		}
	}
	return false
}

func projectRetrospectiveFacilitatorsEqual(left, right []retrospectiveDomain.Participant) bool {
	leftSet := make(map[string]struct{})
	rightSet := make(map[string]struct{})
	for _, participant := range left {
		if participant.Role == retrospectiveDomain.RoleFacilitator {
			leftSet[participant.MemberID] = struct{}{}
		}
	}
	for _, participant := range right {
		if participant.Role == retrospectiveDomain.RoleFacilitator {
			rightSet[participant.MemberID] = struct{}{}
		}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for memberID := range leftSet {
		if _, found := rightSet[memberID]; !found {
			return false
		}
	}
	return true
}

func readProjectRetrospectiveSnapshotOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID string, revision int64) (retrospectiveDomain.Content, []retrospectiveDomain.Participant, error) {
	var contentJSON string
	err := connection.QueryRowContext(ctx, `SELECT content_json FROM workspace_project_retrospective_revisions
		WHERE workspace_id=? AND project_id=? AND retrospective_id=? AND revision=?`, workspaceID, projectID, retrospectiveID, revision).Scan(&contentJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return retrospectiveDomain.Content{}, nil, contract.ErrProjectRetrospectiveNotFound
	}
	if err != nil {
		return retrospectiveDomain.Content{}, nil, fmt.Errorf("read Project Retrospective snapshot: %w", err)
	}
	var content retrospectiveDomain.Content
	if err = json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return retrospectiveDomain.Content{}, nil, fmt.Errorf("decode Project Retrospective snapshot: %w", contract.ErrInvalidProjectRetrospective)
	}
	content, err = retrospectiveDomain.NormalizeContent(content)
	if err != nil {
		return retrospectiveDomain.Content{}, nil, fmt.Errorf("validate Project Retrospective snapshot: %w", contract.ErrInvalidProjectRetrospective)
	}
	participants, err := readProjectRetrospectiveParticipantsDomain(ctx, connection, workspaceID, projectID, retrospectiveID, revision)
	if err != nil {
		return retrospectiveDomain.Content{}, nil, err
	}
	return content, participants, nil
}

func readProjectRetrospectiveParticipantsDomain(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID string, revision int64) ([]retrospectiveDomain.Participant, error) {
	rows, err := connection.QueryContext(ctx, `SELECT member_id,role FROM workspace_project_retrospective_participants
		WHERE workspace_id=? AND project_id=? AND retrospective_id=? AND revision=? ORDER BY member_id`, workspaceID, projectID, retrospectiveID, revision)
	if err != nil {
		return nil, fmt.Errorf("read Project Retrospective snapshot participants: %w", err)
	}
	defer rows.Close()
	participants := make([]retrospectiveDomain.Participant, 0)
	seen := make(map[string]struct{})
	for rows.Next() {
		var participant retrospectiveDomain.Participant
		if err = rows.Scan(&participant.MemberID, &participant.Role); err != nil {
			return nil, err
		}
		if participant.MemberID == "" || (participant.Role != retrospectiveDomain.RoleParticipant && participant.Role != retrospectiveDomain.RoleFacilitator) {
			return nil, fmt.Errorf("validate Project Retrospective snapshot participants: %w", contract.ErrInvalidProjectRetrospective)
		}
		if _, duplicate := seen[participant.MemberID]; duplicate {
			return nil, fmt.Errorf("validate Project Retrospective snapshot participants: %w", contract.ErrInvalidProjectRetrospective)
		}
		seen[participant.MemberID] = struct{}{}
		participants = append(participants, participant)
		if len(participants) > 100 {
			return nil, fmt.Errorf("validate Project Retrospective snapshot participants: %w", contract.ErrInvalidProjectRetrospective)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(participants) == 0 {
		return nil, fmt.Errorf("validate Project Retrospective snapshot participants: %w", contract.ErrInvalidProjectRetrospective)
	}
	return participants, nil
}

func readProjectRetrospectiveLinkedActionItems(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID string) (map[string]struct{}, error) {
	rows, err := connection.QueryContext(ctx, `SELECT action_item_id FROM workspace_project_retrospective_action_links
		WHERE workspace_id=? AND project_id=? AND retrospective_id=?`, workspaceID, projectID, retrospectiveID)
	if err != nil {
		return nil, fmt.Errorf("read Project Retrospective action links: %w", err)
	}
	defer rows.Close()
	result := make(map[string]struct{})
	for rows.Next() {
		var actionItemID string
		if err = rows.Scan(&actionItemID); err != nil {
			return nil, err
		}
		result[actionItemID] = struct{}{}
	}
	return result, rows.Err()
}

type storedProjectRetrospectiveTargetLink struct {
	Link        contract.ProjectRetrospectiveActionLink
	RequestHash string
}

func readProjectRetrospectiveTargetLinkOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID, actionItemID string) (storedProjectRetrospectiveTargetLink, bool, error) {
	var stored storedProjectRetrospectiveTargetLink
	var targetID sql.NullString
	err := connection.QueryRowContext(ctx, `SELECT action_item_id,source_revision,state,target_kind,target_id,request_hash,claimed_by,claimed_at
		FROM workspace_project_retrospective_action_links
		WHERE workspace_id=? AND project_id=? AND retrospective_id=? AND action_item_id=?`, workspaceID, projectID, retrospectiveID, actionItemID).Scan(
		&stored.Link.ActionItemID, &stored.Link.SourceRevision, &stored.Link.State, &stored.Link.TargetKind, &targetID,
		&stored.RequestHash, &stored.Link.CreatedBy, &stored.Link.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return storedProjectRetrospectiveTargetLink{}, false, nil
	}
	if err != nil {
		return storedProjectRetrospectiveTargetLink{}, false, fmt.Errorf("read Project Retrospective target link: %w", err)
	}
	stored.Link.RetrospectiveID = retrospectiveID
	stored.Link.TargetID = nullableText(targetID)
	if stored.Link.ActionItemID == "" || stored.Link.SourceRevision < 1 || (stored.Link.TargetKind != "task" && stored.Link.TargetKind != "issue") || len(stored.RequestHash) != 64 ||
		(stored.Link.State == "pending" && stored.Link.TargetID != "") || (stored.Link.State == "linked" && stored.Link.TargetID == "") || (stored.Link.State != "pending" && stored.Link.State != "linked") {
		return storedProjectRetrospectiveTargetLink{}, false, fmt.Errorf("validate Project Retrospective target link: %w", contract.ErrInvalidGovernanceMutation)
	}
	return stored, true, nil
}

func projectRetrospectiveTargetAuthorized(value contract.ProjectRetrospective, authority projectRetrospectiveAuthority) bool {
	if value.Current == nil {
		return false
	}
	participants := make([]retrospectiveDomain.Participant, len(value.Current.Participants))
	for index, participant := range value.Current.Participants {
		participants[index] = retrospectiveDomain.Participant{MemberID: participant.MemberID, Role: participant.Role}
	}
	return authority.manager() || projectRetrospectiveHasFacilitator(participants, authority.membership.MemberID)
}

func projectRetrospectiveActionItem(value contract.ProjectRetrospective, actionItemID string) (contract.ProjectRetrospectiveActionItem, bool) {
	if value.Current == nil {
		return contract.ProjectRetrospectiveActionItem{}, false
	}
	for _, item := range value.Current.Content.ActionItems {
		if item.ID == actionItemID {
			return item, true
		}
	}
	return contract.ProjectRetrospectiveActionItem{}, false
}

func projectRetrospectiveTargetChildKey(workspaceID, projectID, retrospectiveID, actionItemID string, sourceRevision int64, targetKind string) string {
	payload := strings.Join([]string{
		"project-retrospective-target-child-v1", workspaceID, projectID, retrospectiveID, actionItemID,
		strconv.FormatInt(sourceRevision, 10), targetKind,
	}, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "retro-target-" + hex.EncodeToString(sum[:])
}

func insertProjectRetrospectiveTargetAuditAndOutbox(ctx context.Context, connection *sql.Conn, command application.CompleteProjectRetrospectiveTargetCommand, link contract.ProjectRetrospectiveActionLink) error {
	metadata, _ := json.Marshal(map[string]any{
		"version": "project-retrospective-target-v1", "action_item_id": link.ActionItemID,
		"source_revision": link.SourceRevision, "target_kind": link.TargetKind, "target_id": link.TargetID,
	})
	payload, _ := json.Marshal(map[string]any{
		"version": "project-retrospective-target-v1", "retrospective_id": command.RetrospectiveID,
		"project_id": command.ProjectID, "action_item_id": link.ActionItemID, "source_revision": link.SourceRevision,
		"target_kind": link.TargetKind, "target_id": link.TargetID,
	})
	timestamp := command.OccurredAt.UTC().Format(time.RFC3339Nano)
	auditID := command.RetrospectiveID + "-" + command.ActionItemID + "-target-linked-audit"
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_audit_entries(
		workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, command.WorkspaceID, timestamp, auditID, command.Actor.Type, command.Actor.ID,
		projectRetrospectiveTargetLinkedAction, projectRetrospectiveResourceKind, command.RetrospectiveID, link.SourceRevision,
		command.IdempotencyKey, string(metadata)); err != nil {
		return fmt.Errorf("record Project Retrospective target audit: %w", err)
	}
	eventID := command.RetrospectiveID + "-" + command.ActionItemID + "-target-linked-event"
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_outbox_events(
		state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,payload_json,actor_type,actor_id,attempt_count,created_at
	) VALUES('ready',?,?,?,?,?,?,?,?,?,?,0,?)`, timestamp, command.WorkspaceID, eventID, "retrospective:action_item_linked",
		projectRetrospectiveResourceKind, command.RetrospectiveID, link.SourceRevision, string(payload), command.Actor.Type, command.Actor.ID, timestamp); err != nil {
		return fmt.Errorf("record Project Retrospective target outbox: %w", err)
	}
	return nil
}

func readProjectRetrospectiveOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID string, authority projectRetrospectiveAuthority) (contract.ProjectRetrospective, error) {
	result, err := scanProjectRetrospectiveHead(connection.QueryRowContext(ctx, `SELECT id,workspace_id,project_id,status,current_revision,published_revision,created_by,created_at,updated_at
		FROM workspace_project_retrospectives WHERE workspace_id=? AND project_id=? AND id=?`, workspaceID, projectID, retrospectiveID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveNotFound
	} else if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("read Project Retrospective head: %w", err)
	}
	values := []contract.ProjectRetrospective{result}
	if err = hydrateProjectRetrospectivesOnConnection(ctx, connection, workspaceID, projectID, values, authority); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	return values[0], nil
}

type projectRetrospectiveScanner interface {
	Scan(...any) error
}

func scanProjectRetrospectiveHead(scanner projectRetrospectiveScanner) (contract.ProjectRetrospective, error) {
	var result contract.ProjectRetrospective
	var published sql.NullInt64
	if err := scanner.Scan(
		&result.ID, &result.WorkspaceID, &result.ProjectID, &result.Status, &result.CurrentRevision, &published,
		&result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if published.Valid {
		value := published.Int64
		result.PublishedRevision = &value
	}
	return result, nil
}

func hydrateProjectRetrospectivesOnConnection(
	ctx context.Context,
	connection *sql.Conn,
	workspaceID, projectID string,
	values []contract.ProjectRetrospective,
	authority projectRetrospectiveAuthority,
) error {
	if len(values) == 0 {
		return nil
	}
	indexes := make(map[string]int, len(values))
	arguments := make([]any, 0, len(values)+2)
	arguments = append(arguments, workspaceID, projectID)
	for index := range values {
		value := &values[index]
		if value.ID == "" || value.WorkspaceID != workspaceID || value.ProjectID != projectID || value.CurrentRevision < 1 {
			return fmt.Errorf("validate Project Retrospective head: %w", contract.ErrInvalidProjectRetrospective)
		}
		if value.PublishedRevision != nil && (*value.PublishedRevision < 1 || *value.PublishedRevision > value.CurrentRevision) {
			return fmt.Errorf("validate Project Retrospective published revision: %w", contract.ErrInvalidProjectRetrospective)
		}
		if _, duplicate := indexes[value.ID]; duplicate {
			return fmt.Errorf("validate Project Retrospective head identity: %w", contract.ErrInvalidProjectRetrospective)
		}
		indexes[value.ID] = index
		arguments = append(arguments, value.ID)
		value.Current = nil
		value.History = make([]contract.ProjectRetrospectiveRevision, 0, value.CurrentRevision)
		value.ActionLinks = []contract.ProjectRetrospectiveActionLink{}
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(values)), ",")
	revisionRows, err := connection.QueryContext(ctx, `SELECT retrospective_id,revision,lifecycle_status,action,content_json,actor_id,created_at
		FROM workspace_project_retrospective_revisions WHERE workspace_id=? AND project_id=? AND retrospective_id IN (`+placeholders+`)
		ORDER BY retrospective_id,revision`, arguments...)
	if err != nil {
		return fmt.Errorf("read Project Retrospective history: %w", err)
	}
	for revisionRows.Next() {
		var retrospectiveID string
		var revision contract.ProjectRetrospectiveRevision
		var lifecycleStatus, contentJSON string
		if err = revisionRows.Scan(&retrospectiveID, &revision.Revision, &lifecycleStatus, &revision.Action, &contentJSON, &revision.ActorID, &revision.CreatedAt); err != nil {
			revisionRows.Close()
			return err
		}
		index, found := indexes[retrospectiveID]
		if !found {
			revisionRows.Close()
			return fmt.Errorf("validate Project Retrospective history ownership: %w", contract.ErrInvalidProjectRetrospective)
		}
		var content retrospectiveDomain.Content
		if err = json.Unmarshal([]byte(contentJSON), &content); err != nil {
			revisionRows.Close()
			return fmt.Errorf("decode Project Retrospective history: %w", contract.ErrInvalidProjectRetrospective)
		}
		content, err = retrospectiveDomain.NormalizeContent(content)
		if err != nil {
			revisionRows.Close()
			return fmt.Errorf("validate Project Retrospective history: %w", contract.ErrInvalidProjectRetrospective)
		}
		revision.Status = lifecycleStatus
		if lifecycleStatus == retrospectiveDomain.StatusPublished && values[index].PublishedRevision != nil && revision.Revision < *values[index].PublishedRevision {
			revision.Status = "superseded"
		}
		revision.Content = projectRetrospectiveContentToContract(content)
		values[index].History = append(values[index].History, revision)
	}
	if err = revisionRows.Close(); err != nil {
		return err
	}
	participantRows, err := connection.QueryContext(ctx, `SELECT retrospective_id,revision,member_id,role FROM workspace_project_retrospective_participants
		WHERE workspace_id=? AND project_id=? AND retrospective_id IN (`+placeholders+`) ORDER BY retrospective_id,revision,member_id`, arguments...)
	if err != nil {
		return fmt.Errorf("read Project Retrospective history participants: %w", err)
	}
	participantCounts := make(map[string]int)
	participantMembers := make(map[string]struct{})
	for participantRows.Next() {
		var retrospectiveID string
		var revision int64
		var participant contract.ProjectRetrospectiveParticipant
		if err = participantRows.Scan(&retrospectiveID, &revision, &participant.MemberID, &participant.Role); err != nil {
			participantRows.Close()
			return err
		}
		index, found := indexes[retrospectiveID]
		if !found || revision < 1 || revision > values[index].CurrentRevision || participant.MemberID == "" || (participant.Role != retrospectiveDomain.RoleParticipant && participant.Role != retrospectiveDomain.RoleFacilitator) {
			participantRows.Close()
			return fmt.Errorf("validate Project Retrospective history participants: %w", contract.ErrInvalidProjectRetrospective)
		}
		historyIndex := int(revision - 1)
		if historyIndex >= len(values[index].History) || values[index].History[historyIndex].Revision != revision {
			participantRows.Close()
			return fmt.Errorf("validate Project Retrospective history sequence: %w", contract.ErrInvalidProjectRetrospective)
		}
		countKey := retrospectiveID + "\x00" + strconv.FormatInt(revision, 10)
		memberKey := countKey + "\x00" + participant.MemberID
		if _, duplicate := participantMembers[memberKey]; duplicate {
			participantRows.Close()
			return fmt.Errorf("validate Project Retrospective history participant identity: %w", contract.ErrInvalidProjectRetrospective)
		}
		participantMembers[memberKey] = struct{}{}
		values[index].History[historyIndex].Participants = append(values[index].History[historyIndex].Participants, participant)
		participantCounts[countKey]++
		if participantCounts[countKey] > 100 {
			participantRows.Close()
			return fmt.Errorf("validate Project Retrospective history participants: %w", contract.ErrInvalidProjectRetrospective)
		}
	}
	if err = participantRows.Close(); err != nil {
		return err
	}
	linkRows, err := connection.QueryContext(ctx, `SELECT retrospective_id,action_item_id,source_revision,state,target_kind,target_id,claimed_by,claimed_at
		FROM workspace_project_retrospective_action_links WHERE workspace_id=? AND project_id=? AND retrospective_id IN (`+placeholders+`) ORDER BY retrospective_id,action_item_id`, arguments...)
	if err != nil {
		return fmt.Errorf("read Project Retrospective links: %w", err)
	}
	for linkRows.Next() {
		var retrospectiveID string
		var link contract.ProjectRetrospectiveActionLink
		var targetID sql.NullString
		if err = linkRows.Scan(&retrospectiveID, &link.ActionItemID, &link.SourceRevision, &link.State, &link.TargetKind, &targetID, &link.CreatedBy, &link.CreatedAt); err != nil {
			linkRows.Close()
			return err
		}
		index, found := indexes[retrospectiveID]
		if !found || link.ActionItemID == "" || link.SourceRevision < 1 || link.SourceRevision > values[index].CurrentRevision {
			linkRows.Close()
			return fmt.Errorf("validate Project Retrospective link ownership: %w", contract.ErrInvalidProjectRetrospective)
		}
		link.RetrospectiveID = retrospectiveID
		link.TargetID = nullableText(targetID)
		values[index].ActionLinks = append(values[index].ActionLinks, link)
	}
	if err = linkRows.Close(); err != nil {
		return err
	}
	for valueIndex := range values {
		value := &values[valueIndex]
		if len(value.History) != int(value.CurrentRevision) {
			return fmt.Errorf("validate Project Retrospective history: %w", contract.ErrInvalidProjectRetrospective)
		}
		for historyIndex := range value.History {
			revision := &value.History[historyIndex]
			if revision.Revision != int64(historyIndex+1) || len(revision.Participants) == 0 {
				return fmt.Errorf("validate Project Retrospective history sequence: %w", contract.ErrInvalidProjectRetrospective)
			}
			if revision.Revision == value.CurrentRevision {
				if revision.Status != value.Status {
					return fmt.Errorf("validate Project Retrospective current status: %w", contract.ErrInvalidProjectRetrospective)
				}
				value.Current = revision
			}
		}
		if value.Current == nil {
			return fmt.Errorf("validate Project Retrospective current revision: %w", contract.ErrInvalidProjectRetrospective)
		}
		currentParticipants := make([]retrospectiveDomain.Participant, len(value.Current.Participants))
		for index, participant := range value.Current.Participants {
			currentParticipants[index] = retrospectiveDomain.Participant{MemberID: participant.MemberID, Role: participant.Role}
		}
		creator := value.CreatedBy == authority.actorID || value.CreatedBy == authority.membership.MemberID || value.CreatedBy == authority.membership.UserID
		facilitator := projectRetrospectiveHasFacilitator(currentParticipants, authority.membership.MemberID)
		privileged := authority.manager() || facilitator
		value.Access = contract.ProjectRetrospectiveAccess{
			CanEdit:    value.Status == retrospectiveDomain.StatusDraft && (creator || privileged),
			CanPublish: value.Status != retrospectiveDomain.StatusArchived && privileged,
			CanArchive: value.Status != retrospectiveDomain.StatusArchived && privileged,
		}
	}
	return nil
}

func nullableInt64Value(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func readProjectRetrospectiveRevisionOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID string, revision int64, authority projectRetrospectiveAuthority) (contract.ProjectRetrospective, error) {
	var result contract.ProjectRetrospective
	var published sql.NullInt64
	if err := connection.QueryRowContext(ctx, `SELECT id,workspace_id,project_id,status,current_revision,published_revision,created_by,created_at,updated_at
		FROM workspace_project_retrospectives WHERE workspace_id=? AND project_id=? AND id=?`, workspaceID, projectID, retrospectiveID).Scan(
		&result.ID, &result.WorkspaceID, &result.ProjectID, &result.Status, &result.CurrentRevision, &published, &result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	); errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveNotFound
	} else if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if published.Valid {
		value := published.Int64
		result.PublishedRevision = &value
	}
	var lifecycleStatus, action, contentJSON, actorID, createdAt string
	if err := connection.QueryRowContext(ctx, `SELECT lifecycle_status,action,content_json,actor_id,created_at
		FROM workspace_project_retrospective_revisions WHERE workspace_id=? AND project_id=? AND retrospective_id=? AND revision=?`, workspaceID, projectID, retrospectiveID, revision).Scan(
		&lifecycleStatus, &action, &contentJSON, &actorID, &createdAt,
	); errors.Is(err, sql.ErrNoRows) {
		return contract.ProjectRetrospective{}, contract.ErrProjectRetrospectiveNotFound
	} else if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	var content retrospectiveDomain.Content
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("decode Project Retrospective content: %w", contract.ErrInvalidProjectRetrospective)
	}
	content, err := retrospectiveDomain.NormalizeContent(content)
	if err != nil {
		return contract.ProjectRetrospective{}, fmt.Errorf("validate Project Retrospective content: %w", contract.ErrInvalidProjectRetrospective)
	}
	participants, err := readProjectRetrospectiveParticipants(ctx, connection, workspaceID, projectID, retrospectiveID, revision)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	current := contract.ProjectRetrospectiveRevision{Revision: revision, Status: lifecycleStatus, Action: action, Content: projectRetrospectiveContentToContract(content), Participants: participants, ActorID: actorID, CreatedAt: createdAt}
	result.Status, result.CurrentRevision, result.UpdatedAt = lifecycleStatus, revision, createdAt
	result.Current = &current
	result.History = []contract.ProjectRetrospectiveRevision{current}
	result.ActionLinks = []contract.ProjectRetrospectiveActionLink{}
	canEdit := result.CreatedBy == authority.actorID || result.CreatedBy == authority.membership.MemberID || result.CreatedBy == authority.membership.UserID || authority.manager()
	result.Access = contract.ProjectRetrospectiveAccess{CanEdit: lifecycleStatus == retrospectiveDomain.StatusDraft && canEdit, CanPublish: authority.manager(), CanArchive: authority.manager()}
	return result, nil
}

func readProjectRetrospectiveParticipants(ctx context.Context, connection *sql.Conn, workspaceID, projectID, retrospectiveID string, revision int64) ([]contract.ProjectRetrospectiveParticipant, error) {
	rows, err := connection.QueryContext(ctx, `SELECT member_id,role FROM workspace_project_retrospective_participants
		WHERE workspace_id=? AND project_id=? AND retrospective_id=? AND revision=? ORDER BY member_id`, workspaceID, projectID, retrospectiveID, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]contract.ProjectRetrospectiveParticipant, 0)
	for rows.Next() {
		var participant contract.ProjectRetrospectiveParticipant
		if err = rows.Scan(&participant.MemberID, &participant.Role); err != nil {
			return nil, err
		}
		result = append(result, participant)
	}
	return result, rows.Err()
}

func insertProjectRetrospectiveAuditAndOutbox(ctx context.Context, connection *sql.Conn, value contract.ProjectRetrospective, actor contract.WorkspaceActor, requestID, action, eventType string, occurredAt time.Time) error {
	participantCount, actionItemCount := 0, 0
	if value.Current != nil {
		participantCount = len(value.Current.Participants)
		actionItemCount = len(value.Current.Content.ActionItems)
	}
	metadata, _ := json.Marshal(map[string]any{"version": "project-retrospective-v1", "action": action, "status": value.Status, "participant_count": participantCount, "action_item_count": actionItemCount})
	payload, _ := json.Marshal(map[string]any{"version": "project-retrospective-v1", "retrospective_id": value.ID, "project_id": value.ProjectID, "revision": value.CurrentRevision, "status": value.Status, "participant_count": participantCount, "action_item_count": actionItemCount})
	timestamp := occurredAt.UTC().Format(time.RFC3339Nano)
	auditID := value.ID + "-" + action + "-" + strconv.FormatInt(value.CurrentRevision, 10) + "-audit"
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_audit_entries(
		workspace_id,occurred_at,id,actor_type,actor_id,action,resource_kind,resource_id,resource_revision,request_id,metadata_json
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, value.WorkspaceID, timestamp, auditID, actor.Type, actor.ID, "workspace.project.retrospective."+action,
		projectRetrospectiveResourceKind, value.ID, value.CurrentRevision, requestID, string(metadata)); err != nil {
		return fmt.Errorf("record Project Retrospective audit: %w", err)
	}
	eventID := value.ID + "-" + action + "-" + strconv.FormatInt(value.CurrentRevision, 10) + "-event"
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_outbox_events(
		state,available_at,workspace_id,id,event_type,aggregate_kind,aggregate_id,aggregate_revision,payload_json,actor_type,actor_id,attempt_count,created_at
	) VALUES('ready',?,?,?,?,?,?,?,?,?,?,0,?)`, timestamp, value.WorkspaceID, eventID, eventType, projectRetrospectiveResourceKind,
		value.ID, value.CurrentRevision, string(payload), actor.Type, actor.ID, timestamp); err != nil {
		return fmt.Errorf("record Project Retrospective outbox: %w", err)
	}
	return nil
}

var _ application.ProjectRetrospectiveRepository = (*ProjectRetrospectiveRepository)(nil)
