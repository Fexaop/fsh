package ws

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 2048
)

type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan []byte
	memberID       string
	allowedMembers map[string]struct{}
}

func NewClient(hub *Hub, conn *websocket.Conn, memberID string, allowedMembers map[string]struct{}) *Client {
	normalizedAllowed := make(map[string]struct{}, len(allowedMembers)+1)
	for allowedID := range allowedMembers {
		normalizedID := normalizeMemberID(allowedID)
		if normalizedID == "" {
			continue
		}
		normalizedAllowed[normalizedID] = struct{}{}
	}

	normalizedMemberID := normalizeMemberID(memberID)
	if normalizedMemberID != "" {
		normalizedAllowed[normalizedMemberID] = struct{}{}
	}

	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, 256),
		memberID:       normalizedMemberID,
		allowedMembers: normalizedAllowed,
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
			if isExpectedReadClose(err) {
				log.Info().Err(err).Str("member_id", c.memberID).Msg("WebSocket client disconnected")
			} else {
				log.Error().Err(err).Str("member_id", c.memberID).Msg("WebSocket read error")
			}
			break
		}

		var payload inboundLocationMessage
		if err := json.Unmarshal(message, &payload); err != nil {
			log.Warn().Err(err).Str("member_id", c.memberID).Msg("Ignoring malformed WebSocket payload")
			continue
		}

		if payload.Type != messageTypeLocationUpdate {
			continue
		}

		if !isValidCoordinates(payload.Latitude, payload.Longitude) {
			log.Warn().Str("member_id", c.memberID).Msg("Ignoring invalid location coordinates")
			continue
		}

		c.hub.SubmitLocation(MemberLocation{
			MemberID:  c.memberID,
			Latitude:  payload.Latitude,
			Longitude: payload.Longitude,
			UpdatedAt: time.Now().Unix(),
		})
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

			w, err := c.conn.NextWriter(websocket.TextMessage)
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

func (c *Client) CanView(memberID string) bool {
	_, ok := c.allowedMembers[normalizeMemberID(memberID)]
	return ok
}

func isValidCoordinates(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func isExpectedReadClose(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) {
		return true
	}

	if websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
	) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "unexpected eof")
}
