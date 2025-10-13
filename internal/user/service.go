package user

import (
	"errors"
	"never-price-match-server/internal/infra/logger"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Service defines the business logic interface for users
type Service interface {
	List() ([]*User, error)
	Get(id string) (*User, error)
	Create(name, email, password string) (*User, error)
	CheckEmailExist(email string) (bool, error)
	Login(email, password string) (*User, error)
}

// service is the private implementation of the Service interface
type service struct {
	repo Repo
}

// NewService creates a new user service instance
func NewService(r Repo) Service {
	return &service{repo: r}
}

// The receiver for List, Get, Create, CheckEmailExist, Login methods changed from *Service to *service
func (s *service) List() ([]*User, error) {
	return s.repo.GetAll()
}

func (s *service) Get(id string) (*User, error) {
	return s.repo.GetByID(id)
}

func (s *service) Create(name, email, password string) (*User, error) {
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

func (s *service) CheckEmailExist(email string) (bool, error) {
	logger.L.Info("checking email existence", logger.Str("email", email))
	e := strings.ToLower(strings.TrimSpace(email))
	return s.repo.CheckEmailExists(e)
}

var ErrInvalidEmail = errors.New("Invalid email")
var ErrInvalidPassword = errors.New("Invalid password")

func (s *service) Login(email string, password string) (*User, error) {
	e := strings.ToLower(strings.TrimSpace(email))

	u, err := s.repo.GetByEmail(e)
	if err != nil || u == nil {
		logger.L.Warn("login failed: user not found", logger.Str("email", e), logger.Err(err))
		return nil, ErrInvalidEmail
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		logger.L.Warn("login failed: incorrect password", logger.Str("email", e))
		return nil, ErrInvalidPassword
	}

	return u, nil
}
