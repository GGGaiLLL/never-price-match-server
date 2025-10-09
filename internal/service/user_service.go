package service

import (
    "never-price-match-server/internal/dao"
    "never-price-match-server/internal/model"
)

func CreateUser(name, email string) error {
    user := &model.User{Name: name, Email: email}
    return dao.CreateUser(user)
}

func ListUsers() ([]model.User, error) {
    return dao.ListUsers()
}
