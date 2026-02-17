package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// SavedScan is a named scan target that can be re-run.
type SavedScan struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
}

// SavedScanRepo persists saved scan definitions.
type SavedScanRepo struct {
	DB *sql.DB
}

// NewSavedScanRepo returns a new SavedScanRepo.
func NewSavedScanRepo(db *sql.DB) *SavedScanRepo {
	return &SavedScanRepo{DB: db}
}

// Create inserts a saved scan and returns it.
func (r *SavedScanRepo) Create(ctx context.Context, name, target string) (*SavedScan, error) {
	var s SavedScan
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO saved_scans (name, target) VALUES ($1, $2) RETURNING id, name, target, created_at`,
		name, target,
	).Scan(&s.ID, &s.Name, &s.Target, &s.CreatedAt)
	return &s, err
}

// GetByID returns a saved scan by id, or nil if not found.
func (r *SavedScanRepo) GetByID(ctx context.Context, id int) (*SavedScan, error) {
	var s SavedScan
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, name, target, created_at FROM saved_scans WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.Target, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// List returns all saved scans ordered by name.
func (r *SavedScanRepo) List(ctx context.Context) ([]SavedScan, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, name, target, created_at FROM saved_scans ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SavedScan
	for rows.Next() {
		var s SavedScan
		if err := rows.Scan(&s.ID, &s.Name, &s.Target, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// Update updates name and target for a saved scan.
func (r *SavedScanRepo) Update(ctx context.Context, id int, name, target string) (*SavedScan, error) {
	var s SavedScan
	err := r.DB.QueryRowContext(ctx,
		`UPDATE saved_scans SET name = $1, target = $2 WHERE id = $3 RETURNING id, name, target, created_at`,
		name, target, id,
	).Scan(&s.ID, &s.Name, &s.Target, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// Delete removes a saved scan by id.
func (r *SavedScanRepo) Delete(ctx context.Context, id int) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM saved_scans WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("saved scan not found")
	}
	return nil
}
