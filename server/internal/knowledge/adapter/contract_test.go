package adapter_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/knowledge"
	"github.com/multica-ai/multica/server/internal/knowledge/adapter/memory"
	knowledgeSqlite "github.com/multica-ai/multica/server/internal/knowledge/adapter/sqlite"
)

type repositoryFactory struct {
	name string
	open func(*testing.T) (knowledge.Repository, func())
}

func TestRepositoryContractStoresPublishedKnowledgeByWorkspace(t *testing.T) {
	factories := []repositoryFactory{
		{
			name: "memory",
			open: func(*testing.T) (knowledge.Repository, func()) {
				return memory.New(), func() {}
			},
		},
		{
			name: "sqlite",
			open: func(t *testing.T) (knowledge.Repository, func()) {
				store, err := knowledgeSqlite.Open(filepath.Join(t.TempDir(), "knowledge.db"))
				if err != nil {
					t.Fatal(err)
				}
				return store, func() { _ = store.Close() }
			},
		},
	}

	for _, factory := range factories {
		t.Run(factory.name, func(t *testing.T) {
			store, closeStore := factory.open(t)
			t.Cleanup(closeStore)
			service := knowledge.NewService(store, nil)
			ctx := context.Background()
			candidate, err := service.Propose(ctx, knowledge.ProposalInput{
				WorkspaceID: "workspace-1",
				Kind:        knowledge.KindDecision,
				Title:       "Use a replaceable knowledge store",
				Content:     "The domain must not expose SQL types.",
				Reason:      "Database portability is a product invariant.",
				ProposedBy:  "user-1",
			})
			if err != nil {
				t.Fatal(err)
			}
			_, published, err := service.Review(ctx, knowledge.ReviewInput{
				WorkspaceID:      "workspace-1",
				CandidateID:      candidate.ID,
				ExpectedRevision: 1,
				Action:           knowledge.ReviewApprove,
				ReviewerID:       "admin-1",
				Rationale:        "The adapter contract verifies this boundary.",
			})
			if err != nil {
				t.Fatal(err)
			}

			entry, err := store.GetEntry(ctx, "workspace-1", published.ID)
			if err != nil {
				t.Fatal(err)
			}
			if entry.ID != published.ID || entry.Revisions[0].Title != "Use a replaceable knowledge store" {
				t.Fatalf("entry = %#v", entry)
			}
			if _, err := store.GetEntry(ctx, "workspace-2", published.ID); err != knowledge.ErrNotFound {
				t.Fatalf("cross-workspace get error = %v, want %v", err, knowledge.ErrNotFound)
			}
		})
	}
}

func TestRepositoryContractSearchesOnlyPublishedWorkspaceKnowledge(t *testing.T) {
	factories := []repositoryFactory{
		{
			name: "memory",
			open: func(*testing.T) (knowledge.Repository, func()) {
				return memory.New(), func() {}
			},
		},
		{
			name: "sqlite",
			open: func(t *testing.T) (knowledge.Repository, func()) {
				store, err := knowledgeSqlite.Open(filepath.Join(t.TempDir(), "knowledge.db"))
				if err != nil {
					t.Fatal(err)
				}
				return store, func() { _ = store.Close() }
			},
		},
	}

	for _, factory := range factories {
		t.Run(factory.name, func(t *testing.T) {
			store, closeStore := factory.open(t)
			t.Cleanup(closeStore)
			service := knowledge.NewService(store, nil)
			ctx := context.Background()
			for _, workspaceID := range []string{"workspace-1", "workspace-2"} {
				candidate, err := service.Propose(ctx, knowledge.ProposalInput{
					WorkspaceID: workspaceID,
					Kind:        knowledge.KindProcedure,
					Title:       "SQLite recovery",
					Content:     "Checkpoint WAL before backup.",
					Reason:      "Keep recovery safe.",
					ProposedBy:  "user-1",
				})
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err := service.Review(ctx, knowledge.ReviewInput{
					WorkspaceID:      workspaceID,
					CandidateID:      candidate.ID,
					ExpectedRevision: 1,
					Action:           knowledge.ReviewApprove,
					ReviewerID:       "admin-1",
					Rationale:        "Verified in a restore test.",
				}); err != nil {
					t.Fatal(err)
				}
			}

			search, ok := store.(knowledge.SearchIndex)
			if !ok {
				t.Fatal("repository does not implement the search port")
			}
			if err := search.Rebuild(ctx); err != nil {
				t.Fatalf("rebuild search index: %v", err)
			}
			page, err := search.Search(ctx, knowledge.SearchQuery{
				WorkspaceID: "workspace-1",
				Text:        "checkpoint",
				Limit:       10,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Results) != 1 || page.Results[0].Entry.WorkspaceID != "workspace-1" {
				t.Fatalf("search page = %#v", page)
			}
		})
	}
}

func TestRepositoryContractListsCandidatesByWorkspaceAndStatus(t *testing.T) {
	factories := []repositoryFactory{
		{
			name: "memory",
			open: func(*testing.T) (knowledge.Repository, func()) {
				return memory.New(), func() {}
			},
		},
		{
			name: "sqlite",
			open: func(t *testing.T) (knowledge.Repository, func()) {
				store, err := knowledgeSqlite.Open(filepath.Join(t.TempDir(), "knowledge.db"))
				if err != nil {
					t.Fatal(err)
				}
				return store, func() { _ = store.Close() }
			},
		},
	}

	for _, factory := range factories {
		t.Run(factory.name, func(t *testing.T) {
			store, closeStore := factory.open(t)
			t.Cleanup(closeStore)
			service := knowledge.NewService(store, nil)
			for _, workspaceID := range []string{"workspace-1", "workspace-2"} {
				if _, err := service.Propose(context.Background(), knowledge.ProposalInput{
					WorkspaceID: workspaceID,
					Kind:        knowledge.KindLesson,
					Title:       "Recovery lesson",
					Content:     "Retain evidence during retry.",
					Reason:      "Captured during a retrospective.",
					ProposedBy:  "user-1",
				}); err != nil {
					t.Fatal(err)
				}
			}
			page, err := store.ListCandidates(context.Background(), knowledge.CandidateQuery{
				WorkspaceID: "workspace-1",
				Statuses:    []knowledge.Status{knowledge.StatusCandidate},
				Limit:       10,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Candidates) != 1 || page.Candidates[0].WorkspaceID != "workspace-1" {
				t.Fatalf("candidate page = %#v", page)
			}
		})
	}
}
