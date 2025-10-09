package dao

import (
	"never-price-match-server/internal/db"
	"never-price-match-server/internal/model"
)

func CreateUser(u *model.User) error {
    return db.DB.Create(u).Error
}

func ListUsers() ([]model.User, error) {
    var users []model.User
    err := db.DB.Find(&users).Error
    return users, err
}
