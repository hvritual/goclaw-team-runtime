package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/auth/internal/application"
)

const (
	authCookieName = "multica_auth"
	csrfCookieName = "multica_csrf"
)

type LocalAuthHandler struct{ service *application.LocalAuthUseCase }

func NewLocalAuthHandler(service *application.LocalAuthUseCase) *LocalAuthHandler {
	return &LocalAuthHandler{service: service}
}

func (h *LocalAuthHandler) ResolveUserID(request *http.Request) (string, error) {
	token, _ := requestToken(request)
	if token == "" {
		return "", application.ErrInvalidToken
	}
	user, err := h.service.Resolve(request.Context(), token)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (h *LocalAuthHandler) Register(server *kratoshttp.Server) {
	router := server.Route("/")
	router.POST("/auth/send-code", h.sendCode)
	router.POST("/auth/verify-code", h.verifyCode)
	router.POST("/auth/logout", h.logout)
	router.GET("/api/me", h.me)
}

func (h *LocalAuthHandler) sendCode(ctx kratoshttp.Context) error {
	var body struct {
		Email string `json:"email"`
	}
	if !decodeJSON(ctx, &body) {
		return nil
	}
	if err := h.service.SendCode(body.Email); err != nil {
		return writeError(ctx, http.StatusBadRequest, err.Error())
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func (h *LocalAuthHandler) verifyCode(ctx kratoshttp.Context) error {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !decodeJSON(ctx, &body) {
		return nil
	}
	login, err := h.service.VerifyCode(ctx.Request().Context(), body.Email, body.Code)
	if errors.Is(err, application.ErrInvalidCode) {
		return writeError(ctx, http.StatusUnauthorized, err.Error())
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to create local user")
	}
	csrf, err := newCSRFToken(login.Token)
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to create session")
	}
	h.setCookies(ctx, login.Token, csrf, h.service.Now().Add(h.service.SessionTTL()))
	return ctx.JSON(http.StatusOK, login)
}

func (h *LocalAuthHandler) me(ctx kratoshttp.Context) error {
	token, _ := requestToken(ctx.Request())
	if token == "" {
		return writeError(ctx, http.StatusUnauthorized, "authentication required")
	}
	user, err := h.service.Resolve(ctx.Request().Context(), token)
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "invalid token")
	}
	return ctx.JSON(http.StatusOK, user)
}

func (h *LocalAuthHandler) logout(ctx kratoshttp.Context) error {
	token, cookieAuth := requestToken(ctx.Request())
	if cookieAuth && !validCSRF(ctx.Request(), token) {
		return writeError(ctx, http.StatusForbidden, "invalid CSRF token")
	}
	if err := h.service.Revoke(ctx.Request().Context(), token); err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to revoke token")
	}
	h.clearCookies(ctx)
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

func decodeJSON(ctx kratoshttp.Context, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(ctx.Request().Body, 2<<20))
	if err := decoder.Decode(target); err != nil {
		_ = writeError(ctx, http.StatusBadRequest, "invalid request body")
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		_ = writeError(ctx, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func requestToken(request *http.Request) (string, bool) {
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if strings.HasPrefix(authorization, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")), false
	}
	if cookie, err := request.Cookie(authCookieName); err == nil {
		return strings.TrimSpace(cookie.Value), true
	}
	return "", false
}

func newCSRFToken(token string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write(nonce)
	return hex.EncodeToString(nonce) + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func validCSRF(request *http.Request, token string) bool {
	parts := strings.SplitN(request.Header.Get("X-CSRF-Token"), ".", 2)
	if len(parts) != 2 || token == "" {
		return false
	}
	nonce, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	signature, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write(nonce)
	return hmac.Equal(mac.Sum(nil), signature)
}

func (h *LocalAuthHandler) setCookies(ctx kratoshttp.Context, token, csrf string, expires time.Time) {
	maxAge := int(h.service.SessionTTL().Seconds())
	http.SetCookie(ctx.Response(), &http.Cookie{Name: authCookieName, Value: token, Path: "/", MaxAge: maxAge, Expires: expires, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(ctx.Response(), &http.Cookie{Name: csrfCookieName, Value: csrf, Path: "/", MaxAge: maxAge, Expires: expires, SameSite: http.SameSiteStrictMode})
}

func (h *LocalAuthHandler) clearCookies(ctx kratoshttp.Context) {
	expires := time.Unix(0, 0)
	http.SetCookie(ctx.Response(), &http.Cookie{Name: authCookieName, Path: "/", MaxAge: -1, Expires: expires, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.SetCookie(ctx.Response(), &http.Cookie{Name: csrfCookieName, Path: "/", MaxAge: -1, Expires: expires, SameSite: http.SameSiteStrictMode})
}

func writeError(ctx kratoshttp.Context, status int, message string) error {
	return ctx.JSON(status, map[string]string{"error": message})
}
