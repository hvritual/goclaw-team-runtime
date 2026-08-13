package workspace

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	httpadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	protoadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/proto"
	workspacev1 "github.com/hvritual/workspace/rpc/pb/workspace/v1"
	stdgrpc "google.golang.org/grpc"
)

// IssueMetadataExtension is user-owned because the frozen dddgen binary is unavailable.
// It is installed only by the explicit SQLite composition.
type IssueMetadataExtension struct {
	local  contract.IssueMetadataService
	server *protoadapter.IssueMetadataServer
	http   *httpadapter.IssueMetadataHandler
}

func newIssueMetadataExtension(service contract.IssueMetadataService, identity contract.WorkspaceHTTPIdentityResolver, authenticate func(*http.Request) (string, error), mutation func(*http.Request) error) *IssueMetadataExtension {
	return &IssueMetadataExtension{local: service, server: protoadapter.NewIssueMetadataServer(service), http: httpadapter.NewIssueMetadataHandler(service, identity, authenticate, mutation)}
}

func (e *IssueMetadataExtension) Local() contract.IssueMetadataService   { return e.local }
func (e *IssueMetadataExtension) RegisterHTTP(server *kratoshttp.Server) { e.http.Register(server) }
func (e *IssueMetadataExtension) RegisterGRPC(registrar stdgrpc.ServiceRegistrar) {
	workspacev1.RegisterIssueMetadataServiceServer(registrar, e.server)
}

func (m *Module) IssueMetadataLocal() contract.IssueMetadataService {
	for _, extension := range m.extensions {
		if typed, ok := extension.(*IssueMetadataExtension); ok {
			return typed.Local()
		}
	}
	panic("Issue metadata extension is not registered")
}
