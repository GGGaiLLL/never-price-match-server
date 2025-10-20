package product

// Repo defines the interface for product data persistence.
type Repo interface {
	// SearchProductsByName finds products by a search term.
	SearchProductsByName(name string) ([]Product, error)
	// SaveProducts saves a list of products to the database.
	SaveProducts(products []Product) error
	// GetProductNamesByName retrieves a list of unique product names for suggestions.
	GetProductNamesByName(name string) ([]string, error)
}
