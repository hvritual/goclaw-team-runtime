package teamcontrol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenBudgetUsageIsIdempotentBoundedAndConcurrent(t *testing.T) {
	fixture := newTestFixture(t)
	budget, err := fixture.service.PutTokenBudget(fixture.alice.ID, PutTokenBudgetInput{
		ID: "budget-bob", ProjectID: fixture.projectA.ID,
		UserID: fixture.bob.ID, LimitTokens: 100,
	})
	require.NoError(t, err)

	first := RecordTokenUsageInput{
		ID: "usage-00", ProjectID: fixture.projectA.ID,
		BudgetID: budget.ID, Tokens: 10, TaskID: "task-00",
		Metadata: map[string]string{"model": "workspace"},
	}
	event, err := fixture.service.RecordTokenUsage(fixture.alice.ID, first)
	require.NoError(t, err)
	replayed, err := fixture.service.RecordTokenUsage(fixture.alice.ID, first)
	require.NoError(t, err)
	require.Equal(t, event, replayed)

	conflictInput := first
	conflictInput.Tokens = 11
	_, err = fixture.service.RecordTokenUsage(fixture.alice.ID, conflictInput)
	require.ErrorIs(t, err, ErrConflict)

	const additional = 9
	var wg sync.WaitGroup
	errs := make(chan error, additional)
	for index := 1; index <= additional; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, usageErr := fixture.service.RecordTokenUsage(
				fixture.alice.ID,
				RecordTokenUsageInput{
					ID:        fmt.Sprintf("usage-%02d", index),
					ProjectID: fixture.projectA.ID,
					BudgetID:  budget.ID,
					Tokens:    10,
					TaskID:    fmt.Sprintf("task-%02d", index),
				},
			)
			errs <- usageErr
		}(index)
	}
	wg.Wait()
	close(errs)
	for usageErr := range errs {
		require.NoError(t, usageErr)
	}

	budgets, err := fixture.service.ListTokenBudgets(
		fixture.mallory.ID,
		fixture.projectA.ID,
	)
	require.NoError(t, err)
	require.Len(t, budgets, 1)
	require.Equal(t, int64(100), budgets[0].UsedTokens)

	_, err = fixture.service.RecordTokenUsage(
		fixture.alice.ID,
		RecordTokenUsageInput{
			ID: "usage-over", ProjectID: fixture.projectA.ID,
			BudgetID: budget.ID, Tokens: 1,
		},
	)
	require.ErrorIs(t, err, ErrConflict)

	_, err = fixture.service.PutTokenBudget(fixture.alice.ID, PutTokenBudgetInput{
		ID: budget.ID, ProjectID: fixture.projectA.ID,
		UserID: fixture.bob.ID, LimitTokens: 99,
	})
	require.ErrorIs(t, err, ErrConflict)

	_, err = fixture.service.PutTokenBudget(fixture.mallory.ID, PutTokenBudgetInput{
		ID: "viewer-budget", ProjectID: fixture.projectA.ID, LimitTokens: 1,
	})
	require.ErrorIs(t, err, ErrForbidden)

	_, err = fixture.service.PutTokenBudget(fixture.alice.ID, PutTokenBudgetInput{
		ID: "overflow-budget", ProjectID: fixture.projectA.ID,
		LimitTokens: MaxTokenBudget + 1,
	})
	require.ErrorContains(t, err, "exceeds")
}

func TestRegistryAndContextCompilerAreProjectScopedAndDeterministic(t *testing.T) {
	fixture := newTestFixture(t)
	repository, err := fixture.service.CreateRepository(
		fixture.alice.ID,
		CreateRepositoryInput{
			ID: "repo-context", ProjectID: fixture.projectA.ID,
			Name: "Context Repo", DefaultBranch: "main",
			RemoteURL: "https://example.invalid/context.git",
		},
	)
	require.NoError(t, err)
	checksumA := strings.Repeat("a", 64)
	checksumB := strings.Repeat("b", 64)
	checksumC := strings.Repeat("c", 64)

	budget, err := fixture.service.PutTokenBudget(
		fixture.alice.ID,
		PutTokenBudgetInput{
			ID: "budget-context", ProjectID: fixture.projectA.ID,
			UserID: fixture.bob.ID, LimitTokens: 1000,
		},
	)
	require.NoError(t, err)
	knowledge, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "knowledge-v1", ProjectID: fixture.projectA.ID,
			Name: "Architecture", URI: "file:///vault/architecture.md",
			Revision: "git:abc1234", SHA256: checksumA,
			Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	skill, err := fixture.service.PutSkillRelease(
		fixture.alice.ID,
		PutSkillReleaseInput{
			ID: "skill-v1", ProjectID: fixture.projectA.ID,
			Name: "go-style", Version: "1.0.0",
			URI:    "git+https://example.invalid/skills/go-style",
			SHA256: checksumB, Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	release, err := fixture.service.PutRunnerRelease(
		fixture.alice.ID,
		PutRunnerReleaseInput{
			ID: "runner-linux-amd64-v1", ProjectID: fixture.projectA.ID,
			Channel: "pilot", Version: "1.0.0", OS: "linux", Arch: "amd64",
			URI: "https://example.invalid/runner.tar.gz", SHA256: checksumC,
			MinProtocol: "1", Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "linux", release.OS)

	input := CompileContextInput{
		ProjectID: fixture.projectA.ID, RepositoryID: repository.ID,
		UserID: fixture.bob.ID, BudgetID: budget.ID,
		KnowledgeIDs: []string{knowledge.ID}, SkillIDs: []string{skill.ID},
	}
	first, err := fixture.service.CompileContext(fixture.alice.ID, input)
	require.NoError(t, err)
	second, err := fixture.service.CompileContext(
		fixture.alice.ID,
		CompileContextInput{
			ProjectID: fixture.projectA.ID, RepositoryID: repository.ID,
			UserID: fixture.bob.ID, BudgetID: budget.ID,
			KnowledgeIDs: []string{knowledge.ID, knowledge.ID},
			SkillIDs:     []string{skill.ID, skill.ID},
		},
	)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.Hash, second.Hash)
	require.Equal(t, ContextCompilerVersion, first.CompilerVersion)
	require.Len(t, first.Knowledge, 1)
	require.Len(t, first.Skills, 1)

	_, err = fixture.service.RecordTokenUsage(
		fixture.alice.ID,
		RecordTokenUsageInput{
			ID: "context-usage", ProjectID: fixture.projectA.ID,
			BudgetID: budget.ID, Tokens: 10,
		},
	)
	require.NoError(t, err)
	third, err := fixture.service.CompileContext(fixture.alice.ID, input)
	require.NoError(t, err)
	require.NotEqual(t, first.Hash, third.Hash)

	draft, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "knowledge-draft", ProjectID: fixture.projectA.ID,
			Name: "Draft", URI: "file:///vault/draft.md", Revision: "1",
			SHA256: checksumA, Status: RegistryDraft,
		},
	)
	require.NoError(t, err)
	input.KnowledgeIDs = []string{draft.ID}
	_, err = fixture.service.CompileContext(fixture.alice.ID, input)
	require.ErrorIs(t, err, ErrConflict)

	crossProject, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "knowledge-project-b", ProjectID: fixture.projectB.ID,
			Name: "Other", URI: "file:///vault/other.md", Revision: "1",
			SHA256: checksumA, Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	input.KnowledgeIDs = []string{crossProject.ID}
	_, err = fixture.service.CompileContext(fixture.alice.ID, input)
	require.ErrorIs(t, err, ErrNotFound)

	input.KnowledgeIDs = []string{knowledge.ID}
	_, err = fixture.service.CompileContext(fixture.bob.ID, input)
	require.ErrorIs(t, err, ErrForbidden)

	releases, err := fixture.service.ListRunnerReleases(
		fixture.mallory.ID,
		fixture.projectA.ID,
	)
	require.NoError(t, err)
	require.Equal(t, []RunnerRelease{release}, releases)
	bundles, err := fixture.service.ListContextBundles(
		fixture.mallory.ID,
		fixture.projectA.ID,
	)
	require.NoError(t, err)
	require.Len(t, bundles, 2)
}

func TestRegistryRequiresChecksumAndImmutableIdentity(t *testing.T) {
	fixture := newTestFixture(t)
	input := PutKnowledgeSourceInput{
		ID: "knowledge", ProjectID: fixture.projectA.ID,
		Name: "Knowledge", URI: "file:///knowledge.md", Revision: "1",
		Status: RegistryApproved,
	}
	_, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.ErrorContains(t, err, "sha256 is required")

	input.SHA256 = strings.Repeat("d", 64)
	created, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.NoError(t, err)
	input.Status = RegistryDisabled
	disabled, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.NoError(t, err)
	require.Equal(t, RegistryDisabled, disabled.Status)
	require.Equal(t, created.SHA256, disabled.SHA256)

	input.Revision = "2"
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.ErrorIs(t, err, ErrConflict)
}

func TestLegacyStateWithoutControlPlaneMapsLoadsLosslessly(t *testing.T) {
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	statePath := filepath.Join(root, stateFileName)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	for _, key := range []string{
		"token_budgets", "token_usage_events", "knowledge_sources",
		"skill_releases", "runner_releases", "context_bundles",
	} {
		delete(raw, key)
	}
	legacy, err := json.Marshal(raw)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, legacy, 0o600))

	reopened, err := Open(root)
	require.NoError(t, err)
	project, err := reopened.GetProject(fixture.alice.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.projectA.ID, project.ID)
	budgets, err := reopened.ListTokenBudgets(fixture.alice.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Empty(t, budgets)
}
