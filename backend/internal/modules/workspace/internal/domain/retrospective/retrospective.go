package retrospective

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"

	ActionSaveDraft       = "save_draft"
	ActionPublish         = "publish"
	ActionPublishRevision = "publish_revision"
	ActionArchive         = "archive"

	RoleParticipant = "participant"
	RoleFacilitator = "facilitator"

	MaxSummaryRunes           = 5000
	MaxActionItems            = 100
	maxListItems              = 100
	maxListItemRunes          = 2000
	maxActionTitleRunes       = 500
	maxActionDescriptionRunes = 5000
	maxParticipants           = 100
)

var (
	ErrInvalidContent          = errors.New("invalid Retrospective content")
	ErrInvalidParticipants     = errors.New("invalid Retrospective participants")
	ErrInvalidTransition       = errors.New("invalid Retrospective transition")
	ErrLinkedActionItemChanged = errors.New("linked Retrospective action item changed")
	actionIDPattern            = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

type ActionItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

type Content struct {
	Summary     string       `json:"summary"`
	Successes   []string     `json:"successes"`
	Problems    []string     `json:"problems"`
	Lessons     []string     `json:"lessons"`
	ActionItems []ActionItem `json:"action_items"`
}

type Participant struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

func NormalizeContent(input Content) (Content, error) {
	result := input
	result.Summary = strings.TrimSpace(input.Summary)
	if result.Summary == "" || utf8.RuneCountInString(result.Summary) > MaxSummaryRunes || len(input.ActionItems) > MaxActionItems {
		return Content{}, ErrInvalidContent
	}
	var err error
	if result.Successes, err = normalizeTextList(input.Successes, false); err != nil {
		return Content{}, err
	}
	if result.Problems, err = normalizeTextList(input.Problems, false); err != nil {
		return Content{}, err
	}
	if result.Lessons, err = normalizeTextList(input.Lessons, true); err != nil {
		return Content{}, err
	}
	result.ActionItems = make([]ActionItem, len(input.ActionItems))
	seen := make(map[string]struct{}, len(input.ActionItems))
	for index, item := range input.ActionItems {
		item.ID = strings.TrimSpace(item.ID)
		item.Title = strings.TrimSpace(item.Title)
		item.Description = strings.TrimSpace(item.Description)
		item.AssigneeID = strings.TrimSpace(item.AssigneeID)
		item.DueDate = strings.TrimSpace(item.DueDate)
		if !actionIDPattern.MatchString(item.ID) || item.Title == "" || utf8.RuneCountInString(item.Title) > maxActionTitleRunes || utf8.RuneCountInString(item.Description) > maxActionDescriptionRunes || !validDueDate(item.DueDate) {
			return Content{}, ErrInvalidContent
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return Content{}, ErrInvalidContent
		}
		seen[item.ID] = struct{}{}
		result.ActionItems[index] = item
	}
	return result, nil
}

func normalizeTextList(input []string, required bool) ([]string, error) {
	if len(input) > maxListItems {
		return nil, ErrInvalidContent
	}
	result := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, value := range input {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if utf8.RuneCountInString(value) > maxListItemRunes {
			return nil, ErrInvalidContent
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if required && len(result) == 0 {
		return nil, ErrInvalidContent
	}
	return result, nil
}

func NormalizeParticipants(input []Participant, creatorMemberID string) ([]Participant, error) {
	creatorMemberID = strings.TrimSpace(creatorMemberID)
	if creatorMemberID == "" || len(input) > maxParticipants {
		return nil, ErrInvalidParticipants
	}
	result := make([]Participant, 0, len(input)+1)
	seen := make(map[string]struct{}, len(input)+1)
	for _, participant := range input {
		participant.MemberID = strings.TrimSpace(participant.MemberID)
		participant.Role = strings.TrimSpace(participant.Role)
		if participant.MemberID == "" || (participant.Role != RoleParticipant && participant.Role != RoleFacilitator) {
			return nil, ErrInvalidParticipants
		}
		if _, duplicate := seen[participant.MemberID]; duplicate {
			return nil, ErrInvalidParticipants
		}
		seen[participant.MemberID] = struct{}{}
		result = append(result, participant)
	}
	if _, exists := seen[creatorMemberID]; !exists {
		if len(result) >= maxParticipants {
			return nil, ErrInvalidParticipants
		}
		result = append(result, Participant{MemberID: creatorMemberID, Role: RoleParticipant})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].MemberID < result[j].MemberID })
	return result, nil
}

func validDueDate(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := time.Parse(time.DateOnly, value)
	return err == nil && parsed.Format(time.DateOnly) == value
}

func NextStatus(current, action string) (string, error) {
	switch {
	case current == StatusDraft && action == ActionSaveDraft:
		return StatusDraft, nil
	case current == StatusDraft && action == ActionPublish:
		return StatusPublished, nil
	case current == StatusPublished && action == ActionPublishRevision:
		return StatusPublished, nil
	case (current == StatusDraft || current == StatusPublished) && action == ActionArchive:
		return StatusArchived, nil
	default:
		return "", ErrInvalidTransition
	}
}

func ValidateLinkedActionItemsUnchanged(previous, next Content, linked map[string]struct{}) error {
	if len(linked) == 0 {
		return nil
	}
	nextByID := make(map[string]ActionItem, len(next.ActionItems))
	for _, item := range next.ActionItems {
		nextByID[item.ID] = item
	}
	for _, item := range previous.ActionItems {
		if _, required := linked[item.ID]; !required {
			continue
		}
		candidate, exists := nextByID[item.ID]
		if !exists || candidate != item {
			return ErrLinkedActionItemChanged
		}
	}
	return nil
}
