package teamcontrol

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testFixture struct {
	service  *Service
	alice    User
	bob      User
	mallory  User
	team     Team
	projectA Project
	projectB Project
}

func newTestFixture(t *testing.T) testFixture {
	t.Helper()
	service, err := Open(t.TempDir())
	require.NoError(t, err)
	alice, err := service.CreateUser(CreateUserInput{
		ID: "alice", DisplayName: "Alice", Email: "alice@example.com",
	})
	require.NoError(t, err)
	bob, err := service.CreateUser(CreateUserInput{
		ID: "bob", DisplayName: "Bob", Email: "bob@example.com",
	})
	require.NoError(t, err)
	mallory, err := service.CreateUser(CreateUserInput{
		ID: "mallory", DisplayName: "Mallory", Email: "mallory@example.com",
	})
	require.NoError(t, err)
	team, err := service.CreateTeam(alice.ID, CreateTeamInput{
		ID: "team-iot", Name: "IoT",
	})
	require.NoError(t, err)
	_, err = service.AddTeamMember(alice.ID, team.ID, AddTeamMemberInput{
		UserID: bob.ID, Role: TeamRegularMember,
	})
	require.NoError(t, err)
	_, err = service.AddTeamMember(alice.ID, team.ID, AddTeamMemberInput{
		UserID: mallory.ID, Role: TeamRegularMember,
	})
	require.NoError(t, err)
	projectA, err := service.CreateProject(alice.ID, CreateProjectInput{
		ID: "project-a", TeamID: team.ID, Key: "a", Name: "Project A",
	})
	require.NoError(t, err)
	projectB, err := service.CreateProject(alice.ID, CreateProjectInput{
		ID: "project-b", TeamID: team.ID, Key: "b", Name: "Project B",
	})
	require.NoError(t, err)
	_, err = service.AddProjectMember(alice.ID, projectA.ID, AddProjectMemberInput{
		UserID:          bob.ID,
		Role:            ProjectDeveloper,
		BusinessDomains: []string{" Device ", "device", "Telemetry"},
		CapacityPoints:  40,
	})
	require.NoError(t, err)
	_, err = service.AddProjectMember(alice.ID, projectA.ID, AddProjectMemberInput{
		UserID: mallory.ID, Role: ProjectViewer,
	})
	require.NoError(t, err)
	return testFixture{
		service: service, alice: alice, bob: bob, mallory: mallory,
		team: team, projectA: projectA, projectB: projectB,
	}
}

func TestDefaultDisabledAndBootstrapUserDiscovery(t *testing.T) {
	require.False(t, DefaultConfig().Enabled)

	root := t.TempDir()
	service, err := Open(root)
	require.NoError(t, err)
	require.Equal(t, Config{Enabled: true, Root: filepath.Clean(root)}, service.Config())
	hasUsers, err := service.HasUsers()
	require.NoError(t, err)
	require.False(t, hasUsers)

	alice, err := service.CreateUser(CreateUserInput{
		ID: "alice", DisplayName: "Alice",
	})
	require.NoError(t, err)
	hasUsers, err = service.HasUsers()
	require.NoError(t, err)
	require.True(t, hasUsers)

	bob, err := service.CreateUser(CreateUserInput{
		ID: "bob", DisplayName: "Bob",
	})
	require.NoError(t, err)
	users, err := service.ListUsers(alice.ID)
	require.NoError(t, err)
	require.Equal(t, []User{alice}, users)

	team, err := service.CreateTeam(alice.ID, CreateTeamInput{
		ID: "team", Name: "Team",
	})
	require.NoError(t, err)
	_, err = service.AddTeamMember(alice.ID, team.ID, AddTeamMemberInput{
		UserID: bob.ID, Role: TeamRegularMember,
	})
	require.NoError(t, err)
	users, err = service.ListUsers(alice.ID)
	require.NoError(t, err)
	require.Equal(t, []string{alice.ID, bob.ID}, []string{users[0].ID, users[1].ID})
}

func TestProjectIsolationAndRoleAuthorization(t *testing.T) {
	fixture := newTestFixture(t)
	issueA, err := fixture.service.CreateIssue(fixture.bob.ID, CreateIssueInput{
		ID:        "issue-a",
		ProjectID: fixture.projectA.ID,
		Type:      IssueBug,
		Title:     "A scoped bug",
		Severity:  SeverityMedium,
	})
	require.NoError(t, err)
	issueB, err := fixture.service.CreateIssue(fixture.alice.ID, CreateIssueInput{
		ID:        "issue-b",
		ProjectID: fixture.projectB.ID,
		Type:      IssueBug,
		Title:     "B scoped bug",
		Severity:  SeverityHigh,
	})
	require.NoError(t, err)

	got, err := fixture.service.GetIssue(fixture.bob.ID, fixture.projectA.ID, issueA.ID)
	require.NoError(t, err)
	require.Equal(t, issueA.ID, got.ID)

	_, err = fixture.service.GetIssue(fixture.bob.ID, fixture.projectA.ID, issueB.ID)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = fixture.service.GetIssue(fixture.bob.ID, fixture.projectB.ID, issueB.ID)
	require.ErrorIs(t, err, ErrForbidden)

	_, err = fixture.service.CreateIssue(fixture.mallory.ID, CreateIssueInput{
		ProjectID: fixture.projectA.ID,
		Type:      IssueBug,
		Title:     "viewer must not write",
		Severity:  SeverityLow,
	})
	require.ErrorIs(t, err, ErrForbidden)
	require.NoError(t, fixture.service.Authorize(
		fixture.mallory.ID,
		fixture.projectA.ID,
		ActionIssueRead,
	))
	require.ErrorIs(t, fixture.service.Authorize(
		fixture.mallory.ID,
		fixture.projectA.ID,
		ActionIssueWrite,
	), ErrForbidden)
	for _, action := range []Action{
		ActionBudgetRead,
		ActionKnowledgeRead,
		ActionSkillRead,
		ActionRunnerReleaseRead,
		ActionContextRead,
	} {
		require.NoError(t, fixture.service.Authorize(
			fixture.mallory.ID,
			fixture.projectA.ID,
			action,
		))
	}
}

func TestIssueLifecycleRequiresFixEvidenceAndTracksReopen(t *testing.T) {
	fixture := newTestFixture(t)
	due := time.Now().UTC().Add(24 * time.Hour)
	issue, err := fixture.service.CreateIssue(fixture.alice.ID, CreateIssueInput{
		ID:          "issue-lifecycle",
		ProjectID:   fixture.projectA.ID,
		Type:        IssueBug,
		Title:       "Reconnect loses telemetry",
		Severity:    SeverityHigh,
		Priority:    PriorityP1,
		Module:      "device-connectivity",
		Environment: "field",
		Labels:      []string{"regression", "weak-network"},
		DueAt:       &due,
		SLAMinutes:  120,
	})
	require.NoError(t, err)
	require.NotNil(t, issue.SLADeadline)
	require.Equal(t, PriorityP1, issue.Priority)

	issue, err = fixture.service.TransitionIssue(
		fixture.alice.ID, fixture.projectA.ID, issue.ID, IssueTriaged, "",
	)
	require.NoError(t, err)
	_, err = fixture.service.Assign(fixture.alice.ID, AssignInput{
		ProjectID:  fixture.projectA.ID,
		TargetType: AssignmentIssue,
		TargetID:   issue.ID,
		UserID:     fixture.bob.ID,
		Role:       AssignmentOwner,
	})
	require.NoError(t, err)
	issue, err = fixture.service.GetIssue(fixture.bob.ID, fixture.projectA.ID, issue.ID)
	require.NoError(t, err)
	require.Equal(t, IssueAssigned, issue.Status)

	for _, status := range []IssueStatus{IssueInProgress, IssueVerifying} {
		issue, err = fixture.service.TransitionIssue(
			fixture.bob.ID, fixture.projectA.ID, issue.ID, status, "",
		)
		require.NoError(t, err)
	}
	_, err = fixture.service.TransitionIssue(
		fixture.bob.ID, fixture.projectA.ID, issue.ID, IssueResolved, "",
	)
	require.Error(t, err)

	commit, err := fixture.service.RegisterArtifact(fixture.bob.ID, RegisterArtifactInput{
		ID:           "commit-fix",
		ProjectID:    fixture.projectA.ID,
		ResourceType: ResourceCommit,
		Kind:         ArtifactCommit,
		Name:         "Fix reconnect ordering",
		URI:          "git://project-a/commit/0123456789abcdef",
		SHA256:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	require.NoError(t, err)
	_, err = fixture.service.CreateLink(fixture.bob.ID, CreateLinkInput{
		ProjectID:  fixture.projectA.ID,
		SourceType: ResourceCommit,
		SourceID:   commit.ID,
		TargetType: ResourceIssue,
		TargetID:   issue.ID,
		Relation:   "fixes",
	})
	require.NoError(t, err)
	issue, err = fixture.service.TransitionIssue(
		fixture.bob.ID, fixture.projectA.ID, issue.ID, IssueResolved, "",
	)
	require.NoError(t, err)
	issue, err = fixture.service.TransitionIssue(
		fixture.bob.ID, fixture.projectA.ID, issue.ID, IssueClosed, "",
	)
	require.NoError(t, err)
	issue, err = fixture.service.TransitionIssue(
		fixture.bob.ID, fixture.projectA.ID, issue.ID, IssueReopened, "",
	)
	require.NoError(t, err)
	require.Equal(t, 1, issue.ReopenCount)
	require.Empty(t, issue.Resolution)
}

func TestWorkItemsRegistriesAndCorrelation(t *testing.T) {
	fixture := newTestFixture(t)
	repository, err := fixture.service.CreateRepository(
		fixture.alice.ID,
		CreateRepositoryInput{
			ID: "repo-a", ProjectID: fixture.projectA.ID, Name: "platform",
			RemoteURL: "https://example.com/platform.git", DefaultBranch: "main",
		},
	)
	require.NoError(t, err)
	component, err := fixture.service.RegisterComponent(
		fixture.alice.ID,
		RegisterComponentInput{
			ID: "component-api", ProjectID: fixture.projectA.ID,
			RepositoryID: repository.ID, Name: "Device API", Kind: ComponentService,
			RootPath: "services/device-api", OwnerIDs: []string{fixture.bob.ID},
		},
	)
	require.NoError(t, err)
	document, err := fixture.service.RegisterDocument(
		fixture.alice.ID,
		RegisterDocumentInput{
			ID: "doc-prd", ProjectID: fixture.projectA.ID, Key: "device-api-prd",
			Title: "Device API PRD", Kind: DocumentPRD, Status: DocumentActive,
			URI: "obsidian://project-a/01-prd/device-api.md", Revision: "v1",
		},
	)
	require.NoError(t, err)
	issue, err := fixture.service.CreateIssue(fixture.bob.ID, CreateIssueInput{
		ID: "issue-api", ProjectID: fixture.projectA.ID, Type: IssueTask,
		Title: "Implement endpoint", Severity: SeverityMedium,
		ComponentIDs: []string{component.ID},
	})
	require.NoError(t, err)
	dueAt := time.Now().UTC().Add(48 * time.Hour)
	first, err := fixture.service.CreateWorkItem(fixture.bob.ID, CreateWorkItemInput{
		ID: "work-schema", ProjectID: fixture.projectA.ID, IssueID: issue.ID,
		Title: "Schema", Instructions: "Add the endpoint schema.",
		BusinessDomain:       " Device ",
		EstimatePoints:       8,
		DueAt:                &dueAt,
		ComponentIDs:         []string{component.ID},
		VerificationCommands: [][]string{{"go", "test", "./..."}},
	})
	require.NoError(t, err)
	require.Equal(t, "device", first.BusinessDomain)
	require.Equal(t, PriorityP2, first.Priority)
	require.Equal(t, 8, first.EstimatePoints)
	require.NotNil(t, first.DueAt)
	second, err := fixture.service.CreateWorkItem(fixture.bob.ID, CreateWorkItemInput{
		ID: "work-handler", ProjectID: fixture.projectA.ID, IssueID: issue.ID,
		Title: "Handler", Instructions: "Implement the handler.",
		DependsOn: []string{first.ID}, ComponentIDs: []string{component.ID},
	})
	require.NoError(t, err)
	require.Equal(t, WorkItemPending, second.Status)
	_, err = fixture.service.TransitionWorkItem(
		fixture.bob.ID, fixture.projectA.ID, second.ID, WorkItemReady,
	)
	require.ErrorIs(t, err, ErrConflict)
	for _, status := range []WorkItemStatus{
		WorkItemInProgress, WorkItemVerifying, WorkItemDone,
	} {
		first, err = fixture.service.TransitionWorkItem(
			fixture.bob.ID, fixture.projectA.ID, first.ID, status,
		)
		require.NoError(t, err)
	}
	second, err = fixture.service.TransitionWorkItem(
		fixture.bob.ID, fixture.projectA.ID, second.ID, WorkItemReady,
	)
	require.NoError(t, err)
	_, err = fixture.service.Assign(fixture.bob.ID, AssignInput{
		ProjectID: fixture.projectA.ID, TargetType: AssignmentWorkItem,
		TargetID: second.ID, UserID: fixture.bob.ID, Role: AssignmentOwner,
	})
	require.NoError(t, err)
	_, err = fixture.service.CreateLink(fixture.bob.ID, CreateLinkInput{
		ProjectID: fixture.projectA.ID, SourceType: ResourceWorkItem,
		SourceID: second.ID, TargetType: ResourceDocument,
		TargetID: document.ID, Relation: "implements",
	})
	require.NoError(t, err)

	documents, err := fixture.service.ListDocuments(
		fixture.bob.ID, fixture.projectA.ID,
	)
	require.NoError(t, err)
	require.Len(t, documents, 1)
	components, err := fixture.service.ListComponents(
		fixture.bob.ID, fixture.projectA.ID,
	)
	require.NoError(t, err)
	require.Len(t, components, 1)
	assignments, err := fixture.service.ListAssignments(
		fixture.bob.ID, fixture.projectA.ID, AssignmentWorkItem, second.ID,
	)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	links, err := fixture.service.ListLinks(
		fixture.bob.ID, fixture.projectA.ID, ResourceWorkItem, second.ID,
	)
	require.NoError(t, err)
	require.Len(t, links, 1)

	members, err := fixture.service.ListProjectMembers(
		fixture.alice.ID, fixture.projectA.ID,
	)
	require.NoError(t, err)
	var bobMembership ProjectMember
	for _, member := range members {
		if member.UserID == fixture.bob.ID {
			bobMembership = member
		}
	}
	require.Equal(t, []string{"device", "telemetry"}, bobMembership.BusinessDomains)
	require.Equal(t, 40, bobMembership.CapacityPoints)
}

func TestCapacityFieldsRejectOutOfRangeValues(t *testing.T) {
	fixture := newTestFixture(t)
	_, err := fixture.service.AddProjectMember(
		fixture.alice.ID,
		fixture.projectB.ID,
		AddProjectMemberInput{
			UserID: fixture.bob.ID, Role: ProjectDeveloper, CapacityPoints: 10_001,
		},
	)
	require.ErrorContains(t, err, "capacity_points")

	_, err = fixture.service.CreateWorkItem(
		fixture.alice.ID,
		CreateWorkItemInput{
			ProjectID: fixture.projectA.ID, Title: "Too large",
			Instructions: "Reject estimate.", EstimatePoints: -1,
		},
	)
	require.ErrorContains(t, err, "estimate_points")
}

func TestAtomicStorePersistsConcurrentMutations(t *testing.T) {
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	const count = 20
	var wg sync.WaitGroup
	errorsFound := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := fixture.service.RegisterArtifact(
				fixture.alice.ID,
				RegisterArtifactInput{
					ID:        fmt.Sprintf("artifact-%02d", index),
					ProjectID: fixture.projectA.ID,
					Kind:      ArtifactLog,
					Name:      fmt.Sprintf("log-%02d", index),
					URI:       fmt.Sprintf("file:///tmp/log-%02d.txt", index),
				},
			)
			errorsFound <- err
		}(i)
	}
	wg.Wait()
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}
	reopened, err := Open(root)
	require.NoError(t, err)
	artifacts, err := reopened.ListArtifacts(fixture.alice.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Len(t, artifacts, count)
}

func newTestFixtureAt(t *testing.T, root string) testFixture {
	t.Helper()
	service, err := Open(root)
	require.NoError(t, err)
	alice, err := service.CreateUser(CreateUserInput{ID: "alice", DisplayName: "Alice"})
	require.NoError(t, err)
	team, err := service.CreateTeam(alice.ID, CreateTeamInput{ID: "team", Name: "Team"})
	require.NoError(t, err)
	project, err := service.CreateProject(alice.ID, CreateProjectInput{
		ID: "project", TeamID: team.ID, Key: "project", Name: "Project",
	})
	require.NoError(t, err)
	return testFixture{
		service: service, alice: alice, team: team, projectA: project,
	}
}

func TestInvalidIssueTransitionHasTypedError(t *testing.T) {
	fixture := newTestFixture(t)
	issue, err := fixture.service.CreateIssue(fixture.bob.ID, CreateIssueInput{
		ProjectID: fixture.projectA.ID, Type: IssueBug,
		Title: "bad transition", Severity: SeverityLow,
	})
	require.NoError(t, err)
	_, err = fixture.service.TransitionIssue(
		fixture.bob.ID, fixture.projectA.ID, issue.ID, IssueClosed, "not allowed",
	)
	require.True(t, errors.Is(err, ErrInvalidTransition))
}
