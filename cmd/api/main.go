package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/crucial707/hci-asset/internal/config"
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
}
