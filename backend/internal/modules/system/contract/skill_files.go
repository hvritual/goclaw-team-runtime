package contract

import (
	"context"
	"io"
	"time"
)

type SkillFileBody struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	Content   []byte `json:"-"`
	Checksum  string `json:"checksum"`
	SizeBytes int64  `json:"size_bytes"`
}

type SkillImportPreview struct {
	Token       string          `json:"preview_token"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Warnings    []string        `json:"warnings"`
	Checksum    string          `json:"checksum"`
	TotalBytes  int64           `json:"total_bytes"`
	Files       []SkillFileBody `json:"files"`
}

type SkillImportPreviewRecord struct {
	TokenHash        string
	WorkspaceID      string
	ActorID          string
	ValidatorVersion string
	SourceChecksum   string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

type ImportSkillRequest struct {
	Identity         SkillIdentity
	Name             string
	Description      string
	SourceChecksum   string
	PreviewTokenHash string
	ConflictMode     string
	ExpectedRevision int64
	IdempotencyKey   string
	RequestHash      string
	SkillID          string
	VersionID        string
	Files            []SkillFileManifest
}

type SkillFileManifest struct {
	ID            string `json:"id"`
	SkillID       string `json:"skill_id"`
	VersionID     string `json:"version_id"`
	Path          string `json:"path"`
	SpaceObjectID string `json:"space_object_id"`
	MediaType     string `json:"media_type"`
	SizeBytes     int64  `json:"size_bytes"`
	Checksum      string `json:"checksum"`
	CreatedAt     string `json:"created_at"`
}

type SkillFileMutation struct {
	Path             string
	Delete           bool
	ExpectedRevision int64
	Object           *SkillFileManifest
}

type SkillObjectPromoter func(context.Context, SkillCreateExecutor, string) error

type SkillFileRepository interface {
	ListFiles(context.Context, SkillIdentity, string, string) ([]SkillFileManifest, error)
	CreateFileVersion(context.Context, SkillIdentity, string, SkillFileMutation, string, time.Time, SkillObjectPromoter) (SkillCatalogEntry, error)
}

type SkillImportRepository interface {
	SavePreview(context.Context, SkillImportPreviewRecord) error
	GetPreview(context.Context, string) (SkillImportPreviewRecord, error)
	DiscardPreview(context.Context, string) error
	FindImportResult(context.Context, string, string, string) (SkillCatalogEntry, bool, error)
	Import(context.Context, ImportSkillRequest, time.Time, SkillCreateBinding, SkillObjectPromoter) (SkillCatalogEntry, error)
}

type SkillImportService interface {
	PreviewArchive(context.Context, SkillIdentity, []byte) (SkillImportPreview, error)
	PreviewURL(context.Context, SkillIdentity, string) (SkillImportPreview, error)
	ImportArchive(context.Context, SkillIdentity, []byte, string, string, int64, string) (SkillCatalogEntry, error)
	ImportURL(context.Context, SkillIdentity, string, string, string, int64, string) (SkillCatalogEntry, error)
}

type SkillFileContent struct {
	SkillFileManifest
	Content string `json:"content"`
}

type SkillFileService interface {
	List(context.Context, SkillIdentity, string, string) ([]SkillFileManifest, error)
	Read(context.Context, SkillIdentity, string, string, string) (SkillFileManifest, []byte, error)
	Mutate(context.Context, SkillIdentity, string, string, string, []byte, int64) (SkillCatalogEntry, error)
	Delete(context.Context, SkillIdentity, string, string, int64) (SkillCatalogEntry, error)
}

type SkillFileOpener interface {
	Open(context.Context, string, string) (SkillFileManifest, io.ReadCloser, error)
}
