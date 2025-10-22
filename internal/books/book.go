package books

import "context"

// Book represents a single book in the collection.
type Book struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	ISBN          string `json:"isbn"`
	PublishedYear int    `json:"publishedYear"`
}

// Repository describes the behaviour required for storing books.
type Repository interface {
	List(ctx context.Context) ([]Book, error)
	Get(ctx context.Context, id string) (Book, bool, error)
	Create(ctx context.Context, book Book) (Book, error)
	Update(ctx context.Context, id string, book Book) (Book, bool, error)
	Delete(ctx context.Context, id string) (bool, error)
}
