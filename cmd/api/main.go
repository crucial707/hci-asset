package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/db"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	cfg := config.Load()
	log.Println("Starting server on :*" + cfg.Port)
	err := http.ListenAndServe("*"+cfg.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
	database, err := db.Connect(
		cfg.DBHost,
		cfg.DBport,
		cfg.DBName,
		cfg.DBUser,
		cfg.DBPass,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Successfully connected to the database")
	_ = database
}
