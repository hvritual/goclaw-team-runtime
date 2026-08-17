package application

import (
	"context"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

const (
	TaskActionCreate  = "workspace.task.create"
	TaskActionUpdate  = "workspace.task.update"
	TaskActionArchive = "workspace.task.archive"
	TaskActionRestore = "workspace.task.restore"
	TaskActionReorder = "workspace.task.reorder"
)

type todoGovernanceCommandKey struct{}

type TodoGovernanceCommand struct {
	Action             string
	IdempotencyKey     string
	RequestFingerprint string
}

func WithTodoGovernanceAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, todoGovernanceCommandKey{}, TodoGovernanceCommand{Action: strings.TrimSpace(action)})
}

func WithTodoCreateGovernance(ctx context.Context, idempotencyKey, requestFingerprint string) context.Context {
	return context.WithValue(ctx, todoGovernanceCommandKey{}, TodoGovernanceCommand{
		Action: TaskActionCreate, IdempotencyKey: strings.TrimSpace(idempotencyKey), RequestFingerprint: strings.TrimSpace(requestFingerprint),
	})
}

func WithTodoReorderGovernance(ctx context.Context, idempotencyKey, requestFingerprint string) context.Context {
	return context.WithValue(ctx, todoGovernanceCommandKey{}, TodoGovernanceCommand{
		Action: TaskActionReorder, IdempotencyKey: strings.TrimSpace(idempotencyKey), RequestFingerprint: strings.TrimSpace(requestFingerprint),
	})
}

func TodoGovernanceActionFromContext(ctx context.Context) (string, bool) {
	command, ok := TodoGovernanceCommandFromContext(ctx)
	return command.Action, ok
}

func TodoGovernanceCommandFromContext(ctx context.Context) (TodoGovernanceCommand, bool) {
	command, ok := ctx.Value(todoGovernanceCommandKey{}).(TodoGovernanceCommand)
	return command, ok && command.Action != ""
}

type TaskGovernancePolicyProvider struct{}

func (TaskGovernancePolicyProvider) ResolveGovernancePolicy(ctx context.Context, workspaceID, requestID, action, resourceKind string) (contract.MutationIdentity, GovernanceActionPolicy, error) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || !validTaskGovernanceBinding(action, resourceKind) {
		return contract.MutationIdentity{}, GovernanceActionPolicy{}, contract.ErrGovernanceUnavailable
	}
	return contract.MutationIdentity{
		WorkspaceID: workspaceID, ActorType: actor.Type, ActorID: actor.ID, RequestID: requestID,
	}, installedTaskGovernancePolicy(action, resourceKind), nil
}

func (TaskGovernancePolicyProvider) ResolveGovernanceEventPolicy(_ context.Context, eventType, aggregateKind string) (GovernanceEventPolicy, error) {
	if _, ok := taskActionForEvent(eventType); !ok || (aggregateKind != "task" && aggregateKind != "task_order") {
		return GovernanceEventPolicy{}, contract.ErrGovernanceUnavailable
	}
	return GovernanceEventPolicy{EventType: eventType, AggregateKind: aggregateKind, Schema: taskGovernanceStateSchema()}, nil
}

func validTaskGovernanceBinding(action, resourceKind string) bool {
	if action == TaskActionReorder {
		return resourceKind == "task_order"
	}
	switch action {
	case TaskActionCreate, TaskActionUpdate, TaskActionArchive, TaskActionRestore:
		return resourceKind == "task"
	default:
		return false
	}
}

func installedTaskGovernancePolicy(action, resourceKind string) GovernanceActionPolicy {
	eventType, _ := taskEventForAction(action)
	requestSchema := taskGovernanceStateSchema()
	if action == TaskActionCreate {
		requestSchema = EnvelopeSchema{"fingerprint": {Kind: SafeIdentifier, MaxLength: 64, Required: true}}
	} else if action == TaskActionReorder {
		requestSchema = EnvelopeSchema{"fingerprint": {Kind: SafeSHA256, Required: true}}
	}
	return GovernanceActionPolicy{
		Action: action, ResourceKind: resourceKind,
		RequestSchema: requestSchema,
		ReplaySchema:  taskGovernanceStateSchema(),
		AuditSchema:   taskGovernanceStateSchema(),
		EventSchemas:  map[string]EnvelopeSchema{eventType: taskGovernanceStateSchema()},
	}
}

func taskGovernanceStateSchema() EnvelopeSchema {
	return EnvelopeSchema{
		"id":       {Kind: SafeIdentifier, MaxLength: 200, Required: true},
		"revision": {Kind: SafeNonNegativeInteger, Required: true},
		"status": {
			Kind: SafeEnum, Required: true,
			EnumValues: []string{"todo", "in_progress", "done", "cancelled", "removed", "reordered"},
		},
	}
}

func taskEventForAction(action string) (string, bool) {
	switch action {
	case TaskActionCreate:
		return "task:created", true
	case TaskActionUpdate:
		return "task:updated", true
	case TaskActionArchive:
		return "task:updated", true
	case TaskActionRestore:
		return "task:updated", true
	case TaskActionReorder:
		return "task:updated", true
	default:
		return "", false
	}
}

func taskActionForEvent(eventType string) (string, bool) {
	for _, action := range []string{TaskActionCreate, TaskActionUpdate, TaskActionArchive, TaskActionRestore, TaskActionReorder} {
		if candidate, _ := taskEventForAction(action); candidate == eventType {
			return action, true
		}
	}
	return "", false
}
