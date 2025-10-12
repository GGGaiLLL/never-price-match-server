package repo

import (
	"errors"

	"gorm.io/gorm"
	"never-price-match-server/internal/user"
)

// 实现 user.Repo 接口
type UserGormRepo struct{ db *gorm.DB }

func NewUserGormRepo(db *gorm.DB) *UserGormRepo { return &UserGormRepo{db: db} }

func (r *UserGormRepo) GetAll() ([]*user.User, error) {
	var list []*user.User
	if err := r.db.Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *UserGormRepo) GetByID(id string) (*user.User, error) {
	var u user.User
	if err := r.db.First(&u, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserGormRepo) Create(u *user.User) error {
	return r.db.Create(u).Error
}

func (r *UserGormRepo) CheckEmailExists(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&user.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserGormRepo) GetByEmail(email string) (*user.User, error) {
	var u user.User
	if err := r.db.First(&u, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &u, nil
}
