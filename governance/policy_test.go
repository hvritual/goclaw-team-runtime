package governance

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestAuthenticatedReviewerAndSeparationOfDuties(t *testing.T) {
	sum := sha256.Sum256([]byte("secret-token"))
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.RequireAuthenticatedReviewers = true
	cfg.RequireCounterargument = true
	cfg.ForbidSelfApproval = true
	cfg.Reviewers = map[string]ReviewerConfig{
		"alice-spec": {
			TokenSHA256: hex.EncodeToString(sum[:]),
			Roles:       []string{RoleSeedApprove},
			TeamUserID:  "alice",
		},
	}
	review, err := ResolveReviewer(cfg, Credential{
		ReviewerID: "alice",
		Token:      "secret-token",
		Source:     "gateway",
	}, RoleSeedApprove)
	if err != nil {
		t.Fatal(err)
	}
	review.Rationale = "The scope and deterministic evidence plan are acceptable."
	review.Counterargument = "The plan may still underestimate integration risk."
	if err := ValidateApproval(cfg, review, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateApproval(cfg, review, "alice"); err == nil {
		t.Fatal("self approval must be rejected")
	}
	if _, err := ResolveReviewer(cfg, Credential{
		ReviewerID: "alice",
		Token:      "wrong",
		Source:     "gateway",
	}, RoleSeedApprove); err == nil {
		t.Fatal("wrong reviewer token must be rejected")
	}
}
