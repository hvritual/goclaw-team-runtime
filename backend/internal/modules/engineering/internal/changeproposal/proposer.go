package changeproposal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	githubsource "github.com/hvritual/workspace/internal/modules/engineering/internal/source/github"
)

var (
	ErrRepositoryRequired   = errors.New("change proposal repository is required")
	ErrSourceRequired       = errors.New("change proposal source is required")
	ErrInvalidInput         = errors.New("invalid change proposal input")
	ErrPullRequestNotMerged = errors.New("github pull request is not merged")
	ErrSourceBindingMissing = errors.New("github source binding is missing")
	ErrWorkLinkMissing      = errors.New("workspace engineering work link is missing")
	ErrProposalConflict     = errors.New("existing change conflicts with github proposal")
)

type Source interface {
	GetPullRequest(context.Context, string, int) (githubsource.PullRequest, error)
}

type Repository interface {
	domain.Repository
	domain.SourceBindingLookupRepository
}

type Proposer struct {
	repository Repository
	source     Source
	now        func() time.Time
}

type Input struct {
	WorkspaceID       string
	RepositoryLocator string
	PullRequestNumber int
	ProjectID         string
	RequirementID     string
	WorkItem          domain.NodeRef
	RunID             string
}

type Result struct {
	Change      domain.Change
	PullRequest githubsource.PullRequest
}

func New(repository Repository, source Source, now func() time.Time) (*Proposer, error) {
	if repository == nil {
		return nil, ErrRepositoryRequired
	}
	if source == nil {
		return nil, ErrSourceRequired
	}
	if now == nil {
		now = time.Now
	}
	return &Proposer{repository: repository, source: source, now: now}, nil
}

func (p *Proposer) Propose(ctx context.Context, input Input) (Result, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	locator := strings.TrimSpace(input.RepositoryLocator)
	projectID := strings.TrimSpace(input.ProjectID)
	requirementID := strings.TrimSpace(input.RequirementID)
	runID := strings.TrimSpace(input.RunID)
	workRelation, err := expectedWorkRelation(input.WorkItem)
	if workspaceID == "" || locator == "" || input.PullRequestNumber <= 0 || err != nil {
		return Result{}, ErrInvalidInput
	}
	pull, err := p.source.GetPullRequest(ctx, locator, input.PullRequestNumber)
	if err != nil {
		return Result{}, fmt.Errorf("read github pull request: %w", err)
	}
	mergeSHA := strings.ToLower(strings.TrimSpace(pull.MergeCommitSHA))
	if !pull.Merged || !isImmutableGitSHA(mergeSHA) {
		return Result{}, ErrPullRequestNotMerged
	}
	binding, err := p.repository.FindSourceBindingBySource(ctx, workspaceID, "github", pull.RepositoryLocator)
	if errors.Is(err, domain.ErrNotFound) {
		return Result{}, ErrSourceBindingMissing
	}
	if err != nil {
		return Result{}, fmt.Errorf("find github source binding: %w", err)
	}
	if binding.Authority() != domain.AuthorityAuthoritative {
		return Result{}, ErrSourceBindingMissing
	}
	if err := p.requireWorkLink(ctx, workspaceID, input.WorkItem, workRelation, binding.EntityID()); err != nil {
		return Result{}, err
	}
	if _, err := p.repository.GetEntity(ctx, workspaceID, binding.EntityID()); err != nil {
		return Result{}, fmt.Errorf("read source-bound engineering entity: %w", err)
	}

	pullLocator := pull.RepositoryLocator + "/pull/" + strconv.Itoa(pull.Number)
	artifacts := make([]domain.ArtifactRef, 0, 2)
	pullArtifact, err := domain.NewArtifactRef("pull_request", pullLocator, mergeSHA)
	if err != nil {
		return Result{}, err
	}
	artifacts = append(artifacts, pullArtifact)
	commitArtifact, err := domain.NewArtifactRef("commit", pull.RepositoryLocator+"/commit/"+mergeSHA, mergeSHA)
	if err != nil {
		return Result{}, err
	}
	artifacts = append(artifacts, commitArtifact)
	provenance, err := domain.NewProvenance("github_pr", pullLocator, mergeSHA, p.now().UTC())
	if err != nil {
		return Result{}, err
	}
	changeID := proposalID(workspaceID, pull.RepositoryLocator, pull.Number, input.WorkItem)
	change, err := domain.NewChange(
		changeID, workspaceID, projectID, requirementID, &input.WorkItem, runID,
		fmt.Sprintf("GitHub PR #%d merged", pull.Number), []string{binding.EntityID()}, artifacts, provenance, p.now().UTC(),
	)
	if err != nil {
		return Result{}, fmt.Errorf("build proposed change: %w", err)
	}
	if existing, getErr := p.repository.GetChange(ctx, workspaceID, changeID); getErr == nil {
		if sameProposal(existing, change) {
			return Result{Change: existing, PullRequest: pull}, nil
		}
		return Result{}, ErrProposalConflict
	} else if !errors.Is(getErr, domain.ErrNotFound) {
		return Result{}, fmt.Errorf("read existing proposed change: %w", getErr)
	}
	if err := p.repository.PutChange(ctx, change); err != nil {
		return Result{}, fmt.Errorf("persist proposed change: %w", err)
	}
	return Result{Change: change, PullRequest: pull}, nil
}

func (p *Proposer) requireWorkLink(ctx context.Context, workspaceID string, work domain.NodeRef, relation domain.RelationType, entityID string) error {
	edges, err := p.repository.ListThreadEdges(ctx, workspaceID, work)
	if err != nil {
		return fmt.Errorf("list work engineering links: %w", err)
	}
	for _, edge := range edges {
		if edge.From().Equal(work) && edge.Relation() == relation && edge.To().Kind() == domain.NodeKindEngineeringEntity && edge.To().ID() == entityID && edge.Authority() == domain.AuthorityAuthoritative && edge.Provenance().SourceType() == "workspace" {
			return nil
		}
	}
	return ErrWorkLinkMissing
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

func proposalID(workspaceID, locator string, number int, work domain.NodeRef) string {
	payload := strings.Join([]string{workspaceID, locator, strconv.Itoa(number), string(work.Kind()), work.ID()}, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "github-change:" + hex.EncodeToString(sum[:])
}

func sameProposal(left, right domain.Change) bool {
	if left.ID() != right.ID() || left.WorkspaceID() != right.WorkspaceID() || left.ProjectID() != right.ProjectID() || left.RequirementID() != right.RequirementID() || left.RunID() != right.RunID() || left.Summary() != right.Summary() {
		return false
	}
	leftWork, rightWork := left.WorkItem(), right.WorkItem()
	if leftWork == nil || rightWork == nil || !leftWork.Equal(*rightWork) {
		return false
	}
	leftProvenance, rightProvenance := left.Provenance(), right.Provenance()
	if leftProvenance.SourceType() != rightProvenance.SourceType() || leftProvenance.Locator() != rightProvenance.Locator() || leftProvenance.Revision() != rightProvenance.Revision() {
		return false
	}
	leftAffected, rightAffected := left.AffectedEntityIDs(), right.AffectedEntityIDs()
	if len(leftAffected) != len(rightAffected) {
		return false
	}
	for index := range leftAffected {
		if leftAffected[index] != rightAffected[index] {
			return false
		}
	}
	leftArtifacts, rightArtifacts := left.Artifacts(), right.Artifacts()
	if len(leftArtifacts) != len(rightArtifacts) {
		return false
	}
	for index := range leftArtifacts {
		if leftArtifacts[index].Kind() != rightArtifacts[index].Kind() || leftArtifacts[index].Locator() != rightArtifacts[index].Locator() || leftArtifacts[index].Revision() != rightArtifacts[index].Revision() {
			return false
		}
	}
	return true
}

func isImmutableGitSHA(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
