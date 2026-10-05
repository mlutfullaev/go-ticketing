package order

import (
	"ticketing/internal/event"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	EventID       uint
	UserID        uint
	PaymentMethod string
	PaymentStatus string
	TotalAmount   uint
	Seats         []OrderSeat
}

type OrderSeat struct {
	OrderID uint `gorm:"primaryKey"`
	SeatID  uint `gorm:"primaryKey"`
	Seat    event.Seat
}

type CreateOrder struct {
	gorm.Model
	HoldID string `json:"hold_id" validate:"required"`
}
