package routes

import (
	"net/http"
	"strings"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/Fexaop/fsh/backend/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func RegisterWSRoutes(router *gin.Engine, hub *ws.Hub) {
	router.GET("/ws", func(c *gin.Context) {
		identity, status, errMessage := resolveSessionIdentity(c.Query("sessionToken"))
		if identity == nil {
			c.JSON(status, gin.H{"error": errMessage})
			return
		}

		allowedMembers, err := buildAllowedMemberSet(identity.credential.ID, identity.credential.Email)
		if err != nil {
			log.Error().Err(err).Msg("failed to load allowed family members")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start websocket connection"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade error")
			return
		}

		client := ws.NewClient(hub, conn, identity.credential.Email, allowedMembers)
		hub.Register(client)

		// start client pumps
		go client.WritePump()
		go client.ReadPump()
	})
}

func buildAllowedMemberSet(ownerCredentialID uint, ownerEmail string) (map[string]struct{}, error) {
	allowedMembers := map[string]struct{}{}

	ownerEmail = strings.ToLower(strings.TrimSpace(ownerEmail))
	if ownerEmail != "" {
		allowedMembers[ownerEmail] = struct{}{}
	}

	familyMembers, err := query.ListFamilyMembers(ownerCredentialID)
	if err != nil {
		return nil, err
	}

	for _, member := range familyMembers {
		email := strings.ToLower(strings.TrimSpace(member.Email))
		if email == "" {
			continue
		}
		allowedMembers[email] = struct{}{}
	}

	return allowedMembers, nil
}
