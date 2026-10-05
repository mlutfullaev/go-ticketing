package user

import (
	"errors"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

type UserRepository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	GetByID(ID uint) (*User, error)
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &repository{db}
}

func (r *repository) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *repository) FindByEmail(email string) (*User, error) {
	var user User

	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	return &user, nil
}

func (r *repository) GetByID(ID uint) (*User, error) {
	var user User

	if err := r.db.Where("id = ?", ID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	return &user, nil
}
