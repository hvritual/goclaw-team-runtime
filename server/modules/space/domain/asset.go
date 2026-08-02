// Package domain owns workspace-isolated Asset identity and upload metadata.
package domain

import (
	"errors"
	"time"
)

// ErrInvalidAsset reports a violation of the persisted Asset invariant.
var ErrInvalidAsset = errors.New("invalid asset")

// UploadedAssetParams contains the facts required to construct an uploaded Asset.
type UploadedAssetParams struct {
	ID           string
	WorkspaceID  string
	UploaderType UploaderType
	UploaderID   string
	Filename     string
	URL          string
	ContentType  string
	SizeBytes    int64
	Checksum     string
	CreatedAt    time.Time
}

// UploaderType identifies the actor category recorded for an Asset upload.
type UploaderType string

const (
	// UploaderMember is a human workspace member.
	UploaderMember UploaderType = "member"
)

// Asset is a workspace-isolated stored object. Issue and comment attachment
// semantics remain owned by their consumer contexts and are not represented
// in this aggregate.
type Asset struct {
	id           string
	workspaceID  string
	uploaderType UploaderType
	uploaderID   string
	filename     string
	url          string
	contentType  string
	sizeBytes    int64
	checksum     string
	createdAt    time.Time
}

// NewUploadedAsset creates a workspace-scoped Asset after object storage succeeds.
func NewUploadedAsset(params UploadedAssetParams) (Asset, error) {
	if params.ID == "" || params.WorkspaceID == "" || params.UploaderID == "" || params.URL == "" || params.Checksum == "" || params.SizeBytes < 0 {
		return Asset{}, ErrInvalidAsset
	}
	if params.UploaderType != UploaderMember {
		return Asset{}, ErrInvalidAsset
	}

	return Asset{
		id:           params.ID,
		workspaceID:  params.WorkspaceID,
		uploaderType: params.UploaderType,
		uploaderID:   params.UploaderID,
		filename:     params.Filename,
		url:          params.URL,
		contentType:  params.ContentType,
		sizeBytes:    params.SizeBytes,
		checksum:     params.Checksum,
		createdAt:    params.CreatedAt,
	}, nil
}

// ID returns the stable Asset identity.
func (a Asset) ID() string { return a.id }

// WorkspaceID returns the owning workspace boundary.
func (a Asset) WorkspaceID() string { return a.workspaceID }

// UploaderType returns the actor category recorded for compatibility.
func (a Asset) UploaderType() UploaderType { return a.uploaderType }

// UploaderID returns the actor identity that uploaded the Asset.
func (a Asset) UploaderID() string { return a.uploaderID }

// Filename returns the original client filename.
func (a Asset) Filename() string { return a.filename }

// URL returns the storage-provider URL persisted with the Asset.
func (a Asset) URL() string { return a.url }

// ContentType returns the server-detected media type.
func (a Asset) ContentType() string { return a.contentType }

// SizeBytes returns the uploaded byte length.
func (a Asset) SizeBytes() int64 { return a.sizeBytes }

// Checksum returns the server-computed content digest. During S1a it is an
// upload-time fact; durable checksum persistence remains explicit S1 work.
func (a Asset) Checksum() string { return a.checksum }

// CreatedAt returns the persistence creation time.
func (a Asset) CreatedAt() time.Time { return a.createdAt }
