package repo

import (
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

// List returns schedules, most recent first. limit/offset for pagination.
func (r *ScheduleRepo) List(limit, offset int) ([]models.Schedule, error) {
	query := `
		SELECT id, target, cron_expr, enabled, created_at
		FROM scan_schedules
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.DB.Query(query, limit, offset)
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
func (r *ScheduleRepo) ListEnabled() ([]models.Schedule, error) {
	query := `
		SELECT id, target, cron_expr, enabled, created_at
		FROM scan_schedules
		WHERE enabled = true
		ORDER BY id
	`
	rows, err := r.DB.Query(query)
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
func (r *ScheduleRepo) GetByID(id int) (*models.Schedule, error) {
	query := `
		SELECT id, target, cron_expr, enabled, created_at
		FROM scan_schedules
		WHERE id = $1
	`
	s := &models.Schedule{}
	err := r.DB.QueryRow(query, id).Scan(&s.ID, &s.Target, &s.CronExpr, &s.Enabled, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Create inserts a new schedule and returns it with id set.
func (r *ScheduleRepo) Create(target, cronExpr string, enabled bool) (*models.Schedule, error) {
	query := `
		INSERT INTO scan_schedules (target, cron_expr, enabled)
		VALUES ($1, $2, $3)
		RETURNING id, target, cron_expr, enabled, created_at
	`
	s := &models.Schedule{}
	err := r.DB.QueryRow(query, target, cronExpr, enabled).
		Scan(&s.ID, &s.Target, &s.CronExpr, &s.Enabled, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Update updates target, cron_expr, and enabled for the given id.
func (r *ScheduleRepo) Update(id int, target, cronExpr string, enabled bool) error {
	_, err := r.DB.Exec(
		`UPDATE scan_schedules SET target = $1, cron_expr = $2, enabled = $3 WHERE id = $4`,
		target, cronExpr, enabled, id,
	)
	return err
}

// Delete removes a schedule by id.
func (r *ScheduleRepo) Delete(id int) error {
	_, err := r.DB.Exec(`DELETE FROM scan_schedules WHERE id = $1`, id)
	return err
}
