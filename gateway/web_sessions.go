package gateway

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	teamSessionCookie = "goclaw_team_session"
	csrfHeader        = "X-GoClaw-CSRF"
)

type webSession struct {
	PrincipalID string    `json:"principal_id"`
	CSRFToken   string    `json:"csrf_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type webSessionStore struct {
	mu       sync.Mutex
	ttl      time.Duration
	sessions map[[sha256.Size]byte]webSession
	now      func() time.Time
}

func newWebSessionStore(ttl time.Duration) *webSessionStore {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &webSessionStore{
		ttl:      ttl,
		sessions: make(map[[sha256.Size]byte]webSession),
		now:      time.Now,
	}
}

func (s *webSessionStore) create(principalID string) (string, webSession, error) {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return "", webSession{}, errors.New("principal id is required")
	}
	token, err := randomWebSecret()
	if err != nil {
		return "", webSession{}, err
	}
	csrf, err := randomWebSecret()
	if err != nil {
		return "", webSession{}, err
	}
	session := webSession{
		PrincipalID: principalID,
		CSRFToken:   csrf,
		ExpiresAt:   s.now().UTC().Add(s.ttl),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpiredLocked()
	s.sessions[sha256.Sum256([]byte(token))] = session
	return token, session, nil
}

func (s *webSessionStore) authenticate(token string) (webSession, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return webSession{}, false
	}
	key := sha256.Sum256([]byte(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[key]
	if !ok {
		return webSession{}, false
	}
	if !session.ExpiresAt.After(s.now().UTC()) {
		delete(s.sessions, key)
		return webSession{}, false
	}
	return session, true
}

func (s *webSessionStore) revoke(token string) {
	if strings.TrimSpace(token) == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, sha256.Sum256([]byte(token)))
	s.mu.Unlock()
}

func (s *webSessionStore) removeExpiredLocked() {
	now := s.now().UTC()
	for key, session := range s.sessions {
		if !session.ExpiresAt.After(now) {
			delete(s.sessions, key)
		}
	}
}

func randomWebSecret() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (s *Server) handleWebSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		s.handleWebSessionStatus(w, r)
	case http.MethodPost:
		s.handleWebSessionCreate(w, r)
	case http.MethodDelete:
		s.handleWebSessionDelete(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWebSessionCreate(w http.ResponseWriter, r *http.Request) {
	if !checkWebSocketOrigin(r) ||
		strings.EqualFold(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")), "cross-site") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	s.mu.RLock()
	service := s.teamSvc
	s.mu.RUnlock()
	if service == nil {
		http.Error(w, "Team control is not enabled", http.StatusServiceUnavailable)
		return
	}

	var input struct {
		GatewayToken string `json:"gateway_token"`
		UserToken    string `json:"user_token"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "Invalid login request", http.StatusBadRequest)
		return
	}
	if s.enableAuth {
		if input.GatewayToken == "" ||
			subtle.ConstantTimeCompare([]byte(input.GatewayToken), []byte(s.authToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	if s.webSessions == nil {
		http.Error(w, "Web sessions are not available", http.StatusServiceUnavailable)
		return
	}
	user, err := service.AuthenticateAccessToken(strings.TrimSpace(input.UserToken))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token, session, err := s.webSessions.create(user.ID)
	if err != nil {
		http.Error(w, "Unable to create session", http.StatusInternalServerError)
		return
	}
	setWebSessionCookie(w, r, token, session.ExpiresAt)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(session)
}

func (s *Server) handleWebSessionStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.authenticateWebSession(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(session)
}

func (s *Server) handleWebSessionDelete(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(teamSessionCookie)
	if err == nil {
		_, session, ok := s.authenticateWebSession(r)
		if ok &&
			!validCSRFHeader(r, session.CSRFToken) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if s.webSessions != nil {
			s.webSessions.revoke(cookie.Value)
		}
	}
	clearWebSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authenticateWebSession(r *http.Request) (string, *webSession, bool) {
	if s.webSessions == nil {
		return "", nil, false
	}
	cookie, err := r.Cookie(teamSessionCookie)
	if err != nil {
		return "", nil, false
	}
	session, ok := s.webSessions.authenticate(cookie.Value)
	if !ok {
		return "", nil, false
	}
	return session.PrincipalID, &session, true
}

func validCSRFHeader(r *http.Request, expected string) bool {
	actual := strings.TrimSpace(r.Header.Get(csrfHeader))
	if actual == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func setWebSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     teamSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteStrictMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func clearWebSessionCookie(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{teamSessionCookie, "dashboard_token"} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   requestIsSecure(r),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
		})
	}
}

func requestIsSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func checkWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if origin == "app://obsidian.md" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}
