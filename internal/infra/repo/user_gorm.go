package repo

import (
	"never-price-match-server/internal/user"

	"gorm.io/gorm"
)

// Implements the user.Repo interface
type userGormRepo struct {
	db *gorm.DB
}

func NewUserGormRepo(db *gorm.DB) user.Repo {
	return &userGormRepo{db: db}
}

func (r *userGormRepo) Create(user *user.User) error {
	return r.db.Create(user).Error
}

func (r *userGormRepo) GetByID(id string) (*user.User, error) {
	var u user.User
	if err := r.db.Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userGormRepo) GetByEmail(email string) (*user.User, error) {
	var u user.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userGormRepo) GetAll() ([]*user.User, error) {
	var users []*user.User
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userGormRepo) CheckEmailExists(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&user.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
