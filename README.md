# go-rest-api

A minimal RESTful API server written in Go, using only the standard library. This project demonstrates how to set up an HTTP server, implement simple routing, and serve a health check endpoint.

## Features (Step 1)
- Go module initialized as `github.com/ugur10/go-rest-api`
- Basic HTTP server listening on port 8081
- Health check endpoint at `/health`

## Features (Step 2)
- Book domain model with JSON struct tags
- Thread-safe in-memory repository with seeded sample data

## Features (Step 3)
- `GET /api/books` returns all books as JSON
- `GET /api/books/{id}` returns a single book or 404 if missing

More features will be added in subsequent steps.

## Getting Started

```bash
go run ./cmd/server
```

Visit [http://localhost:8081/health](http://localhost:8081/health) to verify the server is running.
