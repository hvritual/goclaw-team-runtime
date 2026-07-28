package ouroboros

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) seedFromDraft(
	session Session,
	draft seedModelOutput,
	actor, parentHash string,
	generation int,
) (Seed, error) {
	if len(session.Rounds) == 0 {
		return Seed{}, errors.New("session has no ambiguity assessment")
	}
	seed := Seed{
		SchemaVersion:        SchemaVersion,
		ID:                   "seed-" + uuid.NewString(),
		SessionID:            session.ID,
		Generation:           generation,
		ParentHash:           parentHash,
		Title:                strings.TrimSpace(draft.Title),
		Goal:                 strings.TrimSpace(draft.Goal),
		TaskType:             valueOr(draft.TaskType, "code"),
		ContextSummary:       strings.TrimSpace(draft.ContextSummary),
		Brownfield:           session.Brownfield,
		Constraints:          cleanStrings(draft.Constraints),
		NonGoals:             cleanStrings(draft.NonGoals),
		Assumptions:          cleanStrings(draft.Assumptions),
		AcceptanceCriteria:   draft.AcceptanceCriteria,
		Ontology:             draft.Ontology,
		EvaluationPrinciples: draft.EvaluationPrinciples,
		ExitConditions:       draft.ExitConditions,
		Plan:                 draft.Plan,
		Scope:                draft.Scope,
		Risk:                 draft.Risk,
		Cost:                 draft.Cost,
		Alternatives:         draft.Alternatives,
		Falsifiers:           draft.Falsifiers,
		CostOfInaction:       cleanStrings(draft.CostOfInaction),
		KillConditions:       draft.KillConditions,
		PreMortem:            cleanStrings(draft.PreMortem),
		ReferenceClass:       draft.ReferenceClass,
		Predictions:          draft.Predictions,
		StakeholderClaimIDs:  cleanStrings(draft.StakeholderClaimIDs),
		AmbiguityScore:       session.Rounds[len(session.Rounds)-1].Assessment.Overall,
		CreatedAt:            time.Now().UTC(),
		CreatedBy:            valueOr(actor, "human"),
	}
	if seed.Title == "" {
		seed.Title = session.Title
	}
	if seed.ContextSummary == "" {
		seed.ContextSummary = session.ContextSummary
	}
	if seed.Scope.MaxChangedFiles <= 0 {
		seed.Scope.MaxChangedFiles = 40
	}
	if seed.Scope.MaxChangedLines <= 0 {
		seed.Scope.MaxChangedLines = 2000
	}
	seed.Scope.DeniedPaths = cleanStrings(append(seed.Scope.DeniedPaths,
		".git/**",
		".env",
		".env.*",
		"**/.env",
		"**/.env.*",
		"*credential*",
		"*secret*",
		"**/*credential*",
		"**/*secret*",
	))
	if seed.Risk.Level == "" {
		seed.Risk.Level = "medium"
	}
	if seed.Risk.Rollback == "" {
		seed.Risk.Rollback = "Discard the isolated task worktree and preserve the immutable Seed and evidence."
	}
	seed.Risk.Forbidden = cleanStrings(append(seed.Risk.Forbidden,
		"push or merge remote branches",
		"modify OAuth credentials or secrets",
		"write directly to the synchronized Obsidian authority folders",
	))
	if seed.Cost.MaxRepairAttempts <= 0 {
		seed.Cost.MaxRepairAttempts = 2
	}
	if err := normalizeAndValidateSeed(&seed, s.cfg.MaxGenerations); err != nil {
		return Seed{}, err
	}
	hashable := seed
	hashable.Hash = ""
	data, err := json.Marshal(hashable)
	if err != nil {
		return Seed{}, err
	}
	seed.Hash = sha256Bytes(data)
	return seed, nil
}

func normalizeAndValidateSeed(seed *Seed, maximumGenerations int) error {
	if seed.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported Seed schema version %d", seed.SchemaVersion)
	}
	if err := validateID(seed.ID); err != nil {
		return fmt.Errorf("invalid Seed id: %w", err)
	}
	if err := validateID(seed.SessionID); err != nil {
		return fmt.Errorf("invalid Seed session id: %w", err)
	}
	if seed.Generation < 1 {
		return errors.New("Seed generation must be at least 1")
	}
	if maximumGenerations > 0 && seed.Generation > maximumGenerations {
		return fmt.Errorf("Seed generation %d exceeds hard cap %d", seed.Generation, maximumGenerations)
	}
	if seed.Generation > 1 && len(seed.ParentHash) != 64 {
		return errors.New("successor Seed requires a valid parent_hash")
	}
	if strings.TrimSpace(seed.Title) == "" {
		return errors.New("Seed title is required")
	}
	if strings.TrimSpace(seed.Goal) == "" {
		return errors.New("Seed goal is required")
	}
	if seed.TaskType != "code" {
		return fmt.Errorf("GoClaw Orchestrator Lite currently compiles only code Seeds, got %q", seed.TaskType)
	}
	if len(seed.Constraints) == 0 {
		return errors.New("Seed requires at least one explicit constraint")
	}
	if len(seed.AcceptanceCriteria) == 0 {
		return errors.New("Seed requires at least one acceptance criterion")
	}
	criterionIDs := make(map[string]struct{}, len(seed.AcceptanceCriteria))
	hasVerification := false
	for index := range seed.AcceptanceCriteria {
		criterion := &seed.AcceptanceCriteria[index]
		criterion.ID = strings.TrimSpace(criterion.ID)
		if criterion.ID == "" {
			criterion.ID = fmt.Sprintf("ac-%d", index+1)
		}
		if err := validateID(criterion.ID); err != nil {
			return fmt.Errorf("acceptance criterion %d: %w", index+1, err)
		}
		if _, ok := criterionIDs[criterion.ID]; ok {
			return fmt.Errorf("duplicate acceptance criterion id %q", criterion.ID)
		}
		criterionIDs[criterion.ID] = struct{}{}
		criterion.Description = strings.TrimSpace(criterion.Description)
		if criterion.Description == "" {
			return fmt.Errorf("acceptance criterion %q has no description", criterion.ID)
		}
		criterion.ExpectedArtifacts = cleanStrings(criterion.ExpectedArtifacts)
		if len(criterion.VerifyCommand) > 0 {
			if err := validateVerificationArgv(criterion.VerifyCommand); err != nil {
				return fmt.Errorf("acceptance criterion %q: %w", criterion.ID, err)
			}
			hasVerification = true
		}
	}
	if !hasVerification {
		return errors.New("Seed requires at least one deterministic argv verification command")
	}
	seed.Ontology.Name = strings.TrimSpace(seed.Ontology.Name)
	seed.Ontology.Description = strings.TrimSpace(seed.Ontology.Description)
	if seed.Ontology.Name == "" || len(seed.Ontology.Fields) == 0 {
		return errors.New("Seed ontology requires a name and at least one field")
	}
	fieldNames := make(map[string]struct{}, len(seed.Ontology.Fields))
	for index := range seed.Ontology.Fields {
		field := &seed.Ontology.Fields[index]
		field.Name = strings.TrimSpace(field.Name)
		field.Type = strings.TrimSpace(field.Type)
		field.Description = strings.TrimSpace(field.Description)
		if field.Name == "" || field.Type == "" {
			return fmt.Errorf("ontology field %d requires name and type", index+1)
		}
		key := normalizeText(field.Name)
		if _, ok := fieldNames[key]; ok {
			return fmt.Errorf("duplicate ontology field %q", field.Name)
		}
		fieldNames[key] = struct{}{}
	}
	if strings.TrimSpace(seed.Plan.Summary) == "" || len(seed.Plan.Milestones) == 0 {
		return errors.New("Seed plan requires summary and at least one milestone")
	}
	workItemIDs := make(map[string]struct{})
	for milestoneIndex := range seed.Plan.Milestones {
		milestone := &seed.Plan.Milestones[milestoneIndex]
		if milestone.ID == "" {
			milestone.ID = fmt.Sprintf("m%d", milestoneIndex+1)
		}
		if err := validateID(milestone.ID); err != nil {
			return fmt.Errorf("milestone %d: %w", milestoneIndex+1, err)
		}
		if strings.TrimSpace(milestone.Title) == "" || len(milestone.WorkItems) == 0 {
			return fmt.Errorf("milestone %q requires title and work items", milestone.ID)
		}
		for itemIndex := range milestone.WorkItems {
			item := &milestone.WorkItems[itemIndex]
			if item.ID == "" {
				item.ID = fmt.Sprintf("%s-w%d", milestone.ID, itemIndex+1)
			}
			if err := validateID(item.ID); err != nil {
				return fmt.Errorf("work item %d in %s: %w", itemIndex+1, milestone.ID, err)
			}
			if _, ok := workItemIDs[item.ID]; ok {
				return fmt.Errorf("duplicate work item id %q", item.ID)
			}
			workItemIDs[item.ID] = struct{}{}
			if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Instructions) == "" {
				return fmt.Errorf("work item %q requires title and instructions", item.ID)
			}
			for _, criterionID := range item.CriteriaIDs {
				if _, ok := criterionIDs[criterionID]; !ok {
					return fmt.Errorf("work item %q references unknown criterion %q", item.ID, criterionID)
				}
			}
		}
	}
	if len(seed.Scope.AllowedPaths) == 0 {
		return errors.New("Seed scope requires at least one allowed path")
	}
	for _, path := range append(append([]string{}, seed.Scope.AllowedPaths...), seed.Scope.DeniedPaths...) {
		if err := validateRepositoryPattern(path); err != nil {
			return err
		}
	}
	if seed.Scope.MaxChangedFiles < 1 || seed.Scope.MaxChangedFiles > 10000 {
		return errors.New("Seed scope max_changed_files must be between 1 and 10000")
	}
	if seed.Scope.MaxChangedLines < 1 || seed.Scope.MaxChangedLines > 1_000_000 {
		return errors.New("Seed scope max_changed_lines must be between 1 and 1000000")
	}
	switch seed.Risk.Level {
	case "low", "medium", "high":
	default:
		return fmt.Errorf("unsupported Seed risk level %q", seed.Risk.Level)
	}
	if strings.TrimSpace(seed.Risk.Rollback) == "" {
		return errors.New("Seed risk rollback is required")
	}
	if seed.Cost.MaxRepairAttempts < 1 || seed.Cost.MaxRepairAttempts > 10 {
		return errors.New("Seed max_repair_attempts must be between 1 and 10")
	}
	if len(seed.Alternatives) < 2 {
		return errors.New("Seed requires at least two materially different alternatives")
	}
	selectedAlternatives := 0
	unselectedAlternatives := 0
	alternativeIDs := make(map[string]struct{}, len(seed.Alternatives))
	for index := range seed.Alternatives {
		alternative := &seed.Alternatives[index]
		alternative.ID = strings.TrimSpace(alternative.ID)
		alternative.Title = strings.TrimSpace(alternative.Title)
		alternative.Summary = strings.TrimSpace(alternative.Summary)
		alternative.Tradeoffs = cleanStrings(alternative.Tradeoffs)
		if alternative.ID == "" {
			alternative.ID = fmt.Sprintf("alt-%d", index+1)
		}
		if err := validateID(alternative.ID); err != nil {
			return fmt.Errorf("alternative %d: %w", index+1, err)
		}
		if _, ok := alternativeIDs[alternative.ID]; ok {
			return fmt.Errorf("duplicate alternative id %q", alternative.ID)
		}
		alternativeIDs[alternative.ID] = struct{}{}
		if alternative.Title == "" || alternative.Summary == "" {
			return fmt.Errorf("alternative %q requires title and summary", alternative.ID)
		}
		if alternative.Selected {
			selectedAlternatives++
		} else {
			unselectedAlternatives++
		}
	}
	if selectedAlternatives != 1 || unselectedAlternatives < 1 {
		return errors.New("Seed alternatives require exactly one selected option and at least one rejected/status-quo option")
	}
	seed.CostOfInaction = cleanStrings(seed.CostOfInaction)
	if len(seed.CostOfInaction) == 0 {
		return errors.New("Seed requires an explicit cost_of_inaction")
	}
	falsifierCoverage := make(map[string]struct{}, len(seed.Falsifiers))
	for index := range seed.Falsifiers {
		falsifier := &seed.Falsifiers[index]
		falsifier.CriterionID = strings.TrimSpace(falsifier.CriterionID)
		falsifier.Condition = strings.TrimSpace(falsifier.Condition)
		falsifier.EvidenceRequired = strings.TrimSpace(falsifier.EvidenceRequired)
		if _, ok := criterionIDs[falsifier.CriterionID]; !ok {
			return fmt.Errorf("falsifier %d references unknown criterion %q", index+1, falsifier.CriterionID)
		}
		if falsifier.Condition == "" || falsifier.EvidenceRequired == "" {
			return fmt.Errorf("falsifier for %q requires condition and evidence_required", falsifier.CriterionID)
		}
		falsifierCoverage[falsifier.CriterionID] = struct{}{}
	}
	for criterionID := range criterionIDs {
		if _, ok := falsifierCoverage[criterionID]; !ok {
			return fmt.Errorf("acceptance criterion %q has no falsifier", criterionID)
		}
	}
	if len(seed.KillConditions) == 0 {
		return errors.New("Seed requires at least one kill condition")
	}
	killIDs := make(map[string]struct{}, len(seed.KillConditions))
	for index := range seed.KillConditions {
		condition := &seed.KillConditions[index]
		condition.ID = strings.TrimSpace(condition.ID)
		condition.Condition = strings.TrimSpace(condition.Condition)
		condition.Metric = strings.TrimSpace(condition.Metric)
		condition.Threshold = strings.TrimSpace(condition.Threshold)
		condition.Action = strings.ToLower(strings.TrimSpace(condition.Action))
		if condition.ID == "" {
			condition.ID = fmt.Sprintf("kill-%d", index+1)
		}
		if err := validateID(condition.ID); err != nil {
			return fmt.Errorf("kill condition %d: %w", index+1, err)
		}
		if _, ok := killIDs[condition.ID]; ok {
			return fmt.Errorf("duplicate kill condition id %q", condition.ID)
		}
		killIDs[condition.ID] = struct{}{}
		if condition.Condition == "" || condition.Metric == "" || condition.Threshold == "" {
			return fmt.Errorf("kill condition %q requires condition, metric, and threshold", condition.ID)
		}
		switch condition.Metric {
		case "changed_files", "changed_lines", "input_tokens", "output_tokens", "repair_attempts":
		default:
			return fmt.Errorf("kill condition %q has unsupported metric %q", condition.ID, condition.Metric)
		}
		threshold, err := strconv.ParseFloat(condition.Threshold, 64)
		if err != nil || threshold < 0 {
			return fmt.Errorf("kill condition %q threshold must be a non-negative number", condition.ID)
		}
		switch condition.Action {
		case "stop", "reframe", "rollback", "human_required":
		default:
			return fmt.Errorf("kill condition %q has unsupported action %q", condition.ID, condition.Action)
		}
	}
	seed.PreMortem = cleanStrings(seed.PreMortem)
	if len(seed.PreMortem) == 0 {
		return errors.New("Seed requires at least one pre-mortem failure scenario")
	}
	seed.ReferenceClass.Basis = strings.TrimSpace(seed.ReferenceClass.Basis)
	if seed.ReferenceClass.Basis == "" {
		return errors.New("Seed reference_class basis is required, including an explicit no-data statement")
	}
	if seed.ReferenceClass.SampleSize < 0 ||
		seed.ReferenceClass.BaseFailureRate < 0 ||
		seed.ReferenceClass.BaseFailureRate > 1 {
		return errors.New("Seed reference_class sample_size and base_failure_rate are invalid")
	}
	if seed.ReferenceClass.P50DurationMinutes < 0 ||
		seed.ReferenceClass.P90DurationMinutes < seed.ReferenceClass.P50DurationMinutes ||
		seed.ReferenceClass.P50InputTokens < 0 ||
		seed.ReferenceClass.P90InputTokens < seed.ReferenceClass.P50InputTokens {
		return errors.New("Seed reference_class P90 values must be greater than or equal to non-negative P50 values")
	}
	if len(seed.Predictions) == 0 {
		return errors.New("Seed requires at least one preregistered prediction")
	}
	predictionIDs := make(map[string]struct{}, len(seed.Predictions))
	for index := range seed.Predictions {
		prediction := &seed.Predictions[index]
		prediction.ID = strings.TrimSpace(prediction.ID)
		prediction.Claim = strings.TrimSpace(prediction.Claim)
		prediction.ExpectedOutcome = strings.TrimSpace(prediction.ExpectedOutcome)
		prediction.Horizon = strings.TrimSpace(prediction.Horizon)
		if prediction.ID == "" {
			prediction.ID = fmt.Sprintf("prediction-%d", index+1)
		}
		if err := validateID(prediction.ID); err != nil {
			return fmt.Errorf("prediction %d: %w", index+1, err)
		}
		if _, ok := predictionIDs[prediction.ID]; ok {
			return fmt.Errorf("duplicate prediction id %q", prediction.ID)
		}
		predictionIDs[prediction.ID] = struct{}{}
		if prediction.Claim == "" || prediction.ExpectedOutcome == "" || prediction.Horizon == "" {
			return fmt.Errorf("prediction %q requires claim, expected_outcome, and horizon", prediction.ID)
		}
		if prediction.Confidence < 0 || prediction.Confidence > 1 {
			return fmt.Errorf("prediction %q confidence must be between 0 and 1", prediction.ID)
		}
	}
	for _, claimID := range seed.StakeholderClaimIDs {
		if err := validateID(claimID); err != nil {
			return fmt.Errorf("stakeholder claim id %q: %w", claimID, err)
		}
	}
	if len(seed.EvaluationPrinciples) == 0 || len(seed.ExitConditions) == 0 {
		return errors.New("Seed requires evaluation principles and exit conditions")
	}
	return nil
}

func validateVerificationArgv(argv []string) error {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return errors.New("verify_command must be a non-empty argv array")
	}
	command := strings.ToLower(filepath.Base(strings.TrimSpace(argv[0])))
	switch command {
	case "sh", "bash", "zsh", "fish", "cmd", "cmd.exe", "powershell", "pwsh",
		"rm", "rmdir", "del", "dd", "mkfs", "shutdown", "reboot", "halt":
		return fmt.Errorf("verification executable %q is not allowed", command)
	}
	if command == "git" && len(argv) > 1 {
		switch strings.ToLower(strings.TrimSpace(argv[1])) {
		case "push", "reset", "clean", "checkout", "switch", "commit", "merge", "rebase":
			return fmt.Errorf("verification git subcommand %q is not read-only", argv[1])
		}
	}
	for _, arg := range argv {
		if strings.ContainsRune(arg, '\x00') {
			return errors.New("verification argv contains a NUL byte")
		}
	}
	return nil
}

func validateRepositoryPattern(pattern string) error {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return errors.New("Seed scope contains an empty path")
	}
	if filepath.IsAbs(pattern) || strings.HasPrefix(pattern, "/") {
		return fmt.Errorf("Seed scope path must be repository-relative: %s", pattern)
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == ".." {
			return fmt.Errorf("Seed scope path escapes repository: %s", pattern)
		}
	}
	return nil
}
