package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
)

type projectRetrospectiveRepositoryStub struct {
	createCommands   []CreateProjectRetrospectiveCommand
	mutateCommands   []MutateProjectRetrospectiveCommand
	listQueries      []ProjectRetrospectiveListQuery
	prepareCommands  []PrepareProjectRetrospectiveTargetCommand
	completeCommands []CompleteProjectRetrospectiveTargetCommand
	page             ProjectRetrospectivePage
	targetClaim      ProjectRetrospectiveTargetClaim
	targetLink       contract.ProjectRetrospectiveActionLink
}

type projectRetrospectiveTaskServiceStub struct{}

func (projectRetrospectiveTaskServiceStub) CreateTodo(context.Context, contract.CreateTodoRequest) (contract.CreateTodoResponse, error) {
	return contract.CreateTodoResponse{}, nil
}

type projectRetrospectiveIssueServiceStub struct{}

func (projectRetrospectiveIssueServiceStub) CreateIssueIdempotently(context.Context, contract.IdempotentCreateIssueRequest) (contract.IdempotentCreateIssueResponse, error) {
	return contract.IdempotentCreateIssueResponse{}, nil
}

func (s *projectRetrospectiveRepositoryStub) ReadProjectRetrospective(_ context.Context, workspaceID, projectID, retrospectiveID string, _ contract.WorkspaceActor) (contract.ProjectRetrospective, error) {
	return contract.ProjectRetrospective{ID: retrospectiveID, WorkspaceID: workspaceID, ProjectID: projectID, History: []contract.ProjectRetrospectiveRevision{}, ActionLinks: []contract.ProjectRetrospectiveActionLink{}}, nil
}

func (s *projectRetrospectiveRepositoryStub) ListProjectRetrospectives(_ context.Context, query ProjectRetrospectiveListQuery) (ProjectRetrospectivePage, error) {
	s.listQueries = append(s.listQueries, query)
	return s.page, nil
}

func (s *projectRetrospectiveRepositoryStub) CreateProjectRetrospective(_ context.Context, command CreateProjectRetrospectiveCommand) (contract.ProjectRetrospective, error) {
	s.createCommands = append(s.createCommands, command)
	return contract.ProjectRetrospective{ID: command.RetrospectiveID, WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID, Status: retrospectiveDomain.StatusDraft, CurrentRevision: 1, History: []contract.ProjectRetrospectiveRevision{}, ActionLinks: []contract.ProjectRetrospectiveActionLink{}}, nil
}

func (s *projectRetrospectiveRepositoryStub) MutateProjectRetrospective(_ context.Context, command MutateProjectRetrospectiveCommand) (contract.ProjectRetrospective, error) {
	s.mutateCommands = append(s.mutateCommands, command)
	return contract.ProjectRetrospective{ID: command.RetrospectiveID, WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID, CurrentRevision: command.ExpectedRevision + 1, History: []contract.ProjectRetrospectiveRevision{}, ActionLinks: []contract.ProjectRetrospectiveActionLink{}}, nil
}

func (s *projectRetrospectiveRepositoryStub) PrepareProjectRetrospectiveTarget(_ context.Context, command PrepareProjectRetrospectiveTargetCommand) (ProjectRetrospectiveTargetClaim, error) {
	s.prepareCommands = append(s.prepareCommands, command)
	return s.targetClaim, nil
}

func (s *projectRetrospectiveRepositoryStub) CompleteProjectRetrospectiveTarget(_ context.Context, command CompleteProjectRetrospectiveTargetCommand) (contract.ProjectRetrospectiveActionLink, error) {
	s.completeCommands = append(s.completeCommands, command)
	return s.targetLink, nil
}

type projectRetrospectiveTargetTaskServiceStub struct {
	requests []contract.CreateTodoRequest
	response contract.CreateTodoResponse
	err      error
}

func (s *projectRetrospectiveTargetTaskServiceStub) CreateTodo(_ context.Context, request contract.CreateTodoRequest) (contract.CreateTodoResponse, error) {
	s.requests = append(s.requests, request)
	return s.response, s.err
}

type projectRetrospectiveTargetIssueServiceStub struct {
	requests []contract.IdempotentCreateIssueRequest
	response contract.IdempotentCreateIssueResponse
	err      error
}

func (s *projectRetrospectiveTargetIssueServiceStub) CreateIssueIdempotently(_ context.Context, request contract.IdempotentCreateIssueRequest) (contract.IdempotentCreateIssueResponse, error) {
	s.requests = append(s.requests, request)
	return s.response, s.err
}

func TestProjectRetrospectiveUseCaseCreatesServerOwnedActionIDsWithStableRequestHash(t *testing.T) {
	repository := &projectRetrospectiveRepositoryStub{}
	retrospectiveSequence, actionSequence := 0, 0
	useCase, err := NewProjectRetrospectiveUseCase(
		repository,
		projectRetrospectiveTaskServiceStub{},
		projectRetrospectiveIssueServiceStub{},
		func(context.Context) (string, error) {
			retrospectiveSequence++
			return "retro-" + string(rune('0'+retrospectiveSequence)), nil
		},
		func(context.Context) (string, error) {
			actionSequence++
			return "action-" + string(rune('0'+actionSequence)), nil
		},
		func() time.Time { return time.Date(2026, 8, 19, 19, 0, 0, 0, time.UTC) },
		[]byte("01234567890123456789012345678901"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	request := contract.CreateProjectRetrospectiveRequest{
		Content: contract.ProjectRetrospectiveContentInput{
			Summary: " Summary ", Lessons: []string{" Lesson "},
			ActionItems: []contract.ProjectRetrospectiveActionItemInput{{Title: " Follow up "}},
		},
		Participants: []contract.ProjectRetrospectiveParticipantInput{{MemberID: "member-1", Role: retrospectiveDomain.RoleParticipant}},
	}
	for range 2 {
		if _, err = useCase.CreateProjectRetrospective(ctx, " workspace-1 ", " project-1 ", " create-key ", request); err != nil {
			t.Fatal(err)
		}
	}
	if len(repository.createCommands) != 2 {
		t.Fatalf("create command count = %d", len(repository.createCommands))
	}
	first, second := repository.createCommands[0], repository.createCommands[1]
	if first.RetrospectiveID != "retro-1" || len(first.Content.ActionItems) != 1 || first.Content.ActionItems[0].ID != "action-1" {
		t.Fatalf("first command = %#v", first)
	}
	if second.RetrospectiveID != "retro-2" || second.Content.ActionItems[0].ID != "action-2" {
		t.Fatalf("second command = %#v", second)
	}
	if len(first.RequestHash) != 64 || first.RequestHash != second.RequestHash {
		t.Fatalf("request hashes = %q / %q", first.RequestHash, second.RequestHash)
	}
	if first.Actor != (contract.WorkspaceActor{Type: "member", ID: "member-1"}) || first.WorkspaceID != "workspace-1" || first.ProjectID != "project-1" {
		t.Fatalf("trusted command = %#v", first)
	}
	request.Content.ActionItems[0].ID = "client-owned"
	if _, err = useCase.CreateProjectRetrospective(ctx, "workspace-1", "project-1", "create-key", request); !errors.Is(err, ErrInvalidProjectRetrospectiveRequest) {
		t.Fatalf("client action ID error = %v", err)
	}
}

func TestProjectRetrospectiveUseCaseMapsCompleteRevisionAndArchiveCommands(t *testing.T) {
	repository := &projectRetrospectiveRepositoryStub{}
	useCase, err := NewProjectRetrospectiveUseCase(
		repository,
		projectRetrospectiveTaskServiceStub{},
		projectRetrospectiveIssueServiceStub{},
		func(context.Context) (string, error) { return "retro", nil },
		func(context.Context) (string, error) { return "action", nil },
		func() time.Time { return time.Date(2026, 8, 19, 19, 30, 0, 0, time.UTC) },
		[]byte("01234567890123456789012345678901"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "lead-1")
	content := contract.ProjectRetrospectiveContentInput{
		Summary: "Revised", Lessons: []string{"Learned"},
		ActionItems: []contract.ProjectRetrospectiveActionItemInput{{ID: "action-1", Title: "Keep"}},
	}
	participants := []contract.ProjectRetrospectiveParticipantInput{{MemberID: "lead-member", Role: retrospectiveDomain.RoleParticipant}}
	if _, err = useCase.UpdateProjectRetrospective(ctx, "workspace-1", "project-1", "retro-1", contract.UpdateProjectRetrospectiveRequest{
		ExpectedRevision: 2, Action: retrospectiveDomain.ActionPublishRevision, Content: &content, Participants: &participants,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = useCase.ArchiveProjectRetrospective(ctx, "workspace-1", "project-1", "retro-1", 3); err != nil {
		t.Fatal(err)
	}
	if len(repository.mutateCommands) != 2 {
		t.Fatalf("mutation count = %d", len(repository.mutateCommands))
	}
	update, archive := repository.mutateCommands[0], repository.mutateCommands[1]
	if update.Action != retrospectiveDomain.ActionPublishRevision || update.Content == nil || update.Content.ActionItems[0].ID != "action-1" || update.Participants == nil {
		t.Fatalf("update command = %#v", update)
	}
	if archive.Action != retrospectiveDomain.ActionArchive || archive.ExpectedRevision != 3 || archive.Content != nil || archive.Participants != nil {
		t.Fatalf("archive command = %#v", archive)
	}
	if update.RequestID == "" || archive.RequestID == "" || update.RequestID == archive.RequestID {
		t.Fatalf("request IDs = %q / %q", update.RequestID, archive.RequestID)
	}
}

func TestProjectRetrospectiveUseCaseSignsWorkspaceScopedStableCursor(t *testing.T) {
	repository := &projectRetrospectiveRepositoryStub{page: ProjectRetrospectivePage{
		Retrospectives: []contract.ProjectRetrospective{
			{ID: "retro-2", UpdatedAt: "2026-08-19T20:00:00Z"},
			{ID: "retro-1", UpdatedAt: "2026-08-19T19:00:00Z"},
		},
		HasMore: true,
	}}
	useCase, err := NewProjectRetrospectiveUseCase(
		repository,
		projectRetrospectiveTaskServiceStub{},
		projectRetrospectiveIssueServiceStub{},
		func(context.Context) (string, error) { return "retro", nil },
		func(context.Context) (string, error) { return "action", nil },
		time.Now,
		[]byte("01234567890123456789012345678901"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	first, err := useCase.ListProjectRetrospectives(ctx, "workspace-1", "project-1", 2, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Retrospectives) != 2 || first.NextCursor == "" || strings.Contains(first.NextCursor, "retro-1") {
		t.Fatalf("first page = %#v", first)
	}
	if _, err = useCase.ListProjectRetrospectives(ctx, "workspace-1", "project-1", 2, first.NextCursor, false); err != nil {
		t.Fatal(err)
	}
	if len(repository.listQueries) != 2 || repository.listQueries[1].Cursor == nil || repository.listQueries[1].Cursor.UpdatedAt != "2026-08-19T19:00:00Z" || repository.listQueries[1].Cursor.ID != "retro-1" {
		t.Fatalf("decoded cursor query = %#v", repository.listQueries)
	}
	if _, err = useCase.ListProjectRetrospectives(ctx, "workspace-2", "project-1", 2, first.NextCursor, false); !errors.Is(err, ErrInvalidProjectRetrospectiveRequest) {
		t.Fatalf("foreign Workspace cursor error = %v", err)
	}
	if _, err = useCase.ListProjectRetrospectives(ctx, "workspace-1", "project-1", 2, first.NextCursor+"x", false); !errors.Is(err, ErrInvalidProjectRetrospectiveRequest) {
		t.Fatalf("tampered cursor error = %v", err)
	}
}

func TestProjectRetrospectiveUseCaseCreatesDefaultTaskTargetThroughGovernedClaim(t *testing.T) {
	projectID, assigneeType, assigneeID, dueDate := "project-1", "member", "member-2", "2026-08-30"
	repository := &projectRetrospectiveRepositoryStub{
		targetClaim: ProjectRetrospectiveTargetClaim{
			ActionItem:     contract.ProjectRetrospectiveActionItem{ID: "action-1", Title: "Follow up", Description: "Close the loop", AssigneeID: assigneeID, DueDate: dueDate},
			SourceRevision: 4, TargetKind: "task", ChildIdempotencyKey: "retrospective-target-child",
		},
		targetLink: contract.ProjectRetrospectiveActionLink{ActionItemID: "action-1", SourceRevision: 4, State: "linked", TargetKind: "task", TargetID: "task-1"},
	}
	tasks := &projectRetrospectiveTargetTaskServiceStub{response: contract.CreateTodoResponse{Todo: &contract.Todo{Id: "task-1"}}}
	issues := &projectRetrospectiveTargetIssueServiceStub{}
	useCase, err := NewProjectRetrospectiveUseCase(
		repository, tasks, issues,
		func(context.Context) (string, error) { return "retro", nil },
		func(context.Context) (string, error) { return "action", nil },
		func() time.Time { return time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC) },
		[]byte("01234567890123456789012345678901"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "lead-1")
	link, err := useCase.CreateProjectRetrospectiveTarget(ctx, " workspace-1 ", " project-1 ", " retro-1 ", " action-1 ", " target-key ", contract.CreateProjectRetrospectiveTargetRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if link.TargetID != "task-1" || len(repository.prepareCommands) != 1 || len(repository.completeCommands) != 1 {
		t.Fatalf("target result = %#v, prepare = %#v, complete = %#v", link, repository.prepareCommands, repository.completeCommands)
	}
	prepare, complete := repository.prepareCommands[0], repository.completeCommands[0]
	if prepare.TargetKind != "task" || prepare.IdempotencyKey != "target-key" || len(prepare.RequestHash) != 64 || prepare.Actor.ID != "lead-1" {
		t.Fatalf("prepare command = %#v", prepare)
	}
	if complete.TargetID != "task-1" || complete.RequestHash != prepare.RequestHash || complete.IdempotencyKey != "target-key" {
		t.Fatalf("complete command = %#v", complete)
	}
	if len(tasks.requests) != 1 || len(issues.requests) != 0 {
		t.Fatalf("target calls = tasks %#v, issues %#v", tasks.requests, issues.requests)
	}
	task := tasks.requests[0]
	if task.IdempotencyKey != "retrospective-target-child" || task.WorkspaceId != "workspace-1" || task.Title != "Follow up" || task.Description != "Close the loop" || task.ProjectId == nil || *task.ProjectId != projectID || task.AssigneeType == nil || *task.AssigneeType != assigneeType || task.AssigneeId == nil || *task.AssigneeId != assigneeID || task.DueDate == nil || *task.DueDate != dueDate || task.Status != "todo" || task.Priority != "none" {
		t.Fatalf("task request = %#v", task)
	}
}

func TestProjectRetrospectiveUseCaseDistinguishesOmittedExplicitAndInvalidTargetKinds(t *testing.T) {
	issueKind, blankKind, unknownKind := " issue ", " ", "epic"
	repository := &projectRetrospectiveRepositoryStub{
		targetClaim: ProjectRetrospectiveTargetClaim{
			ActionItem:     contract.ProjectRetrospectiveActionItem{ID: "action-1", Title: "Follow up"},
			SourceRevision: 2, TargetKind: "issue", ChildIdempotencyKey: "issue-child",
		},
		targetLink: contract.ProjectRetrospectiveActionLink{ActionItemID: "action-1", SourceRevision: 2, State: "linked", TargetKind: "issue", TargetID: "issue-1"},
	}
	tasks := &projectRetrospectiveTargetTaskServiceStub{}
	issues := &projectRetrospectiveTargetIssueServiceStub{response: contract.IdempotentCreateIssueResponse{Issue: &contract.Issue{Id: "issue-1"}}}
	useCase, err := NewProjectRetrospectiveUseCase(repository, tasks, issues,
		func(context.Context) (string, error) { return "retro", nil }, func(context.Context) (string, error) { return "action", nil }, time.Now,
		[]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "lead-1")
	if _, err = useCase.CreateProjectRetrospectiveTarget(ctx, "workspace-1", "project-1", "retro-1", "action-1", "issue-key", contract.CreateProjectRetrospectiveTargetRequest{TargetKind: &issueKind}); err != nil {
		t.Fatal(err)
	}
	if len(issues.requests) != 1 || len(tasks.requests) != 0 || issues.requests[0].IdempotencyKey != "issue-child" || issues.requests[0].RequestHash == "" || issues.requests[0].CreateIssueRequest.Status != "todo" || issues.requests[0].CreateIssueRequest.Priority != "none" {
		t.Fatalf("issue requests = %#v", issues.requests)
	}
	for _, kind := range []*string{&blankKind, &unknownKind} {
		if _, err = useCase.CreateProjectRetrospectiveTarget(ctx, "workspace-1", "project-1", "retro-1", "action-1", "invalid-key", contract.CreateProjectRetrospectiveTargetRequest{TargetKind: kind}); !errors.Is(err, ErrInvalidProjectRetrospectiveRequest) {
			t.Fatalf("target kind %q error = %v", *kind, err)
		}
	}
}

func TestProjectRetrospectiveUseCaseLeavesClaimPendingOnTargetFailureAndResumes(t *testing.T) {
	targetFailure := errors.New("target denied")
	repository := &projectRetrospectiveRepositoryStub{
		targetClaim: ProjectRetrospectiveTargetClaim{
			ActionItem:     contract.ProjectRetrospectiveActionItem{ID: "action-1", Title: "Follow up"},
			SourceRevision: 2, TargetKind: "task", ChildIdempotencyKey: "child-key",
		},
		targetLink: contract.ProjectRetrospectiveActionLink{RetrospectiveID: "retro-1", ActionItemID: "action-1", SourceRevision: 2, State: "linked", TargetKind: "task", TargetID: "task-1"},
	}
	tasks := &projectRetrospectiveTargetTaskServiceStub{err: targetFailure}
	useCase, err := NewProjectRetrospectiveUseCase(repository, tasks, &projectRetrospectiveTargetIssueServiceStub{},
		func(context.Context) (string, error) { return "retro", nil }, func(context.Context) (string, error) { return "action", nil }, time.Now,
		[]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "lead-1")
	request := contract.CreateProjectRetrospectiveTargetRequest{}
	if _, err = useCase.CreateProjectRetrospectiveTarget(ctx, "workspace-1", "project-1", "retro-1", "action-1", "target-key", request); !errors.Is(err, targetFailure) {
		t.Fatalf("target failure = %v", err)
	}
	if len(repository.completeCommands) != 0 {
		t.Fatalf("completion recorded after target failure: %#v", repository.completeCommands)
	}
	tasks.err = nil
	tasks.response = contract.CreateTodoResponse{Todo: &contract.Todo{Id: "task-1"}}
	link, err := useCase.CreateProjectRetrospectiveTarget(ctx, "workspace-1", "project-1", "retro-1", "action-1", "target-key", request)
	if err != nil || link.TargetID != "task-1" || len(repository.prepareCommands) != 2 || len(repository.completeCommands) != 1 {
		t.Fatalf("resumed link = %#v, error %v, prepare %#v complete %#v", link, err, repository.prepareCommands, repository.completeCommands)
	}
	repository.targetClaim.TargetID = "task-1"
	tasks.requests = nil
	if _, err = useCase.CreateProjectRetrospectiveTarget(ctx, "workspace-1", "project-1", "retro-1", "action-1", "second-key", request); err != nil {
		t.Fatal(err)
	}
	if len(tasks.requests) != 0 {
		t.Fatalf("linked claim recreated Task: %#v", tasks.requests)
	}
}
