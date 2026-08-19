package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

type projectResourceBarrierRepository struct {
	application.ProjectResourceRepository
	entered chan struct{}
	release chan struct{}
}

func (r *projectResourceBarrierRepository) MutateProjectResource(ctx context.Context, command application.ProjectResourceMutation) (contract.ProjectResource, error) {
	close(r.entered)
	<-r.release
	return r.ProjectResourceRepository.MutateProjectResource(ctx, command)
}

type transactionAuthorizerStub struct{}

func (transactionAuthorizerStub) AuthorizeWorkspace(context.Context, string, string) error {
	return nil
}

type transactionMembershipReaderStub struct {
	membership contract.WorkspaceMembership
}

func (r transactionMembershipReaderStub) ListForUser(context.Context, string) ([]contract.WorkspaceMembership, error) {
	return []contract.WorkspaceMembership{r.membership}, nil
}

func (r transactionMembershipReaderStub) FindForUserAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return r.membership, true, nil
}

func (r transactionMembershipReaderStub) FindByMemberAndWorkspace(context.Context, string, string) (contract.WorkspaceMembership, bool, error) {
	return r.membership, true, nil
}

type transactionConnectionCheckerStub struct{}

func (transactionConnectionCheckerStub) Check(context.Context, contract.ProjectResourceConnectionRequest) (contract.ProjectResourceConnection, error) {
	return contract.ProjectResourceConnection{State: "available"}, nil
}

func TestProjectResourceUseCaseWriteBarrierRevalidatesLeadAndProjectStatus(t *testing.T) {
	for _, test := range []struct {
		name       string
		actorID    string
		membership contract.WorkspaceMembership
		prepare    func(*testing.T, sqlExecutor)
		change     func(*testing.T, sqlExecutor)
	}{
		{
			name:    "lead changed after application authorization",
			actorID: "lead-user",
			membership: contract.WorkspaceMembership{
				MemberID: "lead-member", UserID: "lead-user", WorkspaceID: "workspace-1", Role: "member",
			},
			prepare: func(t *testing.T, db sqlExecutor) {
				t.Helper()
				if _, err := db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES('lead-member','workspace-1','lead-user','member','2026-08-19T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`UPDATE workspace_projects SET lead_type='member',lead_id='lead-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
					t.Fatal(err)
				}
			},
			change: func(t *testing.T, db sqlExecutor) {
				t.Helper()
				if _, err := db.Exec(`UPDATE workspace_projects SET lead_id='another-member' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:    "project completed after application authorization",
			actorID: "owner-1",
			membership: contract.WorkspaceMembership{
				MemberID: "owner-member", UserID: "owner-1", WorkspaceID: "workspace-1", Role: "owner",
			},
			prepare: func(*testing.T, sqlExecutor) {},
			change: func(t *testing.T, db sqlExecutor) {
				t.Helper()
				if _, err := db.Exec(`UPDATE workspace_projects SET status='completed' WHERE workspace_id='workspace-1' AND id='project-1'`); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openProjectResourceDB(t)
			test.prepare(t, db)
			repository, err := persistence.NewProjectResourceRepository(persistence.Config{DB: db})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
			actor := contract.WorkspaceActor{Type: "member", ID: test.actorID}
			if _, err = repository.CreateProjectResource(context.Background(), application.ProjectResourceCreate{
				ID: "resource-1", WorkspaceID: "workspace-1", ProjectID: "project-1", ResourceType: "url",
				ResourceRef: contract.ProjectResourceRef{URL: "https://example.com/docs"}, Fingerprint: strings.Repeat("a", 64),
				IdempotencyKey: "create-1", RequestHash: strings.Repeat("1", 64), Actor: actor, OccurredAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			barrier := &projectResourceBarrierRepository{
				ProjectResourceRepository: repository,
				entered:                   make(chan struct{}),
				release:                   make(chan struct{}),
			}
			service, err := application.NewProjectResourceUseCase(
				barrier,
				transactionAuthorizerStub{},
				transactionMembershipReaderStub{membership: test.membership},
				transactionConnectionCheckerStub{},
				func(context.Context) (string, error) { return "unused", nil },
				func() time.Time { return now.Add(time.Second) },
			)
			if err != nil {
				t.Fatal(err)
			}
			label := "must not persist"
			result := make(chan error, 1)
			go func() {
				_, updateErr := service.UpdateProjectResource(
					contract.WithWorkspaceActor(context.Background(), "member", test.actorID),
					"workspace-1", "project-1", "resource-1",
					contract.UpdateProjectResourceRequest{Action: "update", ExpectedRevision: 1, Label: &label},
				)
				result <- updateErr
			}()
			<-barrier.entered
			test.change(t, db)
			close(barrier.release)
			if updateErr := <-result; !errors.Is(updateErr, contract.ErrWorkspacePermissionDenied) {
				t.Fatalf("UpdateProjectResource() error = %v", updateErr)
			}
			list, err := repository.ListProjectResources(context.Background(), "workspace-1", "project-1", false)
			if err != nil || list.Revision != 1 || list.Resources[0].Label != "" {
				t.Fatalf("list = %#v, %v", list, err)
			}
		})
	}
}

type sqlExecutor interface {
	Exec(string, ...any) (sql.Result, error)
}
