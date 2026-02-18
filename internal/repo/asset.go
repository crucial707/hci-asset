package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/lib/pq"
)

type AssetRepo struct {
	db *sql.DB
}

// ErrAssetNotFound is returned when an asset cannot be found.
var ErrAssetNotFound = errors.New("asset not found")

// ==========================
// Constructor
// ==========================
func NewAssetRepo(db *sql.DB) *AssetRepo {
	return &AssetRepo{db: db}
}

// ==========================
// Create a new asset
// ==========================
func (r *AssetRepo) Create(ctx context.Context, name, description string, tags []string) (*models.Asset, error) {
	if tags == nil {
		tags = []string{}
	}
	var id int
	err := r.db.QueryRowContext(ctx,
		"INSERT INTO assets (name, description, tags) VALUES ($1, $2, $3) RETURNING id",
		name, description, pq.Array(tags),
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &models.Asset{
		ID:          id,
		Name:        name,
		Description: description,
		Tags:        tags,
	}, nil
}

// ==========================
// Find asset by name
// ==========================
func (r *AssetRepo) FindByName(ctx context.Context, name string) (*models.Asset, error) {
	var a models.Asset
	var lastSeen sql.NullTime
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, description, COALESCE(tags, '{}'), last_seen, COALESCE(network_name, '') FROM assets WHERE name=$1",
		name,
	).Scan(&a.ID, &a.Name, &a.Description, pq.Array(&a.Tags), &lastSeen, &a.NetworkName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}
	if lastSeen.Valid {
		a.LastSeen = &lastSeen.Time
	}
	return &a, nil
}

// ==========================
// Upsert discovered asset (idempotent)
// ==========================
// UpsertDiscovered either finds an existing asset by name or creates it.
// This is primarily used by scan jobs to avoid duplicating assets.
func (r *AssetRepo) UpsertDiscovered(ctx context.Context, name, description string) (*models.Asset, error) {
	a, err := r.FindByName(ctx, name)
	if err == nil {
		// Optionally update description if it has changed, but avoid errors
		if a.Description != description {
			updated, updateErr := r.Update(ctx, a.ID, a.Name, description, a.Tags)
			if updateErr == nil {
				return updated, nil
			}
		}
		return a, nil
	}

	// If not found, create a new asset
	if errors.Is(err, ErrAssetNotFound) {
		return r.Create(ctx, name, description, nil)
	}

	return nil, err
}

// Count returns the total number of assets.
func (r *AssetRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM assets").Scan(&n)
	return n, err
}

// CountByTag returns the number of assets with the given tag.
func (r *AssetRepo) CountByTag(ctx context.Context, tag string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM assets WHERE $1 = ANY(COALESCE(tags, '{}'))", tag).Scan(&n)
	return n, err
}

// CountSearch returns the number of assets matching the search query (name or description).
func (r *AssetRepo) CountSearch(ctx context.Context, query string) (int, error) {
	likeQuery := "%" + strings.ToLower(query) + "%"
	var n int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM assets WHERE LOWER(name) LIKE $1 OR LOWER(description) LIKE $1", likeQuery).Scan(&n)
	return n, err
}

// ==========================
// List assets with pagination
// ==========================
func (r *AssetRepo) List(ctx context.Context, limit, offset int) ([]models.Asset, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, description, COALESCE(tags, '{}'), last_seen, COALESCE(network_name, '') FROM assets ORDER BY id LIMIT $1 OFFSET $2",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssetRows(rows)
}

// ListByTag returns assets that have the given tag.
func (r *AssetRepo) ListByTag(ctx context.Context, tag string, limit, offset int) ([]models.Asset, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, description, COALESCE(tags, '{}'), last_seen, COALESCE(network_name, '') FROM assets WHERE $1 = ANY(COALESCE(tags, '{}')) ORDER BY id LIMIT $2 OFFSET $3",
		tag, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssetRows(rows)
}

func (r *AssetRepo) scanAssetRows(rows *sql.Rows) ([]models.Asset, error) {
	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		var lastSeen sql.NullTime
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, pq.Array(&a.Tags), &lastSeen, &a.NetworkName); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			a.LastSeen = &lastSeen.Time
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// ==========================
// Search assets with pagination
// ==========================
func (r *AssetRepo) Search(ctx context.Context, query string, limit, offset int) ([]models.Asset, error) {
	likeQuery := "%" + strings.ToLower(query) + "%"
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, description, COALESCE(tags, '{}'), last_seen, COALESCE(network_name, '') FROM assets WHERE LOWER(name) LIKE $1 OR LOWER(description) LIKE $1 ORDER BY id LIMIT $2 OFFSET $3",
		likeQuery, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanAssetRows(rows)
}

// ==========================
// Get an asset by ID
// ==========================
func (r *AssetRepo) Get(ctx context.Context, id int) (*models.Asset, error) {
	var a models.Asset
	var lastSeen sql.NullTime
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, description, COALESCE(tags, '{}'), last_seen, COALESCE(network_name, '') FROM assets WHERE id=$1", id,
	).Scan(&a.ID, &a.Name, &a.Description, pq.Array(&a.Tags), &lastSeen, &a.NetworkName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, err
	}
	if lastSeen.Valid {
		a.LastSeen = &lastSeen.Time
	}
	return &a, nil
}

// ==========================
// Heartbeat updates last_seen for an asset (e.g. agent check-in).
// ==========================
func (r *AssetRepo) Heartbeat(ctx context.Context, id int) (*models.Asset, error) {
	res, err := r.db.ExecContext(ctx, "UPDATE assets SET last_seen = NOW() WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("asset not found")
	}
	return r.Get(ctx, id)
}

// ==========================
// Update an asset by ID
// ==========================
func (r *AssetRepo) Update(ctx context.Context, id int, name, description string, tags []string) (*models.Asset, error) {
	if tags == nil {
		tags = []string{}
	}
	res, err := r.db.ExecContext(ctx,
		"UPDATE assets SET name=$1, description=$2, tags=$3 WHERE id=$4",
		name, description, pq.Array(tags), id,
	)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("asset not found")
	}
	return r.Get(ctx, id)
}

// ==========================
// UpdateNetworkName sets the network_name (e.g. IP) for an asset.
// Used by scan jobs to persist the discovered IP.
// ==========================
func (r *AssetRepo) UpdateNetworkName(ctx context.Context, id int, networkName string) error {
	res, err := r.db.ExecContext(ctx, "UPDATE assets SET network_name = $1 WHERE id = $2", networkName, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("asset not found")
	}
	return nil
}

// ==========================
// Delete an asset by ID
// ==========================
func (r *AssetRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM assets WHERE id=$1", id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("asset not found")
	}
	return nil
}
