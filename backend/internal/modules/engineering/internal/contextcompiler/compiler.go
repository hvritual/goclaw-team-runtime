package contextcompiler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/scope"
)

const (
	DefaultMaxReferences      = 32
	HardMaxReferences         = 128
	DefaultMaxEstimatedTokens = 12000
	HardMaxEstimatedTokens    = 128000
	DefaultMaxRecentChanges   = 8
	HardMaxRecentChanges      = 64
)

var (
	ErrRepositoryRequired    = errors.New("context compiler repository is required")
	ErrScopeResolverRequired = errors.New("context compiler scope resolver is required")
	ErrInvalidInput          = errors.New("invalid context compiler input")
	ErrBudgetTooSmall        = errors.New("context compiler budget too small for required source pins")
)

type ScopeResolver interface {
	Resolve(context.Context, scope.Input) (scope.Result, error)
}

type Compiler struct {
	repository domain.Repository
	scope      ScopeResolver
	governed   contract.PublishedContextReferenceReader
	incidents  contract.IncidentContextReferenceReader
	now        func() time.Time
}

type candidate struct {
	kind            domain.ContextKind
	id              string
	revision        string
	checksum        string
	entityID        string
	source          string
	score           int
	estimatedTokens int
	required        bool
}

func New(repository domain.Repository, resolver ScopeResolver, governed contract.PublishedContextReferenceReader, incidents contract.IncidentContextReferenceReader, now func() time.Time) (*Compiler, error) {
	if repository == nil {
		return nil, ErrRepositoryRequired
	}
	if resolver == nil {
		return nil, ErrScopeResolverRequired
	}
	if now == nil {
		now = time.Now
	}
	return &Compiler{repository: repository, scope: resolver, governed: governed, incidents: incidents, now: now}, nil
}

func (c *Compiler) Compile(ctx context.Context, request contract.CompileContextRequest) (contract.CompileContextResult, error) {
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	packID := strings.TrimSpace(request.PackID)
	workRevision := strings.TrimSpace(request.WorkItemRevision)
	policy, err := normalizePolicy(request.Policy)
	if workspaceID == "" || packID == "" || workRevision == "" || err != nil {
		return contract.CompileContextResult{}, ErrInvalidInput
	}
	workItem, err := domain.NewNodeRef(domain.NodeKind(strings.TrimSpace(request.WorkItem.Kind)), strings.TrimSpace(request.WorkItem.ID))
	if err != nil {
		return contract.CompileContextResult{}, ErrInvalidInput
	}

	resolved, err := c.scope.Resolve(ctx, scope.Input{
		WorkspaceID: workspaceID,
		WorkItem:    workItem,
		Policy: scope.Policy{
			MaxDepth:         policy.MaxDepth,
			MaxEntities:      policy.MaxEntities,
			SourceStaleAfter: policy.SourceStaleAfter,
		},
	})
	if err != nil {
		return contract.CompileContextResult{}, err
	}

	entityIDs := make([]string, 0, len(resolved.Entities))
	depthByEntity := make(map[string]int, len(resolved.Entities))
	for _, entity := range resolved.Entities {
		entityIDs = append(entityIDs, entity.ID)
		depthByEntity[entity.ID] = entity.Depth
	}
	sort.Strings(entityIDs)

	warnings := scopeWarnings(resolved.Warnings)
	candidates, sourceWarnings := sourceCandidates(resolved, depthByEntity)
	warnings = append(warnings, sourceWarnings...)

	if c.governed != nil {
		values, readErr := c.governed.ListPublishedContextReferences(ctx, workspaceID, append([]string(nil), entityIDs...))
		if readErr != nil {
			return contract.CompileContextResult{}, fmt.Errorf("resolve governed context references: %w", readErr)
		}
		resolvedCandidates, candidateWarnings := externalCandidates(values, "governed", entityIDs, policy.KnowledgeStaleAfter, c.now().UTC())
		candidates = append(candidates, resolvedCandidates...)
		warnings = append(warnings, candidateWarnings...)
	}

	changeCandidates, err := c.recentChanges(ctx, workspaceID, entityIDs, policy.MaxRecentChanges)
	if err != nil {
		return contract.CompileContextResult{}, err
	}
	candidates = append(candidates, changeCandidates...)

	if c.incidents != nil {
		values, readErr := c.incidents.ListIncidentContextReferences(ctx, workspaceID, append([]string(nil), entityIDs...))
		if readErr != nil {
			return contract.CompileContextResult{}, fmt.Errorf("resolve incident context references: %w", readErr)
		}
		resolvedCandidates, candidateWarnings := externalCandidates(values, "incident", entityIDs, 0, c.now().UTC())
		candidates = append(candidates, resolvedCandidates...)
		warnings = append(warnings, candidateWarnings...)
	}

	selected, selectionWarnings, estimatedTokens, err := selectCandidates(candidates, policy)
	if err != nil {
		return contract.CompileContextResult{}, err
	}
	warnings = append(warnings, selectionWarnings...)
	refs := make([]domain.ContextReference, 0, len(selected))
	selections := make([]contract.ContextSelection, 0, len(selected))
	for _, item := range selected {
		ref, refErr := domain.NewContextReference(item.kind, item.id, item.revision, item.checksum)
		if refErr != nil {
			return contract.CompileContextResult{}, fmt.Errorf("build context reference %s/%s: %w", item.kind, item.id, refErr)
		}
		refs = append(refs, ref)
		selections = append(selections, contract.ContextSelection{
			Reference: contract.ContextReference{Kind: string(item.kind), ID: item.id, Revision: item.revision, Checksum: item.checksum},
			Source:    item.source, Score: item.score, EstimatedTokens: item.estimatedTokens,
		})
	}

	now := c.now().UTC()
	pack, err := domain.NewContextPack(packID, workspaceID, workItem, workRevision, entityIDs, refs, policy.Version, now)
	if err != nil {
		return contract.CompileContextResult{}, fmt.Errorf("build frozen context pack: %w", err)
	}
	if existing, getErr := c.repository.GetContextPack(ctx, workspaceID, packID); getErr == nil {
		if existing.Checksum() != pack.Checksum() {
			return contract.CompileContextResult{}, domain.ErrConflict
		}
		pack = existing
	} else if !errors.Is(getErr, domain.ErrNotFound) {
		return contract.CompileContextResult{}, fmt.Errorf("read existing context pack: %w", getErr)
	} else if putErr := c.repository.PutContextPack(ctx, pack); putErr != nil {
		return contract.CompileContextResult{}, fmt.Errorf("persist frozen context pack: %w", putErr)
	}

	sortWarnings(warnings)
	return contract.CompileContextResult{
		Pack:            toContractPack(pack),
		ScopeEntityIDs:  entityIDs,
		Selections:      selections,
		Warnings:        warnings,
		EstimatedTokens: estimatedTokens,
	}, nil
}

func normalizePolicy(value contract.ContextCompilePolicy) (contract.ContextCompilePolicy, error) {
	value.Version = strings.TrimSpace(value.Version)
	if value.Version == "" || value.MaxDepth < 0 || value.MaxEntities < 0 || value.SourceStaleAfter < 0 || value.KnowledgeStaleAfter < 0 || value.MaxReferences < 0 || value.MaxEstimatedTokens < 0 || value.MaxRecentChanges < 0 {
		return contract.ContextCompilePolicy{}, ErrInvalidInput
	}
	if value.MaxReferences == 0 {
		value.MaxReferences = DefaultMaxReferences
	}
	if value.MaxEstimatedTokens == 0 {
		value.MaxEstimatedTokens = DefaultMaxEstimatedTokens
	}
	if value.MaxRecentChanges == 0 {
		value.MaxRecentChanges = DefaultMaxRecentChanges
	}
	if value.MaxReferences > HardMaxReferences || value.MaxEstimatedTokens > HardMaxEstimatedTokens || value.MaxRecentChanges > HardMaxRecentChanges {
		return contract.ContextCompilePolicy{}, ErrInvalidInput
	}
	return value, nil
}

func sourceCandidates(resolved scope.Result, depthByEntity map[string]int) ([]candidate, []contract.ContextCompileWarning) {
	byEntity := make(map[string][]scope.SourceRef)
	for _, source := range resolved.Sources {
		byEntity[source.EntityID] = append(byEntity[source.EntityID], source)
	}
	var candidates []candidate
	var warnings []contract.ContextCompileWarning
	for _, entity := range resolved.Entities {
		values := byEntity[entity.ID]
		sort.Slice(values, func(i, j int) bool { return betterSource(values[i], values[j]) })
		var selected *scope.SourceRef
		for index := range values {
			value := values[index]
			if strings.TrimSpace(value.Revision) == "" || (value.Authority != string(domain.AuthorityAuthoritative) && value.Authority != string(domain.AuthorityObserved)) {
				continue
			}
			selected = &value
			break
		}
		if selected == nil {
			warnings = append(warnings, contract.ContextCompileWarning{Code: "source_unpinned", EntityID: entity.ID, Detail: "no pinned authoritative or observed source is available"})
			continue
		}
		checksum := sourceChecksum(*selected)
		candidates = append(candidates, candidate{
			kind: domain.ContextKindEngineeringEntity, id: entity.ID, revision: selected.Revision, checksum: checksum,
			entityID: entity.ID, source: "source_binding", score: 2000 - depthByEntity[entity.ID]*10, estimatedTokens: 32, required: true,
		})
	}
	return candidates, warnings
}

func betterSource(left, right scope.SourceRef) bool {
	if left.Stale != right.Stale {
		return !left.Stale
	}
	leftRank, rightRank := sourceAuthorityRank(left.Authority), sourceAuthorityRank(right.Authority)
	if leftRank != rightRank {
		return leftRank > rightRank
	}
	if !left.ObservedAt.Equal(right.ObservedAt) {
		return left.ObservedAt.After(right.ObservedAt)
	}
	return left.BindingID < right.BindingID
}

func sourceAuthorityRank(value string) int {
	switch value {
	case string(domain.AuthorityAuthoritative):
		return 2
	case string(domain.AuthorityObserved):
		return 1
	default:
		return 0
	}
}

func sourceChecksum(value scope.SourceRef) string {
	payload := struct {
		EntityID, BindingID, SourceType, Locator, Revision, Authority string
	}{value.EntityID, value.BindingID, value.SourceType, value.Locator, value.Revision, value.Authority}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func externalCandidates(values []contract.ContextReferenceCandidate, source string, scopedEntityIDs []string, staleAfter time.Duration, now time.Time) ([]candidate, []contract.ContextCompileWarning) {
	scoped := make(map[string]struct{}, len(scopedEntityIDs))
	for _, id := range scopedEntityIDs {
		scoped[id] = struct{}{}
	}
	var result []candidate
	var warnings []contract.ContextCompileWarning
	for _, value := range values {
		kind := domain.ContextKind(strings.TrimSpace(value.Kind))
		id, revision, checksum := strings.TrimSpace(value.ID), strings.TrimSpace(value.Revision), strings.TrimSpace(value.Checksum)
		if !allowedExternalKind(kind, source) || id == "" || revision == "" || checksum == "" || (!value.Global && !overlapsScope(value.EntityIDs, scoped)) {
			warnings = append(warnings, contract.ContextCompileWarning{Code: "reference_invalid", RefKind: string(kind), RefID: id, Detail: "reference is unversioned, unsupported, or outside resolved scope"})
			continue
		}
		stale := staleAfter > 0 && !value.UpdatedAt.IsZero() && value.UpdatedAt.Before(now.Add(-staleAfter))
		if stale {
			warnings = append(warnings, contract.ContextCompileWarning{Code: "reference_stale", RefKind: string(kind), RefID: id, Detail: "reference exceeds configured freshness window"})
		}
		tokens := value.EstimatedTokens
		if tokens <= 0 {
			tokens = defaultTokens(kind)
		}
		score := kindScore(kind) + value.Priority
		if stale {
			score -= 300
		}
		result = append(result, candidate{kind: kind, id: id, revision: revision, checksum: checksum, source: source, score: score, estimatedTokens: tokens})
	}
	return result, warnings
}

func allowedExternalKind(kind domain.ContextKind, source string) bool {
	if source == "incident" {
		return kind == domain.ContextKindIncident
	}
	switch kind {
	case domain.ContextKindArchitecture, domain.ContextKindADR, domain.ContextKindStandard, domain.ContextKindRunbook, domain.ContextKindKnowledge, domain.ContextKindRequirement:
		return true
	default:
		return false
	}
}

func overlapsScope(values []string, scoped map[string]struct{}) bool {
	for _, raw := range values {
		if _, ok := scoped[strings.TrimSpace(raw)]; ok {
			return true
		}
	}
	return false
}

func kindScore(kind domain.ContextKind) int {
	switch kind {
	case domain.ContextKindStandard:
		return 950
	case domain.ContextKindArchitecture:
		return 900
	case domain.ContextKindADR:
		return 875
	case domain.ContextKindRequirement:
		return 850
	case domain.ContextKindRunbook:
		return 800
	case domain.ContextKindIncident:
		return 775
	case domain.ContextKindChange:
		return 750
	case domain.ContextKindKnowledge:
		return 650
	default:
		return 500
	}
}

func defaultTokens(kind domain.ContextKind) int {
	switch kind {
	case domain.ContextKindStandard, domain.ContextKindADR, domain.ContextKindRequirement:
		return 384
	case domain.ContextKindArchitecture, domain.ContextKindRunbook, domain.ContextKindIncident:
		return 512
	case domain.ContextKindKnowledge:
		return 640
	case domain.ContextKindChange:
		return 192
	default:
		return 128
	}
}

func (c *Compiler) recentChanges(ctx context.Context, workspaceID string, entityIDs []string, limit int) ([]candidate, error) {
	byID := make(map[string]domain.Change)
	for _, entityID := range entityIDs {
		values, err := c.repository.ListChanges(ctx, workspaceID, entityID)
		if err != nil {
			return nil, fmt.Errorf("list recent changes for %s: %w", entityID, err)
		}
		for _, value := range values {
			if value.Status() == domain.ChangeStatusAccepted && value.AcceptedAt() != nil {
				byID[value.ID()] = value
			}
		}
	}
	values := make([]domain.Change, 0, len(byID))
	for _, value := range byID {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		left, right := *values[i].AcceptedAt(), *values[j].AcceptedAt()
		if !left.Equal(right) {
			return left.After(right)
		}
		return values[i].ID() < values[j].ID()
	})
	if len(values) > limit {
		values = values[:limit]
	}
	result := make([]candidate, 0, len(values))
	for index, value := range values {
		acceptedAt := value.AcceptedAt().UTC()
		result = append(result, candidate{
			kind: domain.ContextKindChange, id: value.ID(), revision: acceptedAt.Format(time.RFC3339Nano), checksum: changeChecksum(value),
			source: "accepted_change", score: kindScore(domain.ContextKindChange) - index, estimatedTokens: defaultTokens(domain.ContextKindChange),
		})
	}
	return result, nil
}

func changeChecksum(value domain.Change) string {
	type artifact struct{ Kind, Locator, Revision string }
	artifacts := make([]artifact, 0, len(value.Artifacts()))
	for _, item := range value.Artifacts() {
		artifacts = append(artifacts, artifact{item.Kind(), item.Locator(), item.Revision()})
	}
	sort.Slice(artifacts, func(i, j int) bool {
		left := artifacts[i].Kind + "\x00" + artifacts[i].Locator + "\x00" + artifacts[i].Revision
		right := artifacts[j].Kind + "\x00" + artifacts[j].Locator + "\x00" + artifacts[j].Revision
		return left < right
	})
	provenance := value.Provenance()
	payload := struct {
		ID, Summary, Status, SourceType, Locator, SourceRevision string
		Affected                                                 []string
		Artifacts                                                []artifact
		AcceptedAt                                               string
	}{
		ID: value.ID(), Summary: value.Summary(), Status: string(value.Status()), SourceType: provenance.SourceType(), Locator: provenance.Locator(), SourceRevision: provenance.Revision(),
		Affected: value.AffectedEntityIDs(), Artifacts: artifacts, AcceptedAt: value.AcceptedAt().UTC().Format(time.RFC3339Nano),
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func selectCandidates(values []candidate, policy contract.ContextCompilePolicy) ([]candidate, []contract.ContextCompileWarning, int, error) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].required != values[j].required {
			return values[i].required
		}
		if values[i].score != values[j].score {
			return values[i].score > values[j].score
		}
		left := string(values[i].kind) + "\x00" + values[i].id + "\x00" + values[i].revision + "\x00" + values[i].checksum
		right := string(values[j].kind) + "\x00" + values[j].id + "\x00" + values[j].revision + "\x00" + values[j].checksum
		return left < right
	})
	selected := make([]candidate, 0, min(len(values), policy.MaxReferences))
	seen := make(map[string]struct{}, len(values))
	var warnings []contract.ContextCompileWarning
	tokens := 0
	for _, value := range values {
		key := string(value.kind) + "\x00" + value.id
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if len(selected) >= policy.MaxReferences {
			if value.required {
				return nil, nil, 0, ErrBudgetTooSmall
			}
			warnings = append(warnings, contract.ContextCompileWarning{Code: "reference_limit", RefKind: string(value.kind), RefID: value.id, Detail: "reference omitted by max-reference policy"})
			continue
		}
		if value.estimatedTokens < 0 || tokens+value.estimatedTokens > policy.MaxEstimatedTokens {
			if value.required {
				return nil, nil, 0, ErrBudgetTooSmall
			}
			warnings = append(warnings, contract.ContextCompileWarning{Code: "token_budget", RefKind: string(value.kind), RefID: value.id, Detail: "reference omitted by estimated token budget"})
			continue
		}
		selected = append(selected, value)
		tokens += value.estimatedTokens
	}
	return selected, warnings, tokens, nil
}

func scopeWarnings(values []scope.Warning) []contract.ContextCompileWarning {
	result := make([]contract.ContextCompileWarning, 0, len(values))
	for _, value := range values {
		result = append(result, contract.ContextCompileWarning{Code: value.Code, EntityID: value.EntityID, Detail: value.Detail})
	}
	return result
}

func sortWarnings(values []contract.ContextCompileWarning) {
	sort.Slice(values, func(i, j int) bool {
		left := values[i].Code + "\x00" + values[i].EntityID + "\x00" + values[i].RefKind + "\x00" + values[i].RefID + "\x00" + values[i].Detail
		right := values[j].Code + "\x00" + values[j].EntityID + "\x00" + values[j].RefKind + "\x00" + values[j].RefID + "\x00" + values[j].Detail
		return left < right
	})
}

func toContractPack(value domain.ContextPack) contract.ContextPack {
	references := value.References()
	converted := make([]contract.ContextReference, len(references))
	for index, reference := range references {
		converted[index] = contract.ContextReference{Kind: string(reference.Kind()), ID: reference.ID(), Revision: reference.Revision(), Checksum: reference.Checksum()}
	}
	work := value.WorkItem()
	return contract.ContextPack{
		ID: value.ID(), WorkspaceID: value.WorkspaceID(), WorkItem: contract.NodeRef{Kind: string(work.Kind()), ID: work.ID()}, WorkItemRevision: value.WorkItemRevision(),
		TargetEntityIDs: value.TargetEntityIDs(), References: converted, PolicyVersion: value.PolicyVersion(), Checksum: value.Checksum(), CreatedAt: value.CreatedAt(),
	}
}
