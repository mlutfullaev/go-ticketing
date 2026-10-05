package order

import (
	"context"

	"gorm.io/gorm"
)

type repo struct {
	db *gorm.DB
}
type Repo interface {
	CreateOrder(ctx context.Context, Order *Order) error
	Transaction(ctx context.Context, calls func(r Repo) error) error
}

func NewRepository(db *gorm.DB) Repo {
	return &repo{db}
}

func (r *repo) CreateOrder(ctx context.Context, Order *Order) error {
	return r.db.WithContext(ctx).Create(&Order).Error
}

func (r *repo) Transaction(ctx context.Context, calls func(r Repo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return calls(&repo{db: tx})
	})
}
