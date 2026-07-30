package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/multica-ai/multica/server/internal/auth"
	"github.com/redis/go-redis/v9"
)

const redisTestDB = 13

func newRedisTestClient(t *testing.T) *redis.Client {
	t.Helper()
	rawURL := os.Getenv("REDIS_TEST_URL")
	if rawURL == "" {
		t.Skip("REDIS_TEST_URL not set")
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		t.Fatalf("parse REDIS_TEST_URL: %v", err)
	}
	options.DB = redisTestDB
	client := redis.NewClient(options)
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		client.Close()
		t.Skipf("Redis unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = client.FlushDB(context.Background()).Err()
		_ = client.Close()
	})
	return client
}

func signedUserToken(t *testing.T, userID string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": "member@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	value, err := token.SignedString(auth.JWTSecret())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return value
}

func TestAuthRejectsMissingToken(t *testing.T) {
	handler := Auth(nil, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next must not be called")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestAuthAcceptsUserJWT(t *testing.T) {
	var gotUserID, gotEmail string
	handler := Auth(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.Header.Get("X-User-ID")
		gotEmail = r.Header.Get("X-User-Email")
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+signedUserToken(t, "member-1"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || gotUserID != "member-1" || gotEmail != "member@example.com" {
		t.Fatalf("unexpected auth result: status=%d user=%q email=%q", response.Code, gotUserID, gotEmail)
	}
}

func TestAuthClearsClientActorSource(t *testing.T) {
	var actorSource string
	handler := Auth(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorSource = r.Header.Get("X-Actor-Source")
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+signedUserToken(t, "member-1"))
	request.Header.Set("X-Actor-Source", "machine")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if actorSource != "" {
		t.Fatalf("expected actor source to be cleared, got %q", actorSource)
	}
}
