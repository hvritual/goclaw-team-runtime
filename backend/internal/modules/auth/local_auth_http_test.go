package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func TestLocalAuthProjectsPersistedOnboardedAtAcrossRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "auth.db")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", databasePath+"?_pragma=busy_timeout(5000)")
		if err != nil {
			t.Fatal(err)
		}
		if err := MigrateSqlite(context.Background(), db); err != nil {
			db.Close()
			t.Fatal(err)
		}
		return db
	}
	const onboardedAt = "2026-08-14T01:02:03Z"
	const createdAt = "2026-08-13T01:02:03Z"
	db := open()
	if _, err := db.Exec(`INSERT INTO auth_users(id,name,email,onboarded_at,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
		"fixture-user", "Fixture", "fixture@example.com", onboardedAt, createdAt, createdAt); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)`,
		"fixture-member", "fixture-workspace", "fixture-user", "owner", createdAt); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_workspace_membership_roots(workspace_id,user_id,member_id,created_at) VALUES(?,?,?,?)`,
		"fixture-workspace", "fixture-user", "fixture-member", createdAt); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()
	expectedFixture := map[string]any{
		"id": "fixture-user", "name": "Fixture", "email": "fixture@example.com",
		"avatar_url": nil, "onboarded_at": onboardedAt, "onboarding_questionnaire": map[string]any{},
		"starter_content_state": nil, "language": nil, "profile_description": "", "timezone": nil,
		"created_at": createdAt, "updated_at": createdAt,
	}

	for restart := 0; restart < 2; restart++ {
		db = open()
		now := time.Date(2026, 8, 14, 2, 3, 4, 0, time.UTC)
		module, err := NewWithSqliteLocalAuth(SqlitePersistenceConfig{DB: db}, LocalAuthConfig{
			VerificationCode: "888888", SessionTTL: time.Hour,
			Now:   func() time.Time { return now },
			NewID: func(context.Context) (string, error) { return fmt.Sprintf("restart-token-%d", restart), nil },
		})
		if err != nil {
			db.Close()
			t.Fatal(err)
		}
		server := kratoshttp.NewServer()
		module.RegisterHTTP(server)
		verified := authRequest(server, http.MethodPost, "/auth/verify-code", `{"email":"fixture@example.com","code":"888888"}`, nil)
		if verified.Code != http.StatusOK {
			db.Close()
			t.Fatalf("restart %d verify = %d %s", restart, verified.Code, verified.Body.String())
		}
		var login map[string]any
		if err := json.Unmarshal(verified.Body.Bytes(), &login); err != nil {
			db.Close()
			t.Fatal(err)
		}
		if login["token"] != fmt.Sprintf("restart-token-%d", restart) || !reflect.DeepEqual(login["user"], expectedFixture) {
			db.Close()
			t.Fatalf("restart %d login = %#v", restart, login)
		}
		me := authRequest(server, http.MethodGet, "/api/me", "", map[string]string{"Authorization": "Bearer " + fmt.Sprintf("restart-token-%d", restart)})
		var currentUser map[string]any
		if me.Code != http.StatusOK || json.Unmarshal(me.Body.Bytes(), &currentUser) != nil || !reflect.DeepEqual(currentUser, expectedFixture) {
			db.Close()
			t.Fatalf("restart %d me = %d %#v", restart, me.Code, currentUser)
		}
		db.Close()
	}

	db = open()
	defer db.Close()
	newUserNow := time.Date(2026, 8, 14, 3, 4, 5, 0, time.UTC)
	module, err := NewWithSqliteLocalAuth(SqlitePersistenceConfig{DB: db}, LocalAuthConfig{
		VerificationCode: "888888", SessionTTL: time.Hour,
		Now:   func() time.Time { return newUserNow },
		NewID: func(context.Context) (string, error) { return "new-user-token", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)
	newUser := authRequest(server, http.MethodPost, "/auth/verify-code", `{"email":"new@example.com","code":"888888"}`, nil)
	if newUser.Code != http.StatusOK {
		t.Fatalf("new user = %d %s", newUser.Code, newUser.Body.String())
	}
	var newLogin map[string]any
	if err := json.Unmarshal(newUser.Body.Bytes(), &newLogin); err != nil {
		t.Fatal(err)
	}
	newUserBody, ok := newLogin["user"].(map[string]any)
	if !ok {
		t.Fatalf("new login user = %#v", newLogin["user"])
	}
	newID, _ := newUserBody["id"].(string)
	expectedNew := map[string]any{
		"id": newID, "name": "new", "email": "new@example.com",
		"avatar_url": nil, "onboarded_at": nil, "onboarding_questionnaire": map[string]any{},
		"starter_content_state": nil, "language": nil, "profile_description": "", "timezone": nil,
		"created_at": newUserNow.Format(time.RFC3339Nano), "updated_at": newUserNow.Format(time.RFC3339Nano),
	}
	if newLogin["token"] != "new-user-token" || newID == "" || !reflect.DeepEqual(newUserBody, expectedNew) {
		t.Fatalf("new login = %#v", newLogin)
	}
	newMe := authRequest(server, http.MethodGet, "/api/me", "", map[string]string{"Authorization": "Bearer new-user-token"})
	var newCurrent map[string]any
	if newMe.Code != http.StatusOK || json.Unmarshal(newMe.Body.Bytes(), &newCurrent) != nil || !reflect.DeepEqual(newCurrent, expectedNew) {
		t.Fatalf("new me = %d %#v", newMe.Code, newCurrent)
	}
}

func TestLocalAuthHTTPJourneyAndRevocation(t *testing.T) {
	db := openAuthTestDB(t)
	now := time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC)
	module, err := NewWithSqliteLocalAuth(
		SqlitePersistenceConfig{DB: db},
		LocalAuthConfig{
			VerificationCode: "888888",
			SessionTTL:       7 * 24 * time.Hour,
			Now:              func() time.Time { return now },
			NewID: func(context.Context) (string, error) {
				return "session-token", nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)

	badEmail := authRequest(server, http.MethodPost, "/auth/send-code", `{"email":"invalid"}`, nil)
	assertAuthResponse(t, badEmail, http.StatusBadRequest, `{"error":"valid email is required"}`)
	sent := authRequest(server, http.MethodPost, "/auth/send-code", `{"email":"Owner@Example.com"}`, nil)
	assertAuthResponse(t, sent, http.StatusNoContent, "")
	badCode := authRequest(server, http.MethodPost, "/auth/verify-code", `{"email":"owner@example.com","code":"000000"}`, nil)
	assertAuthResponse(t, badCode, http.StatusUnauthorized, `{"error":"invalid verification code"}`)

	verified := authRequest(server, http.MethodPost, "/auth/verify-code", `{"email":"Owner@Example.com","code":"888888"}`, nil)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d: %s", verified.Code, verified.Body.String())
	}
	var login struct {
		Token string `json:"token"`
		User  struct {
			ID, Name, Email string
		} `json:"user"`
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	if login.Token != "session-token" || login.User.ID == "" || login.User.Name != "owner" || login.User.Email != "owner@example.com" {
		t.Fatalf("login = %#v", login)
	}
	authCookie, csrfCookie := responseCookies(t, verified)
	if !authCookie.HttpOnly || csrfCookie.HttpOnly || authCookie.Value != login.Token || csrfCookie.Value == "" || authCookie.SameSite != http.SameSiteStrictMode || csrfCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookies auth=%#v csrf=%#v", authCookie, csrfCookie)
	}

	bearerMe := authRequest(server, http.MethodGet, "/api/me", "", map[string]string{"Authorization": "Bearer " + login.Token})
	if bearerMe.Code != http.StatusOK || !strings.Contains(bearerMe.Body.String(), `"email":"owner@example.com"`) {
		t.Fatalf("bearer me = %d %s", bearerMe.Code, bearerMe.Body.String())
	}
	cookieMe := authRequestWithCookies(server, http.MethodGet, "/api/me", "", nil, authCookie)
	if cookieMe.Code != http.StatusOK {
		t.Fatalf("cookie me = %d %s", cookieMe.Code, cookieMe.Body.String())
	}

	missingCSRF := authRequestWithCookies(server, http.MethodPost, "/auth/logout", "", nil, authCookie)
	assertAuthResponse(t, missingCSRF, http.StatusForbidden, `{"error":"invalid CSRF token"}`)
	loggedOut := authRequestWithCookies(server, http.MethodPost, "/auth/logout", "", map[string]string{"X-CSRF-Token": csrfCookie.Value}, authCookie)
	assertAuthResponse(t, loggedOut, http.StatusNoContent, "")
	clearedAuth, clearedCSRF := responseCookies(t, loggedOut)
	if clearedAuth.MaxAge != -1 || clearedCSRF.MaxAge != -1 {
		t.Fatalf("cleared cookies auth=%#v csrf=%#v", clearedAuth, clearedCSRF)
	}
	revoked := authRequest(server, http.MethodGet, "/api/me", "", map[string]string{"Authorization": "Bearer " + login.Token})
	assertAuthResponse(t, revoked, http.StatusUnauthorized, `{"error":"invalid token"}`)
}

func TestLocalAuthExpiredAndMissingIdentityFailClosed(t *testing.T) {
	db := openAuthTestDB(t)
	now := time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC)
	module, err := NewWithSqliteLocalAuth(SqlitePersistenceConfig{DB: db}, LocalAuthConfig{
		VerificationCode: "888888", SessionTTL: time.Minute,
		Now:   func() time.Time { return now },
		NewID: func(context.Context) (string, error) { return "expiring-token", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)
	missing := authRequest(server, http.MethodGet, "/api/me", "", nil)
	assertAuthResponse(t, missing, http.StatusUnauthorized, `{"error":"authentication required"}`)
	verified := authRequest(server, http.MethodPost, "/auth/verify-code", `{"email":"owner@example.com","code":"888888"}`, nil)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify = %d %s", verified.Code, verified.Body.String())
	}
	now = now.Add(2 * time.Minute)
	expired := authRequest(server, http.MethodGet, "/api/me", "", map[string]string{"Authorization": "Bearer expiring-token"})
	assertAuthResponse(t, expired, http.StatusUnauthorized, `{"error":"invalid token"}`)
}

func TestLocalAuthRequiresExplicitDevelopmentConfiguration(t *testing.T) {
	db := openAuthTestDB(t)
	for _, code := range []string{"", "abc", "12345", "1234567"} {
		if _, err := NewWithSqliteLocalAuth(SqlitePersistenceConfig{DB: db}, LocalAuthConfig{VerificationCode: code}); err == nil {
			t.Fatalf("local auth accepted invalid development verification code %q", code)
		}
	}
}

func TestLocalAuthConcurrentFirstLoginUsesOneUser(t *testing.T) {
	db := openAuthTestDB(t)
	var sequence atomic.Int64
	module, err := NewWithSqliteLocalAuth(SqlitePersistenceConfig{DB: db}, LocalAuthConfig{
		VerificationCode: "888888", SessionTTL: time.Hour,
		NewID: func(context.Context) (string, error) { return fmt.Sprintf("token-%d", sequence.Add(1)), nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := kratoshttp.NewServer()
	module.RegisterHTTP(server)
	const count = 12
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, count)
	var group sync.WaitGroup
	for index := 0; index < count; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			responses <- authRequest(server, http.MethodPost, "/auth/verify-code", `{"email":"concurrent@example.com","code":"888888"}`, nil)
		}()
	}
	close(start)
	group.Wait()
	close(responses)
	for response := range responses {
		if response.Code != http.StatusOK {
			t.Fatalf("concurrent verify = %d %s", response.Code, response.Body.String())
		}
	}
	var users int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_users WHERE email='concurrent@example.com'`).Scan(&users); err != nil || users != 1 {
		t.Fatalf("users = %d, %v", users, err)
	}
	var sessions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_sessions`).Scan(&sessions); err != nil || sessions != count {
		t.Fatalf("sessions = %d, %v", sessions, err)
	}
}

func authRequest(server *kratoshttp.Server, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	return authRequestWithCookies(server, method, path, body, headers)
}

func authRequestWithCookies(server *kratoshttp.Server, method, path, body string, headers map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	return response
}

func assertAuthResponse(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status || strings.TrimSpace(response.Body.String()) != body {
		t.Fatalf("response = %d %q, want %d %q", response.Code, strings.TrimSpace(response.Body.String()), status, body)
	}
}

func responseCookies(t *testing.T, response *httptest.ResponseRecorder) (*http.Cookie, *http.Cookie) {
	t.Helper()
	var authCookie, csrfCookie *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		switch cookie.Name {
		case "multica_auth":
			authCookie = cookie
		case "multica_csrf":
			csrfCookie = cookie
		}
	}
	if authCookie == nil || csrfCookie == nil {
		t.Fatalf("missing auth cookies: %s", fmt.Sprint(response.Header().Values("Set-Cookie")))
	}
	return authCookie, csrfCookie
}
