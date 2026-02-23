package routes

import (
	"net/http"

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
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade error")
			return
		}

		client := ws.NewClient(hub, conn)
		hub.Register(client)

		// start client pumps
		go client.WritePump()
		go client.ReadPump()
	})
}

