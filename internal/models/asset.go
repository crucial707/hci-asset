package models

import "time"

type Asset struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tags        []string   `json:"tags,omitempty"`
	NetworkName string     `json:"network_name,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
}
