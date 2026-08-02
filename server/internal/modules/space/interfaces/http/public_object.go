package http

import (
	"bufio"
	"io"
	"mime"
	stdhttp "net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/modules/space/contract"
)

// PublicObjectHandler preserves the installed direct-object URL contract used
// by avatars. Workspace assets are deliberately excluded: their access must go
// through AssetService so workspace membership remains authoritative.
type PublicObjectHandler struct {
	objects contract.ObjectStore
}

func NewPublicObjectHandler(objects contract.ObjectStore) *PublicObjectHandler {
	return &PublicObjectHandler{objects: objects}
}

func (h *PublicObjectHandler) ServeHTTP(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	key := strings.TrimPrefix(chi.URLParam(request, "*"), "/")
	if key == "" {
		key = strings.TrimPrefix(request.URL.Path, "/uploads/")
	}
	if !isPublicDirectObjectKey(key) || h.objects == nil || !h.objects.Available() {
		stdhttp.NotFound(writer, request)
		return
	}
	reader, err := h.objects.GetReader(request.Context(), key)
	if err != nil {
		stdhttp.NotFound(writer, request)
		return
	}
	defer func() { _ = reader.Close() }()

	buffered := bufio.NewReader(reader)
	prefix, _ := buffered.Peek(512)
	mediaType := mime.TypeByExtension(strings.ToLower(path.Ext(key)))
	if mediaType == "" {
		mediaType = stdhttp.DetectContentType(prefix)
	}
	writer.Header().Set("Content-Type", mediaType)
	writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; img-src 'self' data:; media-src 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'none'; form-action 'none'",
	)
	_, _ = io.Copy(writer, buffered)
}

func isPublicDirectObjectKey(key string) bool {
	if key == "" || path.Clean(key) != key || strings.Contains(key, "\\") {
		return false
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[0] != "users" {
		return false
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return false
	}
	filename := parts[2]
	extension := path.Ext(filename)
	objectID := strings.TrimSuffix(filename, extension)
	_, err := uuid.Parse(objectID)
	return err == nil
}
