package ouroboros

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type interviewAssessmentResult struct {
	index    int
	role     string
	output   interviewModelOutput
	response ModelResponse
	err      error
}

func (s *Service) runInterviewAssessors(
	ctx context.Context,
	session Session,
) ([]interviewAssessmentResult, error) {
	count := s.cfg.AssessmentReviewers
	if count < 1 {
		count = 1
	}
	if err := s.ensureModelBudget(session, count*2); err != nil {
		return nil, err
	}
	roles := []string{"primary", "skeptical", "operator", "risk", "status-quo"}
	results := make(chan interviewAssessmentResult, count)
	var group sync.WaitGroup
	for index := 0; index < count; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			role := roles[index%len(roles)]
			var output interviewModelOutput
			payload := interviewPayload(session)
			payload["assessment_role"] = role
			response, err := s.invokeModel(
				ctx,
				fmt.Sprintf("ouroboros.interview.%s", role),
				interviewSystemPrompt(s.cfg.MaxQuestionsPerRound, session.Brownfield, role),
				payload,
				s.assessmentModel(index),
				&output,
			)
			results <- interviewAssessmentResult{
				index: index, role: role, output: output, response: response, err: err,
			}
		}(index)
	}
	group.Wait()
	close(results)
	ordered := make([]interviewAssessmentResult, count)
	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		ordered[result.index] = result
	}
	return ordered, nil
}

func (s *Service) assessmentModel(index int) string {
	if len(s.cfg.AssessmentModels) == 0 {
		return s.cfg.Model
	}
	return s.cfg.AssessmentModels[index%len(s.cfg.AssessmentModels)]
}

func aggregateInterviewAssessments(
	results []interviewAssessmentResult,
	session Session,
	roundNumber, readyStreak int,
	cfg Config,
) (interviewModelOutput, AmbiguityAssessment, error) {
	var merged interviewModelOutput
	if len(results) == 0 {
		return merged, AmbiguityAssessment{}, fmt.Errorf("no interview assessment was produced")
	}
	merged = results[0].output
	var (
		goal, constraint, success, contextScore        float64
		goalWhy, constraintWhy, successWhy, contextWhy []string
		votes                                          []AssessmentVote
		minimums                                       = map[Dimension]float64{
			DimensionGoal: math.Inf(1), DimensionConstraint: math.Inf(1),
			DimensionSuccess: math.Inf(1), DimensionContext: math.Inf(1),
		}
		maximums = map[Dimension]float64{
			DimensionGoal: math.Inf(-1), DimensionConstraint: math.Inf(-1),
			DimensionSuccess: math.Inf(-1), DimensionContext: math.Inf(-1),
		}
		overallMinimum = math.Inf(1)
		overallMaximum = math.Inf(-1)
		models         = make(map[string]struct{})
	)
	for index, result := range results {
		output := result.output
		goal += output.Goal.Clarity
		constraint += output.Constraint.Clarity
		success += output.Success.Clarity
		contextScore += output.Context.Clarity
		goalWhy = append(goalWhy, result.role+": "+output.Goal.Justification)
		constraintWhy = append(constraintWhy, result.role+": "+output.Constraint.Justification)
		successWhy = append(successWhy, result.role+": "+output.Success.Justification)
		contextWhy = append(contextWhy, result.role+": "+output.Context.Justification)
		input := clarityInput{
			Goal: output.Goal.Clarity, Constraint: output.Constraint.Clarity,
			Success: output.Success.Clarity, Context: output.Context.Clarity,
		}
		voteAssessment, err := calculateAmbiguity(
			input, session.Brownfield, cfg.AmbiguityThreshold,
			roundNumber, 0, 1, output.Summary, output.Unresolved,
		)
		if err != nil {
			return merged, AmbiguityAssessment{}, fmt.Errorf("assessor %d: %w", index+1, err)
		}
		dimensions := map[Dimension]float64{
			DimensionGoal:       output.Goal.Clarity,
			DimensionConstraint: output.Constraint.Clarity,
			DimensionSuccess:    output.Success.Clarity,
		}
		if session.Brownfield {
			dimensions[DimensionContext] = output.Context.Clarity
		}
		for dimension, value := range dimensions {
			if value < minimums[dimension] {
				minimums[dimension] = value
			}
			if value > maximums[dimension] {
				maximums[dimension] = value
			}
		}
		if voteAssessment.Overall < overallMinimum {
			overallMinimum = voteAssessment.Overall
		}
		if voteAssessment.Overall > overallMaximum {
			overallMaximum = voteAssessment.Overall
		}
		votes = append(votes, AssessmentVote{
			Role: result.role, Model: result.response.Model,
			Overall: voteAssessment.Overall, Dimensions: dimensions,
			Summary: strings.TrimSpace(output.Summary),
		})
		if model := strings.TrimSpace(result.response.Model); model != "" {
			models[model] = struct{}{}
		}
		if index > 0 {
			merged.Questions = append(merged.Questions, output.Questions...)
			merged.Assumptions = append(merged.Assumptions, output.Assumptions...)
			merged.Unresolved = append(merged.Unresolved, output.Unresolved...)
			merged.Decisions = append(merged.Decisions, output.Decisions...)
			merged.ProblemFrames = append(merged.ProblemFrames, output.ProblemFrames...)
			merged.StakeholderClaims = append(merged.StakeholderClaims, output.StakeholderClaims...)
			merged.DecisionConflicts = append(merged.DecisionConflicts, output.DecisionConflicts...)
		}
	}
	count := float64(len(results))
	assessment, err := calculateAmbiguity(
		clarityInput{
			Goal: goal / count, GoalWhy: strings.Join(goalWhy, " | "),
			Constraint: constraint / count, ConstraintWhy: strings.Join(constraintWhy, " | "),
			Success: success / count, SuccessWhy: strings.Join(successWhy, " | "),
			Context: contextScore / count, ContextWhy: strings.Join(contextWhy, " | "),
		},
		session.Brownfield, cfg.AmbiguityThreshold, roundNumber, readyStreak,
		cfg.RequiredReadyStreak, merged.Summary, cleanStrings(merged.Unresolved),
	)
	if err != nil {
		return merged, AmbiguityAssessment{}, err
	}
	spread := overallMaximum - overallMinimum
	for dimension, minimum := range minimums {
		if math.IsInf(minimum, 0) {
			continue
		}
		if value := maximums[dimension] - minimum; value > spread {
			spread = value
		}
	}
	assessment.AssessorVotes = votes
	assessment.ScoreSpread = round4(spread)
	assessment.DistinctModels = len(models)
	assessment.CalibrationVersion = CalibrationVersion
	assessment.GrayZone = math.Abs(assessment.Overall-cfg.AmbiguityThreshold) <= cfg.AssessmentGrayZone
	correlatedAssessors := len(results) > 1 && assessment.DistinctModels < 2
	if correlatedAssessors {
		assessment.Unresolved = cleanStrings(append(
			assessment.Unresolved,
			"all requirement assessors used the same model; correlated judgments require human disposition",
		))
	}
	if assessment.ScoreSpread > cfg.AssessmentMaxSpread ||
		assessment.GrayZone ||
		correlatedAssessors {
		assessment.HumanDecisionRequired = true
		assessment.Ready = false
		assessment.ReadyStreak = 0
	}
	return merged, assessment, nil
}

func normalizeProblemFrames(values []ProblemFrame, round int) []ProblemFrame {
	seen := make(map[string]struct{})
	result := make([]ProblemFrame, 0, len(values))
	for index, frame := range values {
		frame.Perspective = strings.TrimSpace(frame.Perspective)
		frame.Problem = strings.TrimSpace(frame.Problem)
		frame.ExpectedBenefit = strings.TrimSpace(frame.ExpectedBenefit)
		frame.CostOfInaction = strings.TrimSpace(frame.CostOfInaction)
		frame.Risks = cleanStrings(frame.Risks)
		frame.Assumptions = cleanStrings(frame.Assumptions)
		if frame.Problem == "" {
			continue
		}
		key := normalizeText(frame.Perspective + frame.Problem)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if validateID(frame.ID) != nil {
			frame.ID = fmt.Sprintf("frame-%d-%d", round, index+1)
		}
		result = append(result, frame)
	}
	return result
}

func normalizeStakeholderClaims(values []StakeholderClaim, round int, now time.Time) []StakeholderClaim {
	seen := make(map[string]struct{})
	result := make([]StakeholderClaim, 0, len(values))
	for index, claim := range values {
		claim.Stakeholder = strings.TrimSpace(claim.Stakeholder)
		claim.Statement = strings.TrimSpace(claim.Statement)
		claim.Source = strings.TrimSpace(claim.Source)
		if claim.Statement == "" || claim.Source == "" {
			continue
		}
		key := normalizeText(claim.Stakeholder + claim.Statement)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if validateID(claim.ID) != nil {
			claim.ID = fmt.Sprintf("claim-%d-%d", round, index+1)
		}
		switch claim.Status {
		case "asserted", "contested", "resolved":
		default:
			claim.Status = "asserted"
		}
		claim.Round = round
		claim.CreatedAt = now
		result = append(result, claim)
	}
	return result
}

func normalizeDecisionConflicts(values []DecisionConflict, round int) []DecisionConflict {
	seen := make(map[string]struct{})
	result := make([]DecisionConflict, 0, len(values))
	for index, conflict := range values {
		conflict.Description = strings.TrimSpace(conflict.Description)
		conflict.ClaimIDs = cleanStrings(conflict.ClaimIDs)
		if conflict.Description == "" {
			continue
		}
		key := normalizeText(conflict.Description)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if validateID(conflict.ID) != nil {
			conflict.ID = fmt.Sprintf("conflict-%d-%d", round, index+1)
		}
		if conflict.Status != "resolved" {
			conflict.Status = "open"
			conflict.Resolution = ""
			conflict.ResolvedBy = ""
			conflict.ResolvedAt = nil
		}
		result = append(result, conflict)
	}
	return result
}

func mergeProblemFrames(existing, proposed []ProblemFrame) []ProblemFrame {
	byKey := make(map[string]ProblemFrame)
	for _, frame := range append(append([]ProblemFrame(nil), existing...), proposed...) {
		byKey[normalizeText(frame.Perspective+frame.Problem)] = frame
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]ProblemFrame, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func mergeStakeholderClaims(existing, proposed []StakeholderClaim) []StakeholderClaim {
	byKey := make(map[string]StakeholderClaim)
	for _, claim := range existing {
		byKey[normalizeText(claim.Stakeholder+claim.Statement)] = claim
	}
	for _, claim := range proposed {
		key := normalizeText(claim.Stakeholder + claim.Statement)
		if prior, ok := byKey[key]; ok && prior.Status == "resolved" {
			continue
		}
		byKey[key] = claim
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]StakeholderClaim, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func mergeDecisionConflicts(existing, proposed []DecisionConflict) []DecisionConflict {
	byKey := make(map[string]DecisionConflict)
	for _, conflict := range existing {
		byKey[normalizeText(conflict.Description)] = conflict
	}
	for _, conflict := range proposed {
		key := normalizeText(conflict.Description)
		if prior, ok := byKey[key]; ok && prior.Status == "resolved" {
			continue
		}
		byKey[key] = conflict
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]DecisionConflict, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func hasOpenConflict(conflicts []DecisionConflict) bool {
	for _, conflict := range conflicts {
		if conflict.Status != "resolved" {
			return true
		}
	}
	return false
}

func dedupeQuestions(values []Question) []Question {
	seen := make(map[string]struct{}, len(values))
	result := make([]Question, 0, len(values))
	for _, question := range values {
		key := normalizeText(question.Text)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		// Model-provided IDs often collide across assessors. The normalizer
		// assigns deterministic round-local IDs after aggregation.
		question.ID = ""
		result = append(result, question)
	}
	return result
}

func (s *Service) ensureModelBudget(session Session, requiredCalls int) error {
	if s.cfg.MaxSessionModelCalls > 0 &&
		session.ModelUsage.Calls+requiredCalls > s.cfg.MaxSessionModelCalls {
		return fmt.Errorf(
			"session model-call budget would be exceeded: used=%d required=%d maximum=%d",
			session.ModelUsage.Calls, requiredCalls, s.cfg.MaxSessionModelCalls,
		)
	}
	if s.cfg.MaxSessionModelTokens > 0 &&
		session.ModelUsage.TotalTokens >= s.cfg.MaxSessionModelTokens {
		return fmt.Errorf(
			"session model-token budget is exhausted: used=%d maximum=%d",
			session.ModelUsage.TotalTokens, s.cfg.MaxSessionModelTokens,
		)
	}
	return nil
}
