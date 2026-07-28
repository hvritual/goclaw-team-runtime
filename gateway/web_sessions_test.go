package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSessionLifecycle(t *testing.T) {
	store := newWebSessionStore(time.Hour)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	token, session, err := store.create("alice")
	if err != nil {
		t.Fatal(err)
	}
	if session.PrincipalID != "alice" || session.CSRFToken == "" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if got, ok := store.authenticate(token); !ok || got.PrincipalID != "alice" {
		t.Fatalf("session was not authenticated: %+v %v", got, ok)
	}

	now = now.Add(2 * time.Hour)
	if _, ok := store.authenticate(token); ok {
		t.Fatal("expired session was accepted")
	}
}

func TestWebSessionRevocation(t *testing.T) {
	store := newWebSessionStore(time.Hour)
	token, _, err := store.create("alice")
	if err != nil {
		t.Fatal(err)
	}
	store.revoke(token)
	if _, ok := store.authenticate(token); ok {
		t.Fatal("revoked session was accepted")
	}
}

func TestWebSocketOriginPolicy(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "native client", host: "goclaw.example.com", want: true},
		{name: "same origin", host: "goclaw.example.com", origin: "https://goclaw.example.com", want: true},
		{name: "same origin with port", host: "localhost:28789", origin: "http://localhost:28789", want: true},
		{name: "vite direct cross-port", host: "localhost:28789", origin: "http://localhost:5173", want: false},
		{name: "optional obsidian client", host: "localhost:28789", origin: "app://obsidian.md", want: true},
		{name: "cross site", host: "goclaw.example.com", origin: "https://evil.example", want: false},
		{name: "malformed", host: "goclaw.example.com", origin: "not a url", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Host: tt.host, Header: make(http.Header)}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := checkWebSocketOrigin(req); got != tt.want {
				t.Fatalf("checkWebSocketOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCSRFHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/rpc", nil)
	if err != nil {
		t.Fatal(err)
	}
	if validCSRFHeader(req, "expected") {
		t.Fatal("missing CSRF header was accepted")
	}
	req.Header.Set(csrfHeader, "expected")
	if !validCSRFHeader(req, "expected") {
		t.Fatal("valid CSRF header was rejected")
	}
	if validCSRFHeader(req, "different") {
		t.Fatal("invalid CSRF header was accepted")
	}
}

func TestBrowserSessionRPCRequiresCSRF(t *testing.T) {
	fixture := newGatewayTeamFixture(t)
	store := newWebSessionStore(time.Hour)
	token, session, err := store.create(fixture.alice.ID)
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetTeamControlService(&fixture.service)
	handler.registry.Register("health", func(
		_ string,
		_ map[string]interface{},
	) (interface{}, error) {
		return "ok", nil
	})
	server := &Server{
		enableAuth:  true,
		authToken:   "outer-test-token",
		teamSvc:     &fixture.service,
		webSessions: store,
		handler:     handler,
		wsConfig:    &WebSocketConfig{MaxMessageSize: 1024},
	}

	tests := []struct {
		name       string
		csrf       string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusForbidden},
		{name: "wrong", csrf: "wrong", wantStatus: http.StatusForbidden},
		{name: "correct", csrf: session.CSRFToken, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"http://localhost/rpc",
				strings.NewReader(`{"jsonrpc":"2.0","id":"csrf","method":"health"}`),
			)
			request.AddCookie(&http.Cookie{Name: teamSessionCookie, Value: token})
			if tt.csrf != "" {
				request.Header.Set(csrfHeader, tt.csrf)
			}
			response := httptest.NewRecorder()
			server.handleJSONRPC(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestWebSessionCreateRejectsCrossSiteRequest(t *testing.T) {
	server := &Server{webSessions: newWebSessionStore(time.Hour)}
	request, err := http.NewRequest(http.MethodPost, "https://goclaw.example/auth/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "goclaw.example"
	request.Header.Set("Origin", "https://evil.example")
	response := &testResponseWriter{header: make(http.Header)}
	server.handleWebSessionCreate(response, request)
	if response.status != http.StatusForbidden {
		t.Fatalf("cross-site login status = %d", response.status)
	}
}

func TestWebSessionCreateRejectsUnsafeBrowserContext(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		origin       string
		secFetchSite string
	}{
		{
			name:   "same hostname different port",
			host:   "localhost:28789",
			origin: "http://localhost:5173",
		},
		{
			name:         "browser reports cross-site",
			host:         "localhost:28789",
			origin:       "http://localhost:28789",
			secFetchSite: "cross-site",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{webSessions: newWebSessionStore(time.Hour)}
			request := httptest.NewRequest(
				http.MethodPost,
				"http://"+tt.host+"/auth/session",
				strings.NewReader(`{}`),
			)
			request.Host = tt.host
			request.Header.Set("Origin", tt.origin)
			if tt.secFetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			response := httptest.NewRecorder()
			server.handleWebSessionCreate(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
		})
	}
}

type testResponseWriter struct {
	header http.Header
	status int
}

func (w *testResponseWriter) Header() http.Header { return w.header }
func (w *testResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(body), nil
}
func (w *testResponseWriter) WriteHeader(status int) { w.status = status }
