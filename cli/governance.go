package cli

import (
	"os"

	"github.com/smallnest/goclaw/governance"
	"github.com/spf13/cobra"
)

var (
	decisionCounterargument string
	decisionEvidence        []string
)

func cliReview(
	policy governance.Config,
	reviewer, role, rationale string,
) (governance.Review, error) {
	review, err := governance.ResolveReviewer(policy, governance.Credential{
		ReviewerID: reviewer,
		Token:      os.Getenv("GOCLAW_REVIEWER_TOKEN"),
		Source:     "local-cli",
	}, role)
	if err != nil {
		return governance.Review{}, err
	}
	review.Rationale = rationale
	review.Counterargument = decisionCounterargument
	review.EvidenceRefs = append([]string(nil), decisionEvidence...)
	return review, nil
}

func addDecisionFlags(command *cobra.Command) {
	command.Flags().StringVar(
		&decisionCounterargument,
		"counterargument",
		"",
		"Strongest reason this approval could be wrong",
	)
	command.Flags().StringSliceVar(
		&decisionEvidence,
		"evidence-ref",
		nil,
		"Evidence reference; repeat as needed",
	)
}
