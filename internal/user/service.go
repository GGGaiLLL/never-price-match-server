package user

import (
	"never-price-match-server/internal/infra/logger"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repo
}

type CreateUserInput struct {
	Email    string
	Name     string
	Password string
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

func (s *Service) Create(name, email, password string) (*User, error) {
	// Hash the password before storing it
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.L.Error("password hashing failed", logger.Err(err))
		return nil, err
	}
	u := &User{Name: name, Email: strings.ToLower((strings.TrimSpace(email))), PasswordHash: string(hashedPassword)}
	if err := s.repo.Create(u); err != nil {
		logger.L.Error("create user failed", logger.Err(err))
		return nil, err
	}
	logger.L.Info("user created", logger.Str("id", u.ID))
	return u, nil
}

func (s *Service) CheckEmailExist(email string) (bool, error) {
	logger.L.Info("checking email existence", logger.Str("email", email))
	e := strings.ToLower(strings.TrimSpace(email))
	return s.repo.CheckEmailExists(e)
}
