package repo

import (
	"database/sql"

	"github.com/crucial707/hci-asset/internal/models"
)

type AssetRepo struct {
	DB *sql.DB
}

func NewAssetRepo(db *sql.DB) *AssetRepo {
	return &AssetRepo{DB: db}
}

func (r *AssetRepo) Create(name, description string) (models.Asset, error) {
	var asset models.Asset
	err := r.DB.QueryRow(
		"INSERT INTO assets (name, description) VALUES ($1, $2) RETURNING id, name, description, created_at",
		name, description,
	).Scan(&asset.ID, &asset.Name, &asset.Description, &asset.CreatedAt)
	return asset, err
}
