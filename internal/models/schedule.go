package models

import "time"

// Schedule represents a recurring scan schedule (cron-like).
type Schedule struct {
	ID        int       `json:"id"`
	Target    string    `json:"target"`
	CronExpr  string    `json:"cron_expr"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}
