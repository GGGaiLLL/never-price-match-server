package product

import "time"

type Product struct {
	ID                   uint   `gorm:"primarykey"`
	Name                 string `gorm:"index: uniq, name: ux_product_key"`
	Category             string
	Brand                string
	Model                string
	ImageURL             string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Prices               []Price `gorm:"foreignKey:ProductID"`
	LowestPrice          float64
	LowestPricePlatform  string
	LowestPriceUpdatedAt time.Time
	LowestPriceLink      string
}

// DELETED the old, incorrect ScrapedInfo struct from this file.

type Price struct {
	ID        uint `gorm:"primarykey"`
	ProductID uint
	Platform  string
	Price     float64
	Link      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
