package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/space/contract"
)

const MaxAttachmentSize int64 = 100 << 20

type StoredAttachment struct {
	ID, WorkspaceID, VersionID, UploaderType, UploaderID string
	Filename, ObjectKey, ContentType, Checksum           string
	SizeBytes                                            int64
	CreatedAt                                            time.Time
	Binding                                              contract.AttachmentBinding
}

type AttachmentRepository interface {
	Create(context.Context, StoredAttachment, func(contract.AttachmentExecutor) error) error
	FindByID(context.Context, string) (StoredAttachment, error)
	FindMany(context.Context, []string) ([]StoredAttachment, error)
	Delete(context.Context, StoredAttachment, func(contract.AttachmentExecutor) error) error
	ObjectKeys(context.Context) ([]string, error)
	FindForCleanup(context.Context, contract.AttachmentExecutor, string, []string) ([]StoredAttachment, error)
	DeleteForCleanup(context.Context, contract.AttachmentExecutor, string, []StoredAttachment) error
}

type AttachmentObjectStore interface {
	Put(context.Context, string, []byte) error
	Open(context.Context, string) (io.ReadCloser, error)
	Quarantine(context.Context, string) (string, error)
	Restore(context.Context, string, string) error
	Remove(context.Context, string) error
	Reconcile(context.Context, []string) error
}

type AttachmentService struct {
	repository AttachmentRepository
	objects    AttachmentObjectStore
	relations  contract.AttachmentRelations
	newID      func() (string, error)
	now        func() time.Time
}

func NewAttachmentService(repository AttachmentRepository, objects AttachmentObjectStore, relations contract.AttachmentRelations) (*AttachmentService, error) {
	if repository == nil || objects == nil || relations == nil {
		return nil, errors.New("attachment repository, object store and relations are required")
	}
	return &AttachmentService{repository: repository, objects: objects, relations: relations, newID: func() (string, error) { return uuid.NewString(), nil }, now: time.Now}, nil
}

func (s *AttachmentService) SetIDGenerator(generator func() (string, error)) {
	if generator != nil {
		s.newID = generator
	}
}

func (s *AttachmentService) SetClock(clock func() time.Time) {
	if clock != nil {
		s.now = clock
	}
}

func (s *AttachmentService) Upload(ctx context.Context, request contract.UploadAttachmentRequest) (contract.Attachment, error) {
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.UploaderType = strings.TrimSpace(request.UploaderType)
	request.UploaderID = strings.TrimSpace(request.UploaderID)
	request.Filename = normalizeFilename(request.Filename)
	request.ContentType = strings.TrimSpace(request.ContentType)
	if request.WorkspaceID == "" || request.UploaderType != "member" || request.UploaderID == "" || request.Filename == "" || request.ContentType == "" || len(request.Content) == 0 {
		return contract.Attachment{}, contract.ErrAttachmentInvalid
	}
	if int64(len(request.Content)) > MaxAttachmentSize {
		return contract.Attachment{}, contract.ErrAttachmentTooLarge
	}
	binding, err := s.relations.ResolveBinding(ctx, request.WorkspaceID, request.IssueID, request.CommentID)
	if err != nil {
		return contract.Attachment{}, err
	}
	assetID, err := s.newID()
	if err != nil {
		return contract.Attachment{}, fmt.Errorf("generate attachment id: %w", err)
	}
	versionID, err := s.newID()
	if err != nil {
		return contract.Attachment{}, fmt.Errorf("generate attachment version id: %w", err)
	}
	createdAt := s.now().UTC()
	objectKey := request.WorkspaceID + "/" + assetID + "/" + versionID + ".blob"
	value := StoredAttachment{
		ID: assetID, WorkspaceID: request.WorkspaceID, VersionID: versionID,
		UploaderType: request.UploaderType, UploaderID: request.UploaderID,
		Filename: request.Filename, ObjectKey: objectKey, ContentType: request.ContentType,
		SizeBytes: int64(len(request.Content)), Checksum: checksum(request.Content), CreatedAt: createdAt,
		Binding: binding,
	}
	if err := s.objects.Put(ctx, objectKey, request.Content); err != nil {
		return contract.Attachment{}, fmt.Errorf("store attachment object: %w", err)
	}
	created := false
	defer func() {
		if !created {
			_ = s.objects.Remove(context.WithoutCancel(ctx), objectKey)
		}
	}()
	if err := s.repository.Create(ctx, value, func(executor contract.AttachmentExecutor) error {
		return s.relations.Bind(ctx, executor, value.WorkspaceID, value.ID, binding)
	}); err != nil {
		return contract.Attachment{}, err
	}
	created = true
	return renderAttachment(value), nil
}

func (s *AttachmentService) Get(ctx context.Context, attachmentID string) (contract.Attachment, error) {
	value, err := s.repository.FindByID(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return contract.Attachment{}, err
	}
	value.Binding, err = s.relations.Locate(ctx, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.Attachment{}, err
	}
	return renderAttachment(value), nil
}

func (s *AttachmentService) ListIssue(ctx context.Context, workspaceID, issueID string) ([]contract.Attachment, error) {
	ids, err := s.relations.ListIssueAssetIDs(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(issueID))
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []contract.Attachment{}, nil
	}
	values, err := s.repository.FindMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]contract.Attachment, 0, len(values))
	for _, value := range values {
		if value.WorkspaceID != strings.TrimSpace(workspaceID) {
			return nil, contract.ErrAttachmentNotFound
		}
		value.Binding, err = s.relations.Locate(ctx, value.WorkspaceID, value.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, renderAttachment(value))
	}
	return result, nil
}

func (s *AttachmentService) Open(ctx context.Context, attachmentID string) (contract.Attachment, io.ReadCloser, error) {
	value, err := s.repository.FindByID(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return contract.Attachment{}, nil, err
	}
	value.Binding, err = s.relations.Locate(ctx, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.Attachment{}, nil, err
	}
	reader, err := s.objects.Open(ctx, value.ObjectKey)
	if err != nil {
		return contract.Attachment{}, nil, fmt.Errorf("open attachment object: %w", err)
	}
	return renderAttachment(value), reader, nil
}

func (s *AttachmentService) Delete(ctx context.Context, attachmentID string) error {
	value, err := s.repository.FindByID(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return err
	}
	tombstone, err := s.objects.Quarantine(ctx, value.ObjectKey)
	if err != nil {
		return fmt.Errorf("quarantine attachment object: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = s.objects.Restore(context.WithoutCancel(ctx), tombstone, value.ObjectKey)
		}
	}()
	if err := s.repository.Delete(ctx, value, func(executor contract.AttachmentExecutor) error {
		return s.relations.Unbind(ctx, executor, value.WorkspaceID, value.ID)
	}); err != nil {
		return err
	}
	committed = true
	_ = s.objects.Remove(context.WithoutCancel(ctx), tombstone)
	return nil
}

func (s *AttachmentService) PrepareDelete(ctx context.Context, executor contract.AttachmentExecutor, workspaceID string, rawIDs []string) (contract.AttachmentCleanup, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if executor == nil || workspaceID == "" {
		return nil, contract.ErrAttachmentInvalid
	}
	ids := make([]string, 0, len(rawIDs))
	seen := make(map[string]struct{}, len(rawIDs))
	for _, rawID := range rawIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return nil, contract.ErrAttachmentInvalid
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	values, err := s.repository.FindForCleanup(ctx, executor, workspaceID, ids)
	if err != nil {
		return nil, err
	}
	cleanup := &preparedAttachmentCleanup{objects: s.objects}
	for _, value := range values {
		tombstone, quarantineErr := s.objects.Quarantine(ctx, value.ObjectKey)
		if quarantineErr != nil {
			return nil, errors.Join(fmt.Errorf("quarantine attachment object: %w", quarantineErr), cleanup.Rollback(context.WithoutCancel(ctx)))
		}
		cleanup.values = append(cleanup.values, preparedAttachmentObject{key: value.ObjectKey, tombstone: tombstone})
	}
	if err := s.repository.DeleteForCleanup(ctx, executor, workspaceID, values); err != nil {
		return nil, errors.Join(err, cleanup.Rollback(context.WithoutCancel(ctx)))
	}
	return cleanup, nil
}

func (s *AttachmentService) AssetBelongsToWorkspace(ctx context.Context, workspaceID, attachmentID string) (bool, error) {
	value, err := s.repository.FindByID(ctx, strings.TrimSpace(attachmentID))
	if errors.Is(err, contract.ErrAttachmentNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value.WorkspaceID == strings.TrimSpace(workspaceID), nil
}

func (s *AttachmentService) Reconcile(ctx context.Context) error {
	keys, err := s.repository.ObjectKeys(ctx)
	if err != nil {
		return err
	}
	return s.objects.Reconcile(ctx, keys)
}

func renderAttachment(value StoredAttachment) contract.Attachment {
	download := "/api/attachments/" + value.ID + "/download"
	return contract.Attachment{
		ID: value.ID, WorkspaceID: value.WorkspaceID, IssueID: value.Binding.IssueID,
		CommentID: value.Binding.CommentID, UploaderType: value.UploaderType,
		UploaderID: value.UploaderID, Filename: value.Filename, URL: download,
		DownloadURL: download, MarkdownURL: download, ContentType: value.ContentType,
		SizeBytes: value.SizeBytes, CreatedAt: value.CreatedAt.Format(time.RFC3339Nano),
	}
}

func normalizeFilename(raw string) string {
	value := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	parts := strings.Split(value, "/")
	value = strings.Map(func(value rune) rune {
		if unicode.IsControl(value) {
			return -1
		}
		return value
	}, parts[len(parts)-1])
	value = strings.TrimSpace(value)
	for len(value) > 255 {
		_, size := utf8.DecodeLastRuneInString(value)
		value = value[:len(value)-size]
	}
	return value
}

func checksum(content []byte) string {
	value := sha256.Sum256(content)
	return hex.EncodeToString(value[:])
}

type preparedAttachmentObject struct{ key, tombstone string }

type preparedAttachmentCleanup struct {
	objects AttachmentObjectStore
	values  []preparedAttachmentObject
}

func (c *preparedAttachmentCleanup) Commit(ctx context.Context) {
	for _, value := range c.values {
		_ = c.objects.Remove(context.WithoutCancel(ctx), value.tombstone)
	}
}

func (c *preparedAttachmentCleanup) Rollback(ctx context.Context) error {
	var result error
	for index := len(c.values) - 1; index >= 0; index-- {
		value := c.values[index]
		if err := c.objects.Restore(context.WithoutCancel(ctx), value.tombstone, value.key); err != nil {
			result = errors.Join(result, err)
		}
	}
	c.values = nil
	return result
}
