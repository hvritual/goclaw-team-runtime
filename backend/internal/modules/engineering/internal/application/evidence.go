package application

import (
	"context"
	"strings"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Service) RecordEvidence(ctx context.Context, actor contract.Actor, workspaceID string, request contract.RecordEvidenceRequest) (contract.Evidence, error) {
	if err := s.authorize(ctx, actor, workspaceID, true); err != nil {
		return contract.Evidence{}, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	subject, err := domain.NewNodeRef(domain.NodeKind(request.Subject.Kind), request.Subject.ID)
	if err != nil {
		return contract.Evidence{}, invalid(err)
	}
	if err := s.requireEvidenceSubject(ctx, workspaceID, subject); err != nil {
		return contract.Evidence{}, err
	}
	source, err := domain.NewEvidenceSource(request.Source.SourceType, request.Source.Locator, request.Source.Revision, request.Source.Digest, request.Source.ObservedAt)
	if err != nil {
		return contract.Evidence{}, invalid(err)
	}
	value, err := domain.NewEvidenceEnvelope(
		request.ID,
		workspaceID,
		domain.EvidenceKind(request.Kind),
		subject,
		source,
		request.ProducerID,
		request.ArtifactURI,
		request.ArtifactDigest,
		s.now().UTC(),
	)
	if err != nil {
		return contract.Evidence{}, invalid(err)
	}
	if err := s.repository.PutEvidence(ctx, value); err != nil {
		return contract.Evidence{}, repositoryError(err)
	}
	persisted, err := s.repository.GetEvidence(ctx, workspaceID, value.ID())
	if err != nil {
		return contract.Evidence{}, repositoryError(err)
	}
	return toEvidence(persisted), nil
}

func (s *Service) GetEvidence(ctx context.Context, actor contract.Actor, workspaceID, id string) (contract.Evidence, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return contract.Evidence{}, err
	}
	value, err := s.repository.GetEvidence(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(id))
	if err != nil {
		return contract.Evidence{}, repositoryError(err)
	}
	return toEvidence(value), nil
}

func (s *Service) ListEvidence(ctx context.Context, actor contract.Actor, workspaceID string, subject *contract.NodeRef) ([]contract.Evidence, error) {
	if err := s.authorize(ctx, actor, workspaceID, false); err != nil {
		return nil, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	var domainSubject *domain.NodeRef
	if subject != nil {
		value, err := domain.NewNodeRef(domain.NodeKind(subject.Kind), subject.ID)
		if err != nil {
			return nil, invalid(err)
		}
		if err := s.requireEvidenceSubject(ctx, workspaceID, value); err != nil {
			return nil, err
		}
		domainSubject = &value
	}
	values, err := s.repository.ListEvidence(ctx, workspaceID, domainSubject)
	if err != nil {
		return nil, repositoryError(err)
	}
	result := make([]contract.Evidence, len(values))
	for index, value := range values {
		result[index] = toEvidence(value)
	}
	return result, nil
}

func (s *Service) requireEvidenceSubject(ctx context.Context, workspaceID string, subject domain.NodeRef) error {
	switch subject.Kind() {
	case domain.NodeKindEngineeringEntity:
		_, err := s.repository.GetEntity(ctx, workspaceID, subject.ID())
		return repositoryError(err)
	case domain.NodeKindChange:
		_, err := s.repository.GetChange(ctx, workspaceID, subject.ID())
		return repositoryError(err)
	default:
		return contract.ErrInvalidArgument
	}
}

func toEvidence(value domain.EvidenceEnvelope) contract.Evidence {
	source := value.Source()
	return contract.Evidence{
		ID:          value.ID(),
		WorkspaceID: value.WorkspaceID(),
		Kind:        string(value.Kind()),
		Subject:     toNodeRef(value.Subject()),
		Source: contract.EvidenceSource{
			SourceType: source.SourceType(),
			Locator:    source.Locator(),
			Revision:   source.Revision(),
			Digest:     source.Digest(),
			ObservedAt: source.ObservedAt(),
		},
		ProducerID:      value.ProducerID(),
		ArtifactURI:     value.ArtifactURI(),
		ArtifactDigest:  value.ArtifactDigest(),
		CapturedAt:      value.CapturedAt(),
		ContentChecksum: value.ContentChecksum(),
	}
}
