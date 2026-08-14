package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

var (
	ErrInvalidWorkspace      = errors.New("valid name and slug are required")
	ErrWorkspaceSlugConflict = errors.New("workspace slug already exists")
)

var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type WorkspaceCreationRepository interface {
	Create(context.Context, WorkspaceCreation) (contract.WorkspaceSelection, error)
}

type WorkspaceCreation struct {
	WorkspaceID string
	MemberID    string
	UserID      string
	Name        string
	Slug        string
	Description *string
	Context     *string
	IssuePrefix string
	CreatedAt   time.Time
}

type WorkspaceCreationUseCase struct {
	repository     WorkspaceCreationRepository
	newWorkspaceID func(context.Context) (string, error)
	newMemberID    func(context.Context) (string, error)
	now            func() time.Time
}

func NewWorkspaceCreationUseCase(repository WorkspaceCreationRepository, newWorkspaceID, newMemberID func(context.Context) (string, error), now func() time.Time) (*WorkspaceCreationUseCase, error) {
	if repository == nil {
		return nil, errors.New("workspace creation repository is required")
	}
	if newWorkspaceID == nil || newMemberID == nil {
		return nil, errors.New("workspace creation id generators are required")
	}
	if now == nil {
		return nil, errors.New("workspace creation clock is required")
	}
	return &WorkspaceCreationUseCase{repository: repository, newWorkspaceID: newWorkspaceID, newMemberID: newMemberID, now: now}, nil
}

func (u *WorkspaceCreationUseCase) Create(ctx context.Context, userID string, request contract.CreateWorkspaceRequest) (contract.WorkspaceSelection, error) {
	userID = strings.TrimSpace(userID)
	name := strings.TrimSpace(request.Name)
	slug := strings.ToLower(strings.TrimSpace(request.Slug))
	if userID == "" || name == "" || !workspaceSlugPattern.MatchString(slug) {
		return contract.WorkspaceSelection{}, ErrInvalidWorkspace
	}
	workspaceID, err := u.newWorkspaceID(ctx)
	if err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("generate workspace id: %w", err)
	}
	memberID, err := u.newMemberID(ctx)
	if err != nil {
		return contract.WorkspaceSelection{}, fmt.Errorf("generate workspace owner id: %w", err)
	}
	return u.repository.Create(ctx, WorkspaceCreation{
		WorkspaceID: workspaceID, MemberID: memberID, UserID: userID,
		Name: name, Slug: slug, Description: request.Description, Context: request.Context,
		IssuePrefix: workspaceIssuePrefix(name), CreatedAt: u.now().UTC(),
	})
}

func workspaceIssuePrefix(name string) string {
	var letters strings.Builder
	for _, character := range name {
		if character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' {
			letters.WriteRune(character)
		}
	}
	value := strings.ToUpper(letters.String())
	if value == "" {
		return "WS"
	}
	if len(value) > 3 {
		return value[:3]
	}
	return value
}
