package gateway

import (
	"strings"

	"github.com/smallnest/goclaw/governance"
)

func (h *Handler) humanReview(
	sessionID string,
	params map[string]interface{},
	role string,
) (governance.Review, error) {
	policy := governance.DefaultConfig()
	if h.cfg != nil {
		policy = h.cfg.Governance
	}
	reviewerID := ""
	if h.teamSvc != nil {
		var err error
		reviewerID, err = h.principalID(sessionID)
		if err != nil {
			return governance.Review{}, err
		}
	} else {
		reviewerID = stringParam(params["reviewer_id"])
		if reviewerID == "" {
			reviewerID = stringParam(params["reviewer"])
		}
	}
	if reviewerID == "" && !policy.RequireAuthenticatedReviewers {
		reviewerID = sessionID
	}
	review, err := governance.ResolveReviewer(policy, governance.Credential{
		ReviewerID: reviewerID,
		Token:      stringParam(params["reviewer_token"]),
		Source:     "gateway",
	}, role)
	if err != nil {
		return governance.Review{}, err
	}
	review.Rationale = stringParam(params["rationale"])
	if review.Rationale == "" {
		review.Rationale = stringParam(params["comment"])
	}
	review.Counterargument = stringParam(params["counterargument"])
	review.EvidenceRefs = stringSliceParam(params["evidence_refs"])
	review.Rationale = strings.TrimSpace(review.Rationale)
	return review, nil
}
