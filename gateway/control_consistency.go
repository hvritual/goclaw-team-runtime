package gateway

import (
	"fmt"
	"strings"
	"time"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

type consistencySeverity string

const (
	consistencyWarning  consistencySeverity = "warning"
	consistencyCritical consistencySeverity = "critical"
)

type consistencyFinding struct {
	Code         string              `json:"code"`
	Severity     consistencySeverity `json:"severity"`
	ResourceType string              `json:"resource_type"`
	ResourceID   string              `json:"resource_id"`
	Message      string              `json:"message"`
}

type consistencyReport struct {
	SchemaVersion  string               `json:"schema_version"`
	ProjectID      string               `json:"project_id"`
	Consistent     bool                 `json:"consistent"`
	EnqueueAllowed bool                 `json:"enqueue_allowed"`
	AcceptAllowed  bool                 `json:"accept_allowed"`
	CheckedAt      time.Time            `json:"checked_at"`
	Development    int                  `json:"development_tasks"`
	Workstation    int                  `json:"workstation_tasks"`
	Findings       []consistencyFinding `json:"findings"`
}

func (h *Handler) registerControlConsistencyMethods() {
	h.registry.Register(
		"control.consistency.check",
		func(sessionID string, params map[string]interface{}) (interface{}, error) {
			actorID, err := h.authorizeProject(
				sessionID,
				stringParam(params["project_id"]),
				teamcontrol.ActionProjectManage,
			)
			if err != nil {
				return nil, err
			}
			return h.controlConsistency(actorID, stringParam(params["project_id"]))
		},
	)
}

// requireControlConsistency is evaluated immediately before irreversible
// cross-store transitions. It deliberately does not guess how to repair an
// ambiguous state: a stopped Gateway, cold backup, and explicit operator
// reconciliation are required first.
func (h *Handler) requireControlConsistency(actorID, projectID string) error {
	report, err := h.controlConsistency(actorID, projectID)
	if err != nil {
		return err
	}
	if !report.Consistent {
		return fmt.Errorf(
			"control-plane consistency gate blocked enqueue/accept with %d finding(s); run control.consistency.check",
			len(report.Findings),
		)
	}
	return nil
}

func (h *Handler) controlConsistency(
	actorID, projectID string,
) (consistencyReport, error) {
	report := consistencyReport{
		SchemaVersion:  "goclaw.control-consistency/v1",
		ProjectID:      strings.TrimSpace(projectID),
		Consistent:     true,
		EnqueueAllowed: true,
		AcceptAllowed:  true,
		CheckedAt:      time.Now().UTC(),
	}
	add := func(
		code string,
		severity consistencySeverity,
		resourceType, resourceID, message string,
	) {
		report.Findings = append(report.Findings, consistencyFinding{
			Code: code, Severity: severity,
			ResourceType: resourceType, ResourceID: resourceID,
			Message: message,
		})
		if severity == consistencyCritical {
			report.Consistent = false
			report.EnqueueAllowed = false
			report.AcceptAllowed = false
		}
	}
	if h.teamSvc == nil || h.devSvc == nil || h.runnerSvc == nil {
		add(
			"services.incomplete",
			consistencyCritical,
			"project",
			report.ProjectID,
			"team control, development, and workstation services must all be enabled",
		)
		return report, nil
	}
	devTasks, err := h.devSvc.ListTasks(report.ProjectID)
	if err != nil {
		return report, err
	}
	queueTasks, err := h.runnerSvc.ListTasks(workstation.TaskFilter{
		ProjectID: report.ProjectID,
	})
	if err != nil {
		return report, err
	}
	report.Development = len(devTasks)
	report.Workstation = len(queueTasks)
	devByID := make(map[string]dev.Task, len(devTasks))
	queueByRevision := make(map[string]workstation.Task, len(queueTasks))
	issueTasks := make(map[string][]dev.Task)
	for _, task := range devTasks {
		devByID[task.ID] = task
		for _, issueID := range task.IssueIDs {
			issueTasks[issueID] = append(issueTasks[issueID], task)
		}
	}
	for _, task := range queueTasks {
		devID := strings.TrimSpace(task.ExecutionPack.Metadata["dev_task_id"])
		if devID == "" {
			add(
				"queue.unbound",
				consistencyCritical,
				"workstation_task",
				task.ID,
				"team workstation task has no dev_task_id",
			)
			continue
		}
		development, ok := devByID[devID]
		if !ok {
			add(
				"queue.orphan",
				consistencyCritical,
				"workstation_task",
				task.ID,
				"referenced development task does not exist",
			)
			continue
		}
		revision := task.ExecutionPack.TaskRevision
		if revision < 1 {
			add(
				"queue.revision-missing",
				consistencyCritical,
				"workstation_task",
				task.ID,
				"queue execution pack has no positive task revision",
			)
			continue
		}
		revisionKey := fmt.Sprintf("%s#%d", devID, revision)
		if prior, duplicate := queueByRevision[revisionKey]; duplicate && prior.ID != task.ID {
			add(
				"queue.duplicate-revision",
				consistencyCritical,
				"development_task",
				devID,
				"more than one workstation task refers to the same development revision",
			)
		}
		queueByRevision[revisionKey] = task
		expectedID := fmt.Sprintf("%s-r%d", devID, revision)
		if task.ID != expectedID {
			add(
				"queue.identity-drift",
				consistencyCritical,
				"workstation_task",
				task.ID,
				"queue identity does not match its immutable development revision",
			)
		}
		if revision != development.Compile.Revision {
			if task.Status == workstation.TaskQueued ||
				task.Status == workstation.TaskLeased {
				add(
					"queue.stale-revision-active",
					consistencyCritical,
					"workstation_task",
					task.ID,
					"an older development revision remains runnable",
				)
			}
			// Terminal historical revisions remain part of the audit trail but
			// do not participate in the current revision's convergence gate.
			continue
		}
		if task.ExecutionPack.Metadata["assignee_id"] != development.AssigneeID {
			add(
				"queue.assignee-drift",
				consistencyCritical,
				"workstation_task",
				task.ID,
				"queue assignee differs from the frozen development task",
			)
		}
		if development.Wave == nil ||
			task.ExecutionPack.Metadata["wave_id"] != development.Wave.WaveID ||
			task.ExecutionPack.Metadata["wave_plan_sha256"] != development.Wave.PlanSHA256 {
			add(
				"queue.wave-drift",
				consistencyCritical,
				"workstation_task",
				task.ID,
				"queue Wave binding differs from the frozen development task",
			)
		}
		if task.Status == workstation.TaskCompleted &&
			development.Status != dev.TaskAwaitingAcceptance &&
			development.Status != dev.TaskDone {
			add(
				"completion.not-imported",
				consistencyCritical,
				"development_task",
				devID,
				"signed workstation completion has not converged to awaiting_acceptance/done",
			)
		}
	}
	for _, task := range devTasks {
		currentRevisionKey := fmt.Sprintf("%s#%d", task.ID, task.Compile.Revision)
		queueTask, queued := queueByRevision[currentRevisionKey]
		if task.Status == dev.TaskAwaitingAcceptance || task.Status == dev.TaskDone {
			if !queued || queueTask.Status != workstation.TaskCompleted {
				add(
					"development.missing-completion",
					consistencyCritical,
					"development_task",
					task.ID,
					"accepted or acceptance-ready task has no completed workstation evidence",
				)
			}
		}
		for _, workItemID := range developmentWorkItemIDs(task) {
			item, err := h.teamSvc.GetWorkItem(actorID, report.ProjectID, workItemID)
			if err != nil {
				add(
					"work-item.missing",
					consistencyCritical,
					"work_item",
					workItemID,
					"development task references a missing or inaccessible WorkItem",
				)
				continue
			}
			if task.Status == dev.TaskDone && item.Status != teamcontrol.WorkItemDone {
				add(
					"work-item.not-closed",
					consistencyCritical,
					"work_item",
					workItemID,
					"development task is done but WorkItem is not done",
				)
			}
			if task.Status == dev.TaskAwaitingAcceptance &&
				item.Status != teamcontrol.WorkItemVerifying {
				add(
					"work-item.not-verifying",
					consistencyCritical,
					"work_item",
					workItemID,
					"development task awaits acceptance but WorkItem is not verifying",
				)
			}
		}
	}
	for issueID, linked := range issueTasks {
		issue, err := h.teamSvc.GetIssue(actorID, report.ProjectID, issueID)
		if err != nil {
			add(
				"issue.missing",
				consistencyCritical,
				"issue",
				issueID,
				"development task references a missing or inaccessible Issue",
			)
			continue
		}
		allDone := true
		for _, task := range linked {
			if task.Status != dev.TaskDone {
				allDone = false
				break
			}
		}
		if !allDone &&
			(issue.Status == teamcontrol.IssueResolved ||
				issue.Status == teamcontrol.IssueClosed) {
			add(
				"issue.closed-early",
				consistencyCritical,
				"issue",
				issueID,
				"linked development tasks remain unfinished",
			)
		}
		if allDone &&
			issue.Status != teamcontrol.IssueResolved &&
			issue.Status != teamcontrol.IssueClosed {
			add(
				"issue.close-pending",
				consistencyWarning,
				"issue",
				issueID,
				"all linked development tasks are done but Issue has not reached a terminal state",
			)
		}
	}
	return report, nil
}
