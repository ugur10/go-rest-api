package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

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
	mux.HandleFunc("/api/books", app.booksHandler)
	mux.HandleFunc("/api/books/", app.bookHandler)

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

func (app *application) booksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.listBooks(w, r)
	case http.MethodPost:
		app.createBook(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *application) bookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(strings.TrimPrefix(r.URL.Path, "/api/books"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		app.getBook(w, r, id)
	case http.MethodDelete:
		app.deleteBook(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func extractID(path string) (string, error) {
	if len(path) < 2 {
		return "", errors.New("missing id")
	}

	if path[0] != '/' {
		return "", errors.New("invalid path")
	}

	id := strings.Trim(path[1:], "/")
	if id == "" || strings.Contains(id, "/") {
		return "", errors.New("invalid id")
	}

	return id, nil
}

func (app *application) listBooks(w http.ResponseWriter, r *http.Request) {
	books, err := app.store.List(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, books)
}

func (app *application) getBook(w http.ResponseWriter, r *http.Request, id string) {
	book, ok, err := app.store.Get(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, book)
}

func (app *application) createBook(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload struct {
		Title         string `json:"title"`
		Author        string `json:"author"`
		ISBN          string `json:"isbn"`
		PublishedYear int    `json:"publishedYear"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Title == "" || payload.Author == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	book := books.Book{
		Title:         payload.Title,
		Author:        payload.Author,
		ISBN:          payload.ISBN,
		PublishedYear: payload.PublishedYear,
	}

	created, err := app.store.Create(r.Context(), book)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/books/%s", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

func (app *application) deleteBook(w http.ResponseWriter, r *http.Request, id string) {
	deleted, err := app.store.Delete(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !deleted {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write json response: %v", err)
	}
}
