package routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type familyMemberPayload struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Relation string `json:"relation"`
}

type familyMemberResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Relation string `json:"relation"`
}

func RegisterFamilyRoutes(router *gin.Engine) {
	router.GET("/family/members", listFamilyMembers)
	router.POST("/family/members", createFamilyMember)
	router.PUT("/family/members/:id", updateFamilyMember)
	router.DELETE("/family/members/:id", deleteFamilyMember)
}

func listFamilyMembers(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	members, err := query.ListFamilyMembers(identity.credential.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list family members")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch family members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": mapFamilyMembers(members),
	})
}

func createFamilyMember(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	payload, ok := parseAndValidateFamilyPayload(c)
	if !ok {
		return
	}

	member, err := query.CreateFamilyMember(identity.credential.ID, payload.Name, payload.Email, payload.Relation)
	if err != nil {
		if isUniqueConstraintError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "family member with this email already exists"})
			return
		}
		log.Error().Err(err).Msg("failed to create family member")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create family member"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"member": mapFamilyMember(*member)})
}

func updateFamilyMember(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	memberID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	payload, ok := parseAndValidateFamilyPayload(c)
	if !ok {
		return
	}

	member, err := query.UpdateFamilyMember(identity.credential.ID, memberID, payload.Name, payload.Email, payload.Relation)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "family member not found"})
		case isUniqueConstraintError(err):
			c.JSON(http.StatusConflict, gin.H{"error": "family member with this email already exists"})
		default:
			log.Error().Err(err).Msg("failed to update family member")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update family member"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"member": mapFamilyMember(*member)})
}

func deleteFamilyMember(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	memberID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	if err := query.DeleteFamilyMember(identity.credential.ID, memberID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "family member not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete family member")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete family member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func parseAndValidateFamilyPayload(c *gin.Context) (*familyMemberPayload, bool) {
	var payload familyMemberPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return nil, false
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.Email = strings.ToLower(strings.TrimSpace(payload.Email))
	payload.Relation = strings.TrimSpace(payload.Relation)

	if payload.Name == "" || payload.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and email are required"})
		return nil, false
	}

	if !strings.Contains(payload.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is invalid"})
		return nil, false
	}

	if payload.Relation == "" {
		payload.Relation = "Family"
	}

	return &payload, true
}

func parseUintParam(c *gin.Context, key string) (uint, bool) {
	raw := c.Param(key)
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}

	return uint(value), true
}

func mapFamilyMembers(members []query.FamilyMember) []familyMemberResponse {
	result := make([]familyMemberResponse, 0, len(members))
	for _, member := range members {
		result = append(result, mapFamilyMember(member))
	}

	return result
}

func mapFamilyMember(member query.FamilyMember) familyMemberResponse {
	return familyMemberResponse{
		ID:       member.ID,
		Name:     member.Name,
		Email:    member.Email,
		Relation: member.Relation,
	}
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
