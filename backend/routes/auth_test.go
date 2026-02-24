package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
)

func TestUpdateCurrentUserProfile_RequiresSessionToken(t *testing.T) {
	initTestDB(t)
	router := newAuthRoutesTestRouter()

	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(`{"name":"Updated User"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if got := errorFromResponse(t, rec); got != "missing session token" {
		t.Fatalf("expected missing session token error, got %q", got)
	}
}

func TestUpdateCurrentUserProfile_UpdatesNameAndPicture(t *testing.T) {
	initTestDB(t)
	router := newAuthRoutesTestRouter()

	credential, session := createTestCredentialAndSession(t, "update-profile@example.com", "Original Name")
	body := `{"name":"Updated Name","picture":"https://example.com/avatar.png"}`

	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Token", session.SessionToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response authSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "Updated Name" {
		t.Fatalf("expected updated name in response, got %q", response.Name)
	}
	if response.Picture != "https://example.com/avatar.png" {
		t.Fatalf("expected updated picture in response, got %q", response.Picture)
	}

	updatedCredential, err := query.GetGoogleCredentialByID(credential.ID)
	if err != nil {
		t.Fatalf("failed to reload credential: %v", err)
	}
	if updatedCredential.Name != "Updated Name" {
		t.Fatalf("expected updated name in DB, got %q", updatedCredential.Name)
	}
	if updatedCredential.Picture != "https://example.com/avatar.png" {
		t.Fatalf("expected updated picture in DB, got %q", updatedCredential.Picture)
	}
}

func TestUpdateCurrentUserProfile_RejectsEmptyName(t *testing.T) {
	initTestDB(t)
	router := newAuthRoutesTestRouter()

	_, session := createTestCredentialAndSession(t, "empty-name@example.com", "Original Name")
	body := `{"name":"   "}`

	req := httptest.NewRequest(http.MethodPut, "/auth/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Token", session.SessionToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if got := errorFromResponse(t, rec); got != "name cannot be empty" {
		t.Fatalf("expected empty name validation error, got %q", got)
	}
}

func newAuthRoutesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterAuthRoutes(router)
	return router
}
