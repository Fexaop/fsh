package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
)

func TestListFamilyMembers_RequiresSessionToken(t *testing.T) {
	initTestDB(t)
	router := newFamilyTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/family/members", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	if got := errorFromResponse(t, rec); got != "missing session token" {
		t.Fatalf("expected missing session token error, got %q", got)
	}
}

func TestListFamilyMembers_ReturnsMembersForAuthenticatedOwnerOnly(t *testing.T) {
	initTestDB(t)
	router := newFamilyTestRouter()

	ownerA, sessionA := createTestCredentialAndSession(t, "owner-a@example.com", "Owner A")
	ownerB, _ := createTestCredentialAndSession(t, "owner-b@example.com", "Owner B")

	if _, err := query.CreateFamilyMember(ownerA.ID, "A Child", "child-a@example.com", "Child"); err != nil {
		t.Fatalf("failed to create owner A member: %v", err)
	}
	if _, err := query.CreateFamilyMember(ownerB.ID, "B Child", "child-b@example.com", "Child"); err != nil {
		t.Fatalf("failed to create owner B member: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/family/members", nil)
	req.Header.Set("X-Session-Token", sessionA.SessionToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Members []familyMemberResponse `json:"members"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Members) != 1 {
		t.Fatalf("expected 1 family member, got %d", len(response.Members))
	}
	if response.Members[0].Email != "child-a@example.com" {
		t.Fatalf("expected owner A member email, got %q", response.Members[0].Email)
	}
}

func TestFamilyMembers_CRUD(t *testing.T) {
	initTestDB(t)
	router := newFamilyTestRouter()

	_, session := createTestCredentialAndSession(t, "owner-c@example.com", "Owner C")

	createBody := `{"name":"Mom","email":"mom@example.com","relation":"Mother"}`
	createReq := httptest.NewRequest(http.MethodPost, "/family/members", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Session-Token", session.SessionToken)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createRec.Code)
	}

	var createResp struct {
		Member familyMemberResponse `json:"member"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	updateBody := `{"name":"Mom Updated","email":"mom@example.com","relation":"Parent"}`
	updateReq := httptest.NewRequest(
		http.MethodPut,
		"/family/members/"+uintToString(createResp.Member.ID),
		strings.NewReader(updateBody),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("X-Session-Token", session.SessionToken)
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, updateRec.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/family/members/"+uintToString(createResp.Member.ID), nil)
	deleteReq.Header.Set("X-Session-Token", session.SessionToken)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, deleteRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/family/members", nil)
	listReq.Header.Set("X-Session-Token", session.SessionToken)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRec.Code)
	}

	var listResp struct {
		Members []familyMemberResponse `json:"members"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	if len(listResp.Members) != 0 {
		t.Fatalf("expected 0 family members after delete, got %d", len(listResp.Members))
	}
}

func newFamilyTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterFamilyRoutes(router)
	return router
}

func createTestCredentialAndSession(t *testing.T, email, name string) (*query.GoogleCredential, *query.OAuthSession) {
	t.Helper()

	credential, err := query.UpsertGoogleCredential(
		"google-"+email,
		email,
		name,
		"",
		"access-token",
		"refresh-token",
		"id-token",
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to upsert test credential: %v", err)
	}

	session, err := query.CreateOAuthSession(credential.ID, credential.Email, time.Hour)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	return credential, session
}

func uintToString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
