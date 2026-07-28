package orchestratorlite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/smallnest/goclaw/governance"
	"github.com/stretchr/testify/require"
)

type fakeHand struct {
	execute func(HandRequest) error
	review  IndependentReview
}

func (h *fakeHand) Execute(_ context.Context, request HandRequest) (HandResult, error) {
	if h.execute != nil {
		if err := h.execute(request); err != nil {
			return HandResult{}, err
		}
	}
	events := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thread-test"}`,
		`{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":20}}`,
	}, "\n") + "\n"
	_ = os.WriteFile(request.EventsPath, []byte(events), 0o600)
	return HandResult{
		ThreadID:  "thread-test",
		FinalText: "Implemented and checked the frozen task.",
		Usage:     CodexUsage{InputTokens: 100, OutputTokens: 20},
	}, nil
}

func (h *fakeHand) Review(_ context.Context, _ ReviewRequest) (IndependentReview, HandResult, error) {
	review := h.review
	if review.Summary == "" {
		review = IndependentReview{Passed: true, Summary: "Independent review passed."}
	}
	return review, HandResult{ThreadID: "review-thread"}, nil
}

func TestTaskRequiresFourReviewsBeforeFreeze(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createTestTask(t, service, repo, []string{"README.md"})

	_, err := service.FreezeTask(context.Background(), task.ID, "alice")
	require.ErrorContains(t, err, "requires all four approved reviews")

	for _, kind := range RequiredReviewKinds {
		task, err = service.ReviewTask(task.ID, kind, ReviewApproved, "alice", "review evidence accepted")
		require.NoError(t, err)
	}
	require.Equal(t, TaskReadyToFreeze, task.Status)

	task, err = service.FreezeTask(context.Background(), task.ID, "alice")
	require.NoError(t, err)
	require.Equal(t, TaskFrozen, task.Status)
	require.NotEmpty(t, task.Compile.BaseCommit)
	require.NotEmpty(t, task.Compile.ExecutionBundleHash)

	events, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Len(t, events, 6)
	require.Equal(t, "task.frozen", events[len(events)-1].Type)
}

func TestRunProducesEvidenceAndRequiresHumanAcceptance(t *testing.T) {
	service, repo := newTestService(t, true)
	service.SetHand(&fakeHand{
		execute: func(request HandRequest) error {
			return os.WriteFile(filepath.Join(request.Task.WorktreePath, "README.md"), []byte("# changed\n"), 0o644)
		},
	})
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})

	task, err := service.RunTask(context.Background(), task.ID, "runner")
	require.NoError(t, err)
	require.Equal(t, TaskAwaitingAcceptance, task.Status)
	require.NotNil(t, task.LastGate)
	require.True(t, task.LastGate.Passed)
	require.FileExists(t, filepath.Join(task.LastEvidence, "evidence.json"))
	require.FileExists(t, filepath.Join(task.LastEvidence, "donegate.json"))
	require.FileExists(t, filepath.Join(task.LastEvidence, "diff.patch"))

	task, err = service.AcceptTask(context.Background(), task.ID, "alice", "Evidence reviewed")
	require.NoError(t, err)
	require.Equal(t, TaskDone, task.Status)

	runGitTest(t, task.WorktreePath, "config", "user.name", "Test")
	runGitTest(t, task.WorktreePath, "config", "user.email", "test@example.com")
	task, err = service.CommitTask(context.Background(), task.ID, "alice", "Accept verified README update")
	require.NoError(t, err)
	require.NotEmpty(t, task.CommitSHA)
	commitLog, logErr := runGit(context.Background(), task.WorktreePath, "log", "-1", "--pretty=%B")
	require.NoError(t, logErr)
	require.Contains(t, commitLog.Stdout, "Task-ID: "+task.ID)
	require.Contains(t, commitLog.Stdout, "Work-Item: implementation")
	status, err := runGit(context.Background(), task.WorktreePath, "status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(status.Stdout))

	task, err = service.RecordPullRequest(
		task.ID,
		"alice",
		"https://example.test/pull/42",
	)
	require.NoError(t, err)
	require.Equal(t, "https://example.test/pull/42", task.PullRequestURL)
	beforeRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	task, err = service.RecordPullRequest(
		task.ID,
		"alice",
		"https://example.test/pull/42",
	)
	require.NoError(t, err)
	afterRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Len(t, afterRetry, len(beforeRetry))
	_, err = service.RecordPullRequest(
		task.ID,
		"alice",
		"https://example.test/pull/99",
	)
	require.ErrorContains(t, err, "already linked")
	_, err = parsePullRequestURL(
		"https://token@example.test/pull/42?access_token=secret",
	)
	require.ErrorContains(t, err, "without credentials")

	events, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Equal(t, "task.pull_request_linked", events[len(events)-1].Type)
}

func TestRevisionUsesIndependentWorktreeAndBranch(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})
	task, _, err := service.ensureWorktree(context.Background(), task)
	require.NoError(t, err)
	firstWorktree := task.WorktreePath
	firstBranch := task.Branch

	task, err = service.ReviseTask(task.ID, "alice", "change the accepted scope", Task{
		Goal: GoalSpec{Objective: "Revise README with the new accepted scope."},
	})
	require.NoError(t, err)
	for _, kind := range RequiredReviewKinds {
		task, err = service.ReviewTask(task.ID, kind, ReviewApproved, "alice", "reviewed revised task")
		require.NoError(t, err)
	}
	task, err = service.FreezeTask(context.Background(), task.ID, "alice")
	require.NoError(t, err)
	task, _, err = service.ensureWorktree(context.Background(), task)
	require.NoError(t, err)
	require.NotEqual(t, firstWorktree, task.WorktreePath)
	require.NotEqual(t, firstBranch, task.Branch)
	require.Contains(t, task.Branch, "-r2")
}

func TestWorkItemAttributionUsesFrozenScopePaths(t *testing.T) {
	task := Task{
		IssueIDs: []string{"bug-1"},
		Plan: PlanSpec{Milestones: []Milestone{{
			ID: "m1",
			WorkItems: []WorkItem{
				{ID: "backend", ScopePaths: []string{"server/**"}},
				{ID: "frontend", ScopePaths: []string{"web/**"}},
			},
		}}},
	}
	attribution, unattributed := attributeChangedFiles(task, []string{
		"server/service.go",
		"web/src/app.ts",
		"README.md",
	})
	require.Len(t, attribution, 2)
	require.Equal(t, []string{"README.md"}, unattributed)
	require.Equal(t, []string{"bug-1"}, attribution[0].IssueIDs)
}

func TestPolicyViolationBlocksDoneGate(t *testing.T) {
	service, repo := newTestService(t, false)
	service.SetHand(&fakeHand{
		execute: func(request HandRequest) error {
			return os.WriteFile(filepath.Join(request.Task.WorktreePath, ".env"), []byte("SECRET=x\n"), 0o600)
		},
	})
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})

	task, err := service.RunTask(context.Background(), task.ID, "runner")
	require.NoError(t, err)
	require.Equal(t, TaskRepairPending, task.Status)
	require.NotNil(t, task.LastGate)
	require.False(t, task.LastGate.Passed)
	require.Contains(t, strings.Join(task.LastGate.Reasons, "\n"), "denied path changed")
}

func TestRunVerificationsFailsClosedWithoutSandbox(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	require.NoError(t, os.Mkdir(repo, 0o755))
	service, err := NewService(Config{
		Root:     filepath.Join(root, "runtime"),
		RepoPath: repo,
	})
	require.NoError(t, err)
	runDir := filepath.Join(root, "run")
	require.NoError(t, os.Mkdir(runDir, 0o700))
	marker := filepath.Join(root, "verification-executed")

	results := service.runVerifications(
		context.Background(),
		Task{
			WorktreePath: repo,
			EvidencePlan: EvidencePlan{Commands: []CommandSpec{{
				Name: "must not execute",
				Argv: []string{
					"/bin/sh",
					"-c",
					"touch " + shellSingleQuote(marker),
				},
			}}},
		},
		runDir,
	)

	require.Len(t, results, 1)
	require.False(t, results[0].Passed)
	require.Equal(t, -1, results[0].ExitCode)
	require.Contains(t, results[0].Stderr, "verification_sandbox is required")
	_, statErr := os.Stat(marker)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	require.FileExists(t, filepath.Join(runDir, "verification-01.json"))
}

func TestRunVerificationsUsesValidatedSandboxContract(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	require.NoError(t, os.Mkdir(repo, 0o755))
	wrapper := filepath.Join(root, "verification-wrapper")
	wrapperScript := `#!/bin/sh
test "$1" = "--static-fixture" || exit 91
shift
worktree="$1"
home="$2"
shift 2
test "$1" = "--" || exit 92
shift
test "$PWD" = "$worktree" || exit 93
test "$HOME" = "$home" || exit 94
test "$CODEX_HOME" = "$home/.codex" || exit 95
test -z "$OPENAI_API_KEY" || exit 96
exec "$@"
`
	require.NoError(t, os.WriteFile(wrapper, []byte(wrapperScript), 0o700))
	t.Setenv("OPENAI_API_KEY", "must-not-reach-verification")
	service, err := NewService(Config{
		Root:                filepath.Join(root, "runtime"),
		RepoPath:            repo,
		VerificationSandbox: []string{wrapper, "--static-fixture"},
	})
	require.NoError(t, err)
	runDir := filepath.Join(root, "run")
	require.NoError(t, os.Mkdir(runDir, 0o700))

	results := service.runVerifications(
		context.Background(),
		Task{
			WorktreePath: repo,
			EvidencePlan: EvidencePlan{Commands: []CommandSpec{{
				Name: "sandbox plumbing",
				Argv: []string{
					"/bin/sh",
					"-c",
					`printf '%s|%s' "$HOME" "$PWD"`,
				},
			}}},
		},
		runDir,
	)

	require.Len(t, results, 1)
	require.True(t, results[0].Passed, results[0].Stderr)
	verificationHome := filepath.Join(runDir, "verification-home")
	require.Equal(t, verificationHome+"|"+repo, results[0].Stdout)
	require.DirExists(t, verificationHome)
	require.FileExists(t, filepath.Join(runDir, "verification-01.json"))
}

func TestNewServiceValidatesVerificationSandbox(t *testing.T) {
	root := t.TempDir()
	_, err := NewService(Config{
		Root:                filepath.Join(root, "relative-runtime"),
		VerificationSandbox: []string{"relative-wrapper"},
	})
	require.ErrorContains(t, err, "must be an absolute path")

	notExecutable := filepath.Join(root, "not-executable")
	require.NoError(t, os.WriteFile(notExecutable, []byte("fixture"), 0o600))
	_, err = NewService(Config{
		Root:                filepath.Join(root, "non-executable-runtime"),
		VerificationSandbox: []string{notExecutable},
	})
	require.ErrorContains(t, err, "regular executable file")

	if runtime.GOOS == "windows" {
		return
	}
	executable := filepath.Join(root, "executable")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o700))
	_, err = NewService(Config{
		Root:                   filepath.Join(root, "mutually-exclusive-runtime"),
		VerificationSandbox:    []string{executable},
		UnsafeHostVerification: true,
	})
	require.ErrorContains(t, err, "mutually exclusive")
	groupWritable := filepath.Join(root, "group-writable")
	require.NoError(t, os.WriteFile(groupWritable, []byte("fixture"), 0o700))
	require.NoError(t, os.Chmod(groupWritable, 0o720))
	_, err = NewService(Config{
		Root:                filepath.Join(root, "writable-runtime"),
		VerificationSandbox: []string{groupWritable},
	})
	require.ErrorContains(t, err, "must not be writable by group or others")
}

func TestDoneGateRejectsCommitCreatedByCodexHand(t *testing.T) {
	service, repo := newTestService(t, false)
	service.SetHand(&fakeHand{
		execute: func(request HandRequest) error {
			if err := os.WriteFile(
				filepath.Join(request.Task.WorktreePath, "README.md"),
				[]byte("# committed by hand\n"),
				0o644,
			); err != nil {
				return err
			}
			if result, err := runGit(
				context.Background(),
				request.Task.WorktreePath,
				"add",
				"README.md",
			); err != nil {
				return fmt.Errorf("stage fixture: %w: %s", err, result.Stderr)
			}
			if result, err := runGit(
				context.Background(),
				request.Task.WorktreePath,
				"-c",
				"user.name=Codex Hand",
				"-c",
				"user.email=codex-hand@example.test",
				"commit",
				"-m",
				"unauthorized automatic commit",
			); err != nil {
				return fmt.Errorf("commit fixture: %w: %s", err, result.Stderr)
			}
			return nil
		},
	})
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})

	task, err := service.RunTask(context.Background(), task.ID, "runner")
	require.NoError(t, err)
	require.Equal(t, TaskRepairPending, task.Status)
	require.NotNil(t, task.LastGate)
	require.False(t, task.LastGate.Passed)
	require.Contains(
		t,
		strings.Join(task.LastGate.Reasons, "\n"),
		"does not match frozen base",
	)
	require.Contains(
		t,
		strings.Join(task.LastGate.Reasons, "\n"),
		"Codex Hand must not commit",
	)
}

func TestDoneGateRejectsRepositoryHeadDrift(t *testing.T) {
	const (
		baseCommit = "1111111111111111111111111111111111111111"
		headCommit = "2222222222222222222222222222222222222222"
	)
	tests := []struct {
		name   string
		mutate func(*EvidencePackage)
		match  string
	}{
		{
			name: "before",
			mutate: func(pack *EvidencePackage) {
				pack.Before.HeadCommit = headCommit
			},
			match: "repository before HEAD",
		},
		{
			name: "after",
			mutate: func(pack *EvidencePackage) {
				pack.After.HeadCommit = headCommit
			},
			match: "repository after HEAD",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := Task{Compile: CompileRecord{BaseCommit: baseCommit}}
			pack := EvidencePackage{
				Before: RepositorySnapshot{HeadCommit: baseCommit},
				After:  RepositorySnapshot{HeadCommit: baseCommit},
			}
			test.mutate(&pack)

			gate := evaluateDoneGate(task, pack, "run", "evidence", "tree")
			require.False(t, gate.Passed)
			require.Contains(t, strings.Join(gate.Reasons, "\n"), test.match)
			require.Contains(t, strings.Join(gate.Reasons, "\n"), baseCommit)
		})
	}
}

func TestEnsureWorktreeRejectsAdvancedHeadForCurrentRun(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})
	task, _, err := service.ensureWorktree(context.Background(), task)
	require.NoError(t, err)
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(task.WorktreePath, "README.md"),
			[]byte("# advanced\n"),
			0o644,
		),
	)
	runGitTest(t, task.WorktreePath, "add", "README.md")
	runGitTest(
		t,
		task.WorktreePath,
		"-c",
		"user.name=Test",
		"-c",
		"user.email=test@example.com",
		"commit",
		"-m",
		"advance worktree",
	)
	task.CurrentRunID = "run-existing"

	_, _, err = service.ensureWorktree(context.Background(), task)
	require.ErrorContains(t, err, "does not match frozen base")
}

func TestGovernanceSeparatesReviewKindsAndFinalAcceptance(t *testing.T) {
	service, repo := newTestService(t, true)
	service.SetGovernancePolicy(governance.Config{
		Enabled:                           true,
		RequireRationale:                  true,
		MinRationaleRunes:                 1,
		MinDistinctTaskReviewers:          2,
		MaxTaskReviewKindsPerReviewer:     2,
		ForbidSelfApproval:                true,
		ForbidFinalApproverFromTaskReview: true,
	})
	service.SetHand(&fakeHand{
		execute: func(request HandRequest) error {
			return os.WriteFile(
				filepath.Join(request.Task.WorktreePath, "README.md"),
				[]byte("# governed\n"),
				0o644,
			)
		},
	})
	task, err := service.CreateTask(CreateRequest{
		ProjectID: "test",
		Title:     "Update README",
		RepoPath:  repo,
		Request: RequestFrame{
			RawRequest: "Update README with the approved text.",
		},
		EvidencePlan: EvidencePlan{Commands: []CommandSpec{{
			Name: "git diff check",
			Argv: []string{"git", "diff", "--check"},
		}}},
		Scope: ScopePolicy{
			AllowedPaths:    []string{"README.md"},
			MaxChangedFiles: 2,
			MaxChangedLines: 20,
		},
		CreatedBy:   "planner-service",
		RequestedBy: "alice",
	})
	require.NoError(t, err)
	require.Equal(t, "planner-service", task.CreatedBy)
	require.Equal(t, "alice", task.RequestedBy)
	review := func(reviewer string, kind ReviewKind) governance.Review {
		return governance.Review{
			ReviewerID: reviewer,
			Role:       governanceRoleForReview(kind),
			Rationale:  "reviewed",
			Source:     "test",
			CreatedAt:  time.Now().UTC(),
		}
	}
	task, err = service.ReviewTaskWithReview(
		task.ID, ReviewScenario, ReviewApproved, review("alice", ReviewScenario))
	require.NoError(t, err)
	task, err = service.ReviewTaskWithReview(
		task.ID, ReviewCapacity, ReviewApproved, review("alice", ReviewCapacity))
	require.NoError(t, err)
	_, err = service.ReviewTaskWithReview(
		task.ID, ReviewRisk, ReviewApproved, review("alice", ReviewRisk))
	require.ErrorContains(t, err, "maximum 2 task review kinds")
	task, err = service.ReviewTaskWithReview(
		task.ID, ReviewRisk, ReviewApproved, review("bob", ReviewRisk))
	require.NoError(t, err)
	task, err = service.ReviewTaskWithReview(
		task.ID, ReviewCost, ReviewApproved, review("bob", ReviewCost))
	require.NoError(t, err)
	require.Equal(t, TaskReadyToFreeze, task.Status)

	task, err = service.FreezeTask(context.Background(), task.ID, "freezer")
	require.NoError(t, err)
	task, err = service.RunTask(context.Background(), task.ID, "runner")
	require.NoError(t, err)
	require.Equal(t, TaskAwaitingAcceptance, task.Status)
	_, err = service.AcceptTaskWithReview(context.Background(), task.ID, governance.Review{
		ReviewerID: "alice",
		Role:       governance.RoleTaskAccept,
		Rationale:  "accept",
		Source:     "test",
		CreatedAt:  time.Now().UTC(),
	})
	require.ErrorContains(t, err, "participated in task review")
	task, err = service.AcceptTaskWithReview(context.Background(), task.ID, governance.Review{
		ReviewerID: "carol",
		Role:       governance.RoleTaskAccept,
		Rationale:  "accept",
		Source:     "test",
		CreatedAt:  time.Now().UTC(),
	})
	require.NoError(t, err)
	require.Equal(t, TaskDone, task.Status)
}

func TestEventTamperIsDetected(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createTestTask(t, service, repo, []string{"README.md"})
	path := service.eventsPath(task.ID)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var event SessionEvent
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(string(data))), &event))
	event.Snapshot.Title = "tampered"
	tampered, err := json.Marshal(event)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(tampered, '\n'), 0o600))

	_, err = service.GetTask(task.ID)
	require.ErrorContains(t, err, "event content hash mismatch")
}

func newTestService(t *testing.T, independentReview bool) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	runGitTest(t, repo, "init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte("# initial\n"), 0o644))
	runGitTest(t, repo, "add", "README.md")
	runGitTest(t, repo, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")

	service, err := NewService(Config{
		Root:                      filepath.Join(root, "runtime"),
		RepoPath:                  repo,
		IndependentReview:         independentReview,
		RequireHumanFinalApproval: true,
		UnsafeHostVerification:    true,
		MaxRepairAttempts:         2,
	})
	require.NoError(t, err)
	return service, repo
}

func createTestTask(t *testing.T, service *Service, repo string, allowed []string) Task {
	t.Helper()
	task, err := service.CreateTask(CreateRequest{
		ProjectID: "test",
		Title:     "Update README",
		RepoPath:  repo,
		Request:   RequestFrame{RawRequest: "Update README with the approved text."},
		EvidencePlan: EvidencePlan{Commands: []CommandSpec{{
			Name: "git diff check",
			Argv: []string{"git", "diff", "--check"},
		}}},
		Scope: ScopePolicy{
			AllowedPaths:    allowed,
			MaxChangedFiles: 2,
			MaxChangedLines: 20,
		},
		CreatedBy: "alice",
	})
	require.NoError(t, err)
	return task
}

func createAndFreezeTask(t *testing.T, service *Service, repo string, allowed []string) Task {
	t.Helper()
	task := createTestTask(t, service, repo, allowed)
	var err error
	for _, kind := range RequiredReviewKinds {
		task, err = service.ReviewTask(task.ID, kind, ReviewApproved, "alice", "review evidence accepted")
		require.NoError(t, err)
	}
	task, err = service.FreezeTask(context.Background(), task.ID, "alice")
	require.NoError(t, err)
	return task
}

func TestCreateTaskWithStableIDIsIdempotent(t *testing.T) {
	service, repo := newTestService(t, false)
	request := CreateRequest{
		ID:        "task-stable-create",
		ProjectID: "test",
		Title:     "Stable create",
		RepoPath:  repo,
		Request: RequestFrame{
			RawRequest: "Create this task exactly once.",
		},
		CreatedBy: "alice",
	}
	first, err := service.CreateTask(request)
	require.NoError(t, err)
	second, err := service.CreateTask(request)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	events, err := service.ListEvents(first.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)

	request.Title = "Changed create payload"
	_, err = service.CreateTask(request)
	require.ErrorContains(t, err, "different create request")
}

func TestReviewAndFreezeRetriesDoNotAppendDuplicateEvents(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createTestTask(t, service, repo, []string{"README.md"})
	var err error
	for _, kind := range RequiredReviewKinds {
		task, err = service.ReviewTask(
			task.ID,
			kind,
			ReviewApproved,
			"alice",
			"review evidence accepted",
		)
		require.NoError(t, err)
		eventsBeforeRetry, err := service.ListEvents(task.ID)
		require.NoError(t, err)
		task, err = service.ReviewTask(
			task.ID,
			kind,
			ReviewApproved,
			"alice",
			"review evidence accepted",
		)
		require.NoError(t, err)
		eventsAfterRetry, err := service.ListEvents(task.ID)
		require.NoError(t, err)
		require.Len(t, eventsAfterRetry, len(eventsBeforeRetry))
	}
	task, err = service.FreezeTask(context.Background(), task.ID, "alice")
	require.NoError(t, err)
	eventsBeforeRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	retried, err := service.FreezeTask(context.Background(), task.ID, "alice")
	require.NoError(t, err)
	require.Equal(t, task.Compile.ExecutionBundleHash, retried.Compile.ExecutionBundleHash)
	eventsAfterRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Len(t, eventsAfterRetry, len(eventsBeforeRetry))
}

func TestTaskMutationLockPreventsCancelDuringEvidenceOperation(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})
	release, err := service.acquireRun(task.ID, false)
	require.NoError(t, err)
	_, err = service.CancelTask(
		task.ID,
		"alice",
		"must not race an evidence import",
	)
	require.ErrorContains(t, err, "already running")
	current, err := service.GetTask(task.ID)
	require.NoError(t, err)
	require.Equal(t, TaskFrozen, current.Status)
	release()

	cancelled, err := service.CancelTask(
		task.ID,
		"alice",
		"cancel after the evidence operation releases its lock",
	)
	require.NoError(t, err)
	require.Equal(t, TaskCancelled, cancelled.Status)
}

func runGitTest(t *testing.T, repo string, args ...string) {
	t.Helper()
	result, err := runGit(context.Background(), repo, args...)
	require.NoErrorf(t, err, "git %v failed: %s", args, result.Stderr)
}
