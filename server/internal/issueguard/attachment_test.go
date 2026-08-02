package issueguard

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type recordingDB struct {
	query string
	args  []any
	row   pgx.Row
}

func (*recordingDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (*recordingDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (r *recordingDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	r.query = query
	r.args = args
	return r.row
}

type attachmentRow struct {
	values []any
}

func (r attachmentRow) Scan(destinations ...any) error {
	for index, value := range r.values {
		switch destination := destinations[index].(type) {
		case *pgtype.UUID:
			*destination = value.(pgtype.UUID)
		case *string:
			*destination = value.(string)
		case *int64:
			*destination = value.(int64)
		case *pgtype.Timestamptz:
			*destination = value.(pgtype.Timestamptz)
		}
	}
	return nil
}

func TestCreateForIssueUsesOneInsertContainingIssueRelation(t *testing.T) {
	assetID := testUUID("01980000-0000-7000-8000-000000000001")
	issueID := testUUID("01980000-0000-7000-8000-000000000002")
	workspaceID := testUUID("01980000-0000-7000-8000-000000000003")
	uploaderID := testUUID("01980000-0000-7000-8000-000000000004")
	createdAt := pgtype.Timestamptz{Time: time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC), Valid: true}
	recorder := &recordingDB{row: attachmentRow{values: []any{
		assetID, workspaceID, issueID, pgtype.UUID{}, "member", uploaderID,
		"notes.txt", "/uploads/notes.txt", "text/plain", int64(5), createdAt,
	}}}
	references := NewAttachmentReferences(db.New(recorder))

	stored, err := references.CreateForIssue(context.Background(), PreparedAsset{
		ID:           uuidString(assetID),
		WorkspaceID:  uuidString(workspaceID),
		UploaderType: "member",
		UploaderID:   uuidString(uploaderID),
		Filename:     "notes.txt",
		URL:          "/uploads/notes.txt",
		ContentType:  "text/plain",
		SizeBytes:    5,
		Checksum:     "sha256:test",
	}, uuidString(issueID))

	if err != nil {
		t.Fatalf("CreateForIssue: %v", err)
	}
	if !strings.Contains(recorder.query, "INSERT INTO attachment") || strings.Contains(recorder.query, "UPDATE") {
		t.Fatalf("query = %q", recorder.query)
	}
	if len(recorder.args) != 10 || recorder.args[8] != issueID {
		t.Fatalf("CreateAttachment args = %#v", recorder.args)
	}
	if stored.ID != uuidString(assetID) || stored.WorkspaceID != uuidString(workspaceID) || stored.Checksum != "sha256:test" {
		t.Fatalf("stored = %#v", stored)
	}
}

func testUUID(value string) pgtype.UUID {
	return pgtype.UUID{Bytes: uuid.MustParse(value), Valid: true}
}

func uuidString(value pgtype.UUID) string {
	return uuid.UUID(value.Bytes).String()
}
