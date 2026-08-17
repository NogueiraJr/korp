package models

import (
	"encoding/json"
	"time"
)

type ActivityLog struct {
	ID        int64           `json:"id"`
	Event     string          `json:"event"`
	Entity    string          `json:"entity"`
	Details   json.RawMessage `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}