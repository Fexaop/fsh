package routes

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const defaultInvitationTTL = 7 * 24 * time.Hour

type createInvitationPayload struct {
	Relation string `json:"relation"`
}

type respondInvitationPayload struct {
	Action string `json:"action"`
}

type familyInvitationResponse struct {
	ID           uint   `json:"id"`
	InviteID     string `json:"inviteId"`
	InviterName  string `json:"inviterName"`
	InviterEmail string `json:"inviterEmail"`
	InviteeEmail string `json:"inviteeEmail,omitempty"`
	Relation     string `json:"relation"`
	Status       string `json:"status"`
	CanRespond   bool   `json:"canRespond"`
	CreatedAt    int64  `json:"createdAt"`
	ExpiresAt    int64  `json:"expiresAt"`
	RespondedAt  *int64 `json:"respondedAt,omitempty"`
}

func RegisterInvitationRoutes(router *gin.Engine) {
	router.POST("/family/invitations", createFamilyInvitation)
	router.GET("/family/invitations/outgoing", listOutgoingFamilyInvitations)
	router.GET("/family/invitations/:inviteID", getFamilyInvitationByID)
	router.POST("/family/invitations/:inviteID/respond", respondToFamilyInvitation)
}

func createFamilyInvitation(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	var payload createInvitationPayload
	if err := c.ShouldBindJSON(&payload); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	invitation, err := query.CreateFamilyInvitation(identity.credential, payload.Relation, defaultInvitationTTL)
	if err != nil {
		log.Error().Err(err).Msg("failed creating family invitation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"invitation": mapFamilyInvitation(*invitation, identity.credential.ID),
	})
}

func listOutgoingFamilyInvitations(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	invitations, err := query.ListOutgoingFamilyInvitations(identity.credential.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed listing outgoing family invitations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch invitations"})
		return
	}

	response := make([]familyInvitationResponse, 0, len(invitations))
	for _, invitation := range invitations {
		response = append(response, mapFamilyInvitation(invitation, identity.credential.ID))
	}

	c.JSON(http.StatusOK, gin.H{
		"invitations": response,
	})
}

func getFamilyInvitationByID(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	inviteID := strings.ToUpper(strings.TrimSpace(c.Param("inviteID")))
	if inviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "inviteID is required"})
		return
	}

	invitation, err := query.GetFamilyInvitationByInviteID(inviteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
			return
		}
		log.Error().Err(err).Msg("failed fetching invitation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch invitation"})
		return
	}

	if !canViewInvitation(*invitation, identity.credential.ID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invitation": mapFamilyInvitation(*invitation, identity.credential.ID),
	})
}

func respondToFamilyInvitation(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	inviteID := strings.ToUpper(strings.TrimSpace(c.Param("inviteID")))
	if inviteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "inviteID is required"})
		return
	}

	var payload respondInvitationPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	invitation, err := query.RespondToFamilyInvitation(inviteID, identity.credential, payload.Action)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		case errors.Is(err, query.ErrInvalidInvitationAction):
			c.JSON(http.StatusBadRequest, gin.H{"error": "action must be accept or decline"})
		case errors.Is(err, query.ErrInvitationSelfResponse):
			c.JSON(http.StatusForbidden, gin.H{"error": "you cannot respond to your own invitation"})
		case errors.Is(err, query.ErrInvitationExpired):
			c.JSON(http.StatusGone, gin.H{"error": "invitation has expired"})
		case errors.Is(err, query.ErrInvitationNotPending):
			c.JSON(http.StatusConflict, gin.H{"error": "invitation is already resolved"})
		default:
			log.Error().Err(err).Msg("failed responding to invitation")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to respond to invitation"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invitation": mapFamilyInvitation(*invitation, identity.credential.ID),
	})
}

func canViewInvitation(invitation query.FamilyInvitation, viewerCredentialID uint) bool {
	if invitation.InviterCredentialID == viewerCredentialID {
		return true
	}
	if invitation.Status == query.FamilyInvitationStatusPending {
		return true
	}
	if invitation.InviteeCredentialID != nil && *invitation.InviteeCredentialID == viewerCredentialID {
		return true
	}
	return false
}

func mapFamilyInvitation(invitation query.FamilyInvitation, viewerCredentialID uint) familyInvitationResponse {
	var respondedAt *int64
	if invitation.RespondedAt != nil {
		value := invitation.RespondedAt.Unix()
		respondedAt = &value
	}

	canRespond := invitation.Status == query.FamilyInvitationStatusPending &&
		invitation.InviterCredentialID != viewerCredentialID

	return familyInvitationResponse{
		ID:           invitation.ID,
		InviteID:     invitation.InviteID,
		InviterName:  invitation.InviterName,
		InviterEmail: invitation.InviterEmail,
		InviteeEmail: invitation.InviteeEmail,
		Relation:     invitation.Relation,
		Status:       invitation.Status,
		CanRespond:   canRespond,
		CreatedAt:    invitation.CreatedAt.Unix(),
		ExpiresAt:    invitation.ExpiresAt.Unix(),
		RespondedAt:  respondedAt,
	}
}
