package auth

import (
	"ticketing/internal/user"
	"ticketing/pkg/config"
	"time"
)

type service struct {
	userService user.Service
	jwt         config.JWTConfig
	repo        Repository
}

type Service interface {
	Register(body *user.CreateUser) (*user.User, error)
	Login(body *LoginBody) (*LoginResponse, error)
	Refresh(token string) (*LoginResponse, error)
}

func NewAuthService(userService user.Service, jwt config.JWTConfig, repo Repository) Service {
	return &service{userService, jwt, repo}
}

func (s *service) Register(body *user.CreateUser) (*user.User, error) {
	return s.userService.CreateUser(body)
}

func (s *service) Login(body *LoginBody) (*LoginResponse, error) {
	user, err := s.userService.Authenticate(body.Email, body.Password)
	if err != nil {
		return nil, err
	}

	accessToken, err := IssueToken(user.ID, user.Role, s.jwt.Secret, s.jwt.AccessTTL)
	if err != nil {
		return nil, err
	}

	newToken := NewToken()
	newRefreshToken := &RefreshToken{
		RefreshToken: hashToken(newToken),
		UserID:       user.ID,
		ExpiresAt:    time.Now().Add(s.jwt.RefreshTTL),
	}
	if err := s.repo.Create(newRefreshToken); err != nil {
		return nil, err
	}

	return &LoginResponse{
		RefreshToken: newToken,
		ExpiresIn:    newRefreshToken.ExpiresAt,
		AccessToken:  accessToken,
	}, nil
}

func (s *service) Refresh(token string) (*LoginResponse, error) {
	existingToken, err := s.repo.GetByHash(hashToken(token))
	if err != nil {
		return nil, err
	}

	newToken := NewToken()
	newRefreshToken := &RefreshToken{
		RefreshToken: hashToken(newToken),
		UserID:       existingToken.UserID,
		ExpiresAt:    time.Now().Add(s.jwt.RefreshTTL),
	}

	if time.Now().After(existingToken.ExpiresAt) {
		if err := s.repo.Delete(existingToken.RefreshToken); err != nil {
			return nil, err
		}

		return nil, ErrTokenExpired
	}

	existingUser, err := s.userService.GetByID(existingToken.UserID)

	err = s.repo.Transaction(func(r Repository) error {
		if err := r.Delete(existingToken.RefreshToken); err != nil {
			return err
		}

		if err := r.Create(newRefreshToken); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := IssueToken(existingToken.UserID, existingUser.Role, s.jwt.Secret, s.jwt.AccessTTL)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newToken,
		ExpiresIn:    newRefreshToken.ExpiresAt,
	}, nil
}
