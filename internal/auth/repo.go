package auth

import (
	"errors"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

type Repository interface {
	Create(token *RefreshToken) error
	GetByHash(hash string) (*RefreshToken, error)
	Delete(tokenHash string) error
	Transaction(callback func(r Repository) error) error
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) Create(token *RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *repository) GetByHash(hash string) (*RefreshToken, error) {
	var token RefreshToken

	if err := r.db.Where("refresh_token = ?", hash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	return &token, nil
}

func (r *repository) Delete(tokenHash string) error {
	db := r.db.Where("refresh_token = ?", tokenHash).Delete(&RefreshToken{})

	if db.Error != nil {
		return db.Error
	}

	if db.RowsAffected == 0 {
		return ErrTokenNotFound
	}

	return nil
}

func (r *repository) Transaction(calls func(r Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return calls(&repository{tx})
	})
}
