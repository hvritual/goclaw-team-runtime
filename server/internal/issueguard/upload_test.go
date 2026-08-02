package issueguard

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeSpaceUploads struct {
	sequence     *[]string
	available    bool
	uploadResult SpaceUploadResult
	prepared     PreparedAsset
	uploadCalls  int
	prepareCalls int
}

func (f *fakeSpaceUploads) Available() bool { return f.available }

func (f *fakeSpaceUploads) Upload(_ context.Context, _ SpaceUploadCommand) (SpaceUploadResult, error) {
	f.uploadCalls++
	*f.sequence = append(*f.sequence, "space.upload")
	return f.uploadResult, nil
}

func (f *fakeSpaceUploads) PrepareWorkspaceAsset(_ context.Context, _ SpaceUploadCommand) (PreparedAsset, error) {
	f.prepareCalls++
	*f.sequence = append(*f.sequence, "space.prepare")
	return f.prepared, nil
}

type fakeWorkspaceAccess struct {
	sequence *[]string
	allowed  bool
}

func (f fakeWorkspaceAccess) IsMember(_ context.Context, _, _ string) bool {
	*f.sequence = append(*f.sequence, "workspace.member")
	return f.allowed
}

type fakeIssueAttachments struct {
	sequence    *[]string
	exists      bool
	createErr   error
	created     PreparedAsset
	existsCalls int
	createCalls int
}

func (f *fakeIssueAttachments) ExistsInWorkspace(_ context.Context, _, _ string) bool {
	f.existsCalls++
	*f.sequence = append(*f.sequence, "issue.exists")
	return f.exists
}

func (f *fakeIssueAttachments) CreateForIssue(_ context.Context, asset PreparedAsset, _ string) (PreparedAsset, error) {
	f.createCalls++
	f.created = asset
	*f.sequence = append(*f.sequence, "issue.create")
	return asset, f.createErr
}

func TestIssueUploadRejectsNonMemberBeforeIssueLookup(t *testing.T) {
	sequence := []string{}
	space := &fakeSpaceUploads{sequence: &sequence, available: true}
	issues := &fakeIssueAttachments{sequence: &sequence, exists: true}
	workflow := NewUploadWorkflow(space, fakeWorkspaceAccess{sequence: &sequence, allowed: false}, issues)
	issueID := "issue-1"

	_, err := workflow.Upload(context.Background(), UploadCommand{
		UserID: "user-1", WorkspaceID: "workspace-1", IssueID: &issueID, Filename: "notes.txt",
	})

	if !errors.Is(err, ErrNotWorkspaceMember) {
		t.Fatalf("Upload error = %v, want ErrNotWorkspaceMember", err)
	}
	if !reflect.DeepEqual(sequence, []string{"workspace.member"}) {
		t.Fatalf("call sequence = %v", sequence)
	}
	if issues.existsCalls != 0 || space.prepareCalls != 0 || space.uploadCalls != 0 {
		t.Fatalf("denied workflow called issue=%d prepare=%d upload=%d", issues.existsCalls, space.prepareCalls, space.uploadCalls)
	}
}

func TestIssueUploadRejectsForeignIssueBeforeObjectUpload(t *testing.T) {
	sequence := []string{}
	space := &fakeSpaceUploads{sequence: &sequence, available: true}
	issues := &fakeIssueAttachments{sequence: &sequence, exists: false}
	workflow := NewUploadWorkflow(space, fakeWorkspaceAccess{sequence: &sequence, allowed: true}, issues)
	issueID := "01980000-0000-7000-8000-000000000002"

	_, err := workflow.Upload(context.Background(), UploadCommand{
		UserID: "user-1", WorkspaceID: "workspace-1", IssueID: &issueID, Filename: "notes.txt",
	})

	if !errors.Is(err, ErrIssueNotAccessible) {
		t.Fatalf("Upload error = %v, want ErrIssueNotAccessible", err)
	}
	if !reflect.DeepEqual(sequence, []string{"workspace.member", "issue.exists"}) {
		t.Fatalf("call sequence = %v", sequence)
	}
	if space.prepareCalls != 0 || issues.createCalls != 0 {
		t.Fatalf("foreign Issue prepared=%d created=%d", space.prepareCalls, issues.createCalls)
	}
}

func TestIssueUploadRejectsMalformedIssueAfterMembership(t *testing.T) {
	sequence := []string{}
	space := &fakeSpaceUploads{sequence: &sequence, available: true}
	issues := &fakeIssueAttachments{sequence: &sequence, exists: true}
	workflow := NewUploadWorkflow(space, fakeWorkspaceAccess{sequence: &sequence, allowed: true}, issues)
	issueID := "not-a-uuid"

	_, err := workflow.Upload(context.Background(), UploadCommand{
		UserID: "user-1", WorkspaceID: "workspace-1", IssueID: &issueID, Filename: "notes.txt",
	})

	if !errors.Is(err, ErrInvalidIssueID) {
		t.Fatalf("Upload error = %v, want ErrInvalidIssueID", err)
	}
	if !reflect.DeepEqual(sequence, []string{"workspace.member"}) {
		t.Fatalf("call sequence = %v", sequence)
	}
	if issues.existsCalls != 0 || space.prepareCalls != 0 {
		t.Fatalf("malformed Issue queried=%d prepared=%d", issues.existsCalls, space.prepareCalls)
	}
}

func TestIssueUploadPersistsAssetAndRelationInOneConsumerOperation(t *testing.T) {
	sequence := []string{}
	asset := testAsset(t)
	space := &fakeSpaceUploads{sequence: &sequence, available: true, prepared: asset}
	issues := &fakeIssueAttachments{sequence: &sequence, exists: true}
	workflow := NewUploadWorkflow(space, fakeWorkspaceAccess{sequence: &sequence, allowed: true}, issues)
	issueID := "01980000-0000-7000-8000-000000000002"

	result, err := workflow.Upload(context.Background(), UploadCommand{
		UserID: "user-1", WorkspaceID: "workspace-1", IssueID: &issueID, Filename: "notes.txt",
	})

	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !reflect.DeepEqual(sequence, []string{"workspace.member", "issue.exists", "space.prepare", "issue.create"}) {
		t.Fatalf("call sequence = %v", sequence)
	}
	if issues.createCalls != 1 || space.uploadCalls != 0 {
		t.Fatalf("create calls=%d ordinary Space uploads=%d", issues.createCalls, space.uploadCalls)
	}
	if result.Asset == nil || result.Asset.ID != asset.ID || result.IssueID == nil || *result.IssueID != issueID {
		t.Fatalf("result = %#v", result)
	}
}

func TestIssueUploadPreservesDirectLinkWhenAtomicMetadataInsertFails(t *testing.T) {
	sequence := []string{}
	asset := testAsset(t)
	persistenceErr := errors.New("database unavailable")
	space := &fakeSpaceUploads{sequence: &sequence, available: true, prepared: asset}
	issues := &fakeIssueAttachments{sequence: &sequence, exists: true, createErr: persistenceErr}
	workflow := NewUploadWorkflow(space, fakeWorkspaceAccess{sequence: &sequence, allowed: true}, issues)
	issueID := "01980000-0000-7000-8000-000000000002"

	_, err := workflow.Upload(context.Background(), UploadCommand{
		UserID: "user-1", WorkspaceID: "workspace-1", IssueID: &issueID, Filename: "notes.txt",
	})

	var metadataErr *MetadataPersistenceError
	if !errors.As(err, &metadataErr) || !errors.Is(err, persistenceErr) {
		t.Fatalf("Upload error = %v, want MetadataPersistenceError", err)
	}
	if metadataErr.Result.ID != "" || metadataErr.Result.URL != asset.URL || metadataErr.Result.Filename != asset.Filename {
		t.Fatalf("fallback = %#v", metadataErr.Result)
	}
}

func TestPersonalUploadIgnoresExtraneousMalformedIssueID(t *testing.T) {
	sequence := []string{}
	space := &fakeSpaceUploads{
		sequence:     &sequence,
		available:    true,
		uploadResult: SpaceUploadResult{ID: "asset-1", URL: "/uploads/users/user-1/asset-1.txt", Filename: "notes.txt"},
	}
	issues := &fakeIssueAttachments{sequence: &sequence, exists: true}
	workflow := NewUploadWorkflow(space, fakeWorkspaceAccess{sequence: &sequence, allowed: false}, issues)
	issueID := "not-a-uuid"

	result, err := workflow.Upload(context.Background(), UploadCommand{
		UserID: "user-1", IssueID: &issueID, Filename: "notes.txt",
	})

	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.ID != "asset-1" || !reflect.DeepEqual(sequence, []string{"space.upload"}) {
		t.Fatalf("result=%#v sequence=%v", result, sequence)
	}
	if issues.existsCalls != 0 {
		t.Fatalf("personal upload queried Issue %d times", issues.existsCalls)
	}
}

func testAsset(t *testing.T) PreparedAsset {
	t.Helper()
	return PreparedAsset{
		ID:           "asset-1",
		WorkspaceID:  "workspace-1",
		UploaderType: "member",
		UploaderID:   "user-1",
		Filename:     "notes.txt",
		URL:          "/uploads/workspaces/workspace-1/asset-1.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		Checksum:     "sha256:test",
		CreatedAt:    time.Now(),
	}
}
