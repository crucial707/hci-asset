package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func Connect(
	host, port, name, user, password string,
) (*sql.DB, error) {

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host, port, name, user, password,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
