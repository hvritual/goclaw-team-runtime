package relationship

import (
	"errors"
	"testing"
)

func TestProjectActorRelationRoleMatrix(t *testing.T) {
	tests := []struct {
		name      string
		actorType string
		role      string
		wantErr   error
	}{
		{name: "member lead", actorType: ActorTypeMember, role: RoleLead},
		{name: "member member", actorType: ActorTypeMember, role: RoleMember},
		{name: "agent agent", actorType: ActorTypeAgent, role: RoleAgent},
		{name: "member cannot use agent role", actorType: ActorTypeMember, role: RoleAgent, wantErr: ErrRoleInvalid},
		{name: "agent cannot lead", actorType: ActorTypeAgent, role: RoleLead, wantErr: ErrRoleInvalid},
		{name: "unknown actor type", actorType: "team", role: RoleMember, wantErr: ErrActorTypeInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := New("workspace-1", "project-1", tt.actorType, "actor-1", tt.role)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (value.ActorType() != tt.actorType || value.Role() != tt.role) {
				t.Fatalf("unexpected relation: %+v", value)
			}
		})
	}
}

func TestValidateProjectActorReference(t *testing.T) {
	for _, tt := range []struct {
		name                         string
		workspace, project, kind, id string
		wantErr                      error
	}{
		{name: "valid", workspace: "w", project: "p", kind: ActorTypeMember, id: "a"},
		{name: "workspace", project: "p", kind: ActorTypeMember, id: "a", wantErr: ErrWorkspaceRequired},
		{name: "project", workspace: "w", kind: ActorTypeMember, id: "a", wantErr: ErrProjectRequired},
		{name: "kind", workspace: "w", project: "p", kind: "team", id: "a", wantErr: ErrActorTypeInvalid},
		{name: "actor", workspace: "w", project: "p", kind: ActorTypeMember, wantErr: ErrActorIDRequired},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateReference(tt.workspace, tt.project, tt.kind, tt.id); !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateReference() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
