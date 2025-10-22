package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ugur10/go-rest-api/internal/books"
)

var (
	errUnsupportedMediaType = errors.New("unsupported media type")
	errInvalidJSON          = errors.New("invalid json")
	errInvalidPayload       = errors.New("invalid payload")
)

// application bundles the dependencies required by HTTP handlers.
type application struct {
	store   books.Repository
	timeout time.Duration
}

// middleware is a function that decorates an http.Handler.
type middleware func(http.Handler) http.Handler

// responseRecorder captures status and body size for logging middleware.
type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader stores the status code before delegating to the underlying writer.
func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

// Write records the response size while streaming to the client.
func (rr *responseRecorder) Write(b []byte) (int, error) {
	if rr.status == 0 {
		rr.status = http.StatusOK
	}

	n, err := rr.ResponseWriter.Write(b)
	rr.size += n
	return n, err
}

// chain wraps a handler with the supplied middleware in declaration order.
func chain(h http.Handler, middlewares ...middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// main wires dependencies and starts the HTTP server.
func main() {
	store := books.NewMemoryRepository(books.SeedData())
	app := &application{store: store, timeout: 10 * time.Second}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.healthHandler)
	mux.HandleFunc("/api/books", app.booksHandler)
	mux.HandleFunc("/api/books/", app.bookHandler)

	addr := ":8081"
	log.Printf("Starting server on %s", addr)

	handler := chain(mux, app.timeoutMiddleware, app.loggingMiddleware, corsMiddleware)

	srv := &http.Server{Addr: addr, Handler: handler}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	shutdown(srv)
}

// shutdown blocks until an interrupt signal is received and then gracefully stops the server.
func shutdown(srv *http.Server) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Shutdown signal received, stopping server")

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxTimeout); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := srv.Close(); err != nil {
			log.Printf("forced close failed: %v", err)
		}
	}
}

// timeoutMiddleware attaches a deadline to each request context.
func (app *application) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), app.timeout)
		defer cancel()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware emits request method/path/response metrics for each call.
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

// corsMiddleware enables cross-origin requests and handles preflight checks.
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

// healthHandler reports basic server liveness.
func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, `{"status":"ok"}`)
}

// booksHandler routes collection-level requests (list/create).
func (app *application) booksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		app.listBooks(w, r)
	case http.MethodPost:
		app.createBook(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// bookHandler routes item-level requests (read/update/delete).
func (app *application) bookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := extractID(strings.TrimPrefix(r.URL.Path, "/api/books"))
	if err != nil {
		writeError(w, http.StatusNotFound, "book not found")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// extractID parses the book identifier from the request path.
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

// listBooks returns the full catalogue as JSON.
func (app *application) listBooks(w http.ResponseWriter, r *http.Request) {
	books, err := app.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list books")
		return
	}

	if contextDone(w, r.Context()) {
		return
	}

	writeJSON(w, http.StatusOK, books)
}

// getBook fetches a single book by ID or returns 404 if not found.
func (app *application) getBook(w http.ResponseWriter, r *http.Request, id string) {
	book, ok, err := app.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch book")
		return
	}

	if contextDone(w, r.Context()) {
		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}

	writeJSON(w, http.StatusOK, book)
}

// createBook inserts a new book after validating the JSON payload.
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
		writeError(w, http.StatusInternalServerError, "failed to create book")
		return
	}

	if contextDone(w, r.Context()) {
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/books/%s", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

// updateBook replaces an existing book when the payload is valid.
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
		writeError(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	if contextDone(w, r.Context()) {
		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// deleteBook removes the book identified by id, returning 404 if missing.
func (app *application) deleteBook(w http.ResponseWriter, r *http.Request, id string) {
	deleted, err := app.store.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	if contextDone(w, r.Context()) {
		return
	}

	if !deleted {
		writeError(w, http.StatusNotFound, "book not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// bookPayload mirrors the expected JSON body for create/update operations.
type bookPayload struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn"`
	PublishedYear int    `json:"publishedYear"`
}

// readBookPayload validates headers, limits body size, and decodes JSON input.
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

// handlePayloadError maps payload parsing failures to HTTP responses.
func handlePayloadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errUnsupportedMediaType):
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
	case errors.Is(err, errInvalidJSON):
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
	case errors.Is(err, errInvalidPayload):
		writeError(w, http.StatusBadRequest, "title and author are required")
	default:
		writeError(w, http.StatusBadRequest, "invalid request body")
	}
}

// contextDone inspects the request context and writes a timeout response if needed.
func contextDone(w http.ResponseWriter, ctx context.Context) bool {
	if err := ctx.Err(); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			writeError(w, http.StatusGatewayTimeout, "request timed out")
		case errors.Is(err, context.Canceled):
			writeError(w, http.StatusRequestTimeout, "request canceled")
		default:
			writeError(w, http.StatusRequestTimeout, "request canceled")
		}
		return true
	}
	return false
}

// writeError emits a JSON error response with the given status code.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeJSON serialises the supplied value as JSON with the provided status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write json response: %v", err)
	}
}
