package repo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestScheduleRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}).
			AddRow(2, "10.0.0.0/24", "0 * * * *", true, now).
			AddRow(1, "192.168.1.0/24", "*/5 * * * *", false, now.Add(-time.Hour)))

	r := NewScheduleRepo(db)
	list, err := r.List(context.Background(), 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if list[0].ID != 2 || list[0].Target != "10.0.0.0/24" || list[0].CronExpr != "0 * * * *" || !list[0].Enabled {
		t.Errorf("unexpected first item: %+v", list[0])
	}
	if list[1].ID != 1 || list[1].Target != "192.168.1.0/24" || !list[1].CreatedAt.Before(list[0].CreatedAt) {
		t.Errorf("unexpected second item: %+v", list[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_List_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}))

	r := NewScheduleRepo(db)
	list, err := r.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_ListEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "cron_expr", "enabled", "created_at"}).
			AddRow(1, "192.168.1.0/24", "0 * * * *", true, now))

	r := NewScheduleRepo(db)
	list, err := r.ListEnabled(context.Background())
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 item, got %d", len(list))
	}
	if list[0].ID != 1 || list[0].Target != "192.168.1.0/24" || !list[0].Enabled {
		t.Errorf("unexpected item: %+v", list[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_GetByID(t *testing.T) {
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

	r := NewScheduleRepo(db)
	s, err := r.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if s == nil {
		t.Fatal("expected schedule, got nil")
	}
	if s.ID != 1 || s.Target != "192.168.1.0/24" || s.CronExpr != "0 * * * *" || !s.Enabled {
		t.Errorf("unexpected schedule: %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, target, cron_expr, enabled, created_at`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	r := NewScheduleRepo(db)
	s, err := r.GetByID(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if s != nil {
		t.Errorf("expected nil, got %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_Create(t *testing.T) {
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

	r := NewScheduleRepo(db)
	s, err := r.Create(context.Background(), "192.168.1.0/24", "0 * * * *", true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID != 1 || s.Target != "192.168.1.0/24" || s.CronExpr != "0 * * * *" || !s.Enabled {
		t.Errorf("unexpected schedule: %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`UPDATE scan_schedules SET target`).
		WithArgs("10.0.0.0/24", "*/15 * * * *", false, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := NewScheduleRepo(db)
	err = r.Update(context.Background(), 1, "10.0.0.0/24", "*/15 * * * *", false)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestScheduleRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`DELETE FROM scan_schedules WHERE id`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := NewScheduleRepo(db)
	err = r.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
