package ouroboros

import "testing"

func TestAssessorDisagreementFailsClosedIntoHumanDecision(t *testing.T) {
	output := func(clarity float64, summary string) interviewModelOutput {
		return interviewModelOutput{
			Summary:    summary,
			Goal:       modelDimension{Clarity: clarity, Justification: summary},
			Constraint: modelDimension{Clarity: clarity, Justification: summary},
			Success:    modelDimension{Clarity: clarity, Justification: summary},
			Context:    modelDimension{Clarity: clarity, Justification: summary},
		}
	}
	_, assessment, err := aggregateInterviewAssessments(
		[]interviewAssessmentResult{
			{
				role:     "primary",
				output:   output(0.95, "requirements appear explicit"),
				response: ModelResponse{Model: "model-a"},
			},
			{
				role:     "skeptical",
				output:   output(0.55, "constraints remain underspecified"),
				response: ModelResponse{Model: "model-b"},
			},
		},
		Session{Brownfield: false},
		1,
		0,
		Config{
			AmbiguityThreshold:  0.20,
			RequiredReadyStreak: 1,
			AssessmentMaxSpread: 0.15,
			AssessmentGrayZone:  0.03,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !assessment.HumanDecisionRequired || assessment.Ready {
		t.Fatalf("assessor disagreement must fail closed: %#v", assessment)
	}
	if assessment.ScoreSpread <= 0.15 || len(assessment.AssessorVotes) != 2 {
		t.Fatalf("disagreement evidence was not preserved: %#v", assessment)
	}
}

func TestCorrelatedRequirementAssessorsRequireHumanDisposition(t *testing.T) {
	output := interviewModelOutput{
		Summary:    "all dimensions appear clear",
		Goal:       modelDimension{Clarity: 0.95, Justification: "clear"},
		Constraint: modelDimension{Clarity: 0.95, Justification: "clear"},
		Success:    modelDimension{Clarity: 0.95, Justification: "clear"},
		Context:    modelDimension{Clarity: 0.95, Justification: "clear"},
	}
	_, assessment, err := aggregateInterviewAssessments(
		[]interviewAssessmentResult{
			{role: "primary", output: output, response: ModelResponse{Model: "same-model"}},
			{role: "skeptical", output: output, response: ModelResponse{Model: "same-model"}},
		},
		Session{},
		1,
		0,
		Config{
			AmbiguityThreshold:  0.20,
			RequiredReadyStreak: 1,
			AssessmentMaxSpread: 0.15,
			AssessmentGrayZone:  0.03,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !assessment.HumanDecisionRequired || assessment.DistinctModels != 1 {
		t.Fatalf("correlated assessors must not masquerade as independent: %#v", assessment)
	}
}
