package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

var (
	ErrIssueCatalogInvalid     = errors.New("invalid Issue catalog request")
	ErrIssueLabelNotFound      = errors.New("label not found")
	ErrIssuePropertyNotFound   = errors.New("property not found")
	ErrIssueCatalogConflict    = errors.New("Issue catalog conflict")
	ErrIssuePropertyLimit      = errors.New("Issue property active limit reached")
	ErrIssueAcceptanceConflict = errors.New("Issue acceptance conflict")
)

type IssuePropertyOptionUsage struct {
	Name  string
	Count int
}

type IssuePropertyOptionsInUseError struct{ Options []IssuePropertyOptionUsage }

func (e *IssuePropertyOptionsInUseError) Error() string {
	parts := make([]string, 0, len(e.Options))
	for _, option := range e.Options {
		noun := "issues"
		if option.Count == 1 {
			noun = "issue"
		}
		parts = append(parts, fmt.Sprintf("%q (%d %s)", option.Name, option.Count, noun))
	}
	return "cannot remove options still in use: " + strings.Join(parts, ", ") + "; clear or change those values first"
}

const (
	PermissionLabelList       = "workspace.issue.label.list"
	PermissionLabelWrite      = "workspace.issue.label.write"
	PermissionPropertyList    = "workspace.issue.property.list"
	PermissionPropertyWrite   = "workspace.issue.property.write"
	PermissionAcceptanceList  = "workspace.issue.acceptance.list"
	PermissionAcceptanceWrite = "workspace.issue.acceptance.write"
)

type IssueCatalogRepository interface {
	ResolveIssue(context.Context, string, string) (string, error)
	ListLabels(context.Context, string) ([]contract.IssueLabel, error)
	GetLabel(context.Context, string, string) (contract.IssueLabel, error)
	CreateLabel(context.Context, contract.IssueLabel) (contract.IssueLabel, error)
	UpdateLabel(context.Context, contract.IssueLabel) (contract.IssueLabel, error)
	DeleteLabel(context.Context, string, string) error
	ListIssueLabels(context.Context, string, string) ([]contract.IssueLabel, error)
	AttachIssueLabel(context.Context, string, string, string, string) ([]contract.IssueLabel, error)
	DetachIssueLabel(context.Context, string, string, string) ([]contract.IssueLabel, error)

	ListProperties(context.Context, string, bool) ([]contract.IssuePropertyDefinition, error)
	GetProperty(context.Context, string, string) (contract.IssuePropertyDefinition, error)
	CreateProperty(context.Context, contract.IssuePropertyDefinition) (contract.IssuePropertyDefinition, error)
	UpdateProperty(context.Context, contract.IssuePropertyDefinition, []string) (contract.IssuePropertyDefinition, error)
	SetIssueProperty(context.Context, string, string, string, json.RawMessage, string) (string, map[string]any, error)
	UnsetIssueProperty(context.Context, string, string, string, string) (string, map[string]any, error)

	ListAcceptanceConclusions(context.Context, string, string) ([]contract.AcceptanceConclusion, error)
	CreateAcceptanceConclusion(context.Context, string, string, contract.AcceptanceConclusion, bool, string) (issueDomain.Issue, contract.AcceptanceConclusion, error)
}

type IssueCatalogUseCase struct {
	repository  IssueCatalogRepository
	authorizer  contract.WorkspaceAccessAuthorizer
	actors      contract.WorkspaceActorReader
	memberships contract.WorkspaceMembershipReader
	newID       ProjectIDGenerator
	now         Clock
}

func NewIssueCatalogUseCase(repository IssueCatalogRepository, authorizer contract.WorkspaceAccessAuthorizer, actors contract.WorkspaceActorReader, memberships contract.WorkspaceMembershipReader, newID ProjectIDGenerator, now Clock) (*IssueCatalogUseCase, error) {
	if repository == nil || authorizer == nil || actors == nil || memberships == nil || newID == nil || now == nil {
		return nil, errors.New("Issue catalog dependencies are required")
	}
	return &IssueCatalogUseCase{repository: repository, authorizer: authorizer, actors: actors, memberships: memberships, newID: newID, now: now}, nil
}

func (u *IssueCatalogUseCase) ListIssueLabels(ctx context.Context, workspaceID string) ([]contract.IssueLabel, error) {
	if err := u.authorize(ctx, workspaceID, PermissionLabelList); err != nil {
		return nil, err
	}
	values, err := u.repository.ListLabels(ctx, strings.TrimSpace(workspaceID))
	if values == nil {
		values = []contract.IssueLabel{}
	}
	return values, err
}

func (u *IssueCatalogUseCase) GetIssueLabel(ctx context.Context, workspaceID, labelID string) (contract.IssueLabel, error) {
	if err := u.authorize(ctx, workspaceID, PermissionLabelList); err != nil {
		return contract.IssueLabel{}, err
	}
	return u.repository.GetLabel(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(labelID))
}

func (u *IssueCatalogUseCase) CreateIssueLabel(ctx context.Context, request contract.CreateIssueLabelRequest) (contract.IssueLabel, error) {
	if err := u.authorize(ctx, request.WorkspaceID, PermissionLabelWrite); err != nil {
		return contract.IssueLabel{}, err
	}
	name, err := validateCatalogName(request.Name)
	if err != nil {
		return contract.IssueLabel{}, err
	}
	color, err := normalizeCatalogColor(request.Color)
	if err != nil {
		return contract.IssueLabel{}, err
	}
	resourceType := strings.TrimSpace(request.ResourceType)
	if resourceType == "" {
		resourceType = "issue"
	}
	if resourceType != "issue" {
		return contract.IssueLabel{}, ErrIssueCatalogInvalid
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.IssueLabel{}, fmt.Errorf("generate Issue label id: %w", err)
	}
	now := u.timestamp()
	return u.repository.CreateLabel(ctx, contract.IssueLabel{ID: id, WorkspaceID: strings.TrimSpace(request.WorkspaceID), ResourceType: resourceType, Name: name, Description: cleanCatalogText(request.Description), Color: color, CreatedAt: now, UpdatedAt: now})
}

func (u *IssueCatalogUseCase) UpdateIssueLabel(ctx context.Context, request contract.UpdateIssueLabelRequest) (contract.IssueLabel, error) {
	if err := u.authorize(ctx, request.WorkspaceID, PermissionLabelWrite); err != nil {
		return contract.IssueLabel{}, err
	}
	value, err := u.repository.GetLabel(ctx, strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.LabelID))
	if err != nil {
		return contract.IssueLabel{}, err
	}
	if request.Name != nil {
		value.Name, err = validateCatalogName(*request.Name)
		if err != nil {
			return contract.IssueLabel{}, err
		}
	}
	if request.Description != nil {
		value.Description = cleanCatalogText(*request.Description)
	}
	if request.Color != nil {
		value.Color, err = normalizeCatalogColor(*request.Color)
		if err != nil {
			return contract.IssueLabel{}, err
		}
	}
	value.UpdatedAt = u.timestamp()
	return u.repository.UpdateLabel(ctx, value)
}

func (u *IssueCatalogUseCase) DeleteIssueLabel(ctx context.Context, workspaceID, labelID string) error {
	if err := u.authorize(ctx, workspaceID, PermissionLabelWrite); err != nil {
		return err
	}
	return u.repository.DeleteLabel(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(labelID))
}

func (u *IssueCatalogUseCase) ListLabelsForIssue(ctx context.Context, workspaceID, issueID string) (string, []contract.IssueLabel, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionLabelList)
	if err != nil {
		return "", nil, err
	}
	values, err := u.repository.ListIssueLabels(ctx, strings.TrimSpace(workspaceID), resolved)
	if values == nil {
		values = []contract.IssueLabel{}
	}
	return resolved, values, err
}

func (u *IssueCatalogUseCase) AttachLabelToIssue(ctx context.Context, workspaceID, issueID, labelID string) (string, []contract.IssueLabel, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionLabelWrite)
	if err != nil {
		return "", nil, err
	}
	values, err := u.repository.AttachIssueLabel(ctx, strings.TrimSpace(workspaceID), resolved, strings.TrimSpace(labelID), u.timestamp())
	return resolved, values, err
}

func (u *IssueCatalogUseCase) DetachLabelFromIssue(ctx context.Context, workspaceID, issueID, labelID string) (string, []contract.IssueLabel, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionLabelWrite)
	if err != nil {
		return "", nil, err
	}
	values, err := u.repository.DetachIssueLabel(ctx, strings.TrimSpace(workspaceID), resolved, strings.TrimSpace(labelID))
	return resolved, values, err
}

func (u *IssueCatalogUseCase) ListIssueProperties(ctx context.Context, workspaceID string, includeArchived bool) ([]contract.IssuePropertyDefinition, error) {
	if err := u.authorize(ctx, workspaceID, PermissionPropertyList); err != nil {
		return nil, err
	}
	values, err := u.repository.ListProperties(ctx, strings.TrimSpace(workspaceID), includeArchived)
	if values == nil {
		values = []contract.IssuePropertyDefinition{}
	}
	return values, err
}

func (u *IssueCatalogUseCase) GetIssueProperty(ctx context.Context, workspaceID, propertyID string) (contract.IssuePropertyDefinition, error) {
	if err := u.authorize(ctx, workspaceID, PermissionPropertyList); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	return u.repository.GetProperty(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(propertyID))
}

func (u *IssueCatalogUseCase) CreateIssueProperty(ctx context.Context, request contract.CreateIssuePropertyRequest) (contract.IssuePropertyDefinition, error) {
	if err := u.requirePropertyAdmin(ctx, request.WorkspaceID); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	name, err := validatePropertyName(request.Name)
	if err != nil || !validPropertyType(request.Type) {
		return contract.IssuePropertyDefinition{}, ErrIssueCatalogInvalid
	}
	icon, err := validatePropertyIcon(request.Icon)
	if err != nil || utf8.RuneCountInString(request.Description) > 500 {
		return contract.IssuePropertyDefinition{}, ErrIssueCatalogInvalid
	}
	config, err := u.normalizePropertyConfig(ctx, request.Type, request.Config)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.IssuePropertyDefinition{}, fmt.Errorf("generate Issue property id: %w", err)
	}
	now := u.timestamp()
	return u.repository.CreateProperty(ctx, contract.IssuePropertyDefinition{ID: id, WorkspaceID: strings.TrimSpace(request.WorkspaceID), Name: name, Type: request.Type, Description: cleanCatalogText(request.Description), Icon: icon, Config: config, CreatedAt: now, UpdatedAt: now})
}

func (u *IssueCatalogUseCase) UpdateIssueProperty(ctx context.Context, request contract.UpdateIssuePropertyRequest) (contract.IssuePropertyDefinition, error) {
	if err := u.requirePropertyAdmin(ctx, request.WorkspaceID); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	value, err := u.repository.GetProperty(ctx, strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.PropertyID))
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	if request.Name != nil {
		value.Name, err = validatePropertyName(*request.Name)
		if err != nil {
			return contract.IssuePropertyDefinition{}, err
		}
	}
	if request.Description != nil {
		if utf8.RuneCountInString(*request.Description) > 500 {
			return contract.IssuePropertyDefinition{}, ErrIssueCatalogInvalid
		}
		value.Description = cleanCatalogText(*request.Description)
	}
	if request.Icon != nil {
		value.Icon, err = validatePropertyIcon(*request.Icon)
		if err != nil {
			return contract.IssuePropertyDefinition{}, err
		}
	}
	removed := []string{}
	if request.Config != nil {
		config, configErr := u.normalizePropertyConfig(ctx, value.Type, request.Config)
		if configErr != nil {
			return contract.IssuePropertyDefinition{}, configErr
		}
		removed = removedPropertyOptions(value.Config, config)
		value.Config = config
	}
	if request.Archived != nil {
		value.Archived = *request.Archived
		if *request.Archived {
			now := u.timestamp()
			value.ArchivedAt = &now
		} else {
			value.ArchivedAt = nil
		}
	}
	value.UpdatedAt = u.timestamp()
	return u.repository.UpdateProperty(ctx, value, removed)
}

func (u *IssueCatalogUseCase) SetIssueProperty(ctx context.Context, workspaceID, issueID, propertyID string, raw json.RawMessage) (string, map[string]any, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionPropertyWrite)
	if err != nil {
		return "", nil, err
	}
	return u.repository.SetIssueProperty(ctx, strings.TrimSpace(workspaceID), resolved, strings.TrimSpace(propertyID), raw, u.timestamp())
}

func (u *IssueCatalogUseCase) UnsetIssueProperty(ctx context.Context, workspaceID, issueID, propertyID string) (string, map[string]any, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionPropertyWrite)
	if err != nil {
		return "", nil, err
	}
	return u.repository.UnsetIssueProperty(ctx, strings.TrimSpace(workspaceID), resolved, strings.TrimSpace(propertyID), u.timestamp())
}

func (u *IssueCatalogUseCase) ListAcceptanceConclusions(ctx context.Context, workspaceID, issueID string) (string, []contract.AcceptanceConclusion, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionAcceptanceList)
	if err != nil {
		return "", nil, err
	}
	values, err := u.repository.ListAcceptanceConclusions(ctx, strings.TrimSpace(workspaceID), resolved)
	if values == nil {
		values = []contract.AcceptanceConclusion{}
	}
	return resolved, values, err
}

func (u *IssueCatalogUseCase) CreateAcceptanceConclusion(ctx context.Context, workspaceID, issueID string, input contract.AcceptanceConclusionInput) (contract.AcceptanceConclusionMutation, error) {
	return u.acceptance(ctx, workspaceID, issueID, input, false)
}

func (u *IssueCatalogUseCase) CompleteIssueWithAcceptance(ctx context.Context, workspaceID, issueID string, input contract.AcceptanceConclusionInput) (contract.AcceptanceConclusionMutation, error) {
	return u.acceptance(ctx, workspaceID, issueID, input, true)
}

func (u *IssueCatalogUseCase) acceptance(ctx context.Context, workspaceID, issueID string, input contract.AcceptanceConclusionInput, complete bool) (contract.AcceptanceConclusionMutation, error) {
	resolved, err := u.issue(ctx, workspaceID, issueID, PermissionAcceptanceWrite)
	if err != nil {
		return contract.AcceptanceConclusionMutation{}, err
	}
	input.Result, input.Rationale = strings.TrimSpace(input.Result), strings.TrimSpace(input.Rationale)
	if input.Result != "accepted" && input.Result != "conditional" && input.Result != "rejected" || input.Rationale == "" {
		return contract.AcceptanceConclusionMutation{}, ErrIssueCatalogInvalid
	}
	refs := make([]string, 0, len(input.EvidenceRefs))
	for _, ref := range input.EvidenceRefs {
		if value := strings.TrimSpace(ref); value != "" {
			refs = append(refs, value)
		}
	}
	actor, err := u.actor(ctx, strings.TrimSpace(workspaceID))
	if err != nil {
		return contract.AcceptanceConclusionMutation{}, err
	}
	id, err := u.newID(ctx)
	if err != nil {
		return contract.AcceptanceConclusionMutation{}, fmt.Errorf("generate acceptance conclusion id: %w", err)
	}
	now := u.timestamp()
	conclusion := contract.AcceptanceConclusion{ID: id, WorkspaceID: strings.TrimSpace(workspaceID), IssueID: resolved, Result: input.Result, Rationale: input.Rationale, EvidenceRefs: refs, ActorID: actor.ID, CreatedAt: now, UpdatedAt: now}
	issue, created, err := u.repository.CreateAcceptanceConclusion(ctx, strings.TrimSpace(workspaceID), resolved, conclusion, complete, now)
	if err != nil {
		return contract.AcceptanceConclusionMutation{}, err
	}
	publicIssue := issueToContract(issue)
	return contract.AcceptanceConclusionMutation{Conclusion: created, Issue: &publicIssue}, nil
}

func (u *IssueCatalogUseCase) authorize(ctx context.Context, workspaceID, permission string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return ErrIssueCatalogInvalid
	}
	if err := u.authorizer.AuthorizeWorkspace(ctx, workspaceID, permission); err != nil {
		return err
	}
	_, err := u.actor(ctx, workspaceID)
	return err
}

func (u *IssueCatalogUseCase) actor(ctx context.Context, workspaceID string) (contract.WorkspaceActor, error) {
	actor, ok := contract.WorkspaceActorFromContext(ctx)
	if !ok || (actor.Type != "member" && actor.Type != "agent") {
		return contract.WorkspaceActor{}, contract.ErrWorkspaceActorRequired
	}
	belongs, err := u.actors.ActorBelongsToWorkspace(ctx, workspaceID, actor.Type, actor.ID)
	if err != nil {
		return contract.WorkspaceActor{}, err
	}
	if !belongs {
		return contract.WorkspaceActor{}, contract.ErrActorOutsideWorkspace
	}
	return actor, nil
}

func (u *IssueCatalogUseCase) issue(ctx context.Context, workspaceID, issueID, permission string) (string, error) {
	workspaceID, issueID = strings.TrimSpace(workspaceID), strings.TrimSpace(issueID)
	if issueID == "" {
		return "", ErrIssueCatalogInvalid
	}
	if err := u.authorize(ctx, workspaceID, permission); err != nil {
		return "", err
	}
	return u.repository.ResolveIssue(ctx, workspaceID, issueID)
}

func (u *IssueCatalogUseCase) requirePropertyAdmin(ctx context.Context, workspaceID string) error {
	if err := u.authorize(ctx, workspaceID, PermissionPropertyWrite); err != nil {
		return err
	}
	actor, _ := contract.WorkspaceActorFromContext(ctx)
	if actor.Type != "member" {
		return contract.ErrWorkspacePermissionDenied
	}
	membership, found, err := u.memberships.FindForUserAndWorkspace(ctx, actor.ID, strings.TrimSpace(workspaceID))
	if err != nil {
		return err
	}
	if !found || (membership.Role != "owner" && membership.Role != "admin") {
		return contract.ErrWorkspacePermissionDenied
	}
	return nil
}

func (u *IssueCatalogUseCase) timestamp() string { return u.now().UTC().Format(time.RFC3339Nano) }

var catalogColorPattern = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)

func validateCatalogName(raw string) (string, error) {
	for _, value := range raw {
		if unicode.IsControl(value) {
			return "", ErrIssueCatalogInvalid
		}
	}
	value := strings.TrimSpace(raw)
	if value == "" || utf8.RuneCountInString(value) > 32 {
		return "", ErrIssueCatalogInvalid
	}
	return value, nil
}

func normalizeCatalogColor(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !catalogColorPattern.MatchString(value) {
		return "", ErrIssueCatalogInvalid
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	return strings.ToLower(value), nil
}

func cleanCatalogText(raw string) string {
	return strings.ReplaceAll(strings.TrimSpace(raw), "\x00", "")
}

var reservedPropertyNames = map[string]struct{}{
	"status": {}, "priority": {}, "assignee": {}, "project": {}, "parent": {}, "stage": {}, "label": {}, "labels": {},
	"start_date": {}, "due_date": {}, "title": {}, "description": {}, "creator": {}, "created_at": {}, "updated_at": {},
	"metadata": {}, "properties": {},
}

func validatePropertyName(raw string) (string, error) {
	value, err := validateCatalogName(raw)
	if err != nil {
		return "", err
	}
	if _, reserved := reservedPropertyNames[strings.ReplaceAll(strings.ToLower(value), " ", "_")]; reserved {
		return "", ErrIssueCatalogInvalid
	}
	return value, nil
}

func validPropertyType(value string) bool {
	switch value {
	case "text", "number", "select", "multi_select", "date", "checkbox", "url":
		return true
	default:
		return false
	}
}

var validPropertyIcons = map[string]struct{}{
	"": {}, "circle-dot": {}, "signal-high": {}, "user-round": {}, "folder-kanban": {}, "calendar-days": {}, "tag": {}, "milestone": {}, "flag": {}, "bookmark": {}, "star": {}, "target": {}, "shield": {}, "bug": {}, "zap": {}, "rocket": {}, "sparkles": {}, "lightbulb": {}, "globe-2": {}, "link": {}, "hash": {}, "list-checks": {}, "circle-check": {}, "clock-3": {}, "briefcase-business": {}, "layers-3": {}, "gauge": {}, "database": {}, "code-2": {}, "palette": {}, "megaphone": {}, "map-pin": {}, "package": {}, "wrench": {}, "heart": {}, "circle-alert": {}, "lock-keyhole": {},
}

func validatePropertyIcon(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if _, ok := validPropertyIcons[value]; !ok || utf8.RuneCountInString(value) > 32 {
		return "", ErrIssueCatalogInvalid
	}
	return value, nil
}

func (u *IssueCatalogUseCase) normalizePropertyConfig(ctx context.Context, propertyType string, config *contract.IssuePropertyConfig) (contract.IssuePropertyConfig, error) {
	if propertyType != "select" && propertyType != "multi_select" {
		if config != nil && len(config.Options) != 0 {
			return contract.IssuePropertyConfig{}, ErrIssueCatalogInvalid
		}
		return contract.IssuePropertyConfig{}, nil
	}
	if config == nil || len(config.Options) == 0 || len(config.Options) > 50 {
		return contract.IssuePropertyConfig{}, ErrIssueCatalogInvalid
	}
	result := contract.IssuePropertyConfig{Options: make([]contract.IssuePropertyOption, 0, len(config.Options))}
	ids, names := map[string]struct{}{}, map[string]struct{}{}
	for _, option := range config.Options {
		name, err := validateCatalogName(option.Name)
		if err != nil {
			return contract.IssuePropertyConfig{}, err
		}
		lower := strings.ToLower(name)
		if _, exists := names[lower]; exists {
			return contract.IssuePropertyConfig{}, ErrIssueCatalogInvalid
		}
		names[lower] = struct{}{}
		color, err := normalizeCatalogColor(option.Color)
		if err != nil {
			return contract.IssuePropertyConfig{}, err
		}
		id := strings.TrimSpace(option.ID)
		if id == "" {
			id, err = u.newID(ctx)
			if err != nil {
				return contract.IssuePropertyConfig{}, fmt.Errorf("generate property option id: %w", err)
			}
		}
		if _, exists := ids[id]; exists {
			return contract.IssuePropertyConfig{}, ErrIssueCatalogInvalid
		}
		ids[id] = struct{}{}
		result.Options = append(result.Options, contract.IssuePropertyOption{ID: id, Name: name, Color: color})
	}
	return result, nil
}

func removedPropertyOptions(before, after contract.IssuePropertyConfig) []string {
	kept := make(map[string]struct{}, len(after.Options))
	for _, option := range after.Options {
		kept[option.ID] = struct{}{}
	}
	removed := make([]string, 0)
	for _, option := range before.Options {
		if _, ok := kept[option.ID]; !ok {
			removed = append(removed, option.ID)
		}
	}
	return removed
}

func ValidateIssuePropertyValue(definition contract.IssuePropertyDefinition, raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, ErrIssueCatalogInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) || value == nil {
		return nil, ErrIssueCatalogInvalid
	}
	switch definition.Type {
	case "text":
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > 2000 {
			return nil, ErrIssueCatalogInvalid
		}
		return strings.ReplaceAll(text, "\x00", ""), nil
	case "url":
		text, ok := value.(string)
		parsed, err := url.Parse(strings.TrimSpace(text))
		if !ok || err != nil || len(text) > 2048 || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, ErrIssueCatalogInvalid
		}
		return strings.TrimSpace(text), nil
	case "number":
		if _, ok := value.(json.Number); !ok {
			return nil, ErrIssueCatalogInvalid
		}
		return value, nil
	case "checkbox":
		if _, ok := value.(bool); !ok {
			return nil, ErrIssueCatalogInvalid
		}
		return value, nil
	case "date":
		text, ok := value.(string)
		if !ok {
			return nil, ErrIssueCatalogInvalid
		}
		if _, err := time.Parse("2006-01-02", text); err != nil {
			return nil, ErrIssueCatalogInvalid
		}
		return text, nil
	case "select":
		text, ok := value.(string)
		if !ok || propertyOptionIndex(definition.Config, text) < 0 {
			return nil, ErrIssueCatalogInvalid
		}
		return text, nil
	case "multi_select":
		items, ok := value.([]any)
		if !ok || len(items) == 0 {
			return nil, ErrIssueCatalogInvalid
		}
		seen := map[string]struct{}{}
		values := make([]string, 0, len(items))
		for _, rawItem := range items {
			item, ok := rawItem.(string)
			if !ok || propertyOptionIndex(definition.Config, item) < 0 {
				return nil, ErrIssueCatalogInvalid
			}
			if _, duplicate := seen[item]; !duplicate {
				seen[item] = struct{}{}
				values = append(values, item)
			}
		}
		sort.SliceStable(values, func(left, right int) bool {
			return propertyOptionIndex(definition.Config, values[left]) < propertyOptionIndex(definition.Config, values[right])
		})
		return values, nil
	default:
		return nil, ErrIssueCatalogInvalid
	}
}

func propertyOptionIndex(config contract.IssuePropertyConfig, id string) int {
	for index, option := range config.Options {
		if option.ID == id {
			return index
		}
	}
	return -1
}

var _ contract.IssueCatalogService = (*IssueCatalogUseCase)(nil)
