package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/crucial707/hci-asset/internal/repo"
)

func TestScheduleHandler_ListSchedules(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}).
			AddRow(1, "192.168.1.0/24", "0 * * * *", true, now))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM scan_schedules`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := httptest.NewRequest("GET", "/schedules", nil)
	rr := httptest.NewRecorder()
	h.ListSchedules(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListSchedules status: got %d, want 200", rr.Code)
	}
	var listResp struct {
		Items []struct {
			ID        int    `json:"id"`
			Target    string `json:"target"`
			CronExpr  string `json:"cron_expr"`
			Enabled   bool   `json:"enabled"`
			CreatedAt string `json:"created_at"`
		} `json:"items"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].ID != 1 || listResp.Items[0].Target != "192.168.1.0/24" {
		t.Errorf("unexpected list: %+v", listResp.Items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_ListSchedules_QueryParams(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(10, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM scan_schedules`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := httptest.NewRequest("GET", "/schedules?limit=10&offset=20", nil)
	rr := httptest.NewRecorder()
	h.ListSchedules(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListSchedules status: got %d, want 200", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_GetSchedule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}).
			AddRow(1, "192.168.1.0/24", "0 * * * *", true, now))

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := requestWithChiURLParams("GET", "/schedules/1", nil, map[string]string{"id": "1"})
	rr := httptest.NewRecorder()
	h.GetSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetSchedule status: got %d, want 200", rr.Code)
	}
	var s struct {
		ID       int    `json:"id"`
		Target   string `json:"target"`
		CronExpr string `json:"cron_expr"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&s); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if s.ID != 1 || s.Target != "192.168.1.0/24" || !s.Enabled {
		t.Errorf("unexpected schedule: %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_GetSchedule_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := requestWithChiURLParams("GET", "/schedules/999", nil, map[string]string{"id": "999"})
	rr := httptest.NewRecorder()
	h.GetSchedule(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GetSchedule status: got %d, want 404", rr.Code)
	}
	var out map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&out)
	if out["error"] != "schedule not found" {
		t.Errorf("unexpected error body: %v", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_GetSchedule_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := requestWithChiURLParams("GET", "/schedules/abc", nil, map[string]string{"id": "abc"})
	rr := httptest.NewRecorder()
	h.GetSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("GetSchedule status: got %d, want 400", rr.Code)
	}
}

func TestScheduleHandler_CreateSchedule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`INSERT INTO scan_schedules`).
		WithArgs("192.168.1.0/24", "0 * * * *", true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}).
			AddRow(1, "192.168.1.0/24", "0 * * * *", true, now))

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	body, _ := json.Marshal(map[string]interface{}{"target": "192.168.1.0/24", "cron_expr": "0 * * * *", "enabled": true})
	req := httptest.NewRequest("POST", "/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateSchedule(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("CreateSchedule status: got %d, want 201", rr.Code)
	}
	var s struct {
		ID       int    `json:"id"`
		Target   string `json:"target"`
		CronExpr string `json:"cron_expr"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&s); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if s.ID != 1 || s.Target != "192.168.1.0/24" || !s.Enabled {
		t.Errorf("unexpected schedule: %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_CreateSchedule_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	body, _ := json.Marshal(map[string]string{"target": "", "cron_expr": "0 * * * *"})
	req := httptest.NewRequest("POST", "/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("CreateSchedule status: got %d, want 400", rr.Code)
	}
}

func TestScheduleHandler_UpdateSchedule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`UPDATE scan_schedules SET target`).
		WithArgs("10.0.0.0/24", "*/15 * * * *", false, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	now := time.Now()
	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}).
			AddRow(1, "10.0.0.0/24", "*/15 * * * *", false, now))

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	body, _ := json.Marshal(map[string]interface{}{"target": "10.0.0.0/24", "cron_expr": "*/15 * * * *", "enabled": false})
	req := requestWithChiURLParams("PUT", "/schedules/1", body, map[string]string{"id": "1"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateSchedule(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("UpdateSchedule status: got %d, want 200", rr.Code)
	}
	var s struct {
		ID       int    `json:"id"`
		Target   string `json:"target"`
		CronExpr string `json:"cron_expr"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&s); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if s.Target != "10.0.0.0/24" || s.CronExpr != "*/15 * * * *" || s.Enabled {
		t.Errorf("unexpected schedule: %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_UpdateSchedule_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	body, _ := json.Marshal(map[string]string{"target": "10.0.0.0/24", "cron_expr": "0 * * * *"})
	req := requestWithChiURLParams("PUT", "/schedules/xyz", body, map[string]string{"id": "xyz"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("UpdateSchedule status: got %d, want 400", rr.Code)
	}
}

func TestScheduleHandler_DeleteSchedule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`DELETE FROM scan_schedules WHERE id`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := requestWithChiURLParams("DELETE", "/schedules/1", nil, map[string]string{"id": "1"})
	rr := httptest.NewRecorder()
	h.DeleteSchedule(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("DeleteSchedule status: got %d, want 204", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleHandler_DeleteSchedule_InvalidID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	scheduleRepo := repo.NewScheduleRepo(db)
	h := &ScheduleHandler{Repo: scheduleRepo}

	req := requestWithChiURLParams("DELETE", "/schedules/bad", nil, map[string]string{"id": "bad"})
	rr := httptest.NewRecorder()
	h.DeleteSchedule(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("DeleteSchedule status: got %d, want 400", rr.Code)
	}
}
