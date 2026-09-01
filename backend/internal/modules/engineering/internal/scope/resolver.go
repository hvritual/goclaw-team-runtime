package scope

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

const (
	DefaultMaxDepth    = 2
	HardMaxDepth       = 4
	DefaultMaxEntities = 64
	HardMaxEntities    = 256
)

var (
	ErrRepositoryRequired = errors.New("scope resolver repository is required")
	ErrInvalidInput       = errors.New("invalid scope resolver input")
	ErrNoScope            = errors.New("no authoritative engineering scope")
)

type Repository interface {
	domain.Repository
}

type Policy struct {
	MaxDepth         int
	MaxEntities      int
	SourceStaleAfter time.Duration
}

type Input struct {
	WorkspaceID string
	WorkItem    domain.NodeRef
	Policy      Policy
}

type ScopedEntity struct {
	ID          string
	Type        string
	Name        string
	Status      string
	OwnerRef    string
	Depth       int
	ParentID    string
	ViaEdgeID   string
	ViaRelation string
	Direction   string
}

type SourceRef struct {
	BindingID  string
	EntityID   string
	SourceType string
	Locator    string
	Revision   string
	Authority  string
	ObservedAt time.Time
	Stale      bool
}

type Warning struct {
	Code      string
	EntityID  string
	EdgeID    string
	BindingID string
	Detail    string
}

type Result struct {
	WorkspaceID string
	WorkItem    domain.NodeRef
	Entities    []ScopedEntity
	Sources     []SourceRef
	Warnings    []Warning
	Truncated   bool
}

type Resolver struct {
	repository Repository
	now        func() time.Time
}

type queueItem struct {
	entity      domain.EngineeringEntity
	depth       int
	parentID    string
	viaEdgeID   string
	viaRelation domain.RelationType
	direction   string
}

func New(repository Repository, now func() time.Time) (*Resolver, error) {
	if repository == nil {
		return nil, ErrRepositoryRequired
	}
	if now == nil {
		now = time.Now
	}
	return &Resolver{repository: repository, now: now}, nil
}

func (r *Resolver) Resolve(ctx context.Context, input Input) (Result, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	policy, err := normalizePolicy(input.Policy)
	if workspaceID == "" || err != nil {
		return Result{}, ErrInvalidInput
	}
	workRelation, err := expectedWorkRelation(input.WorkItem)
	if err != nil {
		return Result{}, ErrInvalidInput
	}

	workEdges, err := r.repository.ListThreadEdges(ctx, workspaceID, input.WorkItem)
	if err != nil {
		return Result{}, fmt.Errorf("list workspace engineering scope links: %w", err)
	}
	sort.Slice(workEdges, func(i, j int) bool { return workEdges[i].ID() < workEdges[j].ID() })

	result := Result{WorkspaceID: workspaceID, WorkItem: input.WorkItem}
	visited := make(map[string]struct{})
	queue := make([]queueItem, 0)
	for _, edge := range workEdges {
		if !isAuthoritativeWorkSeed(edge, input.WorkItem, workRelation) {
			continue
		}
		entity, getErr := r.repository.GetEntity(ctx, workspaceID, edge.To().ID())
		if errors.Is(getErr, domain.ErrNotFound) {
			result.Warnings = append(result.Warnings, Warning{Code: "dangling_work_link", EntityID: edge.To().ID(), EdgeID: edge.ID(), Detail: "workspace work-link target is missing"})
			continue
		}
		if getErr != nil {
			return Result{}, fmt.Errorf("read workspace scope seed %s: %w", edge.To().ID(), getErr)
		}
		if _, exists := visited[entity.ID()]; exists {
			continue
		}
		if len(visited) >= policy.MaxEntities {
			result.Truncated = true
			result.Warnings = appendLimitWarning(result.Warnings, policy.MaxEntities)
			break
		}
		visited[entity.ID()] = struct{}{}
		queue = append(queue, queueItem{entity: entity, depth: 0, viaEdgeID: edge.ID(), viaRelation: edge.Relation(), direction: "seed"})
	}
	if len(queue) == 0 {
		if len(result.Warnings) > 0 {
			sortWarnings(result.Warnings)
		}
		return Result{}, ErrNoScope
	}

	for cursor := 0; cursor < len(queue); cursor++ {
		current := queue[cursor]
		result.Entities = append(result.Entities, scopedEntity(current))
		if current.entity.Status() == domain.EntityStatusArchived {
			result.Warnings = append(result.Warnings, Warning{Code: "archived_entity", EntityID: current.entity.ID(), Detail: "archived engineering entity retained in scope but not expanded"})
			continue
		}
		if current.depth >= policy.MaxDepth || result.Truncated {
			continue
		}
		node, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, current.entity.ID())
		edges, listErr := r.repository.ListThreadEdges(ctx, workspaceID, node)
		if listErr != nil {
			return Result{}, fmt.Errorf("expand engineering scope from %s: %w", current.entity.ID(), listErr)
		}
		sort.Slice(edges, func(i, j int) bool { return edges[i].ID() < edges[j].ID() })
		for _, edge := range edges {
			neighborID, direction, ok := traversableNeighbor(edge, node)
			if !ok {
				continue
			}
			if _, exists := visited[neighborID]; exists {
				continue
			}
			neighbor, getErr := r.repository.GetEntity(ctx, workspaceID, neighborID)
			if errors.Is(getErr, domain.ErrNotFound) {
				result.Warnings = append(result.Warnings, Warning{Code: "dangling_edge", EntityID: current.entity.ID(), EdgeID: edge.ID(), Detail: "engineering edge target is missing: " + neighborID})
				continue
			}
			if getErr != nil {
				return Result{}, fmt.Errorf("read engineering scope neighbor %s: %w", neighborID, getErr)
			}
			if len(visited) >= policy.MaxEntities {
				result.Truncated = true
				result.Warnings = appendLimitWarning(result.Warnings, policy.MaxEntities)
				break
			}
			visited[neighbor.ID()] = struct{}{}
			queue = append(queue, queueItem{
				entity: neighbor, depth: current.depth + 1, parentID: current.entity.ID(),
				viaEdgeID: edge.ID(), viaRelation: edge.Relation(), direction: direction,
			})
		}
	}

	sort.Slice(result.Entities, func(i, j int) bool {
		if result.Entities[i].Depth != result.Entities[j].Depth {
			return result.Entities[i].Depth < result.Entities[j].Depth
		}
		return result.Entities[i].ID < result.Entities[j].ID
	})

	now := r.now().UTC()
	for _, scoped := range result.Entities {
		bindings, listErr := r.repository.ListSourceBindings(ctx, workspaceID, scoped.ID)
		if listErr != nil {
			return Result{}, fmt.Errorf("list engineering sources for %s: %w", scoped.ID, listErr)
		}
		sort.Slice(bindings, func(i, j int) bool { return bindings[i].ID() < bindings[j].ID() })
		for _, binding := range bindings {
			provenance := binding.Provenance()
			stale := policy.SourceStaleAfter > 0 && provenance.ObservedAt().Before(now.Add(-policy.SourceStaleAfter))
			ref := SourceRef{
				BindingID: binding.ID(), EntityID: binding.EntityID(), SourceType: provenance.SourceType(), Locator: provenance.Locator(),
				Revision: provenance.Revision(), Authority: string(binding.Authority()), ObservedAt: provenance.ObservedAt(), Stale: stale,
			}
			result.Sources = append(result.Sources, ref)
			if stale {
				result.Warnings = append(result.Warnings, Warning{Code: "stale_source", EntityID: scoped.ID, BindingID: binding.ID(), Detail: "source observation exceeds configured freshness window"})
			}
			if (binding.Authority() == domain.AuthorityAuthoritative || binding.Authority() == domain.AuthorityObserved) && strings.TrimSpace(provenance.Revision()) == "" {
				result.Warnings = append(result.Warnings, Warning{Code: "unpinned_source", EntityID: scoped.ID, BindingID: binding.ID(), Detail: "source has no immutable revision"})
			}
		}
	}
	sort.Slice(result.Sources, func(i, j int) bool {
		if result.Sources[i].EntityID != result.Sources[j].EntityID {
			return result.Sources[i].EntityID < result.Sources[j].EntityID
		}
		return result.Sources[i].BindingID < result.Sources[j].BindingID
	})
	sortWarnings(result.Warnings)
	return result, nil
}

func normalizePolicy(value Policy) (Policy, error) {
	if value.MaxDepth < 0 || value.MaxEntities < 0 || value.SourceStaleAfter < 0 {
		return Policy{}, ErrInvalidInput
	}
	if value.MaxDepth == 0 {
		value.MaxDepth = DefaultMaxDepth
	}
	if value.MaxEntities == 0 {
		value.MaxEntities = DefaultMaxEntities
	}
	if value.MaxDepth > HardMaxDepth || value.MaxEntities > HardMaxEntities {
		return Policy{}, ErrInvalidInput
	}
	return value, nil
}

func expectedWorkRelation(work domain.NodeRef) (domain.RelationType, error) {
	if strings.TrimSpace(work.ID()) == "" {
		return "", ErrInvalidInput
	}
	switch work.Kind() {
	case domain.NodeKindProject:
		return domain.RelationChanges, nil
	case domain.NodeKindRequirement, domain.NodeKindTask:
		return domain.RelationAffects, nil
	default:
		return "", ErrInvalidInput
	}
}

func isAuthoritativeWorkSeed(edge domain.ThreadEdge, work domain.NodeRef, relation domain.RelationType) bool {
	return edge.From().Equal(work) && edge.Relation() == relation && edge.To().Kind() == domain.NodeKindEngineeringEntity &&
		edge.Authority() == domain.AuthorityAuthoritative && edge.Provenance().SourceType() == "workspace"
}

func traversableNeighbor(edge domain.ThreadEdge, current domain.NodeRef) (string, string, bool) {
	if edge.Authority() != domain.AuthorityAuthoritative || edge.From().Kind() != domain.NodeKindEngineeringEntity || edge.To().Kind() != domain.NodeKindEngineeringEntity {
		return "", "", false
	}
	if edge.From().Equal(current) && outboundRelation(edge.Relation()) {
		return edge.To().ID(), "outbound", true
	}
	if edge.To().Equal(current) && inboundRelation(edge.Relation()) {
		return edge.From().ID(), "inbound", true
	}
	return "", "", false
}

func outboundRelation(relation domain.RelationType) bool {
	switch relation {
	case domain.RelationPartOf, domain.RelationDependsOn, domain.RelationImplements, domain.RelationProvides, domain.RelationUses,
		domain.RelationConstrains, domain.RelationGoverns, domain.RelationOperates:
		return true
	default:
		return false
	}
}

func inboundRelation(relation domain.RelationType) bool {
	switch relation {
	case domain.RelationPartOf, domain.RelationImplements, domain.RelationProvides:
		return true
	default:
		return false
	}
}

func scopedEntity(item queueItem) ScopedEntity {
	return ScopedEntity{
		ID: item.entity.ID(), Type: string(item.entity.Type()), Name: item.entity.Name(), Status: string(item.entity.Status()), OwnerRef: item.entity.OwnerRef(),
		Depth: item.depth, ParentID: item.parentID, ViaEdgeID: item.viaEdgeID, ViaRelation: string(item.viaRelation), Direction: item.direction,
	}
}

func appendLimitWarning(values []Warning, maxEntities int) []Warning {
	for _, value := range values {
		if value.Code == "entity_limit" {
			return values
		}
	}
	return append(values, Warning{Code: "entity_limit", Detail: fmt.Sprintf("scope truncated at %d engineering entities", maxEntities)})
}

func sortWarnings(values []Warning) {
	sort.Slice(values, func(i, j int) bool {
		left := values[i].Code + "\x00" + values[i].EntityID + "\x00" + values[i].EdgeID + "\x00" + values[i].BindingID + "\x00" + values[i].Detail
		right := values[j].Code + "\x00" + values[j].EntityID + "\x00" + values[j].EdgeID + "\x00" + values[j].BindingID + "\x00" + values[j].Detail
		return left < right
	})
}
