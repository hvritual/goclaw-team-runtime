package controlplane

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpstreamIdentityResolverForwardsBearerAndBuildsSnapshot(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer desktop-token" || request.Header.Get("Cookie") != "" {
			t.Fatalf("unexpected forwarded credential: authorization=%q cookie=%q", request.Header.Get("Authorization"), request.Header.Get("Cookie"))
		}
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/me":
			_, _ = response.Write([]byte(`{"id":"member-1"}`))
		case "/api/workspaces/workspace-1":
			_, _ = response.Write([]byte(`{"id":"workspace-1","name":"Primary"}`))
		case "/api/workspaces/workspace-1/members":
			_, _ = response.Write([]byte(`[{"workspace_id":"workspace-1","user_id":"owner-1","role":"owner"},{"workspace_id":"workspace-1","user_id":"member-1","role":"member"}]`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer upstream.Close()

	resolver, err := NewUpstreamIdentityResolver(upstream.URL, &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/workspaces/workspace-1", nil)
	request.SetPathValue("workspace", "workspace-1")
	request.Header.Set("Authorization", "Bearer desktop-token")
	resolved, err := resolver(request)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Actor.ID != "member-1" || resolved.Actor.Kind != ActorHuman || resolved.Snapshot == nil || len(resolved.Snapshot.Members) != 2 {
		t.Fatalf("resolved identity = %#v", resolved)
	}
}

func TestUpstreamIdentityResolverRequiresCookieCSRFForMutation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/me":
			_, _ = response.Write([]byte(`{"id":"owner-1"}`))
		case "/api/workspaces/workspace-1":
			_, _ = response.Write([]byte(`{"id":"workspace-1","name":"Primary"}`))
		case "/api/workspaces/workspace-1/members":
			_, _ = response.Write([]byte(`[{"workspace_id":"workspace-1","user_id":"owner-1","role":"owner"}]`))
		}
	}))
	defer upstream.Close()
	resolver, _ := NewUpstreamIdentityResolver(upstream.URL, &http.Client{Timeout: time.Second})

	request := httptest.NewRequest(http.MethodPost, "/v1/workspaces/workspace-1/projects/project-1/commands", nil)
	request.SetPathValue("workspace", "workspace-1")
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-secret"})
	if _, err := resolver(request); !errors.Is(err, ErrDenied) {
		t.Fatalf("missing CSRF error = %v, want denied", err)
	}

	token := csrfToken("session-secret")
	request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	request.Header.Set("X-CSRF-Token", token)
	if _, err := resolver(request); err != nil {
		t.Fatalf("valid CSRF error = %v", err)
	}
}

func TestUpstreamIdentityResolverFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		user       string
		workspace  string
		members    string
		status     int
		delay      time.Duration
		timeout    time.Duration
		wantDenied bool
	}{
		{name: "actor absent", user: `{"id":"member-1"}`, workspace: `{"id":"workspace-1","name":"Primary"}`, members: `[{"workspace_id":"workspace-1","user_id":"owner-1","role":"owner"}]`, wantDenied: true},
		{name: "no owner", user: `{"id":"member-1"}`, workspace: `{"id":"workspace-1","name":"Primary"}`, members: `[{"workspace_id":"workspace-1","user_id":"member-1","role":"member"}]`},
		{name: "cross workspace member", user: `{"id":"member-1"}`, workspace: `{"id":"workspace-1","name":"Primary"}`, members: `[{"workspace_id":"workspace-2","user_id":"member-1","role":"owner"}]`},
		{name: "malformed", user: `{`},
		{name: "oversized", user: `{"id":"` + strings.Repeat("x", maxIdentityBody) + `"}`},
		{name: "unauthorized", status: http.StatusUnauthorized, wantDenied: true},
		{name: "forbidden", status: http.StatusForbidden, wantDenied: true},
		{name: "not found", status: http.StatusNotFound, wantDenied: true},
		{name: "upstream failure", status: http.StatusBadGateway},
		{name: "timeout", delay: 50 * time.Millisecond, timeout: 10 * time.Millisecond},
	}
	for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if test.delay > 0 {
					time.Sleep(test.delay)
				}
				if test.status != 0 {
					response.WriteHeader(test.status)
					return
				}
				switch request.URL.Path {
				case "/api/me":
					_, _ = response.Write([]byte(test.user))
				case "/api/workspaces/workspace-1":
					_, _ = response.Write([]byte(test.workspace))
				case "/api/workspaces/workspace-1/members":
					_, _ = response.Write([]byte(test.members))
				}
			}))
			defer upstream.Close()
			timeout := test.timeout
			if timeout == 0 {
				timeout = time.Second
			}
			resolver, _ := NewUpstreamIdentityResolver(upstream.URL, &http.Client{Timeout: timeout})
			request := httptest.NewRequest(http.MethodGet, "/v1/workspaces/workspace-1", nil)
			request.SetPathValue("workspace", "workspace-1")
			request.Header.Set("Authorization", "Bearer token")
			_, err := resolver(request)
			if test.wantDenied && !errors.Is(err, ErrDenied) {
				t.Fatalf("error = %v, want denied", err)
			}
			if !test.wantDenied && !errors.Is(err, ErrUnavailable) {
				t.Fatalf("error = %v, want unavailable", err)
			}
		})
	}
}

func csrfToken(secret string) string {
	nonce := []byte("0123456789abcdef")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(nonce)
	return hex.EncodeToString(nonce) + "." + hex.EncodeToString(mac.Sum(nil))
}
