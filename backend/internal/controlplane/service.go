package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	PermissionRead          = "workspace.read"
	PermissionWrite         = "workspace.write"
	PermissionManageMembers = "workspace.members.manage"
	PermissionReview        = "delivery.review"
	PermissionRun           = "delivery.run"
	PermissionAccept        = "delivery.accept"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type Service struct {
	repository Repository
	now        Clock
}

func NewService(repository Repository, clock Clock) (*Service, error) {
	if repository == nil {
		return nil, invalid("new service", "repository", "is required")
	}
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{repository: repository, now: clock}, nil
}

func (s *Service) CreateWorkspace(ctx context.Context, actor Actor, id, name string) (Workspace, error) {
	const op = "create workspace"
	if err := validateActor(actor, false); err != nil {
		return Workspace{}, err
	}
	if actor.Kind != ActorHuman {
		return Workspace{}, denied(op, "a human owner is required to create a workspace")
	}
	if err := validateIdentifier(op, "workspace_id", id); err != nil {
		return Workspace{}, err
	}
	if strings.TrimSpace(name) == "" {
		return Workspace{}, invalid(op, "name", "is required")
	}
	now := s.now()
	workspace := Workspace{ID: id, Name: strings.TrimSpace(name), State: WorkspaceActive, Version: 1, CreatedAt: now, UpdatedAt: now}
	owner := Member{WorkspaceID: id, ID: actor.ID, Kind: actor.Kind, Role: RoleOwner, State: MemberActive, Version: 1, CreatedAt: now, UpdatedAt: now}
	audit := s.audit(id, actor.ID, "workspace.create", "workspace", id, nil)
	if err := s.repository.CreateWorkspace(ctx, workspace, owner, audit); err != nil {
		return Workspace{}, fmt.Errorf("%s: %w", op, err)
	}
	return workspace, nil
}

func (s *Service) GetWorkspace(ctx context.Context, actor Actor) (Workspace, error) {
	if err := s.Authorize(ctx, actor, PermissionRead); err != nil {
		return Workspace{}, err
	}
	return s.repository.GetWorkspace(ctx, actor.WorkspaceID)
}

func (s *Service) UpdateWorkspace(ctx context.Context, actor Actor, name string, state WorkspaceState, expectedVersion int64) (Workspace, error) {
	const op = "update workspace"
	if err := s.Authorize(ctx, actor, PermissionWrite); err != nil {
		return Workspace{}, err
	}
	workspace, err := s.repository.GetWorkspace(ctx, actor.WorkspaceID)
	if err != nil {
		return Workspace{}, err
	}
	if workspace.Version != expectedVersion {
		return Workspace{}, conflict(op, "expected version does not match")
	}
	if strings.TrimSpace(name) == "" {
		return Workspace{}, invalid(op, "name", "is required")
	}
	if state != WorkspaceActive && state != WorkspaceArchived {
		return Workspace{}, invalid(op, "state", "is unsupported")
	}
	workspace.Name, workspace.State = strings.TrimSpace(name), state
	workspace.Version++
	workspace.UpdatedAt = s.now()
	audit := s.audit(workspace.ID, actor.ID, "workspace.update", "workspace", workspace.ID, nil)
	if err := s.repository.UpdateWorkspace(ctx, workspace, expectedVersion, audit); err != nil {
		return Workspace{}, err
	}
	return workspace, nil
}

func (s *Service) AddMember(ctx context.Context, actor Actor, id string, kind ActorKind, role Role) (Member, error) {
	const op = "add member"
	if err := s.Authorize(ctx, actor, PermissionManageMembers); err != nil {
		return Member{}, err
	}
	if err := validateIdentifier(op, "member_id", id); err != nil {
		return Member{}, err
	}
	if err := validateKindRole(op, kind, role); err != nil {
		return Member{}, err
	}
	actorMember, err := s.repository.GetMember(ctx, actor.WorkspaceID, actor.ID)
	if err != nil {
		return Member{}, err
	}
	if role == RoleOwner && actorMember.Role != RoleOwner {
		return Member{}, denied(op, "only an owner can add another owner")
	}
	now := s.now()
	member := Member{WorkspaceID: actor.WorkspaceID, ID: id, Kind: kind, Role: role, State: MemberActive, Version: 1, CreatedAt: now, UpdatedAt: now}
	audit := s.audit(actor.WorkspaceID, actor.ID, "member.add", "member", id, map[string]string{"role": string(role)})
	if err := s.repository.SaveMember(ctx, member, 0, audit); err != nil {
		return Member{}, err
	}
	return member, nil
}

func (s *Service) ChangeMemberRole(ctx context.Context, actor Actor, memberID string, role Role, expectedVersion int64) (Member, error) {
	const op = "change member role"
	if err := s.Authorize(ctx, actor, PermissionManageMembers); err != nil {
		return Member{}, err
	}
	member, err := s.repository.GetMember(ctx, actor.WorkspaceID, memberID)
	if err != nil {
		return Member{}, err
	}
	if member.Version != expectedVersion {
		return Member{}, conflict(op, "expected version does not match")
	}
	if err := validateKindRole(op, member.Kind, role); err != nil {
		return Member{}, err
	}
	actorMember, err := s.repository.GetMember(ctx, actor.WorkspaceID, actor.ID)
	if err != nil {
		return Member{}, err
	}
	if (member.Role == RoleOwner || role == RoleOwner) && actorMember.Role != RoleOwner {
		return Member{}, denied(op, "only an owner can change ownership")
	}
	if member.Role == RoleOwner && role != RoleOwner {
		if err := s.requireAnotherOwner(ctx, actor.WorkspaceID, member.ID); err != nil {
			return Member{}, err
		}
	}
	member.Role = role
	member.Version++
	member.UpdatedAt = s.now()
	audit := s.audit(actor.WorkspaceID, actor.ID, "member.role.change", "member", member.ID, map[string]string{"role": string(role)})
	if err := s.repository.SaveMember(ctx, member, expectedVersion, audit); err != nil {
		return Member{}, err
	}
	return member, nil
}

func (s *Service) RemoveMember(ctx context.Context, actor Actor, memberID string, expectedVersion int64) (Member, error) {
	const op = "remove member"
	if err := s.Authorize(ctx, actor, PermissionManageMembers); err != nil {
		return Member{}, err
	}
	member, err := s.repository.GetMember(ctx, actor.WorkspaceID, memberID)
	if err != nil {
		return Member{}, err
	}
	if member.Version != expectedVersion {
		return Member{}, conflict(op, "expected version does not match")
	}
	if member.Role == RoleOwner {
		actorMember, getErr := s.repository.GetMember(ctx, actor.WorkspaceID, actor.ID)
		if getErr != nil {
			return Member{}, getErr
		}
		if actorMember.Role != RoleOwner {
			return Member{}, denied(op, "only an owner can remove an owner")
		}
		if err := s.requireAnotherOwner(ctx, actor.WorkspaceID, member.ID); err != nil {
			return Member{}, err
		}
	}
	member.State = MemberRemoved
	member.Version++
	member.UpdatedAt = s.now()
	audit := s.audit(actor.WorkspaceID, actor.ID, "member.remove", "member", member.ID, nil)
	if err := s.repository.SaveMember(ctx, member, expectedVersion, audit); err != nil {
		return Member{}, err
	}
	return member, nil
}

func (s *Service) ListMembers(ctx context.Context, actor Actor, includeRemoved bool) ([]Member, error) {
	if err := s.Authorize(ctx, actor, PermissionRead); err != nil {
		return nil, err
	}
	return s.repository.ListMembers(ctx, actor.WorkspaceID, includeRemoved)
}

func (s *Service) Authorize(ctx context.Context, actor Actor, permission string) error {
	const op = "authorize"
	if err := validateActor(actor, true); err != nil {
		return err
	}
	member, err := s.repository.GetMember(ctx, actor.WorkspaceID, actor.ID)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%s: resolve member: %w", op, err)
		}
		return denied(op, "actor is not an active workspace member")
	}
	if member.State != MemberActive || member.Kind != actor.Kind {
		return denied(op, "actor identity is inactive or mismatched")
	}
	if !roleAllows(member.Role, permission) {
		return denied(op, "role does not grant "+permission)
	}
	return nil
}

func (s *Service) SaveRecord(ctx context.Context, actor Actor, record Record, expectedVersion int64) (Record, error) {
	const op = "save record"
	if err := s.Authorize(ctx, actor, PermissionWrite); err != nil {
		return Record{}, err
	}
	if record.WorkspaceID != actor.WorkspaceID {
		return Record{}, denied(op, "record workspace does not match actor workspace")
	}
	if err := validateIdentifier(op, "record_id", record.ID); err != nil {
		return Record{}, err
	}
	if strings.TrimSpace(record.Kind) == "" || strings.TrimSpace(record.State) == "" || !json.Valid(record.Payload) {
		return Record{}, invalid(op, "record", "kind, state, and valid JSON payload are required")
	}
	now := s.now()
	if expectedVersion == 0 {
		record.Version, record.CreatedAt = 1, now
	} else {
		record.Version = expectedVersion + 1
		current, err := s.repository.GetRecord(ctx, actor.WorkspaceID, record.Kind, record.ID)
		if err != nil {
			return Record{}, err
		}
		record.CreatedAt = current.CreatedAt
	}
	record.UpdatedAt = now
	audit := s.audit(actor.WorkspaceID, actor.ID, "record.save", record.Kind, record.ID, nil)
	return s.repository.SaveRecord(ctx, record, expectedVersion, audit)
}

func (s *Service) GetRecord(ctx context.Context, actor Actor, kind, id string) (Record, error) {
	if err := s.Authorize(ctx, actor, PermissionRead); err != nil {
		return Record{}, err
	}
	return s.repository.GetRecord(ctx, actor.WorkspaceID, kind, id)
}

func (s *Service) ListRecords(ctx context.Context, actor Actor, kind string, page Page) ([]Record, error) {
	if err := s.Authorize(ctx, actor, PermissionRead); err != nil {
		return nil, err
	}
	return s.repository.ListRecords(ctx, actor.WorkspaceID, kind, page)
}

func (s *Service) ListAudit(ctx context.Context, actor Actor, page Page) ([]AuditEntry, error) {
	if err := s.Authorize(ctx, actor, PermissionReview); err != nil {
		return nil, err
	}
	return s.repository.ListAudit(ctx, actor.WorkspaceID, page)
}

func (s *Service) requireAnotherOwner(ctx context.Context, workspaceID, excludedID string) error {
	members, err := s.repository.ListMembers(ctx, workspaceID, false)
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.ID != excludedID && member.Role == RoleOwner && member.State == MemberActive {
			return nil
		}
	}
	return invariant("change ownership", "workspace must retain at least one active owner")
}

func (s *Service) audit(workspaceID, actorID, action, resource, resourceID string, metadata map[string]string) AuditEntry {
	encoded, _ := json.Marshal(metadata)
	return AuditEntry{ID: uuid.NewString(), WorkspaceID: workspaceID, ActorID: actorID, Action: action,
		Resource: resource, ResourceID: resourceID, Metadata: encoded, OccurredAt: s.now()}
}

func validateActor(actor Actor, requireWorkspace bool) error {
	if err := validateIdentifier("validate actor", "actor_id", actor.ID); err != nil {
		return err
	}
	if requireWorkspace {
		if err := validateIdentifier("validate actor", "workspace_id", actor.WorkspaceID); err != nil {
			return err
		}
	}
	if actor.Kind != ActorHuman && actor.Kind != ActorAgent {
		return invalid("validate actor", "kind", "must be human or agent")
	}
	return nil
}

func validateIdentifier(op, field, value string) error {
	if !identifierPattern.MatchString(value) {
		return invalid(op, field, "must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
	}
	return nil
}

func validateKindRole(op string, kind ActorKind, role Role) error {
	if kind != ActorHuman && kind != ActorAgent {
		return invalid(op, "kind", "must be human or agent")
	}
	switch role {
	case RoleOwner, RoleAdmin, RoleMember, RoleReviewer, RoleViewer:
	default:
		return invalid(op, "role", "is unsupported")
	}
	if kind == ActorAgent && (role == RoleOwner || role == RoleAdmin || role == RoleReviewer) {
		return invariant(op, "agents cannot own, administer, or independently review a workspace")
	}
	return nil
}

func roleAllows(role Role, permission string) bool {
	switch permission {
	case PermissionRead:
		return role == RoleOwner || role == RoleAdmin || role == RoleMember || role == RoleReviewer || role == RoleViewer
	case PermissionWrite, PermissionRun:
		return role == RoleOwner || role == RoleAdmin || role == RoleMember
	case PermissionManageMembers:
		return role == RoleOwner || role == RoleAdmin
	case PermissionReview:
		return role == RoleOwner || role == RoleAdmin || role == RoleReviewer
	case PermissionAccept:
		return role == RoleOwner || role == RoleReviewer
	default:
		return false
	}
}
