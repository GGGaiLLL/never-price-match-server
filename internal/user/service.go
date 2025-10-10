package user

import (
	"never-price-match-server/internal/infra/logger"
)

type Service struct {
	repo Repo
}

func NewService(r Repo) *Service {
	return &Service{repo: r}
}

func (s *Service) List() ([]*User, error) {
	return s.repo.GetAll()
}

func (s *Service) Get(id string) (*User, error) {
	return s.repo.GetByID(id)
}

func (s *Service) Create(name, email string) (*User, error) {
	u := &User{Name: name, Email: email}
	if err := s.repo.Create(u); err != nil {
		logger.L.Error("create user failed", logger.Err(err))
		return nil, err
	}
	logger.L.Info("user created", logger.Str("id", u.ID))
	return u, nil
}
