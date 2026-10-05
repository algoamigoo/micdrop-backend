package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

func TestUpdateResponse_Success(t *testing.T) {
	repo := &InMemoryRepository{Response: &models.Response{ResponseID: 3, Body: "edited"}}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("PATCH", "/api/v1/responses/3", strings.NewReader(`{"body":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.UpdateResponseCalledWith.ResponseID != 3 || repo.UpdateResponseCalledWith.UserID != "alice" {
		t.Errorf("unexpected repo args: %+v", repo.UpdateResponseCalledWith)
	}
}

func TestUpdateResponse_Unauthorized(t *testing.T) {
	rr := postRequest(t, "PATCH", "/api/v1/responses/3", "", `{"body":"edited"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestUpdateResponse_NotAuthor(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrNotAuthor}
	token := generateTestToken(jwtSecret, "mallory", false)

	req := httptest.NewRequest("PATCH", "/api/v1/responses/3", strings.NewReader(`{"body":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestDeleteResponse_Success(t *testing.T) {
	repo := &InMemoryRepository{}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("DELETE", "/api/v1/responses/3", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.DeleteResponseCalledWith.ResponseID != 3 {
		t.Errorf("expected response_id 3, got %d", repo.DeleteResponseCalledWith.ResponseID)
	}
}

func TestDeleteResponse_Unauthorized(t *testing.T) {
	rr := postRequest(t, "DELETE", "/api/v1/responses/3", "", "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}
