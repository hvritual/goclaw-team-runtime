package system

import (
	"database/sql"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/system/contract"
	"github.com/hvritual/workspace/internal/modules/system/internal/application"
	sqliteinfra "github.com/hvritual/workspace/internal/modules/system/internal/infrastructure/sqlite"
	httpadapter "github.com/hvritual/workspace/internal/modules/system/internal/interfaces/http"
	"google.golang.org/grpc"
)

type SkillCatalogDependencies struct {
	Identity  contract.SkillIdentityResolver
	Mutation  contract.SkillMutationAuthorizer
	Authorize contract.SkillAccessAuthorizer
	Bind      contract.SkillVisibilityBinder
	Resolve   contract.SkillVisibilityResolver
	List      contract.SkillVisibilityLister
}

type skillCatalogExtension struct {
	handler *httpadapter.SkillCatalogHandler
}

func (e *skillCatalogExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (*skillCatalogExtension) RegisterGRPC(grpc.ServiceRegistrar)       {}

func NewWithSQLiteSkillCatalog(db *sql.DB, dependencies SkillCatalogDependencies) (*Module, error) {
	service := application.NewSkillCatalog(sqliteinfra.NewSkillCatalogRepository(db), dependencies.Authorize, dependencies.Bind, dependencies.Resolve, dependencies.List)
	module := New()
	module.extensions = append(module.extensions, &skillCatalogExtension{handler: httpadapter.NewSkillCatalogHandler(service, dependencies.Identity, dependencies.Mutation)})
	return module, nil
}
