package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/algoamigoo/micdrop/internal/config"
	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	Repo        Repository
	Config      *config.Config
	OAuthConfig *oauth2.Config
}

var (
	validUserID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	reservedIDs = map[string]struct{}{
		"me": {}, "admin": {}, "api": {}, "auth": {}, "root": {},
		"micdrop": {}, "support": {},
	}
)

func NewAuthHandler(repo Repository, cfg *config.Config) *AuthHandler {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	return &AuthHandler{Repo: repo, Config: cfg, OAuthConfig: oauthCfg}
}

// GoogleLogin redirects the user to Google's consent page.
// GET /api/v1/auth/google/login
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.OAuthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback exchanges the code, then either:
//   - returns a session JWT (existing user), or
//   - returns a short-lived onboarding JWT (new user must pick a user_id).
//
// GET /api/v1/auth/google/callback
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Code not found")
		return
	}

	token, err := h.OAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Failed to exchange token: "+err.Error())
		return
	}

	client := h.OAuthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Failed to get user info")
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to parse user info")
		return
	}

	googleID := userInfo.ID

	// Existing user? Issue a session JWT and send them to the app.
	user, err := h.Repo.GetUserByGoogleID(r.Context(), googleID)
	if err == nil {
		sessionToken, signErr := h.signSessionToken(user.UserID)
		if signErr != nil {
			respondError(w, http.StatusInternalServerError, "Failed to sign token")
			return
		}
		http.Redirect(w, r, h.Config.FrontendURL+"?token="+sessionToken, http.StatusTemporaryRedirect)
		return
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		handleAppError(w, err)
		return
	}

	// New user — issue an onboarding JWT (short-lived, no user_id yet).
	onboardingToken, signErr := h.signOnboardingToken(googleID)
	if signErr != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sign token")
		return
	}
	http.Redirect(w, r, h.Config.FrontendURL+"?onboarding="+onboardingToken, http.StatusTemporaryRedirect)
}

type completeSignupRequest struct {
	UserID string `json:"user_id"`
}

// CompleteSignup creates the user row with the chosen user_id.
// Authorization: Bearer <onboarding JWT>
// POST /api/v1/auth/complete-signup
func (h *AuthHandler) CompleteSignup(w http.ResponseWriter, r *http.Request) {
	googleID, err := h.parseOnboardingToken(r.Header.Get("Authorization"))
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req completeSignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)

	if len(req.UserID) < 3 || len(req.UserID) > 30 {
		respondError(w, http.StatusBadRequest, "user_id must be 3-30 characters")
		return
	}
	if !validUserID.MatchString(req.UserID) {
		respondError(w, http.StatusBadRequest, "user_id may only contain letters, numbers, _ and -")
		return
	}
	if _, reserved := reservedIDs[strings.ToLower(req.UserID)]; reserved {
		respondError(w, http.StatusBadRequest, "user_id is reserved")
		return
	}

	user, err := h.Repo.CreateUser(r.Context(), req.UserID, googleID)
	if err != nil {
		if errors.Is(err, repository.ErrUserIDTaken) {
			respondError(w, http.StatusConflict, "that username is already taken")
			return
		}
		handleAppError(w, err)
		return
	}

	sessionToken, err := h.signSessionToken(user.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sign token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"token": sessionToken,
		"user":  user,
	})
}

// GetMe returns the profile of the currently authenticated user.
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.Repo.GetUserByID(r.Context(), userID)
	if err != nil {
		handleAppError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, user)
}

// --- token helpers ---

func (h *AuthHandler) signSessionToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"purpose": "session",
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(h.Config.JWTSecret))
}

func (h *AuthHandler) signOnboardingToken(googleID string) (string, error) {
	claims := jwt.MapClaims{
		"google_id": googleID,
		"purpose":   "onboarding",
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
		"iat":       time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(h.Config.JWTSecret))
}

func (h *AuthHandler) parseOnboardingToken(authHeader string) (string, error) {
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("missing token")
	}
	raw := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(h.Config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired onboarding token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["purpose"] != "onboarding" {
		return "", errors.New("not an onboarding token")
	}
	googleID, _ := claims["google_id"].(string)
	if googleID == "" {
		return "", errors.New("token missing google_id")
	}
	return googleID, nil
}
