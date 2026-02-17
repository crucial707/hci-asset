package repo

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO users \(username, password_hash, role\)`).
		WithArgs("alice", nil, "viewer").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).AddRow(1, "alice", "viewer"))

	repo := NewUserRepo(db)
	user, err := repo.Create(context.Background(), "alice", "", "viewer")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID != 1 || user.Username != "alice" || user.Role != "viewer" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash, role`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "role"}).AddRow(1, "bob", nil, "viewer"))

	repo := NewUserRepo(db)
	user, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if user.ID != 1 || user.Username != "bob" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash, role`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	repo := NewUserRepo(db)
	_, err = repo.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing user")
	}
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash, role`).
		WithArgs("charlie").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "role"}).AddRow(2, "charlie", nil, "viewer"))

	repo := NewUserRepo(db)
	user, err := repo.GetByUsername(context.Background(), "charlie")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if user.ID != 2 || user.Username != "charlie" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
