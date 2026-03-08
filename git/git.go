package git

type Repository interface {
	// Query

	// Diff

	// Index

	// Branch Management

	// Remote

	// Stash

	// Meta
	Root() string
}