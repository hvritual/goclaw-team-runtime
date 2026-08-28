package domain

import (
	"errors"
	"testing"
	"time"
)

var testNow = time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

func mustProvenance(t *testing.T) Provenance {
	t.Helper()
	value, err := NewProvenance("github", "github://hvritual/device-cloud", "abc123", testNow)
	if err != nil {
		t.Fatalf("NewProvenance() error = %v", err)
	}
	return value
}

func mustNode(t *testing.T, kind NodeKind, id string) NodeRef {
	t.Helper()
	value, err := NewNodeRef(kind, id)
	if err != nil {
		t.Fatalf("NewNodeRef() error = %v", err)
	}
	return value
}

func TestEngineeringEntityVocabularyAndValidation(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		workspace  string
		entityType EntityType
		entityName string
		status     EntityStatus
		wantErr    error
	}{
		{name: "valid service", id: "service:device-gateway", workspace: "ws-1", entityType: EntityTypeService, entityName: "Device Gateway", status: EntityStatusActive},
		{name: "missing id", workspace: "ws-1", entityType: EntityTypeService, entityName: "Device Gateway", status: EntityStatusActive, wantErr: ErrEntityIDRequired},
		{name: "missing workspace", id: "service:device-gateway", entityType: EntityTypeService, entityName: "Device Gateway", status: EntityStatusActive, wantErr: ErrWorkspaceIDRequired},
		{name: "existing System name is not an engineering type", id: "system:device-cloud", workspace: "ws-1", entityType: EntityType("system"), entityName: "Device Cloud", status: EntityStatusActive, wantErr: ErrEntityTypeInvalid},
		{name: "invalid status", id: "service:device-gateway", workspace: "ws-1", entityType: EntityTypeService, entityName: "Device Gateway", status: EntityStatus("deleted"), wantErr: ErrEntityStatusInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewEngineeringEntity(test.id, test.workspace, test.entityType, test.entityName, test.status, "member-1")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestProvenanceIsMandatory(t *testing.T) {
	if _, err := NewProvenance("", "github://repo", "abc", testNow); !errors.Is(err, ErrSourceTypeRequired) {
		t.Fatalf("source type error = %v", err)
	}
	if _, err := NewProvenance("github", "", "abc", testNow); !errors.Is(err, ErrSourceLocatorRequired) {
		t.Fatalf("locator error = %v", err)
	}
	if _, err := NewProvenance("github", "github://repo", "abc", time.Time{}); !errors.Is(err, ErrObservedAtRequired) {
		t.Fatalf("observed-at error = %v", err)
	}
}

func TestSourceBindingAuthorityOnlyPromotes(t *testing.T) {
	binding, err := NewSourceBinding("binding-1", "ws-1", "service:device-gateway", mustProvenance(t), AuthorityInferred)
	if err != nil {
		t.Fatalf("NewSourceBinding() error = %v", err)
	}
	observed, err := binding.PromoteAuthority(AuthorityObserved)
	if err != nil {
		t.Fatalf("PromoteAuthority() error = %v", err)
	}
	if observed.Authority() != AuthorityObserved {
		t.Fatalf("authority = %q", observed.Authority())
	}
	if _, err := observed.PromoteAuthority(AuthorityInferred); !errors.Is(err, ErrAuthorityPromotionInvalid) {
		t.Fatalf("downgrade error = %v, want %v", err, ErrAuthorityPromotionInvalid)
	}
}

func TestThreadEdgeRejectsGenericOrUnprovenRelationships(t *testing.T) {
	project := mustNode(t, NodeKindProject, "project-1")
	service := mustNode(t, NodeKindEngineeringEntity, "service:device-gateway")

	if _, err := NewThreadEdge("edge-1", "ws-1", project, RelationChanges, service, AuthorityAuthoritative, mustProvenance(t)); err != nil {
		t.Fatalf("NewThreadEdge() error = %v", err)
	}
	if _, err := NewThreadEdge("edge-2", "ws-1", service, RelationDependsOn, service, AuthorityObserved, mustProvenance(t)); !errors.Is(err, ErrSelfThreadEdge) {
		t.Fatalf("self edge error = %v, want %v", err, ErrSelfThreadEdge)
	}
	if _, err := NewThreadEdge("edge-3", "ws-1", project, RelationType("related_to"), service, AuthorityObserved, mustProvenance(t)); !errors.Is(err, ErrRelationTypeInvalid) {
		t.Fatalf("generic relation error = %v, want %v", err, ErrRelationTypeInvalid)
	}
	if _, err := NewThreadEdge("edge-4", "ws-1", project, RelationChanges, service, AuthorityObserved, Provenance{}); !errors.Is(err, ErrProvenanceRequired) {
		t.Fatalf("provenance error = %v, want %v", err, ErrProvenanceRequired)
	}
}

func TestThreadEdgeAuthorityPromotionRequiresStrongerSource(t *testing.T) {
	project := mustNode(t, NodeKindProject, "project-1")
	service := mustNode(t, NodeKindEngineeringEntity, "service:device-gateway")
	edge, err := NewThreadEdge("edge-1", "ws-1", project, RelationChanges, service, AuthorityProposed, mustProvenance(t))
	if err != nil {
		t.Fatalf("NewThreadEdge() error = %v", err)
	}
	promoted, err := edge.PromoteAuthority(AuthorityAuthoritative, mustProvenance(t))
	if err != nil {
		t.Fatalf("PromoteAuthority() error = %v", err)
	}
	if promoted.Authority() != AuthorityAuthoritative {
		t.Fatalf("authority = %q", promoted.Authority())
	}
	if _, err := promoted.PromoteAuthority(AuthorityObserved, mustProvenance(t)); !errors.Is(err, ErrAuthorityPromotionInvalid) {
		t.Fatalf("downgrade error = %v, want %v", err, ErrAuthorityPromotionInvalid)
	}
}
