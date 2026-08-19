package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
)

const (
	DefaultProjectRetrospectiveListLimit = 50
	MaxProjectRetrospectiveListLimit     = 100
)

var ErrInvalidProjectRetrospectiveRequest = errors.New("invalid Project Retrospective request")

type ProjectRetrospectiveListCursor struct {
	UpdatedAt string
	ID        string
}

type ProjectRetrospectiveListQuery struct {
	WorkspaceID     string
	ProjectID       string
	Limit           int
	Cursor          *ProjectRetrospectiveListCursor
	IncludeArchived bool
	Actor           contract.WorkspaceActor
}

type ProjectRetrospectivePage struct {
	Retrospectives []contract.ProjectRetrospective
	HasMore        bool
}

type ProjectRetrospectiveTaskCreationService interface {
	CreateTodo(context.Context, contract.CreateTodoRequest) (contract.CreateTodoResponse, error)
}

type CreateProjectRetrospectiveCommand struct {
	WorkspaceID     string
	ProjectID       string
	RetrospectiveID string
	Content         retrospectiveDomain.Content
	Participants    []retrospectiveDomain.Participant
	IdempotencyKey  string
	RequestHash     string
	Actor           contract.WorkspaceActor
	OccurredAt      time.Time
}

type MutateProjectRetrospectiveCommand struct {
	WorkspaceID      string
	ProjectID        string
	RetrospectiveID  string
	ExpectedRevision int64
	Action           string
	Content          *retrospectiveDomain.Content
	Participants     *[]retrospectiveDomain.Participant
	RequestID        string
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type PrepareProjectRetrospectiveTargetCommand struct {
	WorkspaceID, ProjectID, RetrospectiveID, ActionItemID string
	TargetKind, IdempotencyKey, RequestHash               string
	Actor                                                 contract.WorkspaceActor
	OccurredAt                                            time.Time
}

type ProjectRetrospectiveTargetClaim struct {
	ActionItem          contract.ProjectRetrospectiveActionItem
	SourceRevision      int64
	TargetKind          string
	TargetID            string
	ChildIdempotencyKey string
}

type CompleteProjectRetrospectiveTargetCommand struct {
	WorkspaceID, ProjectID, RetrospectiveID, ActionItemID string
	SourceRevision                                        int64
	TargetKind, TargetID, IdempotencyKey, RequestHash     string
	Actor                                                 contract.WorkspaceActor
	OccurredAt                                            time.Time
}

type ProjectRetrospectiveRepository interface {
	ReadProjectRetrospective(context.Context, string, string, string, contract.WorkspaceActor) (contract.ProjectRetrospective, error)
	ListProjectRetrospectives(context.Context, ProjectRetrospectiveListQuery) (ProjectRetrospectivePage, error)
	CreateProjectRetrospective(context.Context, CreateProjectRetrospectiveCommand) (contract.ProjectRetrospective, error)
	MutateProjectRetrospective(context.Context, MutateProjectRetrospectiveCommand) (contract.ProjectRetrospective, error)
	PrepareProjectRetrospectiveTarget(context.Context, PrepareProjectRetrospectiveTargetCommand) (ProjectRetrospectiveTargetClaim, error)
	CompleteProjectRetrospectiveTarget(context.Context, CompleteProjectRetrospectiveTargetCommand) (contract.ProjectRetrospectiveActionLink, error)
}

type ProjectRetrospectiveUseCase struct {
	repository      ProjectRetrospectiveRepository
	tasks           ProjectRetrospectiveTaskCreationService
	issues          contract.IdempotentIssueCreationService
	newID           func(context.Context) (string, error)
	newActionItemID func(context.Context) (string, error)
	now             func() time.Time
	cursorKey       []byte
}

func NewProjectRetrospectiveUseCase(
	repository ProjectRetrospectiveRepository,
	tasks ProjectRetrospectiveTaskCreationService,
	issues contract.IdempotentIssueCreationService,
	newID func(context.Context) (string, error),
	newActionItemID func(context.Context) (string, error),
	now func() time.Time,
	cursorKey []byte,
) (*ProjectRetrospectiveUseCase, error) {
	if repository == nil || tasks == nil || issues == nil || newID == nil || newActionItemID == nil || now == nil || len(cursorKey) < 32 {
		return nil, errors.New("Project Retrospective dependencies are required")
	}
	return &ProjectRetrospectiveUseCase{
		repository: repository, tasks: tasks, issues: issues, newID: newID,
		newActionItemID: newActionItemID, now: now, cursorKey: append([]byte(nil), cursorKey...),
	}, nil
}

func (u *ProjectRetrospectiveUseCase) ListProjectRetrospectives(ctx context.Context, workspaceID, projectID string, limit int, cursor string, includeArchived bool) (contract.ProjectRetrospectiveList, error) {
	workspaceID, projectID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID)
	actor, err := projectRetrospectiveActor(ctx)
	if err != nil {
		return contract.ProjectRetrospectiveList{}, err
	}
	if workspaceID == "" || projectID == "" {
		return contract.ProjectRetrospectiveList{}, ErrInvalidProjectRetrospectiveRequest
	}
	if limit == 0 {
		limit = DefaultProjectRetrospectiveListLimit
	}
	if limit < 1 || limit > MaxProjectRetrospectiveListLimit {
		return contract.ProjectRetrospectiveList{}, ErrInvalidProjectRetrospectiveRequest
	}
	query := ProjectRetrospectiveListQuery{WorkspaceID: workspaceID, ProjectID: projectID, Limit: limit, IncludeArchived: includeArchived, Actor: actor}
	if strings.TrimSpace(cursor) != "" {
		decoded, decodeErr := u.decodeProjectRetrospectiveCursor(strings.TrimSpace(cursor), workspaceID, projectID, includeArchived)
		if decodeErr != nil {
			return contract.ProjectRetrospectiveList{}, ErrInvalidProjectRetrospectiveRequest
		}
		query.Cursor = decoded
	}
	page, err := u.repository.ListProjectRetrospectives(ctx, query)
	if err != nil {
		return contract.ProjectRetrospectiveList{}, err
	}
	if page.Retrospectives == nil {
		page.Retrospectives = []contract.ProjectRetrospective{}
	}
	result := contract.ProjectRetrospectiveList{Retrospectives: page.Retrospectives}
	if page.HasMore {
		if len(page.Retrospectives) == 0 {
			return contract.ProjectRetrospectiveList{}, fmt.Errorf("invalid Project Retrospective page: %w", contract.ErrInvalidProjectRetrospective)
		}
		last := page.Retrospectives[len(page.Retrospectives)-1]
		result.NextCursor, err = u.encodeProjectRetrospectiveCursor(workspaceID, projectID, includeArchived, last.UpdatedAt, last.ID)
		if err != nil {
			return contract.ProjectRetrospectiveList{}, err
		}
	}
	return result, nil
}

func (u *ProjectRetrospectiveUseCase) GetProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string) (contract.ProjectRetrospective, error) {
	workspaceID, projectID, retrospectiveID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), strings.TrimSpace(retrospectiveID)
	actor, err := projectRetrospectiveActor(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if workspaceID == "" || projectID == "" || retrospectiveID == "" {
		return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
	}
	return u.repository.ReadProjectRetrospective(ctx, workspaceID, projectID, retrospectiveID, actor)
}

func (u *ProjectRetrospectiveUseCase) CreateProjectRetrospective(ctx context.Context, workspaceID, projectID, idempotencyKey string, request contract.CreateProjectRetrospectiveRequest) (contract.ProjectRetrospective, error) {
	workspaceID, projectID, idempotencyKey = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), strings.TrimSpace(idempotencyKey)
	actor, err := projectRetrospectiveActor(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if workspaceID == "" || projectID == "" || idempotencyKey == "" || len(idempotencyKey) > 200 {
		return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
	}
	content, hashContent, err := normalizeProjectRetrospectiveCreateContent(request.Content)
	if err != nil {
		return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
	}
	for index := range content.ActionItems {
		content.ActionItems[index].ID, err = u.newActionItemID(ctx)
		if err != nil || strings.TrimSpace(content.ActionItems[index].ID) == "" {
			return contract.ProjectRetrospective{}, fmt.Errorf("generate Project Retrospective action item ID: %w", err)
		}
	}
	participants := projectRetrospectiveParticipantsFromInput(request.Participants)
	hashParticipants := append([]retrospectiveDomain.Participant(nil), participants...)
	sort.Slice(hashParticipants, func(i, j int) bool {
		if hashParticipants[i].MemberID == hashParticipants[j].MemberID {
			return hashParticipants[i].Role < hashParticipants[j].Role
		}
		return hashParticipants[i].MemberID < hashParticipants[j].MemberID
	})
	requestHash, err := projectRetrospectiveHash(struct {
		Version      string                            `json:"version"`
		WorkspaceID  string                            `json:"workspace_id"`
		ProjectID    string                            `json:"project_id"`
		Content      retrospectiveDomain.Content       `json:"content"`
		Participants []retrospectiveDomain.Participant `json:"participants"`
	}{"project-retrospective-create-v1", workspaceID, projectID, hashContent, hashParticipants})
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	retrospectiveID, err := u.newID(ctx)
	if err != nil || strings.TrimSpace(retrospectiveID) == "" {
		return contract.ProjectRetrospective{}, fmt.Errorf("generate Project Retrospective ID: %w", err)
	}
	return u.repository.CreateProjectRetrospective(ctx, CreateProjectRetrospectiveCommand{
		WorkspaceID: workspaceID, ProjectID: projectID, RetrospectiveID: strings.TrimSpace(retrospectiveID),
		Content: content, Participants: participants, IdempotencyKey: idempotencyKey, RequestHash: requestHash,
		Actor: actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectRetrospectiveUseCase) UpdateProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string, request contract.UpdateProjectRetrospectiveRequest) (contract.ProjectRetrospective, error) {
	workspaceID, projectID, retrospectiveID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), strings.TrimSpace(retrospectiveID)
	actor, err := projectRetrospectiveActor(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	request.Action = strings.TrimSpace(request.Action)
	if workspaceID == "" || projectID == "" || retrospectiveID == "" || request.ExpectedRevision < 1 {
		return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
	}
	command := MutateProjectRetrospectiveCommand{
		WorkspaceID: workspaceID, ProjectID: projectID, RetrospectiveID: retrospectiveID,
		ExpectedRevision: request.ExpectedRevision, Action: request.Action,
		RequestID: projectRetrospectiveRequestID(retrospectiveID, request.Action, request.ExpectedRevision+1),
		Actor:     actor, OccurredAt: u.now().UTC(),
	}
	switch request.Action {
	case retrospectiveDomain.ActionSaveDraft, retrospectiveDomain.ActionPublishRevision:
		if request.Content == nil || request.Participants == nil {
			return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
		}
		content, normalizeErr := normalizeProjectRetrospectiveUpdateContent(*request.Content)
		if normalizeErr != nil {
			return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
		}
		participants := projectRetrospectiveParticipantsFromInput(*request.Participants)
		command.Content, command.Participants = &content, &participants
	case retrospectiveDomain.ActionPublish:
		if request.Content != nil || request.Participants != nil {
			return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
		}
	default:
		return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
	}
	return u.repository.MutateProjectRetrospective(ctx, command)
}

func (u *ProjectRetrospectiveUseCase) ArchiveProjectRetrospective(ctx context.Context, workspaceID, projectID, retrospectiveID string, expectedRevision int64) (contract.ProjectRetrospective, error) {
	workspaceID, projectID, retrospectiveID = strings.TrimSpace(workspaceID), strings.TrimSpace(projectID), strings.TrimSpace(retrospectiveID)
	actor, err := projectRetrospectiveActor(ctx)
	if err != nil {
		return contract.ProjectRetrospective{}, err
	}
	if workspaceID == "" || projectID == "" || retrospectiveID == "" || expectedRevision < 1 {
		return contract.ProjectRetrospective{}, ErrInvalidProjectRetrospectiveRequest
	}
	return u.repository.MutateProjectRetrospective(ctx, MutateProjectRetrospectiveCommand{
		WorkspaceID: workspaceID, ProjectID: projectID, RetrospectiveID: retrospectiveID,
		ExpectedRevision: expectedRevision, Action: retrospectiveDomain.ActionArchive,
		RequestID: projectRetrospectiveRequestID(retrospectiveID, retrospectiveDomain.ActionArchive, expectedRevision+1),
		Actor:     actor, OccurredAt: u.now().UTC(),
	})
}

func (u *ProjectRetrospectiveUseCase) CreateProjectRetrospectiveTarget(
	ctx context.Context,
	workspaceID, projectID, retrospectiveID, actionItemID, idempotencyKey string,
	request contract.CreateProjectRetrospectiveTargetRequest,
) (contract.ProjectRetrospectiveActionLink, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	retrospectiveID = strings.TrimSpace(retrospectiveID)
	actionItemID = strings.TrimSpace(actionItemID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	actor, err := projectRetrospectiveActor(ctx)
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, err
	}
	if workspaceID == "" || projectID == "" || retrospectiveID == "" || actionItemID == "" || idempotencyKey == "" || len(idempotencyKey) > 200 {
		return contract.ProjectRetrospectiveActionLink{}, ErrInvalidProjectRetrospectiveRequest
	}
	targetKind := "task"
	if request.TargetKind != nil {
		targetKind = strings.TrimSpace(*request.TargetKind)
		if targetKind == "" {
			return contract.ProjectRetrospectiveActionLink{}, ErrInvalidProjectRetrospectiveRequest
		}
	}
	if targetKind != "task" && targetKind != "issue" {
		return contract.ProjectRetrospectiveActionLink{}, ErrInvalidProjectRetrospectiveRequest
	}
	requestHash, err := projectRetrospectiveHash(struct {
		Version, WorkspaceID, ProjectID, RetrospectiveID, ActionItemID, TargetKind string
	}{"project-retrospective-target-v1", workspaceID, projectID, retrospectiveID, actionItemID, targetKind})
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, err
	}
	when := u.now().UTC()
	claim, err := u.repository.PrepareProjectRetrospectiveTarget(ctx, PrepareProjectRetrospectiveTargetCommand{
		WorkspaceID: workspaceID, ProjectID: projectID, RetrospectiveID: retrospectiveID, ActionItemID: actionItemID,
		TargetKind: targetKind, IdempotencyKey: idempotencyKey, RequestHash: requestHash, Actor: actor, OccurredAt: when,
	})
	if err != nil {
		return contract.ProjectRetrospectiveActionLink{}, err
	}
	if claim.ActionItem.ID != actionItemID || strings.TrimSpace(claim.ActionItem.Title) == "" || claim.SourceRevision < 1 || claim.TargetKind != targetKind || strings.TrimSpace(claim.ChildIdempotencyKey) == "" || len(claim.ChildIdempotencyKey) > 200 {
		return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("invalid Project Retrospective target claim: %w", contract.ErrInvalidGovernanceMutation)
	}
	targetID := strings.TrimSpace(claim.TargetID)
	if targetID == "" {
		projectReference := projectID
		var assigneeType, assigneeID, dueDate *string
		if value := strings.TrimSpace(claim.ActionItem.AssigneeID); value != "" {
			typeValue, idValue := "member", value
			assigneeType, assigneeID = &typeValue, &idValue
		}
		if value := strings.TrimSpace(claim.ActionItem.DueDate); value != "" {
			dueDate = &value
		}
		switch targetKind {
		case "task":
			response, createErr := u.tasks.CreateTodo(ctx, contract.CreateTodoRequest{
				IdempotencyKey: claim.ChildIdempotencyKey, WorkspaceId: workspaceID,
				Title: claim.ActionItem.Title, Description: claim.ActionItem.Description,
				ProjectId: &projectReference, AssigneeType: assigneeType, AssigneeId: assigneeID,
				Status: "todo", Priority: "none", DueDate: dueDate,
			})
			if createErr != nil {
				return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("create Project Retrospective Task target: %w", createErr)
			}
			if response.Todo == nil || strings.TrimSpace(response.Todo.Id) == "" {
				return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("create Project Retrospective Task target: %w", contract.ErrInvalidGovernanceMutation)
			}
			targetID = strings.TrimSpace(response.Todo.Id)
		case "issue":
			var description *string
			if value := strings.TrimSpace(claim.ActionItem.Description); value != "" {
				description = &value
			}
			response, createErr := u.issues.CreateIssueIdempotently(ctx, contract.IdempotentCreateIssueRequest{
				CreateIssueRequest: contract.CreateIssueRequest{
					WorkspaceId: workspaceID, Title: claim.ActionItem.Title, Description: description,
					Status: "todo", Priority: "none", AssigneeType: assigneeType, AssigneeId: assigneeID,
					ProjectId: &projectReference, DueDate: dueDate,
				},
				IdempotencyKey: claim.ChildIdempotencyKey, RequestHash: requestHash,
			})
			if createErr != nil {
				return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("create Project Retrospective Issue target: %w", createErr)
			}
			if response.Issue == nil || strings.TrimSpace(response.Issue.Id) == "" {
				return contract.ProjectRetrospectiveActionLink{}, fmt.Errorf("create Project Retrospective Issue target: %w", contract.ErrInvalidGovernanceMutation)
			}
			targetID = strings.TrimSpace(response.Issue.Id)
		}
	}
	return u.repository.CompleteProjectRetrospectiveTarget(ctx, CompleteProjectRetrospectiveTargetCommand{
		WorkspaceID: workspaceID, ProjectID: projectID, RetrospectiveID: retrospectiveID, ActionItemID: actionItemID,
		SourceRevision: claim.SourceRevision, TargetKind: targetKind, TargetID: targetID,
		IdempotencyKey: idempotencyKey, RequestHash: requestHash, Actor: actor, OccurredAt: when,
	})
}

func normalizeProjectRetrospectiveCreateContent(input contract.ProjectRetrospectiveContentInput) (retrospectiveDomain.Content, retrospectiveDomain.Content, error) {
	content := projectRetrospectiveContentFromInput(input)
	for index, item := range input.ActionItems {
		if strings.TrimSpace(item.ID) != "" {
			return retrospectiveDomain.Content{}, retrospectiveDomain.Content{}, ErrInvalidProjectRetrospectiveRequest
		}
		content.ActionItems[index].ID = "input-" + strconv.Itoa(index+1)
	}
	normalized, err := retrospectiveDomain.NormalizeContent(content)
	if err != nil {
		return retrospectiveDomain.Content{}, retrospectiveDomain.Content{}, err
	}
	commandContent, hashContent := normalized, normalized
	commandContent.ActionItems = append([]retrospectiveDomain.ActionItem(nil), normalized.ActionItems...)
	hashContent.ActionItems = append([]retrospectiveDomain.ActionItem(nil), normalized.ActionItems...)
	return commandContent, hashContent, nil
}

func normalizeProjectRetrospectiveUpdateContent(input contract.ProjectRetrospectiveContentInput) (retrospectiveDomain.Content, error) {
	content := projectRetrospectiveContentFromInput(input)
	for _, item := range content.ActionItems {
		if item.ID == "" {
			return retrospectiveDomain.Content{}, ErrInvalidProjectRetrospectiveRequest
		}
	}
	return retrospectiveDomain.NormalizeContent(content)
}

func projectRetrospectiveContentFromInput(input contract.ProjectRetrospectiveContentInput) retrospectiveDomain.Content {
	actions := make([]retrospectiveDomain.ActionItem, len(input.ActionItems))
	for index, item := range input.ActionItems {
		actions[index] = retrospectiveDomain.ActionItem{ID: strings.TrimSpace(item.ID), Title: item.Title, Description: item.Description, AssigneeID: item.AssigneeID, DueDate: item.DueDate}
	}
	return retrospectiveDomain.Content{Summary: input.Summary, Successes: input.Successes, Problems: input.Problems, Lessons: input.Lessons, ActionItems: actions}
}

func projectRetrospectiveParticipantsFromInput(input []contract.ProjectRetrospectiveParticipantInput) []retrospectiveDomain.Participant {
	result := make([]retrospectiveDomain.Participant, len(input))
	for index, participant := range input {
		result[index] = retrospectiveDomain.Participant{MemberID: strings.TrimSpace(participant.MemberID), Role: strings.TrimSpace(participant.Role)}
	}
	return result
}

func projectRetrospectiveActor(ctx context.Context) (contract.WorkspaceActor, error) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || strings.TrimSpace(actor.Type) == "" || strings.TrimSpace(actor.ID) == "" {
		return contract.WorkspaceActor{}, contract.ErrWorkspaceActorRequired
	}
	actor.Type, actor.ID = strings.TrimSpace(actor.Type), strings.TrimSpace(actor.ID)
	return actor, nil
}

func projectRetrospectiveHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func projectRetrospectiveRequestID(retrospectiveID, action string, revision int64) string {
	return retrospectiveID + ":" + action + ":" + strconv.FormatInt(revision, 10)
}

type projectRetrospectiveCursorEnvelope struct {
	Version         int    `json:"version"`
	WorkspaceID     string `json:"workspace_id"`
	ProjectID       string `json:"project_id"`
	IncludeArchived bool   `json:"include_archived"`
	UpdatedAt       string `json:"updated_at"`
	ID              string `json:"id"`
}

func (u *ProjectRetrospectiveUseCase) encodeProjectRetrospectiveCursor(workspaceID, projectID string, includeArchived bool, updatedAt, id string) (string, error) {
	envelope := projectRetrospectiveCursorEnvelope{Version: 1, WorkspaceID: workspaceID, ProjectID: projectID, IncludeArchived: includeArchived, UpdatedAt: updatedAt, ID: id}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, u.cursorKey)
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (u *ProjectRetrospectiveUseCase) decodeProjectRetrospectiveCursor(value, workspaceID, projectID string, includeArchived bool) (*ProjectRetrospectiveListCursor, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidProjectRetrospectiveRequest
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, u.cursorKey)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, ErrInvalidProjectRetrospectiveRequest
	}
	var envelope projectRetrospectiveCursorEnvelope
	if err = json.Unmarshal(payload, &envelope); err != nil || envelope.Version != 1 || envelope.WorkspaceID != workspaceID || envelope.ProjectID != projectID || envelope.IncludeArchived != includeArchived || strings.TrimSpace(envelope.UpdatedAt) == "" || strings.TrimSpace(envelope.ID) == "" {
		return nil, ErrInvalidProjectRetrospectiveRequest
	}
	return &ProjectRetrospectiveListCursor{UpdatedAt: envelope.UpdatedAt, ID: envelope.ID}, nil
}
