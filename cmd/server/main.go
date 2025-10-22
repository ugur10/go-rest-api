package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ugur10/go-rest-api/internal/books"
)

type application struct {
	store books.Repository
}

func main() {
	store := books.NewMemoryRepository(books.SeedData())
	app := &application{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.healthHandler)

	addr := ":8081"
	log.Printf("Starting server on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"status":"ok"}`)
}
