package ws

import "strings"

const (
	messageTypeLocationUpdate   = "location_update"
	messageTypeLocationSnapshot = "location_snapshot"
)

type MemberLocation struct {
	MemberID  string  `json:"memberId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	UpdatedAt int64   `json:"updatedAt"`
}

type inboundLocationMessage struct {
	Type      string  `json:"type"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type outboundLocationUpdate struct {
	Type     string         `json:"type"`
	Location MemberLocation `json:"location"`
}

type outboundLocationSnapshot struct {
	Type      string           `json:"type"`
	Locations []MemberLocation `json:"locations"`
}

func normalizeMemberID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
