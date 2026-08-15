package http

import (
	"bytes"
	"errors"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/hvritual/workspace/internal/modules/space/contract"
)

func TestDecodeMultipartRequestBoundsFileBeforeApplicationBuffering(t *testing.T) {
	request := multipartAttachmentRequest(t, []multipartPart{{name: "file", filename: "large.txt", content: bytes.Repeat([]byte("x"), 33)}})
	_, err := decodeMultipartRequest(httptest.NewRecorder(), request, 32)
	if !errors.Is(err, contract.ErrAttachmentTooLarge) {
		t.Fatalf("decode oversized multipart = %v", err)
	}
}

func TestDecodeMultipartRequestRejectsDuplicateAndUnknownFields(t *testing.T) {
	for _, test := range []struct {
		name  string
		parts []multipartPart
	}{
		{name: "duplicate file", parts: []multipartPart{{name: "file", filename: "one.txt", content: []byte("one")}, {name: "file", filename: "two.txt", content: []byte("two")}}},
		{name: "unknown field", parts: []multipartPart{{name: "file", filename: "one.txt", content: []byte("one")}, {name: "workspace_id", content: []byte("caller-owned")}}},
		{name: "duplicate target", parts: []multipartPart{{name: "file", filename: "one.txt", content: []byte("one")}, {name: "issue_id", content: []byte("one")}, {name: "issue_id", content: []byte("two")}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeMultipartRequest(httptest.NewRecorder(), multipartAttachmentRequest(t, test.parts), 1024)
			if !errors.Is(err, contract.ErrAttachmentInvalid) {
				t.Fatalf("decode multipart = %v", err)
			}
		})
	}
}

type multipartPart struct {
	name, filename string
	content        []byte
}

func multipartAttachmentRequest(t *testing.T, parts []multipartPart) *stdhttp.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, value := range parts {
		var partWriter interface{ Write([]byte) (int, error) }
		var err error
		if value.filename != "" {
			partWriter, err = writer.CreateFormFile(value.name, value.filename)
		} else {
			partWriter, err = writer.CreateFormField(value.name)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := partWriter.Write(value.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/upload-file", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
