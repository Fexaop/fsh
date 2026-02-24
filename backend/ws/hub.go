package ws

import (
	"encoding/json"
	"sort"

	"github.com/rs/zerolog/log"
)

type Hub struct {
	register        chan *Client
	unregister      chan *Client
	locationUpdates chan MemberLocation
	sosAlerts       chan SOSAlert

	clients         map[*Client]struct{}
	latestLocations map[string]MemberLocation
}

func NewHub() *Hub {
	return &Hub{
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		locationUpdates: make(chan MemberLocation),
		sosAlerts:       make(chan SOSAlert),
		clients:         make(map[*Client]struct{}),
		latestLocations: make(map[string]MemberLocation),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
			log.Info().Str("member_id", client.memberID).Msg("WebSocket client registered")
			h.sendSnapshot(client)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Info().Msg("WebSocket client unregistered")
			}
		case location := <-h.locationUpdates:
			h.latestLocations[location.MemberID] = location
			h.broadcastLocationUpdate(location)
		case alert := <-h.sosAlerts:
			h.broadcastSOS(alert)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) SubmitLocation(location MemberLocation) {
	h.locationUpdates <- location
}

func (h *Hub) SubmitSOS(alert SOSAlert) {
	h.sosAlerts <- alert
}

func (h *Hub) sendSnapshot(client *Client) {
	locations := make([]MemberLocation, 0, len(h.latestLocations))
	for _, location := range h.latestLocations {
		if client.CanView(location.MemberID) {
			locations = append(locations, location)
		}
	}

	if len(locations) == 0 {
		return
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].MemberID < locations[j].MemberID
	})

	payload, err := json.Marshal(outboundLocationSnapshot{
		Type:      messageTypeLocationSnapshot,
		Locations: locations,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal location snapshot")
		return
	}

	h.sendToClient(client, payload)
}

func (h *Hub) broadcastLocationUpdate(location MemberLocation) {
	payload, err := json.Marshal(outboundLocationUpdate{
		Type:     messageTypeLocationUpdate,
		Location: location,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal location update")
		return
	}

	for client := range h.clients {
		if !client.CanView(location.MemberID) {
			continue
		}
		h.sendToClient(client, payload)
	}
}

func (h *Hub) broadcastSOS(alert SOSAlert) {
	payload, err := json.Marshal(outboundSOSAlert{
		Type:      messageTypeSOSAlert,
		MemberID:  alert.MemberID,
		Message:   alert.Message,
		CreatedAt: alert.CreatedAt,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal sos alert")
		return
	}

	for client := range h.clients {
		if client.memberID == alert.MemberID {
			continue
		}
		if !client.CanView(alert.MemberID) {
			continue
		}

		h.sendToClient(client, payload)
	}
}

func (h *Hub) sendToClient(client *Client, payload []byte) {
	select {
	case client.send <- payload:
	default:
		close(client.send)
		delete(h.clients, client)
	}
}
