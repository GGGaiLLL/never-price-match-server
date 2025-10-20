package product

import "time"

// Product represents a single product item scraped from a platform.
// Instead of a unique name, each entry is a distinct record of what was found.
type Product struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"index"` // Name is indexed for faster searching but is not unique.
	Platform  string
	Price     float64
	Link      string
	ImageURL  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Price struct is no longer needed as Price is now part of the Product itself.

type scrapeProductParams struct {
	SearchTerm        string
	Platform          string
	SearchURL         string
	ContainerSelector string
	TitleSelector     string
	PriceSelectors    []string
	ImageSelector     string
	LinkSelector      string
	ImageAttr         string
}
