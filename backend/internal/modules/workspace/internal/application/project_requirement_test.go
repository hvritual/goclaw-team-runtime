package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

func TestProjectRequirementUseCaseBuildsCanonicalCreateAndOutlineCommands(t *testing.T) {
	repository := &projectRequirementRepositoryStub{}
	now := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC)
	baselineIDs := 0
	outlineIDs := 0
	useCase, err := NewProjectRequirementUseCase(
		repository,
		func(context.Context) (string, error) { baselineIDs++; return "baseline-1", nil },
		func(context.Context) (string, error) { outlineIDs++; return "outline-1", nil },
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	request := contract.SaveProjectRequirementDraftRequest{
		ExpectedRevision: 0,
		Content: contract.ProjectRequirementContent{
			ProblemStatement: "  Govern delivery  ",
			Goals:            []contract.ProjectRequirementItem{{Key: "goal-1", Text: "Ship"}},
		},
		ChangeSummary: "Initial baseline",
	}
	if _, err = useCase.SaveProjectRequirement(ctx, " workspace-1 ", " project-1 ", "create-key", request); err != nil {
		t.Fatal(err)
	}
	if baselineIDs != 1 || repository.saved.BaselineID != "baseline-1" || repository.saved.WorkspaceID != "workspace-1" || repository.saved.ProjectID != "project-1" {
		t.Fatalf("save command = %#v, baseline IDs = %d", repository.saved, baselineIDs)
	}
	if repository.saved.Actor.Type != "member" || repository.saved.Actor.ID != "member-1" || repository.saved.IdempotencyKey != "create-key" || len(repository.saved.RequestHash) != 64 {
		t.Fatalf("save authority/replay = %#v", repository.saved)
	}
	firstHash := repository.saved.RequestHash
	if _, err = useCase.SaveProjectRequirement(ctx, "workspace-1", "project-1", "create-key", request); err != nil {
		t.Fatal(err)
	}
	if repository.saved.RequestHash != firstHash || baselineIDs != 2 {
		t.Fatalf("retry hash = %q want %q, generated IDs = %d", repository.saved.RequestHash, firstHash, baselineIDs)
	}

	outlineRequest := contract.CreateProjectOutlineNodeRequest{ExpectedRevision: 0, Title: " Root scope "}
	if _, err = useCase.CreateProjectOutlineNode(ctx, "workspace-1", "project-1", "outline-key", outlineRequest); err != nil {
		t.Fatal(err)
	}
	if outlineIDs != 1 || repository.outlineCreated.NodeID != "outline-1" || repository.outlineCreated.Title != "Root scope" || len(repository.outlineCreated.RequestHash) != 64 {
		t.Fatalf("outline command = %#v, outline IDs = %d", repository.outlineCreated, outlineIDs)
	}
}

func TestProjectRequirementUseCaseRequiresTrustedMemberActorAndMapsTransition(t *testing.T) {
	repository := &projectRequirementRepositoryStub{}
	useCase, err := NewProjectRequirementUseCase(
		repository,
		func(context.Context) (string, error) { return "baseline-1", nil },
		func(context.Context) (string, error) { return "outline-1", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.SaveProjectRequirement(context.Background(), "workspace-1", "project-1", "key", contract.SaveProjectRequirementDraftRequest{
		ExpectedRevision: 0,
		Content:          contract.ProjectRequirementContent{ProblemStatement: "Body"},
		ChangeSummary:    "Initial",
	})
	if !errors.Is(err, contract.ErrWorkspaceActorRequired) {
		t.Fatalf("SaveProjectRequirement(no actor) error = %v", err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	if _, err = useCase.TransitionProjectRequirement(ctx, "workspace-1", "project-1", "submit-review", contract.ProjectRequirementTransitionRequest{ExpectedRevision: 7}); err != nil {
		t.Fatal(err)
	}
	if repository.transitioned.Action != "submit_review" || repository.transitioned.ExpectedRevision != 7 || repository.transitioned.Actor.ID != "member-1" {
		t.Fatalf("transition command = %#v", repository.transitioned)
	}
	if _, err = useCase.TransitionProjectRequirement(ctx, "workspace-1", "project-1", "unknown", contract.ProjectRequirementTransitionRequest{ExpectedRevision: 7}); !errors.Is(err, ErrInvalidProjectRequirementRequest) {
		t.Fatalf("TransitionProjectRequirement(unknown) error = %v", err)
	}
}

func TestProjectRequirementUseCaseReadsCoverageWithTrustedActor(t *testing.T) {
	status := "retired"
	repository := &projectRequirementRepositoryStub{
		coverage: contract.ProjectRequirementCoverage{BaselineStatus: &status},
	}
	useCase, err := NewProjectRequirementUseCase(
		repository,
		func(context.Context) (string, error) { return "baseline-1", nil },
		func(context.Context) (string, error) { return "outline-1", nil },
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = useCase.GetProjectRequirementCoverage(context.Background(), "workspace-1", "project-1"); !errors.Is(err, contract.ErrWorkspaceActorRequired) {
		t.Fatalf("GetProjectRequirementCoverage(no actor) error = %v", err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	coverage, err := useCase.GetProjectRequirementCoverage(ctx, " workspace-1 ", " project-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if coverage.BaselineStatus == nil || *coverage.BaselineStatus != "retired" || repository.coverageWorkspaceID != "workspace-1" || repository.coverageProjectID != "project-1" || repository.coverageActor.ID != "member-1" {
		t.Fatalf("coverage = %#v; repository read = workspace %q project %q actor %#v", coverage, repository.coverageWorkspaceID, repository.coverageProjectID, repository.coverageActor)
	}
	if _, err = useCase.GetProjectRequirementCoverage(ctx, "", "project-1"); !errors.Is(err, ErrInvalidProjectRequirementRequest) {
		t.Fatalf("GetProjectRequirementCoverage(blank workspace) error = %v", err)
	}
}

type projectRequirementRepositoryStub struct {
	saved               ProjectRequirementSave
	transitioned        ProjectRequirementTransition
	linked              ProjectRequirementLinkMutation
	accessReplaced      ProjectRequirementAccessReplace
	outlineCreated      ProjectOutlineNodeCreate
	coverage            contract.ProjectRequirementCoverage
	coverageWorkspaceID string
	coverageProjectID   string
	coverageActor       contract.WorkspaceActor
}

func (s *projectRequirementRepositoryStub) ReadProjectRequirement(context.Context, string, string, contract.WorkspaceActor) (contract.ProjectRequirementBaselineResponse, error) {
	return contract.ProjectRequirementBaselineResponse{}, nil
}

func (s *projectRequirementRepositoryStub) ReadProjectRequirementCoverage(_ context.Context, workspaceID, projectID string, actor contract.WorkspaceActor) (contract.ProjectRequirementCoverage, error) {
	s.coverageWorkspaceID, s.coverageProjectID, s.coverageActor = workspaceID, projectID, actor
	return s.coverage, nil
}

func (s *projectRequirementRepositoryStub) SaveProjectRequirement(_ context.Context, command ProjectRequirementSave) (contract.ProjectRequirementBaselineResponse, error) {
	s.saved = command
	return contract.ProjectRequirementBaselineResponse{}, nil
}

func (s *projectRequirementRepositoryStub) TransitionProjectRequirement(_ context.Context, command ProjectRequirementTransition) (contract.ProjectRequirementBaselineResponse, error) {
	s.transitioned = command
	return contract.ProjectRequirementBaselineResponse{}, nil
}

func (s *projectRequirementRepositoryStub) MutateProjectRequirementLink(_ context.Context, command ProjectRequirementLinkMutation) (contract.ProjectRequirementBaselineResponse, error) {
	s.linked = command
	return contract.ProjectRequirementBaselineResponse{}, nil
}

func (s *projectRequirementRepositoryStub) ReplaceProjectRequirementAccess(_ context.Context, command ProjectRequirementAccessReplace) (contract.ProjectRequirementAccessSet, error) {
	s.accessReplaced = command
	return contract.ProjectRequirementAccessSet{}, nil
}

func (s *projectRequirementRepositoryStub) ReadProjectRequirementAccess(context.Context, string, string, contract.WorkspaceActor) (contract.ProjectRequirementAccessSet, error) {
	return contract.ProjectRequirementAccessSet{}, nil
}

func (s *projectRequirementRepositoryStub) CreateProjectOutlineNode(_ context.Context, command ProjectOutlineNodeCreate) (contract.ProjectOutline, error) {
	s.outlineCreated = command
	return contract.ProjectOutline{}, nil
}

func (s *projectRequirementRepositoryStub) ReadProjectOutline(context.Context, string, string, contract.WorkspaceActor) (contract.ProjectOutline, error) {
	return contract.ProjectOutline{}, nil
}

func TestProjectRequirementRequestHashDoesNotContainRawContent(t *testing.T) {
	hash, err := projectRequirementRequestHash(struct{ Secret string }{Secret: strings.Repeat("sensitive", 20)})
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) != 64 || strings.Contains(hash, "sensitive") {
		t.Fatalf("request hash = %q", hash)
	}
}
