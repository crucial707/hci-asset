package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/crucial707/hci-asset/internal/repo"
)

func TestUserHandler_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO users \(username, password_hash\)`).
		WithArgs("charlie", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(3, "charlie"))

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	body, _ := json.Marshal(map[string]string{"username": "charlie"})
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("CreateUser status: got %d, want 200", rr.Code)
	}
	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 3 || user.Username != "charlie" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_CreateUser_BadRequest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	body, _ := json.Marshal(map[string]string{"username": ""})
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("CreateUser status: got %d, want 400", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_ListUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash FROM users ORDER BY id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash"}).
			AddRow(1, "alice", nil).
			AddRow(2, "bob", nil))

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	req := httptest.NewRequest("GET", "/users", nil)
	rr := httptest.NewRecorder()
	h.ListUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListUsers status: got %d, want 200", rr.Code)
	}
	var list []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 2 || list[0].Username != "alice" || list[1].Username != "bob" {
		t.Errorf("unexpected list: %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_GetUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash"}).AddRow(1, "alice", nil))

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	req := requestWithChiURLParams("GET", "/users/1", nil, map[string]string{"id": "1"})
	rr := httptest.NewRecorder()
	h.GetUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetUser status: got %d, want 200", rr.Code)
	}
	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 1 || user.Username != "alice" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	req := requestWithChiURLParams("GET", "/users/999", nil, map[string]string{"id": "999"})
	rr := httptest.NewRecorder()
	h.GetUser(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GetUser status: got %d, want 404", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_GetUser_InvalidID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	req := requestWithChiURLParams("GET", "/users/abc", nil, map[string]string{"id": "abc"})
	rr := httptest.NewRecorder()
	h.GetUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("GetUser status: got %d, want 400", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`UPDATE users`).
		WithArgs("alice2", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash"}).AddRow(1, "alice2", nil))

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	body, _ := json.Marshal(map[string]string{"username": "alice2"})
	req := requestWithChiURLParams("PUT", "/users/1", body, map[string]string{"id": "1"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("UpdateUser status: got %d, want 200", rr.Code)
	}
	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Username != "alice2" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestUserHandler_DeleteUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	userRepo := repo.NewUserRepo(db)
	h := &UserHandler{Repo: userRepo}

	req := requestWithChiURLParams("DELETE", "/users/1", nil, map[string]string{"id": "1"})
	rr := httptest.NewRecorder()
	h.DeleteUser(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("DeleteUser status: got %d, want 204", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
