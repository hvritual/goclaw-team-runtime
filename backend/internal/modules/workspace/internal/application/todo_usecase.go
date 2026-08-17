package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

var ErrTodoRecordNotFound = errors.New("todo record not found")

const (
	DefaultTodoListLimit = 50
	MaxTodoListLimit     = 100
)

type TodoListQuery struct {
	WorkspaceID string
	ProjectID   *string
	IssueID     *string
	Status      string
	Limit       int
	Cursor      *TodoListCursor
}

type TodoListCursor struct {
	Position  float64
	CreatedAt string
	ID        string
}

type TodoRepository interface {
	Create(context.Context, todoDomain.Todo) (todoDomain.Todo, error)
	FindByID(context.Context, string, string) (todoDomain.Todo, error)
	List(context.Context, TodoListQuery) ([]todoDomain.Todo, error)
	Update(context.Context, todoDomain.Todo) error
	Reorder(context.Context, string, []TodoPositionUpdate, time.Time) ([]todoDomain.Todo, error)
}

type TodoPositionUpdate struct {
	TodoID           string
	Position         float64
	ExpectedRevision int64
}

type TodoUseCase struct {
	repository TodoRepository
	projects   ProjectRepository
	issues     IssueReferenceRepository
	authorizer contract.WorkspaceAccessAuthorizer
	actors     contract.WorkspaceActorReader
	newID      ProjectIDGenerator
	now        Clock
	cursorKey  []byte
}

func NewTodoUseCase(repository TodoRepository, projects ProjectRepository, issues IssueReferenceRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, newID ProjectIDGenerator, now Clock, cursorKey []byte) (*TodoUseCase, error) {
	if repository == nil || projects == nil || issues == nil || authorizer == nil || actors == nil || newID == nil || now == nil || len(cursorKey) < 32 {
		return nil, errors.New("Todo dependencies are required")
	}
	return &TodoUseCase{repository: repository, projects: projects, issues: issues, authorizer: authorizer, actors: actors, newID: newID, now: now, cursorKey: append([]byte(nil), cursorKey...)}, nil
}

func (s *TodoUseCase) CreateTodo(ctx context.Context, request contract.CreateTodoRequest) (contract.CreateTodoResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidTodo)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskCreate); err != nil {
		return contract.CreateTodoResponse{}, err
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return contract.CreateTodoResponse{}, contract.ErrWorkspaceActorRequired
	}
	if err := s.requireWorkspaceActor(ctx, workspaceID, actor.Type, actor.ID); err != nil {
		return contract.CreateTodoResponse{}, err
	}
	projectID := cleanOptionalString(request.ProjectId)
	if err := s.validateProject(ctx, workspaceID, projectID); err != nil {
		return contract.CreateTodoResponse{}, err
	}
	issueID, err := s.canonicalIssueID(ctx, workspaceID, cleanOptionalString(request.IssueId))
	if err != nil {
		return contract.CreateTodoResponse{}, err
	}
	startDate, err := parseTodoTime(request.StartDate)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: invalid start date", contract.ErrInvalidTodo)
	}
	dueDate, err := parseTodoTime(request.DueDate)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: invalid due date", contract.ErrInvalidTodo)
	}
	id, err := s.newID(ctx)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("generate Todo id: %w", err)
	}
	value, err := todoDomain.New(
		id, workspaceID, request.Title, request.Description, request.Status, request.Priority,
		projectID, issueID, request.AssigneeType, request.AssigneeId,
		actor.Type, actor.ID, request.Position, startDate, dueDate, s.now(),
	)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTodo, err)
	}
	if value.AssigneeID != nil {
		if err := s.requireWorkspaceActor(ctx, workspaceID, *value.AssigneeType, *value.AssigneeID); err != nil {
			return contract.CreateTodoResponse{}, err
		}
	}
	fingerprint, err := todoCreateFingerprint(value)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("fingerprint Todo create: %w", err)
	}
	value, err = s.repository.Create(WithTodoCreateGovernance(ctx, request.IdempotencyKey, fingerprint), value)
	if err != nil {
		return contract.CreateTodoResponse{}, fmt.Errorf("create Todo: %w", err)
	}
	result := todoToContract(value)
	return contract.CreateTodoResponse{Todo: &result}, nil
}

func todoCreateFingerprint(value todoDomain.Todo) (string, error) {
	payload, err := json.Marshal(struct {
		WorkspaceID, Title, Description, Status, Priority, CreatorType, CreatorID string
		ProjectID, IssueID, AssigneeType, AssigneeID                              *string
		Position                                                                  float64
		StartDate, DueDate                                                        *time.Time
	}{
		WorkspaceID: value.WorkspaceID, Title: value.Title, Description: value.Description,
		Status: value.Status, Priority: value.Priority, CreatorType: value.CreatorType, CreatorID: value.CreatorID,
		ProjectID: value.ProjectID, IssueID: value.IssueID, AssigneeType: value.AssigneeType, AssigneeID: value.AssigneeID,
		Position: value.Position, StartDate: value.StartDate, DueDate: value.DueDate,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func (s *TodoUseCase) GetTodo(ctx context.Context, request contract.GetTodoRequest) (contract.GetTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.GetTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.GetTodoResponse{}, err
	}
	value, err := s.findTodo(ctx, workspaceID, todoID)
	if err != nil {
		return contract.GetTodoResponse{}, err
	}
	result := todoToContract(value)
	return contract.GetTodoResponse{Todo: &result}, nil
}

func (s *TodoUseCase) ListTodos(ctx context.Context, request contract.ListTodosRequest) (contract.ListTodosResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	if workspaceID == "" {
		return contract.ListTodosResponse{}, fmt.Errorf("%w: workspace id is required", contract.ErrInvalidTodo)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.ListTodosResponse{}, err
	}
	status := request.Status
	if status != "" && !todoDomain.ValidStatus(status) {
		return contract.ListTodosResponse{}, fmt.Errorf("%w: invalid status", contract.ErrInvalidTodo)
	}
	limit := int(request.Limit)
	if limit < 0 {
		return contract.ListTodosResponse{}, fmt.Errorf("%w: limit must be a positive integer", contract.ErrInvalidTodo)
	}
	if limit == 0 {
		limit = DefaultTodoListLimit
	}
	if limit > MaxTodoListLimit {
		limit = MaxTodoListLimit
	}
	projectID := cleanOptionalString(request.ProjectId)
	issueID := cleanOptionalString(request.IssueId)
	filterHash, err := todoListFilterHash(workspaceID, projectID, issueID, status)
	if err != nil {
		return contract.ListTodosResponse{}, fmt.Errorf("hash Todo list filters: %w", err)
	}
	cursor, err := decodeTodoListCursor(request.Cursor, filterHash, s.cursorKey)
	if err != nil {
		return contract.ListTodosResponse{}, fmt.Errorf("%w: %w", contract.ErrInvalidTodo, contract.ErrInvalidTodoCursor)
	}
	values, err := s.repository.List(ctx, TodoListQuery{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		IssueID:     issueID,
		Status:      status,
		Limit:       limit,
		Cursor:      cursor,
	})
	if err != nil {
		return contract.ListTodosResponse{}, fmt.Errorf("list Todos: %w", err)
	}
	var nextCursor *string
	if len(values) > limit {
		values = values[:limit]
		encoded, encodeErr := encodeTodoListCursor(filterHash, values[len(values)-1], s.cursorKey)
		if encodeErr != nil {
			return contract.ListTodosResponse{}, fmt.Errorf("encode Todo list cursor: %w", encodeErr)
		}
		nextCursor = &encoded
	}
	result := make([]contract.Todo, len(values))
	for index, value := range values {
		result[index] = todoToContract(value)
	}
	return contract.ListTodosResponse{Todos: result, Total: int32(len(result)), NextCursor: nextCursor}, nil
}

const todoListCursorVersion = "task-list-v1"

type todoListCursorPayload struct {
	Version    string  `json:"v"`
	FilterHash string  `json:"f"`
	Position   float64 `json:"p"`
	CreatedAt  string  `json:"c"`
	ID         string  `json:"i"`
}

func todoListFilterHash(workspaceID string, projectID, issueID *string, status string) (string, error) {
	payload, err := json.Marshal(struct {
		WorkspaceID string  `json:"workspace_id"`
		ProjectID   *string `json:"project_id"`
		IssueID     *string `json:"issue_id"`
		Status      string  `json:"status"`
	}{workspaceID, projectID, issueID, status})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func encodeTodoListCursor(filterHash string, value todoDomain.Todo, key []byte) (string, error) {
	payload, err := json.Marshal(todoListCursorPayload{
		Version: todoListCursorVersion, FilterHash: filterHash, Position: value.Position,
		CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano), ID: value.ID,
	})
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func decodeTodoListCursor(raw, filterHash string, key []byte) (*TodoListCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if len(raw) > 2048 {
		return nil, contract.ErrInvalidTodo
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return nil, contract.ErrInvalidTodo
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, contract.ErrInvalidTodo
	}
	signature, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, contract.ErrInvalidTodo
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, contract.ErrInvalidTodo
	}
	var decoded todoListCursorPayload
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil || decoded.Version != todoListCursorVersion || decoded.FilterHash != filterHash || strings.TrimSpace(decoded.ID) == "" || decoded.CreatedAt == "" || math.IsNaN(decoded.Position) || math.IsInf(decoded.Position, 0) {
		return nil, contract.ErrInvalidTodo
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, contract.ErrInvalidTodo
	}
	if _, err := time.Parse(time.RFC3339Nano, decoded.CreatedAt); err != nil {
		return nil, contract.ErrInvalidTodo
	}
	return &TodoListCursor{Position: decoded.Position, CreatedAt: decoded.CreatedAt, ID: strings.TrimSpace(decoded.ID)}, nil
}

func (s *TodoUseCase) UpdateTodo(ctx context.Context, request contract.UpdateTodoRequest) (contract.UpdateTodoResponse, error) {
	return s.updateTodo(ctx, request)
}

func (s *TodoUseCase) UpdateTodoStatus(ctx context.Context, request contract.UpdateTodoStatusRequest) (contract.UpdateTodoStatusResponse, error) {
	status := request.Status
	updated, err := s.updateTodo(ctx, contract.UpdateTodoRequest{
		WorkspaceId:      request.WorkspaceId,
		TodoId:           request.TodoId,
		Status:           &status,
		ExpectedRevision: request.ExpectedRevision,
	})
	if err != nil {
		return contract.UpdateTodoStatusResponse{}, err
	}
	return contract.UpdateTodoStatusResponse{Todo: updated.Todo}, nil
}

func (s *TodoUseCase) DeleteTodo(ctx context.Context, request contract.DeleteTodoRequest) (contract.DeleteTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.DeleteTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.DeleteTodoResponse{}, err
	}
	value, err := s.findTodo(ctx, workspaceID, todoID)
	if err != nil {
		return contract.DeleteTodoResponse{}, err
	}
	if err := s.authorizeTaskMutation(ctx, workspaceID, value); err != nil {
		return contract.DeleteTodoResponse{}, err
	}
	if request.ExpectedRevision != value.Revision {
		return contract.DeleteTodoResponse{}, contract.RevisionConflictError{CurrentRevision: value.Revision}
	}
	archived, err := value.Archive(s.now())
	if err != nil {
		return contract.DeleteTodoResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTodo, err)
	}
	if err := s.repository.Update(WithTodoGovernanceAction(ctx, TaskActionArchive), archived); errors.Is(err, ErrTodoRecordNotFound) {
		return contract.DeleteTodoResponse{}, contract.ErrTodoNotFound
	} else if err != nil {
		return contract.DeleteTodoResponse{}, fmt.Errorf("archive Todo: %w", err)
	}
	return contract.DeleteTodoResponse{}, nil
}

func (s *TodoUseCase) RestoreTodo(ctx context.Context, request contract.RestoreTodoRequest) (contract.RestoreTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.RestoreTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.RestoreTodoResponse{}, err
	}
	value, err := s.findTodo(ctx, workspaceID, todoID)
	if err != nil {
		return contract.RestoreTodoResponse{}, err
	}
	if err := s.authorizeTaskMutation(ctx, workspaceID, value); err != nil {
		return contract.RestoreTodoResponse{}, err
	}
	if request.ExpectedRevision != value.Revision {
		return contract.RestoreTodoResponse{}, contract.RevisionConflictError{CurrentRevision: value.Revision}
	}
	restored, err := value.Restore(s.now())
	if err != nil {
		return contract.RestoreTodoResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTodo, err)
	}
	if err := s.repository.Update(WithTodoGovernanceAction(ctx, TaskActionRestore), restored); errors.Is(err, ErrTodoRecordNotFound) {
		return contract.RestoreTodoResponse{}, contract.ErrTodoNotFound
	} else if err != nil {
		return contract.RestoreTodoResponse{}, fmt.Errorf("restore Todo: %w", err)
	}
	result := todoToContract(restored)
	return contract.RestoreTodoResponse{Todo: &result}, nil
}

func (s *TodoUseCase) ReorderTodos(ctx context.Context, request contract.ReorderTodosRequest) (contract.ReorderTodosResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceId)
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if workspaceID == "" || len(request.Items) == 0 || idempotencyKey == "" {
		return contract.ReorderTodosResponse{}, fmt.Errorf("%w: workspace id and reorder items are required", contract.ErrInvalidTodo)
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.ReorderTodosResponse{}, err
	}
	updates := make([]TodoPositionUpdate, 0, len(request.Items))
	seen := make(map[string]struct{}, len(request.Items))
	for _, item := range request.Items {
		todoID := strings.TrimSpace(item.TodoId)
		if todoID == "" || item.ExpectedRevision < 1 || math.IsNaN(item.Position) || math.IsInf(item.Position, 0) {
			return contract.ReorderTodosResponse{}, fmt.Errorf("%w: invalid reorder item", contract.ErrInvalidTodo)
		}
		if _, duplicate := seen[todoID]; duplicate {
			return contract.ReorderTodosResponse{}, fmt.Errorf("%w: duplicate reorder item", contract.ErrInvalidTodo)
		}
		seen[todoID] = struct{}{}
		value, err := s.findTodo(ctx, workspaceID, todoID)
		if err != nil {
			return contract.ReorderTodosResponse{}, err
		}
		if err := s.authorizeTaskMutation(ctx, workspaceID, value); err != nil {
			return contract.ReorderTodosResponse{}, err
		}
		updates = append(updates, TodoPositionUpdate{TodoID: todoID, Position: item.Position, ExpectedRevision: item.ExpectedRevision})
	}
	fingerprint, err := todoReorderFingerprint(updates)
	if err != nil {
		return contract.ReorderTodosResponse{}, fmt.Errorf("fingerprint Todo reorder: %w", err)
	}
	values, err := s.repository.Reorder(WithTodoReorderGovernance(ctx, idempotencyKey, fingerprint), workspaceID, updates, s.now())
	if err != nil {
		return contract.ReorderTodosResponse{}, err
	}
	result := make([]contract.Todo, len(values))
	for index := range values {
		result[index] = todoToContract(values[index])
	}
	return contract.ReorderTodosResponse{Todos: result}, nil
}

func todoReorderFingerprint(updates []TodoPositionUpdate) (string, error) {
	payload, err := json.Marshal(updates)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func (s *TodoUseCase) updateTodo(ctx context.Context, request contract.UpdateTodoRequest) (contract.UpdateTodoResponse, error) {
	workspaceID, todoID, err := validateTodoIdentity(request.WorkspaceId, request.TodoId)
	if err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionTaskRead); err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	value, err := s.findTodo(ctx, workspaceID, todoID)
	if err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	if err := s.authorizeTaskMutation(ctx, workspaceID, value); err != nil {
		return contract.UpdateTodoResponse{}, err
	}
	if request.ExpectedRevision != value.Revision {
		return contract.UpdateTodoResponse{}, contract.RevisionConflictError{CurrentRevision: value.Revision}
	}
	patch := todoDomain.Patch{
		Title: request.Title, Description: request.Description, Status: request.Status,
		Priority: request.Priority, Position: request.Position,
		ProjectID: todoStringChange(request.ProjectId), IssueID: todoStringChange(request.IssueId),
		AssigneeType: todoStringChange(request.AssigneeType), AssigneeID: todoStringChange(request.AssigneeId),
	}
	if request.ProjectId != nil {
		if err := s.validateProject(ctx, workspaceID, cleanOptionalString(request.ProjectId)); err != nil {
			return contract.UpdateTodoResponse{}, err
		}
	}
	if request.IssueId != nil {
		canonicalID, canonicalErr := s.canonicalIssueID(ctx, workspaceID, cleanOptionalString(request.IssueId))
		if canonicalErr != nil {
			return contract.UpdateTodoResponse{}, canonicalErr
		}
		patch.IssueID.Value = canonicalID
	}
	patch.StartDate, err = parseTodoTimeChange(request.StartDate)
	if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("%w: invalid start date", contract.ErrInvalidTodo)
	}
	patch.DueDate, err = parseTodoTimeChange(request.DueDate)
	if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("%w: invalid due date", contract.ErrInvalidTodo)
	}
	updated, err := value.Apply(patch, s.now())
	if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("%w: %v", contract.ErrInvalidTodo, err)
	}
	if (request.AssigneeType != nil || request.AssigneeId != nil) && updated.AssigneeID != nil {
		if err := s.requireWorkspaceActor(ctx, workspaceID, *updated.AssigneeType, *updated.AssigneeID); err != nil {
			return contract.UpdateTodoResponse{}, err
		}
	}
	if err := s.repository.Update(WithTodoGovernanceAction(ctx, TaskActionUpdate), updated); errors.Is(err, ErrTodoRecordNotFound) {
		return contract.UpdateTodoResponse{}, contract.ErrTodoNotFound
	} else if err != nil {
		return contract.UpdateTodoResponse{}, fmt.Errorf("update Todo: %w", err)
	}
	result := todoToContract(updated)
	return contract.UpdateTodoResponse{Todo: &result}, nil
}

func (s *TodoUseCase) authorizeTaskMutation(ctx context.Context, workspaceID string, value todoDomain.Todo) error {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok {
		return contract.ErrWorkspaceActorRequired
	}
	permission := contract.PermissionTaskManageWorkspace
	if actor.Type == value.CreatorType && actor.ID == value.CreatorID {
		permission = contract.PermissionTaskUpdateOwn
	}
	return s.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission)
}

func (s *TodoUseCase) findTodo(ctx context.Context, workspaceID, todoID string) (todoDomain.Todo, error) {
	value, err := s.repository.FindByID(ctx, workspaceID, todoID)
	if errors.Is(err, ErrTodoRecordNotFound) {
		return todoDomain.Todo{}, contract.ErrTodoNotFound
	}
	if err != nil {
		return todoDomain.Todo{}, fmt.Errorf("get Todo: %w", err)
	}
	return value, nil
}

func (s *TodoUseCase) validateProject(ctx context.Context, workspaceID string, projectID *string) error {
	if projectID == nil {
		return nil
	}
	if _, err := s.projects.FindByID(ctx, workspaceID, *projectID); errors.Is(err, ErrProjectRecordNotFound) {
		return contract.ErrProjectNotFound
	} else if err != nil {
		return fmt.Errorf("validate Todo Project: %w", err)
	}
	return nil
}

func (s *TodoUseCase) canonicalIssueID(ctx context.Context, workspaceID string, issueID *string) (*string, error) {
	if issueID == nil {
		return nil, nil
	}
	issue, err := s.issues.FindByIDOrIdentifier(ctx, workspaceID, *issueID)
	if errors.Is(err, ErrIssueRecordNotFound) {
		return nil, contract.ErrIssueNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate Todo Issue: %w", err)
	}
	canonical := issue.ID
	return &canonical, nil
}

func (s *TodoUseCase) requireWorkspaceActor(ctx context.Context, workspaceID, actorType, actorID string) error {
	belongs, err := s.actors.ActorBelongsToWorkspace(ctx, workspaceID, actorType, actorID)
	if err != nil {
		return fmt.Errorf("validate Todo actor: %w", err)
	}
	if !belongs {
		return contract.ErrActorOutsideWorkspace
	}
	return nil
}

func validateTodoIdentity(workspaceID, todoID string) (string, string, error) {
	workspaceID, todoID = strings.TrimSpace(workspaceID), strings.TrimSpace(todoID)
	if workspaceID == "" || todoID == "" {
		return "", "", fmt.Errorf("%w: workspace id and Todo id are required", contract.ErrInvalidTodo)
	}
	return workspaceID, todoID, nil
}

func todoStringChange(value *string) todoDomain.StringChange {
	if value == nil {
		return todoDomain.StringChange{}
	}
	return todoDomain.StringChange{Set: true, Value: cleanOptionalString(value)}
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseTodoTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		parsed, err = time.Parse(time.DateOnly, *value)
		if err != nil {
			return nil, err
		}
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func parseTodoTimeChange(value *string) (todoDomain.TimeChange, error) {
	if value == nil {
		return todoDomain.TimeChange{}, nil
	}
	parsed, err := parseTodoTime(value)
	if err != nil {
		return todoDomain.TimeChange{}, err
	}
	return todoDomain.TimeChange{Set: true, Value: parsed}, nil
}

func todoToContract(value todoDomain.Todo) contract.Todo {
	return contract.Todo{
		Id: value.ID, WorkspaceId: value.WorkspaceID, Title: value.Title,
		Description: value.Description, Status: value.Status, Priority: value.Priority,
		ProjectId: copyTodoString(value.ProjectID), IssueId: copyTodoString(value.IssueID),
		AssigneeType: copyTodoString(value.AssigneeType), AssigneeId: copyTodoString(value.AssigneeID),
		CreatorType: value.CreatorType, CreatorId: value.CreatorID, Position: value.Position,
		Revision: value.Revision, RestoreStatus: value.RestoreStatus,
		StartDate: formatTodoTime(value.StartDate), DueDate: formatTodoTime(value.DueDate),
		CompletedAt: formatTodoTime(value.CompletedAt),
		ArchivedAt:  formatTodoTime(value.ArchivedAt),
		CreatedAt:   value.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func copyTodoString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func formatTodoTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

var _ contract.TodoService = (*TodoUseCase)(nil)
