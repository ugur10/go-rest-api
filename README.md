# go-rest-api

RESTful Books API powered exclusively by Go's standard library. The project doubles as a learning resource for HTTP handlers, middleware composition, context-aware timeouts, and graceful shutdown patterns without third-party frameworks.

---

## Quick Links
- [Highlights](#highlights)
- [Architecture Overview](#architecture-overview)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Usage Examples](#usage-examples)
- [Middleware Stack](#middleware-stack)
- [Graceful Shutdown](#graceful-shutdown)
- [Project Layout](#project-layout)
- [Testing & Tooling](#testing--tooling)
- [Contributing](#contributing)
- [License](#license)

---

## Highlights
- Clean CRUD endpoints backed by a thread-safe in-memory repository
- Middleware chain for request logging, CORS headers, and per-request timeouts
- Consistent JSON error envelopes across the entire API
- Graceful shutdown responding to `SIGINT`/`SIGTERM` with a 5s drain period
- Incremental commit history that mirrors the learning plan

## Architecture Overview
- **Language:** Go (standard library only)
- **Entry point:** `cmd/server/main.go`
- **Domain layer:** `internal/books` exposes the `Book` model and repository abstraction
- **Storage:** In-memory map protected by `sync.RWMutex`, seeded with sample titles
- **Lifecycle:** Request context deadlines + graceful shutdown manage long-running work

---

## Getting Started
```bash
git clone https://github.com/ugur10/go-rest-api.git
cd go-rest-api
go run ./cmd/server
```

The server listens on **http://localhost:8081**. Stop it with `Ctrl+C`; the shutdown handler drains active requests before exit.

---

## API Reference
| Method | Path              | Description             |
| ------ | ----------------- | ----------------------- |
| GET    | `/health`         | Service health probe    |
| GET    | `/api/books`      | List every book         |
| GET    | `/api/books/{id}` | Retrieve a book by ID   |
| POST   | `/api/books`      | Create a new book       |
| PUT    | `/api/books/{id}` | Update an existing book |
| DELETE | `/api/books/{id}` | Remove a book           |

All responses are JSON. Errors follow this shape:
```json
{
  "error": "human-friendly message"
}
```

Common responses:
- 400 — invalid JSON payload or missing required fields
- 404 — book not found
- 415 — non-JSON `Content-Type`
- 504 — request exceeded the timeout window

---

## Usage Examples
### List books
```bash
curl http://localhost:8081/api/books | jq
```

### Create a book
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

### Update a book
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

### Delete a book
```bash
curl -X DELETE http://localhost:8081/api/books/5 -i
```

---

## Middleware Stack
Middleware is composed manually (no external packages) in the following order:
1. **Timeout** — attaches a 10 second `context.Context` deadline to each request.
2. **Logging** — records method, path, status, bytes written, and duration.
3. **CORS** — enables cross-origin requests and short-circuits `OPTIONS` preflight.

If the client disconnects or the deadline is exceeded, handlers emit a JSON error with the correct status code before returning.

---

## Graceful Shutdown
`main` wires a signal listener (`SIGINT`/`SIGTERM`) and invokes `http.Server.Shutdown` with a five second timeout. Any in-flight requests get a chance to finish; if shutdown fails, the server falls back to `Close`.

---

## Project Layout
```
cmd/server      # HTTP handlers, middleware, server bootstrap
internal/books  # Book model, repository interface, in-memory implementation
```

---

## Testing & Tooling
Run the unit tests:
```bash
go test ./...
```

Format the codebase before committing:
```bash
gofmt -w cmd/server/main.go cmd/server/main_test.go internal/books/*.go
```

---

## Contributing
Issues and pull requests are welcome. For substantial changes, please open an issue to outline scope and alignment with the learning goals before submitting a PR.

---

## License
Distributed under the [MIT License](LICENSE).
