package orchestratorlite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestImportExecutionEvidencePassesDoneGateAndIsIdempotent(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})
	input := validImportedEvidence(task)

	imported, err := service.ImportExecutionEvidence(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, TaskAwaitingAcceptance, imported.Status)
	require.NotNil(t, imported.LastGate)
	require.True(t, imported.LastGate.Passed)
	require.FileExists(t, filepath.Join(imported.LastEvidence, "evidence.json"))
	require.FileExists(t, filepath.Join(imported.LastEvidence, "imported-evidence.json"))
	require.FileExists(t, filepath.Join(imported.LastEvidence, "diff.patch"))
	require.FileExists(t, filepath.Join(imported.LastEvidence, "donegate.json"))

	var pack EvidencePackage
	require.NoError(t, readJSON(filepath.Join(imported.LastEvidence, "evidence.json"), &pack))
	require.NotNil(t, pack.Imported)
	require.Equal(t, input.Evidence.BundleSHA256, pack.Imported.BundleSHA256)
	require.Equal(t, []string{"README.md"}, pack.Policy.ChangedFiles)
	require.Equal(t, 1, pack.Policy.AddedLines)
	require.Equal(t, 1, pack.Policy.DeletedLines)

	eventsBeforeRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	retried, err := service.ImportExecutionEvidence(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, imported.LastEvidence, retried.LastEvidence)
	eventsAfterRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Len(t, eventsAfterRetry, len(eventsBeforeRetry))

	accepted, err := service.AcceptTask(
		context.Background(),
		task.ID,
		"acceptor",
		"reviewed imported signed evidence",
	)
	require.NoError(t, err)
	require.Equal(t, TaskDone, accepted.Status)
	_, err = service.CommitTask(
		context.Background(),
		task.ID,
		"acceptor",
		"must remain on workstation",
	)
	require.ErrorContains(t, err, "commit from the owning workstation")

	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(repo, "README.md"),
			[]byte("# changed\n"),
			0o644,
		),
	)
	runGitTest(t, repo, "add", "README.md")
	runGitTest(
		t,
		repo,
		"commit",
		"-m",
		commitMessageWithTraceability(accepted, "Apply accepted workstation patch"),
	)
	head, err := runGit(context.Background(), repo, "rev-parse", "HEAD")
	require.NoError(t, err)
	commitSHA := strings.TrimSpace(head.Stdout)
	linked, err := service.RecordImportedPullRequest(
		context.Background(),
		task.ID,
		"acceptor",
		commitSHA,
		"https://example.test/pull/42",
	)
	require.NoError(t, err)
	require.Equal(t, commitSHA, linked.CommitSHA)
	require.Equal(t, "https://example.test/pull/42", linked.PullRequestURL)
	linkEvents, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Equal(
		t,
		"task.external_pull_request_linked",
		linkEvents[len(linkEvents)-1].Type,
	)
	var linkEvidence map[string]any
	require.NoError(
		t,
		json.Unmarshal(linkEvents[len(linkEvents)-1].Data, &linkEvidence),
	)
	require.Equal(t, true, linkEvidence["commit_verified"])
	require.Equal(t, false, linkEvidence["pull_request_url_verified"])
	_, hasAmbiguousVerifiedFlag := linkEvidence["verified"]
	require.False(t, hasAmbiguousVerifiedFlag)

	eventsBeforeLinkRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	retriedLink, err := service.RecordImportedPullRequest(
		context.Background(),
		task.ID,
		"acceptor",
		commitSHA,
		"https://example.test/pull/42",
	)
	require.NoError(t, err)
	require.Equal(t, linked.CommitSHA, retriedLink.CommitSHA)
	eventsAfterLinkRetry, err := service.ListEvents(task.ID)
	require.NoError(t, err)
	require.Len(t, eventsAfterLinkRetry, len(eventsBeforeLinkRetry))
}

func TestImportExecutionEvidenceRunsFrozenIndependentReview(t *testing.T) {
	service, repo := newTestService(t, true)
	service.SetHand(&fakeHand{
		review: IndependentReview{
			Passed:  true,
			Summary: "Independent imported-diff review passed.",
		},
	})
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})

	imported, err := service.ImportExecutionEvidence(
		context.Background(),
		validImportedEvidence(task),
	)
	require.NoError(t, err)
	require.Equal(t, TaskAwaitingAcceptance, imported.Status)
	require.NotNil(t, imported.LastGate)
	require.True(t, imported.LastGate.Passed)

	var pack EvidencePackage
	require.NoError(t, readJSON(filepath.Join(imported.LastEvidence, "evidence.json"), &pack))
	require.True(t, pack.Review.Passed)
	require.Equal(t, "Independent imported-diff review passed.", pack.Review.Summary)
	require.FileExists(t, filepath.Join(imported.LastEvidence, "codex-review-result.json"))
}

func TestImportExecutionEvidenceFailedCheckCannotAdvanceToAcceptance(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})
	input := validImportedEvidence(task)
	input.Evidence.Checks[0].Passed = false
	input.Evidence.Checks[0].ExitCode = 1
	input.Evidence.Checks[0].Details = "verification failed"

	imported, err := service.ImportExecutionEvidence(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, TaskRepairPending, imported.Status)
	require.NotNil(t, imported.LastGate)
	require.False(t, imported.LastGate.Passed)
	require.Contains(
		t,
		strings.Join(imported.LastGate.Reasons, "\n"),
		"verification failed: git diff check",
	)
	require.FileExists(t, filepath.Join(imported.LastEvidence, "evidence.json"))
}

func TestImportExecutionEvidenceRejectsDriftBeforePersistence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ImportExecutionEvidenceInput)
		match  string
	}{
		{
			name: "revision",
			mutate: func(input *ImportExecutionEvidenceInput) {
				input.TaskRevision++
			},
			match: "does not match frozen revision",
		},
		{
			name: "execution bundle",
			mutate: func(input *ImportExecutionEvidenceInput) {
				input.ExecutionBundleHash = strings.Repeat("e", 64)
			},
			match: "execution bundle hash does not match",
		},
		{
			name: "head commit",
			mutate: func(input *ImportExecutionEvidenceInput) {
				input.Evidence.HeadCommit = strings.Repeat("1", 40)
			},
			match: "HEAD must equal the frozen base commit",
		},
		{
			name: "automatic commit",
			mutate: func(input *ImportExecutionEvidenceInput) {
				input.Evidence.CommitSHA = strings.Repeat("2", 40)
			},
			match: "must not contain an automatic commit",
		},
		{
			name: "missing frozen check",
			mutate: func(input *ImportExecutionEvidenceInput) {
				input.Evidence.Checks = input.Evidence.Checks[1:]
			},
			match: `requires 1 "git diff check" check`,
		},
		{
			name: "diff changed path mismatch",
			mutate: func(input *ImportExecutionEvidenceInput) {
				input.Evidence.ChangedFiles = []string{"OTHER.md"}
			},
			match: "changed files do not match diff paths",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, repo := newTestService(t, false)
			task := createAndFreezeTask(t, service, repo, []string{"README.md"})
			input := validImportedEvidence(task)
			test.mutate(&input)

			_, err := service.ImportExecutionEvidence(context.Background(), input)
			require.ErrorContains(t, err, test.match)
			current, getErr := service.GetTask(task.ID)
			require.NoError(t, getErr)
			require.Equal(t, TaskFrozen, current.Status)
			require.Empty(t, current.LastEvidence)
			_, statErr := os.Stat(filepath.Join(service.taskDir(task.ID), "runs"))
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}
}

func TestImportedEvidenceAcceptanceDetectsDiffTamper(t *testing.T) {
	service, repo := newTestService(t, false)
	task := createAndFreezeTask(t, service, repo, []string{"README.md"})
	imported, err := service.ImportExecutionEvidence(
		context.Background(),
		validImportedEvidence(task),
	)
	require.NoError(t, err)
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(imported.LastEvidence, "diff.patch"),
			[]byte("tampered"),
			0o600,
		),
	)

	_, err = service.AcceptTask(
		context.Background(),
		task.ID,
		"acceptor",
		"reviewed imported signed evidence",
	)
	require.ErrorContains(t, err, "imported diff changed after DoneGate")
}

func TestRecordImportedPullRequestRejectsUnprovenCommit(t *testing.T) {
	tests := []struct {
		name    string
		content string
		message func(Task) string
		match   string
	}{
		{
			name:    "mismatched patch",
			content: "# different change\n",
			message: func(task Task) string {
				return commitMessageWithTraceability(task, "Apply a different patch")
			},
			match: "does not exactly match",
		},
		{
			name:    "missing trailers",
			content: "# changed\n",
			message: func(Task) string {
				return "Apply accepted patch without task trailers"
			},
			match: "missing traceability trailers",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, repo := newTestService(t, false)
			task := createAndFreezeTask(
				t,
				service,
				repo,
				[]string{"README.md"},
			)
			_, err := service.ImportExecutionEvidence(
				context.Background(),
				validImportedEvidence(task),
			)
			require.NoError(t, err)
			accepted, err := service.AcceptTask(
				context.Background(),
				task.ID,
				"acceptor",
				"reviewed imported signed evidence",
			)
			require.NoError(t, err)
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(repo, "README.md"),
					[]byte(test.content),
					0o644,
				),
			)
			runGitTest(t, repo, "add", "README.md")
			runGitTest(t, repo, "commit", "-m", test.message(accepted))
			head, err := runGit(
				context.Background(),
				repo,
				"rev-parse",
				"HEAD",
			)
			require.NoError(t, err)
			_, err = service.RecordImportedPullRequest(
				context.Background(),
				task.ID,
				"acceptor",
				strings.TrimSpace(head.Stdout),
				"https://example.test/pull/99",
			)
			require.ErrorContains(t, err, test.match)
			current, err := service.GetTask(task.ID)
			require.NoError(t, err)
			require.Empty(t, current.CommitSHA)
			require.Empty(t, current.PullRequestURL)
		})
	}
}

func validImportedEvidence(task Task) ImportExecutionEvidenceInput {
	started := time.Now().UTC().Add(-time.Minute)
	finished := started.Add(30 * time.Second)
	verified := finished.Add(time.Second)
	diff := strings.Join([]string{
		"diff --git a/README.md b/README.md",
		"--- a/README.md",
		"+++ b/README.md",
		"@@ -1 +1 @@",
		"-# initial",
		"+# changed",
		"",
	}, "\n")
	return ImportExecutionEvidenceInput{
		TaskID:              task.ID,
		ProjectID:           task.ProjectID,
		TaskRevision:        task.Compile.Revision,
		ExecutionBundleHash: task.Compile.ExecutionBundleHash,
		DiffPatch:           diff,
		Evidence: ImportedExecutionEvidence{
			Source:              "workstation",
			ExecutionPackSHA256: strings.Repeat("a", 64),
			RunnerID:            "runner-alice",
			LeaseID:             "lease-1",
			Attempt:             1,
			Outcome:             "completed",
			Summary:             "Implemented the frozen task.",
			StartedAt:           started,
			FinishedAt:          finished,
			BaseCommit:          task.Compile.BaseCommit,
			HeadCommit:          task.Compile.BaseCommit,
			Branch:              "goclaw/test-r1",
			ChangedFiles:        []string{"README.md"},
			DiffSHA256:          sha256Bytes([]byte(diff)),
			Checks: []ImportedEvidenceCheck{{
				Name:   "git diff check",
				Argv:   []string{"git", "diff", "--check"},
				Passed: true,
			}, {
				Name:   "runner-setup",
				Passed: true,
			}, {
				Name:   "codex-exec",
				Passed: true,
			}, {
				Name:   "scope-policy",
				Passed: true,
			}, {
				Name:   "no-automatic-commit",
				Passed: true,
			}},
			Artifacts: []ImportedEvidenceArtifact{{
				Name:      "codex-events",
				URI:       "goclaw-local://runner-alice/task/events",
				SHA256:    strings.Repeat("b", 64),
				SizeBytes: 128,
			}},
			TraceIDs:           []string{"trace-1"},
			KeyID:              strings.Repeat("c", 64),
			SignatureAlgorithm: "hmac-sha256",
			BundleSHA256:       strings.Repeat("d", 64),
			Signature:          strings.Repeat("ef", 32),
			VerifiedAt:         verified,
			VerifiedBy:         "workstation-control-plane",
		},
	}
}
