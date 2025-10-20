package repo

import (
	"never-price-match-server/internal/product"

	"gorm.io/gorm"
)

type productGormRepo struct {
	db *gorm.DB
}

// NewProductGormRepo creates a new GORM product repository instance
func NewProductGormRepo(db *gorm.DB) product.Repo {
	return &productGormRepo{db: db}
}

// SearchProductsByName performs a case-insensitive search for products by name.
func (r *productGormRepo) SearchProductsByName(name string) ([]product.Product, error) {
	var products []product.Product
	err := r.db.Where("LOWER(name) LIKE LOWER(?)", "%"+name+"%").Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// SaveProducts saves a slice of new Product entities to the database.
func (r *productGormRepo) SaveProducts(products []product.Product) error {
	if len(products) == 0 {
		return nil
	}
	// This is now a simple batch-create operation.
	return r.db.Create(&products).Error
}

// GetProductsByCategory retrieves products from the database by category
func (r *productGormRepo) GetProductsByCategory(category string) ([]product.Product, error) {
	var products []product.Product
	// Use Preload("Prices") to load associated price information simultaneously
	err := r.db.Where("category = ?", category).Preload("Prices").Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

// GetProductNamesByName performs a distinct, case-insensitive search for product names.
// It's optimized for search suggestions, limiting the result to 10 records.
func (r *productGormRepo) GetProductNamesByName(name string) ([]string, error) {
	var names []string
	err := r.db.Model(&product.Product{}).
		Distinct("name").
		Where("LOWER(name) LIKE LOWER(?)", "%"+name+"%").
		Limit(10).
		Pluck("name", &names).Error

	if err != nil {
		return nil, err
	}
	return names, nil
}
