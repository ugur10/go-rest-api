package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ugur10/go-rest-api/internal/books"
)

var (
	errUnsupportedMediaType = errors.New("unsupported media type")
	errInvalidJSON          = errors.New("invalid json")
	errInvalidPayload       = errors.New("invalid payload")
)

type application struct {
	store books.Repository
}

type middleware func(http.Handler) http.Handler

type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if rr.status == 0 {
		rr.status = http.StatusOK
	}

	n, err := rr.ResponseWriter.Write(b)
	rr.size += n
	return n, err
}

func chain(h http.Handler, middlewares ...middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
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

	handler := chain(mux, app.loggingMiddleware, corsMiddleware)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func (app *application) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rr, r)

		status := rr.status
		if status == 0 {
			status = http.StatusOK
		}

		log.Printf("%s %s %d %dB %s", r.Method, r.URL.Path, status, rr.size, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Location")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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
	case http.MethodPut:
		app.updateBook(w, r, id)
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
	payload, err := readBookPayload(r)
	if err != nil {
		handlePayloadError(w, err)
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

func (app *application) updateBook(w http.ResponseWriter, r *http.Request, id string) {
	payload, err := readBookPayload(r)
	if err != nil {
		handlePayloadError(w, err)
		return
	}

	book := books.Book{
		Title:         payload.Title,
		Author:        payload.Author,
		ISBN:          payload.ISBN,
		PublishedYear: payload.PublishedYear,
	}

	updated, ok, err := app.store.Update(r.Context(), id, book)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, updated)
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

type bookPayload struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn"`
	PublishedYear int    `json:"publishedYear"`
}

func readBookPayload(r *http.Request) (bookPayload, error) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return bookPayload{}, errUnsupportedMediaType
	}

	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return bookPayload{}, err
	}

	var payload bookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return bookPayload{}, errInvalidJSON
	}

	payload.Title = strings.TrimSpace(payload.Title)
	payload.Author = strings.TrimSpace(payload.Author)
	payload.ISBN = strings.TrimSpace(payload.ISBN)

	if payload.Title == "" || payload.Author == "" {
		return bookPayload{}, errInvalidPayload
	}

	return payload, nil
}

func handlePayloadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errUnsupportedMediaType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Is(err, errInvalidJSON):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, errInvalidPayload):
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write json response: %v", err)
	}
}
