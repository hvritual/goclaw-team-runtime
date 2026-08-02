package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

type publicObjectStore struct {
	objects map[string][]byte
}

func (s *publicObjectStore) Available() bool { return s != nil }
func (s *publicObjectStore) Upload(context.Context, string, []byte, string, string) (string, error) {
	return "", errors.New("not used")
}
func (s *publicObjectStore) DeleteObject(context.Context, string) error { return nil }
func (s *publicObjectStore) KeyFromURL(string) string                   { return "" }
func (s *publicObjectStore) GetReader(_ context.Context, key string) (io.ReadCloser, error) {
	value, ok := s.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(value)), nil
}

func TestPublicObjectHandlerServesOnlyGeneratedPersonalObjects(t *testing.T) {
	const key = "users/01980000-0000-7000-8000-000000000001/01980000-0000-7000-8000-000000000002.png"
	handler := NewPublicObjectHandler(&publicObjectStore{objects: map[string][]byte{
		key: {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
	}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), stdhttp.MethodGet, "/uploads/"+key, nil))
	if recorder.Code != stdhttp.StatusOK || !bytes.Equal(recorder.Body.Bytes(), []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.Bytes())
	}
	if recorder.Header().Get("Content-Type") != "image/png" || recorder.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("headers=%v", recorder.Header())
	}
}

func TestPublicObjectHandlerHidesWorkspaceAndUnsafeKeys(t *testing.T) {
	handler := NewPublicObjectHandler(&publicObjectStore{objects: map[string][]byte{}})
	for _, rawPath := range []string{
		"/uploads/workspaces/01980000-0000-7000-8000-000000000001/01980000-0000-7000-8000-000000000002.png",
		"/uploads/users/01980000-0000-7000-8000-000000000001/not-generated.png",
		"/uploads/users/01980000-0000-7000-8000-000000000001/../secret",
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), stdhttp.MethodGet, rawPath, nil))
		if recorder.Code != stdhttp.StatusNotFound {
			t.Fatalf("path=%q status=%d", rawPath, recorder.Code)
		}
	}
}
