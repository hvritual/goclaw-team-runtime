package config

import (
	"strings"
	"testing"

	"github.com/smallnest/goclaw/governance"
)

func TestGovernanceValidationRejectsUnknownRoleAndDeadlockedQuorum(t *testing.T) {
	digest := strings.Repeat("1", 64)
	cfg := &Config{
		Governance: governance.Config{
			Enabled:                       true,
			RequireAuthenticatedReviewers: true,
			MinRationaleRunes:             12,
			SeedApprovalQuorum:            1,
			HighRiskApprovalQuorum:        2,
			EvolutionApprovalQuorum:       1,
			HarnessApprovalQuorum:         1,
			MinDistinctTaskReviewers:      2,
			MaxTaskReviewKindsPerReviewer: 2,
			Reviewers: map[string]governance.ReviewerConfig{
				"alice": {
					TokenSHA256: digest,
					Roles:       []string{"typo_role"},
				},
			},
		},
	}
	validator := NewValidator(false)
	if err := validator.validateGovernance(cfg); err == nil ||
		!strings.Contains(err.Error(), "unknown role") {
		t.Fatalf("expected unknown-role validation error, got %v", err)
	}

	cfg.Ouroboros.Enabled = true
	cfg.Governance.Reviewers["alice"] = governance.ReviewerConfig{
		TokenSHA256: digest,
		Roles:       []string{governance.RoleAny},
	}
	if err := validator.validateGovernance(cfg); err == nil ||
		!strings.Contains(err.Error(), "requires at least 2 configured reviewer") {
		t.Fatalf("expected impossible high-risk quorum error, got %v", err)
	}
}

func TestGovernanceValidationRejectsDuplicateTeamUserBinding(t *testing.T) {
	cfg := &Config{
		Governance: governance.Config{
			Enabled:                       true,
			RequireAuthenticatedReviewers: true,
			MinRationaleRunes:             12,
			SeedApprovalQuorum:            1,
			HighRiskApprovalQuorum:        1,
			EvolutionApprovalQuorum:       1,
			HarnessApprovalQuorum:         1,
			MinDistinctTaskReviewers:      1,
			MaxTaskReviewKindsPerReviewer: 1,
			Reviewers: map[string]governance.ReviewerConfig{
				"alice-spec": {
					TokenSHA256: strings.Repeat("1", 64),
					Roles:       []string{governance.RoleAny},
					TeamUserID:  "Alice",
				},
				"alice-risk": {
					TokenSHA256: strings.Repeat("2", 64),
					Roles:       []string{governance.RoleAny},
					TeamUserID:  "alice",
				},
			},
		},
	}
	if err := NewValidator(false).validateGovernance(cfg); err == nil ||
		!strings.Contains(err.Error(), "same team_user_id") {
		t.Fatalf("expected duplicate team user binding error, got %v", err)
	}
}
