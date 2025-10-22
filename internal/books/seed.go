package books

// SeedData returns example books to pre-populate the repository.
func SeedData() []Book {
	return []Book{
		{
			ID:            "1",
			Title:         "The Go Programming Language",
			Author:        "Alan A. A. Donovan",
			ISBN:          "9780134190440",
			PublishedYear: 2015,
		},
		{
			ID:            "2",
			Title:         "Introducing Go",
			Author:        "Caleb Doxsey",
			ISBN:          "9781491941959",
			PublishedYear: 2016,
		},
		{
			ID:            "3",
			Title:         "Concurrency in Go",
			Author:        "Katherine Cox-Buday",
			ISBN:          "9781491941195",
			PublishedYear: 2017,
		},
		{
			ID:            "4",
			Title:         "Go in Practice",
			Author:        "Matt Butcher",
			ISBN:          "9781633430075",
			PublishedYear: 2016,
		},
	}
}
