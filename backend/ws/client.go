package ws

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("WebSocket read error")
			break
		}
		log.Info().Msgf("Received message: %s", message)
		c.hub.Broadcast(message)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			msgType := websocket.TextMessage
			if len(message) > 4 && message[0] == 0xFF && message[1] == 0xD8 {
				msgType = websocket.BinaryMessage
			} else if len(message) >= 4 {
				if len(message) > 6 && message[4] == 0xFF && message[5] == 0xD8 {
					msgType = websocket.BinaryMessage
				}
			}
			w, err := c.conn.NextWriter(msgType)
			if err != nil {
				log.Error().Err(err).Msg("WebSocket next writer error")
				return
			}
			if _, err := w.Write(message); err != nil {
				log.Error().Err(err).Msg("WebSocket write error")
			}
			if err := w.Close(); err != nil {
				log.Error().Err(err).Msg("WebSocket writer close error")
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error().Err(err).Msg("WebSocket ping error")
				return
			}
		}
	}
}
