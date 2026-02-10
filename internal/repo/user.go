package repo

import (
	"database/sql"
	"errors"

	"github.com/crucial707/hci-asset/internal/models"
)

// ==========================
// UserRepo
// ==========================
type UserRepo struct {
	DB *sql.DB
}

// ==========================
// Constructor
// ==========================
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{DB: db}
}

// ==========================
// Create User
// ==========================
func (r *UserRepo) Create(username string) (*models.User, error) {
	query := `
		INSERT INTO users (username)
		VALUES ($1)
		RETURNING id, username
	`

	user := &models.User{}

	err := r.DB.QueryRow(query, username).
		Scan(&user.ID, &user.Username)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ==========================
// Get By ID
// ==========================
func (r *UserRepo) GetByID(id int) (*models.User, error) {
	query := `
		SELECT id, username
		FROM users
		WHERE id = $1
	`

	user := &models.User{}

	err := r.DB.QueryRow(query, id).
		Scan(&user.ID, &user.Username)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ==========================
// Get By Username
// ==========================
func (r *UserRepo) GetByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username
		FROM users
		WHERE username = $1
	`

	user := &models.User{}

	err := r.DB.QueryRow(query, username).
		Scan(&user.ID, &user.Username)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ==========================
// Update User
// ==========================
func (r *UserRepo) Update(id int, username string) (*models.User, error) {
	query := `
		UPDATE users
		SET username = $1
		WHERE id = $2
		RETURNING id, username
	`

	user := &models.User{}

	err := r.DB.QueryRow(query, username, id).
		Scan(&user.ID, &user.Username)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ==========================
// Delete User
// ==========================
func (r *UserRepo) Delete(id int) error {
	result, err := r.DB.Exec(`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("user not found")
	}

	return nil
}

// ==========================
// List Users
// ==========================
func (r *UserRepo) List() ([]models.User, error) {
	rows, err := r.DB.Query(`SELECT id, username FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}
