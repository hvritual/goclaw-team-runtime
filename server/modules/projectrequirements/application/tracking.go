package application

import (
	"context"

	"github.com/multica-ai/multica/server/modules/projectrequirements/domain"
)

type (
	// TrackableSection identifies a content section that contributes to delivery coverage.
	TrackableSection = domain.TrackableSection
	// LinkedIssue is an issue associated with one tracked requirement item.
	LinkedIssue = domain.LinkedIssue
	// CoverageItem reports coverage for a single requirement key.
	CoverageItem = domain.CoverageItem
	// CoverageSnapshot reports coverage for one baseline revision.
	CoverageSnapshot = domain.CoverageSnapshot
	// Coverage combines current and effective baseline coverage.
	Coverage = domain.Coverage
	// TrackableItem pairs a requirement item with its trackable section.
	TrackableItem = domain.TrackableItem
)

// TrackableGoal, TrackableInScope, TrackableConstraint, and
// TrackableAcceptanceCriteria identify coverage-bearing content sections.
const (
	TrackableGoal               = domain.TrackableGoal
	TrackableInScope            = domain.TrackableInScope
	TrackableConstraint         = domain.TrackableConstraint
	TrackableAcceptanceCriteria = domain.TrackableAcceptanceCriteria
)

// TrackableItems returns the current content items that can receive issue links.
func TrackableItems(content Content) []TrackableItem { return domain.TrackableItems(content) }

// FindTrackableItem finds one trackable item by its stable key.
func FindTrackableItem(content Content, key string) (TrackableItem, bool) {
	return domain.FindTrackableItem(content, key)
}

// LinkInput identifies an issue-to-requirement association.
type LinkInput struct {
	WorkspaceID, ProjectID, RequirementKey, IssueID, ActorID string
	Revision                                                 int
}

// TrackingRepository keeps the tracking contract independent of the storage
// adapter, while the domain owns the coverage vocabulary and item selection.
type TrackingRepository interface {
	Coverage(ctx context.Context, workspaceID, projectID string) (Coverage, error)
	Link(ctx context.Context, input LinkInput) error
	Unlink(ctx context.Context, input LinkInput) error
}

// TrackingService executes requirement coverage and link use cases.
type TrackingService struct{ repository TrackingRepository }

// NewTrackingService constructs the tracking use-case service.
func NewTrackingService(repository TrackingRepository) *TrackingService {
	return &TrackingService{repository: repository}
}

// Coverage returns current and effective requirement coverage for a project.
func (s *TrackingService) Coverage(ctx context.Context, workspaceID, projectID string) (Coverage, error) {
	return s.repository.Coverage(ctx, workspaceID, projectID)
}

// Link associates an issue with one valid requirement item.
func (s *TrackingService) Link(ctx context.Context, input LinkInput) error {
	return s.repository.Link(ctx, input)
}

// Unlink removes an issue association from one valid requirement item.
func (s *TrackingService) Unlink(ctx context.Context, input LinkInput) error {
	return s.repository.Unlink(ctx, input)
}
