package domain

// TrackableSection is deliberately limited to content that contributes to
// delivery coverage.
type TrackableSection string

// TrackableGoal, TrackableInScope, TrackableConstraint, and
// TrackableAcceptanceCriteria are the coverage-bearing content sections.
const (
	TrackableGoal               TrackableSection = "goals"
	TrackableInScope            TrackableSection = "in_scope"
	TrackableConstraint         TrackableSection = "constraints"
	TrackableAcceptanceCriteria TrackableSection = "acceptance_criteria"
)

// LinkedIssue is the issue data shown beside one requirement item.
type LinkedIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	CreatedBy  string `json:"created_by"`
	CreatedAt  string `json:"created_at"`
}

// CoverageItem reports issue coverage for one requirement key.
type CoverageItem struct {
	RequirementKey string           `json:"requirement_key"`
	Section        TrackableSection `json:"section"`
	Issues         []LinkedIssue    `json:"issues"`
}

// CoverageSnapshot reports all coverage calculations for one revision.
type CoverageSnapshot struct {
	Revision           int            `json:"revision"`
	Total              int            `json:"total"`
	Linked             int            `json:"linked"`
	Unlinked           int            `json:"unlinked"`
	LinkedIssueDone    int            `json:"linked_issue_done"`
	LinkedIssueBlocked int            `json:"linked_issue_blocked"`
	Items              []CoverageItem `json:"items"`
}

// Coverage combines coverage for current and effective revisions.
type Coverage struct {
	Current   *CoverageSnapshot `json:"current"`
	Effective *CoverageSnapshot `json:"effective"`
}

// TrackableItem pairs a content item with its coverage section.
type TrackableItem struct {
	Section TrackableSection
	Item    Item
}

// TrackableItems returns the content items allowed to receive issue links.
func TrackableItems(content Content) []TrackableItem {
	groups := []struct {
		section TrackableSection
		items   []Item
	}{
		{TrackableGoal, content.Goals},
		{TrackableInScope, content.InScope},
		{TrackableConstraint, content.Constraints},
		{TrackableAcceptanceCriteria, content.AcceptanceCriteria},
	}
	result := make([]TrackableItem, 0, len(content.Goals)+len(content.InScope)+len(content.Constraints)+len(content.AcceptanceCriteria))
	for _, group := range groups {
		for _, item := range group.items {
			result = append(result, TrackableItem{Section: group.section, Item: item})
		}
	}
	return result
}

// FindTrackableItem finds a trackable content item by its stable key.
func FindTrackableItem(content Content, key string) (TrackableItem, bool) {
	for _, item := range TrackableItems(content) {
		if item.Item.Key == key {
			return item, true
		}
	}
	return TrackableItem{}, false
}
