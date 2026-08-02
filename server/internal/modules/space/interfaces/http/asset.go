package http

import (
	"io"
	"mime"
	stdhttp "net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
)

type AssetHandler struct {
	assets       contract.AssetService
	objects      contract.ObjectStore
	resolveActor ActorResolver
	urlPolicy    URLPolicy
}

func NewAssetHandler(
	assets contract.AssetService,
	objects contract.ObjectStore,
	actorResolver ActorResolver,
	policy URLPolicy,
) *AssetHandler {
	return &AssetHandler{assets: assets, objects: objects, resolveActor: actorResolver, urlPolicy: policy}
}

func (h *AssetHandler) Get(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	value, ok := h.load(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, stdhttp.StatusOK, h.urlPolicy.ResponseFor(value, nil))
}

func (h *AssetHandler) Download(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	value, ok := h.load(writer, request)
	if !ok {
		return
	}
	if h.objects == nil || !h.objects.Available() {
		writeError(writer, stdhttp.StatusServiceUnavailable, "storage not configured")
		return
	}
	reader, err := h.objects.GetReader(request.Context(), value.ObjectKey)
	if err != nil {
		writeError(writer, stdhttp.StatusNotFound, "attachment object not found")
		return
	}
	defer func() { _ = reader.Close() }()
	mediaType := value.MediaType
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	writer.Header().Set("Content-Type", mediaType)
	writer.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": value.Filename}))
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	if seeker, ok := reader.(io.ReadSeeker); ok {
		stdhttp.ServeContent(writer, request, value.Filename, time.Time{}, seeker)
		return
	}
	if size, err := strconv.ParseInt(value.SizeBytes, 10, 64); err == nil {
		writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	if _, err := io.Copy(writer, reader); err != nil {
		return
	}
}

func (h *AssetHandler) load(
	writer stdhttp.ResponseWriter,
	request *stdhttp.Request,
) (contract.Asset_Asset, bool) {
	if h.assets == nil {
		writeError(writer, stdhttp.StatusServiceUnavailable, "storage not configured")
		return contract.Asset_Asset{}, false
	}
	actorUserID := ""
	if h.resolveActor != nil {
		actorUserID = h.resolveActor(request)
	}
	if actorUserID == "" {
		writeError(writer, stdhttp.StatusUnauthorized, "user not authenticated")
		return contract.Asset_Asset{}, false
	}
	result, err := h.assets.GetAsset(
		contract.WithAssetActor(request.Context(), actorUserID),
		contract.Asset_GetAssetRequest{AssetId: chi.URLParam(request, "id")},
	)
	if err != nil || result.Asset == nil {
		writeError(writer, stdhttp.StatusNotFound, "attachment not found")
		return contract.Asset_Asset{}, false
	}
	return *result.Asset, true
}
