// Package main composes the Multica HTTP server.
package main

import (
	"net/http"

	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/multica-ai/multica/server/internal/handler"
	"github.com/multica-ai/multica/server/internal/integration/uploadhttp"
	"github.com/multica-ai/multica/server/internal/issueguard"
	"github.com/multica-ai/multica/server/internal/issueguard/adapter/spaceacl"
	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/storage"
	spaceapp "github.com/multica-ai/multica/server/modules/space/application"
	spacechecksum "github.com/multica-ai/multica/server/modules/space/dependency/checksum"
	spaceid "github.com/multica-ai/multica/server/modules/space/dependency/id"
	spaceobjects "github.com/multica-ai/multica/server/modules/space/dependency/objectstorage"
	spacepostgres "github.com/multica-ai/multica/server/modules/space/dependency/postgres"
	spacehttp "github.com/multica-ai/multica/server/modules/space/interfaces/http"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func newSpaceUploadHandler(
	queries *db.Queries,
	store storage.Storage,
	cfSigner *auth.CloudFrontSigner,
	config handler.Config,
) *spacehttp.UploadHandler {
	spacePersistence := spacepostgres.New(queries)
	memberships := auth.NewWorkspaceMemberships(queries)
	uploadService := spaceapp.NewUploadService(
		spacePersistence,
		memberships,
		spaceobjects.New(store),
		spaceid.UUIDv7{},
		spacechecksum.SHA256{},
	)
	uploadWorkflow := issueguard.NewUploadWorkflow(
		spaceacl.NewProvider(uploadService),
		memberships,
		issueguard.NewAttachmentReferences(queries),
	)
	storageURLsArePublic := false
	if store != nil {
		storageURLsArePublic = store.CdnDomain() != ""
	}
	return spacehttp.NewUploadHandler(
		uploadhttp.New(uploadWorkflow),
		func(request *http.Request) string {
			return middleware.ResolveWorkspaceIDFromRequest(request, queries)
		},
		spacehttp.URLPolicy{
			PublicURL:            config.PublicURL,
			StorageURLsArePublic: storageURLsArePublic,
			Signer:               cfSigner,
			TTL:                  config.AttachmentDownloadURLTTL,
		},
	)
}
