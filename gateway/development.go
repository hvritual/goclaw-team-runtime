package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/smallnest/goclaw/governance"
	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

func (h *Handler) registerDevelopmentMethods() {
	h.registry.Register("dev.tasks", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		projectID := stringParam(params["project_id"])
		if h.teamSvc != nil {
			if _, err := h.authorizeProject(
				sessionID,
				projectID,
				teamcontrol.ActionWorkItemRead,
			); err != nil {
				return nil, err
			}
		}
		return h.devSvc.ListTasks(projectID)
	})

	h.registry.Register("dev.task.get", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		task, _, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionWorkItemRead,
		)
		return task, err
	})

	h.registry.Register("dev.task.events", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		if _, _, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionWorkItemRead,
		); err != nil {
			return nil, err
		}
		return h.devSvc.ListEvents(id)
	})

	h.registry.Register("dev.task.create", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		var request dev.CreateRequest
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		if h.teamSvc != nil {
			if strings.TrimSpace(request.ID) == "" {
				return nil, fmt.Errorf(
					"id is required in team mode for idempotent task creation",
				)
			}
			userID, err := h.authorizeProject(
				sessionID,
				request.ProjectID,
				teamcontrol.ActionWorkItemWrite,
			)
			if err != nil {
				return nil, err
			}
			if request.Wave == nil {
				return nil, fmt.Errorf(
					"wave binding is required for team development tasks",
				)
			}
			waveStepID := strings.TrimSpace(request.Wave.StepID)
			if waveStepID == "" {
				return nil, fmt.Errorf(
					"wave.step_id is required for team development tasks",
				)
			}
			if strings.TrimSpace(request.RepositoryID) == "" {
				return nil, fmt.Errorf("repository_id is required")
			}
			project, err := h.teamSvc.GetProject(userID, request.ProjectID)
			if err != nil {
				return nil, err
			}
			repository, err := h.teamSvc.GetRepository(
				userID,
				request.ProjectID,
				request.RepositoryID,
			)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(repository.LocalPath) == "" {
				return nil, fmt.Errorf(
					"repository %q has no control-plane local_path",
					repository.ID,
				)
			}
			request.TeamID = project.TeamID
			request.RepoPath = repository.LocalPath
			request.CreatedBy = teamcontrol.PlannerServicePrincipal
			request.RequestedBy = userID
			request.AssigneeID = strings.TrimSpace(request.AssigneeID)
			if request.AssigneeID == "" {
				request.AssigneeID = userID
			} else if request.AssigneeID != userID {
				if err := h.teamSvc.Authorize(
					userID,
					request.ProjectID,
					teamcontrol.ActionWorkItemAssign,
				); err != nil {
					return nil, err
				}
			}
			if err := h.validateProjectAssignee(
				userID,
				request.ProjectID,
				request.AssigneeID,
			); err != nil {
				return nil, err
			}
			if strings.TrimSpace(request.BaseRef) == "" {
				request.BaseRef = repository.DefaultBranch
			}
			waveBinding, baseCommit, err := dev.ResolveWaveBinding(
				context.Background(),
				repository.LocalPath,
				request.BaseRef,
				waveStepID,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"resolve repository Wave binding: %w",
					err,
				)
			}
			request.Wave = &waveBinding
			request.BaseRef = baseCommit
		} else if request.CreatedBy == "" {
			request.CreatedBy = sessionID
		}
		return h.devSvc.CreateTask(request)
	})

	h.registry.Register("dev.task.review", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		unlock := h.lockDevelopmentTask(id)
		defer unlock()
		if _, _, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionArtifactWrite,
		); err != nil {
			return nil, err
		}
		kind := dev.ReviewKind(stringParam(params["kind"]))
		role := governance.RoleScenarioReview
		switch kind {
		case dev.ReviewCapacity:
			role = governance.RoleCapacityReview
		case dev.ReviewRisk:
			role = governance.RoleRiskReview
		case dev.ReviewCost:
			role = governance.RoleCostReview
		}
		review, err := h.humanReview(sessionID, params, role)
		if err != nil {
			return nil, err
		}
		return h.devSvc.ReviewTaskWithReview(
			id,
			kind,
			dev.ReviewDecision(stringParam(params["decision"])),
			review,
		)
	})

	h.registry.Register("dev.task.freeze", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		unlock := h.lockDevelopmentTask(id)
		defer unlock()
		task, actor, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionWorkItemWrite,
		)
		if err != nil {
			return nil, err
		}
		if err := h.authorizeDevelopmentTaskOwner(actor, task); err != nil {
			return nil, err
		}
		if h.teamSvc != nil && task.Wave == nil {
			return nil, fmt.Errorf(
				"team development tasks require an immutable wave binding",
			)
		}
		if h.teamSvc == nil {
			actor, err = h.actorForSession(sessionID, stringParam(params["actor"]))
			if err != nil {
				return nil, err
			}
		}
		return h.devSvc.FreezeTask(context.Background(), id, actor)
	})

	h.registry.Register("dev.task.revise", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		unlock := h.lockDevelopmentTask(id)
		defer unlock()
		task, actor, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionWorkItemWrite,
		)
		if err != nil {
			return nil, err
		}
		if err := h.authorizeDevelopmentTaskOwner(actor, task); err != nil {
			return nil, err
		}
		reason := strings.TrimSpace(stringParam(params["reason"]))
		if reason == "" {
			return nil, fmt.Errorf("reason is required")
		}
		if h.teamSvc != nil {
			expectedRevision := intParam(params["expected_revision"], 0)
			if expectedRevision <= 0 {
				return nil, fmt.Errorf(
					"expected_revision is required in team mode",
				)
			}
			if task.Compile.Revision != expectedRevision {
				return nil, fmt.Errorf(
					"development task revision conflict: expected %d, current %d",
					expectedRevision,
					task.Compile.Revision,
				)
			}
			if err := h.ensureDevelopmentQueueRevisionInactive(task); err != nil {
				return nil, err
			}
		}
		var replacement dev.Task
		if raw, ok := params["replacement"]; ok && raw != nil {
			data, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &replacement); err != nil {
				return nil, fmt.Errorf("invalid replacement task: %w", err)
			}
		}
		if h.teamSvc != nil {
			// Team, repository, assignee and policy identity are server-owned.
			replacement.TeamID = ""
			replacement.ProjectID = ""
			replacement.RepositoryID = ""
			replacement.AssigneeID = ""
			replacement.CreatedBy = ""
			replacement.RepoPath = ""
			if len(replacement.IssueIDs) > 0 &&
				!sameDevelopmentIDs(replacement.IssueIDs, task.IssueIDs) {
				return nil, fmt.Errorf(
					"team repair revisions cannot replace linked issue ids; create or relink resources explicitly",
				)
			}
			if len(replacement.Plan.Milestones) > 0 &&
				!sameDevelopmentIDs(
					developmentWorkItemIDs(replacement),
					developmentWorkItemIDs(task),
				) {
				return nil, fmt.Errorf(
					"team repair revisions cannot replace work item ids; revise the existing work item contract",
				)
			}
			policy, err := h.teamSvc.ResolvePolicy(
				actor,
				task.ProjectID,
				task.RepositoryID,
				"",
			)
			if err != nil {
				return nil, err
			}
			replacement.PolicyBundleHash = policy.Hash
			for _, issueID := range replacement.IssueIDs {
				if _, err := h.teamSvc.GetIssue(
					actor,
					task.ProjectID,
					issueID,
				); err != nil {
					return nil, fmt.Errorf(
						"resolve replacement issue %q: %w",
						issueID,
						err,
					)
				}
			}
			if err := h.prepareDevelopmentResourcesForRevision(
				actor,
				task,
			); err != nil {
				return nil, err
			}
		}
		return h.devSvc.ReviseTask(id, actor, reason, replacement)
	})

	h.registry.Register("dev.task.accept", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		unlock := h.lockDevelopmentTask(id)
		defer unlock()
		task, actor, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionIssueTransition,
		)
		if err != nil {
			return nil, err
		}
		if h.teamSvc != nil {
			if err := h.teamSvc.Authorize(
				actor,
				task.ProjectID,
				teamcontrol.ActionProjectManage,
			); err != nil {
				return nil, fmt.Errorf(
					"task acceptance requires project management permission in addition to the configured task_accept reviewer role: %w",
					err,
				)
			}
			if err := h.requireControlConsistency(
				actor,
				task.ProjectID,
			); err != nil {
				return nil, err
			}
			if err := h.devSvc.ValidateWaveBinding(
				context.Background(),
				task,
			); err != nil {
				return nil, fmt.Errorf(
					"revalidate frozen wave binding before acceptance: %w",
					err,
				)
			}
			if err := h.validateQueuedDevelopmentWave(task); err != nil {
				return nil, err
			}
		}
		review, err := h.humanReview(sessionID, params, governance.RoleTaskAccept)
		if err != nil {
			return nil, err
		}
		if h.teamSvc != nil {
			if err := h.validateAcceptedDevelopmentResources(
				actor,
				task,
			); err != nil {
				return nil, err
			}
		}
		accepted := task
		if task.Status != dev.TaskDone {
			accepted, err = h.devSvc.AcceptTaskWithReview(
				context.Background(),
				id,
				review,
			)
			if err != nil {
				return nil, err
			}
		}
		if h.teamSvc != nil {
			if err := h.transitionAcceptedDevelopmentResources(
				actor,
				accepted,
			); err != nil {
				return nil, err
			}
		}
		return accepted, nil
	})

	h.registry.Register("dev.task.cancel", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		unlock := h.lockDevelopmentTask(id)
		defer unlock()
		task, actor, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionWorkItemWrite,
		)
		if err != nil {
			return nil, err
		}
		if err := h.authorizeDevelopmentTaskOwner(actor, task); err != nil {
			return nil, err
		}
		if h.teamSvc != nil {
			if err := h.teamSvc.Authorize(
				actor,
				task.ProjectID,
				teamcontrol.ActionProjectManage,
			); err != nil {
				return nil, fmt.Errorf(
					"development task cancellation requires project management permission: %w",
					err,
				)
			}
		}
		reason := strings.TrimSpace(stringParam(params["reason"]))
		if reason == "" {
			return nil, fmt.Errorf("reason is required")
		}
		review, err := h.humanReview(sessionID, params, governance.RoleTaskCancel)
		if err != nil {
			return nil, err
		}
		if h.teamSvc != nil {
			if err := h.validateDevelopmentResourcesCancellable(actor, task); err != nil {
				return nil, err
			}
			if err := h.cancelQueuedDevelopmentRevision(
				actor,
				task,
				reason,
			); err != nil {
				return nil, err
			}
		}
		cancelled := task
		if task.Status != dev.TaskCancelled {
			cancelled, err = h.devSvc.CancelTaskWithReview(
				id,
				reason,
				review,
			)
			if err != nil {
				return nil, err
			}
		}
		if h.teamSvc != nil {
			if err := h.transitionCancelledDevelopmentResources(
				actor,
				cancelled,
			); err != nil {
				return nil, err
			}
		}
		return cancelled, nil
	})

	h.registry.Register("dev.task.run", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return h.queueDevelopmentRun(sessionID, params, "run")
	})
	h.registry.Register("dev.task.repair", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return h.queueDevelopmentRun(sessionID, params, "repair")
	})
	h.registry.Register("dev.task.resume", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return h.queueDevelopmentRun(sessionID, params, "resume")
	})

	h.registry.Register("dev.task.link-pr", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		unlock := h.lockDevelopmentTask(id)
		defer unlock()
		task, actor, err := h.authorizedDevelopmentTask(
			sessionID,
			id,
			teamcontrol.ActionArtifactWrite,
		)
		if err != nil {
			return nil, err
		}
		if err := h.authorizeDevelopmentTaskOwner(actor, task); err != nil {
			return nil, err
		}
		commitSHA := stringParam(params["commit_sha"])
		if h.teamSvc != nil && strings.TrimSpace(commitSHA) == "" {
			return nil, fmt.Errorf(
				"commit_sha is required for external pull request linking in team mode",
			)
		}
		var linked dev.Task
		if commitSHA != "" {
			linked, err = h.devSvc.RecordImportedPullRequest(
				context.Background(),
				id,
				actor,
				commitSHA,
				stringParam(params["url"]),
			)
		} else {
			linked, err = h.devSvc.RecordPullRequest(
				id,
				actor,
				stringParam(params["url"]),
			)
		}
		if err != nil {
			return nil, err
		}
		if h.teamSvc != nil && commitSHA != "" {
			if err := h.registerExternalDevelopmentTraceability(
				actor,
				task,
				linked,
			); err != nil {
				return nil, err
			}
		}
		return linked, nil
	})

	h.registry.Register("dev.task.enqueue", h.rpcEnqueueDevelopmentTask)
}

func (h *Handler) transitionAcceptedDevelopmentResources(
	actorID string,
	task dev.Task,
) error {
	for _, workItemID := range developmentWorkItemIDs(task) {
		item, err := h.teamSvc.GetWorkItem(
			actorID,
			task.ProjectID,
			workItemID,
		)
		if err != nil {
			return err
		}
		switch item.Status {
		case teamcontrol.WorkItemVerifying:
			if _, err := h.teamSvc.TransitionWorkItem(
				actorID,
				task.ProjectID,
				item.ID,
				teamcontrol.WorkItemDone,
			); err != nil {
				return err
			}
		case teamcontrol.WorkItemDone:
		default:
			return fmt.Errorf(
				"work item %q cannot close after task acceptance from status %q",
				item.ID,
				item.Status,
			)
		}
	}
	for _, issueID := range task.IssueIDs {
		ready, err := h.developmentIssueReadyToResolve(
			actorID,
			task,
			issueID,
		)
		if err != nil {
			return err
		}
		if !ready {
			continue
		}
		issue, err := h.teamSvc.GetIssue(
			actorID,
			task.ProjectID,
			issueID,
		)
		if err != nil {
			return err
		}
		switch issue.Status {
		case teamcontrol.IssueVerifying:
		case teamcontrol.IssueInProgress:
			issue, err = h.teamSvc.TransitionIssue(
				actorID,
				task.ProjectID,
				issue.ID,
				teamcontrol.IssueVerifying,
				"",
			)
			if err != nil {
				return err
			}
		case teamcontrol.IssueBlocked:
			issue, err = h.teamSvc.TransitionIssue(
				actorID,
				task.ProjectID,
				issue.ID,
				teamcontrol.IssueInProgress,
				"",
			)
			if err != nil {
				return err
			}
			issue, err = h.teamSvc.TransitionIssue(
				actorID,
				task.ProjectID,
				issue.ID,
				teamcontrol.IssueVerifying,
				"",
			)
			if err != nil {
				return err
			}
		case teamcontrol.IssueResolved, teamcontrol.IssueClosed:
			continue
		default:
			return fmt.Errorf(
				"issue %q cannot resolve after task acceptance from status %q",
				issue.ID,
				issue.Status,
			)
		}
		if _, err := h.teamSvc.TransitionIssue(
			actorID,
			task.ProjectID,
			issue.ID,
			teamcontrol.IssueResolved,
			"All linked development work was accepted after DoneGate evidence review; final task "+task.ID+".",
		); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) validateAcceptedDevelopmentResources(
	actorID string,
	task dev.Task,
) error {
	for _, workItemID := range developmentWorkItemIDs(task) {
		item, err := h.teamSvc.GetWorkItem(
			actorID,
			task.ProjectID,
			workItemID,
		)
		if err != nil {
			return err
		}
		switch item.Status {
		case teamcontrol.WorkItemVerifying, teamcontrol.WorkItemDone:
		default:
			return fmt.Errorf(
				"work item %q cannot close after task acceptance from status %q",
				item.ID,
				item.Status,
			)
		}
	}
	for _, issueID := range task.IssueIDs {
		issue, err := h.teamSvc.GetIssue(
			actorID,
			task.ProjectID,
			issueID,
		)
		if err != nil {
			return err
		}
		if issue.Status == teamcontrol.IssueCancelled {
			return fmt.Errorf(
				"cancelled issue %q cannot be resolved by task acceptance",
				issue.ID,
			)
		}
	}
	return nil
}

func (h *Handler) developmentIssueReadyToResolve(
	actorID string,
	task dev.Task,
	issueID string,
) (bool, error) {
	if task.Status != dev.TaskDone {
		return false, nil
	}
	tasks, err := h.devSvc.ListTasks(task.ProjectID)
	if err != nil {
		return false, err
	}
	for _, candidate := range tasks {
		if candidate.ID == task.ID ||
			!containsDevelopmentID(candidate.IssueIDs, issueID) {
			continue
		}
		if candidate.Status != dev.TaskDone {
			return false, nil
		}
	}
	workItems, err := h.teamSvc.ListWorkItems(actorID, task.ProjectID)
	if err != nil {
		return false, err
	}
	for _, item := range workItems {
		if item.IssueID != issueID {
			continue
		}
		if item.Status != teamcontrol.WorkItemDone {
			return false, nil
		}
	}
	return true, nil
}

func containsDevelopmentID(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sameDevelopmentIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]int, len(left))
	for _, value := range left {
		seen[value]++
	}
	for _, value := range right {
		if seen[value] == 0 {
			return false
		}
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func developmentQueueRevisionID(task dev.Task) string {
	return fmt.Sprintf("%s-r%d", task.ID, task.Compile.Revision)
}

func (h *Handler) ensureDevelopmentQueueRevisionInactive(task dev.Task) error {
	if h.runnerSvc == nil {
		return nil
	}
	queueID := developmentQueueRevisionID(task)
	queued, err := h.runnerSvc.GetTask(queueID)
	if errors.Is(err, workstation.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	switch queued.Status {
	case workstation.TaskQueued:
		return fmt.Errorf(
			"development revision %d is still queued as %s; cancel it with `goclaw runner cancel %s --reason <reason>` before revising",
			task.Compile.Revision,
			queueID,
			queueID,
		)
	case workstation.TaskLeased:
		return fmt.Errorf(
			"development revision %d is actively leased as %s; wait for the runner to finish or fail before revising",
			task.Compile.Revision,
			queueID,
		)
	default:
		return nil
	}
}

func (h *Handler) prepareDevelopmentResourcesForRevision(
	actorID string,
	task dev.Task,
) error {
	type transition struct {
		id   string
		from teamcontrol.WorkItemStatus
	}
	var transitions []transition
	for _, workItemID := range developmentWorkItemIDs(task) {
		item, err := h.teamSvc.GetWorkItem(
			actorID,
			task.ProjectID,
			workItemID,
		)
		if err != nil {
			return err
		}
		switch item.Status {
		case teamcontrol.WorkItemInProgress,
			teamcontrol.WorkItemVerifying:
			transitions = append(transitions, transition{
				id:   item.ID,
				from: item.Status,
			})
		case teamcontrol.WorkItemPending,
			teamcontrol.WorkItemReady,
			teamcontrol.WorkItemBlocked:
		default:
			return fmt.Errorf(
				"work item %q cannot enter a repair revision from status %q",
				item.ID,
				item.Status,
			)
		}
	}
	for _, item := range transitions {
		if _, err := h.teamSvc.TransitionWorkItem(
			actorID,
			task.ProjectID,
			item.id,
			teamcontrol.WorkItemBlocked,
		); err != nil {
			return fmt.Errorf(
				"block work item %q before revising from %q: %w",
				item.id,
				item.from,
				err,
			)
		}
	}
	return nil
}

func (h *Handler) cancelQueuedDevelopmentRevision(
	actorID string,
	task dev.Task,
	reason string,
) error {
	if h.runnerSvc == nil {
		return nil
	}
	queueID := developmentQueueRevisionID(task)
	queued, err := h.runnerSvc.GetTask(queueID)
	if errors.Is(err, workstation.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	switch queued.Status {
	case workstation.TaskLeased:
		return fmt.Errorf(
			"workstation task %s has an active lease; wait for the runner to finish or fail before cancelling the development task",
			queueID,
		)
	case workstation.TaskQueued, workstation.TaskFailed:
		cancelled, err := h.runnerSvc.Cancel(workstation.CancelRequest{
			TaskID: queueID,
			Actor:  actorID,
			Reason: reason,
			IdempotencyKey: "dev-task-cancel:" + task.ID +
				fmt.Sprintf(":revision:%d", task.Compile.Revision),
		})
		if err != nil {
			return err
		}
		if h.teamSvc != nil {
			return h.transitionExecutionResources(
				actorID,
				cancelled,
				executionTransitionFail,
			)
		}
	}
	return nil
}

func (h *Handler) validateDevelopmentResourcesCancellable(
	actorID string,
	task dev.Task,
) error {
	for _, workItemID := range developmentWorkItemIDs(task) {
		item, err := h.teamSvc.GetWorkItem(
			actorID,
			task.ProjectID,
			workItemID,
		)
		if err != nil {
			return err
		}
		switch item.Status {
		case teamcontrol.WorkItemPending,
			teamcontrol.WorkItemReady,
			teamcontrol.WorkItemInProgress,
			teamcontrol.WorkItemBlocked,
			teamcontrol.WorkItemVerifying,
			teamcontrol.WorkItemCancelled:
		default:
			return fmt.Errorf(
				"work item %q cannot cancel with development task from status %q",
				item.ID,
				item.Status,
			)
		}
	}
	for _, issueID := range task.IssueIDs {
		issue, err := h.teamSvc.GetIssue(
			actorID,
			task.ProjectID,
			issueID,
		)
		if err != nil {
			return err
		}
		switch issue.Status {
		case teamcontrol.IssueNew,
			teamcontrol.IssueTriaged,
			teamcontrol.IssueAssigned,
			teamcontrol.IssueInProgress,
			teamcontrol.IssueBlocked,
			teamcontrol.IssueVerifying,
			teamcontrol.IssueReopened,
			teamcontrol.IssueCancelled:
		case teamcontrol.IssueResolved, teamcontrol.IssueClosed:
			return fmt.Errorf(
				"resolved issue %q cannot be changed by cancelling a development task",
				issue.ID,
			)
		default:
			return fmt.Errorf(
				"issue %q cannot cancel with development task from status %q",
				issue.ID,
				issue.Status,
			)
		}
	}
	return nil
}

func (h *Handler) transitionCancelledDevelopmentResources(
	actorID string,
	task dev.Task,
) error {
	for _, workItemID := range developmentWorkItemIDs(task) {
		item, err := h.teamSvc.GetWorkItem(
			actorID,
			task.ProjectID,
			workItemID,
		)
		if err != nil {
			return err
		}
		if item.Status == teamcontrol.WorkItemCancelled {
			continue
		}
		if item.Status == teamcontrol.WorkItemVerifying {
			item, err = h.teamSvc.TransitionWorkItem(
				actorID,
				task.ProjectID,
				item.ID,
				teamcontrol.WorkItemBlocked,
			)
			if err != nil {
				return err
			}
		}
		if _, err := h.teamSvc.TransitionWorkItem(
			actorID,
			task.ProjectID,
			item.ID,
			teamcontrol.WorkItemCancelled,
		); err != nil {
			return err
		}
	}
	for _, issueID := range task.IssueIDs {
		hasOtherActive, err := h.developmentIssueHasOtherActiveTask(
			task,
			issueID,
		)
		if err != nil {
			return err
		}
		if hasOtherActive {
			continue
		}
		issue, err := h.teamSvc.GetIssue(
			actorID,
			task.ProjectID,
			issueID,
		)
		if err != nil {
			return err
		}
		switch issue.Status {
		case teamcontrol.IssueInProgress, teamcontrol.IssueVerifying:
			if _, err := h.teamSvc.TransitionIssue(
				actorID,
				task.ProjectID,
				issue.ID,
				teamcontrol.IssueBlocked,
				"",
			); err != nil {
				return err
			}
		case teamcontrol.IssueNew,
			teamcontrol.IssueTriaged,
			teamcontrol.IssueAssigned,
			teamcontrol.IssueBlocked,
			teamcontrol.IssueReopened,
			teamcontrol.IssueCancelled:
			// Cancelling one implementation task must never terminally cancel
			// the product issue. It remains open for reassignment or triage.
		default:
			return fmt.Errorf(
				"issue %q cannot remain open after task cancellation from status %q",
				issue.ID,
				issue.Status,
			)
		}
	}
	return nil
}

func (h *Handler) developmentIssueHasOtherActiveTask(
	task dev.Task,
	issueID string,
) (bool, error) {
	tasks, err := h.devSvc.ListTasks(task.ProjectID)
	if err != nil {
		return false, err
	}
	for _, candidate := range tasks {
		if candidate.ID == task.ID ||
			!containsDevelopmentID(candidate.IssueIDs, issueID) {
			continue
		}
		if candidate.Status != dev.TaskDone &&
			candidate.Status != dev.TaskCancelled {
			return true, nil
		}
	}
	return false, nil
}

func (h *Handler) registerExternalDevelopmentTraceability(
	actorID string,
	before, linked dev.Task,
) error {
	evidenceSHA256 := ""
	worktreeTreeSHA := ""
	if linked.LastGate != nil {
		evidenceSHA256 = linked.LastGate.EvidenceSHA256
		worktreeTreeSHA = linked.LastGate.WorktreeTreeSHA
	}
	commitArtifact, err := h.registerDevelopmentArtifact(
		actorID,
		teamcontrol.RegisterArtifactInput{
			ID:           "commit-" + linked.CommitSHA,
			ProjectID:    linked.ProjectID,
			ResourceType: teamcontrol.ResourceCommit,
			Kind:         teamcontrol.ArtifactCommit,
			Name:         "Accepted commit " + linked.CommitSHA,
			URI: "goclaw://repositories/" + linked.RepositoryID +
				"/commits/" + linked.CommitSHA,
			Metadata: map[string]string{
				"task_id":         linked.ID,
				"task_revision":   fmt.Sprintf("%d", linked.Compile.Revision),
				"repository_id":   linked.RepositoryID,
				"base_commit":     linked.Compile.BaseCommit,
				"evidence_sha256": evidenceSHA256,
				"worktree_tree":   worktreeTreeSHA,
			},
		},
	)
	if err != nil {
		return err
	}
	pullRequestArtifact, err := h.registerDevelopmentArtifact(
		actorID,
		teamcontrol.RegisterArtifactInput{
			ID:           "pr-" + linked.CommitSHA,
			ProjectID:    linked.ProjectID,
			ResourceType: teamcontrol.ResourcePullRequest,
			Kind:         teamcontrol.ArtifactPR,
			Name:         "Pull request for " + linked.ID,
			URI:          linked.PullRequestURL,
			Metadata: map[string]string{
				"task_id":       linked.ID,
				"task_revision": fmt.Sprintf("%d", linked.Compile.Revision),
				"repository_id": linked.RepositoryID,
				"commit_sha":    linked.CommitSHA,
			},
		},
	)
	if err != nil {
		return err
	}
	queueTaskID := fmt.Sprintf("%s-r%d", before.ID, before.Compile.Revision)
	for _, link := range []struct {
		sourceType teamcontrol.ResourceType
		sourceID   string
		targetType teamcontrol.ResourceType
		targetID   string
		relation   string
	}{
		{
			teamcontrol.ResourceTask,
			queueTaskID,
			teamcontrol.ResourceCommit,
			commitArtifact.ID,
			"committed_as",
		},
		{
			teamcontrol.ResourceTask,
			queueTaskID,
			teamcontrol.ResourcePullRequest,
			pullRequestArtifact.ID,
			"reviewed_in",
		},
		{
			teamcontrol.ResourceCommit,
			commitArtifact.ID,
			teamcontrol.ResourcePullRequest,
			pullRequestArtifact.ID,
			"proposed_in",
		},
		{
			teamcontrol.ResourceCommit,
			commitArtifact.ID,
			teamcontrol.ResourceRepository,
			linked.RepositoryID,
			"belongs_to",
		},
	} {
		if err := h.ensureCorrelation(
			actorID,
			linked.ProjectID,
			link.sourceType,
			link.sourceID,
			link.targetType,
			link.targetID,
			link.relation,
		); err != nil {
			return err
		}
	}
	for _, workItemID := range developmentWorkItemIDs(linked) {
		if err := h.ensureCorrelation(
			actorID,
			linked.ProjectID,
			teamcontrol.ResourceCommit,
			commitArtifact.ID,
			teamcontrol.ResourceWorkItem,
			workItemID,
			"implements",
		); err != nil {
			return err
		}
	}
	for _, issueID := range linked.IssueIDs {
		if err := h.ensureCorrelation(
			actorID,
			linked.ProjectID,
			teamcontrol.ResourcePullRequest,
			pullRequestArtifact.ID,
			teamcontrol.ResourceIssue,
			issueID,
			"addresses",
		); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) registerDevelopmentArtifact(
	actorID string,
	input teamcontrol.RegisterArtifactInput,
) (teamcontrol.Artifact, error) {
	artifact, err := h.teamSvc.RegisterArtifact(actorID, input)
	if err == nil {
		return artifact, nil
	}
	artifacts, listErr := h.teamSvc.ListArtifacts(actorID, input.ProjectID)
	if listErr != nil {
		return teamcontrol.Artifact{}, err
	}
	for _, existing := range artifacts {
		if existing.ID == input.ID &&
			existing.ResourceType == input.ResourceType &&
			existing.Kind == input.Kind &&
			existing.URI == input.URI {
			return existing, nil
		}
	}
	return teamcontrol.Artifact{}, err
}

func (h *Handler) queueDevelopmentRun(sessionID string, params map[string]interface{}, operation string) (interface{}, error) {
	if h.teamSvc != nil {
		return nil, fmt.Errorf(
			"gateway development execution is disabled in team mode; use dev.task.enqueue with the persistent runner queue",
		)
	}
	if !h.devSvc.Config().GatewayAllowExecution {
		return nil, fmt.Errorf("gateway development execution is disabled; use the local goclaw dev CLI")
	}
	id := stringParam(params["id"])
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	task, actor, err := h.authorizedDevelopmentTask(
		sessionID,
		id,
		teamcontrol.ActionWorkItemWrite,
	)
	if err != nil {
		return nil, err
	}
	if err := h.authorizeDevelopmentTaskOwner(actor, task); err != nil {
		return nil, err
	}
	actor, err = h.actorForSession(sessionID, stringParam(params["actor"]))
	if err != nil {
		return nil, err
	}
	force, _ := params["force"].(bool)
	go func() {
		switch operation {
		case "run":
			_, _ = h.devSvc.RunTask(context.Background(), id, actor)
		case "repair":
			_, _ = h.devSvc.RepairTask(context.Background(), id, actor)
		case "resume":
			_, _ = h.devSvc.ResumeTask(context.Background(), id, actor, force)
		}
	}()
	return map[string]interface{}{
		"status":    "queued",
		"id":        id,
		"operation": operation,
	}, nil
}

func (h *Handler) authorizedDevelopmentTask(
	sessionID, id string,
	action teamcontrol.Action,
) (dev.Task, string, error) {
	if strings.TrimSpace(id) == "" {
		return dev.Task{}, "", fmt.Errorf("id is required")
	}
	task, err := h.devSvc.GetTask(id)
	if err != nil {
		return dev.Task{}, "", err
	}
	if h.teamSvc == nil {
		actor, actorErr := h.actorForSession(sessionID, "")
		return task, actor, actorErr
	}
	userID, err := h.authorizeProject(sessionID, task.ProjectID, action)
	if err != nil {
		return dev.Task{}, "", err
	}
	if task.RepositoryID != "" {
		if _, err := h.teamSvc.GetRepository(
			userID,
			task.ProjectID,
			task.RepositoryID,
		); err != nil {
			return dev.Task{}, "", err
		}
	}
	return task, userID, nil
}

func (h *Handler) authorizeDevelopmentTaskOwner(
	actorID string,
	task dev.Task,
) error {
	if h.teamSvc == nil ||
		strings.TrimSpace(task.AssigneeID) == strings.TrimSpace(actorID) {
		return nil
	}
	if err := h.teamSvc.Authorize(
		actorID,
		task.ProjectID,
		teamcontrol.ActionProjectManage,
	); err != nil {
		return fmt.Errorf(
			"only development task assignee %q or a project manager may mutate task %q: %w",
			task.AssigneeID,
			task.ID,
			err,
		)
	}
	return nil
}

func (h *Handler) rpcEnqueueDevelopmentTask(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	if h.teamSvc == nil || h.runnerSvc == nil {
		return nil, fmt.Errorf(
			"team control and workstation services are required for dev.task.enqueue",
		)
	}
	var request struct {
		TaskID              string   `json:"task_id"`
		Priority            int      `json:"priority"`
		Capabilities        []string `json:"capabilities"`
		MaxAttempts         int      `json:"max_attempts"`
		ExecutionProfile    string   `json:"execution_profile"`
		ClientExecutionPack any      `json:"execution_pack"`
	}
	if err := decodeRPCParams(params, &request); err != nil {
		return nil, err
	}
	unlock := h.lockDevelopmentTask(request.TaskID)
	defer unlock()
	task, userID, err := h.authorizedDevelopmentTask(
		sessionID,
		request.TaskID,
		teamcontrol.ActionWorkItemWrite,
	)
	if err != nil {
		return nil, err
	}
	if err := h.authorizeDevelopmentTaskOwner(userID, task); err != nil {
		return nil, err
	}
	if task.Status != dev.TaskFrozen {
		return nil, fmt.Errorf(
			"task %q must be frozen before workstation enqueue (status %q)",
			task.ID,
			task.Status,
		)
	}
	if task.ProjectID == "" || task.RepositoryID == "" ||
		task.AssigneeID == "" || task.PolicyBundleHash == "" ||
		task.Compile.BaseCommit == "" {
		return nil, fmt.Errorf(
			"frozen task is missing project, repository, assignee, policy, or base commit",
		)
	}
	if task.Wave == nil {
		return nil, fmt.Errorf(
			"frozen team task is missing its immutable wave binding",
		)
	}
	if err := h.devSvc.ValidateWaveBinding(
		context.Background(),
		task,
	); err != nil {
		return nil, fmt.Errorf("revalidate frozen wave binding: %w", err)
	}
	project, err := h.teamSvc.GetProject(userID, task.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := h.requireControlConsistency(userID, task.ProjectID); err != nil {
		return nil, err
	}
	if task.TeamID != project.TeamID {
		return nil, fmt.Errorf(
			"frozen task team %q does not match project team %q",
			task.TeamID,
			project.TeamID,
		)
	}
	repository, err := h.teamSvc.GetRepository(
		userID,
		task.ProjectID,
		task.RepositoryID,
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
	if err := h.validateDevelopmentTraceability(userID, task); err != nil {
		return nil, err
	}
	policy, err := h.teamSvc.ResolvePolicy(
		userID,
		task.ProjectID,
		task.RepositoryID,
		"",
	)
	if err != nil {
		return nil, err
	}
	if policy.Hash != task.PolicyBundleHash {
		return nil, fmt.Errorf(
			"frozen policy bundle drift: task=%s current=%s",
			task.PolicyBundleHash,
			policy.Hash,
		)
	}
	executionProfile, err := workstation.NormalizeExecutionProfile(
		request.ExecutionProfile,
	)
	if err != nil {
		return nil, err
	}
	if err := validateTeamExecutionProfile(policy, executionProfile); err != nil {
		return nil, err
	}
	lifecyclePolicy, err := resolveRunnerLifecyclePolicy(policy)
	if err != nil {
		return nil, err
	}
	if lifecyclePolicy.Paused {
		return nil, fmt.Errorf(
			"%w: project runner rollout is paused",
			teamcontrol.ErrForbidden,
		)
	}
	if err := h.validateDevelopmentWorkItemStates(userID, task); err != nil {
		return nil, err
	}
	pack, err := buildDevelopmentExecutionPack(task, repository)
	if err != nil {
		return nil, err
	}
	if err := validateDevelopmentExecutionPackWave(task, pack); err != nil {
		return nil, err
	}
	pack.ExecutionProfile = executionProfile
	if pack.Metadata == nil {
		pack.Metadata = map[string]string{}
	}
	pack.Metadata["target_version"] = lifecyclePolicy.TargetVersion
	pack.Metadata["target_release_id"] = lifecyclePolicy.TargetReleaseID
	pack.Metadata["release_channel"] = lifecyclePolicy.ReleaseChannel
	// A frozen task revision has exactly one workstation queue identity. Client
	// supplied IDs or idempotency keys would allow the same immutable revision
	// to be executed more than once under alternate names.
	queueID := fmt.Sprintf("%s-r%d", task.ID, task.Compile.Revision)
	idempotencyKey := fmt.Sprintf(
		"dev:%s:revision:%d:bundle:%s",
		task.ID,
		task.Compile.Revision,
		task.Compile.ExecutionBundleHash,
	)
	requiredCapabilities := appendRequiredCapability(
		request.Capabilities,
		requiredExecutionProfileCapability(executionProfile),
	)
	if lifecyclePolicy.TargetVersion != "" {
		capability, capabilityErr := workstation.RunnerVersionCapability(
			lifecyclePolicy.TargetVersion,
		)
		if capabilityErr != nil {
			return nil, capabilityErr
		}
		requiredCapabilities = appendRequiredCapability(
			requiredCapabilities,
			capability,
		)
	}
	if lifecyclePolicy.TargetReleaseID != "" {
		capability, capabilityErr := workstation.RunnerReleaseCapability(
			lifecyclePolicy.TargetReleaseID,
		)
		if capabilityErr != nil {
			return nil, capabilityErr
		}
		requiredCapabilities = appendRequiredCapability(
			requiredCapabilities,
			capability,
		)
	}
	queued, err := h.runnerSvc.Enqueue(workstation.EnqueueRequest{
		ID:                   queueID,
		IdempotencyKey:       idempotencyKey,
		ProjectID:            task.ProjectID,
		Priority:             request.Priority,
		RequiredCapabilities: requiredCapabilities,
		MaxAttempts:          request.MaxAttempts,
		ExecutionPack:        pack,
	})
	if err != nil {
		return nil, err
	}
	if err := validateDevelopmentQueueContract(task, queued); err != nil {
		return nil, err
	}
	if err := h.registerQueueTraceability(userID, queued); err != nil {
		return nil, err
	}
	if err := h.transitionExecutionResources(
		userID,
		queued,
		executionTransitionStart,
	); err != nil {
		return nil, err
	}
	return queued, nil
}

type runnerLifecyclePolicy struct {
	TargetVersion   string
	TargetReleaseID string
	ReleaseChannel  string
	Paused          bool
}

func resolveRunnerLifecyclePolicy(
	policy teamcontrol.ResolvedPolicy,
) (runnerLifecyclePolicy, error) {
	var result runnerLifecyclePolicy
	for key, target := range map[string]*string{
		"runner.target_version":    &result.TargetVersion,
		"runner.target_release_id": &result.TargetReleaseID,
		"runner.release_channel":   &result.ReleaseChannel,
	} {
		raw, exists := policy.Rules[key]
		if !exists {
			continue
		}
		if err := json.Unmarshal(raw, target); err != nil ||
			strings.TrimSpace(*target) == "" {
			return runnerLifecyclePolicy{}, fmt.Errorf(
				"resolved policy %s must be a non-empty string",
				key,
			)
		}
		*target = strings.TrimSpace(*target)
	}
	if raw, exists := policy.Rules["runner.rollout_paused"]; exists {
		if err := json.Unmarshal(raw, &result.Paused); err != nil {
			return runnerLifecyclePolicy{}, errors.New(
				"resolved policy runner.rollout_paused must be boolean",
			)
		}
	}
	return result, nil
}

func requiredExecutionProfileCapability(
	profile workstation.ExecutionProfile,
) string {
	if profile == workstation.ExecutionProfileCodexDelegated {
		return workstation.RunnerCodexDelegatedCapability
	}
	return workstation.RunnerLinuxCapability
}

func validateTeamExecutionProfile(
	policy teamcontrol.ResolvedPolicy,
	profile workstation.ExecutionProfile,
) error {
	raw, configured := policy.Rules["runner.execution_profiles"]
	if !configured {
		if profile == workstation.ExecutionProfileStrict {
			return nil
		}
		return fmt.Errorf(
			"%w: project policy does not allow runner execution profile %q",
			teamcontrol.ErrForbidden,
			profile,
		)
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return fmt.Errorf(
			"resolved policy runner.execution_profiles must be a string array",
		)
	}
	if len(values) == 0 {
		return fmt.Errorf(
			"%w: project policy allows no runner execution profiles",
			teamcontrol.ErrForbidden,
		)
	}
	for _, value := range values {
		allowed, err := workstation.NormalizeExecutionProfile(value)
		if err != nil {
			return fmt.Errorf(
				"resolved policy contains an unsupported runner execution profile",
			)
		}
		if allowed == profile {
			return nil
		}
	}
	return fmt.Errorf(
		"%w: project policy does not allow runner execution profile %q",
		teamcontrol.ErrForbidden,
		profile,
	)
}

func (h *Handler) validateDevelopmentTraceability(
	userID string,
	task dev.Task,
) error {
	if err := h.validateProjectAssignee(
		userID,
		task.ProjectID,
		task.AssigneeID,
	); err != nil {
		return err
	}
	workItemIDs := developmentWorkItemIDs(task)
	if len(workItemIDs) == 0 {
		return fmt.Errorf("frozen task requires at least one work item")
	}
	tasks, err := h.devSvc.ListTasks(task.ProjectID)
	if err != nil {
		return err
	}
	for _, candidate := range tasks {
		if candidate.ID == task.ID {
			continue
		}
		for _, workItemID := range developmentWorkItemIDs(candidate) {
			if containsDevelopmentID(workItemIDs, workItemID) {
				return fmt.Errorf(
					"work item %q is already bound to development task %q; use a revision of that task or create a distinct work item",
					workItemID,
					candidate.ID,
				)
			}
		}
	}
	for _, workItemID := range workItemIDs {
		if _, err := h.teamSvc.GetWorkItem(
			userID,
			task.ProjectID,
			workItemID,
		); err != nil {
			return fmt.Errorf("resolve work item %q: %w", workItemID, err)
		}
		assignments, err := h.teamSvc.ListAssignments(
			userID,
			task.ProjectID,
			teamcontrol.AssignmentWorkItem,
			workItemID,
		)
		if err != nil {
			return fmt.Errorf("resolve work item %q assignments: %w", workItemID, err)
		}
		var activeOwners []teamcontrol.Assignment
		for _, assignment := range assignments {
			if assignment.Status == teamcontrol.AssignmentActive &&
				assignment.Role == teamcontrol.AssignmentOwner {
				activeOwners = append(activeOwners, assignment)
			}
		}
		if len(activeOwners) != 1 {
			return fmt.Errorf(
				"work item %q requires exactly one active owner assignment, got %d",
				workItemID,
				len(activeOwners),
			)
		}
		if activeOwners[0].UserID != task.AssigneeID {
			return fmt.Errorf(
				"work item %q owner %q does not match frozen task assignee %q",
				workItemID,
				activeOwners[0].UserID,
				task.AssigneeID,
			)
		}
	}
	for _, issueID := range task.IssueIDs {
		if _, err := h.teamSvc.GetIssue(
			userID,
			task.ProjectID,
			issueID,
		); err != nil {
			return fmt.Errorf("resolve issue %q: %w", issueID, err)
		}
	}
	return nil
}

func (h *Handler) validateProjectAssignee(
	userID string,
	projectID string,
	assigneeID string,
) error {
	assignee, err := h.teamSvc.GetUser(assigneeID)
	if err != nil {
		return fmt.Errorf("resolve task assignee: %w", err)
	}
	if assignee.Status != teamcontrol.UserActive {
		return fmt.Errorf("task assignee %q is not active", assigneeID)
	}
	members, err := h.teamSvc.ListProjectMembers(userID, projectID)
	if err != nil {
		return err
	}
	assigneeIsMember := false
	for _, member := range members {
		if member.UserID == assigneeID &&
			member.Status == teamcontrol.MembershipActive {
			assigneeIsMember = true
			break
		}
	}
	if !assigneeIsMember {
		return fmt.Errorf(
			"task assignee %q is not an active project member",
			assigneeID,
		)
	}
	return nil
}

func (h *Handler) validateDevelopmentWorkItemStates(
	userID string,
	task dev.Task,
) error {
	for _, workItemID := range developmentWorkItemIDs(task) {
		item, err := h.teamSvc.GetWorkItem(userID, task.ProjectID, workItemID)
		if err != nil {
			return err
		}
		switch item.Status {
		case teamcontrol.WorkItemReady,
			teamcontrol.WorkItemInProgress,
			teamcontrol.WorkItemBlocked:
		default:
			return fmt.Errorf(
				"work item %q cannot be enqueued from status %q",
				item.ID,
				item.Status,
			)
		}
	}
	return nil
}

func buildDevelopmentExecutionPack(
	task dev.Task,
	repository teamcontrol.Repository,
) (workstation.ExecutionPack, error) {
	verification := make([]workstation.CommandSpec, 0)
	for _, command := range task.EvidencePlan.Commands {
		verification = append(verification, workstation.CommandSpec{
			Name: command.Name,
			Argv: append([]string(nil), command.Argv...),
		})
	}
	if len(verification) == 0 {
		for _, milestone := range task.Plan.Milestones {
			for _, item := range milestone.WorkItems {
				for _, command := range item.VerificationCommands {
					verification = append(verification, workstation.CommandSpec{
						Name: command.Name,
						Argv: append([]string(nil), command.Argv...),
					})
				}
			}
		}
	}
	if len(verification) == 0 {
		return workstation.ExecutionPack{}, fmt.Errorf(
			"frozen task requires deterministic verification commands",
		)
	}
	payload, err := json.Marshal(struct {
		ID                 string           `json:"id"`
		Title              string           `json:"title"`
		RequestedBy        string           `json:"requested_by,omitempty"`
		Wave               *dev.WaveBinding `json:"wave"`
		Request            dev.RequestFrame `json:"request"`
		Goal               dev.GoalSpec     `json:"goal"`
		Plan               dev.PlanSpec     `json:"plan"`
		DoneGate           dev.DoneGateSpec `json:"done_gate"`
		SpecRefs           []string         `json:"spec_refs,omitempty"`
		DocumentRefs       []string         `json:"document_refs,omitempty"`
		PolicyInstructions []string         `json:"policy_instructions,omitempty"`
	}{
		ID:                 task.ID,
		Title:              task.Title,
		RequestedBy:        task.RequestedBy,
		Wave:               task.Wave,
		Request:            task.Request,
		Goal:               task.Goal,
		Plan:               task.Plan,
		DoneGate:           task.DoneGate,
		SpecRefs:           task.SpecRefs,
		DocumentRefs:       task.DocumentRefs,
		PolicyInstructions: task.PolicyInstructions,
	})
	if err != nil {
		return workstation.ExecutionPack{}, err
	}
	prompt := strings.Join([]string{
		"Execute the frozen GoClaw development task exactly as specified.",
		"Task: " + task.ID + " — " + task.Title,
		"Objective: " + task.Goal.Objective,
		"Request: " + task.Request.RawRequest,
		"Policy:\n" + strings.Join(task.PolicyInstructions, "\n"),
		"Do not change paths outside the allowed scope. Run every verification command.",
	}, "\n\n")
	correlationID := task.CorrelationID
	if correlationID == "" {
		correlationID = task.ID
	}
	metadata := map[string]string{
		"team_id":               task.TeamID,
		"assignee_id":           task.AssigneeID,
		"dev_task_id":           task.ID,
		"module":                task.Module,
		"execution_bundle_hash": task.Compile.ExecutionBundleHash,
		"requested_by":          task.RequestedBy,
	}
	if task.Wave != nil {
		metadata["wave_id"] = task.Wave.WaveID
		metadata["wave_revision"] = fmt.Sprintf("%d", task.Wave.PlanRevision)
		metadata["wave_step"] = task.Wave.StepID
		metadata["wave_plan_path"] = task.Wave.PlanPath
		metadata["wave_registry_sha256"] = task.Wave.RegistrySHA256
		metadata["wave_plan_sha256"] = task.Wave.PlanSHA256
	}
	return workstation.ExecutionPack{
		TaskRevision:     task.Compile.Revision,
		ProjectID:        task.ProjectID,
		CorrelationID:    correlationID,
		IssueIDs:         append([]string(nil), task.IssueIDs...),
		SpecHash:         task.Compile.ExecutionBundleHash,
		WorkItemIDs:      developmentWorkItemIDs(task),
		RepositoryID:     task.RepositoryID,
		RepositoryURL:    repository.RemoteURL,
		BaseRef:          task.Compile.BaseRef,
		BaseCommit:       task.Compile.BaseCommit,
		Branch:           task.Branch,
		Prompt:           prompt,
		AllowedPaths:     append([]string(nil), task.Scope.AllowedPaths...),
		DeniedPaths:      append([]string(nil), task.Scope.DeniedPaths...),
		Verification:     verification,
		PolicyBundleHash: task.PolicyBundleHash,
		Metadata:         metadata,
		Payload:          payload,
	}, nil
}

func appendRequiredCapability(values []string, required string) []string {
	result := append([]string(nil), values...)
	for _, value := range result {
		if strings.EqualFold(strings.TrimSpace(value), required) {
			return result
		}
	}
	return append(result, required)
}

func validateDevelopmentExecutionPackWave(
	task dev.Task,
	pack workstation.ExecutionPack,
) error {
	if task.Wave == nil {
		return fmt.Errorf("development task is missing its Wave binding")
	}
	expected := map[string]string{
		"wave_id":              task.Wave.WaveID,
		"wave_revision":        fmt.Sprintf("%d", task.Wave.PlanRevision),
		"wave_step":            task.Wave.StepID,
		"wave_plan_path":       task.Wave.PlanPath,
		"wave_registry_sha256": task.Wave.RegistrySHA256,
		"wave_plan_sha256":     task.Wave.PlanSHA256,
	}
	for key, value := range expected {
		if pack.Metadata[key] != value {
			return fmt.Errorf(
				"execution pack Wave metadata %q does not match the frozen task",
				key,
			)
		}
	}
	var payload struct {
		Wave *dev.WaveBinding `json:"wave"`
	}
	if err := json.Unmarshal(pack.Payload, &payload); err != nil {
		return fmt.Errorf("decode execution pack Wave payload: %w", err)
	}
	if payload.Wave == nil || *payload.Wave != *task.Wave {
		return fmt.Errorf(
			"execution pack Wave payload does not match the frozen task",
		)
	}
	if pack.TaskRevision != task.Compile.Revision ||
		pack.ProjectID != task.ProjectID ||
		pack.RepositoryID != task.RepositoryID ||
		pack.BaseCommit != task.Compile.BaseCommit ||
		pack.PolicyBundleHash != task.PolicyBundleHash ||
		pack.SpecHash != task.Compile.ExecutionBundleHash {
		return fmt.Errorf(
			"execution pack immutable task identity does not match the frozen task",
		)
	}
	return nil
}

func (h *Handler) validateQueuedDevelopmentWave(task dev.Task) error {
	if h.runnerSvc == nil {
		return fmt.Errorf(
			"workstation queue is required to validate the accepted execution pack",
		)
	}
	queued, err := h.runnerSvc.GetTask(developmentQueueRevisionID(task))
	if err != nil {
		return fmt.Errorf("load queued execution pack before acceptance: %w", err)
	}
	if queued.Status != workstation.TaskCompleted {
		return fmt.Errorf(
			"queued execution pack %q is not completed (status %q)",
			queued.ID,
			queued.Status,
		)
	}
	return validateDevelopmentQueueContract(task, queued)
}

func validateDevelopmentQueueContract(
	task dev.Task,
	queued workstation.Task,
) error {
	if err := validateDevelopmentExecutionPackWave(
		task,
		queued.ExecutionPack,
	); err != nil {
		return err
	}
	if !containsDevelopmentID(
		queued.RequiredCapabilities,
		"goclaw-runtime-linux-v1",
	) {
		return fmt.Errorf(
			"queued execution pack does not require goclaw-runtime-linux-v1",
		)
	}
	digest, err := workstation.HashExecutionPack(queued.ExecutionPack)
	if err != nil {
		return fmt.Errorf("hash queued execution pack: %w", err)
	}
	if digest != queued.ExecutionPackSHA256 {
		return fmt.Errorf(
			"queued execution pack hash mismatch: stored=%s actual=%s",
			queued.ExecutionPackSHA256,
			digest,
		)
	}
	return nil
}

func developmentWorkItemIDs(task dev.Task) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, milestone := range task.Plan.Milestones {
		for _, item := range milestone.WorkItems {
			if item.ID == "" {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			result = append(result, item.ID)
		}
	}
	return result
}
