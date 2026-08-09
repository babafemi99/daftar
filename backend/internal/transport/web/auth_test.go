package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/babafemi99/daftar/backend/internal/buhari"
	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/model"
	"github.com/babafemi99/daftar/backend/internal/pkg/lid"
)

type fakeAuthUsers struct {
	user        model.User
	registerErr error
	loginErr    error
	getErr      error
}

type fakeSessions struct {
	user *model.User
	used bool
}

func (sessions *fakeSessions) Issue(context.Context, string) (string, time.Time, error) {
	return "refresh-one", time.Now().Add(time.Hour), nil
}
func (sessions *fakeSessions) Rotate(_ context.Context, raw string) (*model.User, string, time.Time, error) {
	if raw != "refresh-one" || sessions.used {
		return nil, "", time.Time{}, buhari.New(buhari.CodeUnauthorized, "The refresh session is invalid or expired.")
	}
	sessions.used = true
	return sessions.user, "refresh-two", time.Now().Add(time.Hour), nil
}
func (*fakeSessions) Revoke(context.Context, string) error { return nil }

func (f *fakeAuthUsers) Register(context.Context, *model.CreateUserRequest) (*model.User, error) {
	return &f.user, f.registerErr
}
func (f *fakeAuthUsers) Login(context.Context, *model.LoginRequest) (*model.User, error) {
	return &f.user, f.loginErr
}
func (f *fakeAuthUsers) GetByID(_ context.Context, id string) (*model.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if id != f.user.ID {
		return nil, buhari.New(buhari.CodeNotFound, "User not found.")
	}
	return &f.user, nil
}

func configuredAuthAPI(t *testing.T, users AuthUsers) *API {
	t.Helper()
	api := NewAPI(cfg.HTTP{CORSAllowedOrigins: []string{"https://app.example.com"}})
	err := api.ConfigureAuth(users, cfg.JWT{
		Secret: "01234567890123456789012345678901", Issuer: "daftar-test",
		Audience: "daftar-web-test", AccessTTL: 10 * time.Hour,
	}, cfg.Cookie{Name: "daftar_session", Path: "/", SameSite: "lax"})
	if err != nil {
		t.Fatalf("ConfigureAuth() error = %v", err)
	}
	return api
}

func authRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://app.example.com")
	return request
}

func TestRegisterReturnsAuthenticatedCookie(t *testing.T) {
	user := model.User{ID: lid.NewUser(), Email: "user@example.com", FirstName: "Ada", LastName: "Lovelace"}
	api := configuredAuthAPI(t, &fakeAuthUsers{user: user})
	recorder := httptest.NewRecorder()
	request := authRequest(http.MethodPost, "/api/v1/auth/register", `{"email":"user@example.com","password":"strong-password","first_name":"Ada","last_name":"Lovelace"}`)

	api.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Name != "daftar_session" || cookies[0].Value == "" {
		t.Fatalf("cookies = %#v", cookies)
	}
}

func TestDuplicateRegistrationAndIncorrectLogin(t *testing.T) {
	api := configuredAuthAPI(t, &fakeAuthUsers{
		registerErr: buhari.New(buhari.CodeEmailAlreadyRegistered, "An account with this email already exists."),
		loginErr:    buhari.New(buhari.CodeUnauthorized, "Invalid email or password."),
	})

	register := httptest.NewRecorder()
	api.Handler().ServeHTTP(register, authRequest(http.MethodPost, "/api/v1/auth/register", `{"email":"user@example.com","password":"strong-password","first_name":"Ada","last_name":"Lovelace"}`))
	if register.Code != http.StatusConflict {
		t.Fatalf("register status = %d", register.Code)
	}

	login := httptest.NewRecorder()
	api.Handler().ServeHTTP(login, authRequest(http.MethodPost, "/api/v1/auth/login", `{"email":"user@example.com","password":"wrong-password"}`))
	if login.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d", login.Code)
	}
}

func TestLoginMeAndLogout(t *testing.T) {
	user := model.User{ID: lid.NewUser(), Email: "user@example.com", FirstName: "Ada", LastName: "Lovelace"}
	api := configuredAuthAPI(t, &fakeAuthUsers{user: user})
	login := httptest.NewRecorder()
	api.Handler().ServeHTTP(login, authRequest(http.MethodPost, "/api/v1/auth/login", `{"email":"user@example.com","password":"strong-password"}`))
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
	}
	cookie := login.Result().Cookies()[0]

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meRequest.AddCookie(cookie)
	me := httptest.NewRecorder()
	api.Handler().ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", me.Code, me.Body.String())
	}

	logoutRequest := authRequest(http.MethodPost, "/api/v1/auth/logout", "")
	logoutRequest.AddCookie(cookie)
	logout := httptest.NewRecorder()
	api.Handler().ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("logout status/cookies = %d %#v", logout.Code, logout.Result().Cookies())
	}
}

func TestMeRejectsUnauthenticatedAndOtherUser(t *testing.T) {
	userB := model.User{ID: lid.NewUser(), Email: "b@example.com"}
	api := configuredAuthAPI(t, &fakeAuthUsers{user: userB})
	unauthenticated := httptest.NewRecorder()
	api.Handler().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticated.Code)
	}

	userA := model.User{ID: lid.NewUser(), Email: "a@example.com"}
	issuer := configuredAuthAPI(t, &fakeAuthUsers{user: userA})
	login := httptest.NewRecorder()
	issuer.Handler().ServeHTTP(login, authRequest(http.MethodPost, "/api/v1/auth/login", `{"email":"a@example.com","password":"strong-password"}`))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.AddCookie(login.Result().Cookies()[0])
	other := httptest.NewRecorder()
	api.Handler().ServeHTTP(other, request)
	if other.Code != http.StatusNotFound {
		t.Fatalf("cross-user status = %d, body = %s", other.Code, other.Body.String())
	}
}

func TestAuthRejectsTrailingJSON(t *testing.T) {
	api := configuredAuthAPI(t, &fakeAuthUsers{})
	response := httptest.NewRecorder()
	request := authRequest(http.MethodPost, "/api/v1/auth/login", `{"email":"user@example.com","password":"password"} {"extra":true}`)
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRefreshRotatesCookieAndRejectsReplay(t *testing.T) {
	user := model.User{ID: lid.NewUser(), Email: "user@example.com", FirstName: "Ada", LastName: "Lovelace"}
	api := configuredAuthAPI(t, &fakeAuthUsers{user: user})
	sessions := &fakeSessions{user: &user}
	if err := api.ConfigureSessions(sessions); err != nil {
		t.Fatal(err)
	}

	request := authRequest(http.MethodPost, "/api/v1/auth/refresh", "")
	request.AddCookie(&http.Cookie{Name: "daftar_refresh", Value: "refresh-one"})
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(response.Result().Cookies()) != 2 {
		t.Fatalf("refresh status=%d cookies=%#v body=%s", response.Code, response.Result().Cookies(), response.Body.String())
	}
	if response.Result().Cookies()[1].Value != "refresh-two" {
		t.Fatalf("rotated cookies=%#v", response.Result().Cookies())
	}

	replayRequest := authRequest(http.MethodPost, "/api/v1/auth/refresh", "")
	replayRequest.AddCookie(&http.Cookie{Name: "daftar_refresh", Value: "refresh-one"})
	replay := httptest.NewRecorder()
	api.Handler().ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body.String())
	}
}
