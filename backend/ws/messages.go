package ws

import "strings"

const (
	messageTypeLocationUpdate   = "location_update"
	messageTypeLocationSnapshot = "location_snapshot"
	messageTypeSOSAlert         = "sos_alert"
)

type MemberLocation struct {
	MemberID  string  `json:"memberId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	UpdatedAt int64   `json:"updatedAt"`
}

type inboundClientMessage struct {
	Type      string  `json:"type"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Message   string  `json:"message"`
}

type outboundLocationUpdate struct {
	Type     string         `json:"type"`
	Location MemberLocation `json:"location"`
}

type outboundLocationSnapshot struct {
	Type      string           `json:"type"`
	Locations []MemberLocation `json:"locations"`
}

type SOSAlert struct {
	MemberID  string `json:"memberId"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"createdAt"`
}

type outboundSOSAlert struct {
	Type      string `json:"type"`
	MemberID  string `json:"memberId"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"createdAt"`
}

func normalizeMemberID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
