package requirement

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusInReview Status = "in_review"
	StatusApproved Status = "approved"
	StatusFrozen   Status = "frozen"
	StatusChanged  Status = "changed"
	StatusRetired  Status = "retired"
)

type Action string

const (
	ActionCreate         Action = "create"
	ActionSaveDraft      Action = "save_draft"
	ActionSubmitReview   Action = "submit_review"
	ActionWithdrawReview Action = "withdraw_review"
	ActionApprove        Action = "approve"
	ActionFreeze         Action = "freeze"
	ActionMaterialChange Action = "material_change"
	ActionRetire         Action = "retire"
	ActionLinkIssue      Action = "link_issue"
	ActionUnlinkIssue    Action = "unlink_issue"
	ActionLinkOutline    Action = "link_outline"
	ActionUnlinkOutline  Action = "unlink_outline"
	ActionIssueDeleted   Action = "issue_deleted"
	ActionLegacyImport   Action = "legacy_import"
)

var (
	ErrInvalidBaseline             = errors.New("invalid Requirement baseline")
	ErrRevisionConflict            = errors.New("Requirement revision conflict")
	ErrInvalidTransition           = errors.New("invalid Requirement transition")
	ErrIndependentApprovalRequired = errors.New("independent Requirement approval required")
	ErrMaterialChangeRequired      = errors.New("material Requirement change required")
	traceabilityKeyPattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
)

type RevisionConflictError struct {
	Current int64
}

func (e RevisionConflictError) Error() string {
	return fmt.Sprintf("%s: current revision %d", ErrRevisionConflict, e.Current)
}

func (e RevisionConflictError) Unwrap() error { return ErrRevisionConflict }

type Item struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type Content struct {
	ProblemStatement   string `json:"problem_statement"`
	Goals              []Item `json:"goals"`
	InScope            []Item `json:"in_scope"`
	OutOfScope         []Item `json:"out_of_scope"`
	Constraints        []Item `json:"constraints"`
	AcceptanceCriteria []Item `json:"acceptance_criteria"`
	Dependencies       []Item `json:"dependencies"`
}

type Baseline struct {
	ID                  string
	WorkspaceID         string
	ProjectID           string
	Status              Status
	CurrentRevision     int64
	ApprovedRevision    *int64
	EffectiveRevision   *int64
	ReviewOrigin        Status
	Content             Content
	LatestContentAuthor string
	SubmittedBy         *string
	SubmittedAt         *time.Time
	ApprovedBy          *string
	ApprovedAt          *time.Time
	FrozenBy            *string
	FrozenAt            *time.Time
	RetiredBy           *string
	RetiredAt           *time.Time
	LegacyRequirementID *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Revision struct {
	BaselineID    string
	Revision      int64
	Content       Content
	Status        Status
	Action        Action
	ChangeSummary string
	ActorID       string
	SubmittedBy   *string
	SubmittedAt   *time.Time
	ApprovedBy    *string
	ApprovedAt    *time.Time
	FrozenBy      *string
	FrozenAt      *time.Time
	CreatedAt     time.Time
}

func NewBaseline(id, workspaceID, projectID string, content Content, changeSummary, actorID string, now time.Time) (Baseline, Revision, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	projectID = strings.TrimSpace(projectID)
	actorID = strings.TrimSpace(actorID)
	changeSummary = strings.TrimSpace(changeSummary)
	if id == "" || workspaceID == "" || projectID == "" || actorID == "" || !validChangeSummary(changeSummary) {
		return Baseline{}, Revision{}, ErrInvalidBaseline
	}
	normalized, err := normalizeContent(content)
	if err != nil {
		return Baseline{}, Revision{}, err
	}
	now = now.UTC()
	baseline := Baseline{
		ID:                  id,
		WorkspaceID:         workspaceID,
		ProjectID:           projectID,
		Status:              StatusDraft,
		CurrentRevision:     1,
		Content:             normalized,
		LatestContentAuthor: actorID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	return baseline, baseline.snapshot(ActionCreate, changeSummary, actorID, now), nil
}

func (b Baseline) SaveDraft(expectedRevision int64, content Content, changeSummary, actorID string, materialChange bool, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	changeSummary = strings.TrimSpace(changeSummary)
	if actorID == "" || !validChangeSummary(changeSummary) {
		return Baseline{}, Revision{}, ErrInvalidBaseline
	}
	normalized, err := normalizeContent(content)
	if err != nil {
		return Baseline{}, Revision{}, err
	}
	action := ActionSaveDraft
	switch b.Status {
	case StatusDraft, StatusChanged:
		if materialChange {
			return Baseline{}, Revision{}, ErrInvalidTransition
		}
	case StatusFrozen:
		if !materialChange {
			return Baseline{}, Revision{}, ErrInvalidTransition
		}
		if reflect.DeepEqual(b.Content, normalized) {
			return Baseline{}, Revision{}, ErrMaterialChangeRequired
		}
		b.Status = StatusChanged
		action = ActionMaterialChange
	default:
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	b.Content = normalized
	b.LatestContentAuthor = actorID
	b.ReviewOrigin = ""
	b.SubmittedBy = nil
	b.SubmittedAt = nil
	return b.advanceOK(action, changeSummary, actorID, now)
}

func (b Baseline) SubmitReview(expectedRevision int64, actorID string, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || (b.Status != StatusDraft && b.Status != StatusChanged) {
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	b.ReviewOrigin = b.Status
	b.Status = StatusInReview
	b.SubmittedBy = stringPointer(actorID)
	b.SubmittedAt = timePointer(now.UTC())
	return b.advanceOK(ActionSubmitReview, "Submitted for review", actorID, now)
}

func (b Baseline) WithdrawReview(expectedRevision int64, actorID string, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || b.Status != StatusInReview || (b.ReviewOrigin != StatusDraft && b.ReviewOrigin != StatusChanged) {
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	b.Status = b.ReviewOrigin
	b.ReviewOrigin = ""
	b.SubmittedBy = nil
	b.SubmittedAt = nil
	return b.advanceOK(ActionWithdrawReview, "Withdrew review", actorID, now)
}

func (b Baseline) Approve(expectedRevision int64, actorID string, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || b.Status != StatusInReview {
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	if actorID == b.LatestContentAuthor || (b.SubmittedBy != nil && actorID == *b.SubmittedBy) {
		return Baseline{}, Revision{}, ErrIndependentApprovalRequired
	}
	b.Status = StatusApproved
	b.ReviewOrigin = ""
	b.ApprovedRevision = int64Pointer(b.CurrentRevision + 1)
	b.ApprovedBy = stringPointer(actorID)
	b.ApprovedAt = timePointer(now.UTC())
	return b.advanceOK(ActionApprove, "Approved Requirement", actorID, now)
}

func (b Baseline) Freeze(expectedRevision int64, actorID string, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || b.Status != StatusApproved {
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	b.Status = StatusFrozen
	b.EffectiveRevision = int64Pointer(b.CurrentRevision + 1)
	b.FrozenBy = stringPointer(actorID)
	b.FrozenAt = timePointer(now.UTC())
	return b.advanceOK(ActionFreeze, "Froze Requirement", actorID, now)
}

func (b Baseline) Retire(expectedRevision int64, actorID string, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || b.Status == StatusRetired || !validStatus(b.Status) {
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	b.Status = StatusRetired
	b.ReviewOrigin = ""
	b.RetiredBy = stringPointer(actorID)
	b.RetiredAt = timePointer(now.UTC())
	return b.advanceOK(ActionRetire, "Retired Requirement", actorID, now)
}

func (b Baseline) RecordTraceabilityMutation(expectedRevision int64, action Action, actorID string, now time.Time) (Baseline, Revision, error) {
	if err := b.checkExpected(expectedRevision); err != nil {
		return Baseline{}, Revision{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || !validTraceabilityAction(action) || (b.Status == StatusRetired && action != ActionIssueDeleted) {
		return Baseline{}, Revision{}, ErrInvalidTransition
	}
	if b.Status == StatusFrozen {
		b.EffectiveRevision = int64Pointer(b.CurrentRevision + 1)
	}
	return b.advanceOK(action, traceabilitySummary(action), actorID, now)
}

func (b Baseline) advance(action Action, summary, actorID string, now time.Time) (Baseline, Revision) {
	now = now.UTC()
	b.CurrentRevision++
	b.UpdatedAt = now
	return b, b.snapshot(action, summary, actorID, now)
}

func (b Baseline) advanceOK(action Action, summary, actorID string, now time.Time) (Baseline, Revision, error) {
	next, revision := b.advance(action, summary, actorID, now)
	return next, revision, nil
}

func (b Baseline) snapshot(action Action, summary, actorID string, now time.Time) Revision {
	return Revision{
		BaselineID:    b.ID,
		Revision:      b.CurrentRevision,
		Content:       cloneContent(b.Content),
		Status:        b.Status,
		Action:        action,
		ChangeSummary: summary,
		ActorID:       actorID,
		SubmittedBy:   cloneStringPointer(b.SubmittedBy),
		SubmittedAt:   cloneTimePointer(b.SubmittedAt),
		ApprovedBy:    cloneStringPointer(b.ApprovedBy),
		ApprovedAt:    cloneTimePointer(b.ApprovedAt),
		FrozenBy:      cloneStringPointer(b.FrozenBy),
		FrozenAt:      cloneTimePointer(b.FrozenAt),
		CreatedAt:     now.UTC(),
	}
}

func (b Baseline) checkExpected(expected int64) error {
	if expected != b.CurrentRevision {
		return RevisionConflictError{Current: b.CurrentRevision}
	}
	return nil
}

func normalizeContent(content Content) (Content, error) {
	content.ProblemStatement = strings.TrimSpace(content.ProblemStatement)
	if len(content.ProblemStatement) > 8000 {
		return Content{}, ErrInvalidBaseline
	}
	sections := []*[]Item{
		&content.Goals,
		&content.InScope,
		&content.OutOfScope,
		&content.Constraints,
		&content.AcceptanceCriteria,
		&content.Dependencies,
	}
	seen := make(map[string]struct{})
	meaningful := content.ProblemStatement != ""
	for _, section := range sections {
		if len(*section) > 100 {
			return Content{}, ErrInvalidBaseline
		}
		items := make([]Item, len(*section))
		for index, item := range *section {
			item.Key = strings.TrimSpace(item.Key)
			item.Text = strings.TrimSpace(item.Text)
			if !traceabilityKeyPattern.MatchString(item.Key) || item.Text == "" || len(item.Text) > 2000 {
				return Content{}, ErrInvalidBaseline
			}
			if _, duplicate := seen[item.Key]; duplicate {
				return Content{}, ErrInvalidBaseline
			}
			seen[item.Key] = struct{}{}
			items[index] = item
			meaningful = true
		}
		*section = items
	}
	if !meaningful {
		return Content{}, ErrInvalidBaseline
	}
	return content, nil
}

func (c Content) TraceableSection(key string) (string, bool) {
	key = strings.TrimSpace(key)
	sections := []struct {
		name  string
		items []Item
	}{
		{name: "goals", items: c.Goals},
		{name: "in_scope", items: c.InScope},
		{name: "constraints", items: c.Constraints},
		{name: "acceptance_criteria", items: c.AcceptanceCriteria},
	}
	for _, section := range sections {
		for _, item := range section.items {
			if item.Key == key {
				return section.name, true
			}
		}
	}
	return "", false
}

func validStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusInReview, StatusApproved, StatusFrozen, StatusChanged, StatusRetired:
		return true
	default:
		return false
	}
}

func validTraceabilityAction(action Action) bool {
	switch action {
	case ActionLinkIssue, ActionUnlinkIssue, ActionLinkOutline, ActionUnlinkOutline, ActionIssueDeleted:
		return true
	default:
		return false
	}
}

func traceabilitySummary(action Action) string {
	switch action {
	case ActionLinkIssue:
		return "Linked Issue"
	case ActionUnlinkIssue:
		return "Unlinked Issue"
	case ActionLinkOutline:
		return "Linked outline node"
	case ActionUnlinkOutline:
		return "Unlinked outline node"
	case ActionIssueDeleted:
		return "Removed deleted Issue links"
	default:
		return "Updated traceability"
	}
}

func validChangeSummary(value string) bool { return value != "" && len(value) <= 500 }

func cloneContent(value Content) Content {
	value.Goals = append([]Item(nil), value.Goals...)
	value.InScope = append([]Item(nil), value.InScope...)
	value.OutOfScope = append([]Item(nil), value.OutOfScope...)
	value.Constraints = append([]Item(nil), value.Constraints...)
	value.AcceptanceCriteria = append([]Item(nil), value.AcceptanceCriteria...)
	value.Dependencies = append([]Item(nil), value.Dependencies...)
	return value
}

func int64Pointer(value int64) *int64 { return &value }

func stringPointer(value string) *string { return &value }

func timePointer(value time.Time) *time.Time { return &value }

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	return stringPointer(*value)
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	return timePointer(*value)
}
