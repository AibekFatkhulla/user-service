package domain

import "time"

type AuditEvent struct {
	Service    string                 `json:"service"`
	EventType  string                 `json:"event_type"`
	EntityID   string                 `json:"entity_id"`
	Actor      string                 `json:"actor,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
	Payload    map[string]interface{} `json:"payload"`
}
