package ouroboros

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

func (s *Service) ResolveReadiness(
	id string,
	review governance.Review,
	ready bool,
) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if len(session.Rounds) == 0 {
		return Session{}, errors.New("session has no ambiguity assessment")
	}
	if err := governance.ValidateRole(review, governance.RoleReadinessOverride); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, review); err != nil {
		return Session{}, err
	}
	round := &session.Rounds[len(session.Rounds)-1]
	if ready {
		if !round.Assessment.HumanDecisionRequired {
			return Session{}, errors.New("human readiness override is allowed only for an explicit gray-zone or assessor-conflict escalation")
		}
		if hasOpenConflict(session.DecisionConflicts) {
			return Session{}, errors.New("readiness cannot be approved while stakeholder conflicts remain open")
		}
		for _, question := range round.Questions {
			if question.Blocking {
				return Session{}, fmt.Errorf("readiness cannot be approved while blocking question %q remains active", question.ID)
			}
		}
	}
	decision := governance.ToDecision(review, map[bool]string{true: "ready", false: "not_ready"}[ready])
	override := ReadinessOverride{
		Ready:     ready,
		Reviewer:  decision.ReviewerID,
		Rationale: decision.Rationale,
		Decision:  decision,
		CreatedAt: decision.CreatedAt,
	}
	round.Assessment.HumanOverride = &override
	round.Assessment.Ready = ready
	if ready {
		round.Assessment.ReadyStreak = round.Assessment.RequiredReadyStreak
		session.Status = StatusSeedReady
	} else {
		round.Assessment.ReadyStreak = 0
		session.Status = StatusClarificationNeeded
	}
	session.UpdatedAt = decision.CreatedAt
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "readiness_override",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	if err := s.appendEventUnlocked(session, "interview.readiness_resolved", decision.ReviewerID, override); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) ResolveConflict(
	id, conflictID, resolution string,
	review governance.Review,
) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(review, governance.RoleConflictResolve); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, review); err != nil {
		return Session{}, err
	}
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		return Session{}, errors.New("conflict resolution is required")
	}
	now := review.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	found := false
	var resolved DecisionConflict
	for index := range session.DecisionConflicts {
		conflict := &session.DecisionConflicts[index]
		if conflict.ID != conflictID {
			continue
		}
		if conflict.Status == "resolved" {
			return Session{}, fmt.Errorf("conflict %q is already resolved", conflictID)
		}
		conflict.Status = "resolved"
		conflict.Resolution = resolution
		conflict.ResolvedBy = review.ReviewerID
		conflict.ResolvedAt = &now
		resolved = *conflict
		found = true
		for claimIndex := range session.StakeholderClaims {
			for _, claimID := range conflict.ClaimIDs {
				if session.StakeholderClaims[claimIndex].ID == claimID {
					session.StakeholderClaims[claimIndex].Status = "resolved"
				}
			}
		}
		break
	}
	if !found {
		return Session{}, fmt.Errorf("conflict %q does not exist", conflictID)
	}
	session.Status = StatusInterviewing
	session.UpdatedAt = now
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "stakeholder_conflict",
		Decision:  resolution,
		Rationale: review.Rationale,
		Actor:     review.ReviewerID,
		CreatedAt: now,
	})
	if err := s.appendEventUnlocked(session, "interview.conflict_resolved", review.ReviewerID, map[string]any{
		"conflict": resolved,
		"review":   governance.ToDecision(review, "resolved"),
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

// ResolveEvaluation adjudicates a disputed model evaluation. This decision can
// unblock the specification-evolution loop, but it cannot override a failed
// mechanical gate or act as task/deployment acceptance.
func (s *Service) ResolveEvaluation(
	id, evaluationID string,
	accepted bool,
	review governance.Review,
) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(review, governance.RoleEvaluationResolve); err != nil {
		return Session{}, err
	}
	validationDecision := "rejected"
	if accepted {
		validationDecision = "approved"
	}
	if err := governance.ValidateDecision(s.governance, review, validationDecision); err != nil {
		return Session{}, err
	}
	var evaluation *Evaluation
	for index := range session.Evaluations {
		if session.Evaluations[index].ID == evaluationID {
			evaluation = &session.Evaluations[index]
			break
		}
	}
	if evaluation == nil {
		return Session{}, fmt.Errorf("evaluation %q does not exist", evaluationID)
	}
	if !evaluation.HumanDecisionRequired || evaluation.HumanDisposition != nil {
		return Session{}, fmt.Errorf("evaluation %q has no unresolved human disposition", evaluationID)
	}
	if accepted && (!evaluation.Mechanical.Passed || !evaluation.Semantic.Passed) {
		return Session{}, errors.New("human disposition cannot override failed mechanical or semantic gates")
	}

	decisionText := "rejected"
	if accepted {
		decisionText = "accepted"
	}
	decision := governance.ToDecision(review, decisionText)
	evaluation.HumanDecisionRequired = false
	evaluation.HumanDisposition = &EvaluationDisposition{
		Accepted: accepted,
		Decision: decision,
	}
	evaluation.Passed = accepted &&
		evaluation.Mechanical.Passed &&
		evaluation.Semantic.Passed
	if evaluation.Consensus.Metadata == nil {
		evaluation.Consensus.Metadata = make(map[string]string)
	}
	evaluation.Consensus.Metadata["human_disposition"] = decisionText
	evaluation.Consensus.Metadata["human_reviewer"] = decision.ReviewerID
	evaluation.Consensus.Summary = strings.TrimSpace(evaluation.Consensus.Summary +
		" Human evidence disposition: " + decisionText + ".")

	riskLevel := ""
	if seed, loadErr := s.loadSeedUnlocked(evaluation.SeedHash); loadErr == nil {
		riskLevel = seed.Risk.Level
	}
	s.appendEvaluationOutcomeUnlocked(&session, *evaluation, riskLevel, decision.ReviewerID, review)
	session.ReferenceClass = s.referenceClassWithCurrentUnlocked(session)
	session.PendingEvolution = nil
	session.BlockedReasons = removeEvaluationDispositionReasons(session.BlockedReasons)
	if len(session.BlockedReasons) == 0 {
		session.Status = StatusEvaluated
	}
	session.UpdatedAt = decision.CreatedAt
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "evaluation_disposition",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	if err := s.appendEventUnlocked(session, "evaluation.disposition_resolved", decision.ReviewerID, map[string]any{
		"evaluation_id": evaluation.ID,
		"passed":        evaluation.Passed,
		"decision":      decision,
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) RecordOutcome(id string, request OutcomeRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(request.Review, governance.RoleOutcomeRecord); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, request.Review); err != nil {
		return Session{}, err
	}
	if err := validateOutcomeRequest(request); err != nil {
		return Session{}, err
	}
	if err := s.resolveOutcomeReferencesUnlocked(session, &request); err != nil {
		return Session{}, err
	}
	record := s.appendOutcomeRecordUnlocked(&session, request, request.Review.ReviewerID)
	session.ReferenceClass = s.referenceClassWithCurrentUnlocked(session)
	session.UpdatedAt = record.CreatedAt
	if err := s.appendEventUnlocked(session, "outcome.recorded", record.Actor, record); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) TriggerKillCondition(
	id, conditionID, reason string,
	evidenceRefs []string,
	review governance.Review,
) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(review, governance.RoleKillSwitch); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, review); err != nil {
		return Session{}, err
	}
	if strings.TrimSpace(reason) == "" {
		return Session{}, errors.New("kill-trigger rationale is required")
	}
	seedHash := session.ActiveSeedHash
	if seedHash == "" {
		seedHash = session.PendingSeedHash
	}
	if seedHash == "" {
		return Session{}, errors.New("session has no Seed with a kill condition")
	}
	seed, err := s.loadSeedUnlocked(seedHash)
	if err != nil {
		return Session{}, err
	}
	var condition *KillCondition
	for index := range seed.KillConditions {
		if seed.KillConditions[index].ID == conditionID {
			condition = &seed.KillConditions[index]
			break
		}
	}
	if condition == nil {
		return Session{}, fmt.Errorf("kill condition %q does not exist in active Seed", conditionID)
	}
	now := review.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	trigger := KillTrigger{
		ConditionID:  conditionID,
		Reason:       strings.TrimSpace(reason),
		EvidenceRefs: cleanStrings(evidenceRefs),
		TriggeredBy:  review.ReviewerID,
		TriggeredAt:  now,
	}
	session.KillTriggers = append(session.KillTriggers, trigger)
	session.Status = StatusBlocked
	session.BlockedReasons = cleanStrings(append(session.BlockedReasons,
		fmt.Sprintf("kill condition %s triggered: %s", conditionID, reason),
	))
	session.UpdatedAt = now
	taskID := ""
	for index := len(session.CompiledTasks) - 1; index >= 0; index-- {
		if session.CompiledTasks[index].SeedHash == seedHash {
			taskID = session.CompiledTasks[index].TaskID
			break
		}
	}
	s.appendOutcomeRecordUnlocked(&session, OutcomeRequest{
		Kind:         "failed",
		TaskID:       taskID,
		SeedHash:     seedHash,
		RiskLevel:    seed.Risk.Level,
		Reason:       "kill condition triggered: " + strings.TrimSpace(reason),
		EvidenceRefs: evidenceRefs,
	}, review.ReviewerID)
	session.ReferenceClass = s.referenceClassWithCurrentUnlocked(session)
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "kill_switch",
		Decision:  condition.Action,
		Rationale: review.Rationale,
		Actor:     review.ReviewerID,
		CreatedAt: now,
	})
	if err := s.appendEventUnlocked(session, "session.kill_condition_triggered", review.ReviewerID, map[string]any{
		"condition": condition,
		"trigger":   trigger,
		"review":    governance.ToDecision(review, "triggered"),
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) ReferenceClass(projectID string) (ReferenceClassStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, err := os.Stat(s.sessionsDir()); err != nil {
		return ReferenceClassStats{}, err
	}
	return s.referenceClassUnlocked(projectID, ""), nil
}

func validateOutcomeRequest(request OutcomeRequest) error {
	switch strings.ToLower(strings.TrimSpace(request.Kind)) {
	case "passed", "failed", "cancelled", "rolled_back", "no_feedback":
	default:
		return fmt.Errorf("unsupported outcome kind %q", request.Kind)
	}
	if strings.TrimSpace(request.Reason) == "" {
		return errors.New("outcome reason is required")
	}
	if strings.TrimSpace(request.EvaluationID) == "" &&
		strings.TrimSpace(request.TaskID) == "" &&
		strings.TrimSpace(request.SeedHash) == "" {
		return errors.New("outcome must reference an evaluation, task, or Seed")
	}
	return nil
}

func (s *Service) resolveOutcomeReferencesUnlocked(
	session Session,
	request *OutcomeRequest,
) error {
	if evaluationID := strings.TrimSpace(request.EvaluationID); evaluationID != "" {
		found := false
		for _, evaluation := range session.Evaluations {
			if evaluation.ID != evaluationID {
				continue
			}
			found = true
			if request.TaskID == "" {
				request.TaskID = evaluation.TaskID
			}
			if request.SeedHash == "" {
				request.SeedHash = evaluation.SeedHash
			}
			break
		}
		if !found {
			return fmt.Errorf("evaluation %q does not belong to session %q", evaluationID, session.ID)
		}
	}
	if taskID := strings.TrimSpace(request.TaskID); taskID != "" {
		found := false
		for _, compiled := range session.CompiledTasks {
			if compiled.TaskID != taskID {
				continue
			}
			found = true
			if request.SeedHash == "" {
				request.SeedHash = compiled.SeedHash
			}
			break
		}
		if !found {
			return fmt.Errorf("task %q does not belong to session %q", taskID, session.ID)
		}
	}
	if seedHash := strings.TrimSpace(request.SeedHash); seedHash != "" {
		found := false
		for _, reference := range session.SeedHistory {
			if reference.Hash == seedHash {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Seed %q does not belong to session %q", seedHash, session.ID)
		}
		if request.RiskLevel == "" {
			if seed, err := s.loadSeedUnlocked(seedHash); err == nil {
				request.RiskLevel = seed.Risk.Level
			}
		}
	}
	return nil
}

func (s *Service) appendOutcomeRecordUnlocked(
	session *Session,
	request OutcomeRequest,
	actor string,
) OutcomeRecord {
	kind := strings.ToLower(strings.TrimSpace(request.Kind))
	record := OutcomeRecord{
		ID:           "outcome-" + uuid.NewString(),
		Kind:         kind,
		EvaluationID: strings.TrimSpace(request.EvaluationID),
		TaskID:       strings.TrimSpace(request.TaskID),
		SeedHash:     strings.TrimSpace(request.SeedHash),
		RiskLevel:    strings.TrimSpace(request.RiskLevel),
		Passed:       kind == "passed",
		Reason:       strings.TrimSpace(request.Reason),
		EvidenceRefs: cleanStrings(request.EvidenceRefs),
		Actor:        valueOr(actor, "system"),
		CreatedAt:    time.Now().UTC(),
	}
	if !request.Review.CreatedAt.IsZero() {
		record.CreatedAt = request.Review.CreatedAt
	}
	key := outcomeUnitKey(record)
	if key != "" {
		for index := len(session.Outcomes) - 1; index >= 0; index-- {
			if outcomeUnitKey(session.Outcomes[index]) == key {
				record.SupersedesID = session.Outcomes[index].ID
				break
			}
		}
	}
	session.Outcomes = append(session.Outcomes, record)
	return record
}

func removeEvaluationDispositionReasons(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), "evaluation") &&
			strings.Contains(strings.ToLower(value), "human disposition") {
			continue
		}
		result = append(result, value)
	}
	return cleanStrings(result)
}

func (s *Service) referenceClassWithCurrentUnlocked(current Session) ReferenceClassStats {
	stats := s.referenceClassUnlocked(current.ProjectID, current.ID)
	accumulateReferenceClass(&stats, current)
	finalizeReferenceClass(&stats)
	return stats
}

func (s *Service) referenceClassUnlocked(projectID, excludeID string) ReferenceClassStats {
	stats := ReferenceClassStats{ProjectID: projectID}
	entries, err := os.ReadDir(s.sessionsDir())
	if err != nil {
		return stats
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == excludeID {
			continue
		}
		session, err := s.loadSessionUnlocked(entry.Name())
		if err != nil || (projectID != "" && session.ProjectID != projectID) {
			continue
		}
		accumulateReferenceClass(&stats, session)
	}
	finalizeReferenceClass(&stats)
	return stats
}

func accumulateReferenceClass(stats *ReferenceClassStats, session Session) {
	latest := make(map[string]OutcomeRecord)
	for _, outcome := range session.Outcomes {
		key := outcomeUnitKey(outcome)
		if key == "" {
			key = "record:" + outcome.ID
		}
		latest[key] = outcome
	}
	for _, outcome := range latest {
		stats.Total++
		switch outcome.Kind {
		case "passed":
			stats.Passed++
		case "failed":
			stats.Failed++
		case "cancelled":
			stats.Cancelled++
		case "rolled_back":
			stats.RolledBack++
		case "no_feedback":
			stats.NoFeedback++
		}
	}
}

func outcomeUnitKey(outcome OutcomeRecord) string {
	if taskID := strings.TrimSpace(outcome.TaskID); taskID != "" {
		return "task:" + taskID
	}
	if evaluationID := strings.TrimSpace(outcome.EvaluationID); evaluationID != "" {
		return "evaluation:" + evaluationID
	}
	if seedHash := strings.TrimSpace(outcome.SeedHash); seedHash != "" {
		return "seed:" + seedHash
	}
	return ""
}

func finalizeReferenceClass(stats *ReferenceClassStats) {
	if stats.Total == 0 {
		stats.PassRate = 0
		stats.FailureRate = 0
		return
	}
	stats.PassRate = round4(float64(stats.Passed) / float64(stats.Total))
	failures := stats.Failed + stats.Cancelled + stats.RolledBack
	stats.FailureRate = round4(float64(failures) / float64(stats.Total))
}
