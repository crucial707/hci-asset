package repo

import (
	"context"
	"database/sql"

	"github.com/crucial707/hci-asset/internal/models"
)

// AuditRepo persists audit log entries.
type AuditRepo struct {
	db *sql.DB
}

// NewAuditRepo returns a new AuditRepo.
func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Log records an audit entry. action is create|update|delete; resourceType is asset|user.
func (r *AuditRepo) Log(ctx context.Context, userID int, action, resourceType string, resourceID int, details string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (user_id, action, resource_type, resource_id, details) VALUES ($1, $2, $3, $4, $5)`,
		userID, action, resourceType, resourceID, details,
	)
	return err
}

// List returns recent audit entries, newest first.
func (r *AuditRepo) List(ctx context.Context, limit, offset int) ([]models.AuditEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, action, resource_type, resource_id, COALESCE(details,''), created_at FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.ResourceType, &e.ResourceID, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
