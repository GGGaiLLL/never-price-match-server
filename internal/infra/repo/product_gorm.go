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

// Seed inserts some initial product data into the database for testing
func (r *productGormRepo) Seed() error {
	var count int64
	r.db.Model(&product.Product{}).Count(&count)
	if count > 0 {
		return nil // Do not insert if data already exists
	}

	products := []product.Product{
		{
			Name:     "MacBook Pro",
			Category: "Electronics",
			ImageURL: "https://example.com/macbook.jpg",
			Prices: []product.Price{
				{Platform: "Amazon.com", Price: 1299.00, Link: "https://www.amazon.com/dp/B09JQS523B"}, // Example Amazon link
				{Platform: "eBay", Price: 1289.00, Link: "https://www.ebay.com/itm/1234567890"},        // Example eBay link
			},
		},
		{
			Name:     "iPhone 15",
			Category: "Electronics",
			ImageURL: "https://example.com/iphone15.jpg",
			Prices: []product.Price{
				{Platform: "Amazon.com", Price: 799.00, Link: "https://www.amazon.com/dp/B0CHWR36X1"}, // Example Amazon link
				{Platform: "Walmart", Price: 779.00, Link: "https://www.walmart.com/ip/123456789"},    // Example Walmart link
			},
		},
	}

	return r.db.Create(&products).Error
}
