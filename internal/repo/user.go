package repo

import (
	"database/sql"

	"github.com/crucial707/hci-asset/internal/models"
)

type UserRepo struct {
	DB *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{DB: db}
}

// ==========================
// Create User
// ==========================
func (r *UserRepo) Create(username, passwordHash string) (*models.User, error) {
	var id int

	err := r.DB.QueryRow(
		`INSERT INTO users (username, password_hash)
		 VALUES ($1,$2)
		 RETURNING id`,
		username,
		passwordHash,
	).Scan(&id)

	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:       id,
		Username: username,
	}, nil
}

// ==========================
// Get By ID
// ==========================
func (r *UserRepo) GetByID(id int) (*models.User, error) {
	user := models.User{}

	err := r.DB.QueryRow(
		`SELECT id, username FROM users WHERE id=$1`,
		id,
	).Scan(&user.ID, &user.Username)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ==========================
// Get By Username
// ==========================
func (r *UserRepo) GetByUsername(username string) (*models.User, error) {
	user := models.User{}
	var hash string

	err := r.DB.QueryRow(
		`SELECT id, username, password_hash FROM users WHERE username=$1`,
		username,
	).Scan(&user.ID, &user.Username, &hash)

	if err != nil {
		return nil, err
	}

	user.PasswordHash = hash
	return &user, nil
}

// ==========================
// Update User
// ==========================
func (r *UserRepo) Update(id int, username, passwordHash string) (*models.User, error) {
	_, err := r.DB.Exec(
		`UPDATE users
		 SET username=$1, password_hash=$2
		 WHERE id=$3`,
		username,
		passwordHash,
		id,
	)

	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

// ==========================
// Delete User
// ==========================
func (r *UserRepo) Delete(id int) error {
	_, err := r.DB.Exec(
		`DELETE FROM users WHERE id=$1`,
		id,
	)
	return err
}
