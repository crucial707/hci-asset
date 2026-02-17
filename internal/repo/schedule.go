package repo

import (
	"context"
	"database/sql"

	"github.com/crucial707/hci-asset/internal/models"
)

// ScheduleRepo persists scan schedules.
type ScheduleRepo struct {
	DB *sql.DB
}

// NewScheduleRepo returns a new ScheduleRepo.
func NewScheduleRepo(db *sql.DB) *ScheduleRepo {
	return &ScheduleRepo{DB: db}
}

// Count returns the total number of schedules.
func (r *ScheduleRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM scan_schedules").Scan(&n)
	return n, err
}

// List returns schedules, most recent first. limit/offset for pagination.
func (r *ScheduleRepo) List(ctx context.Context, limit, offset int) ([]models.Schedule, error) {
	query := `
		SELECT id, target, cron_expr, enabled, created_at
		FROM scan_schedules
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Schedule
	for rows.Next() {
		var s models.Schedule
		if err := rows.Scan(&s.ID, &s.Target, &s.CronExpr, &s.Enabled, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// ListEnabled returns all enabled schedules (for the cron runner).
func (r *ScheduleRepo) ListEnabled(ctx context.Context) ([]models.Schedule, error) {
	query := `
		SELECT id, target, cron_expr, enabled, created_at
		FROM scan_schedules
		WHERE enabled = true
		ORDER BY id
	`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Schedule
	for rows.Next() {
		var s models.Schedule
		if err := rows.Scan(&s.ID, &s.Target, &s.CronExpr, &s.Enabled, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetByID returns one schedule by id.
func (r *ScheduleRepo) GetByID(ctx context.Context, id int) (*models.Schedule, error) {
	query := `
		SELECT id, target, cron_expr, enabled, created_at
		FROM scan_schedules
		WHERE id = $1
	`
	s := &models.Schedule{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.Target, &s.CronExpr, &s.Enabled, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Create inserts a new schedule and returns it with id set.
func (r *ScheduleRepo) Create(ctx context.Context, target, cronExpr string, enabled bool) (*models.Schedule, error) {
	query := `
		INSERT INTO scan_schedules (target, cron_expr, enabled)
		VALUES ($1, $2, $3)
		RETURNING id, target, cron_expr, enabled, created_at
	`
	s := &models.Schedule{}
	err := r.DB.QueryRowContext(ctx, query, target, cronExpr, enabled).
		Scan(&s.ID, &s.Target, &s.CronExpr, &s.Enabled, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Update updates target, cron_expr, and enabled for the given id.
func (r *ScheduleRepo) Update(ctx context.Context, id int, target, cronExpr string, enabled bool) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE scan_schedules SET target = $1, cron_expr = $2, enabled = $3 WHERE id = $4`,
		target, cronExpr, enabled, id,
	)
	return err
}

// Delete removes a schedule by id.
func (r *ScheduleRepo) Delete(ctx context.Context, id int) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM scan_schedules WHERE id = $1`, id)
	return err
}
