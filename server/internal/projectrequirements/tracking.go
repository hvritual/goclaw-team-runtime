package projectrequirements

import "context"

// TrackableSection is deliberately limited to content that contributes to delivery coverage.
type TrackableSection string

const (
	TrackableGoal               TrackableSection = "goals"
	TrackableInScope            TrackableSection = "in_scope"
	TrackableConstraint         TrackableSection = "constraints"
	TrackableAcceptanceCriteria TrackableSection = "acceptance_criteria"
)

type LinkedIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	CreatedBy  string `json:"created_by"`
	CreatedAt  string `json:"created_at"`
}

type CoverageItem struct {
	RequirementKey string           `json:"requirement_key"`
	Section        TrackableSection `json:"section"`
	Issues         []LinkedIssue    `json:"issues"`
}

type CoverageSnapshot struct {
	Revision           int            `json:"revision"`
	Total              int            `json:"total"`
	Linked             int            `json:"linked"`
	Unlinked           int            `json:"unlinked"`
	LinkedIssueDone    int            `json:"linked_issue_done"`
	LinkedIssueBlocked int            `json:"linked_issue_blocked"`
	Items              []CoverageItem `json:"items"`
}

type Coverage struct {
	Current   *CoverageSnapshot `json:"current"`
	Effective *CoverageSnapshot `json:"effective"`
}

type LinkInput struct {
	WorkspaceID, ProjectID, RequirementKey, IssueID, ActorID string
	Revision                                                 int
}

type TrackableItem struct {
	Section TrackableSection
	Item    Item
}

func TrackableItems(content Content) []TrackableItem {
	groups := []struct {
		section TrackableSection
		items   []Item
	}{{TrackableGoal, content.Goals}, {TrackableInScope, content.InScope}, {TrackableConstraint, content.Constraints}, {TrackableAcceptanceCriteria, content.AcceptanceCriteria}}
	result := make([]TrackableItem, 0, len(content.Goals)+len(content.InScope)+len(content.Constraints)+len(content.AcceptanceCriteria))
	for _, group := range groups {
		for _, item := range group.items {
			result = append(result, TrackableItem{Section: group.section, Item: item})
		}
	}
	return result
}

func FindTrackableItem(content Content, key string) (TrackableItem, bool) {
	for _, item := range TrackableItems(content) {
		if item.Item.Key == key {
			return item, true
		}
	}
	return TrackableItem{}, false
}

// TrackingRepository keeps the tracking contract independent of the storage adapter.
type TrackingRepository interface {
	Coverage(ctx context.Context, workspaceID, projectID string) (Coverage, error)
	Link(ctx context.Context, input LinkInput) error
	Unlink(ctx context.Context, input LinkInput) error
}

type TrackingService struct{ repository TrackingRepository }

func NewTrackingService(repository TrackingRepository) *TrackingService {
	return &TrackingService{repository: repository}
}
func (s *TrackingService) Coverage(ctx context.Context, workspaceID, projectID string) (Coverage, error) {
	return s.repository.Coverage(ctx, workspaceID, projectID)
}
func (s *TrackingService) Link(ctx context.Context, input LinkInput) error {
	return s.repository.Link(ctx, input)
}
func (s *TrackingService) Unlink(ctx context.Context, input LinkInput) error {
	return s.repository.Unlink(ctx, input)
}
