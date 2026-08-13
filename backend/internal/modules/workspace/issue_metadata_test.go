package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	workspacev1 "github.com/hvritual/workspace/rpc/pb/workspace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestIssueMetadataSQLiteLocalContract(t *testing.T) {
	db := openWorkspaceTestDB(t)
	db.SetMaxOpenConns(8)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	seedWorkspace(t, db, "workspace-2", "Globex", "globex")
	now := time.Date(2026, 8, 13, 7, 0, 0, 0, time.UTC)
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, func() time.Time { return now })
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "Metadata"})
	if err != nil {
		t.Fatal(err)
	}
	issueID := created.Issue.Id
	metadata := module.IssueMetadataLocal()
	if _, err := metadata.DeleteIssueMetadata(ctx, contract.DeleteIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: " same "}); !errors.Is(err, contract.ErrInvalidIssueMetadata) {
		t.Fatalf("spaced delete error=%v", err)
	}

	got, err := metadata.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1"})
	if err != nil || len(got.Metadata) != 0 {
		t.Fatalf("GetIssueMetadata() = %#v, %v", got.Metadata, err)
	}
	put, err := metadata.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID, Key: "release.channel", ValueJson: `"beta"`})
	if err != nil || put.Metadata["release.channel"] != "beta" {
		t.Fatalf("PutIssueMetadata() = %#v, %v", put.Metadata, err)
	}
	put.Metadata["release.channel"] = "client mutation"
	got, _ = metadata.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID})
	if got.Metadata["release.channel"] != "beta" {
		t.Fatalf("metadata response was not copied: %#v", got.Metadata)
	}

	const writers = 12
	var group sync.WaitGroup
	errs := make(chan error, writers)
	for index := 0; index < writers; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, writeErr := metadata.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{
				WorkspaceId: "workspace-1", IssueId: issueID, Key: fmt.Sprintf("worker_%02d", index), ValueJson: fmt.Sprintf("%d", index),
			})
			errs <- writeErr
		}(index)
	}
	group.Wait()
	close(errs)
	for writeErr := range errs {
		if writeErr != nil {
			t.Fatalf("concurrent PutIssueMetadata() = %v", writeErr)
		}
	}
	got, _ = metadata.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID})
	if len(got.Metadata) != writers+1 {
		t.Fatalf("concurrent metadata keys = %d, want %d: %#v", len(got.Metadata), writers+1, got.Metadata)
	}

	if _, err := db.Exec(`UPDATE workspace_issues SET metadata = '{"looks_valid":true} trailing' WHERE id = ?`, issueID); err != nil {
		t.Fatal(err)
	}
	got, err = metadata.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID})
	if err != nil || len(got.Metadata) != 0 {
		t.Fatalf("malformed GetIssueMetadata() = %#v, %v", got.Metadata, err)
	}
	if _, err := metadata.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID, Key: "repaired", ValueJson: `true`}); err != nil {
		t.Fatal(err)
	}
	var persisted string
	if err := db.QueryRow(`SELECT metadata FROM workspace_issues WHERE id = ?`, issueID).Scan(&persisted); err != nil || !json.Valid([]byte(persisted)) {
		t.Fatalf("repaired metadata = %q, %v", persisted, err)
	}

	now = now.Add(time.Minute)
	deleted, err := metadata.DeleteIssueMetadata(ctx, contract.DeleteIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID, Key: "absent"})
	if err != nil || deleted.Metadata["repaired"] != true {
		t.Fatalf("DeleteIssueMetadata(absent) = %#v, %v", deleted.Metadata, err)
	}
	var updatedAt string
	if err := db.QueryRow(`SELECT updated_at FROM workspace_issues WHERE id = ?`, issueID).Scan(&updatedAt); err != nil || updatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("delete absent updated_at = %q, %v", updatedAt, err)
	}

	if _, err := metadata.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-2", IssueId: issueID}); !errors.Is(err, contract.ErrIssueNotFound) {
		t.Fatalf("foreign workspace error = %v", err)
	}
	if _, err := metadata.PutIssueMetadata(context.Background(), contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID, Key: "denied", ValueJson: `true`}); !errors.Is(err, contract.ErrWorkspaceActorRequired) {
		t.Fatalf("missing actor error = %v", err)
	}
	unknownActor := contract.WithWorkspaceActor(context.Background(), "service", "service-1")
	if _, err := metadata.PutIssueMetadata(unknownActor, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: issueID, Key: "denied", ValueJson: `true`}); !errors.Is(err, contract.ErrInvalidIssueMetadata) {
		t.Fatalf("unsupported actor type error = %v", err)
	}
}

func TestIssueMainlineUpdatePreservesMetadataAndPropertiesColumns(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, time.Now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := module.IssueMetadataLocal().PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "current", ValueJson: `true`}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE workspace_issues SET properties = '{"property":"current"}' WHERE id = ?`, created.Issue.Id); err != nil {
		t.Fatal(err)
	}
	title := "Updated"
	if _, err := module.IssueLocal().UpdateIssue(ctx, contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Title: &title}); err != nil {
		t.Fatal(err)
	}
	var metadataJSON, propertiesJSON string
	if err := db.QueryRow(`SELECT metadata, properties FROM workspace_issues WHERE id = ?`, created.Issue.Id).Scan(&metadataJSON, &propertiesJSON); err != nil {
		t.Fatal(err)
	}
	if metadataJSON != `{"current":true}` || propertiesJSON != `{"property":"current"}` {
		t.Fatalf("mainline update overwrote metadata/properties: %s / %s", metadataJSON, propertiesJSON)
	}
}

func TestDefaultWorkspaceModuleKeepsMetadataOptIn(t *testing.T) {
	module := New()
	defer func() {
		if recover() == nil {
			t.Fatal("default module unexpectedly registered Issue metadata")
		}
	}()
	_ = module.IssueMetadataLocal()
}

func TestIssueMetadataBufconnContract(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, time.Now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "gRPC metadata"})
	if err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(workspaceActorTestInterceptor))
	module.RegisterGRPCService(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	connection, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	grpcContext := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-workspace-actor-type", "member", "x-workspace-actor-id", "member-1"))
	client := workspacev1.NewIssueMetadataServiceClient(connection)
	put, err := client.PutIssueMetadataKey(grpcContext, &workspacev1.PutIssueMetadataKeyRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "count", ValueJson: "2"})
	if err != nil || put.GetMetadata().AsMap()["count"] != float64(2) {
		t.Fatalf("gRPC put = %#v, %v", put, err)
	}
	got, err := client.GetIssueMetadata(grpcContext, &workspacev1.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: "WSP-1"})
	if err != nil || got.GetMetadata().AsMap()["count"] != float64(2) {
		t.Fatalf("gRPC get = %#v, %v", got, err)
	}
	deleted, err := client.DeleteIssueMetadataKey(grpcContext, &workspacev1.DeleteIssueMetadataKeyRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "count"})
	if err != nil || len(deleted.GetMetadata().AsMap()) != 0 {
		t.Fatalf("gRPC delete = %#v, %v", deleted, err)
	}
}

func TestIssueMetadataHTTPCompatibility(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, time.Now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "HTTP metadata"})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)

	request := func(method, path, body string, headers bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if headers {
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("X-Workspace-Slug", "acme")
		}
		response := httptest.NewRecorder()
		server.ServeHTTP(response, req)
		return response
	}
	missingWorkspace := request(http.MethodGet, "/api/issues/"+created.Issue.Id+"/metadata", "", false)
	if missingWorkspace.Code != 400 || strings.TrimSpace(missingWorkspace.Body.String()) != `{"error":"workspace_id is required"}` {
		t.Fatalf("missing workspace = %d %s", missingWorkspace.Code, missingWorkspace.Body.String())
	}
	unauthInvalid := httptest.NewRequest(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/1bad", strings.NewReader(`{"value":true}`))
	unauthInvalid.Header.Set("X-Workspace-Slug", "acme")
	unauthResponse := httptest.NewRecorder()
	server.ServeHTTP(unauthResponse, unauthInvalid)
	if unauthResponse.Code != 401 || strings.TrimSpace(unauthResponse.Body.String()) != `{"error":"user not authenticated"}` {
		t.Fatalf("unauth invalid=%d %s", unauthResponse.Code, unauthResponse.Body.String())
	}
	forged := httptest.NewRequest(http.MethodGet, "/api/issues/"+created.Issue.Id+"/metadata", nil)
	forged.Header.Set("X-Workspace-Actor-Type", "member")
	forged.Header.Set("X-Workspace-Actor-ID", "member-1")
	forged.Header.Set("X-Workspace-Slug", "acme")
	nilIdentityServer := kratoshttp.NewServer()
	newIssueMetadataExtension(module.IssueMetadataLocal(), nil).RegisterHTTP(nilIdentityServer)
	forgedResponse := httptest.NewRecorder()
	nilIdentityServer.ServeHTTP(forgedResponse, forged)
	if forgedResponse.Code != 401 || strings.TrimSpace(forgedResponse.Body.String()) != `{"error":"user not authenticated"}` {
		t.Fatalf("forged actor=%d %s", forgedResponse.Code, forgedResponse.Body.String())
	}
	foreignServer := kratoshttp.NewServer()
	newIssueMetadataExtension(module.IssueMetadataLocal(), func(*http.Request) (contract.WorkspaceHTTPIdentity, error) {
		return contract.WorkspaceHTTPIdentity{WorkspaceID: "workspace-1", ActorType: "member", ActorID: "foreign-member"}, nil
	}).RegisterHTTP(foreignServer)
	foreignRequest := httptest.NewRequest(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/key", strings.NewReader(`{"value":true}`))
	foreignRequest.Header.Set("X-Workspace-Slug", "acme")
	foreign := httptest.NewRecorder()
	foreignServer.ServeHTTP(foreign, foreignRequest)
	if foreign.Code != 404 || strings.TrimSpace(foreign.Body.String()) != `{"error":"issue not found"}` {
		t.Fatalf("foreign actor=%d %s", foreign.Code, foreign.Body.String())
	}
	validation := []struct{ name, path, body, want string }{
		{"spaced key", "/api/issues/" + created.Issue.Id + "/metadata/%20", `{"value":true}`, `{"error":"key must match ^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$"}`},
		{"bad key", "/api/issues/" + created.Issue.Id + "/metadata/1bad", `{"value":true}`, `{"error":"key must match ^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$"}`},
		{"bad body", "/api/issues/" + created.Issue.Id + "/metadata/key", `{`, `{"error":"invalid request body"}`},
		{"missing value", "/api/issues/" + created.Issue.Id + "/metadata/key", `{}`, `{"error":"value is required"}`},
		{"null", "/api/issues/" + created.Issue.Id + "/metadata/key", `{"value":null,"extra":true}`, `{"error":"value cannot be null (use DELETE to remove a key)"}`},
		{"compound", "/api/issues/" + created.Issue.Id + "/metadata/key", `{"value":{}}`, `{"error":"value must be a primitive: string, number, or bool"}`},
		{"trailing json", "/api/issues/" + created.Issue.Id + "/metadata/key", `{"value":true}{"ignored":1}`, `{"error":"invalid request body"}`},
	}
	deleteSpaces := request(http.MethodDelete, "/api/issues/"+created.Issue.Id+"/metadata/%20key%20", "", true)
	if deleteSpaces.Code != 400 || strings.TrimSpace(deleteSpaces.Body.String()) != `{"error":"key must match ^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$"}` {
		t.Fatalf("delete spaces=%d %s", deleteSpaces.Code, deleteSpaces.Body.String())
	}
	for _, test := range validation {
		response := request(http.MethodPut, test.path, test.body, true)
		if response.Code != 400 || strings.TrimSpace(response.Body.String()) != test.want {
			t.Fatalf("%s = %d %s", test.name, response.Code, response.Body.String())
		}
	}
	invalid := request(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/key", `{"value":null}`, true)
	if invalid.Code != 400 || !strings.HasPrefix(strings.TrimSpace(invalid.Body.String()), `{"error":`) {
		t.Fatalf("invalid value = %d %s", invalid.Code, invalid.Body.String())
	}
	put := request(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/key", `{"value":2}`, true)
	if put.Code != 200 || strings.TrimSpace(put.Body.String()) != `{"metadata":{"key":2}}` {
		t.Fatalf("put = %d %s", put.Code, put.Body.String())
	}
	get := request(http.MethodGet, "/api/issues/WSP-1/metadata", "", true)
	if get.Code != 200 || strings.TrimSpace(get.Body.String()) != `{"metadata":{"key":2}}` {
		t.Fatalf("get = %d %s", get.Code, get.Body.String())
	}
	deleted := request(http.MethodDelete, "/api/issues/"+created.Issue.Id+"/metadata/key", "", true)
	if deleted.Code != 200 || strings.TrimSpace(deleted.Body.String()) != `{"metadata":{}}` {
		t.Fatalf("delete = %d %s", deleted.Code, deleted.Body.String())
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_http_metadata_update BEFORE UPDATE OF metadata ON workspace_issues BEGIN SELECT RAISE(ABORT,'forced HTTP metadata failure'); END`); err != nil {
		t.Fatal(err)
	}
	failedPut := request(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/key", `{"value":true}`, true)
	if failedPut.Code != 500 || strings.TrimSpace(failedPut.Body.String()) != `{"error":"failed to set metadata key"}` {
		t.Fatalf("failed put = %d %s", failedPut.Code, failedPut.Body.String())
	}
	failedDelete := request(http.MethodDelete, "/api/issues/"+created.Issue.Id+"/metadata/key", "", true)
	if failedDelete.Code != 500 || strings.TrimSpace(failedDelete.Body.String()) != `{"error":"failed to delete metadata key"}` {
		t.Fatalf("failed delete = %d %s", failedDelete.Code, failedDelete.Body.String())
	}
	if _, err := db.Exec(`DROP TRIGGER fail_http_metadata_update`); err != nil {
		t.Fatal(err)
	}
	missingIssue := request(http.MethodGet, "/api/issues/missing/metadata", "", true)
	if missingIssue.Code != 404 || strings.TrimSpace(missingIssue.Body.String()) != `{"error":"issue not found"}` {
		t.Fatalf("missing issue = %d %s", missingIssue.Code, missingIssue.Body.String())
	}
	for index := 0; index < 50; index++ {
		_, err := module.IssueMetadataLocal().PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: fmt.Sprintf("limit_%02d", index), ValueJson: "true"})
		if err != nil {
			t.Fatal(err)
		}
	}
	overflow := request(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/overflow", `{"value":true}`, true)
	if overflow.Code != 400 || strings.TrimSpace(overflow.Body.String()) != `{"error":"metadata cannot exceed 50 keys"}` {
		t.Fatalf("overflow = %d %s", overflow.Code, overflow.Body.String())
	}
	large := request(http.MethodPut, "/api/issues/"+created.Issue.Id+"/metadata/limit_00", `{"value":"`+strings.Repeat("x", 8192)+`"}`, true)
	if large.Code != 400 || strings.TrimSpace(large.Body.String()) != `{"error":"metadata exceeds the 8KB size limit"}` {
		t.Fatalf("large = %d %s", large.Code, large.Body.String())
	}
}

func TestIssueMetadataSameKeyLastCommitterAndRollback(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, time.Now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "atomic"})
	if err != nil {
		t.Fatal(err)
	}
	metadata := module.IssueMetadataLocal()
	if _, err := metadata.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "same", ValueJson: "1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := metadata.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "same", ValueJson: "2"}); err != nil {
		t.Fatal(err)
	}
	got, _ := metadata.GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id})
	if fmt.Sprint(got.Metadata["same"]) != "2" {
		t.Fatalf("same key=%v", got.Metadata)
	}
	var before, updated string
	_ = db.QueryRow(`SELECT metadata,updated_at FROM workspace_issues WHERE id=?`, created.Issue.Id).Scan(&before, &updated)
	if _, err := db.Exec(`CREATE TRIGGER fail_metadata_update BEFORE UPDATE OF metadata ON workspace_issues BEGIN SELECT RAISE(ABORT,'forced metadata rollback'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := metadata.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "valid", ValueJson: `true`}); err == nil {
		t.Fatal("expected persistence error")
	}
	if _, err := db.Exec(`DROP TRIGGER fail_metadata_update`); err != nil {
		t.Fatal(err)
	}
	var after, afterUpdated string
	_ = db.QueryRow(`SELECT metadata,updated_at FROM workspace_issues WHERE id=?`, created.Issue.Id).Scan(&after, &afterUpdated)
	if before != after || updated != afterUpdated {
		t.Fatalf("rollback changed row: %s/%s %s/%s", before, after, updated, afterUpdated)
	}
}

func TestIssueMetadataConcurrentSameKeyMatchesLastCompletion(t *testing.T) {
	db := openWorkspaceTestDB(t)
	db.SetMaxOpenConns(8)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, time.Now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "same key concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	delegate, err := persistence.NewIssueMetadataRepository(persistence.Config{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	gated := newGatedMetadataRepository(delegate)
	actors := &workspaceActorCatalog{actors: map[string]bool{"workspace-1/member/member-1": true}}
	service, err := application.NewIssueMetadataUseCase(gated, &workspaceAccessStub{}, actors, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	errs := make(chan error, 2)
	go func() {
		_, err := service.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "same", ValueJson: "1"})
		errs <- err
	}()
	<-gated.enteredA
	go func() {
		_, err := service.PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "same", ValueJson: "2"})
		errs <- err
	}()
	<-gated.enteredB
	close(gated.allowA)
	<-gated.committedA
	close(gated.allowB)
	<-gated.committedB
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	got, err := module.IssueMetadataLocal().GetIssueMetadata(ctx, contract.GetIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got.Metadata["same"]) != "2" || strings.Join(gated.commitOrder, ",") != "A,B" {
		t.Fatalf("commit order=%v final=%v", gated.commitOrder, got.Metadata["same"])
	}
}

type gatedMetadataRepository struct {
	application.IssueMetadataRepository
	enteredA, enteredB, allowA, allowB, committedA, committedB chan struct{}
	mu                                                         sync.Mutex
	commitOrder                                                []string
}

func newGatedMetadataRepository(delegate application.IssueMetadataRepository) *gatedMetadataRepository {
	return &gatedMetadataRepository{IssueMetadataRepository: delegate, enteredA: make(chan struct{}), enteredB: make(chan struct{}), allowA: make(chan struct{}), allowB: make(chan struct{}), committedA: make(chan struct{}), committedB: make(chan struct{})}
}
func (r *gatedMetadataRepository) PutMetadata(ctx context.Context, w, id, key string, value any, now time.Time) (string, map[string]any, time.Time, error) {
	label := "B"
	entered, allow, committed := r.enteredB, r.allowB, r.committedB
	if fmt.Sprint(value) == "1" {
		label = "A"
		entered, allow, committed = r.enteredA, r.allowA, r.committedA
	}
	close(entered)
	<-allow
	issueID, values, updated, err := r.IssueMetadataRepository.PutMetadata(ctx, w, id, key, value, now)
	if err == nil {
		r.mu.Lock()
		r.commitOrder = append(r.commitOrder, label)
		r.mu.Unlock()
	}
	close(committed)
	if label == "A" {
		<-r.committedB
	}
	return issueID, values, updated, err
}

func TestIssueMetadataOverlapsMainlineUpdate(t *testing.T) {
	db := openWorkspaceTestDB(t)
	db.SetMaxOpenConns(8)
	seedWorkspace(t, db, "workspace-1", "Acme", "acme")
	module := newIssueMainlineModule(t, db, &issueIDSequence{}, time.Now)
	ctx := contract.WithWorkspaceActor(context.Background(), "member", "member-1")
	created, err := module.IssueLocal().CreateIssue(ctx, contract.CreateIssueRequest{WorkspaceId: "workspace-1", Title: "overlap"})
	if err != nil {
		t.Fatal(err)
	}
	baseRepo, _ := persistence.NewIssueRepository(persistence.Config{DB: db})
	projects, _ := persistence.NewProjectRepository(persistence.Config{DB: db})
	pause := &pausingIssueRepository{IssueRepository: baseRepo, entered: make(chan struct{}), resume: make(chan struct{})}
	actors := &workspaceActorCatalog{actors: map[string]bool{"workspace-1/member/member-1": true}}
	usecase, err := application.NewIssueUseCase(pause, projects, &workspaceAccessStub{}, actors, &workspaceAssetCatalog{assets: map[string]bool{}}, func(context.Context) (string, error) { return "unused", nil }, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	title := "mainline updated"
	done := make(chan error, 1)
	go func() {
		_, err := usecase.UpdateIssue(ctx, contract.UpdateIssueRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Title: &title})
		done <- err
	}()
	<-pause.entered
	if _, err := module.IssueMetadataLocal().PutIssueMetadata(ctx, contract.PutIssueMetadataRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id, Key: "overlap", ValueJson: "true"}); err != nil {
		t.Fatal(err)
	}
	close(pause.resume)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got, err := module.IssueLocal().GetIssue(ctx, contract.GetIssueRequest{WorkspaceId: "workspace-1", IssueId: created.Issue.Id})
	if err != nil || got.Issue.Title != title || got.Issue.Metadata["overlap"] != true {
		t.Fatalf("overlap result=%#v err=%v", got.Issue, err)
	}
}

type pausingIssueRepository struct {
	application.IssueRepository
	entered chan struct{}
	resume  chan struct{}
}

func (r *pausingIssueRepository) Update(ctx context.Context, value issueDomain.Issue) error {
	close(r.entered)
	<-r.resume
	return r.IssueRepository.Update(ctx, value)
}
