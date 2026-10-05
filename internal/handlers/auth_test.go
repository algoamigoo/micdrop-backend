package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

func generateOnboardingToken(secret string, googleID string) string {
	claims := jwt.MapClaims{
		"purpose":   "onboarding",
		"google_id": googleID,
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func signupRequest(t *testing.T, router http.Handler, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/auth/complete-signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestCompleteSignup_Success(t *testing.T) {
	repo := &InMemoryRepository{
		User: &models.User{UserID: "alice", UserName: "alice"},
	}
	router := setupTestRouter(repo)

	rr := signupRequest(t, router, generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"alice"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.CreateUserCalledWith.UserID != "alice" || repo.CreateUserCalledWith.GoogleID != "google-123" {
		t.Errorf("unexpected CreateUser args: %+v", repo.CreateUserCalledWith)
	}

	var resp struct {
		Data struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Token == "" {
		t.Error("expected a session token in the response")
	}
	if resp.Data.User.UserID != "alice" {
		t.Errorf("expected user alice, got %q", resp.Data.User.UserID)
	}
}

func TestCompleteSignup_UserIDTaken(t *testing.T) {
	repo := &InMemoryRepository{
		User: &models.User{UserID: "alice"},
		Err:  repository.ErrUserIDTaken,
	}
	router := setupTestRouter(repo)

	rr := signupRequest(t, router, generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"alice"}`)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestCompleteSignup_TooShort(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"ab"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompleteSignup_BadChars(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"hi there"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompleteSignup_Reserved(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"admin"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompleteSignup_MissingHeader(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}), "", `{"user_id":"alice"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCompleteSignup_RejectsSessionToken(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateTestToken(jwtSecret, "alice", false), `{"user_id":"alice"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGoogleLogin_SetsStateCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr,
		httptest.NewRequest("GET", "/api/v1/auth/google/login", nil))

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", rr.Code)
	}

	cookie := findCookie(t, rr, oauthStateCookie)
	if cookie.Value == "" {
		t.Fatal("expected a state cookie to be set")
	}
	if !cookie.HttpOnly {
		t.Error("state cookie must be httpOnly so scripts cannot read it")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("state cookie must be SameSite=Lax to survive the Google redirect, got %v", cookie.SameSite)
	}

	loc, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if q := loc.Query().Get("state"); q != cookie.Value {
		t.Errorf("state in redirect = %q, want it to equal the cookie value %q", q, cookie.Value)
	}
}

// Two logins must not reuse the same state value.
func TestGoogleLogin_StateIsNotConstant(t *testing.T) {
	state := func() string {
		rr := httptest.NewRecorder()
		setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr,
			httptest.NewRequest("GET", "/api/v1/auth/google/login", nil))
		return findCookie(t, rr, oauthStateCookie).Value
	}
	if a, b := state(), state(); a == b {
		t.Errorf("expected a fresh state per login, got %q twice", a)
	}
}

func findCookie(t *testing.T, rr *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range (&http.Response{Header: rr.Header()}).Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("cookie %s not set", name)
	return nil
}

func TestGoogleCallback_RejectsStateMismatch(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback?code=fake&state=attacker-state", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "victim-state"})
	rr := httptest.NewRecorder()

	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for mismatched state, got %d", rr.Code)
	}
}

func TestGoogleCallback_RejectsMissingStateCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback?code=fake&state=some-state", nil)
	rr := httptest.NewRecorder()

	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 without a state cookie, got %d", rr.Code)
	}
}

func TestGoogleCallback_RejectsMissingStateParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback?code=fake", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "some-state"})
	rr := httptest.NewRecorder()

	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 without a state param, got %d", rr.Code)
	}
}

// A matching state must clear the cookie so it cannot be replayed.
func TestCheckOAuthState_MatchingStateClearsCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/auth/google/callback?code=fake&state=good-state", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "good-state"})
	rr := httptest.NewRecorder()

	if err := checkOAuthState(rr, req); err != nil {
		t.Fatalf("expected the state check to pass, got %v", err)
	}

	cleared := false
	for _, c := range (&http.Response{Header: rr.Header()}).Cookies() {
		if c.Name == oauthStateCookie && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected the state cookie to be cleared")
	}
}

func TestCheckOAuthState_Rejects(t *testing.T) {
	cases := map[string]struct {
		cookie *http.Cookie
		query  string
	}{
		"mismatched state": {cookie: &http.Cookie{Name: oauthStateCookie, Value: "victim-state"}, query: "attacker-state"},
		"missing cookie":   {query: "some-state"},
		"missing param":    {cookie: &http.Cookie{Name: oauthStateCookie, Value: "some-state"}},
		"empty cookie":     {cookie: &http.Cookie{Name: oauthStateCookie, Value: ""}, query: ""},
	}
	for name, tc := range cases {
		req := httptest.NewRequest("GET", "/api/v1/auth/google/callback?code=fake&state="+tc.query, nil)
		if tc.cookie != nil {
			req.AddCookie(tc.cookie)
		}
		if err := checkOAuthState(httptest.NewRecorder(), req); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}
