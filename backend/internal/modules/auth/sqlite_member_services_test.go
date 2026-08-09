package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	_ "modernc.org/sqlite"
)

func openAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=busy_timeout(5000)", name))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedAuthUser(t *testing.T, db *sql.DB, id, name, email string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO auth_users(id, name, email, created_at, updated_at)
		VALUES (?, ?, ?, '2026-08-03T00:00:00Z', '2026-08-03T00:00:00Z')`, id, name, email)
	if err != nil {
		t.Fatal(err)
	}
}

func newAuthMemberTestModule(t *testing.T, db *sql.DB, ids ...string) *Module {
	t.Helper()
	var mutex sync.Mutex
	index := 0
	module, err := NewWithSqliteMemberServices(SqlitePersistenceConfig{
		DB: db,
		NewMemberID: func(context.Context) (string, error) {
			mutex.Lock()
			defer mutex.Unlock()
			if index >= len(ids) {
				return fmt.Sprintf("member-%d", index+1), nil
			}
			value := ids[index]
			index++
			return value, nil
		},
		Now: func() time.Time { return time.Date(2026, 8, 3, 4, 5, 6, 7, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestAuthSqliteMigrationsAreRepeatableAndOwnOnlyAuthTables(t *testing.T) {
	db := openAuthTestDB(t)
	if err := MigrateSqlite(context.Background(), db); err != nil {
		t.Fatalf("second MigrateSqlite() error = %v", err)
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_schema_migrations`).Scan(&migrationCount); err != nil || migrationCount != 1 {
		t.Fatalf("migration count/error = %d/%v", migrationCount, err)
	}
	for _, table := range []string{"auth_users", "auth_members", "auth_workspace_membership_roots"} {
		var found string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&found); err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
		rows, err := db.Query(`PRAGMA foreign_key_list(` + table + `)`)
		if err != nil {
			t.Fatal(err)
		}
		if rows.Next() {
			rows.Close()
			t.Fatalf("table %s unexpectedly declares a foreign key", table)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO auth_members(id, workspace_id, user_id, role, created_at)
		VALUES ('invalid', 'workspace', 'user', 'viewer', '2026-08-03T00:00:00Z')`); err == nil {
		t.Fatal("invalid member role persisted")
	}
}

func TestAuthSqliteProvisionListAndRoleLifecycle(t *testing.T) {
	ctx := context.Background()
	db := openAuthTestDB(t)
	seedAuthUser(t, db, "owner-user", "Owner", "owner@example.test")
	seedAuthUser(t, db, "member-user", "Member", "member@example.test")
	seedAuthUser(t, db, "outsider-user", "Outsider", "outsider@example.test")
	module := newAuthMemberTestModule(t, db, "owner-member", "replay-unused", "conflict-unused", "missing-unused")
	service := module.MemberLocal()

	created, err := service.ProvisionWorkspaceOwner(ctx, contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "owner-user"})
	if err != nil || created.Member == nil || !created.Created || created.Member.Id != "owner-member" || created.Member.Role != "owner" {
		t.Fatalf("ProvisionWorkspaceOwner() = %+v, %v", created, err)
	}
	replay, err := service.ProvisionWorkspaceOwner(ctx, contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "owner-user"})
	if err != nil || replay.Member == nil || replay.Created || replay.Member.Id != "owner-member" {
		t.Fatalf("replayed ProvisionWorkspaceOwner() = %+v, %v", replay, err)
	}
	if _, err := service.ProvisionWorkspaceOwner(ctx, contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-1", UserId: "outsider-user"}); !errors.Is(err, contract.ErrWorkspaceMembershipInitialized) {
		t.Fatalf("conflicting ProvisionWorkspaceOwner() error = %v", err)
	}
	if _, err := service.ProvisionWorkspaceOwner(ctx, contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-missing", UserId: "missing-user"}); !errors.Is(err, contract.ErrAuthUserNotFound) {
		t.Fatalf("missing-user ProvisionWorkspaceOwner() error = %v", err)
	}
	var missingRootCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_workspace_membership_roots WHERE workspace_id = 'workspace-missing'`).Scan(&missingRootCount); err != nil || missingRootCount != 0 {
		t.Fatalf("rolled-back root count/error = %d/%v", missingRootCount, err)
	}
	_, err = db.Exec(`INSERT INTO auth_members(id, workspace_id, user_id, role, created_at)
		VALUES ('target-member', 'workspace-1', 'member-user', 'member', '2026-08-03T05:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO auth_members(id, workspace_id, user_id, role, created_at)
		VALUES ('foreign-member', 'workspace-2', 'outsider-user', 'owner', '2026-08-03T05:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListMembers(contract.WithMemberActor(ctx, "owner-user"), contract.ListMembersRequest{WorkspaceId: "workspace-1"})
	if err != nil || len(listed.Members) != 2 || listed.Members[0].Id != "owner-member" || listed.Members[1].Id != "target-member" {
		t.Fatalf("ListMembers() = %+v, %v", listed.Members, err)
	}
	if _, err := service.ListMembers(contract.WithMemberActor(ctx, "outsider-user"), contract.ListMembersRequest{WorkspaceId: "workspace-1"}); !errors.Is(err, contract.ErrWorkspaceMembershipHidden) {
		t.Fatalf("outsider ListMembers() error = %v", err)
	}
	if _, err := service.UpdateMemberRole(
		contract.WithMemberActor(ctx, "owner-user"),
		contract.UpdateMemberRoleRequest{WorkspaceId: "workspace-1", MemberId: "foreign-member", Role: "admin"},
	); !errors.Is(err, contract.ErrMemberNotFound) {
		t.Fatalf("foreign UpdateMemberRole() error = %v", err)
	}
	updated, err := service.UpdateMemberRole(
		contract.WithMemberActor(ctx, "owner-user"),
		contract.UpdateMemberRoleRequest{WorkspaceId: "workspace-1", MemberId: "target-member", Role: "admin"},
	)
	if err != nil || updated.Member == nil || updated.Member.Role != "admin" {
		t.Fatalf("UpdateMemberRole() = %+v, %v", updated.Member, err)
	}
	if _, err := service.UpdateMemberRole(
		contract.WithMemberActor(ctx, "owner-user"),
		contract.UpdateMemberRoleRequest{WorkspaceId: "workspace-1", MemberId: "owner-member", Role: "admin"},
	); !errors.Is(err, contract.ErrLastWorkspaceOwner) {
		t.Fatalf("last-owner UpdateMemberRole() error = %v", err)
	}
	var ownerRole string
	if err := db.QueryRow(`SELECT role FROM auth_members WHERE id = 'owner-member'`).Scan(&ownerRole); err != nil || ownerRole != "owner" {
		t.Fatalf("persisted owner role/error = %q/%v", ownerRole, err)
	}
}

func TestAuthSqliteConcurrentProvisionReplayCreatesOneOwner(t *testing.T) {
	db := openAuthTestDB(t)
	seedAuthUser(t, db, "owner-user", "Owner", "owner@example.test")
	var sequence atomic.Int64
	module, err := NewWithSqliteMemberServices(SqlitePersistenceConfig{
		DB: db,
		NewMemberID: func(context.Context) (string, error) {
			return fmt.Sprintf("member-%d", sequence.Add(1)), nil
		},
		Now: func() time.Time { return time.Date(2026, 8, 3, 4, 5, 6, 7, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	responses := make(chan contract.ProvisionWorkspaceOwnerResponse, 2)
	errorsChannel := make(chan error, 2)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			response, callErr := module.MemberLocal().ProvisionWorkspaceOwner(context.Background(), contract.ProvisionWorkspaceOwnerRequest{
				WorkspaceId: "workspace-concurrent", UserId: "owner-user",
			})
			responses <- response
			errorsChannel <- callErr
		}()
	}
	close(start)
	wait.Wait()
	close(responses)
	close(errorsChannel)
	for callErr := range errorsChannel {
		if callErr != nil {
			t.Fatalf("concurrent ProvisionWorkspaceOwner() error = %v", callErr)
		}
	}
	createdCount := 0
	memberID := ""
	for response := range responses {
		if response.Created {
			createdCount++
		}
		if response.Member == nil {
			t.Fatal("concurrent response member is nil")
		}
		if memberID == "" {
			memberID = response.Member.Id
		} else if response.Member.Id != memberID {
			t.Fatalf("concurrent member IDs = %q and %q", memberID, response.Member.Id)
		}
	}
	if createdCount != 1 {
		t.Fatalf("created response count = %d", createdCount)
	}
	var persisted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_members WHERE workspace_id = 'workspace-concurrent'`).Scan(&persisted); err != nil || persisted != 1 {
		t.Fatalf("persisted concurrent members/error = %d/%v", persisted, err)
	}
}

func TestAuthSqliteMemberGRPCContracts(t *testing.T) {
	ctx := context.Background()
	db := openAuthTestDB(t)
	seedAuthUser(t, db, "owner-user", "Owner", "owner@example.test")
	seedAuthUser(t, db, "member-user", "Member", "member@example.test")
	module := newAuthMemberTestModule(t, db, "owner-grpc")
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if incoming, ok := metadata.FromIncomingContext(ctx); ok {
			if userIDs := incoming.Get("x-auth-user-id"); len(userIDs) > 0 {
				ctx = contract.WithMemberActor(ctx, userIDs[0])
			}
		}
		return handler(ctx, request)
	}))
	module.RegisterGRPCService(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	connection, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	client := NewMemberGRPCClient(connection)
	created, err := client.ProvisionWorkspaceOwner(ctx, contract.ProvisionWorkspaceOwnerRequest{WorkspaceId: "workspace-grpc", UserId: "owner-user"})
	if err != nil || created.Member == nil || created.Member.Id != "owner-grpc" {
		t.Fatalf("gRPC ProvisionWorkspaceOwner() = %+v, %v", created.Member, err)
	}
	_, err = db.Exec(`INSERT INTO auth_members(id, workspace_id, user_id, role, created_at)
		VALUES ('member-grpc', 'workspace-grpc', 'member-user', 'member', '2026-08-03T05:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	authorizedCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-auth-user-id", "owner-user"))
	listed, err := client.ListMembers(authorizedCtx, contract.ListMembersRequest{WorkspaceId: "workspace-grpc"})
	if err != nil || len(listed.Members) != 2 {
		t.Fatalf("gRPC ListMembers() = %+v, %v", listed.Members, err)
	}
	updated, err := client.UpdateMemberRole(authorizedCtx, contract.UpdateMemberRoleRequest{WorkspaceId: "workspace-grpc", MemberId: "member-grpc", Role: "admin"})
	if err != nil || updated.Member == nil || updated.Member.Role != "admin" {
		t.Fatalf("gRPC UpdateMemberRole() = %+v, %v", updated.Member, err)
	}
}

func TestNewWithSqliteMemberServicesRequiresDatabase(t *testing.T) {
	if _, err := NewWithSqliteMemberServices(SqlitePersistenceConfig{}); err == nil {
		t.Fatal("NewWithSqliteMemberServices() error = nil")
	}
}
