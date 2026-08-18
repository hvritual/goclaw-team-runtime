package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type CreateKnowledgeCandidateCommand struct {
	Candidate                            contract.KnowledgeCandidate
	IdempotencyKey, RequestHash, AuditID string
}

type CreatedKnowledgeCandidate struct {
	Candidate contract.KnowledgeCandidate
	Replayed  bool
}

type ReviewKnowledgeCandidateCommand struct {
	WorkspaceID, CandidateID, Action, Rationale, ActorID, AuditID, PublicationID string
	ExpectedRevision                                                             int
	Emergency, AllowSelfReview                                                   bool
	OccurredAt                                                                   time.Time
	ValidateSources                                                              func(context.Context, string, []contract.KnowledgeSourceRef) error
}

type KnowledgeReviewRepository interface {
	CreateKnowledgeCandidate(context.Context, CreateKnowledgeCandidateCommand) (CreatedKnowledgeCandidate, error)
	ListKnowledgeCandidates(context.Context, string) ([]contract.KnowledgeCandidate, error)
	ReviewKnowledgeCandidate(context.Context, ReviewKnowledgeCandidateCommand) (contract.ReviewKnowledgeResponse, error)
}

type KnowledgeReviewUseCase struct {
	repository KnowledgeReviewRepository
	authorizer contract.WorkspaceAccessAuthorizer
	assets     contract.WorkspaceAssetReader
	newID      ProjectIDGenerator
	now        Clock
	signingKey []byte
	events     contract.WorkspaceEventPublisher
}

func NewKnowledgeReviewUseCase(repository KnowledgeReviewRepository, authorizer contract.WorkspaceAccessAuthorizer, assets contract.WorkspaceAssetReader, newID ProjectIDGenerator, now Clock, signingKey []byte, events contract.WorkspaceEventPublisher) (*KnowledgeReviewUseCase, error) {
	if repository == nil || authorizer == nil || assets == nil || newID == nil || now == nil || len(signingKey) < 32 {
		return nil, errors.New("Knowledge review dependencies are required")
	}
	return &KnowledgeReviewUseCase{repository: repository, authorizer: authorizer, assets: assets, newID: newID, now: now, signingKey: append([]byte(nil), signingKey...), events: events}, nil
}

func (s *KnowledgeReviewUseCase) ProposeKnowledge(ctx context.Context, request contract.ProposeKnowledgeRequest) (contract.KnowledgeCandidate, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	key := strings.TrimSpace(request.IdempotencyKey)
	if workspaceID == "" || key == "" || len(key) > 200 {
		return contract.KnowledgeCandidate{}, contract.ErrInvalidKnowledgeReview
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionKnowledgePropose); err != nil {
		return contract.KnowledgeCandidate{}, err
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" {
		return contract.KnowledgeCandidate{}, contract.ErrWorkspacePermissionDenied
	}
	candidate, err := normalizeKnowledgeProposal(request)
	if err != nil {
		return contract.KnowledgeCandidate{}, err
	}
	candidate.WorkspaceID, candidate.ProposedBy = workspaceID, actor.ID
	candidate.Status, candidate.Revision = "candidate", 1
	now := s.now().UTC()
	candidate.CreatedAt, candidate.UpdatedAt = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
	candidate.ID, err = s.newID(ctx)
	if err != nil {
		return contract.KnowledgeCandidate{}, err
	}
	auditID, err := s.newID(ctx)
	if err != nil {
		return contract.KnowledgeCandidate{}, err
	}
	for _, source := range candidate.SourceRefs {
		if source.AssetID != nil {
			belongs, checkErr := s.assets.AssetBelongsToWorkspace(ctx, workspaceID, *source.AssetID)
			if checkErr != nil {
				return contract.KnowledgeCandidate{}, checkErr
			}
			if !belongs {
				return contract.KnowledgeCandidate{}, contract.ErrAssetOutsideWorkspace
			}
		}
	}
	hashBody, _ := json.Marshal(struct {
		ProjectID, KnowledgeID       *string
		Kind, Title, Content, Reason string
		Sources                      []contract.KnowledgeSourceRef
	}{candidate.ProjectID, candidate.KnowledgeID, candidate.Kind, candidate.Title, candidate.Content, candidate.Reason, candidate.SourceRefs})
	sum := sha256.Sum256(hashBody)
	created, err := s.repository.CreateKnowledgeCandidate(ctx, CreateKnowledgeCandidateCommand{Candidate: candidate, IdempotencyKey: key, RequestHash: hex.EncodeToString(sum[:]), AuditID: auditID})
	if err == nil && !created.Replayed && s.events != nil {
		s.events.Publish(workspaceID, "knowledge:candidate_updated", map[string]any{"candidate": created.Candidate}, actor.ID, actor.Type)
	}
	return created.Candidate, err
}

func normalizeKnowledgeProposal(request contract.ProposeKnowledgeRequest) (contract.KnowledgeCandidate, error) {
	c := contract.KnowledgeCandidate{ProjectID: trimOptional(request.ProjectID), KnowledgeID: trimOptional(request.KnowledgeID), Kind: strings.TrimSpace(request.Kind), Title: strings.TrimSpace(request.Title), Content: strings.TrimSpace(request.Content), Reason: strings.TrimSpace(request.Reason)}
	validKinds := map[string]bool{"goal": true, "decision": true, "constraint": true, "requirement": true, "procedure": true, "lesson": true, "reference": true}
	if !validKinds[c.Kind] || c.Title == "" || c.Content == "" || c.Reason == "" || (c.ProjectID != nil && c.KnowledgeID != nil) {
		return contract.KnowledgeCandidate{}, contract.ErrInvalidKnowledgeReview
	}
	c.SourceRefs = make([]contract.KnowledgeSourceRef, len(request.SourceRefs))
	for i, source := range request.SourceRefs {
		source.Type, source.ID, source.Revision, source.Citation = strings.TrimSpace(source.Type), strings.TrimSpace(source.ID), strings.TrimSpace(source.Revision), strings.TrimSpace(source.Citation)
		source.AssetID, source.AssetVersionID = trimOptional(source.AssetID), trimOptional(source.AssetVersionID)
		if source.Type == "" || source.ID == "" || source.Revision == "" || source.Citation == "" || (source.AssetID == nil) != (source.AssetVersionID == nil) {
			return contract.KnowledgeCandidate{}, contract.ErrInvalidKnowledgeReview
		}
		c.SourceRefs[i] = source
	}
	return c, nil
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *KnowledgeReviewUseCase) ListKnowledgeCandidates(ctx context.Context, request contract.ListKnowledgeCandidatesRequest) (contract.KnowledgeCandidateListResponse, error) {
	w := strings.TrimSpace(request.WorkspaceID)
	if w == "" || request.Limit < 0 || request.Limit > 100 {
		return contract.KnowledgeCandidateListResponse{}, contract.ErrInvalidKnowledgeReview
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, contract.PermissionKnowledgeReview); err != nil {
		return contract.KnowledgeCandidateListResponse{}, err
	}
	limit := request.Limit
	if limit == 0 {
		limit = 50
	}
	values, err := s.repository.ListKnowledgeCandidates(ctx, w)
	if err != nil {
		return contract.KnowledgeCandidateListResponse{}, err
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].UpdatedAt == values[j].UpdatedAt {
			return values[i].ID < values[j].ID
		}
		return values[i].UpdatedAt > values[j].UpdatedAt
	})
	start := 0
	if strings.TrimSpace(request.Cursor) != "" {
		id, decodeErr := s.decodeCandidateCursor(request.Cursor, w)
		if decodeErr != nil {
			return contract.KnowledgeCandidateListResponse{}, contract.ErrInvalidKnowledgeReview
		}
		found := false
		for i := range values {
			if values[i].ID == id {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return contract.KnowledgeCandidateListResponse{}, contract.ErrInvalidKnowledgeReview
		}
	}
	total := len(values)
	end := start + limit
	if end > total {
		end = total
	}
	page := append(make([]contract.KnowledgeCandidate, 0, end-start), values[start:end]...)
	var next *string
	if end < total {
		encoded, _ := s.encodeCandidateCursor(w, values[end-1].ID)
		next = &encoded
	}
	return contract.KnowledgeCandidateListResponse{Candidates: page, Total: total, NextCursor: next}, nil
}

func (s *KnowledgeReviewUseCase) ReviewKnowledge(ctx context.Context, request contract.ReviewKnowledgeRequest) (contract.ReviewKnowledgeResponse, error) {
	w, id := strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.CandidateID)
	request.Action = strings.TrimSpace(request.Action)
	request.Rationale = strings.TrimSpace(request.Rationale)
	valid := map[string]bool{"approve": true, "reject": true, "quarantine": true, "return": true, "publish": true, "supersede": true, "invalidate": true}
	if w == "" || id == "" || !valid[request.Action] || request.ExpectedRevision <= 0 || request.Rationale == "" {
		return contract.ReviewKnowledgeResponse{}, contract.ErrInvalidKnowledgeReview
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, w, contract.PermissionKnowledgeReview); err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" {
		return contract.ReviewKnowledgeResponse{}, contract.ErrWorkspacePermissionDenied
	}
	allowSelf := false
	if request.Emergency {
		if len([]rune(request.Rationale)) < 12 {
			return contract.ReviewKnowledgeResponse{}, contract.ErrInvalidKnowledgeReview
		}
		if err := s.authorizer.AuthorizeWorkspace(ctx, w, contract.PermissionKnowledgeSelfReviewOverride); err != nil {
			return contract.ReviewKnowledgeResponse{}, contract.ErrKnowledgeSelfReview
		}
		allowSelf = true
	}
	auditID, err := s.newID(ctx)
	if err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	publicationID, err := s.newID(ctx)
	if err != nil {
		return contract.ReviewKnowledgeResponse{}, err
	}
	result, err := s.repository.ReviewKnowledgeCandidate(ctx, ReviewKnowledgeCandidateCommand{WorkspaceID: w, CandidateID: id, Action: request.Action, Rationale: request.Rationale, ActorID: actor.ID, AuditID: auditID, PublicationID: publicationID, ExpectedRevision: request.ExpectedRevision, Emergency: request.Emergency, AllowSelfReview: allowSelf, OccurredAt: s.now().UTC(), ValidateSources: s.validatePublicationSources})
	if err == nil && s.events != nil {
		payload := map[string]any{"candidate": result.Candidate}
		if result.Entry != nil {
			payload["entry"] = result.Entry
		}
		s.events.Publish(w, "knowledge:candidate_updated", payload, actor.ID, actor.Type)
	}
	return result, err
}

func (s *KnowledgeReviewUseCase) validatePublicationSources(ctx context.Context, workspaceID string, sources []contract.KnowledgeSourceRef) error {
	if len(sources) == 0 {
		return contract.ErrInvalidKnowledgeReview
	}
	for _, source := range sources {
		if source.AssetID == nil {
			continue
		}
		belongs, err := s.assets.AssetBelongsToWorkspace(ctx, workspaceID, *source.AssetID)
		if err != nil {
			return err
		}
		if !belongs {
			return contract.ErrAssetOutsideWorkspace
		}
	}
	return nil
}

func (s *KnowledgeReviewUseCase) encodeCandidateCursor(workspaceID, id string) (string, error) {
	body, _ := json.Marshal([]string{workspaceID, id})
	mac := hmac.New(sha256.New, s.signingKey)
	_, _ = mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(append(body, mac.Sum(nil)...)), nil
}
func (s *KnowledgeReviewUseCase) decodeCandidateCursor(raw, workspaceID string) (string, error) {
	signed, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil || len(signed) <= sha256.Size {
		return "", contract.ErrInvalidKnowledgeReview
	}
	body, sig := signed[:len(signed)-sha256.Size], signed[len(signed)-sha256.Size:]
	mac := hmac.New(sha256.New, s.signingKey)
	_, _ = mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", contract.ErrInvalidKnowledgeReview
	}
	var v []string
	if json.Unmarshal(body, &v) != nil || len(v) != 2 || v[0] != workspaceID {
		return "", contract.ErrInvalidKnowledgeReview
	}
	return v[1], nil
}

var _ contract.KnowledgeReviewService = (*KnowledgeReviewUseCase)(nil)
