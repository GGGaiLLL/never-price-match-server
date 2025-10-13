package product

// Repo defines the interface for product data persistence
type Repo interface {
	// GetProductsByCategory retrieves a list of products by category
	GetProductsByCategory(category string) ([]Product, error)
	// Seed creates some initial data for testing
	Seed() error
}
