package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

func followRequest(t *testing.T, repo *InMemoryRepository, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)
	return rr
}

func TestFollowUser_Success(t *testing.T) {
	repo := &InMemoryRepository{}
	token := generateTestToken(jwtSecret, "alice", false)

	rr := followRequest(t, repo, "PUT", "/api/v1/users/bob/follow", token)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.FollowCalledWith.FollowerID != "alice" || repo.FollowCalledWith.FolloweeID != "bob" {
		t.Errorf("unexpected repo args: %+v", repo.FollowCalledWith)
	}
}

func TestFollowUser_Unauthorized(t *testing.T) {
	rr := followRequest(t, &InMemoryRepository{}, "PUT", "/api/v1/users/bob/follow", "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestFollowUser_SelfFollow(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrSelfFollow}
	token := generateTestToken(jwtSecret, "alice", false)

	rr := followRequest(t, repo, "PUT", "/api/v1/users/alice/follow", token)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for a self-follow, got %d", rr.Code)
	}
}

func TestFollowUser_UnknownUser(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrUserNotFound}
	token := generateTestToken(jwtSecret, "alice", false)

	rr := followRequest(t, repo, "PUT", "/api/v1/users/ghost/follow", token)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for an unknown user, got %d", rr.Code)
	}
}

func TestUnfollowUser_Success(t *testing.T) {
	repo := &InMemoryRepository{}
	token := generateTestToken(jwtSecret, "alice", false)

	rr := followRequest(t, repo, "DELETE", "/api/v1/users/bob/follow", token)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.FollowCalledWith.FollowerID != "alice" || repo.FollowCalledWith.FolloweeID != "bob" {
		t.Errorf("unexpected repo args: %+v", repo.FollowCalledWith)
	}
}

func TestUnfollowUser_Unauthorized(t *testing.T) {
	rr := followRequest(t, &InMemoryRepository{}, "DELETE", "/api/v1/users/bob/follow", "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// The profile response carries the follow context the UI needs.
func TestGetUser_IncludesFollowCounts(t *testing.T) {
	repo := &InMemoryRepository{
		User:    &models.User{UserID: "bob"},
		Follows: &models.FollowCounts{FollowersCount: 3, FollowingCount: 5, IsFollowing: true},
	}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("GET", "/api/v1/users/bob", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Follows models.FollowCounts `json:"follows"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Follows.FollowersCount != 3 || !resp.Data.Follows.IsFollowing {
		t.Errorf("unexpected follows block: %+v", resp.Data.Follows)
	}
}

func TestListFollowers(t *testing.T) {
	repo := &InMemoryRepository{FollowerUsers: []models.User{{UserID: "alice"}}}

	req := httptest.NewRequest("GET", "/api/v1/users/bob/followers?limit=10&offset=20", nil)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(repo.ListCalledWith) != 1 {
		t.Fatalf("expected one repo call, got %+v", repo.ListCalledWith)
	}
	call := repo.ListCalledWith[0]
	if call.Kind != "followers" || call.UserID != "bob" || call.Limit != 10 || call.Offset != 20 {
		t.Errorf("unexpected repo args: %+v", call)
	}

	var resp struct {
		Data []models.User `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].UserID != "alice" {
		t.Errorf("unexpected payload: %+v", resp.Data)
	}
}

func TestListFollowing(t *testing.T) {
	repo := &InMemoryRepository{}

	req := httptest.NewRequest("GET", "/api/v1/users/bob/following", nil)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(repo.ListCalledWith) != 1 || repo.ListCalledWith[0].Kind != "following" {
		t.Errorf("unexpected repo calls: %+v", repo.ListCalledWith)
	}
	// Defaults, not zero.
	if repo.ListCalledWith[0].Limit != 20 || repo.ListCalledWith[0].Offset != 0 {
		t.Errorf("expected the default limit of 20, got %+v", repo.ListCalledWith[0])
	}
}

// An empty list must serialize as [] so clients can map over it.
func TestListFollowers_EmptyIsArray(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users/bob/followers", nil)
	rr := httptest.NewRecorder()
	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr, req)

	var resp struct {
		Data []models.User `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected an empty array, got null")
	}
}

func TestListFollowers_InvalidPagination(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/bob/followers?limit=nope", nil)
	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListFollowers_RepoError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/ghost/followers", nil)
	setupTestRouter(&InMemoryRepository{Err: repository.ErrUserNotFound}).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
