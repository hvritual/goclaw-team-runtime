package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/governance"
)

func TestGatewayHumanReviewAuthenticatesReviewerIdentity(t *testing.T) {
	sum := sha256.Sum256([]byte("reviewer-secret"))
	policy := governance.DefaultConfig()
	policy.Enabled = true
	policy.RequireAuthenticatedReviewers = true
	policy.Reviewers = map[string]governance.ReviewerConfig{
		"alice": {
			TokenSHA256: hex.EncodeToString(sum[:]),
			Roles:       []string{governance.RoleEvaluationResolve},
		},
	}
	handler := &Handler{cfg: &config.Config{Governance: policy}}
	if _, err := handler.humanReview("untrusted-session", map[string]interface{}{
		"reviewer_token": "reviewer-secret",
		"rationale":      "sufficient rationale",
	}, governance.RoleEvaluationResolve); err == nil {
		t.Fatal("gateway session id must not become an authenticated reviewer identity")
	}
	if _, err := handler.humanReview("untrusted-session", map[string]interface{}{
		"reviewer_id":    "alice",
		"reviewer_token": "wrong",
		"rationale":      "sufficient rationale",
	}, governance.RoleEvaluationResolve); err == nil {
		t.Fatal("wrong reviewer token must be rejected at the gateway boundary")
	}
	review, err := handler.humanReview("untrusted-session", map[string]interface{}{
		"reviewer_id":     "alice",
		"reviewer_token":  "reviewer-secret",
		"rationale":       "The evidence bundle resolves the disputed model judgment.",
		"counterargument": "The shared provider may still introduce correlated judgment.",
		"evidence_refs":   []interface{}{"artifact:diff", "artifact:test-log"},
	}, governance.RoleEvaluationResolve)
	if err != nil {
		t.Fatal(err)
	}
	if !review.Authenticated || review.ReviewerID != "alice" ||
		len(review.EvidenceRefs) != 2 {
		t.Fatalf("unexpected authenticated review: %#v", review)
	}
}
