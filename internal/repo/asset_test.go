package repo

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestAssetRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO assets \(name, description, tags\) VALUES \(\$1, \$2, \$3\) RETURNING id`).
		WithArgs("my-asset", "my desc", pq.Array([]string{})).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	repo := NewAssetRepo(db)
	asset, err := repo.Create("my-asset", "my desc", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if asset.ID != 42 || asset.Name != "my-asset" || asset.Description != "my desc" {
		t.Errorf("unexpected asset: %+v", asset)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetRepo_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets WHERE id=\$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tags", "last_seen"}).
			AddRow(1, "a1", "desc1", "{}", now))

	repo := NewAssetRepo(db)
	asset, err := repo.Get(1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if asset.ID != 1 || asset.Name != "a1" || asset.Description != "desc1" || asset.LastSeen == nil {
		t.Errorf("unexpected asset: %+v", asset)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetRepo_Get_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets WHERE id=\$1`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	repo := NewAssetRepo(db)
	_, err = repo.Get(999)
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
	if err.Error() != "asset not found" {
		t.Errorf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets ORDER BY id LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tags", "last_seen"}).
			AddRow(1, "n1", "d1", "{}", nil).
			AddRow(2, "n2", "d2", "{}", nil))

	repo := NewAssetRepo(db)
	assets, err := repo.List(10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(assets) != 2 || assets[0].Name != "n1" || assets[1].Name != "n2" {
		t.Errorf("unexpected list: %+v", assets)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`DELETE FROM assets WHERE id=\$1`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewAssetRepo(db)
	err = repo.Delete(1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetRepo_Heartbeat(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`UPDATE assets SET last_seen = NOW\(\) WHERE id = \$1`).
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	now := time.Now()
	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets WHERE id=\$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tags", "last_seen"}).
			AddRow(1, "a1", "d1", "{}", now))

	repo := NewAssetRepo(db)
	asset, err := repo.Heartbeat(1)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if asset.ID != 1 || asset.Name != "a1" {
		t.Errorf("unexpected asset: %+v", asset)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
