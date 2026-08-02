// Package asset owns Space Asset identity, immutable content versions, and upload intent facts.
package asset

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid asset")

const UploaderMember = "member"

// UploadIntent records enough information before object I/O to recover from a
// crash or metadata failure without leaving an untracked object.
type UploadIntent struct {
	ID           string
	AssetID      string
	VersionID    string
	WorkspaceID  string
	UploaderType string
	UploaderID   string
	Filename     string
	ObjectKey    string
	MediaType    string
	SizeBytes    int64
	Checksum     string
	CreatedAt    time.Time
}

func NewUploadIntent(value UploadIntent) (UploadIntent, error) {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.AssetID) == "" ||
		strings.TrimSpace(value.VersionID) == "" || strings.TrimSpace(value.WorkspaceID) == "" ||
		strings.TrimSpace(value.UploaderType) == "" || strings.TrimSpace(value.UploaderID) == "" ||
		strings.TrimSpace(value.Filename) == "" || strings.TrimSpace(value.ObjectKey) == "" ||
		strings.TrimSpace(value.MediaType) == "" || value.SizeBytes < 0 ||
		strings.TrimSpace(value.Checksum) == "" || value.CreatedAt.IsZero() {
		return UploadIntent{}, ErrInvalid
	}
	value.CreatedAt = value.CreatedAt.UTC()
	return value, nil
}

// Asset is a workspace-isolated stored object whose current version is immutable.
type Asset struct {
	id               string
	workspaceID      string
	currentVersionID string
	uploaderType     string
	uploaderID       string
	filename         string
	objectKey        string
	url              string
	mediaType        string
	sizeBytes        int64
	checksum         string
	createdAt        time.Time
}

func Finalize(intent UploadIntent, rawURL string) (Asset, error) {
	if strings.TrimSpace(rawURL) == "" {
		return Asset{}, ErrInvalid
	}
	return Asset{
		id:               intent.AssetID,
		workspaceID:      intent.WorkspaceID,
		currentVersionID: intent.VersionID,
		uploaderType:     intent.UploaderType,
		uploaderID:       intent.UploaderID,
		filename:         intent.Filename,
		objectKey:        intent.ObjectKey,
		url:              rawURL,
		mediaType:        intent.MediaType,
		sizeBytes:        intent.SizeBytes,
		checksum:         intent.Checksum,
		createdAt:        intent.CreatedAt.UTC(),
	}, nil
}

type RehydrateParams struct {
	ID               string
	WorkspaceID      string
	CurrentVersionID string
	UploaderType     string
	UploaderID       string
	Filename         string
	ObjectKey        string
	URL              string
	MediaType        string
	SizeBytes        int64
	Checksum         string
	CreatedAt        time.Time
}

func Rehydrate(value RehydrateParams) (Asset, error) {
	intent, err := NewUploadIntent(UploadIntent{
		ID: "rehydrated", AssetID: value.ID, VersionID: value.CurrentVersionID,
		WorkspaceID: value.WorkspaceID, UploaderType: value.UploaderType,
		UploaderID: value.UploaderID, Filename: value.Filename, ObjectKey: value.ObjectKey,
		MediaType: value.MediaType, SizeBytes: value.SizeBytes, Checksum: value.Checksum,
		CreatedAt: value.CreatedAt,
	})
	if err != nil {
		return Asset{}, err
	}
	return Finalize(intent, value.URL)
}

func (a Asset) ID() string               { return a.id }
func (a Asset) WorkspaceID() string      { return a.workspaceID }
func (a Asset) CurrentVersionID() string { return a.currentVersionID }
func (a Asset) UploaderType() string     { return a.uploaderType }
func (a Asset) UploaderID() string       { return a.uploaderID }
func (a Asset) Filename() string         { return a.filename }
func (a Asset) ObjectKey() string        { return a.objectKey }
func (a Asset) URL() string              { return a.url }
func (a Asset) MediaType() string        { return a.mediaType }
func (a Asset) SizeBytes() int64         { return a.sizeBytes }
func (a Asset) Checksum() string         { return a.checksum }
func (a Asset) CreatedAt() time.Time     { return a.createdAt }
