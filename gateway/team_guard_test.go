package gateway

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/smallnest/goclaw/memory/catalog"
)

func TestTeamGuardDeniesLegacyAndUnknownMethodsBeforeDispatch(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)

	invoked := false
	for _, method := range []string{"sessions.list", "future.global.dump"} {
		handler.registry.Register(method, func(
			_ string,
			_ map[string]interface{},
		) (interface{}, error) {
			invoked = true
			return "unsafe", nil
		})
		response := handler.HandleRequest(
			teamSessionID(fixture.bob.ID),
			&JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      method,
				Method:  method,
			},
		)
		if response.Error == nil ||
			!strings.Contains(response.Error.Message, "team mode") &&
				!strings.Contains(response.Error.Message, "authorization policy") {
			t.Fatalf("%s was not denied by team guard: %+v", method, response)
		}
	}
	if invoked {
		t.Fatal("denied method reached its registered handler")
	}

	handler.registry.Register("health", func(
		_ string,
		_ map[string]interface{},
	) (interface{}, error) {
		return "ok", nil
	})
	response := handler.HandleRequest(
		teamSessionID(fixture.bob.ID),
		&JSONRPCRequest{JSONRPC: "2.0", ID: "health", Method: "health"},
	)
	if response.Error != nil || response.Result != "ok" {
		t.Fatalf("health was not allowed: %+v", response)
	}
}

func TestTeamGuardScopesMemoryCatalogByProject(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	cfg := catalog.DefaultConfig()
	cfg.DatabasePath = filepath.Join(t.TempDir(), "catalog.db")
	service, err := catalog.NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()

	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.SetMemoryCatalog(service)

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "memory-status",
		Method:  "memory.catalog.status",
		Params: map[string]interface{}{
			"project_id": fixture.project.ID,
		},
	}
	if response := handler.HandleRequest(
		teamSessionID(fixture.viewer.ID),
		request,
	); response.Error == nil {
		t.Fatal("cross-project memory status was allowed")
	}
	if response := handler.HandleRequest(
		teamSessionID(fixture.bob.ID),
		request,
	); response.Error != nil {
		t.Fatalf("project member could not read memory status: %+v", response.Error)
	}
}
