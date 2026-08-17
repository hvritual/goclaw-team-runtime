package application

import (
	"context"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	knowledgeDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/knowledge"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
	settingDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/setting"
	todoDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/todo"
)

type todoRepositoryCounter struct{ calls int }

func (r *todoRepositoryCounter) Create(_ context.Context, value todoDomain.Todo) (todoDomain.Todo, error) {
	r.calls++
	return value, nil
}
func (r *todoRepositoryCounter) FindByID(context.Context, string, string) (todoDomain.Todo, error) {
	r.calls++
	return todoDomain.Todo{}, nil
}
func (r *todoRepositoryCounter) List(context.Context, TodoListQuery) ([]todoDomain.Todo, error) {
	r.calls++
	return nil, nil
}
func (r *todoRepositoryCounter) Update(context.Context, todoDomain.Todo) error {
	r.calls++
	return nil
}
func (r *todoRepositoryCounter) Reorder(context.Context, string, []TodoPositionUpdate, time.Time) ([]todoDomain.Todo, error) {
	r.calls++
	return nil, nil
}

type issueRepositoryCounter struct{ calls int }

func (r *issueRepositoryCounter) Create(context.Context, issueDomain.Issue) (issueDomain.Issue, error) {
	r.calls++
	return issueDomain.Issue{}, nil
}

func (r *issueRepositoryCounter) FindByIDOrIdentifier(context.Context, string, string) (issueDomain.Issue, error) {
	r.calls++
	return issueDomain.Issue{}, nil
}
func (r *issueRepositoryCounter) List(context.Context, IssueListQuery) ([]issueDomain.Issue, error) {
	r.calls++
	return nil, nil
}
func (r *issueRepositoryCounter) Update(context.Context, IssueUpdateCommand) (issueDomain.Issue, error) {
	r.calls++
	return issueDomain.Issue{}, nil
}
func (r *issueRepositoryCounter) Move(context.Context, IssueMoveCommand) (issueDomain.Issue, error) {
	r.calls++
	return issueDomain.Issue{}, nil
}
func (r *issueRepositoryCounter) WouldCreateParentCycle(context.Context, string, string, string) (bool, error) {
	r.calls++
	return false, nil
}

type knowledgeRepositoryCounter struct{ calls int }

func (r *knowledgeRepositoryCounter) Create(context.Context, knowledgeDomain.Knowledge) error {
	r.calls++
	return nil
}
func (r *knowledgeRepositoryCounter) FindByID(context.Context, string, string) (knowledgeDomain.Knowledge, error) {
	r.calls++
	return knowledgeDomain.Knowledge{}, nil
}

type requirementRepositoryCounter struct{ calls int }

func (r *requirementRepositoryCounter) FindByID(context.Context, string, string) (requirementDomain.Requirement, requirementDomain.Version, error) {
	r.calls++
	return requirementDomain.Requirement{}, requirementDomain.Version{}, nil
}
func (r *requirementRepositoryCounter) SaveVersion(context.Context, requirementDomain.Requirement, requirementDomain.Version, bool) error {
	r.calls++
	return nil
}

type settingRepositoryCounter struct{ calls int }

func (r *settingRepositoryCounter) PutSetting(context.Context, settingDomain.Setting) error {
	r.calls++
	return nil
}
func (r *settingRepositoryCounter) PutSkillBinding(context.Context, settingDomain.SkillBinding) error {
	r.calls++
	return nil
}

type assetReaderStub struct{}

func (*assetReaderStub) AssetBelongsToWorkspace(context.Context, string, string) (bool, error) {
	return true, nil
}

type skillReaderStub struct{}

func (*skillReaderStub) SkillReferenceExists(context.Context, string, *string) (bool, error) {
	return true, nil
}

func TestWorkspaceChainAuthorizesBeforePersistence(t *testing.T) {
	denied := &accessAuthorizerStub{err: errAccessDenied}
	projects := &projectRepositoryStub{}
	actors := &actorReaderStub{belongs: true}
	newID := func(context.Context) (string, error) { return "id-1", nil }

	todos := &todoRepositoryCounter{}
	issues := &issueRepositoryCounter{}
	todoService, err := NewTodoUseCase(todos, projects, issues, denied, actors, newID, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := todoService.CreateTodo(context.Background(), contract.CreateTodoRequest{WorkspaceId: "workspace-1", Title: "Todo"}); err == nil {
		t.Fatal("CreateTodo() error = nil")
	}
	if todos.calls != 0 || projects.findCalls != 0 || issues.calls != 0 || actors.calls != 0 {
		t.Fatalf("Todo calls = repo:%d project:%d issue:%d actor:%d", todos.calls, projects.findCalls, issues.calls, actors.calls)
	}

	issueService, err := NewIssueUseCase(issues, projects, denied, actors, &assetReaderStub{}, newID, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issueService.UpdateIssueStatus(context.Background(), contract.UpdateIssueStatusRequest{WorkspaceId: "workspace-1", IssueId: "issue-1", Status: "done"}); err == nil {
		t.Fatal("UpdateIssueStatus() error = nil")
	}
	if issues.calls != 0 {
		t.Fatalf("Issue repository calls = %d", issues.calls)
	}

	knowledge := &knowledgeRepositoryCounter{}
	knowledgeService, err := NewKnowledgeUseCase(knowledge, denied, &assetReaderStub{}, newID, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := knowledgeService.CreateKnowledge(context.Background(), contract.CreateKnowledgeRequest{WorkspaceId: "workspace-1", Title: "Knowledge"}); err == nil {
		t.Fatal("CreateKnowledge() error = nil")
	}
	if knowledge.calls != 0 {
		t.Fatalf("Knowledge repository calls = %d", knowledge.calls)
	}

	requirements := &requirementRepositoryCounter{}
	requirementService, err := NewRequirementUseCase(requirements, projects, issues, denied, newID, newID, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requirementService.SaveRequirementVersion(context.Background(), contract.SaveRequirementVersionRequest{WorkspaceId: "workspace-1", ProjectId: "project-1", Title: "Requirement", Content: "content"}); err == nil {
		t.Fatal("SaveRequirementVersion() error = nil")
	}
	if requirements.calls != 0 || projects.findCalls != 0 || issues.calls != 0 {
		t.Fatalf("Requirement calls = repo:%d project:%d issue:%d", requirements.calls, projects.findCalls, issues.calls)
	}

	settings := &settingRepositoryCounter{}
	settingService, err := NewSettingUseCase(settings, denied, actors, &skillReaderStub{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settingService.PutWorkspaceSetting(context.Background(), contract.PutWorkspaceSettingRequest{WorkspaceId: "workspace-1", Key: "key"}); err == nil {
		t.Fatal("PutWorkspaceSetting() error = nil")
	}
	if settings.calls != 0 {
		t.Fatalf("Setting repository calls = %d", settings.calls)
	}
}

var errAccessDenied = &accessDeniedError{}

type accessDeniedError struct{}

func (*accessDeniedError) Error() string { return "access denied" }
