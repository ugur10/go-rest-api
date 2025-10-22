# go-rest-api

A RESTful books API built with Go's standard library. The project showcases idiomatic HTTP handlers, middleware patterns, context-driven timeouts, and graceful shutdown without relying on third-party frameworks.

## Features
- CRUD endpoints for managing books in an in-memory, thread-safe repository
- Middleware stack providing request logging, CORS headers, and per-request timeouts
- Consistent JSON error responses with appropriate HTTP status codes
- Graceful shutdown triggered by OS signals (Ctrl+C)
- Step-by-step commit history for each project milestone

## Getting Started
```bash
git clone https://github.com/ugur10/go-rest-api.git
cd go-rest-api
go run ./cmd/server
```
The server listens on `http://localhost:8081`. Use `Ctrl+C` to stop it gracefully.

## Running Tests
```bash
go test ./...
```
(Integration tests are demonstrated via the curl examples below.)

## API Reference
| Method | Path                 | Description              |
|--------|----------------------|--------------------------|
| GET    | `/health`            | Health check             |
| GET    | `/api/books`         | List all books           |
| GET    | `/api/books/{id}`    | Retrieve a book by ID    |
| POST   | `/api/books`         | Create a new book        |
| PUT    | `/api/books/{id}`    | Update an existing book  |
| DELETE | `/api/books/{id}`    | Remove a book by ID      |

## Request & Response Examples
List books:
```bash
curl http://localhost:8081/api/books | jq
```

Create a book:
```bash
curl -X POST http://localhost:8081/api/books \
  -H 'Content-Type: application/json' \
  -d '{
        "title": "Go Workshop",
        "author": "Jane Doe",
        "isbn": "9780000000000",
        "publishedYear": 2024
      }' | jq
```

Update a book:
```bash
curl -X PUT http://localhost:8081/api/books/5 \
  -H 'Content-Type: application/json' \
  -d '{
        "title": "Go Workshop (2nd Edition)",
        "author": "Jane Doe",
        "isbn": "9780000000000",
        "publishedYear": 2025
      }' | jq
```

Delete a book:
```bash
curl -X DELETE http://localhost:8081/api/books/5 -i
```

## Error Responses
All errors share the following JSON shape:
```json
{
  "error": "human-friendly message"
}
```
Examples:
- `415 Unsupported Media Type` – `content type must be application/json`
- `400 Bad Request` – `invalid JSON payload`
- `404 Not Found` – `book not found`
- `504 Gateway Timeout` – `request timed out`

## Middleware & Timeouts
Middleware is composed without external packages:
1. **Timeout** – attaches a 10s context deadline to each request.
2. **Logging** – records method, path, status, payload size, and duration.
3. **CORS** – enables simple cross-origin access and handles `OPTIONS` preflight.

Requests that exceed the timeout (or have their context cancelled) receive a JSON error with an appropriate status code.

## Graceful Shutdown
`main` listens for `SIGINT`/`SIGTERM` and calls `Shutdown` with a five second deadline, allowing in-flight requests to complete before the server exits. If graceful shutdown fails, the server falls back to `Close`.

## Project Layout
```
cmd/server      # Application entrypoint and HTTP handlers
internal/books  # Book model and in-memory repository implementation
```

## Formatting & Linting
Use `gofmt` before committing changes:
```bash
gofmt -w cmd/server/main.go internal/books/*.go
```

Feel free to build upon each commit to explore additional patterns (authentication, persistence, pagination, etc.).
