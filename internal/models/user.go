package models

// RoleViewer is the only role that can log in without a password.
const RoleViewer = "viewer"
const RoleAdmin = "admin"

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}
