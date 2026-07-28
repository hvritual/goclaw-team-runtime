package governance

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func ResolveReviewer(cfg Config, credential Credential, role string) (Review, error) {
	id := strings.TrimSpace(credential.ReviewerID)
	if id == "" {
		return Review{}, errors.New("reviewer identity is required")
	}
	review := Review{
		ReviewerID: id,
		Role:       role,
		Source:     strings.TrimSpace(credential.Source),
		CreatedAt:  time.Now().UTC(),
	}
	if !cfg.Enabled || !cfg.RequireAuthenticatedReviewers {
		review.Authenticated = credential.Source == "local-cli"
		return review, nil
	}
	configured, ok := cfg.Reviewers[id]
	if !ok {
		for _, candidate := range cfg.Reviewers {
			if strings.EqualFold(strings.TrimSpace(candidate.TeamUserID), id) {
				configured = candidate
				ok = true
				break
			}
		}
		if !ok {
			return Review{}, fmt.Errorf("reviewer %q is not registered", id)
		}
	}
	expected, err := hex.DecodeString(strings.TrimSpace(configured.TokenSHA256))
	if err != nil || len(expected) != sha256.Size {
		return Review{}, fmt.Errorf("reviewer %q has an invalid token_sha256 configuration", id)
	}
	actual := sha256.Sum256([]byte(credential.Token))
	if subtle.ConstantTimeCompare(actual[:], expected) != 1 {
		return Review{}, errors.New("reviewer authentication failed")
	}
	if !hasRole(configured.Roles, role) {
		return Review{}, fmt.Errorf("reviewer %q does not have role %q", id, role)
	}
	review.Authenticated = true
	return review, nil
}

func ValidateApproval(cfg Config, review Review, creators ...string) error {
	review.ReviewerID = strings.TrimSpace(review.ReviewerID)
	review.Rationale = strings.TrimSpace(review.Rationale)
	review.Counterargument = strings.TrimSpace(review.Counterargument)
	if review.ReviewerID == "" {
		return errors.New("reviewer identity is required")
	}
	if cfg.Enabled && cfg.RequireAuthenticatedReviewers && !review.Authenticated {
		return errors.New("authenticated reviewer is required")
	}
	if cfg.RequireRationale || cfg.Enabled {
		minimum := cfg.MinRationaleRunes
		if minimum < 1 {
			minimum = 1
		}
		if utf8.RuneCountInString(review.Rationale) < minimum {
			return fmt.Errorf("review rationale must contain at least %d characters", minimum)
		}
	}
	if cfg.Enabled && cfg.RequireCounterargument && review.Counterargument == "" {
		return errors.New("approval counterargument is required")
	}
	if cfg.Enabled && cfg.ForbidSelfApproval {
		for _, creator := range creators {
			if SameActor(review.ReviewerID, creator) {
				return fmt.Errorf("reviewer %q cannot approve their own proposal", review.ReviewerID)
			}
		}
	}
	return nil
}

func ValidateRole(review Review, expected string) error {
	if strings.TrimSpace(review.Role) != expected {
		return fmt.Errorf("review role %q does not satisfy required role %q", review.Role, expected)
	}
	return nil
}

func ValidateDecision(cfg Config, review Review, decision string, creators ...string) error {
	if strings.EqualFold(strings.TrimSpace(decision), "approved") {
		return ValidateApproval(cfg, review, creators...)
	}
	rejectionPolicy := cfg
	rejectionPolicy.RequireCounterargument = false
	return ValidateApproval(rejectionPolicy, review, creators...)
}

func ToDecision(review Review, decision string) DecisionRecord {
	when := review.CreatedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}
	return DecisionRecord{
		ReviewerID:      strings.TrimSpace(review.ReviewerID),
		Decision:        strings.TrimSpace(decision),
		Rationale:       strings.TrimSpace(review.Rationale),
		Counterargument: strings.TrimSpace(review.Counterargument),
		EvidenceRefs:    cleanStrings(review.EvidenceRefs),
		Role:            strings.TrimSpace(review.Role),
		Source:          strings.TrimSpace(review.Source),
		Authenticated:   review.Authenticated,
		CreatedAt:       when,
	}
}

func RequiredQuorum(cfg Config, highRisk bool, kind string) int {
	required := 1
	switch kind {
	case "seed":
		required = cfg.SeedApprovalQuorum
		if highRisk && cfg.HighRiskApprovalQuorum > required {
			required = cfg.HighRiskApprovalQuorum
		}
	case "evolution":
		required = cfg.EvolutionApprovalQuorum
	case "harness":
		required = cfg.HarnessApprovalQuorum
	}
	if required < 1 {
		return 1
	}
	return required
}

func DistinctApprovals(records []DecisionRecord) int {
	seen := make(map[string]struct{})
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.Decision), "approved") {
			seen[normalizeActor(record.ReviewerID)] = struct{}{}
		}
	}
	delete(seen, "")
	return len(seen)
}

func SameActor(left, right string) bool {
	left = normalizeActor(left)
	right = normalizeActor(right)
	return left != "" && left == right
}

func hasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == RoleAny || role == expected {
			return true
		}
	}
	return false
}

func normalizeActor(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cleanStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
