package repo

import (
	"database/sql"
	"errors"

	"github.com/crucial707/hci-asset/internal/models"
)

type UserRepo struct {
	DB *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{DB: db}
}

// Create a new user
func (r *UserRepo) Create(username, passwordHash string) (models.User, error) {
	var user models.User
	err := r.DB.QueryRow(
		`INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id, username, password_hash, created_at`,
		username, passwordHash,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	return user, err
}

// Get user by username
func (r *UserRepo) GetByUsername(username string) (models.User, error) {
	var user models.User
	err := r.DB.QueryRow(
		`SELECT id, username, password_hash, created_at FROM users WHERE username=$1`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return user, errors.New("user not found")
	}
	return user, err
}
