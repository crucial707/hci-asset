package repo

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/crucial707/hci-asset/internal/models"
)

type AssetRepo struct {
	db *sql.DB
}

// ==========================
// Constructor
// ==========================
func NewAssetRepo(db *sql.DB) *AssetRepo {
	return &AssetRepo{db: db}
}

// ==========================
// Create a new asset
// ==========================
func (r *AssetRepo) Create(name, description string) (*models.Asset, error) {
	var id int
	err := r.db.QueryRow(
		"INSERT INTO assets (name, description) VALUES ($1, $2) RETURNING id",
		name, description,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &models.Asset{
		ID:          id,
		Name:        name,
		Description: description,
	}, nil
}

// ==========================
// List assets with pagination
// ==========================
func (r *AssetRepo) List(limit, offset int) ([]models.Asset, error) {
	rows, err := r.db.Query(
		"SELECT id, name, description FROM assets ORDER BY id LIMIT $1 OFFSET $2",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.Description); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}

	return assets, nil
}

// ==========================
// Search assets with pagination
// ==========================
func (r *AssetRepo) Search(query string, limit, offset int) ([]models.Asset, error) {
	likeQuery := "%" + strings.ToLower(query) + "%"
	rows, err := r.db.Query(
		"SELECT id, name, description FROM assets WHERE LOWER(name) LIKE $1 OR LOWER(description) LIKE $1 ORDER BY id LIMIT $2 OFFSET $3",
		likeQuery, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.Description); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}

	return assets, nil
}

// ==========================
// Get an asset by ID
// ==========================
func (r *AssetRepo) Get(id int) (*models.Asset, error) {
	var a models.Asset
	err := r.db.QueryRow(
		"SELECT id, name, description FROM assets WHERE id=$1", id,
	).Scan(&a.ID, &a.Name, &a.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, err
	}
	return &a, nil
}

// ==========================
// Update an asset by ID
// ==========================
func (r *AssetRepo) Update(id int, name, description string) (*models.Asset, error) {
	res, err := r.db.Exec(
		"UPDATE assets SET name=$1, description=$2 WHERE id=$3",
		name, description, id,
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

	return &models.Asset{
		ID:          id,
		Name:        name,
		Description: description,
	}, nil
}

// ==========================
// Delete an asset by ID
// ==========================
func (r *AssetRepo) Delete(id int) error {
	res, err := r.db.Exec("DELETE FROM assets WHERE id=$1", id)
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
