package ouroboros

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

type Service struct {
	cfg        Config
	mu         sync.RWMutex
	model      Model
	governance governance.Config
}

func NewService(cfg Config) (*Service, error) {
	if strings.TrimSpace(cfg.Root) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.Root = filepath.Join(home, ".goclaw", "ouroboros")
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}
	cfg.Root = root
	if cfg.AmbiguityThreshold == 0 {
		cfg.AmbiguityThreshold = DefaultAmbiguityThreshold
	}
	if cfg.ConvergenceThreshold == 0 {
		cfg.ConvergenceThreshold = DefaultConvergenceThreshold
	}
	if cfg.RequiredReadyStreak == 0 {
		cfg.RequiredReadyStreak = DefaultRequiredReadyStreak
	}
	if cfg.MaxGenerations == 0 {
		cfg.MaxGenerations = DefaultMaxGenerations
	}
	if cfg.ConsensusReviewers == 0 {
		cfg.ConsensusReviewers = DefaultConsensusReviewers
	}
	if cfg.MaxQuestionsPerRound == 0 {
		cfg.MaxQuestionsPerRound = DefaultMaxQuestionsPerRound
	}
	if cfg.MaxContextBytes == 0 {
		cfg.MaxContextBytes = DefaultMaxContextBytes
	}
	if cfg.MaxOutputTokens == 0 {
		cfg.MaxOutputTokens = DefaultMaxOutputTokens
	}
	if cfg.AssessmentReviewers == 0 {
		cfg.AssessmentReviewers = DefaultAssessmentReviewers
	}
	if cfg.AssessmentMaxSpread == 0 {
		cfg.AssessmentMaxSpread = DefaultAssessmentMaxSpread
	}
	if cfg.AssessmentGrayZone == 0 {
		cfg.AssessmentGrayZone = DefaultAssessmentGrayZone
	}
	if cfg.ConsensusMaxSpread == 0 {
		cfg.ConsensusMaxSpread = DefaultConsensusMaxSpread
	}
	if cfg.EvaluationHistoryWindow == 0 {
		cfg.EvaluationHistoryWindow = DefaultEvaluationWindow
	}
	if cfg.RequiredPassingEvaluations == 0 {
		cfg.RequiredPassingEvaluations = DefaultPassingWindow
	}
	if cfg.MaxSessionModelCalls == 0 {
		cfg.MaxSessionModelCalls = DefaultMaxSessionModelCalls
	}
	if cfg.MaxSessionModelTokens == 0 {
		cfg.MaxSessionModelTokens = DefaultMaxSessionModelTokens
	}
	service := &Service{cfg: cfg, governance: governance.DefaultConfig()}
	if err := service.Ensure(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Service) SetModel(model Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
}

func (s *Service) SetGovernancePolicy(policy governance.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.governance = policy
}

func (s *Service) GovernancePolicy() governance.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.governance
}

func (s *Service) Ensure() error {
	for _, dir := range []string{s.sessionsDir(), s.seedsDir()} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Start(ctx context.Context, request StartRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.model == nil {
		return Session{}, errors.New("ouroboros model is not configured")
	}
	if strings.TrimSpace(request.RawRequest) == "" {
		return Session{}, errors.New("raw_request is required")
	}
	if strings.TrimSpace(request.Title) == "" {
		request.Title = firstLine(request.RawRequest, 80)
	}
	if strings.TrimSpace(request.ProjectID) == "" {
		request.ProjectID = "default"
	}
	if strings.TrimSpace(request.TopicID) == "" {
		request.TopicID = "inbox"
	}
	if strings.TrimSpace(request.RepoPath) == "" {
		return Session{}, errors.New("repo_path is required")
	}
	repoPath, repositoryContext, err := inspectRepository(request.RepoPath)
	if err != nil {
		return Session{}, err
	}
	request.RepoPath = repoPath
	if request.BaseRef == "" {
		request.BaseRef = "HEAD"
	}
	if request.ID == "" {
		request.ID = "ouro-" + uuid.NewString()
	}
	if err := validateID(request.ID); err != nil {
		return Session{}, err
	}
	if exists(s.sessionDir(request.ID)) {
		return Session{}, fmt.Errorf("ouroboros session already exists: %s", request.ID)
	}
	now := time.Now().UTC()
	session := Session{
		SchemaVersion:     SchemaVersion,
		ID:                request.ID,
		ProjectID:         request.ProjectID,
		TopicID:           request.TopicID,
		Title:             request.Title,
		RepoPath:          request.RepoPath,
		BaseRef:           request.BaseRef,
		RawRequest:        request.RawRequest,
		ContextSummary:    request.ContextSummary,
		RepositoryContext: repositoryContext,
		Brownfield:        request.Brownfield,
		Stakeholders:      cleanStrings(request.Stakeholders),
		Status:            StatusInterviewing,
		CreatedBy:         valueOr(request.CreatedBy, "human"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	session.ReferenceClass = s.referenceClassUnlocked(session.ProjectID, session.ID)
	if _, err := encodeModelPayload(interviewPayload(session), s.cfg.MaxContextBytes); err != nil {
		return Session{}, err
	}
	if err := s.appendEventUnlocked(session, "session.started", session.CreatedBy, map[string]any{
		"project_id": session.ProjectID,
		"topic_id":   session.TopicID,
		"brownfield": session.Brownfield,
		"repo_path":  session.RepoPath,
		"base_ref":   session.BaseRef,
	}); err != nil {
		return Session{}, err
	}
	return s.assessUnlocked(ctx, session, session.CreatedBy)
}

func (s *Service) Answer(ctx context.Context, id string, request AnswerRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusInterviewing &&
		session.Status != StatusClarificationNeeded &&
		session.Status != StatusSeedReady {
		return Session{}, fmt.Errorf("session %s cannot accept answers in status %s", id, session.Status)
	}
	if len(session.Rounds) == 0 {
		return Session{}, errors.New("session has no interview round")
	}
	if len(request.Answers) == 0 {
		return Session{}, errors.New("at least one answer is required")
	}
	round := &session.Rounds[len(session.Rounds)-1]
	available := make(map[string]Question, len(round.Questions))
	answered := make(map[string]struct{}, len(round.Answers))
	for _, question := range round.Questions {
		available[question.ID] = question
	}
	for _, answer := range round.Answers {
		answered[answer.QuestionID] = struct{}{}
	}
	actor := valueOr(request.Actor, "human")
	now := time.Now().UTC()
	for _, answer := range request.Answers {
		answer.QuestionID = strings.TrimSpace(answer.QuestionID)
		answer.Text = strings.TrimSpace(answer.Text)
		if _, ok := available[answer.QuestionID]; !ok {
			return Session{}, fmt.Errorf("question %q is not active in round %d", answer.QuestionID, round.Number)
		}
		if _, ok := answered[answer.QuestionID]; ok {
			return Session{}, fmt.Errorf("question %q was already answered", answer.QuestionID)
		}
		if answer.Text == "" {
			return Session{}, fmt.Errorf("answer for %q is empty", answer.QuestionID)
		}
		answer.AnsweredBy = actor
		answer.CreatedAt = now
		round.Answers = append(round.Answers, answer)
		answered[answer.QuestionID] = struct{}{}
	}
	session.UpdatedAt = now
	session.Status = StatusInterviewing
	if err := s.appendEventUnlocked(session, "interview.answered", actor, request.Answers); err != nil {
		return Session{}, err
	}
	return s.assessUnlocked(ctx, session, actor)
}

func (s *Service) Reassess(ctx context.Context, id, actor string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	switch session.Status {
	case StatusInterviewing, StatusClarificationNeeded, StatusSeedReady:
	default:
		return Session{}, fmt.Errorf("session %s cannot be reassessed in status %s", id, session.Status)
	}
	return s.assessUnlocked(ctx, session, valueOr(actor, "human"))
}

func (s *Service) assessUnlocked(ctx context.Context, session Session, actor string) (Session, error) {
	results, err := s.runInterviewAssessors(ctx, session)
	if err != nil {
		session.LastError = err.Error()
		session.UpdatedAt = time.Now().UTC()
		_ = s.appendEventUnlocked(session, "interview.failed", actor, map[string]any{"error": err.Error()})
		return Session{}, err
	}

	roundNumber := len(session.Rounds) + 1
	readyStreak := 0
	if len(session.Rounds) > 0 {
		readyStreak = session.Rounds[len(session.Rounds)-1].Assessment.ReadyStreak
	}
	output, assessment, err := aggregateInterviewAssessments(
		results,
		session,
		roundNumber,
		readyStreak,
		s.cfg,
	)
	if err != nil {
		return Session{}, err
	}
	questions, err := normalizeQuestions(
		dedupeQuestions(output.Questions),
		roundNumber,
		s.cfg.MaxQuestionsPerRound,
		session.Brownfield,
	)
	if err != nil {
		return Session{}, err
	}

	now := time.Now().UTC()
	session.ProblemFrames = mergeProblemFrames(
		session.ProblemFrames,
		normalizeProblemFrames(output.ProblemFrames, roundNumber),
	)
	session.StakeholderClaims = mergeStakeholderClaims(
		session.StakeholderClaims,
		normalizeStakeholderClaims(output.StakeholderClaims, roundNumber, now),
	)
	session.DecisionConflicts = mergeDecisionConflicts(
		session.DecisionConflicts,
		normalizeDecisionConflicts(output.DecisionConflicts, roundNumber),
	)
	for _, question := range questions {
		if question.Blocking {
			assessment.Ready = false
			assessment.ReadyStreak = 0
			break
		}
	}
	if hasOpenConflict(session.DecisionConflicts) {
		assessment.HumanDecisionRequired = true
		assessment.Ready = false
		assessment.ReadyStreak = 0
		assessment.Unresolved = cleanStrings(append(
			assessment.Unresolved,
			"one or more stakeholder decision conflicts require human resolution",
		))
	}
	if len(session.Rounds) > 0 {
		assessment.RepeatedQuestionRatio = questionOverlap(
			questions,
			session.Rounds[len(session.Rounds)-1].Questions,
		)
		if assessment.RepeatedQuestionRatio >= 0.70 &&
			session.Rounds[len(session.Rounds)-1].Assessment.RepeatedQuestionRatio >= 0.70 {
			assessment.HumanDecisionRequired = true
			assessment.Ready = false
		}
	}
	assessment.CreatedAt = now
	session.Rounds = append(session.Rounds, InterviewRound{
		Number:     roundNumber,
		Questions:  questions,
		Assessment: assessment,
		CreatedAt:  assessment.CreatedAt,
	})
	session.Assumptions = mergeAssumptions(session.Assumptions, output.Assumptions, assessment.CreatedAt)
	for _, decision := range output.Decisions {
		if strings.TrimSpace(decision.Decision) == "" {
			continue
		}
		session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
			ID:        "decision-" + uuid.NewString(),
			Kind:      valueOr(decision.Kind, "other"),
			Decision:  strings.TrimSpace(decision.Decision),
			Rationale: strings.TrimSpace(decision.Rationale),
			Actor:     "model-proposed",
			CreatedAt: assessment.CreatedAt,
		})
	}
	for _, result := range results {
		addUsage(&session.ModelUsage, result.response.Usage)
	}
	session.LastError = ""
	session.UpdatedAt = assessment.CreatedAt
	switch {
	case assessment.Ready:
		session.Status = StatusSeedReady
	case assessment.HumanDecisionRequired:
		session.Status = StatusClarificationNeeded
	default:
		session.Status = StatusInterviewing
	}
	if err := s.appendEventUnlocked(session, "interview.assessed", actor, assessment); err != nil {
		return Session{}, err
	}
	return session, nil
}

func interviewPayload(session Session) map[string]any {
	return map[string]any{
		"session_id":         session.ID,
		"project_id":         session.ProjectID,
		"title":              session.Title,
		"raw_request":        session.RawRequest,
		"context_summary":    session.ContextSummary,
		"repository_context": session.RepositoryContext,
		"brownfield":         session.Brownfield,
		"stakeholders":       session.Stakeholders,
		"repository_path":    session.RepoPath,
		"base_ref":           session.BaseRef,
		"interview_rounds":   session.Rounds,
		"assumptions":        session.Assumptions,
		"decision_ledger":    session.DecisionLedger,
		"problem_frames":     session.ProblemFrames,
		"stakeholder_claims": session.StakeholderClaims,
		"decision_conflicts": session.DecisionConflicts,
		"reference_class":    session.ReferenceClass,
	}
}

func (s *Service) Crystallize(ctx context.Context, id, actor string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusSeedReady {
		return Session{}, fmt.Errorf("session %s is not seed-ready (status %s)", id, session.Status)
	}
	if len(session.Rounds) == 0 || !session.Rounds[len(session.Rounds)-1].Assessment.Ready {
		return Session{}, errors.New("latest ambiguity assessment is not ready")
	}
	if hasOpenConflict(session.DecisionConflicts) {
		return Session{}, errors.New("stakeholder decision conflicts must be resolved before crystallization")
	}
	if err := s.ensureModelBudget(session, 2); err != nil {
		return Session{}, err
	}
	session.ReferenceClass = s.referenceClassWithCurrentUnlocked(session)
	var output seedModelOutput
	response, err := s.invokeModel(
		ctx,
		"ouroboros.crystallize",
		seedSystemPrompt(),
		map[string]any{
			"session":             session,
			"approved_fact_rule":  "Only human statements and explicit request/context are facts; model assumptions remain labeled.",
			"repository_boundary": "All paths are repository-relative and execution remains owned by GoClaw Orchestrator Lite.",
		},
		s.cfg.Model,
		&output,
	)
	if err != nil {
		return Session{}, err
	}
	seed, err := s.seedFromDraft(session, output, "model-proposed", "", 1)
	if err != nil {
		return Session{}, err
	}
	if err := s.writeSeedUnlocked(seed); err != nil {
		return Session{}, err
	}
	session.PendingSeedHash = seed.Hash
	session.Status = StatusAwaitingSeedApproval
	session.SeedHistory = append(session.SeedHistory, SeedReference{
		Hash:       seed.Hash,
		ID:         seed.ID,
		Generation: seed.Generation,
		ParentHash: seed.ParentHash,
		CreatedAt:  seed.CreatedAt,
	})
	addUsage(&session.ModelUsage, response.Usage)
	session.UpdatedAt = time.Now().UTC()
	if err := s.appendEventUnlocked(session, "seed.crystallized", actor, map[string]any{
		"seed_hash":      seed.Hash,
		"generation":     seed.Generation,
		"ambiguity":      seed.AmbiguityScore,
		"approval_state": "pending",
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) ApproveSeed(id, reviewer, comment string) (Session, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Session{}, errors.New("authenticated governance requires ApproveSeedWithReview")
	}
	return s.ApproveSeedWithReview(id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleSeedApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) ApproveSeedWithReview(id string, review governance.Review) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusAwaitingSeedApproval || session.PendingSeedHash == "" {
		return Session{}, fmt.Errorf("session %s has no Seed awaiting approval", id)
	}
	seed, err := s.loadSeedUnlocked(session.PendingSeedHash)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(review, governance.RoleSeedApprove); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, review, seed.CreatedBy); err != nil {
		return Session{}, err
	}
	decision := governance.ToDecision(review, "approved")
	found := false
	for index := range session.SeedHistory {
		if session.SeedHistory[index].Hash != session.PendingSeedHash {
			continue
		}
		for _, prior := range session.SeedHistory[index].Approvals {
			if governance.SameActor(prior.ReviewerID, decision.ReviewerID) {
				return Session{}, fmt.Errorf("reviewer %q already decided this Seed", decision.ReviewerID)
			}
		}
		session.SeedHistory[index].Approvals = append(session.SeedHistory[index].Approvals, decision)
		found = true
		break
	}
	if !found {
		return Session{}, errors.New("pending Seed is missing from Seed history")
	}
	required := governance.RequiredQuorum(s.governance, seed.Risk.Level == "high", "seed")
	approvals := 0
	for index := range session.SeedHistory {
		if session.SeedHistory[index].Hash == session.PendingSeedHash {
			approvals = governance.DistinctApprovals(session.SeedHistory[index].Approvals)
			if approvals >= required {
				session.SeedHistory[index].Approved = true
				session.SeedHistory[index].ApprovedBy = decision.ReviewerID
				session.SeedHistory[index].ApprovedAt = &decision.CreatedAt
				session.SeedHistory[index].Comment = decision.Rationale
			}
			break
		}
	}
	approvedHash := session.PendingSeedHash
	if approvals >= required {
		session.ActiveSeedHash = approvedHash
		session.PendingSeedHash = ""
		session.Status = StatusApproved
	}
	session.UpdatedAt = decision.CreatedAt
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "seed_approval",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	eventType := "seed.approval_recorded"
	if approvals >= required {
		eventType = "seed.approved"
	}
	if err := s.appendEventUnlocked(session, eventType, decision.ReviewerID, map[string]any{
		"seed_hash": approvedHash,
		"decision":  decision,
		"approvals": approvals,
		"quorum":    required,
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) RejectSeed(id, reviewer, comment string) (Session, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Session{}, errors.New("authenticated governance requires RejectSeedWithReview")
	}
	return s.RejectSeedWithReview(id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleSeedApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) RejectSeedWithReview(id string, review governance.Review) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusAwaitingSeedApproval || session.PendingSeedHash == "" {
		return Session{}, fmt.Errorf("session %s has no Seed awaiting approval", id)
	}
	seed, err := s.loadSeedUnlocked(session.PendingSeedHash)
	if err != nil {
		return Session{}, err
	}
	if err := governance.ValidateRole(review, governance.RoleSeedApprove); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateDecision(s.governance, review, "rejected", seed.CreatedBy); err != nil {
		return Session{}, err
	}
	decision := governance.ToDecision(review, "rejected")
	rejectedHash := session.PendingSeedHash
	for index := range session.SeedHistory {
		if session.SeedHistory[index].Hash == rejectedHash {
			session.SeedHistory[index].Approvals = append(session.SeedHistory[index].Approvals, decision)
			break
		}
	}
	session.PendingSeedHash = ""
	session.Status = StatusClarificationNeeded
	session.UpdatedAt = decision.CreatedAt
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "seed_approval",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	if err := s.appendEventUnlocked(session, "seed.rejected", decision.ReviewerID, map[string]any{
		"seed_hash": rejectedHash,
		"decision":  decision,
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) Cancel(id, actor, reason string) (Session, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Session{}, errors.New("authenticated governance requires CancelWithReview")
	}
	return s.CancelWithReview(id, reason, governance.Review{
		ReviewerID:    actor,
		Rationale:     valueOr(reason, "cancel the session before a verified outcome"),
		Role:          governance.RoleSessionCancel,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) CancelWithReview(
	id, reason string,
	review governance.Review,
) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	if session.Status == StatusConverged || session.Status == StatusCancelled {
		return Session{}, fmt.Errorf("session %s cannot be cancelled in status %s", id, session.Status)
	}
	if strings.TrimSpace(reason) == "" {
		return Session{}, errors.New("cancellation reason is required")
	}
	if err := governance.ValidateRole(review, governance.RoleSessionCancel); err != nil {
		return Session{}, err
	}
	if err := governance.ValidateApproval(s.governance, review); err != nil {
		return Session{}, err
	}
	decision := governance.ToDecision(review, "cancelled")
	session.Status = StatusCancelled
	session.UpdatedAt = decision.CreatedAt
	s.appendOutcomeRecordUnlocked(&session, OutcomeRequest{
		Kind:   "cancelled",
		Reason: strings.TrimSpace(reason),
		Review: review,
	}, decision.ReviewerID)
	session.ReferenceClass = s.referenceClassWithCurrentUnlocked(session)
	session.DecisionLedger = append(session.DecisionLedger, DecisionLedgerRecord{
		ID:        "decision-" + uuid.NewString(),
		Kind:      "session_cancel",
		Decision:  decision.Decision,
		Rationale: decision.Rationale,
		Actor:     decision.ReviewerID,
		CreatedAt: decision.CreatedAt,
	})
	if err := s.appendEventUnlocked(session, "session.cancelled", decision.ReviewerID, map[string]any{
		"reason":   strings.TrimSpace(reason),
		"decision": decision,
	}); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) GetSession(id string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadSessionUnlocked(id)
}

func (s *Service) ListSessions(projectID string) ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.sessionsDir())
	if err != nil {
		return nil, err
	}
	result := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		session, err := s.loadSessionUnlocked(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("load session %s: %w", entry.Name(), err)
		}
		if projectID != "" && session.ProjectID != projectID {
			continue
		}
		result = append(result, session)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (s *Service) ListEvents(id string) ([]SessionEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadEventsUnlocked(id)
}

func (s *Service) GetSeed(hash string) (Seed, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadSeedUnlocked(hash)
}

func (s *Service) appendEventUnlocked(session Session, eventType, actor string, data any) error {
	events, err := s.loadEventsOptionalUnlocked(session.ID)
	if err != nil {
		return err
	}
	var previousHash string
	var sequence int64 = 1
	if len(events) > 0 {
		previousHash = events[len(events)-1].Hash
		sequence = events[len(events)-1].Sequence + 1
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	event := SessionEvent{
		SchemaVersion: SchemaVersion,
		ID:            "event-" + uuid.NewString(),
		SessionID:     session.ID,
		Sequence:      sequence,
		Type:          eventType,
		Actor:         valueOr(actor, "system"),
		CreatedAt:     time.Now().UTC(),
		PreviousHash:  previousHash,
		Data:          raw,
		Snapshot:      session,
	}
	hashable := event
	hashable.Hash = ""
	encoded, err := json.Marshal(hashable)
	if err != nil {
		return err
	}
	event.Hash = sha256Bytes(encoded)
	if err := appendJSONLine(s.eventsPath(session.ID), event); err != nil {
		return err
	}
	return writeJSONAtomic(s.projectionPath(session.ID), session, 0o600)
}

func (s *Service) loadSessionUnlocked(id string) (Session, error) {
	events, err := s.loadEventsUnlocked(id)
	if err != nil {
		return Session{}, err
	}
	session := events[len(events)-1].Snapshot
	if session.ID != id {
		return Session{}, fmt.Errorf("event snapshot session id %q does not match %q", session.ID, id)
	}
	return session, nil
}

func (s *Service) loadEventsOptionalUnlocked(id string) ([]SessionEvent, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	events, err := readEventLines(s.eventsPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := validateEventChain(events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Service) loadEventsUnlocked(id string) ([]SessionEvent, error) {
	events, err := s.loadEventsOptionalUnlocked(id)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, os.ErrNotExist
	}
	return events, nil
}

func (s *Service) writeSeedUnlocked(seed Seed) error {
	return immutableWriteJSON(s.seedPath(seed.Hash), seed)
}

func (s *Service) loadSeedUnlocked(hash string) (Seed, error) {
	if len(hash) != 64 {
		return Seed{}, errors.New("Seed hash must be a 64-character SHA-256")
	}
	var seed Seed
	if err := readJSON(s.seedPath(hash), &seed); err != nil {
		return Seed{}, err
	}
	if seed.Hash != hash {
		return Seed{}, errors.New("Seed filename hash does not match embedded hash")
	}
	hashable := seed
	hashable.Hash = ""
	encoded, err := json.Marshal(hashable)
	if err != nil {
		return Seed{}, err
	}
	if sha256Bytes(encoded) != seed.Hash {
		return Seed{}, errors.New("immutable Seed content hash mismatch")
	}
	return seed, nil
}

func (s *Service) sessionDir(id string) string {
	path, err := safeJoin(s.sessionsDir(), id)
	if err != nil {
		return filepath.Join(s.sessionsDir(), "__invalid__")
	}
	return path
}

func (s *Service) eventsPath(id string) string {
	return filepath.Join(s.sessionDir(id), "events.jsonl")
}

func (s *Service) projectionPath(id string) string {
	return filepath.Join(s.sessionDir(id), "session.json")
}

func (s *Service) seedPath(hash string) string {
	path, err := safeJoin(s.seedsDir(), hash+".json")
	if err != nil {
		return filepath.Join(s.seedsDir(), "__invalid__.json")
	}
	return path
}

func (s *Service) sessionsDir() string { return filepath.Join(s.cfg.Root, "sessions") }
func (s *Service) seedsDir() string    { return filepath.Join(s.cfg.Root, "seeds") }

func normalizeQuestions(values []Question, round, maximum int, brownfield bool) ([]Question, error) {
	if maximum < 1 {
		maximum = DefaultMaxQuestionsPerRound
	}
	if len(values) > maximum {
		values = values[:maximum]
	}
	result := make([]Question, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		value.Text = strings.TrimSpace(value.Text)
		value.Why = strings.TrimSpace(value.Why)
		if value.Text == "" {
			continue
		}
		switch value.Dimension {
		case DimensionGoal, DimensionConstraint, DimensionSuccess:
		case DimensionContext:
			if !brownfield {
				value.Dimension = DimensionConstraint
			}
		default:
			return nil, fmt.Errorf("question %d has unsupported dimension %q", index+1, value.Dimension)
		}
		if value.ID == "" {
			value.ID = "q-" + strconv.Itoa(round) + "-" + strconv.Itoa(index+1)
		}
		if err := validateID(value.ID); err != nil {
			value.ID = "q-" + strconv.Itoa(round) + "-" + strconv.Itoa(index+1)
		}
		if _, ok := seen[value.ID]; ok {
			value.ID = "q-" + strconv.Itoa(round) + "-" + strconv.Itoa(index+1)
		}
		seen[value.ID] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func mergeAssumptions(existing []Assumption, proposed []string, now time.Time) []Assumption {
	seen := make(map[string]struct{}, len(existing))
	for _, assumption := range existing {
		seen[normalizeText(assumption.Text)] = struct{}{}
	}
	for _, text := range cleanStrings(proposed) {
		key := normalizeText(text)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, Assumption{
			ID:        "assumption-" + uuid.NewString(),
			Text:      text,
			Status:    AssumptionProposed,
			Source:    "model",
			UpdatedAt: now,
		})
	}
	return existing
}

func firstLine(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		value = value[:index]
	}
	runes := []rune(value)
	if len(runes) > maxRunes {
		value = string(runes[:maxRunes])
	}
	return value
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
