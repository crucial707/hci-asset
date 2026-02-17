package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/crucial707/hci-asset/internal/models"
	"golang.org/x/crypto/bcrypt"
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
// Create User (password optional for viewer; required for admin; stored as bcrypt hash when set)
// ==========================
func (r *UserRepo) Create(ctx context.Context, username string, password string, role string) (*models.User, error) {
	if role == "" {
		role = models.RoleViewer
	}
	var hash interface{} = nil
	if password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hash = string(h)
	}
	query := `
		INSERT INTO users (username, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, username, role
	`
	user := &models.User{}
	err := r.DB.QueryRowContext(ctx, query, username, hash, role).
		Scan(&user.ID, &user.Username, &user.Role)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ==========================
// Get By ID
// ==========================
func (r *UserRepo) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, role
		FROM users
		WHERE id = $1
	`
	user := &models.User{}
	var pwHash sql.NullString
	err := r.DB.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Username, &pwHash, &user.Role)
	if err != nil {
		return nil, err
	}
	if pwHash.Valid {
		user.PasswordHash = pwHash.String
	}
	return user, nil
}

// ==========================
// Get By Username (includes password_hash for login verification)
// ==========================
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, role
		FROM users
		WHERE username = $1
	`
	user := &models.User{}
	var pwHash sql.NullString
	err := r.DB.QueryRowContext(ctx, query, username).
		Scan(&user.ID, &user.Username, &pwHash, &user.Role)
	if err != nil {
		return nil, err
	}
	if pwHash.Valid {
		user.PasswordHash = pwHash.String
	}
	return user, nil
}

// ==========================
// Update User
// ==========================
func (r *UserRepo) Update(ctx context.Context, id int, username string) (*models.User, error) {
	query := `
		UPDATE users
		SET username = $1
		WHERE id = $2
		RETURNING id, username, password_hash, role
	`
	user := &models.User{}
	var pwHash sql.NullString
	err := r.DB.QueryRowContext(ctx, query, username, id).
		Scan(&user.ID, &user.Username, &pwHash, &user.Role)
	if err != nil {
		return nil, err
	}
	if pwHash.Valid {
		user.PasswordHash = pwHash.String
	}
	return user, nil
}

// ==========================
// Delete User
// ==========================
func (r *UserRepo) Delete(ctx context.Context, id int) error {
	result, err := r.DB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
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
// List Users (password_hash not returned in list)
// ==========================
func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, username, password_hash, role FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var pwHash sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &pwHash, &u.Role); err != nil {
			return nil, err
		}
		// Don't expose password_hash in list
		users = append(users, u)
	}
	return users, nil
}
