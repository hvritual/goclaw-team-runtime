package config

import (
	"testing"

	"github.com/smallnest/goclaw/ouroboros"
)

func TestValidateOuroborosSafetyBounds(t *testing.T) {
	valid := ouroboros.Config{
		Enabled:                    true,
		Root:                       t.TempDir(),
		AmbiguityThreshold:         0.20,
		ConvergenceThreshold:       0.95,
		RequiredReadyStreak:        2,
		MaxGenerations:             30,
		ConsensusReviewers:         3,
		MaxQuestionsPerRound:       5,
		MaxContextBytes:            128 * 1024,
		MaxOutputTokens:            12000,
		AssessmentReviewers:        2,
		AssessmentMaxSpread:        0.15,
		AssessmentGrayZone:         0.03,
		ConsensusMaxSpread:         0.25,
		EvaluationHistoryWindow:    5,
		RequiredPassingEvaluations: 2,
		MaxSessionModelCalls:       120,
		MaxSessionModelTokens:      2_000_000,
	}
	validator := NewValidator(true)
	if err := validator.validateOuroboros(&Config{Ouroboros: valid}); err != nil {
		t.Fatalf("expected valid Ouroboros config, got %v", err)
	}

	invalid := valid
	invalid.AmbiguityThreshold = 1
	if err := validator.validateOuroboros(&Config{Ouroboros: invalid}); err == nil {
		t.Fatal("ambiguity threshold of 1 must be rejected")
	}

	invalid = valid
	invalid.Root = "relative/runtime"
	if err := validator.validateOuroboros(&Config{Ouroboros: invalid}); err == nil {
		t.Fatal("relative runtime root must be rejected")
	}
}
