package teamcontrol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHierarchicalPolicyResolutionAndStableHash(t *testing.T) {
	fixture := newTestFixture(t)
	repository, err := fixture.service.CreateRepository(
		fixture.alice.ID,
		CreateRepositoryInput{
			ID: "repo-policy", ProjectID: fixture.projectA.ID, Name: "Policy Repo",
			RemoteURL: "https://example.com/policy.git",
		},
	)
	require.NoError(t, err)
	component, err := fixture.service.RegisterComponent(
		fixture.alice.ID,
		RegisterComponentInput{
			ID: "component-policy", ProjectID: fixture.projectA.ID,
			RepositoryID: repository.ID, Name: "Policy Component", Kind: ComponentService,
		},
	)
	require.NoError(t, err)

	teamPolicy, err := fixture.service.PutPolicyBundle(
		fixture.alice.ID,
		PutPolicyBundleInput{
			ID: "policy-team-v1", Name: "engineering", Scope: PolicyTeam,
			ScopeID: fixture.team.ID, Version: 1, Enabled: true,
			Rules: map[string]json.RawMessage{
				"code_style": json.RawMessage(`"gofmt"`),
				"max_files":  json.RawMessage(`40`),
			},
		},
	)
	require.NoError(t, err)
	_, err = fixture.service.PutPolicyBundle(
		fixture.alice.ID,
		PutPolicyBundleInput{
			ID: "policy-project-v1", Name: "delivery", Scope: PolicyProject,
			ScopeID: fixture.projectA.ID, Version: 1, Enabled: true,
			Rules: map[string]json.RawMessage{"max_files": json.RawMessage(`20`)},
		},
	)
	require.NoError(t, err)
	projectV2, err := fixture.service.PutPolicyBundle(
		fixture.alice.ID,
		PutPolicyBundleInput{
			ID: "policy-project-v2", Name: "delivery", Scope: PolicyProject,
			ScopeID: fixture.projectA.ID, Version: 2, Enabled: true,
			Rules: map[string]json.RawMessage{"max_files": json.RawMessage(`10`)},
		},
	)
	require.NoError(t, err)
	_, err = fixture.service.PutPolicyBundle(
		fixture.alice.ID,
		PutPolicyBundleInput{
			ID: "policy-repo", Name: "repository", Scope: PolicyRepository,
			ScopeID: repository.ID, Version: 1, Enabled: true,
			Rules: map[string]json.RawMessage{"code_style": json.RawMessage(`"strict"`)},
		},
	)
	require.NoError(t, err)
	_, err = fixture.service.PutPolicyBundle(
		fixture.alice.ID,
		PutPolicyBundleInput{
			ID: "policy-component", Name: "component", Scope: PolicyComponent,
			ScopeID: component.ID, Version: 1, Enabled: true,
			Rules: map[string]json.RawMessage{"require_race_test": json.RawMessage(`true`)},
		},
	)
	require.NoError(t, err)

	resolved, err := fixture.service.ResolvePolicy(
		fixture.bob.ID,
		fixture.projectA.ID,
		repository.ID,
		component.ID,
	)
	require.NoError(t, err)
	require.JSONEq(t, `"strict"`, string(resolved.Rules["code_style"]))
	require.JSONEq(t, `10`, string(resolved.Rules["max_files"]))
	require.JSONEq(t, `true`, string(resolved.Rules["require_race_test"]))
	require.Len(t, resolved.Hash, 64)
	require.Contains(t, resolved.BundleIDs, teamPolicy.ID)
	require.Contains(t, resolved.BundleIDs, projectV2.ID)
	require.NotContains(t, resolved.BundleIDs, "policy-project-v1")

	again, err := fixture.service.ResolvePolicy(
		fixture.bob.ID,
		fixture.projectA.ID,
		repository.ID,
		component.ID,
	)
	require.NoError(t, err)
	require.Equal(t, resolved.Hash, again.Hash)
	require.Equal(t, resolved.BundleIDs, again.BundleIDs)

	_, err = fixture.service.PutPolicyBundle(
		fixture.alice.ID,
		PutPolicyBundleInput{
			ID: "bad-policy", Name: "bad", Scope: PolicyProject,
			ScopeID: fixture.projectA.ID, Version: 1, Enabled: true,
			Rules: map[string]json.RawMessage{"bad": json.RawMessage(`true false`)},
		},
	)
	require.Error(t, err)
}

func TestCorrelationRejectsWrongArtifactTypeAndCrossProject(t *testing.T) {
	fixture := newTestFixture(t)
	issueA, err := fixture.service.CreateIssue(fixture.bob.ID, CreateIssueInput{
		ID: "issue-link-a", ProjectID: fixture.projectA.ID, Type: IssueBug,
		Title: "A", Severity: SeverityLow,
	})
	require.NoError(t, err)
	issueB, err := fixture.service.CreateIssue(fixture.alice.ID, CreateIssueInput{
		ID: "issue-link-b", ProjectID: fixture.projectB.ID, Type: IssueBug,
		Title: "B", Severity: SeverityLow,
	})
	require.NoError(t, err)
	commit, err := fixture.service.RegisterArtifact(
		fixture.bob.ID,
		RegisterArtifactInput{
			ID: "typed-commit", ProjectID: fixture.projectA.ID,
			ResourceType: ResourceCommit, Kind: ArtifactCommit,
			Name: "Commit", URI: "git://a/commit/abcdef0",
		},
	)
	require.NoError(t, err)
	_, err = fixture.service.CreateLink(fixture.bob.ID, CreateLinkInput{
		ProjectID: fixture.projectA.ID, SourceType: ResourceTrace,
		SourceID: commit.ID, TargetType: ResourceIssue,
		TargetID: issueA.ID, Relation: "supports",
	})
	require.ErrorIs(t, err, ErrNotFound)
	_, err = fixture.service.CreateLink(fixture.bob.ID, CreateLinkInput{
		ProjectID: fixture.projectA.ID, SourceType: ResourceCommit,
		SourceID: commit.ID, TargetType: ResourceIssue,
		TargetID: issueB.ID, Relation: "fixes",
	})
	require.ErrorIs(t, err, ErrForbidden)
}
