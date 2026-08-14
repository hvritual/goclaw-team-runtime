package workspace

import (
	"fmt"
	"net/http"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type HTTPUserIDResolver func(*http.Request) (string, error)

type workspaceSelectionExtension struct {
	handler *workspacehttp.WorkspaceSelectionHandler
}

func (e *workspaceSelectionExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
}
func (e *workspaceSelectionExtension) RegisterGRPC(grpc.ServiceRegistrar) {}

func NewSqliteWorkspaceSelection(config SqlitePersistenceConfig, memberships contract.WorkspaceMembershipReader) (contract.WorkspaceSelectionService, error) {
	repository, err := persistence.NewWorkspaceSelectionRepository(config)
	if err != nil {
		return nil, err
	}
	service, err := application.NewWorkspaceSelectionUseCase(memberships, repository)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace selection: %w", err)
	}
	return service, nil
}

func newWorkspaceSelectionExtension(service contract.WorkspaceSelectionService, creator contract.WorkspaceCreationService, identity HTTPUserIDResolver, authorizeMutation func(*http.Request) error) *workspaceSelectionExtension {
	return &workspaceSelectionExtension{handler: workspacehttp.NewWorkspaceSelectionHandler(service, creator, workspacehttp.UserIDResolver(identity), authorizeMutation)}
}

func NewTrustedHTTPIdentityResolver(identity HTTPUserIDResolver, selection contract.WorkspaceSelectionService) contract.WorkspaceHTTPIdentityResolver {
	return func(request *http.Request) (contract.WorkspaceHTTPIdentity, error) {
		userID, err := identity(request)
		if err != nil {
			return contract.WorkspaceHTTPIdentity{}, err
		}
		workspaceID := strings.TrimSpace(request.Header.Get("X-Workspace-ID"))
		workspaceSlug := strings.TrimSpace(request.Header.Get("X-Workspace-Slug"))
		if workspaceSlug != "" {
			resolvedID, resolveErr := selection.ResolveSlug(request.Context(), userID, workspaceSlug)
			if resolveErr != nil {
				return contract.WorkspaceHTTPIdentity{}, resolveErr
			}
			if workspaceID != "" && workspaceID != resolvedID {
				return contract.WorkspaceHTTPIdentity{}, contract.ErrWorkspaceNotFound
			}
			workspaceID = resolvedID
		}
		membership, err := selection.MembershipForID(request.Context(), userID, workspaceID)
		if err != nil {
			return contract.WorkspaceHTTPIdentity{}, err
		}
		return contract.WorkspaceHTTPIdentity{WorkspaceID: workspaceID, ActorType: "member", ActorID: membership.UserID}, nil
	}
}
