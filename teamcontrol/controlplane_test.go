package teamcontrol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	beforeReplay, err := os.ReadFile(fixture.service.store.path)
	require.NoError(t, err)
	beforeRevision := fixture.service.store.state.Revision
	replayed, err := fixture.service.RecordTokenUsage(fixture.alice.ID, first)
	require.NoError(t, err)
	require.Equal(t, event, replayed)
	afterReplay, err := os.ReadFile(fixture.service.store.path)
	require.NoError(t, err)
	require.Equal(t, beforeReplay, afterReplay)
	require.Equal(t, beforeRevision, fixture.service.store.state.Revision)
	_, err = fixture.service.RecordTokenUsage(
		fixture.alice.ID,
		RecordTokenUsageInput{
			ID: "usage-canonical-collision", ProjectID: fixture.projectA.ID,
			BudgetID: budget.ID, Tokens: 1,
			Metadata: map[string]string{"model": "codex", "MODEL": "other"},
		},
	)
	require.ErrorContains(t, err, "duplicates canonical key")

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

func TestControlResourcesUseProjectCompositeIdentity(t *testing.T) {
	fixture := newTestFixture(t)
	checksum := strings.Repeat("a", 64)
	for _, project := range []Project{fixture.projectA, fixture.projectB} {
		budget, err := fixture.service.PutTokenBudget(
			fixture.alice.ID,
			PutTokenBudgetInput{
				ID: "shared-budget", ProjectID: project.ID, LimitTokens: 10,
			},
		)
		require.NoError(t, err)
		require.Equal(t, project.ID, budget.ProjectID)
		usage, err := fixture.service.RecordTokenUsage(
			fixture.alice.ID,
			RecordTokenUsageInput{
				ID: "shared-usage", ProjectID: project.ID,
				BudgetID: budget.ID, Tokens: 1,
			},
		)
		require.NoError(t, err)
		require.Equal(t, project.ID, usage.ProjectID)
		knowledge, err := fixture.service.PutKnowledgeSource(
			fixture.alice.ID,
			PutKnowledgeSourceInput{
				ID: "shared-source", ProjectID: project.ID, Name: project.Name,
				URI: "file:///vault/shared.md", Revision: "1", SHA256: checksum,
			},
		)
		require.NoError(t, err)
		require.Equal(t, project.ID, knowledge.ProjectID)
		skill, err := fixture.service.PutSkillRelease(
			fixture.alice.ID,
			PutSkillReleaseInput{
				ID: "shared-skill", ProjectID: project.ID, Name: project.Name,
				Version: "1", URI: "file:///skills/shared", SHA256: checksum,
			},
		)
		require.NoError(t, err)
		require.Equal(t, project.ID, skill.ProjectID)
		release, err := fixture.service.PutRunnerRelease(
			fixture.alice.ID,
			PutRunnerReleaseInput{
				ID: "shared-runner", ProjectID: project.ID,
				Channel: "pilot", Version: "1", OS: "linux", Arch: "amd64",
				URI: "https://example.invalid/runner.tar.gz", SHA256: checksum,
				MinProtocol: "1",
			},
		)
		require.NoError(t, err)
		require.Equal(t, project.ID, release.ProjectID)
	}

	for _, project := range []Project{fixture.projectA, fixture.projectB} {
		budgets, err := fixture.service.ListTokenBudgets(fixture.alice.ID, project.ID)
		require.NoError(t, err)
		require.Len(t, budgets, 1)
		require.Equal(t, project.ID, budgets[0].ProjectID)
		usage, err := fixture.service.ListTokenUsage(
			fixture.alice.ID, project.ID, "shared-budget",
		)
		require.NoError(t, err)
		require.Len(t, usage, 1)
		require.Equal(t, project.ID, usage[0].ProjectID)
		source, err := fixture.service.GetKnowledgeSource(
			fixture.alice.ID, project.ID, "shared-source",
		)
		require.NoError(t, err)
		require.Equal(t, project.Name, source.Name)
		skill, err := fixture.service.GetSkillRelease(
			fixture.alice.ID, project.ID, "shared-skill",
		)
		require.NoError(t, err)
		require.Equal(t, project.Name, skill.Name)
		release, err := fixture.service.GetRunnerRelease(
			fixture.alice.ID, project.ID, "shared-runner",
		)
		require.NoError(t, err)
		require.Equal(t, project.ID, release.ProjectID)
	}
	onlyB, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "only-project-b", ProjectID: fixture.projectB.ID,
			Name: "Only B", URI: "file:///vault/only-b.md", Revision: "1",
			SHA256: checksum, Status: RegistryDisabled,
		},
	)
	require.NoError(t, err)
	_, err = fixture.service.GetKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, onlyB.ID,
	)
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorIs(t, fixture.service.DeleteKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, onlyB.ID,
	), ErrNotFound)
	_, err = fixture.service.GetKnowledgeSource(
		fixture.alice.ID, fixture.projectB.ID, onlyB.ID,
	)
	require.NoError(t, err)
}

func TestProjectBudgetTotalIsJavaScriptSafe(t *testing.T) {
	fixture := newTestFixture(t)
	_, err := fixture.service.PutTokenBudget(
		fixture.alice.ID,
		PutTokenBudgetInput{
			ID: "safe-a", ProjectID: fixture.projectA.ID,
			LimitTokens: MaxTokenBudget,
		},
	)
	require.NoError(t, err)
	for index := 1; index <= 9; index++ {
		limit := MaxTokenBudget
		if index == 9 {
			limit = MaxProjectTokenTotal - 9*MaxTokenBudget
		}
		_, err = fixture.service.PutTokenBudget(
			fixture.alice.ID,
			PutTokenBudgetInput{
				ID:        fmt.Sprintf("safe-%d", index),
				ProjectID: fixture.projectA.ID, LimitTokens: limit,
			},
		)
		require.NoError(t, err)
	}
	_, err = fixture.service.PutTokenBudget(
		fixture.alice.ID,
		PutTokenBudgetInput{
			ID: "unsafe", ProjectID: fixture.projectA.ID, LimitTokens: 1,
		},
	)
	require.ErrorIs(t, err, ErrConflict)
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
	beforeReplay, err := os.ReadFile(fixture.service.store.path)
	require.NoError(t, err)
	beforeRevision := fixture.service.store.state.Revision
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
	afterReplay, err := os.ReadFile(fixture.service.store.path)
	require.NoError(t, err)
	require.Equal(t, beforeReplay, afterReplay)
	require.Equal(t, beforeRevision, fixture.service.store.state.Revision)
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

func TestProjectBudgetContextPreservesTargetUserIdentity(t *testing.T) {
	fixture := newTestFixture(t)
	budget, err := fixture.service.PutTokenBudget(
		fixture.alice.ID,
		PutTokenBudgetInput{
			ID: "project-budget", ProjectID: fixture.projectA.ID, LimitTokens: 100,
		},
	)
	require.NoError(t, err)
	bob, err := fixture.service.CompileContext(
		fixture.alice.ID,
		CompileContextInput{
			ProjectID: fixture.projectA.ID,
			UserID:    fixture.bob.ID, BudgetID: budget.ID,
		},
	)
	require.NoError(t, err)
	mallory, err := fixture.service.CompileContext(
		fixture.alice.ID,
		CompileContextInput{
			ProjectID: fixture.projectA.ID,
			UserID:    fixture.mallory.ID, BudgetID: budget.ID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, fixture.bob.ID, bob.TargetUserID)
	require.Equal(t, fixture.mallory.ID, mallory.TargetUserID)
	require.Empty(t, bob.Budget.BudgetUserID)
	require.NotEqual(t, bob.Hash, mallory.Hash)
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

func TestRegistryRejectsSecretBearingFieldsAndSupportsCRUD(t *testing.T) {
	fixture := newTestFixture(t)
	checksum := strings.Repeat("d", 64)
	base := PutKnowledgeSourceInput{
		ID: "knowledge-crud", ProjectID: fixture.projectA.ID,
		Name: "Knowledge", URI: "https://example.invalid/knowledge",
		Revision: "1", SHA256: checksum, Status: RegistryDraft,
		Metadata: map[string]string{"source_kind": "documentation"},
	}
	for _, uri := range []string{
		"http://example.invalid/plain",
		"https://user:secret@example.invalid/knowledge",
		"https://example.invalid/knowledge?token=secret",
		"https://example.invalid/knowledge#secret",
		"//attacker/share/knowledge",
		`\\attacker\share\knowledge`,
		"x://attacker/share/knowledge",
		`\\?\C:\vault\knowledge`,
		"file:////attacker/share/knowledge",
		"file://localhost//attacker/share/knowledge",
		"file:////./PIPE/goclaw",
		"file:////%3F/C:/vault/knowledge",
		`C:\NUL`,
		`C:\vault\COM1.txt`,
		"file:///C:/NUL",
		"/dev/zero",
		"file:///dev/zero",
		"/proc/self/environ",
		"file:///sys/kernel",
		"/tmp/../dev/zero",
		"file:///tmp/../dev/zero",
		"/var/../proc/self/environ",
		"file:///opt/../sys/kernel",
		"file:///tmp/%2e%2e/proc/self/environ",
		`C:\vault\COM¹.txt`,
		`C:\vault\LPT².log`,
		"file:///C:/vault/COM%C2%B9.txt",
		"/NUL",
		"/vault/COM1.txt",
		"file:///NUL",
		"file:///vault/LPT².log",
		"file:///C|/NUL",
	} {
		input := base
		input.URI = uri
		_, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
		require.Error(t, err, uri)
	}
	input := base
	input.Metadata = map[string]string{"provider_token": "secret"}
	_, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.ErrorContains(t, err, "unsupported metadata key")
	input.Metadata = map[string]string{"source_kind": "git", "SOURCE_KIND": "file"}
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.ErrorContains(t, err, "duplicates canonical key")
	input.Metadata = map[string]string{"source_kind": "secret-value"}
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-value")
	input = base
	input.URI = "https://example.invalid/%zz-secret"
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "%zz-secret")
	input.URI = "https://example.invalid/" +
		strings.Repeat("x", maximumRegistryURILength)
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
	require.ErrorContains(t, err, "exceeds the maximum length")
	for _, uri := range []string{"/vault/line\nsecret-value", "/vault/\x00secret-value"} {
		input.URI = uri
		_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, input)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret-value")
	}

	created, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, base)
	require.NoError(t, err)
	got, err := fixture.service.GetKnowledgeSource(
		fixture.mallory.ID, fixture.projectA.ID, created.ID,
	)
	require.NoError(t, err)
	require.Equal(t, created, got)
	require.NoError(t, fixture.service.DeleteKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, created.ID,
	))
	_, err = fixture.service.GetKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, created.ID,
	)
	require.ErrorIs(t, err, ErrNotFound)

	approved := base
	approved.ID = "knowledge-approved"
	approved.Status = RegistryApproved
	value, err := fixture.service.PutKnowledgeSource(fixture.alice.ID, approved)
	require.NoError(t, err)
	downgrade := approved
	downgrade.Status = RegistryDraft
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, downgrade)
	require.ErrorIs(t, err, ErrInvalidTransition)
	require.ErrorIs(t, fixture.service.DeleteKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, value.ID,
	), ErrConflict)
	approved.Status = RegistryDisabled
	_, err = fixture.service.PutKnowledgeSource(fixture.alice.ID, approved)
	require.NoError(t, err)
	require.NoError(t, fixture.service.DeleteKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, value.ID,
	))

	skillInput := PutSkillReleaseInput{
		ID: "skill-delete", ProjectID: fixture.projectA.ID, Name: "skill",
		Version: "1", URI: "file:///skills/delete", SHA256: checksum,
		Status: RegistryApproved,
	}
	skill, err := fixture.service.PutSkillRelease(fixture.alice.ID, skillInput)
	require.NoError(t, err)
	skillInput.Status = RegistryDraft
	_, err = fixture.service.PutSkillRelease(fixture.alice.ID, skillInput)
	require.ErrorIs(t, err, ErrInvalidTransition)
	require.ErrorIs(t, fixture.service.DeleteSkillRelease(
		fixture.alice.ID, fixture.projectA.ID, skill.ID,
	), ErrConflict)
	skillInput.Status = RegistryDisabled
	_, err = fixture.service.PutSkillRelease(fixture.alice.ID, skillInput)
	require.NoError(t, err)
	require.NoError(t, fixture.service.DeleteSkillRelease(
		fixture.alice.ID, fixture.projectA.ID, skill.ID,
	))

	releaseInput := PutRunnerReleaseInput{
		ID: "runner-delete", ProjectID: fixture.projectA.ID,
		Channel: "pilot", Version: "1", OS: "linux", Arch: "amd64",
		URI: "https://example.invalid/runner.tar.gz", SHA256: checksum,
		MinProtocol: "1", Status: RegistryApproved,
	}
	release, err := fixture.service.PutRunnerRelease(fixture.alice.ID, releaseInput)
	require.NoError(t, err)
	releaseInput.Status = RegistryDraft
	_, err = fixture.service.PutRunnerRelease(fixture.alice.ID, releaseInput)
	require.ErrorIs(t, err, ErrInvalidTransition)
	require.ErrorIs(t, fixture.service.DeleteRunnerRelease(
		fixture.alice.ID, fixture.projectA.ID, release.ID,
	), ErrConflict)
	releaseInput.Status = RegistryDisabled
	_, err = fixture.service.PutRunnerRelease(fixture.alice.ID, releaseInput)
	require.NoError(t, err)
	require.NoError(t, fixture.service.DeleteRunnerRelease(
		fixture.alice.ID, fixture.projectA.ID, release.ID,
	))
}

func TestLegacyUnsafeRegistryFailsClosedOnReadAndCompile(t *testing.T) {
	fixture := newTestFixture(t)
	value, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "legacy-unsafe", ProjectID: fixture.projectA.ID,
			Name: "Legacy", URI: "file:///vault/legacy.md", Revision: "1",
			SHA256: strings.Repeat("e", 64), Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	require.NoError(t, fixture.service.store.update(func(st *state) error {
		key := projectResourceKey(fixture.projectA.ID, value.ID)
		legacy := st.KnowledgeSources[key]
		legacy.URI = "https://example.invalid/file?token=legacy"
		legacy.Metadata = map[string]string{"provider_token": "legacy"}
		st.KnowledgeSources[key] = legacy
		return nil
	}))

	_, err = fixture.service.ListKnowledgeSources(
		fixture.alice.ID, fixture.projectA.ID,
	)
	require.ErrorContains(t, err, "failed schema validation")
	_, err = fixture.service.CompileContext(
		fixture.alice.ID,
		CompileContextInput{
			ProjectID:    fixture.projectA.ID,
			KnowledgeIDs: []string{value.ID},
		},
	)
	require.ErrorContains(t, err, "failed schema validation")
}

func TestLegacyUnsafeUsageMetadataFailsClosedOnList(t *testing.T) {
	fixture := newTestFixture(t)
	budget, err := fixture.service.PutTokenBudget(
		fixture.alice.ID,
		PutTokenBudgetInput{
			ID: "legacy-usage-budget", ProjectID: fixture.projectA.ID,
			LimitTokens: 10,
		},
	)
	require.NoError(t, err)
	event, err := fixture.service.RecordTokenUsage(
		fixture.alice.ID,
		RecordTokenUsageInput{
			ID: "legacy-usage", ProjectID: fixture.projectA.ID,
			BudgetID: budget.ID, Tokens: 1,
		},
	)
	require.NoError(t, err)
	require.NoError(t, fixture.service.store.update(func(st *state) error {
		key := projectResourceKey(fixture.projectA.ID, event.ID)
		legacy := st.TokenUsageEvents[key]
		legacy.Metadata = map[string]string{"provider_token": "legacy-secret"}
		st.TokenUsageEvents[key] = legacy
		return nil
	}))

	_, err = fixture.service.ListTokenUsage(
		fixture.alice.ID, fixture.projectA.ID, budget.ID,
	)
	require.ErrorContains(t, err, "failed metadata schema validation")
}

func TestStoredContextHashMismatchFailsClosed(t *testing.T) {
	fixture := newTestFixture(t)
	bundle, err := fixture.service.CompileContext(
		fixture.alice.ID,
		CompileContextInput{ProjectID: fixture.projectA.ID},
	)
	require.NoError(t, err)
	require.NoError(t, fixture.service.store.update(func(st *state) error {
		key := projectResourceKey(fixture.projectA.ID, bundle.ID)
		value := st.ContextBundles[key]
		value.TargetUserID = fixture.bob.ID
		st.ContextBundles[key] = value
		return nil
	}))

	_, err = fixture.service.ListContextBundles(
		fixture.alice.ID, fixture.projectA.ID,
	)
	require.ErrorContains(t, err, "hash or id does not match")
}

func TestRegistryCRUDPersistsAcrossReopen(t *testing.T) {
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	value, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "persistent-source", ProjectID: fixture.projectA.ID,
			Name: "Persistent", URI: "file:///vault/persistent.md", Revision: "1",
			SHA256: strings.Repeat("f", 64), Status: RegistryDisabled,
		},
	)
	require.NoError(t, err)
	checksum := strings.Repeat("f", 64)
	skill, err := fixture.service.PutSkillRelease(
		fixture.alice.ID,
		PutSkillReleaseInput{
			ID: "persistent-skill", ProjectID: fixture.projectA.ID,
			Name: "Persistent", Version: "1", URI: "file:///skills/persistent",
			SHA256: checksum, Status: RegistryDisabled,
		},
	)
	require.NoError(t, err)
	release, err := fixture.service.PutRunnerRelease(
		fixture.alice.ID,
		PutRunnerReleaseInput{
			ID: "persistent-runner", ProjectID: fixture.projectA.ID,
			Channel: "pilot", Version: "1", OS: "linux", Arch: "amd64",
			URI: "https://example.invalid/runner.tar.gz", SHA256: checksum,
			MinProtocol: "1", Status: RegistryDisabled,
		},
	)
	require.NoError(t, err)
	reopened, err := Open(root)
	require.NoError(t, err)
	got, err := reopened.GetKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, value.ID,
	)
	require.NoError(t, err)
	require.Equal(t, value, got)
	gotSkill, err := reopened.GetSkillRelease(
		fixture.alice.ID, fixture.projectA.ID, skill.ID,
	)
	require.NoError(t, err)
	require.Equal(t, skill, gotSkill)
	gotRelease, err := reopened.GetRunnerRelease(
		fixture.alice.ID, fixture.projectA.ID, release.ID,
	)
	require.NoError(t, err)
	require.Equal(t, release, gotRelease)
	require.NoError(t, reopened.DeleteKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, value.ID,
	))
	require.NoError(t, reopened.DeleteSkillRelease(
		fixture.alice.ID, fixture.projectA.ID, skill.ID,
	))
	require.NoError(t, reopened.DeleteRunnerRelease(
		fixture.alice.ID, fixture.projectA.ID, release.ID,
	))
	reopenedAgain, err := Open(root)
	require.NoError(t, err)
	_, err = reopenedAgain.GetKnowledgeSource(
		fixture.alice.ID, fixture.projectA.ID, value.ID,
	)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = reopenedAgain.GetSkillRelease(
		fixture.alice.ID, fixture.projectA.ID, skill.ID,
	)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = reopenedAgain.GetRunnerRelease(
		fixture.alice.ID, fixture.projectA.ID, release.ID,
	)
	require.ErrorIs(t, err, ErrNotFound)
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

func TestInitialControlPlaneBareKeysMigrateToProjectKeys(t *testing.T) {
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	budget, err := fixture.service.PutTokenBudget(
		fixture.alice.ID,
		PutTokenBudgetInput{
			ID: "legacy-budget", ProjectID: fixture.projectA.ID, LimitTokens: 10,
		},
	)
	require.NoError(t, err)
	usage, err := fixture.service.RecordTokenUsage(
		fixture.alice.ID,
		RecordTokenUsageInput{
			ID: "legacy-usage", ProjectID: fixture.projectA.ID,
			BudgetID: budget.ID, Tokens: 1,
		},
	)
	require.NoError(t, err)
	currentBudgets, err := fixture.service.ListTokenBudgets(
		fixture.alice.ID, fixture.projectA.ID,
	)
	require.NoError(t, err)
	require.Len(t, currentBudgets, 1)
	budget = currentBudgets[0]
	checksum := strings.Repeat("a", 64)
	knowledge, err := fixture.service.PutKnowledgeSource(
		fixture.alice.ID,
		PutKnowledgeSourceInput{
			ID: "legacy-knowledge", ProjectID: fixture.projectA.ID,
			Name: "Legacy", URI: "file:///vault/legacy.md", Revision: "1",
			SHA256: checksum, Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	skill, err := fixture.service.PutSkillRelease(
		fixture.alice.ID,
		PutSkillReleaseInput{
			ID: "legacy-skill", ProjectID: fixture.projectA.ID,
			Name: "legacy", Version: "1", URI: "file:///skills/legacy",
			SHA256: checksum, Status: RegistryApproved,
		},
	)
	require.NoError(t, err)
	release, err := fixture.service.PutRunnerRelease(
		fixture.alice.ID,
		PutRunnerReleaseInput{
			ID: "legacy-runner", ProjectID: fixture.projectA.ID,
			Channel: "pilot", Version: "1", OS: "linux", Arch: "amd64",
			URI: "https://example.invalid/runner.tar.gz", SHA256: checksum,
			MinProtocol: "1",
		},
	)
	require.NoError(t, err)
	bundle, err := fixture.service.CompileContext(
		fixture.alice.ID,
		CompileContextInput{
			ProjectID: fixture.projectA.ID, BudgetID: budget.ID,
			KnowledgeIDs: []string{knowledge.ID}, SkillIDs: []string{skill.ID},
		},
	)
	require.NoError(t, err)
	statePath := filepath.Join(root, stateFileName)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var stored state
	require.NoError(t, json.Unmarshal(data, &stored))
	stored.TokenBudgets = map[string]TokenBudget{
		budget.ID: stored.TokenBudgets[projectResourceKey(fixture.projectA.ID, budget.ID)],
	}
	stored.TokenUsageEvents = map[string]TokenUsageEvent{
		usage.ID: stored.TokenUsageEvents[projectResourceKey(fixture.projectA.ID, usage.ID)],
	}
	stored.KnowledgeSources = map[string]KnowledgeSource{
		knowledge.ID: stored.KnowledgeSources[projectResourceKey(fixture.projectA.ID, knowledge.ID)],
	}
	stored.SkillReleases = map[string]SkillRelease{
		skill.ID: stored.SkillReleases[projectResourceKey(fixture.projectA.ID, skill.ID)],
	}
	stored.RunnerReleases = map[string]RunnerRelease{
		release.ID: stored.RunnerReleases[projectResourceKey(fixture.projectA.ID, release.ID)],
	}
	stored.ContextBundles = map[string]ContextBundle{
		bundle.ID: stored.ContextBundles[projectResourceKey(fixture.projectA.ID, bundle.ID)],
	}
	legacy, err := json.Marshal(stored)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, legacy, 0o600))

	reopened, err := Open(root)
	require.NoError(t, err)
	budgets, err := reopened.ListTokenBudgets(fixture.alice.ID, fixture.projectA.ID)
	require.NoError(t, err)
	require.Equal(t, []TokenBudget{budget}, budgets)
	require.Contains(t, reopened.store.state.TokenBudgets, projectResourceKey(
		fixture.projectA.ID, budget.ID,
	))
	require.Contains(t, reopened.store.state.TokenUsageEvents, projectResourceKey(
		fixture.projectA.ID, usage.ID,
	))
	require.Contains(t, reopened.store.state.KnowledgeSources, projectResourceKey(
		fixture.projectA.ID, knowledge.ID,
	))
	require.Contains(t, reopened.store.state.SkillReleases, projectResourceKey(
		fixture.projectA.ID, skill.ID,
	))
	require.Contains(t, reopened.store.state.RunnerReleases, projectResourceKey(
		fixture.projectA.ID, release.ID,
	))
	require.Contains(t, reopened.store.state.ContextBundles, projectResourceKey(
		fixture.projectA.ID, bundle.ID,
	))
	persistedData, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var persisted state
	require.NoError(t, json.Unmarshal(persistedData, &persisted))
	require.Contains(t, persisted.TokenBudgets, projectResourceKey(
		fixture.projectA.ID, budget.ID,
	))
	require.Contains(t, persisted.TokenUsageEvents, projectResourceKey(
		fixture.projectA.ID, usage.ID,
	))
	require.Contains(t, persisted.KnowledgeSources, projectResourceKey(
		fixture.projectA.ID, knowledge.ID,
	))
	require.Contains(t, persisted.SkillReleases, projectResourceKey(
		fixture.projectA.ID, skill.ID,
	))
	require.Contains(t, persisted.RunnerReleases, projectResourceKey(
		fixture.projectA.ID, release.ID,
	))
	require.Contains(t, persisted.ContextBundles, projectResourceKey(
		fixture.projectA.ID, bundle.ID,
	))
}

func TestLegacyProjectBudgetAboveJavaScriptSafeIntegerFailsLoad(t *testing.T) {
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	statePath := filepath.Join(root, stateFileName)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var stored state
	require.NoError(t, json.Unmarshal(data, &stored))
	for index := 0; index < 10; index++ {
		id := fmt.Sprintf("legacy-overflow-%d", index)
		budget := TokenBudget{
			ID: id, ProjectID: fixture.projectA.ID,
			LimitTokens: MaxTokenBudget,
		}
		stored.TokenBudgets[id] = budget
	}
	legacy, err := json.Marshal(stored)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, legacy, 0o600))

	_, err = Open(root)
	require.ErrorContains(t, err, "JavaScript safe integer")
}

func TestLegacyCompositeKeyMigrationCollisionFailsLoad(t *testing.T) {
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	statePath := filepath.Join(root, stateFileName)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var stored state
	require.NoError(t, json.Unmarshal(data, &stored))
	value := TokenBudget{
		ID: "duplicate", ProjectID: fixture.projectA.ID, LimitTokens: 10,
	}
	stored.TokenBudgets = map[string]TokenBudget{
		"legacy-a": value,
		"legacy-b": value,
	}
	legacy, err := json.Marshal(stored)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, legacy, 0o600))

	_, err = Open(root)
	require.ErrorContains(t, err, "multiple resources normalize")
	after, readErr := os.ReadFile(statePath)
	require.NoError(t, readErr)
	require.Equal(t, legacy, after)
}

func TestExistingStatePermissionsFailClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix state-file permission bits")
	}
	root := t.TempDir()
	fixture := newTestFixtureAt(t, root)
	statePath := filepath.Join(root, stateFileName)
	require.NoError(t, os.Chmod(statePath, 0o644))

	_, err := Open(root)
	require.ErrorContains(t, err, "permissions")
	require.NoError(t, os.Chmod(statePath, 0o600))
	_, err = Open(root)
	require.NoError(t, err)
	require.NoError(t, os.Chmod(root, 0o777))
	_, err = Open(root)
	require.ErrorContains(t, err, "allow non-owner writes")
	require.NoError(t, os.Chmod(root, 0o700))
	_ = fixture
}
