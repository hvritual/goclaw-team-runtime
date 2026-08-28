package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	"github.com/hvritual/workspace/internal/modules/engineering/internal/manifest"
	githubsource "github.com/hvritual/workspace/internal/modules/engineering/internal/source/github"
)

var (
	ErrWorkspaceRequired       = errors.New("engineering reconciliation workspace is required")
	ErrRepositoryRequired      = errors.New("engineering reconciliation repository is required")
	ErrSourceRequired          = errors.New("engineering reconciliation source is required")
	ErrSourceSnapshotInvalid   = errors.New("invalid engineering source snapshot")
	ErrSourceOwnershipConflict = errors.New("engineering source ownership conflict")
	ErrCanonicalEntityConflict = errors.New("engineering canonical entity conflicts with source manifest")
)

const (
	manifestEdgeSourceType = "github_manifest"
	bindingSourceType      = "github"
)

type Source interface {
	GetRepository(context.Context, string) (githubsource.Repository, error)
	ResolveCommit(context.Context, string, string) (githubsource.Commit, error)
	ReadEngineeringManifestAtCommit(context.Context, string, string) (githubsource.ManifestBlob, error)
}

type Repository interface {
	domain.Repository
	domain.SourceProjectionRepository
}

type Reconciler struct {
	repository Repository
	source     Source
	now        func() time.Time
}

type Input struct {
	WorkspaceID string
	Locator     string
	Ref         string
}

type UnresolvedReference struct {
	Relation     string
	TargetID     string
	ExpectedType string
	Reason       string
}

type Result struct {
	WorkspaceID       string
	RepositoryLocator string
	CommitSHA         string
	ManifestBlobSHA   string
	ManifestChecksum  string
	EntityID          string
	SourceBindingID   string
	UpsertedEdgeIDs   []string
	DeletedEdgeIDs    []string
	Unresolved        []UnresolvedReference
}

func New(repository Repository, source Source, now func() time.Time) (*Reconciler, error) {
	if repository == nil {
		return nil, ErrRepositoryRequired
	}
	if source == nil {
		return nil, ErrSourceRequired
	}
	if now == nil {
		now = time.Now
	}
	return &Reconciler{repository: repository, source: source, now: now}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, input Input) (Result, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return Result{}, ErrWorkspaceRequired
	}
	repositorySnapshot, err := r.source.GetRepository(ctx, input.Locator)
	if err != nil {
		return Result{}, fmt.Errorf("read github repository: %w", err)
	}
	ref := strings.TrimSpace(input.Ref)
	if ref == "" {
		ref = repositorySnapshot.DefaultBranch
	}
	commit, err := r.source.ResolveCommit(ctx, repositorySnapshot.Locator, ref)
	if err != nil {
		return Result{}, fmt.Errorf("resolve github commit: %w", err)
	}
	blob, err := r.source.ReadEngineeringManifestAtCommit(ctx, repositorySnapshot.Locator, commit.SHA)
	if err != nil {
		return Result{}, fmt.Errorf("read engineering manifest: %w", err)
	}
	if commit.RepositoryLocator != repositorySnapshot.Locator || blob.RepositoryLocator != repositorySnapshot.Locator || blob.CommitSHA != commit.SHA {
		return Result{}, ErrSourceSnapshotInvalid
	}
	parsed, err := manifest.Parse(blob.Content)
	if err != nil {
		return Result{}, fmt.Errorf("parse engineering manifest: %w", err)
	}
	if parsed.Source.Locator != repositorySnapshot.Locator || parsed.Source.Type != "github" {
		return Result{}, fmt.Errorf("%w: manifest source does not match fetched repository", ErrSourceSnapshotInvalid)
	}

	desiredEntity, err := domain.NewEngineeringEntity(
		parsed.Entity.ID, workspaceID, domain.EntityType(parsed.Entity.Type), parsed.Entity.Name,
		domain.EntityStatus(parsed.Entity.Status), parsed.Entity.OwnerRef,
	)
	if err != nil {
		return Result{}, fmt.Errorf("build manifest entity: %w", err)
	}
	bindingID := stableID("source-binding", repositorySnapshot.Locator)
	if err := r.validateEntityOwnership(ctx, workspaceID, desiredEntity, bindingID, repositorySnapshot.Locator); err != nil {
		return Result{}, err
	}

	observedAt := r.now().UTC()
	bindingProvenance, err := domain.NewProvenance(bindingSourceType, repositorySnapshot.Locator, commit.SHA, observedAt)
	if err != nil {
		return Result{}, err
	}
	binding, err := domain.NewSourceBinding(bindingID, workspaceID, desiredEntity.ID(), bindingProvenance, domain.AuthorityAuthoritative)
	if err != nil {
		return Result{}, err
	}

	primaryNode, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, desiredEntity.ID())
	existingEdges, err := r.repository.ListThreadEdges(ctx, workspaceID, primaryNode)
	if err != nil {
		return Result{}, fmt.Errorf("list existing engineering source edges: %w", err)
	}
	provenanceLocator := repositorySnapshot.Locator + "/engineering.yaml"
	edges, unresolved, err := r.buildEdges(ctx, workspaceID, desiredEntity, parsed, commit.SHA, provenanceLocator, observedAt)
	if err != nil {
		return Result{}, err
	}
	desiredIDs := make(map[string]struct{}, len(edges))
	upsertIDs := make([]string, 0, len(edges))
	for _, edge := range edges {
		desiredIDs[edge.ID()] = struct{}{}
		upsertIDs = append(upsertIDs, edge.ID())
	}
	staleIDs := staleSourceEdgeIDs(existingEdges, desiredEntity.ID(), provenanceLocator, desiredIDs)
	projection, err := domain.NewSourceProjection(desiredEntity, binding, edges, staleIDs)
	if err != nil {
		return Result{}, fmt.Errorf("build engineering source projection: %w", err)
	}
	if err := r.repository.ApplySourceProjection(ctx, projection); err != nil {
		return Result{}, fmt.Errorf("apply engineering source projection: %w", err)
	}
	sort.Strings(upsertIDs)
	sort.Strings(staleIDs)
	sort.Slice(unresolved, func(i, j int) bool {
		if unresolved[i].Relation != unresolved[j].Relation {
			return unresolved[i].Relation < unresolved[j].Relation
		}
		return unresolved[i].TargetID < unresolved[j].TargetID
	})
	return Result{
		WorkspaceID: workspaceID, RepositoryLocator: repositorySnapshot.Locator, CommitSHA: commit.SHA,
		ManifestBlobSHA: blob.BlobSHA, ManifestChecksum: parsed.Checksum(), EntityID: desiredEntity.ID(), SourceBindingID: binding.ID(),
		UpsertedEdgeIDs: upsertIDs, DeletedEdgeIDs: staleIDs, Unresolved: unresolved,
	}, nil
}

func (r *Reconciler) validateEntityOwnership(ctx context.Context, workspaceID string, desired domain.EngineeringEntity, bindingID, locator string) error {
	existingEntity, entityErr := r.repository.GetEntity(ctx, workspaceID, desired.ID())
	if entityErr != nil && !errors.Is(entityErr, domain.ErrNotFound) {
		return fmt.Errorf("read canonical engineering entity: %w", entityErr)
	}
	existingBinding, bindingErr := r.repository.GetSourceBinding(ctx, workspaceID, bindingID)
	if bindingErr != nil && !errors.Is(bindingErr, domain.ErrNotFound) {
		return fmt.Errorf("read canonical source binding: %w", bindingErr)
	}
	entityExists := entityErr == nil
	bindingExists := bindingErr == nil

	if bindingExists {
		provenance := existingBinding.Provenance()
		if existingBinding.EntityID() != desired.ID() || provenance.SourceType() != bindingSourceType || provenance.Locator() != locator {
			return ErrSourceOwnershipConflict
		}
		if !entityExists {
			return fmt.Errorf("%w: source binding exists without canonical entity", ErrSourceOwnershipConflict)
		}
		if existingEntity.Type() != desired.Type() {
			return fmt.Errorf("%w: source-managed entity type cannot change from %s to %s", ErrCanonicalEntityConflict, existingEntity.Type(), desired.Type())
		}
		return nil
	}
	if !entityExists {
		return nil
	}
	if !sameEntity(existingEntity, desired) {
		return fmt.Errorf("%w: existing entity %s is not source-owned and differs from manifest", ErrCanonicalEntityConflict, desired.ID())
	}
	return nil
}

func (r *Reconciler) buildEdges(ctx context.Context, workspaceID string, primary domain.EngineeringEntity, parsed manifest.Manifest, revision, provenanceLocator string, observedAt time.Time) ([]domain.ThreadEdge, []UnresolvedReference, error) {
	type edgeSpec struct {
		relation     domain.RelationType
		targetID     string
		expectedType domain.EntityType
	}
	var specs []edgeSpec
	for _, target := range parsed.Dependencies {
		specs = append(specs, edgeSpec{relation: domain.RelationDependsOn, targetID: target})
	}
	for _, relation := range parsed.Relations {
		specs = append(specs, edgeSpec{relation: domain.RelationType(relation.Relation), targetID: relation.Target})
	}
	for _, iface := range parsed.Interfaces {
		specs = append(specs, edgeSpec{relation: domain.RelationType(iface.Direction), targetID: iface.ID, expectedType: domain.EntityType(iface.Type)})
	}
	provenance, err := domain.NewProvenance(manifestEdgeSourceType, provenanceLocator, revision, observedAt)
	if err != nil {
		return nil, nil, err
	}
	from, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, primary.ID())
	var edges []domain.ThreadEdge
	var unresolved []UnresolvedReference
	for _, spec := range specs {
		target, err := r.repository.GetEntity(ctx, workspaceID, spec.targetID)
		if errors.Is(err, domain.ErrNotFound) {
			unresolved = append(unresolved, unresolvedReference(spec.relation, spec.targetID, spec.expectedType, "not_found"))
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("resolve manifest edge target %s: %w", spec.targetID, err)
		}
		if target.Status() == domain.EntityStatusArchived {
			unresolved = append(unresolved, unresolvedReference(spec.relation, spec.targetID, spec.expectedType, "archived"))
			continue
		}
		if spec.expectedType != "" && target.Type() != spec.expectedType {
			unresolved = append(unresolved, unresolvedReference(spec.relation, spec.targetID, spec.expectedType, "type_mismatch"))
			continue
		}
		to, _ := domain.NewNodeRef(domain.NodeKindEngineeringEntity, target.ID())
		edgeID := stableID("source-edge", provenanceLocator, primary.ID(), string(spec.relation), target.ID())
		edge, err := domain.NewThreadEdge(edgeID, workspaceID, from, spec.relation, to, domain.AuthorityAuthoritative, provenance)
		if err != nil {
			return nil, nil, fmt.Errorf("build manifest edge: %w", err)
		}
		edges = append(edges, edge)
	}
	return edges, unresolved, nil
}

func unresolvedReference(relation domain.RelationType, targetID string, expectedType domain.EntityType, reason string) UnresolvedReference {
	return UnresolvedReference{Relation: string(relation), TargetID: targetID, ExpectedType: string(expectedType), Reason: reason}
}

func staleSourceEdgeIDs(existing []domain.ThreadEdge, primaryID, provenanceLocator string, desired map[string]struct{}) []string {
	var stale []string
	for _, edge := range existing {
		provenance := edge.Provenance()
		if edge.From().Kind() != domain.NodeKindEngineeringEntity || edge.From().ID() != primaryID || provenance.SourceType() != manifestEdgeSourceType || provenance.Locator() != provenanceLocator {
			continue
		}
		if _, keep := desired[edge.ID()]; !keep {
			stale = append(stale, edge.ID())
		}
	}
	return stale
}

func sameEntity(left, right domain.EngineeringEntity) bool {
	return left.ID() == right.ID() && left.WorkspaceID() == right.WorkspaceID() && left.Type() == right.Type() && left.Name() == right.Name() && left.Status() == right.Status() && left.OwnerRef() == right.OwnerRef()
}

func stableID(prefix string, parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return prefix + ":" + hex.EncodeToString(hash.Sum(nil))
}
