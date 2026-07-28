package workstation

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestAuditRepositoryGitConfigurationRejectsExecutionHooks(t *testing.T) {
	requireGit(t)
	gitCommand, err := findRunnerCommand("git")
	if err != nil {
		t.Skip(err)
	}
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "hooks", key: "core.hooksPath", value: "/tmp/hooks"},
		{name: "fsmonitor", key: "core.fsmonitor", value: "/tmp/monitor"},
		{name: "filter", key: "filter.pilot.process", value: "/tmp/filter"},
		{name: "include", key: "include.path", value: "/tmp/config"},
		{name: "credential", key: "credential.helper", value: "!/tmp/steal"},
		{name: "diff", key: "diff.pilot.command", value: "/tmp/diff"},
		{name: "merge", key: "merge.pilot.driver", value: "/tmp/merge"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, _ := createGitFixture(t)
			runGitFixture(t, repository, "config", "--local", test.key, test.value)
			err := auditRepositoryGitConfiguration(
				context.Background(),
				gitCommand,
				repository,
			)
			if err == nil || !strings.Contains(
				strings.ToLower(err.Error()),
				strings.Split(strings.ToLower(test.key), ".")[0],
			) {
				t.Fatalf("unsafe config %s was not rejected: %v", test.key, err)
			}
		})
	}
}

func TestAuditGitAttributesAtCommitRejectsExternalDrivers(t *testing.T) {
	requireGit(t)
	gitCommand, err := findRunnerCommand("git")
	if err != nil {
		t.Skip(err)
	}
	repository, _ := createGitFixture(t)
	if err := os.WriteFile(
		repository+"/.gitattributes",
		[]byte("*.go filter=pilot\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	runGitFixture(t, repository, "add", ".gitattributes")
	runGitFixture(t, repository, "commit", "-m", "add unsafe attributes")
	commit := strings.TrimSpace(
		runGitFixture(t, repository, "rev-parse", "HEAD"),
	)
	err = auditGitAttributesAtCommit(
		context.Background(),
		gitCommand,
		repository,
		commit,
	)
	if err == nil || !strings.Contains(err.Error(), "filter=pilot") {
		t.Fatalf("unsafe attributes were not rejected: %v", err)
	}
}

func TestUnsafeGitAttributeAllowsOrdinaryAttributes(t *testing.T) {
	if got := unsafeGitAttribute("*.go text eol=lf\n*.png binary\n"); got != "" {
		t.Fatalf("ordinary attributes rejected: %q", got)
	}
	for _, content := range []string{
		"*.go filter=format",
		"*.txt diff=external",
		"*.bin merge=driver",
		"*.txt working-tree-encoding=UTF-16",
	} {
		if got := unsafeGitAttribute(content); got == "" {
			t.Fatalf("unsafe attribute accepted: %q", content)
		}
	}
}
