package models

import "time"

// AuditEntry represents one audit log row.
type AuditEntry struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Action       string    `json:"action"`        // create, update, delete
	ResourceType string    `json:"resource_type"` // asset, user
	ResourceID   int       `json:"resource_id"`
	Details      string    `json:"details,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
