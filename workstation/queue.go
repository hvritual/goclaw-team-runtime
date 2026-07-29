package workstation

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	minTaskPriority       = -1000
	maxTaskPriority       = 1000
	claimPriorityAgingGap = time.Minute
)

func (s *Service) Enqueue(request EnqueueRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalized, requestHash, err := s.normalizeEnqueueRequest(request)
	if err != nil {
		return Task{}, err
	}
	tasks, err := s.listTasksUnlocked()
	if err != nil {
		return Task{}, err
	}
	for _, task := range tasks {
		if task.ProjectID == normalized.ProjectID && task.IdempotencyKey == normalized.IdempotencyKey {
			if task.RequestSHA256 != requestHash {
				return Task{}, fmt.Errorf("%w: enqueue key %q", ErrIdempotencyConflict, normalized.IdempotencyKey)
			}
			return cloneJSON(task)
		}
		if normalized.ID != "" && task.ID == normalized.ID {
			if task.RequestSHA256 != requestHash {
				return Task{}, fmt.Errorf("%w: task id %s already exists", ErrConflict, normalized.ID)
			}
			return cloneJSON(task)
		}
	}
	if normalized.ID == "" {
		normalized.ID = "wstask-" + uuid.NewString()
	}
	normalized.ExecutionPack.TaskID = normalized.ID
	packHash, err := HashExecutionPack(normalized.ExecutionPack)
	if err != nil {
		return Task{}, err
	}
	now := s.now()
	maxAttempts := normalized.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = s.cfg.DefaultMaxAttempts
	}
	task := Task{
		SchemaVersion:        SchemaVersion,
		ID:                   normalized.ID,
		ProjectID:            normalized.ProjectID,
		IdempotencyKey:       normalized.IdempotencyKey,
		RequestSHA256:        requestHash,
		Status:               TaskQueued,
		Priority:             normalized.Priority,
		RequiredCapabilities: normalized.RequiredCapabilities,
		ExecutionPack:        normalized.ExecutionPack,
		ExecutionPackSHA256:  packHash,
		MaxAttempts:          maxAttempts,
		CreatedAt:            now,
		UpdatedAt:            now,
		History: []TaskEvent{{
			Type:      "task.enqueued",
			Actor:     "control-plane",
			CreatedAt: now,
		}},
	}
	if err := s.saveTaskUnlocked(task); err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) Claim(request ClaimRequest) (ClaimResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(request.RunnerID); err != nil {
		return ClaimResult{}, err
	}
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.IdempotencyKey == "" {
		return ClaimResult{}, errors.New("claim idempotency_key is required")
	}
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return ClaimResult{}, err
	}
	runner, err := s.loadRunnerUnlocked(request.RunnerID)
	if err != nil {
		return ClaimResult{}, err
	}
	if runner.Status == RunnerDisabled {
		return ClaimResult{}, fmt.Errorf("%w: runner %s is disabled", ErrUnauthorized, runner.ID)
	}
	requestHash, err := hashJSON(request)
	if err != nil {
		return ClaimResult{}, err
	}
	tasks, err := s.listTasksUnlocked()
	if err != nil {
		return ClaimResult{}, err
	}
	operation := "claim:" + runner.ID
	if task, receipt, found, err := findReceiptAcrossTasks(tasks, operation, request.IdempotencyKey, requestHash); err != nil {
		return ClaimResult{}, err
	} else if found {
		if receipt.Lease == nil ||
			task.Status != TaskLeased ||
			task.Lease == nil ||
			task.Lease.ID != receipt.Lease.ID ||
			task.Lease.RunnerID != runner.ID {
			return ClaimResult{}, fmt.Errorf(
				"%w: claim receipt no longer refers to an active lease; use a new idempotency key",
				ErrConflict,
			)
		}
		taskCopy, cloneErr := cloneJSON(task)
		if cloneErr != nil {
			return ClaimResult{}, cloneErr
		}
		leaseCopy := *task.Lease
		return ClaimResult{Task: taskCopy, Lease: leaseCopy}, nil
	}
	for _, task := range tasks {
		if task.Status == TaskLeased && task.Lease != nil &&
			task.Lease.RunnerID == runner.ID {
			return ClaimResult{}, fmt.Errorf(
				"%w: runner %s already holds active lease %s for task %s",
				ErrConflict,
				runner.ID,
				task.Lease.ID,
				task.ID,
			)
		}
	}
	var selected *Task
	claimAt := s.now()
	for index := range tasks {
		task := &tasks[index]
		if task.Status != TaskQueued {
			continue
		}
		if request.ProjectID != "" && task.ProjectID != request.ProjectID {
			continue
		}
		if !runnerAuthorized(runner, task.ProjectID, task.RequiredCapabilities) {
			continue
		}
		if !runnerSupportsExecutionProfile(runner, task.ExecutionPack.ExecutionProfile) {
			continue
		}
		if assigneeID := strings.TrimSpace(task.ExecutionPack.Metadata["assignee_id"]); assigneeID != "" &&
			assigneeID != runner.OwnerUserID {
			continue
		}
		if selected == nil || claimTaskBefore(*task, *selected, claimAt) {
			selected = task
		}
	}
	if selected == nil {
		return ClaimResult{}, ErrNoTaskAvailable
	}
	now := s.now()
	lease := Lease{
		ID:             "lease-" + uuid.NewString(),
		RunnerID:       runner.ID,
		Attempt:        selected.Attempt + 1,
		ClaimedAt:      now,
		HeartbeatAt:    now,
		ExpiresAt:      now.Add(s.leaseDuration()),
		IdempotencyKey: request.IdempotencyKey,
	}
	selected.Attempt++
	selected.Status = TaskLeased
	selected.Lease = &lease
	selected.UpdatedAt = now
	selected.History = append(selected.History, TaskEvent{
		Type:      "task.claimed",
		Actor:     runner.ID,
		Attempt:   selected.Attempt,
		Message:   lease.ID,
		CreatedAt: now,
	})
	appendReceipt(selected, IdempotencyReceipt{
		Operation:     operation,
		Key:           request.IdempotencyKey,
		RequestSHA256: requestHash,
		ResultStatus:  TaskLeased,
		Lease:         cloneLease(&lease),
		RecordedAt:    now,
	}, s.cfg.MaxIdempotencyReceipts)
	runner.Status = RunnerOnline
	runner.LastHeartbeatAt = now
	runner.UpdatedAt = now
	if err := s.saveRunnerUnlocked(runner); err != nil {
		return ClaimResult{}, err
	}
	if err := s.saveTaskUnlocked(*selected); err != nil {
		return ClaimResult{}, err
	}
	taskCopy, err := cloneJSON(*selected)
	if err != nil {
		return ClaimResult{}, err
	}
	return ClaimResult{Task: taskCopy, Lease: lease}, nil
}

func (s *Service) Heartbeat(request HeartbeatRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateLeaseRequest(request.RunnerID, request.TaskID, request.LeaseID, request.IdempotencyKey); err != nil {
		return Task{}, err
	}
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Task{}, err
	}
	task, err := s.loadTaskUnlocked(request.TaskID)
	if err != nil {
		return Task{}, err
	}
	requestHash, err := hashJSON(request)
	if err != nil {
		return Task{}, err
	}
	operation := "heartbeat:" + request.RunnerID
	if receipt, found, err := findReceipt(task, operation, request.IdempotencyKey, requestHash); err != nil {
		return Task{}, err
	} else if found {
		_ = receipt
		return cloneJSON(task)
	}
	runner, err := s.loadRunnerUnlocked(request.RunnerID)
	if err != nil {
		return Task{}, err
	}
	if runner.Status == RunnerDisabled {
		return Task{}, fmt.Errorf("%w: runner %s is disabled", ErrUnauthorized, runner.ID)
	}
	if err := validateActiveLease(task, request.RunnerID, request.LeaseID); err != nil {
		return Task{}, err
	}
	now := s.now()
	task.Lease.HeartbeatAt = now
	task.Lease.ExpiresAt = now.Add(s.leaseDuration())
	task.UpdatedAt = now
	task.History = append(task.History, TaskEvent{
		Type:      "lease.heartbeat",
		Actor:     runner.ID,
		Attempt:   task.Attempt,
		CreatedAt: now,
	})
	appendReceipt(&task, IdempotencyReceipt{
		Operation:     operation,
		Key:           request.IdempotencyKey,
		RequestSHA256: requestHash,
		ResultStatus:  TaskLeased,
		Lease:         cloneLease(task.Lease),
		RecordedAt:    now,
	}, s.cfg.MaxIdempotencyReceipts)
	runner.Status = RunnerOnline
	runner.LastHeartbeatAt = now
	runner.UpdatedAt = now
	if err := s.saveRunnerUnlocked(runner); err != nil {
		return Task{}, err
	}
	if err := s.saveTaskUnlocked(task); err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) Complete(request CompleteRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateLeaseRequest(request.RunnerID, request.TaskID, request.LeaseID, request.IdempotencyKey); err != nil {
		return Task{}, err
	}
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Task{}, err
	}
	task, err := s.loadTaskUnlocked(request.TaskID)
	if err != nil {
		return Task{}, err
	}
	requestHash, err := hashJSON(request)
	if err != nil {
		return Task{}, err
	}
	operation := "complete:" + request.RunnerID
	if _, found, err := findReceipt(task, operation, request.IdempotencyKey, requestHash); err != nil {
		return Task{}, err
	} else if found {
		return cloneJSON(task)
	}
	runner, err := s.loadRunnerUnlocked(request.RunnerID)
	if err != nil {
		return Task{}, err
	}
	if err := validateActiveLease(task, request.RunnerID, request.LeaseID); err != nil {
		return Task{}, err
	}
	if request.Evidence.Outcome != "completed" {
		return Task{}, fmt.Errorf("%w: completion evidence outcome must be completed", ErrInvalidEvidence)
	}
	if err := validateCompletionEvidence(task, request.Evidence); err != nil {
		return Task{}, err
	}
	evidence, err := s.verifyAndStoreEvidenceUnlocked(task, runner, request.LeaseID, request.Evidence)
	if err != nil {
		return Task{}, err
	}
	now := s.now()
	task.Status = TaskCompleted
	task.Result = &TaskResult{
		RunnerID:    runner.ID,
		LeaseID:     request.LeaseID,
		Attempt:     task.Attempt,
		Summary:     strings.TrimSpace(request.Summary),
		Evidence:    evidence,
		CompletedAt: now,
	}
	task.Lease = nil
	task.UpdatedAt = now
	task.CompletedAt = &now
	task.History = append(task.History, TaskEvent{
		Type:      "task.completed",
		Actor:     runner.ID,
		Attempt:   task.Attempt,
		Message:   evidence.BundleSHA256,
		CreatedAt: now,
	})
	appendReceipt(&task, IdempotencyReceipt{
		Operation:     operation,
		Key:           request.IdempotencyKey,
		RequestSHA256: requestHash,
		ResultStatus:  TaskCompleted,
		RecordedAt:    now,
	}, s.cfg.MaxIdempotencyReceipts)
	runner.Status = RunnerOnline
	runner.LastHeartbeatAt = now
	runner.UpdatedAt = now
	if err := s.saveRunnerUnlocked(runner); err != nil {
		return Task{}, err
	}
	if err := s.saveTaskUnlocked(task); err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) Fail(request FailRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateLeaseRequest(request.RunnerID, request.TaskID, request.LeaseID, request.IdempotencyKey); err != nil {
		return Task{}, err
	}
	if strings.TrimSpace(request.Error) == "" {
		return Task{}, errors.New("failure error is required")
	}
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Task{}, err
	}
	task, err := s.loadTaskUnlocked(request.TaskID)
	if err != nil {
		return Task{}, err
	}
	requestHash, err := hashJSON(request)
	if err != nil {
		return Task{}, err
	}
	operation := "fail:" + request.RunnerID
	if _, found, err := findReceipt(task, operation, request.IdempotencyKey, requestHash); err != nil {
		return Task{}, err
	} else if found {
		return cloneJSON(task)
	}
	runner, err := s.loadRunnerUnlocked(request.RunnerID)
	if err != nil {
		return Task{}, err
	}
	if err := validateActiveLease(task, request.RunnerID, request.LeaseID); err != nil {
		return Task{}, err
	}
	var evidence *EvidenceReference
	if request.Evidence != nil {
		if request.Evidence.Outcome != "failed" {
			return Task{}, fmt.Errorf("%w: failure evidence outcome must be failed", ErrInvalidEvidence)
		}
		reference, verifyErr := s.verifyAndStoreEvidenceUnlocked(task, runner, request.LeaseID, *request.Evidence)
		if verifyErr != nil {
			return Task{}, verifyErr
		}
		evidence = &reference
	}
	now := s.now()
	task.Status = TaskFailed
	task.LastFailure = &TaskFailure{
		RunnerID: runner.ID,
		LeaseID:  request.LeaseID,
		Attempt:  task.Attempt,
		Error:    strings.TrimSpace(request.Error),
		Evidence: evidence,
		FailedAt: now,
	}
	task.Lease = nil
	task.UpdatedAt = now
	task.History = append(task.History, TaskEvent{
		Type:      "task.failed",
		Actor:     runner.ID,
		Attempt:   task.Attempt,
		Message:   task.LastFailure.Error,
		CreatedAt: now,
	})
	appendReceipt(&task, IdempotencyReceipt{
		Operation:     operation,
		Key:           request.IdempotencyKey,
		RequestSHA256: requestHash,
		ResultStatus:  TaskFailed,
		RecordedAt:    now,
	}, s.cfg.MaxIdempotencyReceipts)
	runner.Status = RunnerOnline
	runner.LastHeartbeatAt = now
	runner.UpdatedAt = now
	if err := s.saveRunnerUnlocked(runner); err != nil {
		return Task{}, err
	}
	if err := s.saveTaskUnlocked(task); err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) Requeue(request RequeueRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(request.TaskID); err != nil {
		return Task{}, err
	}
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.IdempotencyKey == "" {
		return Task{}, errors.New("requeue idempotency_key is required")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return Task{}, errors.New("requeue reason is required")
	}
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Task{}, err
	}
	task, err := s.loadTaskUnlocked(request.TaskID)
	if err != nil {
		return Task{}, err
	}
	requestHash, err := hashJSON(request)
	if err != nil {
		return Task{}, err
	}
	operation := "requeue"
	if _, found, err := findReceipt(task, operation, request.IdempotencyKey, requestHash); err != nil {
		return Task{}, err
	} else if found {
		return cloneJSON(task)
	}
	if task.Status != TaskFailed {
		return Task{}, fmt.Errorf("%w: task %s is not failed", ErrConflict, task.ID)
	}
	if task.Attempt >= task.MaxAttempts {
		if !request.Force {
			return Task{}, fmt.Errorf("%w: task exhausted %d attempts", ErrConflict, task.MaxAttempts)
		}
		task.MaxAttempts = task.Attempt + 1
	}
	now := s.now()
	task.Status = TaskQueued
	task.Lease = nil
	task.UpdatedAt = now
	task.CompletedAt = nil
	task.History = append(task.History, TaskEvent{
		Type:      "task.requeued",
		Actor:     valueOr(request.Actor, "operator"),
		Attempt:   task.Attempt,
		Message:   strings.TrimSpace(request.Reason),
		CreatedAt: now,
	})
	appendReceipt(&task, IdempotencyReceipt{
		Operation:     operation,
		Key:           request.IdempotencyKey,
		RequestSHA256: requestHash,
		ResultStatus:  TaskQueued,
		RecordedAt:    now,
	}, s.cfg.MaxIdempotencyReceipts)
	if err := s.saveTaskUnlocked(task); err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) Cancel(request CancelRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(request.TaskID); err != nil {
		return Task{}, err
	}
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.IdempotencyKey == "" {
		return Task{}, errors.New("cancel idempotency_key is required")
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if request.Reason == "" {
		return Task{}, errors.New("cancel reason is required")
	}
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Task{}, err
	}
	task, err := s.loadTaskUnlocked(request.TaskID)
	if err != nil {
		return Task{}, err
	}
	requestHash, err := hashJSON(request)
	if err != nil {
		return Task{}, err
	}
	const operation = "cancel"
	if _, found, err := findReceipt(
		task,
		operation,
		request.IdempotencyKey,
		requestHash,
	); err != nil {
		return Task{}, err
	} else if found {
		return cloneJSON(task)
	}
	switch task.Status {
	case TaskQueued, TaskFailed:
	case TaskLeased:
		return Task{}, fmt.Errorf(
			"%w: task %s has an active lease and cannot be cancelled",
			ErrConflict,
			task.ID,
		)
	default:
		return Task{}, fmt.Errorf(
			"%w: task %s in status %s cannot be cancelled",
			ErrConflict,
			task.ID,
			task.Status,
		)
	}
	now := s.now()
	task.Status = TaskCancelled
	task.Lease = nil
	task.UpdatedAt = now
	task.CompletedAt = &now
	task.History = append(task.History, TaskEvent{
		Type:      "task.cancelled",
		Actor:     valueOr(request.Actor, "operator"),
		Attempt:   task.Attempt,
		Message:   request.Reason,
		CreatedAt: now,
	})
	appendReceipt(&task, IdempotencyReceipt{
		Operation:     operation,
		Key:           request.IdempotencyKey,
		RequestSHA256: requestHash,
		ResultStatus:  TaskCancelled,
		RecordedAt:    now,
	}, s.cfg.MaxIdempotencyReceipts)
	if err := s.saveTaskUnlocked(task); err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) GetTask(id string) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Task{}, err
	}
	task, err := s.loadTaskUnlocked(id)
	if err != nil {
		return Task{}, err
	}
	return cloneJSON(task)
}

func (s *Service) ListTasks(filter TaskFilter) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return nil, err
	}
	tasks, err := s.listTasksUnlocked()
	if err != nil {
		return nil, err
	}
	filtered := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if filter.ProjectID != "" && task.ProjectID != filter.ProjectID {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		if filter.RunnerID != "" && !taskReferencesRunner(task, filter.RunnerID) {
			continue
		}
		filtered = append(filtered, task)
	}
	return cloneJSON(filtered)
}

func (s *Service) Status(projectID string) (QueueStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if _, err := s.recoverExpiredUnlocked(now); err != nil {
		return QueueStatus{}, err
	}
	tasks, err := s.listTasksUnlocked()
	if err != nil {
		return QueueStatus{}, err
	}
	runners, err := s.listRunnersUnlocked()
	if err != nil {
		return QueueStatus{}, err
	}
	status := QueueStatus{
		SchemaVersion: SchemaVersion,
		ProjectID:     projectID,
		TaskCounts: map[TaskStatus]int{
			TaskQueued: 0, TaskLeased: 0, TaskCompleted: 0, TaskFailed: 0,
			TaskCancelled: 0,
		},
		RunnerCounts: map[RunnerStatus]int{
			RunnerOnline: 0, RunnerOffline: 0, RunnerDisabled: 0,
		},
		UpdatedAt: now,
	}
	for _, task := range tasks {
		if projectID != "" && task.ProjectID != projectID {
			continue
		}
		status.TaskCounts[task.Status]++
		if task.Status == TaskQueued &&
			(status.OldestQueuedAt == nil || task.CreatedAt.Before(*status.OldestQueuedAt)) {
			created := task.CreatedAt
			status.OldestQueuedAt = &created
		}
		if task.Status == TaskLeased && task.Lease != nil {
			status.Leased = append(status.Leased, LeaseStatus{
				TaskID: task.ID, ProjectID: task.ProjectID, RunnerID: task.Lease.RunnerID,
				Attempt: task.Attempt, ExpiresAt: task.Lease.ExpiresAt,
			})
		}
	}
	for _, runner := range runners {
		if projectID != "" && !containsString(runner.Projects, projectID, false) &&
			!containsString(runner.Projects, "*", false) {
			continue
		}
		status.RunnerCounts[runner.Status]++
	}
	sort.Slice(status.Leased, func(i, j int) bool {
		if !status.Leased[i].ExpiresAt.Equal(status.Leased[j].ExpiresAt) {
			return status.Leased[i].ExpiresAt.Before(status.Leased[j].ExpiresAt)
		}
		return status.Leased[i].TaskID < status.Leased[j].TaskID
	})
	return status, nil
}

func (s *Service) RecoverExpiredLeases() (RecoveryReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recoverExpiredUnlocked(s.now())
}

func (s *Service) recoverExpiredUnlocked(now time.Time) (RecoveryReport, error) {
	report := RecoveryReport{RecoveredAt: now}
	runners, err := s.listRunnersUnlocked()
	if err != nil {
		return report, err
	}
	for _, runner := range runners {
		if runner.Status == RunnerDisabled || now.Sub(runner.LastHeartbeatAt) <= s.runnerOfflineAfter() {
			continue
		}
		if runner.Status != RunnerOffline {
			runner.Status = RunnerOffline
			runner.UpdatedAt = now
			if err := s.saveRunnerUnlocked(runner); err != nil {
				return report, err
			}
			report.OfflineRunnerIDs = append(report.OfflineRunnerIDs, runner.ID)
		}
	}
	tasks, err := s.listTasksUnlocked()
	if err != nil {
		return report, err
	}
	for _, task := range tasks {
		if task.Status != TaskLeased || task.Lease == nil || task.Lease.ExpiresAt.After(now) {
			continue
		}
		lease := *task.Lease
		task.Lease = nil
		task.LastFailure = &TaskFailure{
			RunnerID: lease.RunnerID,
			LeaseID:  lease.ID,
			Attempt:  task.Attempt,
			Error:    "lease expired before task completion",
			FailedAt: now,
		}
		eventType := "lease.expired_requeued"
		if task.Attempt >= task.MaxAttempts {
			task.Status = TaskFailed
			eventType = "lease.expired_failed"
			report.FailedTaskIDs = append(report.FailedTaskIDs, task.ID)
		} else {
			task.Status = TaskQueued
			report.RequeuedTaskIDs = append(report.RequeuedTaskIDs, task.ID)
		}
		task.UpdatedAt = now
		task.History = append(task.History, TaskEvent{
			Type: eventType, Actor: "lease-reaper", Attempt: task.Attempt,
			Message: lease.ID, CreatedAt: now,
		})
		if err := s.saveTaskUnlocked(task); err != nil {
			return report, err
		}
		report.ExpiredTaskIDs = append(report.ExpiredTaskIDs, task.ID)
	}
	sort.Strings(report.OfflineRunnerIDs)
	sort.Strings(report.ExpiredTaskIDs)
	sort.Strings(report.RequeuedTaskIDs)
	sort.Strings(report.FailedTaskIDs)
	return report, nil
}

func (s *Service) GetEvidenceBundle(taskID string) (EvidenceBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, err := s.loadTaskUnlocked(taskID)
	if err != nil {
		return EvidenceBundle{}, err
	}
	var reference *EvidenceReference
	var runnerID string
	if task.Result != nil {
		reference = &task.Result.Evidence
		runnerID = task.Result.RunnerID
	} else if task.LastFailure != nil && task.LastFailure.Evidence != nil {
		reference = task.LastFailure.Evidence
		runnerID = task.LastFailure.RunnerID
	}
	if reference == nil {
		return EvidenceBundle{}, fmt.Errorf("%w: task %s has no evidence bundle", ErrNotFound, taskID)
	}
	path, err := safeJoin(s.evidenceDir(), reference.Path)
	if err != nil {
		return EvidenceBundle{}, err
	}
	var bundle EvidenceBundle
	if err := readJSON(path, &bundle); err != nil {
		return EvidenceBundle{}, err
	}
	key, err := s.loadCredentialByKeyIDUnlocked(runnerID, bundle.KeyID)
	if err != nil {
		return EvidenceBundle{}, err
	}
	if bundle.BundleSHA256 != reference.BundleSHA256 || bundle.Signature != reference.Signature {
		return EvidenceBundle{}, fmt.Errorf("%w: evidence reference does not match bundle", ErrInvalidEvidence)
	}
	if err := VerifyEvidenceBundle(bundle, key); err != nil {
		return EvidenceBundle{}, err
	}
	return cloneJSON(bundle)
}

func (s *Service) VerifyEvidence(runnerID string, bundle EvidenceBundle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, err := s.loadRunnerUnlocked(runnerID)
	if err != nil {
		return err
	}
	if bundle.RunnerID != runner.ID {
		return fmt.Errorf("%w: evidence identity does not match registered runner", ErrInvalidEvidence)
	}
	key, err := s.loadCredentialByKeyIDUnlocked(runnerID, bundle.KeyID)
	if err != nil {
		return err
	}
	return VerifyEvidenceBundle(bundle, key)
}

func (s *Service) verifyAndStoreEvidenceUnlocked(
	task Task,
	runner Runner,
	leaseID string,
	bundle EvidenceBundle,
) (EvidenceReference, error) {
	if bundle.KeyID != runner.KeyID {
		return EvidenceReference{}, fmt.Errorf("%w: evidence key id does not match registered runner", ErrInvalidSignature)
	}
	if err := validateEvidenceIdentity(task, runner.ID, leaseID, bundle); err != nil {
		return EvidenceReference{}, err
	}
	key, err := s.loadCredentialUnlocked(runner.ID)
	if err != nil {
		return EvidenceReference{}, err
	}
	if err := VerifyEvidenceBundle(bundle, key); err != nil {
		return EvidenceReference{}, err
	}
	path, err := s.evidencePath(task.ID, bundle.BundleSHA256)
	if err != nil {
		return EvidenceReference{}, err
	}
	if err := writeJSONAtomic(path, bundle, 0o600); err != nil {
		return EvidenceReference{}, err
	}
	relative, err := filepath.Rel(s.evidenceDir(), path)
	if err != nil {
		return EvidenceReference{}, err
	}
	return EvidenceReference{
		BundleSHA256:       bundle.BundleSHA256,
		Signature:          bundle.Signature,
		KeyID:              bundle.KeyID,
		SignatureAlgorithm: bundle.SignatureAlgorithm,
		Path:               filepath.ToSlash(relative),
		VerifiedAt:         s.now(),
	}, nil
}

func (s *Service) normalizeEnqueueRequest(request EnqueueRequest) (EnqueueRequest, string, error) {
	request.ID = strings.TrimSpace(request.ID)
	if request.ID != "" {
		if err := validateID(request.ID); err != nil {
			return EnqueueRequest{}, "", err
		}
	}
	request.ProjectID = strings.TrimSpace(request.ProjectID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.ProjectID == "" {
		return EnqueueRequest{}, "", errors.New("project_id is required")
	}
	if request.IdempotencyKey == "" || len(request.IdempotencyKey) > 256 {
		return EnqueueRequest{}, "", errors.New("idempotency_key is required and must not exceed 256 bytes")
	}
	request.RequiredCapabilities = normalizeCapabilities(request.RequiredCapabilities)
	if request.MaxAttempts < 0 {
		return EnqueueRequest{}, "", errors.New("max_attempts cannot be negative")
	}
	if request.Priority < minTaskPriority || request.Priority > maxTaskPriority {
		return EnqueueRequest{}, "", fmt.Errorf(
			"priority must be between %d and %d",
			minTaskPriority,
			maxTaskPriority,
		)
	}
	pack := request.ExecutionPack
	profile, err := NormalizeExecutionProfile(string(pack.ExecutionProfile))
	if err != nil {
		return EnqueueRequest{}, "", err
	}
	pack.ExecutionProfile = profile
	if pack.ProjectID != "" && pack.ProjectID != request.ProjectID {
		return EnqueueRequest{}, "", errors.New("execution_pack.project_id does not match request")
	}
	if pack.TaskID != "" && request.ID != "" && pack.TaskID != request.ID {
		return EnqueueRequest{}, "", errors.New("execution_pack.task_id does not match request")
	}
	pack.SchemaVersion = SchemaVersion
	pack.ProjectID = request.ProjectID
	pack.TaskID = request.ID
	if pack.TaskRevision <= 0 {
		pack.TaskRevision = 1
	}
	pack.RepositoryID = strings.TrimSpace(pack.RepositoryID)
	pack.BaseCommit = strings.TrimSpace(pack.BaseCommit)
	pack.Prompt = strings.TrimSpace(pack.Prompt)
	if pack.RepositoryID == "" || pack.BaseCommit == "" || pack.Prompt == "" {
		return EnqueueRequest{}, "", errors.New("execution pack requires repository_id, base_commit, and prompt")
	}
	if len(pack.Verification) == 0 {
		return EnqueueRequest{}, "", errors.New("execution pack requires at least one verification command")
	}
	for _, command := range pack.Verification {
		if len(command.Argv) == 0 || strings.TrimSpace(command.Argv[0]) == "" {
			return EnqueueRequest{}, "", errors.New("verification commands require non-empty argv")
		}
		switch strings.TrimSpace(valueOr(command.Name, strings.Join(command.Argv, " "))) {
		case "runner-setup", "codex-exec", "scope-policy", "no-automatic-commit":
			return EnqueueRequest{}, "", errors.New(
				"verification command name conflicts with a reserved runner check",
			)
		}
	}
	if len(pack.Payload) > 0 && !json.Valid(pack.Payload) {
		return EnqueueRequest{}, "", errors.New("execution pack payload must be valid JSON")
	}
	pack.WorkItemIDs = normalizeStrings(pack.WorkItemIDs)
	pack.IssueIDs = normalizeStrings(pack.IssueIDs)
	pack.AllowedPaths = normalizeStrings(pack.AllowedPaths)
	pack.DeniedPaths = normalizeStrings(pack.DeniedPaths)
	pack.Metadata = cloneMetadata(pack.Metadata)
	request.ExecutionPack = pack
	fingerprint := request
	fingerprint.ExecutionPack.TaskID = ""
	requestHash, err := hashJSON(fingerprint)
	return request, requestHash, err
}

func claimTaskBefore(left, right Task, now time.Time) bool {
	leftScore := int64(left.Priority) + claimWaitingAge(left, now)
	rightScore := int64(right.Priority) + claimWaitingAge(right, now)
	if leftScore != rightScore {
		return leftScore > rightScore
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		return left.CreatedAt.Before(right.CreatedAt)
	}
	return left.ID < right.ID
}

func claimWaitingAge(task Task, now time.Time) int64 {
	if !now.After(task.CreatedAt) {
		return 0
	}
	return int64(now.Sub(task.CreatedAt) / claimPriorityAgingGap)
}

func validateLeaseRequest(runnerID, taskID, leaseID, idempotencyKey string) error {
	if err := validateID(runnerID); err != nil {
		return err
	}
	if err := validateID(taskID); err != nil {
		return err
	}
	if err := validateID(leaseID); err != nil {
		return err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}
	return nil
}

func validateActiveLease(task Task, runnerID, leaseID string) error {
	if task.Status != TaskLeased || task.Lease == nil {
		if task.LastFailure != nil && task.LastFailure.LeaseID == leaseID &&
			strings.Contains(task.LastFailure.Error, "lease expired") {
			return ErrLeaseExpired
		}
		return fmt.Errorf("%w: task %s is not leased", ErrConflict, task.ID)
	}
	if task.Lease.RunnerID != runnerID || task.Lease.ID != leaseID {
		return fmt.Errorf("%w: lease does not belong to runner", ErrUnauthorized)
	}
	return nil
}

func findReceipt(
	task Task,
	operation, key, requestHash string,
) (IdempotencyReceipt, bool, error) {
	for _, receipt := range task.Receipts {
		if receipt.Operation != operation || receipt.Key != key {
			continue
		}
		if receipt.RequestSHA256 != requestHash {
			return IdempotencyReceipt{}, false,
				fmt.Errorf("%w: operation %s key %q", ErrIdempotencyConflict, operation, key)
		}
		return receipt, true, nil
	}
	return IdempotencyReceipt{}, false, nil
}

func findReceiptAcrossTasks(
	tasks []Task,
	operation, key, requestHash string,
) (Task, IdempotencyReceipt, bool, error) {
	for _, task := range tasks {
		receipt, found, err := findReceipt(task, operation, key, requestHash)
		if err != nil {
			return Task{}, IdempotencyReceipt{}, false, err
		}
		if found {
			return task, receipt, true, nil
		}
	}
	return Task{}, IdempotencyReceipt{}, false, nil
}

func appendReceipt(task *Task, receipt IdempotencyReceipt, limit int) {
	task.Receipts = append(task.Receipts, receipt)
	if limit > 0 && len(task.Receipts) > limit {
		task.Receipts = append([]IdempotencyReceipt(nil), task.Receipts[len(task.Receipts)-limit:]...)
	}
}

func cloneLease(value *Lease) *Lease {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func taskReferencesRunner(task Task, runnerID string) bool {
	return task.Lease != nil && task.Lease.RunnerID == runnerID ||
		task.Result != nil && task.Result.RunnerID == runnerID ||
		task.LastFailure != nil && task.LastFailure.RunnerID == runnerID
}

func (s *Service) leaseDuration() time.Duration {
	return time.Duration(s.cfg.LeaseDurationSeconds) * time.Second
}

func (s *Service) runnerOfflineAfter() time.Duration {
	return time.Duration(s.cfg.RunnerOfflineSeconds) * time.Second
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
