package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/crucial707/hci-asset/internal/models"
)

// ScanJobRow represents a persisted scan job (for API response shape).
type ScanJobRow struct {
	ID          int            `json:"id"`
	Target      string         `json:"target"`
	Status      string         `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
	Assets      []models.Asset `json:"assets,omitempty"`
}

// ScanJobRepo persists scan jobs.
type ScanJobRepo struct {
	DB *sql.DB
}

// NewScanJobRepo returns a new ScanJobRepo.
func NewScanJobRepo(db *sql.DB) *ScanJobRepo {
	return &ScanJobRepo{DB: db}
}

// Create inserts a new scan job with status=running and returns its id.
func (r *ScanJobRepo) Create(ctx context.Context, target string) (int, error) {
	var id int
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO scan_jobs (target, status) VALUES ($1, 'running') RETURNING id`,
		target,
	).Scan(&id)
	return id, err
}

// Update sets status, completed_at, error, and assets for a job.
func (r *ScanJobRepo) Update(ctx context.Context, id int, status string, completedAt *time.Time, errMsg string, assets []models.Asset) error {
	var assetsJSON []byte
	if len(assets) > 0 {
		var err error
		assetsJSON, err = json.Marshal(assets)
		if err != nil {
			return err
		}
	}
	_, err := r.DB.ExecContext(ctx,
		`UPDATE scan_jobs SET status = $1, completed_at = $2, error = $3, assets = $4 WHERE id = $5`,
		status, completedAt, nullString(errMsg), nullJSON(assetsJSON), id,
	)
	return err
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

// GetByID returns a scan job by id, or nil if not found.
func (r *ScanJobRepo) GetByID(ctx context.Context, id int) (*ScanJobRow, error) {
	var row ScanJobRow
	var completedAt sql.NullTime
	var errMsg sql.NullString
	var assetsJSON []byte
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, target, status, started_at, completed_at, error, assets FROM scan_jobs WHERE id = $1`,
		id,
	).Scan(&row.ID, &row.Target, &row.Status, &row.StartedAt, &completedAt, &errMsg, &assetsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		row.CompletedAt = &completedAt.Time
	}
	if errMsg.Valid {
		row.Error = errMsg.String
	}
	if len(assetsJSON) > 0 {
		_ = json.Unmarshal(assetsJSON, &row.Assets)
	}
	return &row, nil
}

// ListEntry is one row for List.
type ListEntry struct {
	ID        int       `json:"id"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// Count returns the total number of scan jobs.
func (r *ScanJobRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM scan_jobs").Scan(&n)
	return n, err
}

// List returns recent scan jobs, ordered by id DESC.
func (r *ScanJobRepo) List(ctx context.Context, limit, offset int) ([]ListEntry, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, target, status, started_at FROM scan_jobs ORDER BY id DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ListEntry
	for rows.Next() {
		var e ListEntry
		if err := rows.Scan(&e.ID, &e.Target, &e.Status, &e.StartedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// DeleteAll removes all scan job records (used to clear the active scans list).
func (r *ScanJobRepo) DeleteAll(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM scan_jobs")
	return err
}
