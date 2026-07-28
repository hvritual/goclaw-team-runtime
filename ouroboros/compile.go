package ouroboros

import (
	"errors"
	"fmt"
	"time"

	dev "github.com/smallnest/goclaw/orchestratorlite"
)

func (s *Service) BuildTaskRequest(id, actor string) (dev.CreateRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return dev.CreateRequest{}, err
	}
	return s.buildTaskRequestUnlocked(session, actor)
}

func (s *Service) CompileTask(id, actor string, development *dev.Service) (dev.Task, error) {
	if development == nil {
		return dev.Task{}, errors.New("Orchestrator Lite development service is not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.loadSessionUnlocked(id)
	if err != nil {
		return dev.Task{}, err
	}
	for _, compiled := range session.CompiledTasks {
		if compiled.SeedHash != session.ActiveSeedHash {
			continue
		}
		task, getErr := development.GetTask(compiled.TaskID)
		if getErr != nil {
			return dev.Task{}, fmt.Errorf("compiled task %s is missing: %w", compiled.TaskID, getErr)
		}
		return task, nil
	}
	request, err := s.buildTaskRequestUnlocked(session, actor)
	if err != nil {
		return dev.Task{}, err
	}
	task, err := development.CreateTask(request)
	if err != nil {
		return dev.Task{}, err
	}
	now := time.Now().UTC()
	seed, err := s.loadSeedUnlocked(session.ActiveSeedHash)
	if err != nil {
		return dev.Task{}, err
	}
	session.CompiledTasks = append(session.CompiledTasks, CompiledTask{
		SeedHash:   seed.Hash,
		Generation: seed.Generation,
		TaskID:     task.ID,
		CompiledBy: valueOr(actor, "human"),
		CompiledAt: now,
	})
	session.Status = StatusCompiled
	session.UpdatedAt = now
	if err := s.appendEventUnlocked(session, "seed.compiled", actor, map[string]any{
		"seed_hash":  seed.Hash,
		"generation": seed.Generation,
		"task_id":    task.ID,
	}); err != nil {
		return dev.Task{}, err
	}
	return task, nil
}

func (s *Service) buildTaskRequestUnlocked(session Session, _ string) (dev.CreateRequest, error) {
	if session.Status != StatusApproved && session.Status != StatusCompiled {
		return dev.CreateRequest{}, fmt.Errorf(
			"session %s requires an approved Seed before compilation (status %s)",
			session.ID,
			session.Status,
		)
	}
	if session.ActiveSeedHash == "" {
		return dev.CreateRequest{}, errors.New("session has no active approved Seed")
	}
	seed, err := s.loadSeedUnlocked(session.ActiveSeedHash)
	if err != nil {
		return dev.CreateRequest{}, err
	}
	approved := false
	for _, reference := range session.SeedHistory {
		if reference.Hash == seed.Hash && reference.Approved {
			approved = true
			break
		}
	}
	if !approved {
		return dev.CreateRequest{}, errors.New("active Seed does not have a human approval record")
	}

	commandsByCriterion := make(map[string][]dev.CommandSpec, len(seed.AcceptanceCriteria))
	var allCommands []dev.CommandSpec
	var successTests []string
	for _, criterion := range seed.AcceptanceCriteria {
		successTests = append(successTests, criterion.Description)
		if len(criterion.VerifyCommand) == 0 {
			continue
		}
		command := dev.CommandSpec{
			Name: criterion.ID + ": " + firstLine(criterion.Description, 60),
			Argv: append([]string(nil), criterion.VerifyCommand...),
		}
		commandsByCriterion[criterion.ID] = append(commandsByCriterion[criterion.ID], command)
		allCommands = append(allCommands, command)
	}
	if len(allCommands) == 0 {
		return dev.CreateRequest{}, errors.New("approved Seed has no deterministic verification command")
	}
	alternatives := make([]dev.DecisionAlternative, 0, len(seed.Alternatives))
	for _, alternative := range seed.Alternatives {
		alternatives = append(alternatives, dev.DecisionAlternative{
			ID: alternative.ID, Title: alternative.Title, Summary: alternative.Summary,
			Tradeoffs: append([]string(nil), alternative.Tradeoffs...), Selected: alternative.Selected,
		})
	}
	falsifiers := make([]dev.TaskFalsifier, 0, len(seed.Falsifiers))
	for _, falsifier := range seed.Falsifiers {
		falsifiers = append(falsifiers, dev.TaskFalsifier{
			CriterionID: falsifier.CriterionID, Condition: falsifier.Condition,
			EvidenceRequired: falsifier.EvidenceRequired,
		})
	}
	predictions := make([]dev.TaskPrediction, 0, len(seed.Predictions))
	for _, prediction := range seed.Predictions {
		predictions = append(predictions, dev.TaskPrediction{
			ID: prediction.ID, Claim: prediction.Claim,
			ExpectedOutcome: prediction.ExpectedOutcome,
			Horizon:         prediction.Horizon,
			Confidence:      prediction.Confidence,
		})
	}
	killConditions := make([]dev.TaskKillCondition, 0, len(seed.KillConditions))
	for _, condition := range seed.KillConditions {
		killConditions = append(killConditions, dev.TaskKillCondition{
			ID: condition.ID, Condition: condition.Condition, Metric: condition.Metric,
			Threshold: condition.Threshold, Action: condition.Action,
		})
	}
	milestones := make([]dev.Milestone, 0, len(seed.Plan.Milestones))
	for _, milestone := range seed.Plan.Milestones {
		workItems := make([]dev.WorkItem, 0, len(milestone.WorkItems))
		for _, item := range milestone.WorkItems {
			var commands []dev.CommandSpec
			for _, criterionID := range item.CriteriaIDs {
				commands = append(commands, commandsByCriterion[criterionID]...)
			}
			workItems = append(workItems, dev.WorkItem{
				ID:           item.ID,
				Title:        item.Title,
				Instructions: item.Instructions,
				DependsOn:    append([]string(nil), item.DependsOn...),
				CapabilityManifest: dev.CapabilityManifest{
					Executor: "codex-exec",
					Model:    s.cfg.Model,
					Tools:    []string{"filesystem", "shell", "test"},
					Sandbox:  "workspace-write",
				},
				VerificationCommands: commands,
			})
		}
		milestones = append(milestones, dev.Milestone{
			ID:        milestone.ID,
			Title:     milestone.Title,
			WorkItems: workItems,
		})
	}
	taskID := fmt.Sprintf("task-%s-g%d", session.ID, seed.Generation)
	return dev.CreateRequest{
		ID:        taskID,
		ProjectID: session.ProjectID,
		Title:     seed.Title,
		RepoPath:  session.RepoPath,
		BaseRef:   session.BaseRef,
		Request: dev.RequestFrame{
			RawRequest: session.RawRequest,
			Source:     "ouroboros:" + session.ID,
		},
		Goal: dev.GoalSpec{
			Objective:      seed.Goal,
			NonGoals:       append([]string(nil), seed.NonGoals...),
			Assumptions:    append([]string(nil), seed.Assumptions...),
			SuccessTests:   successTests,
			Alternatives:   alternatives,
			CostOfInaction: append([]string(nil), seed.CostOfInaction...),
			Falsifiers:     falsifiers,
			Predictions:    predictions,
			PreMortem:      append([]string(nil), seed.PreMortem...),
		},
		Plan: dev.PlanSpec{
			Summary:    seed.Plan.Summary,
			Milestones: milestones,
		},
		EvidencePlan: dev.EvidencePlan{
			Required: []string{
				"immutable_seed",
				"repository_before",
				"repository_after",
				"diff",
				"verification_results",
				"policy_result",
				"independent_review",
				"donegate_result",
				"falsifier_results",
				"prediction_checks",
			},
			Commands: allCommands,
		},
		Scope: dev.ScopePolicy{
			AllowedPaths:       append([]string(nil), seed.Scope.AllowedPaths...),
			DeniedPaths:        append([]string(nil), seed.Scope.DeniedPaths...),
			MaxChangedFiles:    seed.Scope.MaxChangedFiles,
			MaxChangedLines:    seed.Scope.MaxChangedLines,
			AllowNewDependency: seed.Scope.AllowNewDependency,
		},
		Risk: dev.RiskPlan{
			Level:          seed.Risk.Level,
			Forbidden:      append([]string(nil), seed.Risk.Forbidden...),
			Rollback:       seed.Risk.Rollback,
			HumanEscalates: append([]string(nil), seed.Risk.HumanEscalates...),
			KillConditions: killConditions,
			ReferenceClass: dev.TaskReferenceClass{
				Basis:              seed.ReferenceClass.Basis,
				SampleSize:         seed.ReferenceClass.SampleSize,
				BaseFailureRate:    seed.ReferenceClass.BaseFailureRate,
				P50DurationMinutes: seed.ReferenceClass.P50DurationMinutes,
				P90DurationMinutes: seed.ReferenceClass.P90DurationMinutes,
				P50InputTokens:     seed.ReferenceClass.P50InputTokens,
				P90InputTokens:     seed.ReferenceClass.P90InputTokens,
			},
		},
		Cost: dev.CostPlan{
			MaxRepairAttempts: seed.Cost.MaxRepairAttempts,
			MaxInputTokens:    seed.Cost.MaxInputTokens,
			MaxOutputTokens:   seed.Cost.MaxOutputTokens,
		},
		DoneGate: dev.DoneGateSpec{
			RequireChangedFiles:      true,
			RequireAllVerifications:  true,
			RequirePolicyPass:        true,
			RequireIndependentReview: true,
			RequireHumanAcceptance:   true,
		},
		CreatedBy: "ouroboros:" + session.ID,
	}, nil
}
