package ws

import "github.com/rs/zerolog/log"

type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	clients map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		clients:    make(map[*Client]struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
			log.Info().Msg("WebSocket client registered")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Info().Msg("WebSocket client unregistered")
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}
