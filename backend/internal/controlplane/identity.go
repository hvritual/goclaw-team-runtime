package controlplane

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authCookieName  = "multica_auth"
	csrfCookieName  = "multica_csrf"
	maxIdentityBody = 1 << 20
)

type TrustedWorkspaceSnapshot struct {
	ID      string
	Name    string
	ActorID string
	Members []TrustedMember
}

type TrustedMember struct {
	ID   string
	Role Role
}

type ResolvedIdentity struct {
	Actor    Actor
	Snapshot *TrustedWorkspaceSnapshot
}

type IdentityResolver func(*http.Request) (ResolvedIdentity, error)

type upstreamIdentityResolver struct {
	baseURL *url.URL
	client  *http.Client
}

type upstreamUser struct {
	ID string `json:"id"`
}

type upstreamWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type upstreamMember struct {
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	Role        string `json:"role"`
}

func NewUpstreamIdentityResolver(rawBaseURL string, client *http.Client) (IdentityResolver, error) {
	const op = "new upstream identity resolver"
	baseURL, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" || baseURL.User != nil || (baseURL.Path != "" && baseURL.Path != "/") || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, invalid(op, "base_url", "must be an absolute HTTP(S) origin without credentials, path, query, or fragment")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if client.Timeout <= 0 {
		return nil, invalid(op, "client", "must have a positive timeout")
	}
	clone := *client
	clone.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return (&upstreamIdentityResolver{baseURL: baseURL, client: &clone}).Resolve, nil
}

func (r *upstreamIdentityResolver) Resolve(request *http.Request) (ResolvedIdentity, error) {
	const op = "resolve upstream identity"
	workspaceID := request.PathValue("workspace")
	if err := validateIdentifier(op, "workspace_id", workspaceID); err != nil {
		return ResolvedIdentity{}, err
	}
	authorization, cookie, fromCookie, err := identityCredential(request)
	if err != nil {
		return ResolvedIdentity{}, err
	}
	if fromCookie && !validCSRF(request) {
		return ResolvedIdentity{}, denied(op, "cookie-authenticated mutation requires a valid CSRF token")
	}

	var user upstreamUser
	if err := r.getJSON(request, "/api/me", authorization, cookie, &user); err != nil {
		return ResolvedIdentity{}, err
	}
	var workspace upstreamWorkspace
	if err := r.getJSON(request, "/api/workspaces/"+url.PathEscape(workspaceID), authorization, cookie, &workspace); err != nil {
		return ResolvedIdentity{}, err
	}
	var members []upstreamMember
	if err := r.getJSON(request, "/api/workspaces/"+url.PathEscape(workspaceID)+"/members", authorization, cookie, &members); err != nil {
		return ResolvedIdentity{}, err
	}
	if user.ID == "" || workspace.ID != workspaceID || strings.TrimSpace(workspace.Name) == "" {
		return ResolvedIdentity{}, unavailable(op, "identity upstream returned an invalid workspace or subject")
	}

	snapshot := TrustedWorkspaceSnapshot{ID: workspace.ID, Name: workspace.Name, ActorID: user.ID}
	seen := make(map[string]struct{}, len(members))
	actorFound, ownerFound := false, false
	for _, member := range members {
		if member.WorkspaceID != workspaceID || member.UserID == "" {
			return ResolvedIdentity{}, unavailable(op, "identity upstream returned a cross-workspace or empty member")
		}
		if _, exists := seen[member.UserID]; exists {
			return ResolvedIdentity{}, unavailable(op, "identity upstream returned duplicate members")
		}
		seen[member.UserID] = struct{}{}
		role, mapErr := mapUpstreamRole(member.Role)
		if mapErr != nil {
			return ResolvedIdentity{}, mapErr
		}
		actorFound = actorFound || member.UserID == user.ID
		ownerFound = ownerFound || role == RoleOwner
		snapshot.Members = append(snapshot.Members, TrustedMember{ID: member.UserID, Role: role})
	}
	if !actorFound {
		return ResolvedIdentity{}, denied(op, "authenticated subject is not a workspace member")
	}
	if !ownerFound {
		return ResolvedIdentity{}, unavailable(op, "identity upstream returned no active human owner")
	}
	return ResolvedIdentity{
		Actor:    Actor{ID: user.ID, WorkspaceID: workspaceID, Kind: ActorHuman},
		Snapshot: &snapshot,
	}, nil
}

func (r *upstreamIdentityResolver) getJSON(request *http.Request, path, authorization, cookie string, target any) error {
	endpoint := *r.baseURL
	endpoint.Path = strings.TrimRight(r.baseURL.Path, "/") + path
	upstreamRequest, err := http.NewRequestWithContext(request.Context(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return unavailable("resolve upstream identity", "cannot build identity request")
	}
	if authorization != "" {
		upstreamRequest.Header.Set("Authorization", authorization)
	} else {
		upstreamRequest.Header.Set("Cookie", cookie)
	}
	upstreamRequest.Header.Set("Accept", "application/json")
	response, err := r.client.Do(upstreamRequest)
	if err != nil {
		return unavailable("resolve upstream identity", "identity upstream request failed")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound {
		return denied("resolve upstream identity", "session or workspace membership was rejected")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return unavailable("resolve upstream identity", fmt.Sprintf("identity upstream returned status %d", response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxIdentityBody+1))
	if err != nil || len(body) > maxIdentityBody {
		return unavailable("resolve upstream identity", "identity upstream returned an oversized response")
	}
	if err := json.Unmarshal(body, target); err != nil {
		return unavailable("resolve upstream identity", "identity upstream returned malformed JSON")
	}
	return nil
}

func identityCredential(request *http.Request) (authorization, cookie string, fromCookie bool, err error) {
	if value := request.Header.Get("Authorization"); value != "" {
		if len(value) < 8 || !strings.EqualFold(value[:7], "Bearer ") || strings.TrimSpace(value[7:]) == "" {
			return "", "", false, denied("resolve upstream identity", "invalid bearer credential")
		}
		return value, "", false, nil
	}
	authCookie, cookieErr := request.Cookie(authCookieName)
	if cookieErr != nil || authCookie.Value == "" {
		return "", "", false, denied("resolve upstream identity", "missing session credential")
	}
	return "", authCookieName + "=" + authCookie.Value, true, nil
}

func validCSRF(request *http.Request) bool {
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	authCookie, authErr := request.Cookie(authCookieName)
	csrfCookie, csrfErr := request.Cookie(csrfCookieName)
	csrfHeader := request.Header.Get("X-CSRF-Token")
	if authErr != nil || csrfErr != nil || authCookie.Value == "" || csrfHeader == "" || !hmac.Equal([]byte(csrfCookie.Value), []byte(csrfHeader)) {
		return false
	}
	parts := strings.SplitN(csrfHeader, ".", 2)
	if len(parts) != 2 {
		return false
	}
	nonce, nonceErr := hex.DecodeString(parts[0])
	signature, signatureErr := hex.DecodeString(parts[1])
	if nonceErr != nil || signatureErr != nil || len(nonce) != 16 || len(signature) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, []byte(authCookie.Value))
	_, _ = mac.Write(nonce)
	return hmac.Equal(mac.Sum(nil), signature)
}

func mapUpstreamRole(value string) (Role, error) {
	switch Role(strings.ToLower(strings.TrimSpace(value))) {
	case RoleOwner:
		return RoleOwner, nil
	case RoleAdmin:
		return RoleAdmin, nil
	case RoleMember:
		return RoleMember, nil
	default:
		return "", unavailable("resolve upstream identity", "identity upstream returned an unsupported role")
	}
}
