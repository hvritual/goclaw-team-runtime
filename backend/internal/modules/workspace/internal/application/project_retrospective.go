package application

import (
	"context"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	retrospectiveDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/retrospective"
)

type CreateProjectRetrospectiveCommand struct {
	WorkspaceID     string
	ProjectID       string
	RetrospectiveID string
	Content         retrospectiveDomain.Content
	Participants    []retrospectiveDomain.Participant
	IdempotencyKey  string
	RequestHash     string
	Actor           contract.WorkspaceActor
	OccurredAt      time.Time
}

type MutateProjectRetrospectiveCommand struct {
	WorkspaceID      string
	ProjectID        string
	RetrospectiveID  string
	ExpectedRevision int64
	Action           string
	Content          *retrospectiveDomain.Content
	Participants     *[]retrospectiveDomain.Participant
	RequestID        string
	Actor            contract.WorkspaceActor
	OccurredAt       time.Time
}

type ProjectRetrospectiveRepository interface {
	ReadProjectRetrospective(context.Context, string, string, string, contract.WorkspaceActor) (contract.ProjectRetrospective, error)
	CreateProjectRetrospective(context.Context, CreateProjectRetrospectiveCommand) (contract.ProjectRetrospective, error)
	MutateProjectRetrospective(context.Context, MutateProjectRetrospectiveCommand) (contract.ProjectRetrospective, error)
}
