package requirement

import (
	"errors"
	"testing"
	"time"
)

func TestRequirementVersionAndCoverage(t *testing.T) {
	now := time.Now()
	value, err := New("requirement-1", "workspace-1", "project-1", " First ", []string{"issue-1", "issue-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if value.CurrentVersion != 1 || value.ApprovalStatus != "draft" || value.CoverageStatus != "covered" || len(value.IssueIDs) != 1 {
		t.Fatalf("Requirement = %+v", value)
	}
	value, err = value.NextVersion("Second", nil, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if value.CurrentVersion != 2 || value.CoverageStatus != "uncovered" || value.ApprovalStatus != "draft" {
		t.Fatalf("next Requirement = %+v", value)
	}
	if _, err := NewVersion("version-2", value.ID, value.CurrentVersion, " ", now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank content error = %v", err)
	}
}
