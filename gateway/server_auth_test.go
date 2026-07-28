package gateway

import (
	"encoding/base64"
	"mime"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"
)

func TestAuthenticateWebSocketSubprotocolToken(t *testing.T) {
	server := &Server{authToken: "密钥-token"}
	request := httptest.NewRequest("GET", "http://localhost/ws", nil)
	encoded := base64.RawURLEncoding.EncodeToString([]byte("密钥-token"))
	request.Header.Set("Sec-WebSocket-Protocol", "goclaw.v1, goclaw.bearer."+encoded)
	if !server.authenticateWebSocket(request) {
		t.Fatal("expected WebSocket subprotocol token to authenticate")
	}

	bad := httptest.NewRequest("GET", "http://localhost/ws", nil)
	bad.Header.Set("Sec-WebSocket-Protocol", "goclaw.v1, goclaw.bearer.bad")
	if server.authenticateWebSocket(bad) {
		t.Fatal("expected invalid WebSocket subprotocol token to fail")
	}
}

func TestHTTPRPCRequiresBearerWhenGatewayAuthIsEnabled(t *testing.T) {
	server := &Server{
		enableAuth: true,
		authToken:  "a-strong-test-token",
		wsConfig:   &WebSocketConfig{MaxMessageSize: 1024},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"http://localhost/rpc",
		strings.NewReader(`{"jsonrpc":"2.0","id":"1","method":"health"}`),
	)
	response := httptest.NewRecorder()
	server.handleJSONRPC(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated RPC rejection, got %d", response.Code)
	}

	request.Header.Set("Authorization", "Bearer a-strong-test-token")
	if !server.authenticateHTTP(request) {
		t.Fatal("expected matching HTTP Bearer token to authenticate")
	}
	request.Header.Set("Authorization", "Bearer wrong")
	if server.authenticateHTTP(request) {
		t.Fatal("expected invalid HTTP Bearer token to fail")
	}
	request.Header.Del("Authorization")
	request.AddCookie(&http.Cookie{Name: "dashboard_token", Value: "a-strong-test-token"})
	if !server.authenticateHTTP(request) {
		t.Fatal("expected the HttpOnly dashboard cookie to authenticate same-origin RPC")
	}
}

func TestWebSocketAcceptsHttpOnlyGatewayCookie(t *testing.T) {
	server := &Server{authToken: "a-strong-test-token"}
	request := httptest.NewRequest(http.MethodGet, "http://localhost/ws", nil)
	request.AddCookie(&http.Cookie{
		Name:  "dashboard_token",
		Value: "a-strong-test-token",
	})
	if !server.authenticateWebSocket(request) {
		t.Fatal("expected matching HttpOnly gateway cookie to authenticate WebSocket")
	}
}

func TestShortLivedWebSessionSatisfiesOuterGatewayAuth(t *testing.T) {
	store := newWebSessionStore(time.Hour)
	token, _, err := store.create("alice")
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{
		enableAuth:  true,
		authToken:   "a-strong-test-token",
		webSessions: store,
	}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/rpc", nil)
	request.AddCookie(&http.Cookie{Name: teamSessionCookie, Value: token})
	if server.authenticateHTTP(request) {
		t.Fatal("web session must not be confused with the long-lived Gateway token")
	}
	if !server.authenticateHTTPPerimeter(request) {
		t.Fatal("valid short-lived web session should satisfy the outer perimeter")
	}
	if !server.authenticateWebSocketPerimeter(request) {
		t.Fatal("valid short-lived web session should satisfy the WebSocket perimeter")
	}
}

const expectedTeamConsoleCSP = "default-src 'self'; script-src 'self'; " +
	"style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:; " +
	"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

func assertTeamConsoleSecurityHeaders(t *testing.T, header http.Header) {
	t.Helper()
	expected := map[string]string{
		"Content-Security-Policy": expectedTeamConsoleCSP,
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	}
	for name, want := range expected {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestTeamConsoleShellIsPublicButHardened(t *testing.T) {
	handler := TeamConsoleHandler()

	getRequest := httptest.NewRequest(http.MethodGet, "http://localhost/dashboard/", nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf(
			"GET team console status = %d, location = %q",
			getResponse.Code,
			getResponse.Header().Get("Location"),
		)
	}
	if got := getResponse.Header().Get("Location"); got != "" {
		t.Fatalf("GET team console unexpectedly redirected to %q", got)
	}
	if got := getResponse.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("GET Content-Type = %q", got)
	}
	if !strings.Contains(getResponse.Body.String(), `<div id="root"></div>`) {
		t.Fatal("GET response does not contain the Team Console shell marker")
	}
	assertTeamConsoleSecurityHeaders(t, getResponse.Header())
	if got := getResponse.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("GET index cache policy = %q", got)
	}
	getContentLength := getResponse.Header().Get("Content-Length")
	if getContentLength == "" {
		t.Fatal("GET shell has no Content-Length")
	}

	headRequest := httptest.NewRequest(http.MethodHead, "http://localhost/dashboard/", nil)
	headResponse := httptest.NewRecorder()
	handler.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK {
		t.Fatalf(
			"HEAD team console status = %d, location = %q",
			headResponse.Code,
			headResponse.Header().Get("Location"),
		)
	}
	if headResponse.Body.Len() != 0 {
		t.Fatalf("HEAD response body length = %d, want 0", headResponse.Body.Len())
	}
	if got := headResponse.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("HEAD Content-Type = %q", got)
	}
	if got := headResponse.Header().Get("Content-Length"); got != getContentLength {
		t.Errorf("HEAD Content-Length = %q, want GET value %q", got, getContentLength)
	}
	if got := headResponse.Header().Get("Location"); got != "" {
		t.Errorf("HEAD team console unexpectedly redirected to %q", got)
	}
	assertTeamConsoleSecurityHeaders(t, headResponse.Header())
	if got := headResponse.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("HEAD index cache policy = %q", got)
	}
}

func TestTeamConsoleCanonicalRouteAndAssetCaching(t *testing.T) {
	handler := TeamConsoleHandler()

	t.Run("canonical dashboard redirect", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://localhost/dashboard", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMovedPermanently {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusMovedPermanently)
		}
		if got := response.Header().Get("Location"); got != "/dashboard/" {
			t.Fatalf("Location = %q, want /dashboard/", got)
		}
	})

	t.Run("hashed assets are immutable and correctly typed", func(t *testing.T) {
		entries, err := uiDist.ReadDir("ui_dist/assets")
		if err != nil {
			t.Fatal(err)
		}
		expectedTypes := map[string]string{
			".js":  mime.TypeByExtension(".js"),
			".css": mime.TypeByExtension(".css"),
		}
		seen := make(map[string]bool, len(expectedTypes))
		for _, entry := range entries {
			extension := path.Ext(entry.Name())
			expectedType, required := expectedTypes[extension]
			if entry.IsDir() || !required || seen[extension] {
				continue
			}
			stem := strings.TrimSuffix(entry.Name(), extension)
			hashSeparator := strings.LastIndex(stem, "-")
			if hashSeparator < 0 || len(stem)-hashSeparator-1 < 8 {
				t.Fatalf("asset %q does not have a build hash", entry.Name())
			}

			request := httptest.NewRequest(
				http.MethodGet,
				"http://localhost/assets/"+entry.Name(),
				nil,
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("%s status = %d", entry.Name(), response.Code)
			}
			if got := response.Header().Get("Content-Type"); got != expectedType {
				t.Errorf("%s Content-Type = %q, want %q", entry.Name(), got, expectedType)
			}
			if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("%s nosniff = %q", entry.Name(), got)
			}
			if got := response.Header().Get("Cache-Control"); got !=
				"public, max-age=31536000, immutable" {
				t.Errorf("%s cache policy = %q", entry.Name(), got)
			}
			seen[extension] = true
		}
		for extension := range expectedTypes {
			if !seen[extension] {
				t.Errorf("no embedded hashed %s asset was tested", extension)
			}
		}
	})

	t.Run("unknown route does not return the shell", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodGet,
			"http://localhost/dashboard/not-found.js",
			nil,
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
		if got := response.Header().Get("Location"); got != "" {
			t.Errorf("unknown route redirected to %q", got)
		}
		if strings.Contains(response.Body.String(), `<div id="root"></div>`) {
			t.Fatal("unknown route returned the Team Console shell")
		}
		assertTeamConsoleSecurityHeaders(t, response.Header())
	})
}

func TestLocalDashboardCheckDoesNotTrustProxyLoopback(t *testing.T) {
	direct := httptest.NewRequest(http.MethodGet, "http://localhost/dashboard/", nil)
	direct.RemoteAddr = "127.0.0.1:12345"
	if !isLocalRequest(direct) {
		t.Fatal("direct loopback request should be local")
	}

	proxied := httptest.NewRequest(http.MethodGet, "http://localhost/dashboard/", nil)
	proxied.RemoteAddr = "127.0.0.1:12345"
	proxied.Header.Set("X-Forwarded-For", "203.0.113.10")
	if isLocalRequest(proxied) {
		t.Fatal("remote request through a loopback proxy must not inherit local access")
	}

	spoofed := httptest.NewRequest(http.MethodGet, "http://localhost/dashboard/", nil)
	spoofed.RemoteAddr = "203.0.113.10:12345"
	spoofed.Header.Set("X-Forwarded-For", "127.0.0.1")
	if isLocalRequest(spoofed) {
		t.Fatal("non-loopback peer must not spoof local access with forwarding headers")
	}
}

func TestProjectMetadata(t *testing.T) {
	metadata := projectMetadata(map[string]interface{}{
		"project_id": "project-alpha",
		"topic_id":   "architecture",
	})
	if metadata["project_id"] != "project-alpha" || metadata["topic_id"] != "architecture" {
		t.Fatalf("unexpected project metadata: %+v", metadata)
	}
}
