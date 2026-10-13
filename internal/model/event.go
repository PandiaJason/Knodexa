package model

import "time"

// Event represents an append-only audit event in Knodexa's event ledger (events.jsonl).
type Event struct {
	EventID   string                 `json:"event_id"`
	Timestamp time.Time              `json:"timestamp"`
	Actor     string                 `json:"actor"`
	Action    string                 `json:"action"`
	TargetID  string                 `json:"target_id"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}
