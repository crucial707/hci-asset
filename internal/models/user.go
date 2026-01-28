package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // never return password hash in JSON
	CreatedAt    time.Time `json:"created_at"`
}
