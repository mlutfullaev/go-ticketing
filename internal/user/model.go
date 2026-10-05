package user

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email        string `gorm:"unique"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type CreateUser struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=32"`
	Role     string `json:"role"`
}
