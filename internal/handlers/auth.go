package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/algoamigoo/micdrop/internal/config"
	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	Repo        Repository
	Config      *config.Config
	OAuthConfig *oauth2.Config
}

func NewAuthHandler(repo Repository, cfg *config.Config) *AuthHandler { // Changed parameter type
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
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.OAuthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the callback from Google, upserts the user, and issues a JWT.
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

	// Use Google's ID as our internal user_id, prefixed with 'google_'
	userID := "google_" + userInfo.ID
	userName := userInfo.Name
	if userName == "" {
		userName = userInfo.Email
	}

	user, err := h.Repo.GetOrCreateUser(r.Context(), userID, userName)
	if err != nil {
		handleAppError(w, err)
		return
	}

	// Generate JWT
	claims := jwt.MapClaims{
		"user_id": user.UserID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := jwtToken.SignedString([]byte(h.Config.JWTSecret))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sign token")
		return
	}

	// Redirect to frontend with token (Frontend will store this and use it in Auth headers)
	redirectURL := h.Config.FrontendURL + "?token=" + signedToken
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// GetMe returns the profile of the currently authenticated user.
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
