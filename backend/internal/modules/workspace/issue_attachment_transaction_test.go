package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	spacecontract "github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

var errRejectedAttachmentReferences = errors.New("attachment references changed")

type rollbackProvingAttachmentValidator struct{ calls int }

func (v *rollbackProvingAttachmentValidator) ValidateReferences(ctx context.Context, executor spacecontract.AttachmentExecutor, workspaceID string, ids []string) error {
	v.calls++
	if executor == nil || workspaceID != "workspace-1" || len(ids) != 1 || ids[0] == "" {
		return errors.New("invalid transactional attachment validation input")
	}
	if _, err := executor.ExecContext(ctx, `UPDATE workspaces SET name='must roll back' WHERE id=?`, workspaceID); err != nil {
		return err
	}
	return errors.Join(errRejectedAttachmentReferences, spacecontract.ErrAttachmentNotFound)
}

func TestIssueCreateValidatesAttachmentReferencesInsideOwnedTransaction(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	validator := &rollbackProvingAttachmentValidator{}
	repository, err := persistence.NewIssueRepository(persistence.Config{DB: db, AttachmentReferences: validator})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	value, err := issueDomain.New("issue-transaction", "workspace-1", "Atomic create", nil, "todo", "none", nil, nil, nil, nil, "member", "member-1", 0, nil, nil, nil, []string{"asset-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Create(t.Context(), value); !errors.Is(err, errRejectedAttachmentReferences) || !errors.Is(err, contract.ErrAssetOutsideWorkspace) {
		t.Fatalf("Create() error = %v, want transactional cause and Workspace reference error", err)
	}
	var name string
	var issueCount int
	if err := db.QueryRow(`SELECT name FROM workspaces WHERE id='workspace-1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM workspace_issues WHERE workspace_id='workspace-1'`).Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if validator.calls != 1 || name != "Acme" || issueCount != 0 {
		t.Fatalf("create rollback calls=%d workspace=%q issues=%d", validator.calls, name, issueCount)
	}
}

func TestIssueAttachmentReplacementValidationFailureRollsBackWholeUpdate(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	value, err := issueDomain.New("issue-transaction", "workspace-1", "Before", nil, "todo", "none", nil, nil, nil, nil, "member", "member-1", 0, nil, nil, nil, []string{"asset-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	seedRepository, err := persistence.NewIssueRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	created, err := seedRepository.Create(t.Context(), value)
	if err != nil {
		t.Fatal(err)
	}
	validator := &rollbackProvingAttachmentValidator{}
	repository, err := persistence.NewIssueRepository(persistence.Config{DB: db, AttachmentReferences: validator})
	if err != nil {
		t.Fatal(err)
	}
	title := "After"
	_, err = repository.Update(t.Context(), application.IssueUpdateCommand{
		WorkspaceID: "workspace-1", IssueID: created.ID, ExpectedAssetIDs: []string{"asset-1"},
		Patch: issueDomain.Patch{Title: &title, AssetIDs: issueDomain.AssetsChange{Set: true, Values: []string{"asset-2"}}},
		Now:   now.Add(time.Minute),
	})
	if !errors.Is(err, errRejectedAttachmentReferences) || !errors.Is(err, contract.ErrAssetOutsideWorkspace) {
		t.Fatalf("Update() error = %v, want transactional cause and Workspace reference error", err)
	}
	got, err := seedRepository.FindByIDOrIdentifier(t.Context(), "workspace-1", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.QueryRow(`SELECT name FROM workspaces WHERE id='workspace-1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if validator.calls != 1 || name != "Acme" || got.Title != "Before" || len(got.AssetIDs) != 1 || got.AssetIDs[0] != "asset-1" {
		t.Fatalf("update rollback calls=%d workspace=%q issue=%#v", validator.calls, name, got)
	}
}
