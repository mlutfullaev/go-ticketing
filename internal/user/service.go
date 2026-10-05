package user

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type service struct {
	repo UserRepository
}
type Service interface {
	CreateUser(user *CreateUser) (*User, error)
	Authenticate(email, password string) (*User, error)
	GetByID(ID uint) (*User, error)
}

func NewUserService(repo UserRepository) Service {
	return &service{repo}
}

func (s *service) CreateUser(createUser *CreateUser) (*User, error) {
	passHash, err := bcrypt.GenerateFromPassword([]byte(createUser.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		Email:        createUser.Email,
		PasswordHash: string(passHash),
		Role:         createUser.Role,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) GetByID(ID uint) (*User, error) {
	return s.repo.GetByID(ID)
}

func (s *service) Authenticate(email, password string) (*User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	return user, nil
}
