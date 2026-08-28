package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Service) PutWorkLink(ctx context.Context, request contract.PutWorkLinkRequest) (contract.WorkLink, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	workID := strings.TrimSpace(request.WorkID)
	entityID := strings.TrimSpace(request.EntityID)
	nodeKind, relation, err := workLinkSpec(request.WorkKind)
	if err != nil || workspaceID == "" || workID == "" || entityID == "" {
		return contract.WorkLink{}, contract.ErrInvalidArgument
	}
	entity, err := s.repository.GetEntity(ctx, workspaceID, entityID)
	if err != nil {
		return contract.WorkLink{}, repositoryError(err)
	}
	if entity.Status() == domain.EntityStatusArchived {
		return contract.WorkLink{}, contract.ErrConflict
	}
	from, err := domain.NewNodeRef(nodeKind, workID)
	if err != nil {
		return contract.WorkLink{}, invalid(err)
	}
	to, err := domain.NewNodeRef(domain.NodeKindEngineeringEntity, entityID)
	if err != nil {
		return contract.WorkLink{}, invalid(err)
	}
	id := workLinkID(workspaceID, request.WorkKind, workID, entityID)
	if existing, getErr := s.repository.GetThreadEdge(ctx, workspaceID, id); getErr == nil {
		if existing.From().Equal(from) && existing.To().Equal(to) && existing.Relation() == relation {
			return toWorkLink(request.WorkKind, existing), nil
		}
		return contract.WorkLink{}, contract.ErrConflict
	} else if !errors.Is(getErr, domain.ErrNotFound) {
		return contract.WorkLink{}, repositoryError(getErr)
	}
	provenance, err := domain.NewProvenance(
		"workspace",
		fmt.Sprintf("workspace://%s/work/%s/%s", workspaceID, request.WorkKind, workID),
		"",
		s.now().UTC(),
	)
	if err != nil {
		return contract.WorkLink{}, contract.ErrUnavailable
	}
	edge, err := domain.NewThreadEdge(id, workspaceID, from, relation, to, domain.AuthorityAuthoritative, provenance)
	if err != nil {
		return contract.WorkLink{}, invalid(err)
	}
	if err := s.repository.PutThreadEdge(ctx, edge); err != nil {
		return contract.WorkLink{}, repositoryError(err)
	}
	return toWorkLink(request.WorkKind, edge), nil
}

func (s *Service) ListWorkLinks(ctx context.Context, request contract.ListWorkLinksRequest) ([]contract.WorkLink, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	workID := strings.TrimSpace(request.WorkID)
	nodeKind, relation, err := workLinkSpec(request.WorkKind)
	if err != nil || workspaceID == "" || workID == "" {
		return nil, contract.ErrInvalidArgument
	}
	node, err := domain.NewNodeRef(nodeKind, workID)
	if err != nil {
		return nil, invalid(err)
	}
	edges, err := s.repository.ListThreadEdges(ctx, workspaceID, node)
	if err != nil {
		return nil, repositoryError(err)
	}
	result := make([]contract.WorkLink, 0, len(edges))
	for _, edge := range edges {
		if !edge.From().Equal(node) || edge.Relation() != relation || edge.To().Kind() != domain.NodeKindEngineeringEntity || edge.Provenance().SourceType() != "workspace" {
			continue
		}
		result = append(result, toWorkLink(request.WorkKind, edge))
	}
	return result, nil
}

func (s *Service) DeleteWorkLink(ctx context.Context, request contract.DeleteWorkLinkRequest) error {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	workID := strings.TrimSpace(request.WorkID)
	entityID := strings.TrimSpace(request.EntityID)
	nodeKind, relation, err := workLinkSpec(request.WorkKind)
	if err != nil || workspaceID == "" || workID == "" || entityID == "" {
		return contract.ErrInvalidArgument
	}
	id := workLinkID(workspaceID, request.WorkKind, workID, entityID)
	edge, err := s.repository.GetThreadEdge(ctx, workspaceID, id)
	if err != nil {
		return repositoryError(err)
	}
	from, err := domain.NewNodeRef(nodeKind, workID)
	if err != nil {
		return invalid(err)
	}
	to, err := domain.NewNodeRef(domain.NodeKindEngineeringEntity, entityID)
	if err != nil {
		return invalid(err)
	}
	if !edge.From().Equal(from) || !edge.To().Equal(to) || edge.Relation() != relation || edge.Provenance().SourceType() != "workspace" {
		return contract.ErrConflict
	}
	deleter, ok := s.repository.(domain.ThreadEdgeDeleteRepository)
	if !ok {
		return contract.ErrUnavailable
	}
	if err := deleter.DeleteThreadEdge(ctx, workspaceID, id); err != nil {
		return repositoryError(err)
	}
	return nil
}

func workLinkSpec(kind contract.WorkLinkKind) (domain.NodeKind, domain.RelationType, error) {
	switch kind {
	case contract.WorkLinkProject:
		return domain.NodeKindProject, domain.RelationChanges, nil
	case contract.WorkLinkRequirement:
		return domain.NodeKindRequirement, domain.RelationAffects, nil
	case contract.WorkLinkTask:
		return domain.NodeKindTask, domain.RelationAffects, nil
	default:
		return "", "", contract.ErrInvalidArgument
	}
}

func workLinkID(workspaceID string, kind contract.WorkLinkKind, workID, entityID string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{workspaceID, string(kind), workID, entityID}, "\x00")))
	return "worklink-" + hex.EncodeToString(sum[:16])
}

func toWorkLink(kind contract.WorkLinkKind, edge domain.ThreadEdge) contract.WorkLink {
	return contract.WorkLink{
		ID:          edge.ID(),
		WorkspaceID: edge.WorkspaceID(),
		WorkKind:    kind,
		WorkID:      edge.From().ID(),
		EntityID:    edge.To().ID(),
		Relation:    string(edge.Relation()),
		Authority:   string(edge.Authority()),
		Provenance:  toProvenance(edge.Provenance()),
	}
}
