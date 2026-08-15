package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/space/contract"
)

func TestAttachmentUploadUsesOpaqueKeyAndSanitizesDisplayFilename(t *testing.T) {
	repository := &attachmentRepositoryStub{}
	objects := &attachmentObjectsStub{}
	service, err := NewAttachmentService(repository, objects, attachmentRelationsStub{})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"asset-id", "version-id"}
	service.SetIDGenerator(func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	})
	service.SetClock(func() time.Time { return time.Date(2026, 8, 15, 1, 2, 3, 0, time.UTC) })

	value, err := service.Upload(t.Context(), contract.UploadAttachmentRequest{
		WorkspaceID: "workspace-id", UploaderType: "member", UploaderID: "user-id",
		Filename: `../../bad` + "\r\n" + `X-Test: injected.txt`, ContentType: "text/plain", Content: []byte("payload"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(value.Filename, "\r\n\x00") || value.Filename != "badX-Test: injected.txt" {
		t.Fatalf("sanitized filename = %q", value.Filename)
	}
	if repository.created.ObjectKey != "workspace-id/asset-id/version-id.blob" || objects.putKey != repository.created.ObjectKey || strings.Contains(objects.putKey, value.Filename) {
		t.Fatalf("opaque object key: created=%q put=%q filename=%q", repository.created.ObjectKey, objects.putKey, value.Filename)
	}
}

func TestAttachmentUploadCompensatesObjectWhenMetadataTransactionFails(t *testing.T) {
	repository := &attachmentRepositoryStub{createErr: errors.New("blocked")}
	objects := &attachmentObjectsStub{}
	service, err := NewAttachmentService(repository, objects, attachmentRelationsStub{})
	if err != nil {
		t.Fatal(err)
	}
	service.SetIDGenerator(func() (string, error) { return "generated", nil })
	_, err = service.Upload(t.Context(), contract.UploadAttachmentRequest{
		WorkspaceID: "workspace-id", UploaderType: "member", UploaderID: "user-id",
		Filename: "evidence.txt", ContentType: "text/plain", Content: []byte("payload"),
	})
	if err == nil || objects.removedKey == "" || objects.removedKey != objects.putKey {
		t.Fatalf("upload error=%v put=%q removed=%q", err, objects.putKey, objects.removedKey)
	}
}

func TestAttachmentDeleteTreatsPostCommitTombstoneRemovalAsDeferredCleanup(t *testing.T) {
	repository := &attachmentRepositoryStub{found: StoredAttachment{ID: "asset", WorkspaceID: "workspace", ObjectKey: "workspace/asset/version.blob"}}
	objects := &attachmentObjectsStub{removeErr: errors.New("temporary file lock")}
	service, err := NewAttachmentService(repository, objects, attachmentRelationsStub{})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(t.Context(), "asset"); err != nil {
		t.Fatalf("committed logical delete = %v", err)
	}
	if !repository.deleted || objects.removedKey != "workspace/asset/version.blob.deleting" {
		t.Fatalf("delete state: repository=%v tombstone=%q", repository.deleted, objects.removedKey)
	}
}

type attachmentRepositoryStub struct {
	created   StoredAttachment
	found     StoredAttachment
	createErr error
	deleted   bool
}

func (r *attachmentRepositoryStub) Create(_ context.Context, value StoredAttachment, _ func(contract.AttachmentExecutor) error) error {
	r.created = value
	return r.createErr
}
func (r *attachmentRepositoryStub) FindByID(context.Context, string) (StoredAttachment, error) {
	if r.found.ID == "" {
		return StoredAttachment{}, contract.ErrAttachmentNotFound
	}
	return r.found, nil
}
func (*attachmentRepositoryStub) FindMany(context.Context, []string) ([]StoredAttachment, error) {
	return nil, nil
}
func (r *attachmentRepositoryStub) Delete(_ context.Context, _ StoredAttachment, _ func(contract.AttachmentExecutor) error) error {
	r.deleted = true
	return nil
}
func (*attachmentRepositoryStub) ObjectKeys(context.Context) ([]string, error) { return nil, nil }
func (*attachmentRepositoryStub) FindForCleanup(context.Context, contract.AttachmentExecutor, string, []string) ([]StoredAttachment, error) {
	return nil, nil
}
func (*attachmentRepositoryStub) DeleteForCleanup(context.Context, contract.AttachmentExecutor, string, []StoredAttachment) error {
	return nil
}

type attachmentObjectsStub struct {
	putKey, removedKey string
	removeErr          error
}

func (o *attachmentObjectsStub) Put(_ context.Context, key string, _ []byte) error {
	o.putKey = key
	return nil
}
func (*attachmentObjectsStub) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}
func (*attachmentObjectsStub) Quarantine(_ context.Context, key string) (string, error) {
	return key + ".deleting", nil
}
func (*attachmentObjectsStub) Restore(context.Context, string, string) error { return nil }
func (o *attachmentObjectsStub) Remove(_ context.Context, key string) error {
	o.removedKey = key
	return o.removeErr
}
func (*attachmentObjectsStub) Reconcile(context.Context, []string) error { return nil }

type attachmentRelationsStub struct{}

func (attachmentRelationsStub) ResolveBinding(context.Context, string, *string, *string) (contract.AttachmentBinding, error) {
	return contract.AttachmentBinding{}, nil
}
func (attachmentRelationsStub) Bind(context.Context, contract.AttachmentExecutor, string, string, contract.AttachmentBinding) error {
	return nil
}
func (attachmentRelationsStub) Unbind(context.Context, contract.AttachmentExecutor, string, string) error {
	return nil
}
func (attachmentRelationsStub) Locate(context.Context, string, string) (contract.AttachmentBinding, error) {
	return contract.AttachmentBinding{}, nil
}
func (attachmentRelationsStub) ListIssueAssetIDs(context.Context, string, string) ([]string, error) {
	return nil, nil
}
