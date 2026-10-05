package event

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

type Repository interface {
	CreateEvent(event *Event) error
	GetEvents() ([]*Event, error)
	CreateSeat(seat *Seat) error
	GetEventByID(eventID uint) (*Event, error)
	UpdateSeatStatus(ctx context.Context, seatID uint, status string) error
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) CreateEvent(event *Event) error {
	return r.db.Create(event).Error
}

func (r *repository) GetEvents() ([]*Event, error) {
	var events []*Event

	r.db.Find(&events)

	return events, nil
}

func (r *repository) GetEventByID(eventID uint) (*Event, error) {
	var event Event

	if err := r.db.Preload("Seats").First(&event, eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	return &event, nil
}

func (r *repository) CreateSeat(seat *Seat) error {
	return r.db.Create(seat).Error
}

func (r *repository) UpdateSeatStatus(ctx context.Context, seatID uint, status string) error {
	db := r.db.WithContext(ctx).Model(&Seat{}).Where("id = ?", seatID).Where("status = ?", "available").Update("status", status)
	if db.Error != nil {
		return db.Error
	}

	if db.RowsAffected == 0 {
		return ErrSeatsAreNotAvailable
	}

	return nil
}
