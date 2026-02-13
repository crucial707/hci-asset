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
	"golang.org/x/crypto/bcrypt"
)

func TestAuthHandler_Login(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash"}).AddRow(1, "alice", nil))

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	body, _ := json.Marshal(map[string]string{"username": "alice"})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Login status: got %d, want 200", rr.Code)
	}
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Token == "" || out.User.Username != "alice" || out.User.ID != 1 {
		t.Errorf("unexpected response: token=%q user=%+v", out.Token, out.User)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash`).
		WithArgs("nobody").
		WillReturnError(sql.ErrNoRows)

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	body, _ := json.Marshal(map[string]string{"username": "nobody"})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Login status: got %d, want 401", rr.Code)
	}
	var out map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out["error"] != "invalid credentials" {
		t.Errorf("unexpected error: %v", out["error"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAuthHandler_Login_WithPassword_Success(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash"}).AddRow(1, "alice", string(hash)))

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "secret"})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Login status: got %d, want 200", rr.Code)
	}
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Token == "" || out.User.Username != "alice" {
		t.Errorf("unexpected response: %+v", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAuthHandler_Login_WithPassword_WrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, username, password_hash`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash"}).AddRow(1, "alice", string(hash)))

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong"})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Login status: got %d, want 401", rr.Code)
	}
	var out map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out["error"] != "invalid credentials" {
		t.Errorf("unexpected error: %v", out["error"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAuthHandler_Login_BadJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Login status: got %d, want 400", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAuthHandler_Register(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO users \(username, password_hash\)`).
		WithArgs("bob", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(2, "bob"))

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	body, _ := json.Marshal(map[string]string{"username": "bob"})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Register status: got %d, want 200", rr.Code)
	}
	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 2 || user.Username != "bob" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAuthHandler_Register_BadJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	userRepo := repo.NewUserRepo(db)
	h := &AuthHandler{UserRepo: userRepo, Secret: []byte("test-secret")}

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Register status: got %d, want 400", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
