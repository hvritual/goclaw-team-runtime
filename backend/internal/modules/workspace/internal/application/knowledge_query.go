package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	knowledgeDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/knowledge"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type KnowledgeQueryRepository interface {
	ListGovernedKnowledge(context.Context, string) ([]knowledgeDomain.GovernedEntry, error)
}

type KnowledgeQueryUseCase struct {
	repository  KnowledgeQueryRepository
	authorizer  contract.WorkspaceAccessAuthorizer
	memberships contract.WorkspaceMembershipReader
	signingKey  []byte
	now         func() time.Time
}

func NewKnowledgeQueryUseCase(repository KnowledgeQueryRepository, authorizer contract.WorkspaceAccessAuthorizer, memberships contract.WorkspaceMembershipReader, signingKey []byte, now func() time.Time) (*KnowledgeQueryUseCase, error) {
	if repository == nil || authorizer == nil || memberships == nil || len(signingKey) < 32 || now == nil {
		return nil, errors.New("Knowledge query dependencies are required")
	}
	return &KnowledgeQueryUseCase{repository: repository, authorizer: authorizer, memberships: memberships, signingKey: append([]byte(nil), signingKey...), now: now}, nil
}

type knowledgeQuerySpec struct {
	WorkspaceID, Query, SourceType, SourceID, SourceRevision, Applicability, ProjectID string
	Statuses, Kinds                                                                    []string
	Revision, Limit                                                                    int
}

type rankedKnowledge struct {
	entry    knowledgeDomain.GovernedEntry
	revision knowledgeDomain.Revision
	rank     int
	matched  string
}

func (s *KnowledgeQueryUseCase) QueryKnowledge(ctx context.Context, request contract.QueryKnowledgeRequest) (contract.QueryKnowledgeResponse, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if workspaceID == "" || request.Revision < 0 || request.Limit < 0 || request.Limit > 100 {
		return contract.QueryKnowledgeResponse{}, contract.ErrInvalidKnowledgeQuery
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionKnowledgeQuery); err != nil {
		return contract.QueryKnowledgeResponse{}, err
	}
	role, err := s.actorRole(ctx, workspaceID)
	if err != nil {
		return contract.QueryKnowledgeResponse{}, err
	}
	spec, err := normalizeKnowledgeQuery(request, role)
	if err != nil {
		return contract.QueryKnowledgeResponse{}, err
	}
	fingerprint := knowledgeQueryFingerprint(spec)
	var after *knowledgeCursor
	if strings.TrimSpace(request.Cursor) != "" {
		value, err := s.decodeCursor(request.Cursor, fingerprint)
		if err != nil {
			return contract.QueryKnowledgeResponse{}, contract.ErrInvalidKnowledgeQuery
		}
		after = &value
	}
	values, err := s.repository.ListGovernedKnowledge(ctx, workspaceID)
	if err != nil {
		return contract.QueryKnowledgeResponse{}, fmt.Errorf("list governed Knowledge: %w", err)
	}
	ranked := make([]rankedKnowledge, 0, len(values))
	for _, entry := range values {
		if ctx.Err() != nil {
			return contract.QueryKnowledgeResponse{}, ctx.Err()
		}
		selected, ok := selectKnowledgeRevision(entry, spec.Revision)
		if !ok || !containsString(spec.Statuses, entry.Status) || (len(spec.Kinds) > 0 && !containsString(spec.Kinds, entry.Kind)) || !knowledgeApplicabilityMatches(entry, spec) || !knowledgeSourceMatches(selected, spec) {
			continue
		}
		rank, matched, ok := knowledgeTextRank(selected, spec.Query)
		if !ok {
			continue
		}
		ranked = append(ranked, rankedKnowledge{entry: entry, revision: selected, rank: rank, matched: matched})
	}
	sort.Slice(ranked, func(i, j int) bool { return knowledgeLess(ranked[i], ranked[j]) })
	total := len(ranked)
	start := 0
	if after != nil {
		for start < len(ranked) && !knowledgeAfter(ranked[start], *after) {
			start++
		}
	}
	end := start + spec.Limit
	if end > len(ranked) {
		end = len(ranked)
	}
	page := ranked[start:end]
	response := contract.QueryKnowledgeResponse{Entries: make([]contract.GovernedKnowledgeEntry, len(page)), Total: total}
	for index, value := range page {
		response.Entries[index] = projectGovernedKnowledge(value.entry, value.revision, false, value.matched)
	}
	if end < len(ranked) && len(page) > 0 {
		last := page[len(page)-1]
		cursor, err := s.encodeCursor(knowledgeCursor{Fingerprint: fingerprint, Rank: last.rank, UpdatedUnixNano: last.entry.UpdatedAt.UnixNano(), ID: last.entry.ID, ExpiresUnix: s.now().Add(15 * time.Minute).Unix()})
		if err != nil {
			return contract.QueryKnowledgeResponse{}, err
		}
		response.NextCursor = &cursor
	}
	return response, nil
}

func (s *KnowledgeQueryUseCase) GetGovernedKnowledge(ctx context.Context, workspaceID, id string) (contract.GovernedKnowledgeEntry, error) {
	workspaceID, id = strings.TrimSpace(workspaceID), strings.TrimSpace(id)
	if workspaceID == "" || id == "" {
		return contract.GovernedKnowledgeEntry{}, contract.ErrInvalidKnowledgeQuery
	}
	if err := s.authorizer.AuthorizeWorkspace(ctx, workspaceID, contract.PermissionKnowledgeQuery); err != nil {
		return contract.GovernedKnowledgeEntry{}, err
	}
	role, err := s.actorRole(ctx, workspaceID)
	if err != nil {
		return contract.GovernedKnowledgeEntry{}, err
	}
	values, err := s.repository.ListGovernedKnowledge(ctx, workspaceID)
	if err != nil {
		return contract.GovernedKnowledgeEntry{}, fmt.Errorf("get governed Knowledge: %w", err)
	}
	for _, entry := range values {
		if entry.ID != id {
			continue
		}
		if entry.Status == "quarantined" && role != "owner" && role != "admin" {
			break
		}
		if entry.Status != "published" && entry.Status != "superseded" && entry.Status != "quarantined" {
			break
		}
		selected, ok := selectKnowledgeRevision(entry, 0)
		if !ok {
			break
		}
		return projectGovernedKnowledge(entry, selected, true, "detail"), nil
	}
	return contract.GovernedKnowledgeEntry{}, contract.ErrKnowledgeQueryHidden
}

func (s *KnowledgeQueryUseCase) actorRole(ctx context.Context, workspaceID string) (string, error) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || actor.Type != "member" {
		return "", contract.ErrWorkspacePermissionDenied
	}
	membership, found, err := s.memberships.FindForUserAndWorkspace(ctx, actor.ID, workspaceID)
	if err != nil {
		return "", err
	}
	if !found {
		membership, found, err = s.memberships.FindByMemberAndWorkspace(ctx, actor.ID, workspaceID)
	}
	if err != nil {
		return "", err
	}
	if !found {
		return "", contract.ErrWorkspacePermissionDenied
	}
	return membership.Role, nil
}

func normalizeKnowledgeQuery(request contract.QueryKnowledgeRequest, role string) (knowledgeQuerySpec, error) {
	spec := knowledgeQuerySpec{WorkspaceID: strings.TrimSpace(request.WorkspaceID), Query: normalizeKnowledgeText(request.Query), SourceType: strings.TrimSpace(request.SourceType), SourceID: strings.TrimSpace(request.SourceID), SourceRevision: strings.TrimSpace(request.SourceRevision), Applicability: strings.TrimSpace(request.Applicability), ProjectID: strings.TrimSpace(request.ProjectID), Revision: request.Revision, Limit: request.Limit}
	if spec.Limit == 0 {
		spec.Limit = 20
	}
	spec.Statuses = canonicalValues(request.Statuses)
	if len(request.Statuses) > 0 && len(spec.Statuses) == 0 {
		return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
	}
	if len(spec.Statuses) == 0 {
		spec.Statuses = []string{"published"}
	}
	spec.Kinds = canonicalValues(request.Kinds)
	if len(request.Kinds) > 0 && len(spec.Kinds) == 0 {
		return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
	}
	for _, status := range spec.Statuses {
		if status != "published" && status != "superseded" && status != "quarantined" {
			return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
		}
		if status == "quarantined" && role != "owner" && role != "admin" {
			return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
		}
	}
	validKinds := map[string]bool{"goal": true, "decision": true, "constraint": true, "requirement": true, "procedure": true, "lesson": true, "reference": true}
	for _, kind := range spec.Kinds {
		if !validKinds[kind] {
			return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
		}
	}
	if spec.SourceID != "" && spec.SourceType == "" || spec.SourceRevision != "" && (spec.SourceType == "" || spec.SourceID == "") {
		return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
	}
	if spec.Applicability == "" {
		if spec.ProjectID == "" {
			spec.Applicability = "workspace"
		} else {
			spec.Applicability = "project"
		}
	}
	if spec.Applicability != "workspace" && spec.Applicability != "project" {
		return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
	}
	if spec.Applicability == "workspace" && spec.ProjectID != "" || spec.Applicability == "project" && spec.ProjectID == "" {
		return knowledgeQuerySpec{}, contract.ErrInvalidKnowledgeQuery
	}
	return spec, nil
}

func canonicalValues(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				seen[part] = true
			}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

var knowledgeFold = cases.Fold()

func normalizeKnowledgeText(value string) string {
	value = knowledgeFold.String(norm.NFC.String(value))
	var builder strings.Builder
	space := true
	for _, r := range value {
		if unicode.IsSpace(r) {
			if !space {
				builder.WriteByte(' ')
				space = true
			}
			continue
		}
		builder.WriteRune(r)
		space = false
	}
	return strings.TrimSpace(builder.String())
}

func selectKnowledgeRevision(entry knowledgeDomain.GovernedEntry, number int) (knowledgeDomain.Revision, bool) {
	if number == 0 {
		number = entry.CurrentRevision
	}
	for _, revision := range entry.Revisions {
		if revision.Number == number {
			return revision, true
		}
	}
	return knowledgeDomain.Revision{}, false
}

func knowledgeApplicabilityMatches(entry knowledgeDomain.GovernedEntry, spec knowledgeQuerySpec) bool {
	if spec.Applicability == "workspace" {
		return entry.ProjectID == nil
	}
	return entry.ProjectID != nil && *entry.ProjectID == spec.ProjectID
}

func knowledgeSourceMatches(revision knowledgeDomain.Revision, spec knowledgeQuerySpec) bool {
	if spec.SourceType == "" {
		return true
	}
	for _, source := range revision.SourceRefs {
		if source.Type == spec.SourceType && (spec.SourceID == "" || source.ID == spec.SourceID) && (spec.SourceRevision == "" || source.Revision == spec.SourceRevision) {
			return true
		}
	}
	return false
}

func knowledgeTextRank(revision knowledgeDomain.Revision, query string) (int, string, bool) {
	if query == "" {
		return 0, "recent", true
	}
	title, content := normalizeKnowledgeText(revision.Title), normalizeKnowledgeText(revision.Content)
	if title == query {
		return 0, "title_exact", true
	}
	if strings.HasPrefix(title, query) {
		return 1, "title_prefix", true
	}
	if strings.Contains(title, query) {
		return 2, "title", true
	}
	if strings.Contains(content, query) {
		return 3, "content", true
	}
	for _, source := range revision.SourceRefs {
		if strings.Contains(normalizeKnowledgeText(source.Citation), query) {
			return 4, "source", true
		}
	}
	return 0, "", false
}

func knowledgeLess(a, b rankedKnowledge) bool {
	if a.rank != b.rank {
		return a.rank < b.rank
	}
	if !a.entry.UpdatedAt.Equal(b.entry.UpdatedAt) {
		return a.entry.UpdatedAt.After(b.entry.UpdatedAt)
	}
	return a.entry.ID < b.entry.ID
}

func knowledgeAfter(value rankedKnowledge, cursor knowledgeCursor) bool {
	if value.rank != cursor.Rank {
		return value.rank > cursor.Rank
	}
	updated := value.entry.UpdatedAt.UnixNano()
	if updated != cursor.UpdatedUnixNano {
		return updated < cursor.UpdatedUnixNano
	}
	return value.entry.ID > cursor.ID
}

func projectGovernedKnowledge(entry knowledgeDomain.GovernedEntry, selected knowledgeDomain.Revision, detail bool, matched string) contract.GovernedKnowledgeEntry {
	result := contract.GovernedKnowledgeEntry{ID: entry.ID, WorkspaceID: entry.WorkspaceID, ProjectID: entry.ProjectID, CandidateID: entry.CandidateID, Kind: entry.Kind, Status: entry.Status, CurrentRevision: entry.CurrentRevision, Revision: projectKnowledgeRevision(selected), MatchedBy: matched, CreatedAt: entry.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: entry.UpdatedAt.Format(time.RFC3339Nano)}
	if len(selected.SourceRefs) > 0 {
		result.Citation = selected.SourceRefs[0].Citation
	}
	if detail {
		result.Revisions = make([]contract.KnowledgeRevision, len(entry.Revisions))
		for i, revision := range entry.Revisions {
			result.Revisions[i] = projectKnowledgeRevision(revision)
		}
	}
	return result
}

func projectKnowledgeRevision(revision knowledgeDomain.Revision) contract.KnowledgeRevision {
	result := contract.KnowledgeRevision{Number: revision.Number, SupersedesRevision: revision.SupersedesRevision, Title: revision.Title, Content: revision.Content, CreatedBy: revision.CreatedBy, CreatedAt: revision.CreatedAt.Format(time.RFC3339Nano), SourceRefs: make([]contract.KnowledgeSourceRef, len(revision.SourceRefs))}
	for i, source := range revision.SourceRefs {
		result.SourceRefs[i] = contract.KnowledgeSourceRef{Type: source.Type, ID: source.ID, Revision: source.Revision, Citation: source.Citation, AssetID: source.AssetID, AssetVersionID: source.AssetVersionID}
	}
	return result
}

func containsString(values []string, wanted string) bool {
	index := sort.SearchStrings(values, wanted)
	return index < len(values) && values[index] == wanted
}

func knowledgeQueryFingerprint(spec knowledgeQuerySpec) string {
	body, _ := json.Marshal(spec)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

type knowledgeCursor struct {
	Fingerprint     string `json:"f"`
	Rank            int    `json:"r"`
	UpdatedUnixNano int64  `json:"u"`
	ID              string `json:"i"`
	ExpiresUnix     int64  `json:"e"`
}

func (s *KnowledgeQueryUseCase) encodeCursor(value knowledgeCursor) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.signingKey)
	_, _ = mac.Write(body)
	signed := append(body, mac.Sum(nil)...)
	return base64.RawURLEncoding.EncodeToString(signed), nil
}

func (s *KnowledgeQueryUseCase) decodeCursor(raw, fingerprint string) (knowledgeCursor, error) {
	signed, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil || len(signed) <= sha256.Size {
		return knowledgeCursor{}, contract.ErrInvalidKnowledgeQuery
	}
	body, signature := signed[:len(signed)-sha256.Size], signed[len(signed)-sha256.Size:]
	mac := hmac.New(sha256.New, s.signingKey)
	_, _ = mac.Write(body)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return knowledgeCursor{}, contract.ErrInvalidKnowledgeQuery
	}
	var value knowledgeCursor
	if json.Unmarshal(body, &value) != nil || value.Fingerprint != fingerprint || value.ID == "" || value.ExpiresUnix <= s.now().Unix() {
		return knowledgeCursor{}, contract.ErrInvalidKnowledgeQuery
	}
	return value, nil
}

var _ contract.KnowledgeQueryService = (*KnowledgeQueryUseCase)(nil)
