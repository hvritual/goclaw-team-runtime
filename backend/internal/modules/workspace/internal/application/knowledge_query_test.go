package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	knowledgeDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/knowledge"
)

type knowledgeQueryRepositoryStub struct {
	entries []knowledgeDomain.GovernedEntry
}

func (s knowledgeQueryRepositoryStub) ListGovernedKnowledge(context.Context, string) ([]knowledgeDomain.GovernedEntry, error) {
	return s.entries, nil
}

type knowledgeQueryAuthorizerStub struct{ err error }

func (s knowledgeQueryAuthorizerStub) AuthorizeWorkspace(context.Context, string, string) error {
	return s.err
}

type knowledgeQueryMembershipsStub struct{ role string }

func (s knowledgeQueryMembershipsStub) ListForUser(context.Context, string) ([]contract.WorkspaceMembership, error) {
	return nil, nil
}
func (s knowledgeQueryMembershipsStub) FindForUserAndWorkspace(_ context.Context, user, workspace string) (contract.WorkspaceMembership, bool, error) {
	return contract.WorkspaceMembership{UserID: user, WorkspaceID: workspace, Role: s.role}, true, nil
}
func (s knowledgeQueryMembershipsStub) FindByMemberAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return contract.WorkspaceMembership{}, false, nil
}

func governedFixture(id, status, title, content, updated string) knowledgeDomain.GovernedEntry {
	timestamp, _ := time.Parse(time.RFC3339, updated)
	return knowledgeDomain.GovernedEntry{
		ID: id, WorkspaceID: "workspace-1", Kind: "lesson", Status: status,
		CurrentRevision: 1, CreatedAt: timestamp, UpdatedAt: timestamp,
		Revisions: []knowledgeDomain.Revision{{Number: 1, Title: title, Content: content, CreatedBy: "user-2", CreatedAt: timestamp,
			SourceRefs: []knowledgeDomain.SourceRef{{Type: "acceptance_conclusion", ID: "issue-1", Revision: "sha256:abc", Citation: "Acceptance passed"}}}},
	}
}

func TestKnowledgeQueryRanksPagesAndBindsCursor(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	service, err := NewKnowledgeQueryUseCase(knowledgeQueryRepositoryStub{entries: []knowledgeDomain.GovernedEntry{
		governedFixture("content", "published", "Other", "retry delivery", "2026-08-18T08:00:00Z"),
		governedFixture("prefix", "published", "Retry safely", "body", "2026-08-18T07:00:00Z"),
		governedFixture("exact", "published", "Retry", "body", "2026-08-18T06:00:00Z"),
	}}, knowledgeQueryAuthorizerStub{}, knowledgeQueryMembershipsStub{role: "member"}, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
	first, err := service.QueryKnowledge(ctx, contract.QueryKnowledgeRequest{WorkspaceID: "workspace-1", Query: "retry", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Entries) != 2 || first.Entries[0].ID != "exact" || first.Entries[1].ID != "prefix" || first.NextCursor == nil {
		t.Fatalf("first page = %#v", first)
	}
	second, err := service.QueryKnowledge(ctx, contract.QueryKnowledgeRequest{WorkspaceID: "workspace-1", Query: "retry", Limit: 2, Cursor: *first.NextCursor})
	if err != nil || len(second.Entries) != 1 || second.Entries[0].ID != "content" {
		t.Fatalf("second page = %#v, %v", second, err)
	}
	if _, err := service.QueryKnowledge(ctx, contract.QueryKnowledgeRequest{WorkspaceID: "workspace-1", Query: "other", Limit: 2, Cursor: *first.NextCursor}); !errors.Is(err, contract.ErrInvalidKnowledgeQuery) {
		t.Fatalf("cross-filter cursor error = %v", err)
	}
}

func TestKnowledgeQueryHidesQuarantineFromMembers(t *testing.T) {
	entry := governedFixture("quarantined", "quarantined", "Unsafe", "body", "2026-08-18T08:00:00Z")
	for _, test := range []struct {
		role    string
		want    int
		wantErr error
	}{{"member", 0, contract.ErrInvalidKnowledgeQuery}, {"admin", 1, nil}} {
		t.Run(test.role, func(t *testing.T) {
			service, err := NewKnowledgeQueryUseCase(knowledgeQueryRepositoryStub{entries: []knowledgeDomain.GovernedEntry{entry}}, knowledgeQueryAuthorizerStub{}, knowledgeQueryMembershipsStub{role: test.role}, []byte("01234567890123456789012345678901"), time.Now)
			if err != nil {
				t.Fatal(err)
			}
			ctx := contract.WithWorkspaceActor(context.Background(), "member", "user-1")
			response, err := service.QueryKnowledge(ctx, contract.QueryKnowledgeRequest{WorkspaceID: "workspace-1", Statuses: []string{"quarantined"}})
			if !errors.Is(err, test.wantErr) || len(response.Entries) != test.want {
				t.Fatalf("response/error = %#v/%v", response, err)
			}
		})
	}
}
