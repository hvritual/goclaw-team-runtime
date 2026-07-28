package ouroboros

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
	dev "github.com/smallnest/goclaw/orchestratorlite"
)

func (s *Service) EvaluateTask(
	ctx context.Context,
	id, actor string,
	task dev.Task,
	evidence dev.EvidencePackage,
	diff string,
) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if s.model == nil {
		return Session{}, errors.New("ouroboros model is not configured")
	}
	if session.Status != StatusCompiled &&
		session.Status != StatusEvaluated &&
		session.Status != StatusEvolutionPending {
		return Session{}, fmt.Errorf("session %s cannot be evaluated in status %s", id, session.Status)
	}
	if task.ID == "" || task.ProjectID != session.ProjectID {
		return Session{}, errors.New("development task does not belong to the Ouroboros project")
	}
	var seedHash string
	for _, compiled := range session.CompiledTasks {
		if compiled.TaskID == task.ID {
			seedHash = compiled.SeedHash
			break
		}
	}
	if seedHash == "" {
		return Session{}, fmt.Errorf("task %s was not compiled from session %s", task.ID, session.ID)
	}
	seed, err := s.loadSeedUnlocked(seedHash)
	if err != nil {
		return Session{}, err
	}
	mechanical := mechanicalEvaluation(task, evidence)
	evaluation := Evaluation{
		ID:         "evaluation-" + uuid.NewString(),
		SessionID:  session.ID,
		SeedHash:   seed.Hash,
		TaskID:     task.ID,
		Mechanical: mechanical,
		CreatedAt:  time.Now().UTC(),
	}
	if !mechanical.Passed {
		evaluation.Semantic = skippedStage("semantic", "mechanical gate failed")
		evaluation.Consensus = skippedStage("consensus", "mechanical gate failed")
		evaluation.Passed = false
		return s.recordEvaluationUnlocked(session, evaluation, actor)
	}
	reviewerCount := s.cfg.ConsensusReviewers
	if reviewerCount < 1 {
		reviewerCount = DefaultConsensusReviewers
	}
	if err := s.ensureModelBudget(session, (reviewerCount+1)*2); err != nil {
		return Session{}, err
	}

	payload := evaluationPayload(seed, task, evidence, diff)
	blindedPayload := blindedEvaluationPayload(seed, task, evidence, diff)
	var semanticOutput evaluationModelOutput
	semanticResponse, err := s.invokeModel(
		ctx,
		"ouroboros.evaluate.semantic",
		evaluationSystemPrompt("semantic acceptance"),
		payload,
		s.evaluationModel(0),
		&semanticOutput,
	)
	if err != nil {
		return Session{}, err
	}
	evaluation.Semantic = modelEvaluationStage(
		"semantic",
		"semantic-reviewer",
		semanticResponse.Model,
		semanticOutput,
		[]string{task.LastEvidence},
		s.cfg.CriticalFindingVeto,
		false,
	)
	addUsage(&evaluation.ModelUsage, semanticResponse.Usage)
	if !evaluation.Semantic.Passed {
		evaluation.Consensus = skippedStage("consensus", "semantic gate failed")
		evaluation.Passed = false
		return s.recordEvaluationUnlocked(session, evaluation, actor)
	}

	roles := []string{"contrarian", "architect", "evidence"}
	type reviewResult struct {
		index int
		stage EvaluationStage
		usage ModelUsage
		err   error
	}
	results := make(chan reviewResult, reviewerCount)
	var reviewers sync.WaitGroup
	for index := 0; index < reviewerCount; index++ {
		reviewers.Add(1)
		go func(index int) {
			defer reviewers.Done()
			role := roles[index%len(roles)]
			var output evaluationModelOutput
			response, callErr := s.invokeModel(
				ctx,
				fmt.Sprintf("ouroboros.evaluate.consensus.%d", index+1),
				evaluationSystemPrompt(role),
				blindedPayload,
				s.evaluationModel(index+1),
				&output,
			)
			result := reviewResult{index: index, usage: response.Usage, err: callErr}
			if callErr == nil {
				result.stage = modelEvaluationStage(
					"consensus_review",
					role+"-reviewer",
					response.Model,
					output,
					[]string{task.LastEvidence},
					s.cfg.CriticalFindingVeto,
					true,
				)
			}
			results <- result
		}(index)
	}
	reviewers.Wait()
	close(results)
	evaluation.Reviews = make([]EvaluationStage, reviewerCount)
	for result := range results {
		if result.err != nil {
			return Session{}, result.err
		}
		evaluation.Reviews[result.index] = result.stage
		addUsage(&evaluation.ModelUsage, result.usage)
	}
	var humanDecisionRequired bool
	evaluation.Consensus, evaluation.ScoreSpread, evaluation.DistinctModels, humanDecisionRequired =
		consensusStage(evaluation.Reviews, s.cfg)
	evaluation.HumanDecisionRequired = humanDecisionRequired
	evaluation.Passed = evaluation.Mechanical.Passed &&
		evaluation.Semantic.Passed &&
		evaluation.Consensus.Passed &&
		!evaluation.HumanDecisionRequired
	return s.recordEvaluationUnlocked(session, evaluation, actor)
}

func mechanicalEvaluation(task dev.Task, evidence dev.EvidencePackage) EvaluationStage {
	stage := EvaluationStage{
		Name:     "mechanical",
		Passed:   true,
		Score:    1,
		Summary:  "Deterministic EvidencePackage checks passed.",
		Reviewer: "go-core",
	}
	if task.LastEvidence != "" {
		stage.EvidenceRefs = append(stage.EvidenceRefs, task.LastEvidence)
	}
	if evidence.TaskID != task.ID {
		stage.Findings = append(stage.Findings, "EvidencePackage task_id does not match task")
	}
	if evidence.TaskRevision != task.Compile.Revision {
		stage.Findings = append(stage.Findings, "EvidencePackage task revision does not match task")
	}
	if task.LastGate == nil || !task.LastGate.Passed {
		stage.Findings = append(stage.Findings, "Go DoneGate has not passed")
	} else if task.LastGate.EvidenceSHA256 == "" {
		stage.Findings = append(stage.Findings, "DoneGate evidence hash is missing")
	}
	if !evidence.Policy.Passed {
		stage.Findings = append(stage.Findings, "scope and policy verification failed")
	}
	if len(evidence.Verification) == 0 {
		stage.Findings = append(stage.Findings, "no deterministic verification results were recorded")
	}
	for _, result := range evidence.Verification {
		if !result.Passed {
			stage.Findings = append(stage.Findings, fmt.Sprintf("verification %q failed", result.Name))
		}
	}
	falsifiers := make(map[string]dev.FalsifierResult, len(evidence.Falsifiers))
	for _, result := range evidence.Falsifiers {
		falsifiers[result.CriterionID] = result
	}
	for _, falsifier := range task.Goal.Falsifiers {
		result, ok := falsifiers[falsifier.CriterionID]
		if !ok || !result.Checked {
			stage.Findings = append(stage.Findings,
				fmt.Sprintf("falsifier %q has no checked evidence", falsifier.CriterionID))
		} else if result.Triggered {
			stage.Findings = append(stage.Findings,
				fmt.Sprintf("falsifier %q was triggered", falsifier.CriterionID))
		}
	}
	for _, prediction := range evidence.Predictions {
		if prediction.Due && (!prediction.Checked || !prediction.Satisfied) {
			stage.Findings = append(stage.Findings,
				fmt.Sprintf("due prediction %q is not satisfied", prediction.PredictionID))
		}
	}
	for _, check := range evidence.KillChecks {
		if !check.Evaluated {
			stage.Findings = append(stage.Findings,
				fmt.Sprintf("kill condition %q was not evaluated", check.ConditionID))
		} else if check.Triggered {
			stage.Findings = append(stage.Findings,
				fmt.Sprintf("kill condition %q was triggered", check.ConditionID))
		}
	}
	if !evidence.Review.Passed {
		stage.Findings = append(stage.Findings, "independent read-only review failed")
	}
	if evidence.DiffPath == "" {
		stage.Findings = append(stage.Findings, "diff evidence path is missing")
	}
	if len(stage.Findings) > 0 {
		stage.Passed = false
		stage.Score = 0
		stage.Summary = "Mechanical gate failed; semantic and consensus stages were not run."
	}
	return stage
}

func evaluationPayload(
	seed Seed,
	task dev.Task,
	evidence dev.EvidencePackage,
	diff string,
) map[string]any {
	type verificationSummary struct {
		Name       string   `json:"name"`
		Argv       []string `json:"argv"`
		Passed     bool     `json:"passed"`
		ExitCode   int      `json:"exit_code"`
		Stdout     string   `json:"stdout,omitempty"`
		Stderr     string   `json:"stderr,omitempty"`
		DurationMS int64    `json:"duration_ms"`
	}
	verifications := make([]verificationSummary, 0, len(evidence.Verification))
	for _, result := range evidence.Verification {
		verifications = append(verifications, verificationSummary{
			Name:       result.Name,
			Argv:       append([]string(nil), result.Argv...),
			Passed:     result.Passed,
			ExitCode:   result.ExitCode,
			Stdout:     maxText(result.Stdout, 4000),
			Stderr:     maxText(result.Stderr, 4000),
			DurationMS: result.DurationMS,
		})
	}
	return map[string]any{
		"immutable_seed": seed,
		"task": map[string]any{
			"id":             task.ID,
			"revision":       task.Compile.Revision,
			"execution_hash": task.Compile.ExecutionBundleHash,
			"base_commit":    task.Compile.BaseCommit,
			"status":         task.Status,
			"donegate":       task.LastGate,
			"repair_count":   task.RepairCount,
			"scope":          task.Scope,
			"risk":           task.Risk,
		},
		"evidence": map[string]any{
			"run_id":             evidence.RunID,
			"policy":             evidence.Policy,
			"verification":       verifications,
			"independent_review": evidence.Review,
			"falsifier_results":  evidence.Falsifiers,
			"prediction_checks":  evidence.Predictions,
			"kill_checks":        evidence.KillChecks,
			"codex_final":        maxText(evidence.Hand.FinalText, 12000),
			"diff":               maxText(diff, 64000),
			"diff_path":          evidence.DiffPath,
			"repository_before":  evidence.Before,
			"repository_after":   evidence.After,
		},
	}
}

func blindedEvaluationPayload(
	seed Seed,
	task dev.Task,
	evidence dev.EvidencePackage,
	diff string,
) map[string]any {
	payload := evaluationPayload(seed, task, evidence, diff)
	if evidencePayload, ok := payload["evidence"].(map[string]any); ok {
		delete(evidencePayload, "codex_final")
		evidencePayload["blinding"] = "executor final narrative omitted; judge deterministic artifacts and diff only"
	}
	return payload
}

func modelEvaluationStage(
	name, reviewer, model string,
	output evaluationModelOutput,
	evidenceRefs []string,
	criticalFindingVeto bool,
	blinded bool,
) EvaluationStage {
	score := output.Score
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	critical := cleanStrings(output.CriticalFindings)
	passed := output.Passed && score >= 0.70 && len(output.UnmetCriteria) == 0
	if criticalFindingVeto && len(critical) > 0 {
		passed = false
	}
	return EvaluationStage{
		Name:             name,
		Passed:           passed,
		Score:            round4(score),
		Summary:          strings.TrimSpace(output.Summary),
		Findings:         cleanStrings(output.Findings),
		UnmetCriteria:    cleanStrings(output.UnmetCriteria),
		EvidenceRefs:     cleanStrings(evidenceRefs),
		Reviewer:         reviewer,
		Model:            model,
		CriticalFindings: critical,
		Blinded:          blinded,
		IndependenceKey:  sha256Bytes([]byte(strings.TrimSpace(model) + "\x00" + reviewer)),
	}
}

func consensusStage(reviews []EvaluationStage, cfg Config) (EvaluationStage, float64, int, bool) {
	stage := EvaluationStage{
		Name:     "consensus",
		Reviewer: "go-core-majority",
	}
	if len(reviews) == 0 {
		stage.Summary = "No independent consensus reviews were recorded."
		return stage, 0, 0, true
	}
	passes := 0
	totalScore := 0.0
	minScore := 1.0
	maxScore := 0.0
	var findings []string
	var unmet []string
	var critical []string
	models := make(map[string]struct{})
	for _, review := range reviews {
		if review.Passed {
			passes++
		}
		totalScore += review.Score
		if review.Score < minScore {
			minScore = review.Score
		}
		if review.Score > maxScore {
			maxScore = review.Score
		}
		findings = append(findings, review.Findings...)
		unmet = append(unmet, review.UnmetCriteria...)
		critical = append(critical, review.CriticalFindings...)
		if model := strings.TrimSpace(review.Model); model != "" {
			models[model] = struct{}{}
		}
	}
	required := len(reviews)/2 + 1
	stage.Passed = passes >= required
	stage.Score = round4(totalScore / float64(len(reviews)))
	stage.Findings = cleanStrings(findings)
	stage.UnmetCriteria = cleanStrings(unmet)
	stage.CriticalFindings = cleanStrings(critical)
	spread := round4(maxScore - minScore)
	distinctModels := len(models)
	humanDecisionRequired := false
	if cfg.CriticalFindingVeto && len(stage.CriticalFindings) > 0 {
		stage.Passed = false
		humanDecisionRequired = true
		stage.Findings = cleanStrings(append(stage.Findings, "critical finding veto requires human disposition"))
	}
	if cfg.ConsensusMaxSpread > 0 && spread > cfg.ConsensusMaxSpread {
		stage.Passed = false
		humanDecisionRequired = true
		stage.Findings = cleanStrings(append(stage.Findings,
			fmt.Sprintf("review score spread %.4f exceeds maximum %.4f", spread, cfg.ConsensusMaxSpread),
		))
	}
	if len(reviews) > 1 && distinctModels < 2 {
		stage.Passed = false
		humanDecisionRequired = true
		stage.Findings = cleanStrings(append(stage.Findings,
			"all consensus reviews used the same model; correlated judgments require human disposition",
		))
	}
	stage.Summary = fmt.Sprintf("%d/%d independent reviews passed; %d required.", passes, len(reviews), required)
	stage.Metadata = map[string]string{
		"passes":          fmt.Sprintf("%d", passes),
		"reviews":         fmt.Sprintf("%d", len(reviews)),
		"required":        fmt.Sprintf("%d", required),
		"score_spread":    fmt.Sprintf("%.4f", spread),
		"distinct_models": fmt.Sprintf("%d", distinctModels),
	}
	return stage, spread, distinctModels, humanDecisionRequired
}

func skippedStage(name, reason string) EvaluationStage {
	return EvaluationStage{
		Name:     name,
		Passed:   false,
		Score:    0,
		Summary:  reason,
		Reviewer: "not-run",
		Metadata: map[string]string{"skipped": "true"},
	}
}

func (s *Service) recordEvaluationUnlocked(
	session Session,
	evaluation Evaluation,
	actor string,
) (Session, error) {
	session.Evaluations = append(session.Evaluations, evaluation)
	addUsage(&session.ModelUsage, evaluation.ModelUsage)
	session.Status = StatusEvaluated
	session.PendingEvolution = nil
	session.UpdatedAt = time.Now().UTC()
	var riskLevel string
	if seed, err := s.loadSeedUnlocked(evaluation.SeedHash); err == nil {
		riskLevel = seed.Risk.Level
	}
	if !evaluation.HumanDecisionRequired {
		s.appendEvaluationOutcomeUnlocked(
			&session,
			evaluation,
			riskLevel,
			"go-core",
			governance.Review{},
		)
	}
	session.ReferenceClass = s.referenceClassWithCurrentUnlocked(session)
	if err := s.appendEventUnlocked(session, "seed.evaluated", valueOr(actor, "human"), map[string]any{
		"evaluation_id":           evaluation.ID,
		"seed_hash":               evaluation.SeedHash,
		"task_id":                 evaluation.TaskID,
		"passed":                  evaluation.Passed,
		"mechanical":              evaluation.Mechanical.Passed,
		"semantic":                evaluation.Semantic.Passed,
		"consensus":               evaluation.Consensus.Passed,
		"human_decision_required": evaluation.HumanDecisionRequired,
		"score_spread":            evaluation.ScoreSpread,
		"distinct_models":         evaluation.DistinctModels,
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) appendEvaluationOutcomeUnlocked(
	session *Session,
	evaluation Evaluation,
	riskLevel, actor string,
	review governance.Review,
) OutcomeRecord {
	for _, outcome := range session.Outcomes {
		if outcome.EvaluationID == evaluation.ID {
			return outcome
		}
	}
	outcomeKind := "failed"
	if evaluation.Passed {
		outcomeKind = "passed"
	}
	return s.appendOutcomeRecordUnlocked(session, OutcomeRequest{
		Kind:         outcomeKind,
		EvaluationID: evaluation.ID,
		TaskID:       evaluation.TaskID,
		SeedHash:     evaluation.SeedHash,
		RiskLevel:    riskLevel,
		Reason:       evaluationOutcomeReason(evaluation),
		EvidenceRefs: cleanStrings(append(
			append(
				append([]string(nil), evaluation.Mechanical.EvidenceRefs...),
				evaluation.Consensus.EvidenceRefs...,
			),
			review.EvidenceRefs...,
		)),
		Review: review,
	}, actor)
}

func evaluationOutcomeReason(evaluation Evaluation) string {
	if evaluation.HumanDisposition != nil {
		return fmt.Sprintf(
			"disputed evaluation %s by human evidence disposition: %s",
			evaluation.HumanDisposition.Decision.Decision,
			evaluation.HumanDisposition.Decision.Rationale,
		)
	}
	if evaluation.Passed {
		return "mechanical, semantic, and independent consensus gates passed"
	}
	for _, stage := range []EvaluationStage{
		evaluation.Mechanical,
		evaluation.Semantic,
		evaluation.Consensus,
	} {
		if strings.TrimSpace(stage.Summary) != "" && !stage.Passed {
			return stage.Summary
		}
	}
	return "evaluation did not satisfy the immutable Seed"
}

func (s *Service) evaluationModel(index int) string {
	if len(s.cfg.EvaluationModels) == 0 {
		return s.cfg.Model
	}
	return s.cfg.EvaluationModels[index%len(s.cfg.EvaluationModels)]
}

func maxText(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return value[:maximum] + "\n...[truncated by GoClaw Ouroboros context budget]"
}
