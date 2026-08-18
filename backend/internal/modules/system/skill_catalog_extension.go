package system

import (
	"database/sql"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/system/contract"
	"github.com/hvritual/workspace/internal/modules/system/internal/application"
	sqliteinfra "github.com/hvritual/workspace/internal/modules/system/internal/infrastructure/sqlite"
	httpadapter "github.com/hvritual/workspace/internal/modules/system/internal/interfaces/http"
	"google.golang.org/grpc"
)

type SkillCatalogDependencies struct {
	Identity         contract.SkillIdentityResolver
	Mutation         contract.SkillMutationAuthorizer
	Authorize        contract.SkillAccessAuthorizer
	Preflight        contract.SkillVisibilityPreflight
	Bind             contract.SkillVisibilityBinder
	Resolve          contract.SkillVisibilityResolver
	List             contract.SkillVisibilityLister
	Objects          spacecontract.SkillObjectService
	FetchSkillSource application.SkillSourceFetcher
}

type skillCatalogExtension struct {
	handler       *httpadapter.SkillCatalogHandler
	fileHandler   *httpadapter.SkillFileHandler
	importHandler *httpadapter.SkillImportHandler
}

func (e *skillCatalogExtension) RegisterHTTP(server *kratoshttp.Server) {
	e.handler.Register(server)
	if e.fileHandler != nil {
		e.fileHandler.Register(server)
	}
	if e.importHandler != nil {
		e.importHandler.Register(server)
	}
}
func (*skillCatalogExtension) RegisterGRPC(grpc.ServiceRegistrar) {}

func NewWithSQLiteSkillCatalog(db *sql.DB, dependencies SkillCatalogDependencies) (*Module, error) {
	repository := sqliteinfra.NewSkillCatalogRepository(db)
	service := application.NewSkillCatalog(repository, dependencies.Authorize, dependencies.Preflight, dependencies.Bind, dependencies.Resolve, dependencies.List)
	module := New()
	extension := &skillCatalogExtension{handler: httpadapter.NewSkillCatalogHandler(service, dependencies.Identity, dependencies.Mutation)}
	if dependencies.Objects != nil {
		files := application.NewSkillFiles(repository, service, dependencies.Objects, dependencies.Authorize)
		extension.fileHandler = httpadapter.NewSkillFileHandler(files, dependencies.Identity, dependencies.Mutation)
		importer := application.NewSkillImporter(repository, dependencies.Objects, dependencies.Authorize, dependencies.Preflight, dependencies.Bind, dependencies.FetchSkillSource)
		extension.importHandler = httpadapter.NewSkillImportHandler(importer, dependencies.Identity, dependencies.Mutation)
	}
	module.extensions = append(module.extensions, extension)
	return module, nil
}
