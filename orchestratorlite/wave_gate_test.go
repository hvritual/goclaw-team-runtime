package orchestratorlite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type waveGateFixtureOptions struct {
	secondActive     bool
	dependencyStatus string
	registryProduct  bool
	planProduct      bool
	planStatus       string
	planRevision     int
	taskAllowedPaths []string
	bindingMutation  func(*WaveBinding)
}

func TestResolveWaveBindingAtExactBase(t *testing.T) {
	_, request := newWaveGateFixture(t, waveGateFixtureOptions{
		dependencyStatus: "complete",
		registryProduct:  true,
		planProduct:      true,
		planStatus:       "approved",
		planRevision:     1,
		taskAllowedPaths: []string{"gateway/**"},
	})
	binding, baseCommit, err := ResolveWaveBinding(
		context.Background(),
		request.RepoPath,
		"main",
		"PILOT-W00-S03",
	)
	require.NoError(t, err)
	require.Equal(t, *request.Wave, binding)
	require.True(t, validGitCommitReference(baseCommit))
	head, err := runGit(
		context.Background(),
		request.RepoPath,
		"rev-parse",
		"HEAD",
	)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(head.Stdout), baseCommit)
}

func TestResolveWaveBindingRejectsUndeclaredStep(t *testing.T) {
	_, request := newWaveGateFixture(t, waveGateFixtureOptions{
		dependencyStatus: "complete",
		registryProduct:  true,
		planProduct:      true,
		planStatus:       "approved",
		planRevision:     1,
		taskAllowedPaths: []string{"gateway/**"},
	})
	_, _, err := ResolveWaveBinding(
		context.Background(),
		request.RepoPath,
		"main",
		"PILOT-W00-S99",
	)
	require.ErrorContains(t, err, "is not declared")
}

func TestWaveGateFreezeMatrix(t *testing.T) {
	tests := []struct {
		name      string
		options   waveGateFixtureOptions
		wantError string
	}{
		{
			name: "valid active approved wave",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
			},
		},
		{
			name: "multiple active waves",
			options: waveGateFixtureOptions{
				secondActive:     true,
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
			},
			wantError: "exactly one active wave",
		},
		{
			name: "dependency incomplete",
			options: waveGateFixtureOptions{
				dependencyStatus: "blocked",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
			},
			wantError: "is not complete",
		},
		{
			name: "plan not approved",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "draft",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
			},
			wantError: "plan is not approved",
		},
		{
			name: "binding revision drift",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
				bindingMutation: func(binding *WaveBinding) {
					binding.PlanRevision = 2
				},
			},
			wantError: "revision 1 does not match binding 2",
		},
		{
			name: "binding step absent",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
				bindingMutation: func(binding *WaveBinding) {
					binding.StepID = "PILOT-W00-S99"
				},
			},
			wantError: "is not declared",
		},
		{
			name: "registry hash drift",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
				bindingMutation: func(binding *WaveBinding) {
					binding.RegistrySHA256 = strings.Repeat("0", 64)
				},
			},
			wantError: "registry hash mismatch",
		},
		{
			name: "registry denies product changes",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  false,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
			},
			wantError: "does not allow product code changes",
		},
		{
			name: "plan denies product changes",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      false,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"gateway/**"},
			},
			wantError: "does not allow product code changes",
		},
		{
			name: "task scope exceeds wave",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
				taskAllowedPaths: []string{"workstation/**"},
			},
			wantError: "exceeds registry scope",
		},
		{
			name: "empty task scope is unrestricted",
			options: waveGateFixtureOptions{
				dependencyStatus: "complete",
				registryProduct:  true,
				planProduct:      true,
				planStatus:       "approved",
				planRevision:     1,
			},
			wantError: "allowed_paths is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, request := newWaveGateFixture(t, test.options)
			task, err := service.CreateTask(request)
			require.NoError(t, err)
			for _, kind := range RequiredReviewKinds {
				task, err = service.ReviewTask(
					task.ID,
					kind,
					ReviewApproved,
					"alice",
					"wave gate review evidence is accepted",
				)
				require.NoError(t, err)
			}
			task, err = service.FreezeTask(
				context.Background(),
				task.ID,
				"alice",
			)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, TaskFrozen, task.Status)
			require.NoError(
				t,
				service.ValidateWaveBinding(context.Background(), task),
			)
			message := commitMessageWithTraceability(task, "Pilot change")
			require.Contains(t, message, "Wave-ID: PILOT-W00")
			require.Contains(t, message, "Wave-Revision: 1")
			require.Contains(t, message, "Wave-Step: PILOT-W00-S03")
			require.Empty(t, missingCommitTrailers(task, message))
		})
	}
}

func TestWaveGateRejectsMissingIssueBinding(t *testing.T) {
	options := waveGateFixtureOptions{
		dependencyStatus: "complete",
		registryProduct:  true,
		planProduct:      true,
		planStatus:       "approved",
		planRevision:     1,
		taskAllowedPaths: []string{"gateway/**"},
	}
	service, request := newWaveGateFixture(t, options)
	request.IssueIDs = nil
	task, err := service.CreateTask(request)
	require.NoError(t, err)
	for _, kind := range RequiredReviewKinds {
		task, err = service.ReviewTask(
			task.ID,
			kind,
			ReviewApproved,
			"alice",
			"wave gate review evidence is accepted",
		)
		require.NoError(t, err)
	}
	_, err = service.FreezeTask(context.Background(), task.ID, "alice")
	require.ErrorContains(t, err, "at least one issue_id")
}

func newWaveGateFixture(
	t *testing.T,
	options waveGateFixtureOptions,
) (*Service, CreateRequest) {
	t.Helper()
	repo := t.TempDir()
	runGitTest(t, repo, "init", "-b", "main")
	runGitTest(t, repo, "config", "user.email", "wave@example.com")
	runGitTest(t, repo, "config", "user.name", "Wave Test")
	require.NoError(t, os.MkdirAll(
		filepath.Join(repo, "docs", "waves", "pilot"),
		0o755,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(repo, "README.md"),
		[]byte("# pilot\n"),
		0o600,
	))
	plan := []byte("---\n" +
		"schema: goclaw.wave/v1\n" +
		"wave_id: PILOT-W00\n" +
		"revision: " + strconv.Itoa(options.planRevision) + "\n" +
		"plan_status: " + options.planStatus + "\n" +
		"wave_state: active\n" +
		"approved_by:\n  - pilot-review\n" +
		"depends_on:\n  - BASE-W00\n" +
		"steps:\n  - PILOT-W00-S03\n" +
		"allowed_change_scope:\n  - gateway/**\n  - docs/**\n" +
		"product_code_changes_allowed: " +
		booleanString(options.planProduct) + "\n" +
		"---\n\n# Pilot\n")
	planPath := filepath.Join(repo, "docs", "waves", "pilot", "plan-r001.md")
	require.NoError(t, os.WriteFile(planPath, plan, 0o600))
	waves := []waveRegistryRecord{
		{
			ID:                        "BASE-W00",
			Status:                    options.dependencyStatus,
			Document:                  "base/plan-r001.md",
			ProductCodeChangesAllowed: false,
		},
		{
			ID:                        "PILOT-W00",
			Status:                    "active",
			Document:                  "pilot/plan-r001.md",
			DependsOn:                 []string{"BASE-W00"},
			AllowedChangeScope:        []string{"gateway/**", "docs/**"},
			ProductCodeChangesAllowed: options.registryProduct,
		},
	}
	if options.secondActive {
		waves = append(waves, waveRegistryRecord{
			ID:                        "PILOT-W01",
			Status:                    "active",
			Document:                  "pilot/plan-r002.md",
			ProductCodeChangesAllowed: true,
		})
	}
	registry, err := json.MarshalIndent(waveRegistry{
		SchemaVersion: 1,
		ActiveWave:    "PILOT-W00",
		Waves:         waves,
	}, "", "  ")
	require.NoError(t, err)
	registry = append(registry, '\n')
	require.NoError(t, os.WriteFile(
		filepath.Join(repo, filepath.FromSlash(waveRegistryPath)),
		registry,
		0o600,
	))
	runGitTest(t, repo, "add", ".")
	runGitTest(t, repo, "commit", "-m", "pilot base")

	binding := &WaveBinding{
		WaveID:         "PILOT-W00",
		PlanRevision:   options.planRevision,
		StepID:         "PILOT-W00-S03",
		PlanPath:       "docs/waves/pilot/plan-r001.md",
		RegistrySHA256: sha256Bytes(registry),
		PlanSHA256:     sha256Bytes(plan),
	}
	if options.bindingMutation != nil {
		options.bindingMutation(binding)
	}
	service, err := NewService(Config{
		Root:     filepath.Join(t.TempDir(), "runtime"),
		RepoPath: repo,
	})
	require.NoError(t, err)
	return service, CreateRequest{
		ID:           "task-wave-gate",
		ProjectID:    "project-pilot",
		RepositoryID: "repo-pilot",
		AssigneeID:   "alice",
		IssueIDs:     []string{"issue-pilot"},
		Wave:         binding,
		Title:        "Pilot wave gate",
		RepoPath:     repo,
		BaseRef:      "main",
		Request: RequestFrame{
			RawRequest: "Implement the approved pilot step.",
		},
		EvidencePlan: EvidencePlan{Commands: []CommandSpec{{
			Name: "git diff check",
			Argv: []string{"git", "diff", "--check"},
		}}},
		Scope: ScopePolicy{
			AllowedPaths:    options.taskAllowedPaths,
			MaxChangedFiles: 2,
			MaxChangedLines: 20,
		},
		CreatedBy:   "planner-service",
		RequestedBy: "alice",
	}
}

func booleanString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
