package repo

import (
	"database/sql"

	"github.com/crucial707/hci-asset/internal/models"
)

// ========================
// REPOSITORY STRUCT
// ========================

type AssetRepo struct {
	DB *sql.DB
}

func NewAssetRepo(db *sql.DB) *AssetRepo {
	return &AssetRepo{DB: db}
}

// ========================
// CREATE ASSET
// ========================

func (r *AssetRepo) Create(name, description string) (models.Asset, error) {
	var asset models.Asset
	err := r.DB.QueryRow(
		`INSERT INTO assets (name, description)
		 VALUES ($1, $2)
		 RETURNING id, name, description, created_at`,
		name, description,
	).Scan(
		&asset.ID,
		&asset.Name,
		&asset.Description,
		&asset.CreatedAt,
	)
	return asset, err
}

// ========================
// GET ASSET BY ID
// ========================

func (r *AssetRepo) GetByID(id int) (models.Asset, error) {
	var asset models.Asset
	err := r.DB.QueryRow(
		`SELECT id, name, description, created_at
		 FROM assets
		 WHERE id = $1`,
		id,
	).Scan(
		&asset.ID,
		&asset.Name,
		&asset.Description,
		&asset.CreatedAt,
	)
	return asset, err
}

// ========================
// DELETE ASSET BY ID
// ========================

func (r *AssetRepo) DeleteByID(id int) error {
	_, err := r.DB.Exec("DELETE FROM assets WHERE id = $1", id)
	return err
}

// ========================
// UPDATE ASSET BY ID
// ========================

func (r *AssetRepo) UpdateByID(id int, name, description string) (models.Asset, error) {
	var asset models.Asset
	err := r.DB.QueryRow(
		`UPDATE assets 
		 SET name = $1, description = $2
		 WHERE id = $3
		 RETURNING id, name, description, created_at`,
		name, description, id,
	).Scan(
		&asset.ID,
		&asset.Name,
		&asset.Description,
		&asset.CreatedAt,
	)
	return asset, err
}

// ========================
// LIST ALL ASSETS
// ========================

func (r *AssetRepo) List() ([]models.Asset, error) {
	rows, err := r.DB.Query("SELECT id, name, description, created_at FROM assets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// ========================
// LIST ASSETS WITH PAGINATION
// ========================

func (r *AssetRepo) ListPaginated(limit, offset int) ([]models.Asset, error) {
	rows, err := r.DB.Query(
		"SELECT id, name, description, created_at FROM assets ORDER BY id LIMIT $1 OFFSET $2",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// ========================
// SEARCH ASSETS WITH PAGINATION
// ========================

func (r *AssetRepo) SearchPaginated(query string, limit, offset int) ([]models.Asset, error) {
	rows, err := r.DB.Query(`
        SELECT id, name, description, created_at
        FROM assets
        WHERE name ILIKE $1 OR description ILIKE $1
        ORDER BY id
        LIMIT $2 OFFSET $3
    `, "%"+query+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}
