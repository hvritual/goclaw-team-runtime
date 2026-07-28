package gateway

import (
	"strings"
	"testing"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/workstation"
)

func TestControlConsistencyAllowsEmptyProjectAndBlocksOrphanQueue(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	development, err := dev.NewService(dev.Config{
		Root: t.TempDir(), RepoPath: fixture.repoDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	queue, err := workstation.NewService(workstation.Config{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{
		teamSvc: &fixture.service, devSvc: development, runnerSvc: queue,
	}
	report, err := handler.controlConsistency(fixture.alice.ID, fixture.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Consistent || !report.EnqueueAllowed || !report.AcceptAllowed {
		t.Fatalf("empty project should be consistent: %+v", report)
	}

	_, err = queue.Enqueue(workstation.EnqueueRequest{
		ID:                   "wstask-orphan",
		IdempotencyKey:       "orphan-enqueue",
		ProjectID:            fixture.project.ID,
		RequiredCapabilities: []string{"goclaw-runtime-linux-v1"},
		ExecutionPack: workstation.ExecutionPack{
			ProjectID:        fixture.project.ID,
			RepositoryID:     fixture.repo.ID,
			BaseCommit:       strings.Repeat("a", 40),
			Prompt:           "orphan fixture",
			Verification:     []workstation.CommandSpec{{Name: "fixture", Argv: []string{"true"}}},
			Metadata:         map[string]string{"assignee_id": fixture.bob.ID},
			PolicyBundleHash: fixture.policy.Hash,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err = handler.controlConsistency(fixture.alice.ID, fixture.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.Consistent || report.EnqueueAllowed || report.AcceptAllowed {
		t.Fatalf("orphan queue must fail closed: %+v", report)
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Code == "queue.unbound" &&
			finding.Severity == consistencyCritical {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing queue.unbound finding: %+v", report.Findings)
	}
	if err := handler.requireControlConsistency(
		fixture.alice.ID,
		fixture.project.ID,
	); err == nil {
		t.Fatal("expected consistency gate to block")
	}
}

func TestControlConsistencyFailsClosedWithoutAllServices(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	handler := &Handler{teamSvc: &fixture.service}
	report, err := handler.controlConsistency(fixture.alice.ID, fixture.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.Consistent || len(report.Findings) != 1 ||
		report.Findings[0].Code != "services.incomplete" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if err := handler.requireControlConsistency(
		fixture.alice.ID,
		fixture.project.ID,
	); err == nil {
		t.Fatal("expected incomplete services to block")
	}
}
