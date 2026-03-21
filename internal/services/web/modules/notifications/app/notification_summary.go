package app

import "time"

// NotificationSummary is a transport-safe summary for notification listings.
type NotificationSummary struct {
	ID          string     `json:"id"`
	MessageType string     `json:"message_type"`
	PayloadJSON string     `json:"payload_json"`
	Source      string     `json:"source"`
	Read        bool       `json:"read"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}
