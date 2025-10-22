package books

import (
	"context"
	"sort"
	"strconv"
	"sync"
)

// MemoryRepository provides an in-memory implementation of Repository.
type MemoryRepository struct {
	mu     sync.RWMutex
	books  map[string]Book
	nextID int
}

// NewMemoryRepository constructs a MemoryRepository seeded with the provided books.
func NewMemoryRepository(seed []Book) *MemoryRepository {
	repo := &MemoryRepository{
		books:  make(map[string]Book, len(seed)),
		nextID: 1,
	}

	for _, book := range seed {
		repo.books[book.ID] = book

		if id, err := strconv.Atoi(book.ID); err == nil && id >= repo.nextID {
			repo.nextID = id + 1
		}
	}

	return repo
}

// List returns all books in ascending ID order.
func (r *MemoryRepository) List(_ context.Context) ([]Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Book, 0, len(r.books))
	for _, book := range r.books {
		result = append(result, book)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

// Get retrieves a book by its ID.
func (r *MemoryRepository) Get(_ context.Context, id string) (Book, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	book, ok := r.books[id]
	return book, ok, nil
}

// Create adds a new book to the repository, automatically assigning an ID.
func (r *MemoryRepository) Create(_ context.Context, book Book) (Book, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	book.ID = strconv.Itoa(r.nextID)
	r.nextID++

	r.books[book.ID] = book
	return book, nil
}

// Update replaces the book with the given ID if it exists.
func (r *MemoryRepository) Update(_ context.Context, id string, book Book) (Book, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.books[id]; !ok {
		return Book{}, false, nil
	}

	book.ID = id
	r.books[id] = book
	return book, true, nil
}

// Delete removes the book with the provided ID if it exists.
func (r *MemoryRepository) Delete(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.books[id]; !ok {
		return false, nil
	}

	delete(r.books, id)
	return true, nil
}
