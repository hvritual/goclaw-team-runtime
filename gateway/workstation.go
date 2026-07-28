package gateway

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

func (h *Handler) registerWorkstationMethods() {
	h.registry.Register("runner.register", h.rpcRegisterRunner)
	h.registry.Register("runner.list", h.rpcListRunners)
	h.registry.Register("runner.ping", h.rpcPingRunner)
	h.registry.Register("runner.update", h.rpcUpdateRunner)
	h.registry.Register("runner.key.rotate", h.rpcRotateRunnerKey)
	h.registry.Register("runner.enqueue", h.rpcEnqueueRunnerTask)
	h.registry.Register("runner.claim", h.rpcClaimRunnerTask)
	h.registry.Register("runner.heartbeat", h.rpcHeartbeatRunnerTask)
	h.registry.Register("runner.complete", h.rpcCompleteRunnerTask)
	h.registry.Register("runner.fail", h.rpcFailRunnerTask)
	h.registry.Register("runner.requeue", h.rpcRequeueRunnerTask)
	h.registry.Register("runner.cancel", h.rpcCancelRunnerTask)
	h.registry.Register("runner.tasks", h.rpcListRunnerTasks)
	h.registry.Register("runner.status", h.rpcRunnerStatus)
	h.registry.Register("runner.evidence", h.rpcRunnerEvidence)
}

func (h *Handler) rpcUpdateRunner(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	var request struct {
		RunnerID string `json:"runner_id"`
		workstation.UpdateRunnerRequest
	}
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	userID, _, err := h.requireOwnedRunner(
		sessionID,
		request.RunnerID,
		"",
		teamcontrol.ActionWorkItemWrite,
	)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil && request.Projects != nil {
		for _, projectID := range request.Projects {
			if strings.TrimSpace(projectID) == "*" {
				return nil, fmt.Errorf(
					"%w: wildcard runner projects are disabled in team mode",
					teamcontrol.ErrForbidden,
				)
			}
			if err := h.teamSvc.Authorize(
				userID,
				projectID,
				teamcontrol.ActionWorkItemWrite,
			); err != nil {
				return nil, err
			}
		}
	}
	return h.runnerSvc.UpdateRunner(
		request.RunnerID,
		request.UpdateRunnerRequest,
	)
}

func (h *Handler) rpcRegisterRunner(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var request struct {
		workstation.RegisterRunnerRequest
		DeviceKey string `json:"device_key"`
	}
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	request.OwnerUserID = userID
	if len(request.Projects) == 0 {
		return nil, fmt.Errorf("projects is required")
	}
	if h.teamSvc != nil {
		for _, projectID := range request.Projects {
			if projectID == "*" {
				return nil, fmt.Errorf("wildcard runner projects are forbidden in team mode")
			}
			if err := h.teamSvc.Authorize(
				userID,
				projectID,
				teamcontrol.ActionWorkItemWrite,
			); err != nil {
				return nil, err
			}
		}
	}
	deviceKey, err := decodeDeviceKey(request.DeviceKey)
	if err != nil {
		return nil, err
	}
	return h.runnerSvc.RegisterRunner(request.RegisterRunnerRequest, deviceKey)
}

func (h *Handler) rpcListRunners(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	if h.teamSvc != nil {
		if err := h.teamSvc.Authorize(
			userID,
			projectID,
			teamcontrol.ActionProjectRead,
		); err != nil {
			return nil, err
		}
	}
	runners, err := h.runnerSvc.ListRunners()
	if err != nil {
		return nil, err
	}
	tasks, _ := h.runnerSvc.ListTasks(workstation.TaskFilter{ProjectID: projectID})
	leases := make(map[string]workstation.Task)
	for _, task := range tasks {
		if task.Status == workstation.TaskLeased && task.Lease != nil {
			leases[task.Lease.RunnerID] = task
		}
	}
	result := make([]map[string]interface{}, 0, len(runners))
	for _, runner := range runners {
		if projectID != "" && !runnerHasProject(runner, projectID) {
			continue
		}
		status := string(runner.Status)
		if _, busy := leases[runner.ID]; busy {
			status = "busy"
		} else if runner.Status == workstation.RunnerDisabled {
			status = "draining"
		}
		item := map[string]interface{}{
			"id":           runner.ID,
			"member_id":    runner.OwnerUserID,
			"display_name": runner.Name,
			"status":       status,
			"capabilities": runner.Capabilities,
			"metadata":     runner.Metadata,
			"last_seen_at": runner.LastHeartbeatAt,
		}
		if task, busy := leases[runner.ID]; busy && task.Lease != nil {
			item["current_work_id"] = task.ID
			item["lease"] = map[string]interface{}{
				"id":          task.Lease.ID,
				"work_id":     task.ID,
				"acquired_at": task.Lease.ClaimedAt,
				"renewed_at":  task.Lease.HeartbeatAt,
				"expires_at":  task.Lease.ExpiresAt,
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (h *Handler) rpcPingRunner(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	_, runner, err := h.requireOwnedRunner(
		sessionID,
		stringParam(params["runner_id"]),
		"",
		teamcontrol.ActionProjectRead,
	)
	if err != nil {
		return nil, err
	}
	return h.runnerSvc.HeartbeatRunner(runner.ID)
}

func (h *Handler) rpcRotateRunnerKey(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	_, runner, err := h.requireOwnedRunner(
		sessionID,
		stringParam(params["runner_id"]),
		"",
		teamcontrol.ActionProjectRead,
	)
	if err != nil {
		return nil, err
	}
	deviceKey, err := decodeDeviceKey(stringParam(params["device_key"]))
	if err != nil {
		return nil, err
	}
	return h.runnerSvc.RotateRunnerDeviceKey(runner.ID, deviceKey)
}

func (h *Handler) rpcEnqueueRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	if h.teamSvc != nil {
		return nil, fmt.Errorf(
			"%w: raw runner.enqueue is disabled in team mode; enqueue an approved frozen task with dev.task.enqueue",
			teamcontrol.ErrForbidden,
		)
	}
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var request workstation.EnqueueRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		if err := h.teamSvc.Authorize(
			userID,
			request.ProjectID,
			teamcontrol.ActionProjectManage,
		); err != nil {
			return nil, err
		}
		repository, err := h.teamSvc.GetRepository(
			userID,
			request.ProjectID,
			request.ExecutionPack.RepositoryID,
		)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(repository.RemoteURL) == "" {
			return nil, fmt.Errorf(
				"repository %q requires remote_url for workstation execution",
				repository.ID,
			)
		}
		for _, issueID := range request.ExecutionPack.IssueIDs {
			if _, err := h.teamSvc.GetIssue(userID, request.ProjectID, issueID); err != nil {
				return nil, err
			}
		}
		for _, workItemID := range request.ExecutionPack.WorkItemIDs {
			if _, err := h.teamSvc.GetWorkItem(
				userID,
				request.ProjectID,
				workItemID,
			); err != nil {
				return nil, err
			}
		}
		policy, err := h.teamSvc.ResolvePolicy(
			userID,
			request.ProjectID,
			repository.ID,
			"",
		)
		if err != nil {
			return nil, err
		}
		request.ExecutionPack.ProjectID = request.ProjectID
		request.ExecutionPack.RepositoryID = repository.ID
		request.ExecutionPack.RepositoryURL = repository.RemoteURL
		request.ExecutionPack.PolicyBundleHash = policy.Hash
		if strings.TrimSpace(request.ExecutionPack.BaseRef) == "" {
			request.ExecutionPack.BaseRef = repository.DefaultBranch
		}
	}
	task, err := h.runnerSvc.Enqueue(request)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		if err := h.registerQueueTraceability(userID, task); err != nil {
			return nil, err
		}
		if err := h.transitionExecutionResources(
			userID,
			task,
			executionTransitionStart,
		); err != nil {
			return nil, err
		}
	}
	return task, nil
}

func (h *Handler) rpcClaimRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	var request workstation.ClaimRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	if h.teamSvc != nil && request.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required in team mode")
	}
	_, _, err := h.requireOwnedRunner(
		sessionID,
		request.RunnerID,
		request.ProjectID,
		teamcontrol.ActionWorkItemWrite,
	)
	if err != nil {
		return nil, err
	}
	return h.runnerSvc.Claim(request)
}

func (h *Handler) rpcHeartbeatRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	var request workstation.HeartbeatRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	task, err := h.runnerSvc.GetTask(request.TaskID)
	if err != nil {
		return nil, err
	}
	if _, _, err := h.requireOwnedRunner(
		sessionID,
		request.RunnerID,
		task.ProjectID,
		teamcontrol.ActionWorkItemWrite,
	); err != nil {
		return nil, err
	}
	return h.runnerSvc.Heartbeat(request)
}

func (h *Handler) rpcCompleteRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	var request workstation.CompleteRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	current, err := h.runnerSvc.GetTask(request.TaskID)
	if err != nil {
		return nil, err
	}
	if developmentTaskID := strings.TrimSpace(
		current.ExecutionPack.Metadata["dev_task_id"],
	); developmentTaskID != "" {
		unlock := h.lockDevelopmentTask(developmentTaskID)
		defer unlock()
	}
	userID, _, err := h.requireOwnedRunner(
		sessionID,
		request.RunnerID,
		current.ProjectID,
		teamcontrol.ActionWorkItemWrite,
	)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		if err := h.teamSvc.Authorize(
			userID,
			current.ProjectID,
			teamcontrol.ActionArtifactWrite,
		); err != nil {
			return nil, err
		}
	}
	task, err := h.runnerSvc.Complete(request)
	if err != nil {
		return nil, err
	}
	var imported *dev.Task
	if developmentTaskID := strings.TrimSpace(
		task.ExecutionPack.Metadata["dev_task_id"],
	); developmentTaskID != "" {
		if h.devSvc == nil {
			return nil, fmt.Errorf(
				"workstation task %q references development task %q but Orchestrator Lite is disabled",
				task.ID,
				developmentTaskID,
			)
		}
		result, importErr := h.importDevelopmentEvidence(
			context.Background(),
			userID,
			task,
			request.Evidence,
			request.Summary,
		)
		if importErr != nil {
			return nil, fmt.Errorf(
				"import workstation evidence into development task %q: %w",
				developmentTaskID,
				importErr,
			)
		}
		imported = &result
	}
	if h.teamSvc != nil {
		if err := h.registerEvidenceTraceability(userID, task); err != nil {
			return nil, err
		}
		transition := executionTransitionComplete
		if imported != nil &&
			(imported.LastGate == nil || !imported.LastGate.Passed) {
			transition = executionTransitionFail
		}
		if err := h.transitionExecutionResources(
			userID,
			task,
			transition,
		); err != nil {
			return nil, err
		}
	}
	return task, nil
}

func (h *Handler) importDevelopmentEvidence(
	ctx context.Context,
	verifiedBy string,
	task workstation.Task,
	bundle workstation.EvidenceBundle,
	summary string,
) (dev.Task, error) {
	if task.Result == nil {
		return dev.Task{}, fmt.Errorf("completed workstation task has no result")
	}
	checks := make([]dev.ImportedEvidenceCheck, 0, len(bundle.Checks))
	for _, check := range bundle.Checks {
		checks = append(checks, dev.ImportedEvidenceCheck{
			Name:       check.Name,
			Passed:     check.Passed,
			ExitCode:   check.ExitCode,
			Details:    check.Details,
			DurationMS: check.DurationMS,
			Artifacts:  append([]string(nil), check.Artifacts...),
		})
	}
	artifacts := make(
		[]dev.ImportedEvidenceArtifact,
		0,
		len(bundle.Artifacts),
	)
	for _, artifact := range bundle.Artifacts {
		artifacts = append(artifacts, dev.ImportedEvidenceArtifact{
			Name:      artifact.Name,
			URI:       artifact.URI,
			SHA256:    artifact.SHA256,
			SizeBytes: artifact.SizeBytes,
		})
	}
	verifiedAt := task.Result.Evidence.VerifiedAt
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	return h.devSvc.ImportExecutionEvidence(
		ctx,
		dev.ImportExecutionEvidenceInput{
			TaskID:              task.ExecutionPack.Metadata["dev_task_id"],
			ProjectID:           task.ProjectID,
			TaskRevision:        task.ExecutionPack.TaskRevision,
			ExecutionBundleHash: task.ExecutionPack.Metadata["execution_bundle_hash"],
			DiffPatch:           bundle.DiffPatch,
			Evidence: dev.ImportedExecutionEvidence{
				Source:              "workstation",
				ExecutionPackSHA256: bundle.ExecutionPackSHA256,
				RunnerID:            bundle.RunnerID,
				LeaseID:             bundle.LeaseID,
				Attempt:             bundle.Attempt,
				Outcome:             bundle.Outcome,
				Summary:             strings.TrimSpace(summary),
				StartedAt:           bundle.StartedAt,
				FinishedAt:          bundle.FinishedAt,
				BaseCommit:          bundle.BaseCommit,
				HeadCommit:          bundle.HeadCommit,
				CommitSHA:           bundle.CommitSHA,
				Branch:              bundle.Branch,
				ChangedFiles:        append([]string(nil), bundle.ChangedFiles...),
				DiffSHA256:          bundle.DiffSHA256,
				Checks:              checks,
				Artifacts:           artifacts,
				TraceIDs:            append([]string(nil), bundle.TraceIDs...),
				KeyID:               task.Result.Evidence.KeyID,
				SignatureAlgorithm:  task.Result.Evidence.SignatureAlgorithm,
				BundleSHA256:        task.Result.Evidence.BundleSHA256,
				Signature:           task.Result.Evidence.Signature,
				VerifiedAt:          verifiedAt,
				VerifiedBy:          "gateway/workstation:" + verifiedBy,
			},
		},
	)
}

func (h *Handler) rpcFailRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	var request workstation.FailRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	task, err := h.runnerSvc.GetTask(request.TaskID)
	if err != nil {
		return nil, err
	}
	if developmentTaskID := strings.TrimSpace(
		task.ExecutionPack.Metadata["dev_task_id"],
	); developmentTaskID != "" {
		unlock := h.lockDevelopmentTask(developmentTaskID)
		defer unlock()
	}
	if _, _, err := h.requireOwnedRunner(
		sessionID,
		request.RunnerID,
		task.ProjectID,
		teamcontrol.ActionWorkItemWrite,
	); err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		userID, principalErr := h.principalID(sessionID)
		if principalErr != nil {
			return nil, principalErr
		}
		if err := h.teamSvc.Authorize(
			userID,
			task.ProjectID,
			teamcontrol.ActionArtifactWrite,
		); err != nil {
			return nil, err
		}
	}
	failed, err := h.runnerSvc.Fail(request)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		userID, principalErr := h.principalID(sessionID)
		if principalErr != nil {
			return nil, principalErr
		}
		if err := h.transitionExecutionResources(
			userID,
			failed,
			executionTransitionFail,
		); err != nil {
			return nil, err
		}
	}
	return failed, nil
}

func (h *Handler) rpcRequeueRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var request workstation.RequeueRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	task, err := h.runnerSvc.GetTask(request.TaskID)
	if err != nil {
		return nil, err
	}
	if developmentTaskID := strings.TrimSpace(
		task.ExecutionPack.Metadata["dev_task_id"],
	); developmentTaskID != "" {
		unlock := h.lockDevelopmentTask(developmentTaskID)
		defer unlock()
	}
	if h.teamSvc != nil {
		action := teamcontrol.ActionProjectManage
		if !request.Force &&
			strings.TrimSpace(task.ExecutionPack.Metadata["assignee_id"]) == userID {
			action = teamcontrol.ActionWorkItemWrite
		}
		if err := h.teamSvc.Authorize(userID, task.ProjectID, action); err != nil {
			if request.Force {
				return nil, fmt.Errorf(
					"forced runner requeue requires project management permission: %w",
					err,
				)
			}
			return nil, fmt.Errorf(
				"only the frozen task assignee or a project manager may requeue: %w",
				err,
			)
		}
	}
	request.Actor = userID
	requeued, err := h.runnerSvc.Requeue(request)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		if err := h.transitionExecutionResources(
			userID,
			requeued,
			executionTransitionRequeue,
		); err != nil {
			return nil, err
		}
	}
	return requeued, nil
}

func (h *Handler) rpcCancelRunnerTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	var request workstation.CancelRequest
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	task, err := h.runnerSvc.GetTask(request.TaskID)
	if err != nil {
		return nil, err
	}
	if developmentTaskID := strings.TrimSpace(
		task.ExecutionPack.Metadata["dev_task_id"],
	); developmentTaskID != "" {
		unlock := h.lockDevelopmentTask(developmentTaskID)
		defer unlock()
	}
	if h.teamSvc != nil {
		action := teamcontrol.ActionProjectManage
		if strings.TrimSpace(
			task.ExecutionPack.Metadata["assignee_id"],
		) == userID {
			action = teamcontrol.ActionWorkItemWrite
		}
		if err := h.teamSvc.Authorize(
			userID,
			task.ProjectID,
			action,
		); err != nil {
			return nil, fmt.Errorf(
				"only the frozen task assignee or a project manager may cancel queued work: %w",
				err,
			)
		}
	}
	request.Actor = userID
	cancelled, err := h.runnerSvc.Cancel(request)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		if err := h.transitionExecutionResources(
			userID,
			cancelled,
			executionTransitionFail,
		); err != nil {
			return nil, err
		}
	}
	return cancelled, nil
}

func (h *Handler) rpcListRunnerTasks(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	if h.teamSvc != nil {
		if err := h.teamSvc.Authorize(
			userID,
			projectID,
			teamcontrol.ActionWorkItemRead,
		); err != nil {
			return nil, err
		}
	}
	return h.runnerSvc.ListTasks(workstation.TaskFilter{
		ProjectID: projectID,
		Status:    workstation.TaskStatus(stringParam(params["status"])),
		RunnerID:  stringParam(params["runner_id"]),
	})
}

func (h *Handler) rpcRunnerStatus(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	projectID := stringParam(params["project_id"])
	if h.teamSvc != nil {
		if err := h.teamSvc.Authorize(
			userID,
			projectID,
			teamcontrol.ActionProjectRead,
		); err != nil {
			return nil, err
		}
	}
	return h.runnerSvc.Status(projectID)
}

func (h *Handler) rpcRunnerEvidence(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return nil, err
	}
	taskID := stringParam(params["task_id"])
	task, err := h.runnerSvc.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	if h.teamSvc != nil {
		if err := h.teamSvc.Authorize(
			userID,
			task.ProjectID,
			teamcontrol.ActionArtifactRead,
		); err != nil {
			return nil, err
		}
	}
	return h.runnerSvc.GetEvidenceBundle(taskID)
}

func (h *Handler) requireOwnedRunner(
	sessionID, runnerID, projectID string,
	action teamcontrol.Action,
) (string, workstation.Runner, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return "", workstation.Runner{}, err
	}
	runner, err := h.runnerSvc.GetRunner(runnerID)
	if err != nil {
		return "", workstation.Runner{}, err
	}
	if h.teamSvc != nil && runner.OwnerUserID != userID {
		return "", workstation.Runner{}, fmt.Errorf(
			"%w: runner belongs to another user",
			teamcontrol.ErrForbidden,
		)
	}
	if projectID != "" {
		if !runnerHasProject(runner, projectID) {
			return "", workstation.Runner{}, fmt.Errorf(
				"%w: runner is not provisioned for project",
				teamcontrol.ErrForbidden,
			)
		}
		if h.teamSvc != nil {
			if err := h.teamSvc.Authorize(userID, projectID, action); err != nil {
				return "", workstation.Runner{}, err
			}
		}
	}
	return userID, runner, nil
}

func (h *Handler) projectRunners(projectID string) []workstation.Runner {
	if h.runnerSvc == nil {
		return nil
	}
	runners, err := h.runnerSvc.ListRunners()
	if err != nil {
		return nil
	}
	result := make([]workstation.Runner, 0, len(runners))
	for _, runner := range runners {
		if runnerHasProject(runner, projectID) {
			result = append(result, runner)
		}
	}
	return result
}

func runnerHasProject(runner workstation.Runner, projectID string) bool {
	for _, allowed := range runner.Projects {
		if allowed == projectID || allowed == "*" {
			return true
		}
	}
	return false
}

func decodeDeviceKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("device_key is required")
	}
	decoders := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("device_key must be base64 or base64url")
}

func (h *Handler) registerQueueTraceability(
	actorID string,
	task workstation.Task,
) error {
	_, err := h.teamSvc.RegisterArtifact(actorID, teamcontrol.RegisterArtifactInput{
		ID:           task.ID,
		ProjectID:    task.ProjectID,
		ResourceType: teamcontrol.ResourceTask,
		Kind:         teamcontrol.ArtifactOther,
		Name:         "Workstation task " + task.ID,
		URI:          "goclaw://workstation/tasks/" + task.ID,
		Metadata: map[string]string{
			"execution_pack_sha256": task.ExecutionPackSHA256,
			"repository_id":         task.ExecutionPack.RepositoryID,
			"correlation_id":        task.ExecutionPack.CorrelationID,
			"policy_bundle_hash":    task.ExecutionPack.PolicyBundleHash,
		},
	})
	if err != nil {
		existing, listErr := h.teamSvc.ListArtifacts(actorID, task.ProjectID)
		if listErr != nil || !artifactExists(existing, task.ID) {
			return err
		}
	}
	targets := make([]struct {
		kind     teamcontrol.ResourceType
		id       string
		relation string
	}, 0)
	for _, workItemID := range task.ExecutionPack.WorkItemIDs {
		targets = append(targets, struct {
			kind     teamcontrol.ResourceType
			id       string
			relation string
		}{teamcontrol.ResourceWorkItem, workItemID, "implements"})
	}
	for _, issueID := range task.ExecutionPack.IssueIDs {
		targets = append(targets, struct {
			kind     teamcontrol.ResourceType
			id       string
			relation string
		}{teamcontrol.ResourceIssue, issueID, "addresses"})
	}
	targets = append(targets, struct {
		kind     teamcontrol.ResourceType
		id       string
		relation string
	}{teamcontrol.ResourceRepository, task.ExecutionPack.RepositoryID, "changes"})
	for _, target := range targets {
		if err := h.ensureCorrelation(
			actorID,
			task.ProjectID,
			teamcontrol.ResourceTask,
			task.ID,
			target.kind,
			target.id,
			target.relation,
		); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) registerEvidenceTraceability(
	actorID string,
	task workstation.Task,
) error {
	if task.Result == nil {
		return fmt.Errorf("completed runner task has no evidence reference")
	}
	artifact, err := h.teamSvc.RegisterArtifact(
		actorID,
		teamcontrol.RegisterArtifactInput{
			ProjectID:    task.ProjectID,
			ResourceType: teamcontrol.ResourceArtifact,
			Kind:         teamcontrol.ArtifactEvidence,
			Name:         "Signed evidence for " + task.ID,
			URI: "goclaw://workstation/tasks/" + task.ID +
				"/evidence/" + task.Result.Evidence.BundleSHA256,
			ContentType: "application/json",
			Metadata: map[string]string{
				"runner_id":             task.Result.RunnerID,
				"execution_pack_sha256": task.ExecutionPackSHA256,
				"key_id":                task.Result.Evidence.KeyID,
				"bundle_sha256":         task.Result.Evidence.BundleSHA256,
			},
		},
	)
	if err != nil {
		artifacts, listErr := h.teamSvc.ListArtifacts(actorID, task.ProjectID)
		if listErr != nil {
			return err
		}
		for _, candidate := range artifacts {
			if candidate.URI == "goclaw://workstation/tasks/"+task.ID+
				"/evidence/"+task.Result.Evidence.BundleSHA256 {
				artifact = candidate
				err = nil
				break
			}
		}
		if err != nil {
			return err
		}
	}
	return h.ensureCorrelation(
		actorID,
		task.ProjectID,
		teamcontrol.ResourceTask,
		task.ID,
		teamcontrol.ResourceArtifact,
		artifact.ID,
		"produces",
	)
}

func (h *Handler) ensureCorrelation(
	actorID, projectID string,
	sourceType teamcontrol.ResourceType,
	sourceID string,
	targetType teamcontrol.ResourceType,
	targetID, relation string,
) error {
	links, err := h.teamSvc.ListLinks(actorID, projectID, sourceType, sourceID)
	if err == nil {
		for _, link := range links {
			if link.SourceType == sourceType && link.SourceID == sourceID &&
				link.TargetType == targetType && link.TargetID == targetID &&
				link.Relation == relation {
				return nil
			}
		}
	}
	_, err = h.teamSvc.CreateLink(actorID, teamcontrol.CreateLinkInput{
		ProjectID:  projectID,
		SourceType: sourceType,
		SourceID:   sourceID,
		TargetType: targetType,
		TargetID:   targetID,
		Relation:   relation,
	})
	return err
}

func artifactExists(artifacts []teamcontrol.Artifact, id string) bool {
	for _, artifact := range artifacts {
		if artifact.ID == id {
			return true
		}
	}
	return false
}

type executionTransition string

const (
	executionTransitionStart    executionTransition = "start"
	executionTransitionComplete executionTransition = "complete"
	executionTransitionFail     executionTransition = "fail"
	executionTransitionRequeue  executionTransition = "requeue"
)

func (h *Handler) transitionExecutionResources(
	actorID string,
	task workstation.Task,
	transition executionTransition,
) error {
	if h.teamSvc == nil {
		return nil
	}
	for _, workItemID := range task.ExecutionPack.WorkItemIDs {
		item, err := h.teamSvc.GetWorkItem(actorID, task.ProjectID, workItemID)
		if err != nil {
			return fmt.Errorf("load work item %q for %s: %w", workItemID, transition, err)
		}
		next, shouldTransition, err := workItemTransition(item.Status, transition)
		if err != nil {
			return fmt.Errorf("work item %q: %w", workItemID, err)
		}
		if shouldTransition {
			if _, err := h.teamSvc.TransitionWorkItem(
				actorID,
				task.ProjectID,
				workItemID,
				next,
			); err != nil {
				return fmt.Errorf(
					"transition work item %q for %s: %w",
					workItemID,
					transition,
					err,
				)
			}
		}
	}
	for _, issueID := range task.ExecutionPack.IssueIDs {
		issue, err := h.teamSvc.GetIssue(actorID, task.ProjectID, issueID)
		if err != nil {
			return fmt.Errorf("load issue %q for %s: %w", issueID, transition, err)
		}
		next, shouldTransition := issueTransition(issue.Status, transition)
		if shouldTransition {
			if _, err := h.teamSvc.TransitionIssue(
				actorID,
				task.ProjectID,
				issueID,
				next,
				"",
			); err != nil {
				return fmt.Errorf(
					"transition issue %q for %s: %w",
					issueID,
					transition,
					err,
				)
			}
		}
	}
	return nil
}

func workItemTransition(
	current teamcontrol.WorkItemStatus,
	transition executionTransition,
) (teamcontrol.WorkItemStatus, bool, error) {
	switch transition {
	case executionTransitionStart:
		switch current {
		case teamcontrol.WorkItemReady:
			return teamcontrol.WorkItemInProgress, true, nil
		case teamcontrol.WorkItemBlocked:
			return teamcontrol.WorkItemInProgress, true, nil
		case teamcontrol.WorkItemInProgress:
			return "", false, nil
		}
	case executionTransitionComplete:
		switch current {
		case teamcontrol.WorkItemInProgress:
			return teamcontrol.WorkItemVerifying, true, nil
		case teamcontrol.WorkItemVerifying, teamcontrol.WorkItemDone:
			return "", false, nil
		}
	case executionTransitionFail:
		switch current {
		case teamcontrol.WorkItemInProgress:
			return teamcontrol.WorkItemBlocked, true, nil
		case teamcontrol.WorkItemBlocked:
			return "", false, nil
		}
	case executionTransitionRequeue:
		switch current {
		case teamcontrol.WorkItemBlocked:
			return teamcontrol.WorkItemInProgress, true, nil
		case teamcontrol.WorkItemInProgress:
			return "", false, nil
		}
	}
	return "", false, fmt.Errorf(
		"status %q is incompatible with execution transition %q",
		current,
		transition,
	)
}

func issueTransition(
	current teamcontrol.IssueStatus,
	transition executionTransition,
) (teamcontrol.IssueStatus, bool) {
	switch transition {
	case executionTransitionStart:
		switch current {
		case teamcontrol.IssueTriaged, teamcontrol.IssueAssigned,
			teamcontrol.IssueReopened, teamcontrol.IssueBlocked:
			return teamcontrol.IssueInProgress, true
		}
	case executionTransitionComplete:
		if current == teamcontrol.IssueInProgress {
			return teamcontrol.IssueVerifying, true
		}
	case executionTransitionFail:
		if current == teamcontrol.IssueInProgress {
			return teamcontrol.IssueBlocked, true
		}
	case executionTransitionRequeue:
		if current == teamcontrol.IssueBlocked {
			return teamcontrol.IssueInProgress, true
		}
	}
	return "", false
}
