package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	authcontract "github.com/hvritual/workspace/internal/modules/auth/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/local"
	protoadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/proto"
)

// WorkspaceServiceDependencies are explicit cross-boundary capabilities for
// migrated Workspace services. Each opt-in constructor validates the subset it
// requires; the default module does not invent permissive implementations.
type WorkspaceServiceDependencies struct {
	Authorizer              contract.WorkspaceAccessAuthorizer
	Actors                  contract.WorkspaceActorReader
	Assets                  contract.WorkspaceAssetReader
	Skills                  contract.SkillReferenceReader
	NewProjectID            func(context.Context) (string, error)
	NewTodoID               func(context.Context) (string, error)
	NewIssueID              func(context.Context) (string, error)
	NewKnowledgeID          func(context.Context) (string, error)
	NewRequirementID        func(context.Context) (string, error)
	NewRequirementVersionID func(context.Context) (string, error)
	Now                     func() time.Time
	HTTPIdentity            contract.WorkspaceHTTPIdentityResolver
	HTTPMutationAuthorizer  func(*http.Request) error
	IssueMetadataEnabled    *bool
	IssueCreateEnabled      *bool
	Events                  contract.WorkspaceEventPublisher
	Selection               contract.WorkspaceSelectionService
	WorkspaceMemberships    contract.WorkspaceMembershipReader
	HTTPUserIdentity        HTTPUserIDResolver
	WorkspaceOwnerWriter    authcontract.SQLiteWorkspaceOwnerWriter
	NewWorkspaceID          func(context.Context) (string, error)
	NewWorkspaceMemberID    func(context.Context) (string, error)
}

// NewWithSqliteWorkspaceServices explicitly selects the migrated Project and
// Relationship implementations. Other generated Workspace extensions remain
// unchanged.
func NewWithSqliteWorkspaceServices(config SqlitePersistenceConfig, dependencies WorkspaceServiceDependencies) (*Module, error) {
	if dependencies.Authorizer == nil {
		return nil, errors.New("workspace access authorizer is required")
	}
	if dependencies.Actors == nil {
		return nil, errors.New("workspace actor reader is required")
	}
	module, err := NewWithSqlitePersistence(config)
	if err != nil {
		return nil, err
	}
	projects, err := persistence.NewProjectRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Project SQLite persistence: %w", err)
	}
	relations, err := persistence.NewProjectActorRelationRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Relationship SQLite persistence: %w", err)
	}
	newProjectID := dependencies.NewProjectID
	if newProjectID == nil {
		newProjectID = func(context.Context) (string, error) { return uuid.NewString(), nil }
	}
	now := dependencies.Now
	if now == nil {
		now = time.Now
	}
	projectService, err := application.NewProjectUseCase(
		projects,
		dependencies.Authorizer,
		application.ProjectIDGenerator(newProjectID),
		application.Clock(now),
	)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Project application: %w", err)
	}
	relationshipService, err := application.NewRelationshipUseCase(projects, relations, dependencies.Authorizer, dependencies.Actors)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Relationship application: %w", err)
	}
	projectExtension := newProjectExtension(projectService)
	relationshipExtension := newRelationshipExtension(relationshipService)
	projectReplaced := false
	relationshipReplaced := false
	for index, extension := range module.extensions {
		switch extension.(type) {
		case *ProjectExtension:
			module.extensions[index] = projectExtension
			projectReplaced = true
		case *RelationshipExtension:
			module.extensions[index] = relationshipExtension
			relationshipReplaced = true
		}
	}
	if !projectReplaced || !relationshipReplaced {
		return nil, fmt.Errorf("Workspace generated extensions missing: project=%t relationship=%t", projectReplaced, relationshipReplaced)
	}
	return module, nil
}

func newProjectExtension(service contract.ProjectService) *ProjectExtension {
	client := local.NewProject(service)
	return &ProjectExtension{local: client, server: protoadapter.NewProjectServer(client)}
}

func newRelationshipExtension(service contract.RelationshipService) *RelationshipExtension {
	client := local.NewRelationship(service)
	return &RelationshipExtension{local: client, server: protoadapter.NewRelationshipServer(client)}
}
