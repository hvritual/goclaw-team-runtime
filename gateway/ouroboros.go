package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smallnest/goclaw/governance"
	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/ouroboros"
	"github.com/smallnest/goclaw/teamcontrol"
)

func (h *Handler) registerOuroborosMethods() {
	h.registry.Register("ouroboros.sessions", func(_ string, params map[string]interface{}) (interface{}, error) {
		return h.ouroSvc.ListSessions(stringParam(params["project_id"]))
	})
	h.registry.Register("ouroboros.session.get", func(_ string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		return h.ouroSvc.GetSession(id)
	})
	h.registry.Register("ouroboros.session.events", func(_ string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		return h.ouroSvc.ListEvents(id)
	})
	h.registry.Register("ouroboros.seed.get", func(_ string, params map[string]interface{}) (interface{}, error) {
		hash := stringParam(params["hash"])
		if hash == "" {
			return nil, errors.New("hash is required")
		}
		return h.ouroSvc.GetSeed(hash)
	})
	h.registry.Register("ouroboros.session.start", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		var request ouroboros.StartRequest
		if err := decodeRPCParams(params, &request); err != nil {
			return nil, err
		}
		actor, err := h.actorForSession(sessionID, request.CreatedBy)
		if err != nil {
			return nil, err
		}
		request.CreatedBy = actor
		if request.RepoPath == "" && h.devSvc != nil {
			request.RepoPath = h.devSvc.Config().RepoPath
		}
		if request.BaseRef == "" {
			request.BaseRef = "HEAD"
		}
		return h.ouroSvc.Start(context.Background(), request)
	})
	h.registry.Register("ouroboros.session.answer", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		var request ouroboros.AnswerRequest
		if err := decodeRPCParams(params, &request); err != nil {
			return nil, err
		}
		actor, err := h.actorForSession(sessionID, request.Actor)
		if err != nil {
			return nil, err
		}
		request.Actor = actor
		request.Reassess = true
		return h.ouroSvc.Answer(context.Background(), id, request)
	})
	h.registry.Register("ouroboros.session.reassess", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.Reassess(context.Background(), id, actor)
	})
	h.registry.Register("ouroboros.session.crystallize", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.Crystallize(context.Background(), id, actor)
	})
	h.registry.Register("ouroboros.seed.approve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		review, err := h.humanReview(sessionID, params, governance.RoleSeedApprove)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.ApproveSeedWithReview(id, review)
	})
	h.registry.Register("ouroboros.seed.reject", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		review, err := h.humanReview(sessionID, params, governance.RoleSeedApprove)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.RejectSeedWithReview(id, review)
	})
	h.registry.Register("ouroboros.session.compile", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.devSvc == nil {
			return nil, errors.New("Orchestrator Lite development service is not enabled")
		}
		if h.teamSvc != nil {
			return nil, fmt.Errorf(
				"%w: direct Ouroboros compilation is disabled in team mode; create the development task through the Wave-bound task factory",
				teamcontrol.ErrForbidden,
			)
		}
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.CompileTask(id, actor, h.devSvc)
	})
	h.registry.Register("ouroboros.session.evaluate", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.devSvc == nil {
			return nil, errors.New("Orchestrator Lite development service is not enabled")
		}
		id := stringParam(params["id"])
		taskID := stringParam(params["task_id"])
		if id == "" || taskID == "" {
			return nil, errors.New("id and task_id are required")
		}
		task, evidence, diff, err := h.readTaskEvidence(taskID)
		if err != nil {
			return nil, err
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.EvaluateTask(
			context.Background(),
			id,
			actor,
			task,
			evidence,
			diff,
		)
	})
	h.registry.Register("ouroboros.evaluation.resolve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleEvaluationResolve)
		if err != nil {
			return nil, err
		}
		accepted, ok := params["accepted"].(bool)
		if !ok {
			return nil, errors.New("accepted boolean is required")
		}
		return h.ouroSvc.ResolveEvaluation(
			stringParam(params["id"]),
			stringParam(params["evaluation_id"]),
			accepted,
			review,
		)
	})
	h.registry.Register("ouroboros.session.evolve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, errors.New("id is required")
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.ProposeEvolution(context.Background(), id, actor)
	})
	h.registry.Register("ouroboros.evolution.approve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleEvolutionApprove)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.ApproveEvolutionWithReview(stringParam(params["id"]), review)
	})
	h.registry.Register("ouroboros.evolution.reject", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleEvolutionApprove)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.RejectEvolutionWithReview(stringParam(params["id"]), review)
	})
	h.registry.Register("ouroboros.readiness.resolve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleReadinessOverride)
		if err != nil {
			return nil, err
		}
		ready, ok := params["ready"].(bool)
		if !ok {
			return nil, errors.New("ready boolean is required")
		}
		return h.ouroSvc.ResolveReadiness(stringParam(params["id"]), review, ready)
	})
	h.registry.Register("ouroboros.conflict.resolve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleConflictResolve)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.ResolveConflict(
			stringParam(params["id"]),
			stringParam(params["conflict_id"]),
			stringParam(params["resolution"]),
			review,
		)
	})
	h.registry.Register("ouroboros.outcome.record", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleOutcomeRecord)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.RecordOutcome(stringParam(params["id"]), ouroboros.OutcomeRequest{
			Kind:         stringParam(params["kind"]),
			EvaluationID: stringParam(params["evaluation_id"]),
			TaskID:       stringParam(params["task_id"]),
			SeedHash:     stringParam(params["seed_hash"]),
			RiskLevel:    stringParam(params["risk_level"]),
			Reason:       stringParam(params["reason"]),
			EvidenceRefs: stringSliceParam(params["evidence_refs"]),
			Review:       review,
		})
	})
	h.registry.Register("ouroboros.kill.trigger", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleKillSwitch)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.TriggerKillCondition(
			stringParam(params["id"]),
			stringParam(params["condition_id"]),
			stringParam(params["reason"]),
			stringSliceParam(params["evidence_refs"]),
			review,
		)
	})
	h.registry.Register("ouroboros.reference_class", func(_ string, params map[string]interface{}) (interface{}, error) {
		return h.ouroSvc.ReferenceClass(stringParam(params["project_id"]))
	})
	h.registry.Register("ouroboros.session.cancel", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleSessionCancel)
		if err != nil {
			return nil, err
		}
		return h.ouroSvc.CancelWithReview(
			stringParam(params["id"]),
			stringParam(params["reason"]),
			review,
		)
	})
}

func (h *Handler) readTaskEvidence(taskID string) (dev.Task, dev.EvidencePackage, string, error) {
	task, err := h.devSvc.GetTask(taskID)
	if err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	if task.LastEvidence == "" {
		return dev.Task{}, dev.EvidencePackage{}, "", errors.New("task has no EvidencePackage")
	}
	evidenceCandidate := task.LastEvidence
	if info, statErr := os.Stat(evidenceCandidate); statErr == nil && info.IsDir() {
		evidenceCandidate = filepath.Join(evidenceCandidate, "evidence.json")
	}
	evidencePath, err := resolveOuroEvidenceFile(
		h.devSvc.Config().Root,
		evidenceCandidate,
		32*1024*1024,
	)
	if err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	var evidence dev.EvidencePackage
	if err := json.Unmarshal(data, &evidence); err != nil {
		return dev.Task{}, dev.EvidencePackage{}, "", err
	}
	diff := ""
	if evidence.DiffPath != "" {
		diffPath, resolveErr := resolveOuroEvidenceFile(
			h.devSvc.Config().Root,
			evidence.DiffPath,
			8*1024*1024,
		)
		if resolveErr != nil {
			return dev.Task{}, dev.EvidencePackage{}, "", resolveErr
		}
		diffData, readErr := os.ReadFile(diffPath)
		if readErr != nil {
			return dev.Task{}, dev.EvidencePackage{}, "", fmt.Errorf("read diff evidence: %w", readErr)
		}
		diff = string(diffData)
	}
	return task, evidence, diff, nil
}

func resolveOuroEvidenceFile(root, candidate string, maximum int64) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("resolve development runtime root: %w", err)
	}
	absoluteCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	resolvedCandidate, err := filepath.EvalSymlinks(absoluteCandidate)
	if err != nil {
		return "", err
	}
	if resolvedCandidate != resolvedRoot &&
		!strings.HasPrefix(resolvedCandidate, resolvedRoot+string(filepath.Separator)) {
		return "", errors.New("evidence path escapes development runtime root")
	}
	info, err := os.Stat(resolvedCandidate)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("evidence path is not a regular file")
	}
	if maximum > 0 && info.Size() > maximum {
		return "", fmt.Errorf("evidence file exceeds %d bytes", maximum)
	}
	return resolvedCandidate, nil
}

func decodeRPCParams(params map[string]interface{}, target any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}
	return nil
}
