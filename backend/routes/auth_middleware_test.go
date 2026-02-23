package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestAuthRequiredMiddleware_DisabledAllowsProtectedRoute(t *testing.T) {
	router := newTestRouter(false)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthRequiredMiddleware_PublicPathsBypassAuth(t *testing.T) {
	router := newTestRouter(true)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "login path",
			method: http.MethodGet,
			path:   "/auth/google/start",
		},
		{
			name:   "logout path",
			method: http.MethodPost,
			path:   "/auth/logout",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestAuthRequiredMiddleware_ProtectedRouteMissingToken(t *testing.T) {
	router := newTestRouter(true)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if got := errorFromResponse(t, rec); got != "missing session token" {
		t.Fatalf("expected missing session token error, got %q", got)
	}
}

func TestAuthRequiredMiddleware_ProtectedRouteInvalidToken(t *testing.T) {
	initTestDB(t)
	router := newTestRouter(true)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Session-Token", "not-a-real-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if got := errorFromResponse(t, rec); got != "invalid session" {
		t.Fatalf("expected invalid session error, got %q", got)
	}
}

func TestAuthRequiredMiddleware_ProtectedRouteExpiredTokenDeletesSession(t *testing.T) {
	initTestDB(t)
	router := newTestRouter(true)

	session, err := query.CreateOAuthSession(1, "user@example.com", -1*time.Minute)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Session-Token", session.SessionToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if got := errorFromResponse(t, rec); got != "session expired" {
		t.Fatalf("expected session expired error, got %q", got)
	}

	_, err = query.GetOAuthSession(session.SessionToken)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected session to be deleted, got error: %v", err)
	}
}

func TestAuthRequiredMiddleware_ProtectedRouteValidToken(t *testing.T) {
	initTestDB(t)
	router := newTestRouter(true)

	session, err := query.CreateOAuthSession(1, "user@example.com", time.Hour)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-Session-Token", session.SessionToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func newTestRouter(enabled bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthRequiredMiddleware(enabled))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.GET("/auth/google/start", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.POST("/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return router
}

func initTestDB(t *testing.T) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "auth.db")
	if err := query.InitDB(path); err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
}

func errorFromResponse(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	return body["error"]
}
