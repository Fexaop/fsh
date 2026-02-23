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

type invitationTestResponse struct {
	Invitation familyInvitationResponse `json:"invitation"`
}

type invitationListResponse struct {
	Invitations []familyInvitationResponse `json:"invitations"`
}

func TestInvitationAcceptFlowCreatesMutualFamilyMembers(t *testing.T) {
	initTestDB(t)
	router := newInvitationTestRouter()

	inviter, inviterSession := createTestCredentialAndSession(t, "inviter@example.com", "Inviter")
	invitee, inviteeSession := createTestCredentialAndSession(t, "invitee@example.com", "Invitee")

	createReq := httptest.NewRequest(http.MethodPost, "/family/invitations", strings.NewReader(`{"relation":"Sibling"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Session-Token", inviterSession.SessionToken)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createRec.Code)
	}

	var created invitationTestResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.Invitation.InviteID == "" {
		t.Fatalf("expected invite ID to be generated")
	}

	respondReq := httptest.NewRequest(
		http.MethodPost,
		"/family/invitations/"+created.Invitation.InviteID+"/respond",
		strings.NewReader(`{"action":"accept"}`),
	)
	respondReq.Header.Set("Content-Type", "application/json")
	respondReq.Header.Set("X-Session-Token", inviteeSession.SessionToken)
	respondRec := httptest.NewRecorder()
	router.ServeHTTP(respondRec, respondReq)

	if respondRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, respondRec.Code)
	}

	var responded invitationTestResponse
	if err := json.Unmarshal(respondRec.Body.Bytes(), &responded); err != nil {
		t.Fatalf("failed to decode respond response: %v", err)
	}
	if responded.Invitation.Status != query.FamilyInvitationStatusAccepted {
		t.Fatalf("expected accepted invitation status, got %q", responded.Invitation.Status)
	}

	inviterMembers, err := query.ListFamilyMembers(inviter.ID)
	if err != nil {
		t.Fatalf("failed listing inviter members: %v", err)
	}
	if !containsMemberEmail(inviterMembers, invitee.Email) {
		t.Fatalf("expected inviter family members to contain invitee email %q", invitee.Email)
	}

	inviteeMembers, err := query.ListFamilyMembers(invitee.ID)
	if err != nil {
		t.Fatalf("failed listing invitee members: %v", err)
	}
	if !containsMemberEmail(inviteeMembers, inviter.Email) {
		t.Fatalf("expected invitee family members to contain inviter email %q", inviter.Email)
	}
}

func TestInvitationDeclineFlowDoesNotCreateFamilyMembers(t *testing.T) {
	initTestDB(t)
	router := newInvitationTestRouter()

	inviter, inviterSession := createTestCredentialAndSession(t, "inviter-two@example.com", "Inviter Two")
	invitee, inviteeSession := createTestCredentialAndSession(t, "invitee-two@example.com", "Invitee Two")

	createReq := httptest.NewRequest(http.MethodPost, "/family/invitations", strings.NewReader(`{}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Session-Token", inviterSession.SessionToken)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createRec.Code)
	}

	var created invitationTestResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	respondReq := httptest.NewRequest(
		http.MethodPost,
		"/family/invitations/"+created.Invitation.InviteID+"/respond",
		strings.NewReader(`{"action":"decline"}`),
	)
	respondReq.Header.Set("Content-Type", "application/json")
	respondReq.Header.Set("X-Session-Token", inviteeSession.SessionToken)
	respondRec := httptest.NewRecorder()
	router.ServeHTTP(respondRec, respondReq)

	if respondRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, respondRec.Code)
	}

	var responded invitationTestResponse
	if err := json.Unmarshal(respondRec.Body.Bytes(), &responded); err != nil {
		t.Fatalf("failed to decode respond response: %v", err)
	}
	if responded.Invitation.Status != query.FamilyInvitationStatusDeclined {
		t.Fatalf("expected declined invitation status, got %q", responded.Invitation.Status)
	}

	inviterMembers, err := query.ListFamilyMembers(inviter.ID)
	if err != nil {
		t.Fatalf("failed listing inviter members: %v", err)
	}
	if len(inviterMembers) != 0 {
		t.Fatalf("expected no inviter family members, got %d", len(inviterMembers))
	}

	inviteeMembers, err := query.ListFamilyMembers(invitee.ID)
	if err != nil {
		t.Fatalf("failed listing invitee members: %v", err)
	}
	if len(inviteeMembers) != 0 {
		t.Fatalf("expected no invitee family members, got %d", len(inviteeMembers))
	}
}

func TestInvitationOutgoingListIsScopedToInviter(t *testing.T) {
	initTestDB(t)
	router := newInvitationTestRouter()

	_, inviterSession := createTestCredentialAndSession(t, "scope-a@example.com", "Scope A")
	_, otherSession := createTestCredentialAndSession(t, "scope-b@example.com", "Scope B")

	createReq := httptest.NewRequest(http.MethodPost, "/family/invitations", strings.NewReader(`{}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Session-Token", inviterSession.SessionToken)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/family/invitations/outgoing", nil)
	listReq.Header.Set("X-Session-Token", otherSession.SessionToken)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRec.Code)
	}

	var list invitationListResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to decode outgoing list response: %v", err)
	}
	if len(list.Invitations) != 0 {
		t.Fatalf("expected 0 outgoing invitations for other user, got %d", len(list.Invitations))
	}
}

func newInvitationTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterFamilyRoutes(router)
	RegisterInvitationRoutes(router)
	return router
}

func containsMemberEmail(members []query.FamilyMember, email string) bool {
	for _, member := range members {
		if strings.EqualFold(member.Email, email) {
			return true
		}
	}
	return false
}
