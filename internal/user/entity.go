package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	Name      string    `gorm:"type:varchar(128);not null"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}
