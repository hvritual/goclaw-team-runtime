package domain

import (
	"errors"
	"strings"
	"time"
)

type EntityType string

const (
	EntityTypeProduct           EntityType = "product"
	EntityTypeEngineeringSystem EntityType = "engineering_system"
	EntityTypeApplication       EntityType = "application"
	EntityTypeService           EntityType = "service"
	EntityTypeComponent         EntityType = "component"
	EntityTypeRepository        EntityType = "repository"
	EntityTypeAPI               EntityType = "api"
	EntityTypeThingModel        EntityType = "thing_model"
	EntityTypeEnvironment       EntityType = "environment"
)

var validEntityTypes = map[EntityType]struct{}{
	EntityTypeProduct: {}, EntityTypeEngineeringSystem: {}, EntityTypeApplication: {},
	EntityTypeService: {}, EntityTypeComponent: {}, EntityTypeRepository: {},
	EntityTypeAPI: {}, EntityTypeThingModel: {}, EntityTypeEnvironment: {},
}

func (value EntityType) Valid() bool {
	_, ok := validEntityTypes[EntityType(strings.TrimSpace(string(value)))]
	return ok
}

type EntityStatus string

const (
	EntityStatusDraft      EntityStatus = "draft"
	EntityStatusActive     EntityStatus = "active"
	EntityStatusDeprecated EntityStatus = "deprecated"
	EntityStatusArchived   EntityStatus = "archived"
)

var validEntityStatuses = map[EntityStatus]struct{}{
	EntityStatusDraft: {}, EntityStatusActive: {}, EntityStatusDeprecated: {}, EntityStatusArchived: {},
}

func (value EntityStatus) Valid() bool {
	_, ok := validEntityStatuses[EntityStatus(strings.TrimSpace(string(value)))]
	return ok
}

type Authority string

const (
	AuthorityProposed      Authority = "proposed"
	AuthorityInferred      Authority = "inferred"
	AuthorityObserved      Authority = "observed"
	AuthorityAuthoritative Authority = "authoritative"
)

var authorityRank = map[Authority]int{
	AuthorityProposed:      0,
	AuthorityInferred:      1,
	AuthorityObserved:      2,
	AuthorityAuthoritative: 3,
}

func (value Authority) Valid() bool {
	_, ok := authorityRank[Authority(strings.TrimSpace(string(value)))]
	return ok
}

func (value Authority) CanPromoteTo(next Authority) bool {
	currentRank, currentOK := authorityRank[value]
	nextRank, nextOK := authorityRank[next]
	return currentOK && nextOK && nextRank > currentRank
}

type RelationType string

const (
	RelationPartOf        RelationType = "part_of"
	RelationDependsOn     RelationType = "depends_on"
	RelationImplements    RelationType = "implements"
	RelationProvides      RelationType = "provides"
	RelationUses          RelationType = "uses"
	RelationChanges       RelationType = "changes"
	RelationAffects       RelationType = "affects"
	RelationContributesTo RelationType = "contributes_to"
	RelationConstrains    RelationType = "constrains"
	RelationGoverns       RelationType = "governs"
	RelationOperates      RelationType = "operates"
	RelationOwns          RelationType = "owns"
	RelationIntroducedBy  RelationType = "introduced_by"
	RelationIncludedIn    RelationType = "included_in"
	RelationDeployedTo    RelationType = "deployed_to"
)

var validRelationTypes = map[RelationType]struct{}{
	RelationPartOf: {}, RelationDependsOn: {}, RelationImplements: {}, RelationProvides: {},
	RelationUses: {}, RelationChanges: {}, RelationAffects: {}, RelationContributesTo: {},
	RelationConstrains: {}, RelationGoverns: {}, RelationOperates: {}, RelationOwns: {},
	RelationIntroducedBy: {}, RelationIncludedIn: {}, RelationDeployedTo: {},
}

func (value RelationType) Valid() bool {
	_, ok := validRelationTypes[RelationType(strings.TrimSpace(string(value)))]
	return ok
}

type NodeKind string

const (
	NodeKindEngineeringEntity NodeKind = "engineering_entity"
	NodeKindProject           NodeKind = "project"
	NodeKindRequirement       NodeKind = "requirement"
	NodeKindIssue             NodeKind = "issue"
	NodeKindTodo              NodeKind = "todo"
	NodeKindTask              NodeKind = "task"
	NodeKindRun               NodeKind = "run"
	NodeKindChange            NodeKind = "change"
	NodeKindArtifact          NodeKind = "artifact"
	NodeKindEvidence          NodeKind = "evidence"
	NodeKindFact              NodeKind = "fact"
	NodeKindKnowledge         NodeKind = "knowledge"
	NodeKindMember            NodeKind = "member"
	NodeKindSkill             NodeKind = "skill"
	NodeKindRelease           NodeKind = "release"
	NodeKindDeployment        NodeKind = "deployment"
)

var validNodeKinds = map[NodeKind]struct{}{
	NodeKindEngineeringEntity: {}, NodeKindProject: {}, NodeKindRequirement: {}, NodeKindIssue: {},
	NodeKindTodo: {}, NodeKindTask: {}, NodeKindRun: {}, NodeKindChange: {}, NodeKindArtifact: {},
	NodeKindEvidence: {}, NodeKindFact: {}, NodeKindKnowledge: {}, NodeKindMember: {}, NodeKindSkill: {},
	NodeKindRelease: {}, NodeKindDeployment: {},
}

func (value NodeKind) Valid() bool {
	_, ok := validNodeKinds[NodeKind(strings.TrimSpace(string(value)))]
	return ok
}

var (
	ErrWorkspaceIDRequired       = errors.New("workspace id is required")
	ErrEntityIDRequired          = errors.New("engineering entity id is required")
	ErrEntityNameRequired        = errors.New("engineering entity name is required")
	ErrEntityTypeInvalid         = errors.New("invalid engineering entity type")
	ErrEntityStatusInvalid       = errors.New("invalid engineering entity status")
	ErrSourceBindingIDRequired   = errors.New("source binding id is required")
	ErrSourceTypeRequired        = errors.New("source type is required")
	ErrSourceLocatorRequired     = errors.New("source locator is required")
	ErrObservedAtRequired        = errors.New("observed at is required")
	ErrAuthorityInvalid          = errors.New("invalid source authority")
	ErrAuthorityPromotionInvalid = errors.New("authority must move to a strictly stronger class")
	ErrNodeKindInvalid           = errors.New("invalid thread node kind")
	ErrNodeIDRequired            = errors.New("thread node id is required")
	ErrThreadEdgeIDRequired      = errors.New("thread edge id is required")
	ErrRelationTypeInvalid       = errors.New("invalid thread relation type")
	ErrSelfThreadEdge            = errors.New("thread edge cannot point to itself")
	ErrProvenanceRequired        = errors.New("thread edge provenance is required")
)

type Provenance struct {
	sourceType string
	locator    string
	revision   string
	observedAt time.Time
}

func NewProvenance(sourceType, locator, revision string, observedAt time.Time) (Provenance, error) {
	sourceType = strings.TrimSpace(sourceType)
	locator = strings.TrimSpace(locator)
	revision = strings.TrimSpace(revision)
	if sourceType == "" {
		return Provenance{}, ErrSourceTypeRequired
	}
	if locator == "" {
		return Provenance{}, ErrSourceLocatorRequired
	}
	if observedAt.IsZero() {
		return Provenance{}, ErrObservedAtRequired
	}
	return Provenance{sourceType: sourceType, locator: locator, revision: revision, observedAt: observedAt.UTC()}, nil
}

func (value Provenance) Valid() bool {
	return value.sourceType != "" && value.locator != "" && !value.observedAt.IsZero()
}
func (value Provenance) SourceType() string    { return value.sourceType }
func (value Provenance) Locator() string       { return value.locator }
func (value Provenance) Revision() string      { return value.revision }
func (value Provenance) ObservedAt() time.Time { return value.observedAt }

type NodeRef struct {
	kind NodeKind
	id   string
}

func NewNodeRef(kind NodeKind, id string) (NodeRef, error) {
	id = strings.TrimSpace(id)
	if !kind.Valid() {
		return NodeRef{}, ErrNodeKindInvalid
	}
	if id == "" {
		return NodeRef{}, ErrNodeIDRequired
	}
	return NodeRef{kind: kind, id: id}, nil
}

func (value NodeRef) Kind() NodeKind { return value.kind }
func (value NodeRef) ID() string     { return value.id }
func (value NodeRef) Equal(other NodeRef) bool {
	return value.kind == other.kind && value.id == other.id
}

type EngineeringEntity struct {
	id          string
	workspaceID string
	entityType  EntityType
	name        string
	status      EntityStatus
	ownerRef    string
}

func NewEngineeringEntity(id, workspaceID string, entityType EntityType, name string, status EntityStatus, ownerRef string) (EngineeringEntity, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	name = strings.TrimSpace(name)
	ownerRef = strings.TrimSpace(ownerRef)
	if id == "" {
		return EngineeringEntity{}, ErrEntityIDRequired
	}
	if workspaceID == "" {
		return EngineeringEntity{}, ErrWorkspaceIDRequired
	}
	if !entityType.Valid() {
		return EngineeringEntity{}, ErrEntityTypeInvalid
	}
	if name == "" {
		return EngineeringEntity{}, ErrEntityNameRequired
	}
	if status == "" {
		status = EntityStatusDraft
	}
	if !status.Valid() {
		return EngineeringEntity{}, ErrEntityStatusInvalid
	}
	return EngineeringEntity{id: id, workspaceID: workspaceID, entityType: entityType, name: name, status: status, ownerRef: ownerRef}, nil
}

func (value EngineeringEntity) ID() string           { return value.id }
func (value EngineeringEntity) WorkspaceID() string  { return value.workspaceID }
func (value EngineeringEntity) Type() EntityType     { return value.entityType }
func (value EngineeringEntity) Name() string         { return value.name }
func (value EngineeringEntity) Status() EntityStatus { return value.status }
func (value EngineeringEntity) OwnerRef() string     { return value.ownerRef }

type SourceBinding struct {
	id          string
	workspaceID string
	entityID    string
	provenance  Provenance
	authority   Authority
}

func NewSourceBinding(id, workspaceID, entityID string, provenance Provenance, authority Authority) (SourceBinding, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	entityID = strings.TrimSpace(entityID)
	if id == "" {
		return SourceBinding{}, ErrSourceBindingIDRequired
	}
	if workspaceID == "" {
		return SourceBinding{}, ErrWorkspaceIDRequired
	}
	if entityID == "" {
		return SourceBinding{}, ErrEntityIDRequired
	}
	if !provenance.Valid() {
		return SourceBinding{}, ErrProvenanceRequired
	}
	if !authority.Valid() {
		return SourceBinding{}, ErrAuthorityInvalid
	}
	return SourceBinding{id: id, workspaceID: workspaceID, entityID: entityID, provenance: provenance, authority: authority}, nil
}

func (value SourceBinding) PromoteAuthority(next Authority) (SourceBinding, error) {
	if !value.authority.CanPromoteTo(next) {
		return SourceBinding{}, ErrAuthorityPromotionInvalid
	}
	value.authority = next
	return value, nil
}

func (value SourceBinding) ID() string             { return value.id }
func (value SourceBinding) WorkspaceID() string    { return value.workspaceID }
func (value SourceBinding) EntityID() string       { return value.entityID }
func (value SourceBinding) Provenance() Provenance { return value.provenance }
func (value SourceBinding) Authority() Authority   { return value.authority }

type ThreadEdge struct {
	id          string
	workspaceID string
	from        NodeRef
	relation    RelationType
	to          NodeRef
	authority   Authority
	provenance  Provenance
}

func NewThreadEdge(id, workspaceID string, from NodeRef, relation RelationType, to NodeRef, authority Authority, provenance Provenance) (ThreadEdge, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	if id == "" {
		return ThreadEdge{}, ErrThreadEdgeIDRequired
	}
	if workspaceID == "" {
		return ThreadEdge{}, ErrWorkspaceIDRequired
	}
	if !from.Kind().Valid() || from.ID() == "" || !to.Kind().Valid() || to.ID() == "" {
		return ThreadEdge{}, ErrNodeIDRequired
	}
	if from.Equal(to) {
		return ThreadEdge{}, ErrSelfThreadEdge
	}
	if !relation.Valid() {
		return ThreadEdge{}, ErrRelationTypeInvalid
	}
	if !authority.Valid() {
		return ThreadEdge{}, ErrAuthorityInvalid
	}
	if !provenance.Valid() {
		return ThreadEdge{}, ErrProvenanceRequired
	}
	return ThreadEdge{id: id, workspaceID: workspaceID, from: from, relation: relation, to: to, authority: authority, provenance: provenance}, nil
}

func (value ThreadEdge) PromoteAuthority(next Authority, provenance Provenance) (ThreadEdge, error) {
	if !value.authority.CanPromoteTo(next) {
		return ThreadEdge{}, ErrAuthorityPromotionInvalid
	}
	if !provenance.Valid() {
		return ThreadEdge{}, ErrProvenanceRequired
	}
	value.authority = next
	value.provenance = provenance
	return value, nil
}

func (value ThreadEdge) ID() string             { return value.id }
func (value ThreadEdge) WorkspaceID() string    { return value.workspaceID }
func (value ThreadEdge) From() NodeRef          { return value.from }
func (value ThreadEdge) Relation() RelationType { return value.relation }
func (value ThreadEdge) To() NodeRef            { return value.to }
func (value ThreadEdge) Authority() Authority   { return value.authority }
func (value ThreadEdge) Provenance() Provenance { return value.provenance }
