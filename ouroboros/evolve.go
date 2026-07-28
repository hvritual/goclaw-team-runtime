package ouroboros

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

func (s *Service) ProposeEvolution(ctx context.Context, id, actor string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusEvaluated {
		return Session{}, fmt.Errorf("session %s must be evaluated before evolution (status %s)", id, session.Status)
	}
	if len(session.Evaluations) == 0 {
		return Session{}, errors.New("session has no evaluation")
	}
	evaluation := session.Evaluations[len(session.Evaluations)-1]
	history, passingInWindow := evaluationHistory(session.Evaluations, s.cfg.EvaluationHistoryWindow)
	current, err := s.loadSeedUnlocked(evaluation.SeedHash)
	if err != nil {
		return Session{}, err
	}
	if current.Generation >= s.cfg.MaxGenerations {
		proposal := EvolutionProposal{
			ID:                    "evolution-" + uuid.NewString(),
			SessionID:             session.ID,
			FromSeedHash:          current.Hash,
			FromGeneration:        current.Generation,
			ConvergenceThreshold:  s.cfg.ConvergenceThreshold,
			Status:                EvolutionBlocked,
			Action:                "human_required",
			Reasons:               []string{fmt.Sprintf("generation hard cap %d reached", s.cfg.MaxGenerations)},
			HardCapReached:        true,
			CreatedBy:             valueOr(actor, "human"),
			CreatedAt:             time.Now().UTC(),
			HistoryWindow:         len(history),
			PassingInWindow:       passingInWindow,
			CumulativeModelCalls:  session.ModelUsage.Calls,
			CumulativeModelTokens: session.ModelUsage.TotalTokens,
		}
		session.LastEvolution = &proposal
		session.Status = StatusBlocked
		session.BlockedReasons = proposal.Reasons
		session.UpdatedAt = proposal.CreatedAt
		if err := s.appendEventUnlocked(session, "evolution.blocked", actor, proposal); err != nil {
			return Session{}, err
		}
		return session, nil
	}
	if evaluation.HumanDecisionRequired {
		return s.blockEvolutionUnlocked(
			session,
			actor,
			current,
			history,
			passingInWindow,
			"latest evaluation contains correlated or disputed judgments requiring human disposition",
		)
	}
	if err := s.ensureModelBudget(session, 2); err != nil {
		return s.blockEvolutionUnlocked(
			session,
			actor,
			current,
			history,
			passingInWindow,
			err.Error(),
		)
	}

	var output evolutionModelOutput
	response, err := s.invokeModel(
		ctx,
		"ouroboros.evolve",
		evolutionSystemPrompt(),
		map[string]any{
			"current_seed":       current,
			"evaluation":         evaluation,
			"evaluation_history": history,
			"outcomes":           session.Outcomes,
			"reference_class":    session.ReferenceClass,
			"lineage":            session.SeedHistory,
			"cumulative_usage":   session.ModelUsage,
			"rules": map[string]any{
				"candidate_only":               true,
				"human_approval_required":      true,
				"convergence_threshold":        s.cfg.ConvergenceThreshold,
				"max_generations":              s.cfg.MaxGenerations,
				"history_window":               s.cfg.EvaluationHistoryWindow,
				"required_passing_evaluations": s.cfg.RequiredPassingEvaluations,
			},
		},
		s.cfg.Model,
		&output,
	)
	if err != nil {
		return Session{}, err
	}
	candidate, err := s.seedFromDraft(
		session,
		output.Seed,
		"model-proposed",
		current.Hash,
		current.Generation+1,
	)
	if err != nil {
		return Session{}, err
	}
	if err := s.writeSeedUnlocked(candidate); err != nil {
		return Session{}, err
	}
	similarity := ontologySimilarity(current.Ontology, candidate.Ontology)
	proposal := EvolutionProposal{
		ID:                    "evolution-" + uuid.NewString(),
		SessionID:             session.ID,
		FromSeedHash:          current.Hash,
		CandidateSeedHash:     candidate.Hash,
		FromGeneration:        current.Generation,
		CandidateGeneration:   candidate.Generation,
		OntologySimilarity:    similarity,
		ConvergenceThreshold:  s.cfg.ConvergenceThreshold,
		Status:                EvolutionPending,
		Action:                normalizeEvolutionAction(output.Action),
		Reasons:               cleanStrings(output.Reasons),
		KnowledgeGaps:         cleanStrings(output.KnowledgeGaps),
		PossibleRegressions:   cleanStrings(output.PossibleRegressions),
		CreatedBy:             "model-proposed",
		CreatedAt:             time.Now().UTC(),
		HistoryWindow:         len(history),
		PassingInWindow:       passingInWindow,
		CumulativeModelCalls:  session.ModelUsage.Calls + response.Usage.Calls,
		CumulativeModelTokens: session.ModelUsage.TotalTokens + response.Usage.TotalTokens,
	}
	if ancestor, ok := seedAtGeneration(session, current.Generation-1); ok {
		ancestorSeed, loadErr := s.loadSeedUnlocked(ancestor.Hash)
		if loadErr != nil {
			return Session{}, loadErr
		}
		if ontologySimilarity(ancestorSeed.Ontology, candidate.Ontology) >= s.cfg.ConvergenceThreshold &&
			similarity < s.cfg.ConvergenceThreshold {
			proposal.OscillationDetected = true
			proposal.Action = "human_required"
			proposal.Reasons = cleanStrings(append(proposal.Reasons,
				"period-2 ontology oscillation detected; human decision is required",
			))
		}
	}
	session.SeedHistory = append(session.SeedHistory, SeedReference{
		Hash:       candidate.Hash,
		ID:         candidate.ID,
		Generation: candidate.Generation,
		ParentHash: candidate.ParentHash,
		CreatedAt:  candidate.CreatedAt,
	})
	addUsage(&session.ModelUsage, response.Usage)
	if proposal.Action == "converged" && passingInWindow < s.cfg.RequiredPassingEvaluations {
		proposal.Action = "continue"
		proposal.Reasons = cleanStrings(append(proposal.Reasons,
			fmt.Sprintf(
				"only %d/%d recent evaluations passed; convergence requires %d",
				passingInWindow,
				len(history),
				s.cfg.RequiredPassingEvaluations,
			),
		))
	}
	if evaluation.Passed &&
		passingInWindow >= s.cfg.RequiredPassingEvaluations &&
		similarity >= s.cfg.ConvergenceThreshold &&
		proposal.Action == "converged" {
		proposal.Status = EvolutionConverged
		proposal.Action = "converged"
		session.Status = StatusConverged
		session.PendingEvolution = nil
	} else {
		session.Status = StatusEvolutionPending
		session.PendingEvolution = &proposal
	}
	session.LastEvolution = &proposal
	session.UpdatedAt = proposal.CreatedAt
	if err := s.appendEventUnlocked(session, "evolution.proposed", actor, proposal); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) ApproveEvolution(id, reviewer, comment string) (Session, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Session{}, errors.New("authenticated governance requires ApproveEvolutionWithReview")
	}
	return s.ApproveEvolutionWithReview(id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleEvolutionApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) ApproveEvolutionWithReview(id string, review governance.Review) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusEvolutionPending || session.PendingEvolution == nil {
		return Session{}, fmt.Errorf("session %s has no evolution proposal awaiting approval", id)
	}
	proposal := *session.PendingEvolution
	if proposal.Status != EvolutionPending || proposal.CandidateSeedHash == "" {
		return Session{}, errors.New("pending evolution proposal has no candidate Seed")
	}
	candidate, err := s.loadSeedUnlocked(proposal.CandidateSeedHash)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(review, governance.RoleEvolutionApprove); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, review, proposal.CreatedBy, candidate.CreatedBy); err != nil {
		return Session{}, err
	}
	decision := governance.ToDecision(review, "approved")
	for _, prior := range proposal.Approvals {
		if governance.SameActor(prior.ReviewerID, decision.ReviewerID) {
			return Session{}, fmt.Errorf("reviewer %q already decided this evolution", decision.ReviewerID)
		}
	}
	proposal.Approvals = append(proposal.Approvals, decision)
	found := false
	for index := range session.SeedHistory {
		if session.SeedHistory[index].Hash != proposal.CandidateSeedHash {
			continue
		}
		session.SeedHistory[index].Approvals = append(session.SeedHistory[index].Approvals, decision)
		found = true
		break
	}
	if !found {
		return Session{}, errors.New("candidate Seed is missing from lineage")
	}
	required := governance.RequiredQuorum(s.governance, candidate.Risk.Level == "high", "evolution")
	approvals := governance.DistinctApprovals(proposal.Approvals)
	if candidate.Risk.Level == "high" {
		highRiskRequired := governance.RequiredQuorum(s.governance, true, "seed")
		if highRiskRequired > required {
			required = highRiskRequired
		}
	}
	if approvals >= required {
		for index := range session.SeedHistory {
			if session.SeedHistory[index].Hash == proposal.CandidateSeedHash {
				session.SeedHistory[index].Approved = true
				session.SeedHistory[index].ApprovedBy = decision.ReviewerID
				session.SeedHistory[index].ApprovedAt = &decision.CreatedAt
				session.SeedHistory[index].Comment = decision.Rationale
				break
			}
		}
		proposal.Status = EvolutionApproved
		proposal.ReviewedBy = decision.ReviewerID
		proposal.ReviewedAt = &decision.CreatedAt
		proposal.ReviewComment = decision.Rationale
		session.ActiveSeedHash = proposal.CandidateSeedHash
		session.PendingSeedHash = ""
		session.PendingEvolution = nil
		session.Status = StatusApproved
	} else {
		session.PendingEvolution = &proposal
		session.Status = StatusEvolutionPending
	}
	session.LastEvolution = &proposal
	session.UpdatedAt = decision.CreatedAt
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "evolution_approval",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	eventType := "evolution.approval_recorded"
	if approvals >= required {
		eventType = "evolution.approved"
	}
	if err := s.appendEventUnlocked(session, eventType, decision.ReviewerID, map[string]any{
		"proposal":  proposal,
		"approvals": approvals,
		"quorum":    required,
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) RejectEvolution(id, reviewer, comment string) (Session, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Session{}, errors.New("authenticated governance requires RejectEvolutionWithReview")
	}
	return s.RejectEvolutionWithReview(id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleEvolutionApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) RejectEvolutionWithReview(id string, review governance.Review) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusEvolutionPending || session.PendingEvolution == nil {
		return Session{}, fmt.Errorf("session %s has no evolution proposal awaiting approval", id)
	}
	if err := governance.ValidateRole(review, governance.RoleEvolutionApprove); err != nil {
		return Session{}, err
	}
	proposal := *session.PendingEvolution
	if err := governance.ValidateDecision(s.governance, review, "rejected", proposal.CreatedBy); err != nil {
		return Session{}, err
	}
	decision := governance.ToDecision(review, "rejected")
	proposal.Approvals = append(proposal.Approvals, decision)
	proposal.Status = EvolutionRejected
	proposal.ReviewedBy = decision.ReviewerID
	proposal.ReviewedAt = &decision.CreatedAt
	proposal.ReviewComment = decision.Rationale
	session.PendingEvolution = nil
	session.LastEvolution = &proposal
	session.Status = StatusEvaluated
	session.UpdatedAt = decision.CreatedAt
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "evolution_approval",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	if err := s.appendEventUnlocked(session, "evolution.rejected", decision.ReviewerID, proposal); err != nil {
		return Session{}, err
	}
	return session, nil
}

func evaluationHistory(values []Evaluation, maximum int) ([]Evaluation, int) {
	if maximum < 1 || maximum > len(values) {
		maximum = len(values)
	}
	history := append([]Evaluation(nil), values[len(values)-maximum:]...)
	passing := 0
	for _, evaluation := range history {
		if evaluation.Passed && !evaluation.HumanDecisionRequired {
			passing++
		}
	}
	return history, passing
}

func (s *Service) blockEvolutionUnlocked(
	session Session,
	actor string,
	current Seed,
	history []Evaluation,
	passing int,
	reason string,
) (Session, error) {
	proposal := EvolutionProposal{
		ID:                    "evolution-" + uuid.NewString(),
		SessionID:             session.ID,
		FromSeedHash:          current.Hash,
		FromGeneration:        current.Generation,
		ConvergenceThreshold:  s.cfg.ConvergenceThreshold,
		Status:                EvolutionBlocked,
		Action:                "human_required",
		Reasons:               []string{reason},
		CreatedBy:             valueOr(actor, "human"),
		CreatedAt:             time.Now().UTC(),
		HistoryWindow:         len(history),
		PassingInWindow:       passing,
		CumulativeModelCalls:  session.ModelUsage.Calls,
		CumulativeModelTokens: session.ModelUsage.TotalTokens,
	}
	session.LastEvolution = &proposal
	session.Status = StatusBlocked
	session.BlockedReasons = cleanStrings(append(session.BlockedReasons, reason))
	session.UpdatedAt = proposal.CreatedAt
	if err := s.appendEventUnlocked(session, "evolution.blocked", actor, proposal); err != nil {
		return Session{}, err
	}
	return session, nil
}

func normalizeEvolutionAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "continue", "converged", "human_required":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "human_required"
	}
}

func seedAtGeneration(session Session, generation int) (SeedReference, bool) {
	for _, reference := range session.SeedHistory {
		if reference.Generation == generation && reference.Approved {
			return reference, true
		}
	}
	return SeedReference{}, false
}
