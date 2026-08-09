package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/tokoz"
)

const maxJSONBody = 1 << 20

var ErrAuthServiceRequired = errors.New("web: auth service is required")
var ErrSessionServiceRequired = errors.New("web: session service is required")

type Sessions interface {
	Issue(context.Context, string) (string, time.Time, error)
	Rotate(context.Context, string) (*model.User, string, time.Time, error)
	Revoke(context.Context, string) error
}

func (a *API) ConfigureSessions(sessions Sessions) error {
	if sessions == nil {
		return ErrSessionServiceRequired
	}
	a.sessions = sessions
	return nil
}

type AuthUsers interface {
	Register(ctx context.Context, request *model.CreateUserRequest) (*model.User, error)
	Login(ctx context.Context, request *model.LoginRequest) (*model.User, error)
	GetByID(ctx context.Context, requestedUserID string) (*model.User, error)
}

func (a *API) ConfigureAuth(users AuthUsers, jwtConfig cfg.JWT, cookie cfg.Cookie) error {
	if users == nil {
		return ErrAuthServiceRequired
	}
	if err := tokoz.Configure(tokoz.Config{
		Secret: jwtConfig.Secret, Issuer: jwtConfig.Issuer,
		Audience: jwtConfig.Audience, AccessTTL: jwtConfig.AccessTTL,
	}); err != nil {
		return err
	}
	if cookie.Name == "" {
		cookie.Name = "daftar_session"
	}
	if cookie.RefreshName == "" {
		cookie.RefreshName = "daftar_refresh"
	}
	if cookie.Path == "" {
		cookie.Path = "/"
	}
	if cookie.SameSite == "" {
		cookie.SameSite = "lax"
	}
	a.users, a.jwt, a.cookie = users, jwtConfig, cookie
	return nil
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var request model.CreateUserRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	if a.users == nil {
		ResponseError(r.Context(), w, errors.New("auth service not configured"))
		return
	}
	user, err := a.users.Register(r.Context(), &request)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	if err := a.setAuthCookies(r.Context(), w, user); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	Response(r.Context(), w, http.StatusCreated, user)
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var request model.LoginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	if a.users == nil {
		ResponseError(r.Context(), w, errors.New("auth service not configured"))
		return
	}
	user, err := a.users.Login(r.Context(), &request)
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	if err := a.setAuthCookies(r.Context(), w, user); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	Response(r.Context(), w, http.StatusOK, user)
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(a.cookie.RefreshName); err == nil && a.sessions != nil {
		if err := a.sessions.Revoke(r.Context(), cookie.Value); err != nil {
			ResponseError(r.Context(), w, err)
			return
		}
	}
	a.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	if a.sessions == nil {
		ResponseError(r.Context(), w, errors.New("session service not configured"))
		return
	}
	cookie, err := r.Cookie(a.cookie.RefreshName)
	if err != nil || cookie.Value == "" {
		a.clearAuthCookies(w)
		ResponseError(r.Context(), w, buhari.New(buhari.CodeUnauthorized, "A valid refresh session is required."))
		return
	}
	user, raw, expiresAt, err := a.sessions.Rotate(r.Context(), cookie.Value)
	if err != nil {
		a.clearAuthCookies(w)
		ResponseError(r.Context(), w, err)
		return
	}
	if err := a.setAccessCookie(w, user); err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	a.setRefreshCookie(w, raw, expiresAt)
	Response(r.Context(), w, http.StatusOK, user)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	executor, ok := ExecutorFromContext(r.Context())
	if !ok {
		ResponseError(r.Context(), w, buhari.New(buhari.CodeUnauthorized, "Authenticated user context is required."))
		return
	}
	user, err := a.users.GetByID(r.Context(), string(executor.ID))
	if err != nil {
		ResponseError(r.Context(), w, err)
		return
	}
	Response(r.Context(), w, http.StatusOK, user)
}

func (a *API) setAuthCookies(ctx context.Context, w http.ResponseWriter, user *model.User) error {
	if a.sessions == nil {
		return a.setAccessCookie(w, user)
	}
	raw, expiresAt, err := a.sessions.Issue(ctx, user.ID)
	if err != nil {
		return err
	}
	if err := a.setAccessCookie(w, user); err != nil {
		return err
	}
	a.setRefreshCookie(w, raw, expiresAt)
	return nil
}

func (a *API) setAccessCookie(w http.ResponseWriter, user *model.User) error {
	token, err := tokoz.GenerateToken(user.ID, "user", user.Email)
	if err != nil {
		return buhari.Wrap(buhari.CodeInternalError, "Unable to create the authenticated session.", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name: a.cookie.Name, Value: token, Path: a.cookie.Path, Domain: a.cookie.Domain,
		MaxAge: int(a.jwt.AccessTTL.Seconds()), Expires: time.Now().UTC().Add(a.jwt.AccessTTL),
		HttpOnly: true, Secure: a.cookie.Secure, SameSite: sameSite(a.cookie.SameSite),
	})
	return nil
}

func (a *API) setRefreshCookie(w http.ResponseWriter, raw string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{Name: a.cookie.RefreshName, Value: raw, Path: "/api/v1/auth", Domain: a.cookie.Domain, MaxAge: int(time.Until(expiresAt).Seconds()), Expires: expiresAt, HttpOnly: true, Secure: a.cookie.Secure, SameSite: sameSite(a.cookie.SameSite)})
}

func (a *API) clearAuthCookies(w http.ResponseWriter) {
	cookies := []http.Cookie{{Name: a.cookie.Name, Path: a.cookie.Path}}
	if a.sessions != nil {
		cookies = append(cookies, http.Cookie{Name: a.cookie.RefreshName, Path: "/api/v1/auth"})
	}
	for _, cookie := range cookies {
		cookie.Value, cookie.Domain, cookie.MaxAge, cookie.Expires = "", a.cookie.Domain, -1, time.Unix(1, 0)
		cookie.HttpOnly, cookie.Secure, cookie.SameSite = true, a.cookie.Secure, sameSite(a.cookie.SameSite)
		http.SetCookie(w, &cookie)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return buhari.New(buhari.CodeValidationFailed, "The request body is invalid.")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return buhari.New(buhari.CodeValidationFailed, "The request body must contain exactly one JSON value.")
	}
	return nil
}

func sameSite(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
